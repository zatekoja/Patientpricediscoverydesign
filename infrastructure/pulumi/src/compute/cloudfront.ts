import * as pulumi from '@pulumi/pulumi';
import * as aws from '@pulumi/aws';
import { getResourceTags } from '../tagging';

export interface CloudFrontConfig {
  environment: string;
  s3BucketDomainName: pulumi.Output<string>;
  s3BucketArn: pulumi.Output<string>;
  certificateArn?: pulumi.Output<string>; // ACM certificate in us-east-1
  domainAliases?: string[];
}

export interface CloudFrontOutputs {
  distributionId: pulumi.Output<string>;
  distributionArn: pulumi.Output<string>;
  distributionDomainName: pulumi.Output<string>;
  distributionHostedZoneId: pulumi.Output<string>;
}

/**
 * Create Origin Access Identity for S3
 */
export function createOriginAccessIdentity(config: CloudFrontConfig): aws.cloudfront.OriginAccessIdentity {
  return new aws.cloudfront.OriginAccessIdentity(`ohi-${config.environment}-frontend`, {
    comment: `Origin Access Identity for ohi-${config.environment} frontend`,
  });
}

/**
 * Create S3 bucket policy to allow CloudFront access
 */
export function createS3BucketPolicy(
  config: CloudFrontConfig,
  bucketId: pulumi.Output<string>,
  oai: aws.cloudfront.OriginAccessIdentity
): aws.s3.BucketPolicy {
  const policy = pulumi.all([config.s3BucketArn, oai.iamArn]).apply(([bucketArn, iamArn]) =>
    JSON.stringify({
      Version: '2012-10-17',
      Statement: [
        {
          Sid: 'AllowCloudFrontOAI',
          Effect: 'Allow',
          Principal: {
            AWS: iamArn,
          },
          Action: 's3:GetObject',
          Resource: `${bucketArn}/*`,
        },
      ],
    })
  );

  return new aws.s3.BucketPolicy(`ohi-${config.environment}-frontend-policy`, {
    bucket: bucketId,
    policy: policy,
  });
}

/**
 * Get cache policy configuration
 */
function getCachePolicyConfig(environment: string): {
  defaultTtl: number;
  maxTtl: number;
  minTtl: number;
} {
  if (environment === 'prod') {
    return {
      defaultTtl: 86400, // 24 hours
      maxTtl: 31536000, // 1 year
      minTtl: 0,
    };
  }

  // Shorter cache for dev/staging
  return {
    defaultTtl: 300, // 5 minutes
    maxTtl: 3600, // 1 hour
    minTtl: 0,
  };
}

/**
 * Create CloudFront distribution for frontend
 */
export function createCloudFrontDistribution(
  config: CloudFrontConfig,
  oai: aws.cloudfront.OriginAccessIdentity
): aws.cloudfront.Distribution {
  const cacheConfig = getCachePolicyConfig(config.environment);

  const distributionConfig: aws.cloudfront.DistributionArgs = {
    enabled: true,
    isIpv6Enabled: true,
    comment: `OHI ${config.environment} frontend distribution`,
    defaultRootObject: 'index.html',
    priceClass: config.environment === 'prod' ? 'PriceClass_All' : 'PriceClass_100',

    origins: [
      {
        originId: 's3-frontend',
        domainName: config.s3BucketDomainName,
        s3OriginConfig: {
          originAccessIdentity: oai.cloudfrontAccessIdentityPath,
        },
      },
    ],

    defaultCacheBehavior: {
      targetOriginId: 's3-frontend',
      viewerProtocolPolicy: 'redirect-to-https',
      allowedMethods: ['GET', 'HEAD', 'OPTIONS'],
      cachedMethods: ['GET', 'HEAD'],
      compress: true,
      minTtl: cacheConfig.minTtl,
      defaultTtl: cacheConfig.defaultTtl,
      maxTtl: cacheConfig.maxTtl,
      forwardedValues: {
        queryString: false,
        cookies: {
          forward: 'none',
        },
      },
    },

    customErrorResponses: [
      {
        errorCode: 403,
        responseCode: 200,
        responsePagePath: '/index.html',
        errorCachingMinTtl: 300,
      },
      {
        errorCode: 404,
        responseCode: 200,
        responsePagePath: '/index.html',
        errorCachingMinTtl: 300,
      },
    ],

    restrictions: {
      geoRestriction: {
        restrictionType: 'none',
      },
    },

    viewerCertificate: config.certificateArn
      ? {
          acmCertificateArn: config.certificateArn,
          sslSupportMethod: 'sni-only',
          minimumProtocolVersion: 'TLSv1.2_2021',
        }
      : {
          cloudfrontDefaultCertificate: true,
        },

    tags: { ...getResourceTags(config.environment, 'cloudfront') },
  };

  // Add custom domain aliases if provided
  if (config.domainAliases && config.domainAliases.length > 0) {
    distributionConfig.aliases = config.domainAliases;
  }

  return new aws.cloudfront.Distribution(`ohi-${config.environment}-frontend`, distributionConfig);
}

/**
 * Create S3 bucket for frontend hosting
 */
export function createFrontendBucket(config: CloudFrontConfig): aws.s3.Bucket {
  return new aws.s3.Bucket(`ohi-${config.environment}-frontend`, {
    bucket: `ohi-${config.environment}-frontend`,
    acl: 'private', // CloudFront will access via OAI
    versioning: {
      enabled: config.environment === 'prod',
    },
    website: {
      indexDocument: 'index.html',
      errorDocument: 'index.html',
    },
    corsRules: [
      {
        allowedHeaders: ['*'],
        allowedMethods: ['GET', 'HEAD'],
        allowedOrigins: ['*'],
        exposeHeaders: ['ETag'],
        maxAgeSeconds: 3000,
      },
    ],
    lifecycleRules:
      config.environment !== 'prod'
        ? [
            {
              enabled: true,
              expiration: {
                days: 30,
              },
            },
          ]
        : [],
    tags: { ...getResourceTags(config.environment, 's3-frontend') },
  });
}

/**
 * Create CloudFront infrastructure
 */
export function createCloudFrontInfrastructure(config: CloudFrontConfig): CloudFrontOutputs {
  // Create S3 bucket
  const bucket = createFrontendBucket(config);

  // Update config with bucket details
  const fullConfig: CloudFrontConfig = {
    ...config,
    s3BucketDomainName: bucket.bucketRegionalDomainName,
    s3BucketArn: bucket.arn,
  };

  // Create Origin Access Identity
  const oai = createOriginAccessIdentity(fullConfig);

  // Create bucket policy
  createS3BucketPolicy(fullConfig, bucket.id, oai);

  // Create CloudFront distribution
  const distribution = createCloudFrontDistribution(fullConfig, oai);

  return {
    distributionId: distribution.id,
    distributionArn: distribution.arn,
    distributionDomainName: distribution.domainName,
    distributionHostedZoneId: distribution.hostedZoneId,
  };
}

/**
 * Get CloudFront cache behaviors for API routes
 */
export function getApiCacheBehavior(pathPattern: string, albDnsName: string): any {
  return {
    pathPattern: pathPattern,
    targetOriginId: 'alb-api',
    viewerProtocolPolicy: 'redirect-to-https',
    allowedMethods: ['DELETE', 'GET', 'HEAD', 'OPTIONS', 'PATCH', 'POST', 'PUT'],
    cachedMethods: ['GET', 'HEAD'],
    compress: true,
    minTtl: 0,
    defaultTtl: 0, // No caching for API
    maxTtl: 0,
    forwardedValues: {
      queryString: true,
      headers: ['Authorization', 'Host', 'CloudFront-Forwarded-Proto'],
      cookies: {
        forward: 'all',
      },
    },
  };
}

/**
 * Get CloudFront logging configuration
 */
export function getLoggingConfig(environment: string, s3BucketName: string): any {
  if (environment !== 'prod') {
    return undefined; // Only enable for prod
  }

  return {
    bucket: `${s3BucketName}.s3.amazonaws.com`,
    prefix: `cloudfront-logs/${environment}/`,
    includeCookies: false,
  };
}

/**
 * Create CloudFront invalidation
 */
export function createInvalidation(
  distributionId: pulumi.Output<string>,
  paths: string[]
): pulumi.Output<string> {
  // This would be used in CI/CD to invalidate cache after deployment
  return distributionId.apply((id) => {
    return `aws cloudfront create-invalidation --distribution-id ${id} --paths ${paths.join(' ')}`;
  });
}
