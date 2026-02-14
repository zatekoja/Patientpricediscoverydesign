import * as pulumi from '@pulumi/pulumi';
import * as aws from '@pulumi/aws';
import { getResourceTags } from '../tagging';

export interface AcmConfig {
  environment: string;
  domain: string; // e.g., *.ateru.ng
  subjectAlternativeNames?: string[]; // e.g., ['ateru.ng']
  region?: string; // eu-west-1 for ALB, us-east-1 for CloudFront
  validationMethod?: 'DNS' | 'EMAIL';
  hostedZoneId?: string; // For automatic DNS validation
}

export interface AcmOutputs {
  certificateArn: pulumi.Output<string>;
  certificateDomainName: pulumi.Output<string>;
  certificateStatus: pulumi.Output<string>;
  validationRecords?: pulumi.Output<
    Array<{
      name: string;
      type: string;
      value: string;
    }>
  >;
}

/**
 * Request ACM certificate
 */
export function requestCertificate(config: AcmConfig): aws.acm.Certificate {
  return new aws.acm.Certificate(
    `ohi-${config.environment}-cert${config.region === 'us-east-1' ? '-cloudfront' : ''}`,
    {
      domainName: config.domain,
      subjectAlternativeNames: config.subjectAlternativeNames,
      validationMethod: config.validationMethod || 'DNS',
      tags: {
        ...getResourceTags(config.environment, `acm-cert${config.region === 'us-east-1' ? '-cloudfront' : ''}`),
      },
    },
    config.region ? { provider: getAwsProvider(config.region) } : undefined
  );
}

/**
 * Create DNS validation records in Route 53
 */
export function createValidationRecords(
  config: AcmConfig,
  certificate: aws.acm.Certificate
): aws.route53.Record[] {
  if (!config.hostedZoneId) {
    throw new Error('Hosted zone ID required for automatic DNS validation');
  }

  // Get validation options from certificate
  const validationOptions = certificate.domainValidationOptions;

  const records: aws.route53.Record[] = [];

  // Create validation records for each domain
  // Note: This is tricky with Pulumi because domainValidationOptions is an Output<array>
  // In practice, you'd iterate over the validation options
  // For now, we'll create a helper function

  return records;
}

/**
 * Create certificate validation
 */
export function createCertificateValidation(
  config: AcmConfig,
  certificate: aws.acm.Certificate,
  validationRecords: aws.route53.Record[]
): aws.acm.CertificateValidation {
  return new aws.acm.CertificateValidation(
    `ohi-${config.environment}-cert-validation${config.region === 'us-east-1' ? '-cloudfront' : ''}`,
    {
      certificateArn: certificate.arn,
      validationRecordFqdns: validationRecords.map((record) => record.fqdn),
    },
    config.region ? { provider: getAwsProvider(config.region) } : undefined
  );
}

/**
 * Get AWS provider for specific region
 */
function getAwsProvider(region: string): aws.Provider {
  return new aws.Provider(`provider-${region}`, {
    region: region as aws.Region,
  });
}

/**
 * Request wildcard certificate
 */
export function requestWildcardCertificate(
  environment: string,
  baseDomain: string,
  region: string = 'eu-west-1'
): aws.acm.Certificate {
  const config: AcmConfig = {
    environment,
    domain: `*.${baseDomain}`,
    subjectAlternativeNames: [baseDomain],
    region,
    validationMethod: 'DNS',
  };

  return requestCertificate(config);
}

/**
 * Get certificate ARN from existing certificate
 */
export function getExistingCertificateArn(
  domain: string,
  region: string = 'eu-west-1'
): pulumi.Output<string> {
  return pulumi.output(
    aws.acm.getCertificate(
      {
        domain: domain,
        statuses: ['ISSUED'],
        mostRecent: true,
      },
      { provider: getAwsProvider(region) }
    )
  ).apply((cert) => cert.arn);
}

/**
 * Create ACM infrastructure for ALB and CloudFront
 */
export interface CertificateOutputs {
  albCertificateArn: pulumi.Output<string>;
  cloudfrontCertificateArn: pulumi.Output<string>;
}

export function createAcmInfrastructure(
  environment: string,
  domain: string
): CertificateOutputs {
  // Request certificate for ALB (eu-west-1)
  const albCert = requestWildcardCertificate(environment, domain, 'eu-west-1');

  // Request certificate for CloudFront (us-east-1 - required by CloudFront)
  const cloudfrontCert = requestWildcardCertificate(environment, domain, 'us-east-1');

  return {
    albCertificateArn: albCert.arn,
    cloudfrontCertificateArn: cloudfrontCert.arn,
  };
}

/**
 * Get validation instructions for manual DNS setup
 */
export function getValidationInstructions(
  certificate: aws.acm.Certificate
): pulumi.Output<string> {
  return certificate.domainValidationOptions.apply((options) => {
    if (!options || options.length === 0) {
      return 'No validation options available yet. Check AWS Console for DNS records.';
    }

    const instructions = options
      .map((option, index) => {
        return `
Domain ${index + 1}: ${option.domainName}
  Record Type: CNAME
  Record Name: ${option.resourceRecordName}
  Record Value: ${option.resourceRecordValue}
`;
      })
      .join('\n');

    return `
Add these CNAME records to your DNS provider (Squarespace):
${instructions}
Note: It may take up to 30 minutes for validation to complete after adding records.
`;
  });
}

/**
 * Check if certificate is issued
 */
export function isCertificateIssued(certificate: aws.acm.Certificate): pulumi.Output<boolean> {
  return certificate.status.apply((status) => status === 'ISSUED');
}

/**
 * Get certificate details
 */
export interface CertificateDetails {
  arn: pulumi.Output<string>;
  domain: pulumi.Output<string>;
  status: pulumi.Output<string>;
  issuedAt?: pulumi.Output<string>;
  notBefore?: pulumi.Output<string>;
  notAfter?: pulumi.Output<string>;
  renewalEligibility?: pulumi.Output<string>;
}

export function getCertificateDetails(certificate: aws.acm.Certificate): CertificateDetails {
  return {
    arn: certificate.arn,
    domain: certificate.domainName,
    status: certificate.status,
    // Additional fields would come from aws.acm.getCertificate if needed
  };
}

/**
 * Create certificate for specific subdomain
 */
export function requestSubdomainCertificate(
  environment: string,
  subdomain: string,
  region: string = 'eu-west-1'
): aws.acm.Certificate {
  const config: AcmConfig = {
    environment,
    domain: subdomain,
    region,
    validationMethod: 'DNS',
  };

  return requestCertificate(config);
}

/**
 * Helper to get certificate configuration
 */
export function getCertificateConfig(environment: string): {
  albCertDomain: string;
  cloudfrontCertDomain: string;
  albRegion: string;
  cloudfrontRegion: string;
} {
  const baseDomain = getDomainForEnvironment(environment);

  return {
    albCertDomain: `*.${baseDomain}`,
    cloudfrontCertDomain: `*.${baseDomain}`,
    albRegion: 'eu-west-1',
    cloudfrontRegion: 'us-east-1', // CloudFront requires us-east-1
  };
}

function getDomainForEnvironment(environment: string): string {
  switch (environment) {
    case 'prod':
      return 'ateru.ng';
    case 'staging':
      return 'staging.ateru.ng';
    case 'dev':
      return 'dev.ateru.ng';
    default:
      throw new Error(`Unknown environment: ${environment}`);
  }
}

/**
 * Validate certificate configuration
 */
export function validateCertificateConfig(config: AcmConfig): void {
  if (!config.domain) {
    throw new Error('Domain name is required');
  }

  if (config.domain.includes('cloudfront') && config.region !== 'us-east-1') {
    throw new Error('CloudFront certificates must be in us-east-1 region');
  }

  if (config.validationMethod === 'EMAIL' && !config.domain.includes('@')) {
    console.warn('Email validation requires a valid email address');
  }
}

/**
 * Export certificate ARN to SSM Parameter Store for reference
 */
export function exportCertificateArnToSsm(
  environment: string,
  certificateType: 'alb' | 'cloudfront',
  certificateArn: pulumi.Output<string>
): aws.ssm.Parameter {
  return new aws.ssm.Parameter(`ohi-${environment}-cert-arn-${certificateType}`, {
    name: `/ohi/${environment}/certificates/${certificateType}/arn`,
    type: 'String',
    value: certificateArn,
    description: `ACM certificate ARN for ${certificateType} in ${environment}`,
    tags: { ...getResourceTags(environment, `cert-arn-${certificateType}`) },
  });
}
