import * as pulumi from '@pulumi/pulumi';
import * as aws from '@pulumi/aws';
import { getResourceTags } from '../tagging';

export interface AcmConfig {
  environment: string;
  domain: string; // e.g., *.ohealth-ng.com
  subjectAlternativeNames?: string[]; // e.g., ['ohealth-ng.com']
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

// ---------------------------------------------------------------------------
// Provider cache — prevents duplicate Pulumi resource names when the same
// region is referenced more than once (cert + validation both need us-east-1).
// ---------------------------------------------------------------------------
const providerCache: Record<string, aws.Provider> = {};

function getAwsProvider(region: string): aws.Provider {
  if (!providerCache[region]) {
    providerCache[region] = new aws.Provider(`provider-${region}`, {
      region: region as aws.Region,
    });
  }
  return providerCache[region];
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
 * Create DNS validation record in Route 53.
 *
 * For a wildcard cert with SAN (e.g. *.ohealth-ng.com + ohealth-ng.com)
 * AWS returns ONE validation record (both names share the same CNAME).
 * We only use the first entry to avoid duplicate Route 53 records.
 */
export function createValidationRecords(
  environment: string,
  certificate: aws.acm.Certificate,
  hostedZoneId: pulumi.Output<string>,
  suffix: string = ''
): aws.route53.Record {
  const validationRecord = new aws.route53.Record(
    `ohi-${environment}-cert-validation-record${suffix}`,
    {
      zoneId: hostedZoneId,
      name: certificate.domainValidationOptions.apply(
        (opts) => opts[0].resourceRecordName
      ),
      type: certificate.domainValidationOptions.apply(
        (opts) => opts[0].resourceRecordType
      ),
      records: [
        certificate.domainValidationOptions.apply(
          (opts) => opts[0].resourceRecordValue
        ),
      ],
      ttl: 60,
      allowOverwrite: true,
    },
  );

  return validationRecord;
}

/**
 * Create certificate validation (waits for cert to become ISSUED)
 */
export function createCertificateValidation(
  environment: string,
  certificate: aws.acm.Certificate,
  validationRecord: aws.route53.Record,
  region?: string,
  suffix: string = ''
): aws.acm.CertificateValidation {
  return new aws.acm.CertificateValidation(
    `ohi-${environment}-cert-validation${suffix}`,
    {
      certificateArn: certificate.arn,
      validationRecordFqdns: [validationRecord.fqdn],
    },
    region ? { provider: getAwsProvider(region) } : undefined
  );
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
 * Create ACM infrastructure for ALB and CloudFront.
 *
 * When a Route 53 hosted zone ID is supplied the function creates DNS
 * validation records and CertificateValidation resources that wait for
 * the certificates to become ISSUED before returning the ARNs.
 *
 * When no zone ID is supplied the raw (unvalidated) ARNs are returned
 * — useful for testing only.
 */
export interface CertificateOutputs {
  albCertificateArn: pulumi.Output<string>;
  cloudfrontCertificateArn: pulumi.Output<string>;
}

export function createAcmInfrastructure(
  environment: string,
  domain: string,
  hostedZoneId?: pulumi.Output<string>
): CertificateOutputs {
  // Request certificate for ALB (eu-west-1)
  const albCert = requestWildcardCertificate(environment, domain, 'eu-west-1');

  // Request certificate for CloudFront (us-east-1 — required by CloudFront)
  const cloudfrontCert = requestWildcardCertificate(environment, domain, 'us-east-1');

  if (hostedZoneId) {
    // --- ALB cert DNS validation (eu-west-1) --------------------------------
    const albValidationRecord = createValidationRecords(
      environment, albCert, hostedZoneId, ''
    );
    const albCertValidation = createCertificateValidation(
      environment, albCert, albValidationRecord, 'eu-west-1', ''
    );

    // --- CloudFront cert DNS validation (us-east-1) -------------------------
    const cfValidationRecord = createValidationRecords(
      environment, cloudfrontCert, hostedZoneId, '-cloudfront'
    );
    const cfCertValidation = createCertificateValidation(
      environment, cloudfrontCert, cfValidationRecord, 'us-east-1', '-cloudfront'
    );

    return {
      albCertificateArn: albCertValidation.certificateArn,
      cloudfrontCertificateArn: cfCertValidation.certificateArn,
    };
  }

  // No hosted zone — return raw ARNs (unvalidated, for testing only)
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
Add these CNAME records to your DNS provider:
${instructions}
Note: It may take up to 30 minutes for validation to complete after adding records.
`;
  });
}

/**
 * Check if certificate is issued
 */
export function isCertificateIssued(
  certificate: aws.acm.Certificate
): pulumi.Output<boolean> {
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

export function getCertificateDetails(
  certificate: aws.acm.Certificate
): CertificateDetails {
  return {
    arn: certificate.arn,
    domain: certificate.domainName,
    status: certificate.status,
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
    cloudfrontRegion: 'us-east-1',
  };
}

function getDomainForEnvironment(environment: string): string {
  switch (environment) {
    case 'prod':
      return 'ohealth-ng.com';
    case 'staging':
      return 'staging.ohealth-ng.com';
    case 'dev':
      return 'dev.ohealth-ng.com';
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
