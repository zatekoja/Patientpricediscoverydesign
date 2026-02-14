import * as pulumi from '@pulumi/pulumi';
import * as aws from '@pulumi/aws';
import { getResourceTags } from '../tagging';

export interface Route53Config {
  environment: string;
  domain: string; // e.g., ateru.ng
  createHostedZone: boolean; // false for prod (use existing), true for staging/dev
  albDnsName?: pulumi.Output<string>;
  albZoneId?: pulumi.Output<string>;
  cloudfrontDnsName?: pulumi.Output<string>;
  cloudfrontZoneId?: pulumi.Output<string>;
}

export interface Route53Outputs {
  hostedZoneId: pulumi.Output<string>;
  hostedZoneName: pulumi.Output<string>;
  nameServers: pulumi.Output<string[]>;
  recordNames: {
    frontend: pulumi.Output<string>;
    api: pulumi.Output<string>;
    graphql: pulumi.Output<string>;
    sse: pulumi.Output<string>;
    provider: pulumi.Output<string>;
  };
}

/**
 * Get or create hosted zone
 */
export function getOrCreateHostedZone(config: Route53Config): aws.route53.Zone {
  if (config.createHostedZone) {
    // Create new hosted zone for staging/dev subdomains
    return new aws.route53.Zone(`ohi-${config.environment}`, {
      name: config.domain,
      comment: `Hosted zone for OHI ${config.environment} environment`,
      tags: { ...getResourceTags(config.environment, 'hosted-zone') },
    });
  } else {
    // For prod, reference existing zone (assumed to be created manually)
    // This would be the root ateru.ng zone
    throw new Error('Production hosted zone should be referenced, not created. Use aws.route53.Zone.get()');
  }
}

/**
 * Get existing hosted zone by domain name
 */
export function getExistingHostedZone(domainName: string): pulumi.Output<aws.route53.GetZoneResult> {
  return pulumi.output(
    aws.route53.getZone({
      name: domainName,
      privateZone: false,
    })
  );
}

/**
 * Create A record (alias) for ALB
 */
export function createAlbAliasRecord(
  config: Route53Config,
  hostedZone: aws.route53.Zone,
  subdomain: string
): aws.route53.Record {
  if (!config.albDnsName || !config.albZoneId) {
    throw new Error('ALB DNS name and zone ID required for alias record');
  }

  return new aws.route53.Record(`ohi-${config.environment}-${subdomain}`, {
    zoneId: hostedZone.zoneId,
    name: `${subdomain}.${config.domain}`,
    type: 'A',
    aliases: [
      {
        name: config.albDnsName,
        zoneId: config.albZoneId,
        evaluateTargetHealth: true,
      },
    ],
  });
}

/**
 * Create A record (alias) for CloudFront
 */
export function createCloudFrontAliasRecord(
  config: Route53Config,
  hostedZone: aws.route53.Zone,
  subdomain: string = '' // Empty for apex domain
): aws.route53.Record {
  if (!config.cloudfrontDnsName || !config.cloudfrontZoneId) {
    throw new Error('CloudFront DNS name and zone ID required for alias record');
  }

  const recordName = subdomain ? `${subdomain}.${config.domain}` : config.domain;

  return new aws.route53.Record(
    `ohi-${config.environment}-cloudfront${subdomain ? `-${subdomain}` : ''}`,
    {
      zoneId: hostedZone.zoneId,
      name: recordName,
      type: 'A',
      aliases: [
        {
          name: config.cloudfrontDnsName,
          zoneId: config.cloudfrontZoneId, // CloudFront zone ID is always Z2FDTNDATAQYW2
          evaluateTargetHealth: false, // CloudFront doesn't support health checks
        },
      ],
    }
  );
}

/**
 * Create CNAME record
 */
export function createCnameRecord(
  config: Route53Config,
  hostedZone: aws.route53.Zone,
  name: string,
  value: string,
  ttl: number = 300
): aws.route53.Record {
  return new aws.route53.Record(`ohi-${config.environment}-cname-${name}`, {
    zoneId: hostedZone.zoneId,
    name: `${name}.${config.domain}`,
    type: 'CNAME',
    ttl: ttl,
    records: [value],
  });
}

/**
 * Create TXT record for domain verification
 */
export function createTxtRecord(
  config: Route53Config,
  hostedZone: aws.route53.Zone,
  name: string,
  value: string
): aws.route53.Record {
  return new aws.route53.Record(`ohi-${config.environment}-txt-${name}`, {
    zoneId: hostedZone.zoneId,
    name: `${name}.${config.domain}`,
    type: 'TXT',
    ttl: 300,
    records: [value],
  });
}

/**
 * Create NS record for subdomain delegation
 */
export function createNsRecord(
  parentZoneId: string,
  subdomain: string,
  nameServers: pulumi.Output<string[]>
): aws.route53.Record {
  return new aws.route53.Record(`ns-delegation-${subdomain}`, {
    zoneId: parentZoneId,
    name: subdomain,
    type: 'NS',
    ttl: 172800, // 2 days
    records: nameServers,
  });
}

/**
 * Get domain name for environment
 */
export function getDomainForEnvironment(environment: string): string {
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
 * Get subdomain prefix for service
 */
export function getServiceSubdomain(service: string): string {
  const subdomainMap: Record<string, string> = {
    api: 'api',
    graphql: 'graphql',
    sse: 'sse',
    'provider-api': 'provider',
    frontend: '', // Apex domain or www
  };

  return subdomainMap[service] || service;
}

/**
 * Create complete Route 53 infrastructure
 */
export function createRoute53Infrastructure(config: Route53Config): Route53Outputs {
  // Get or create hosted zone
  const hostedZone = getOrCreateHostedZone(config);

  const outputs: Route53Outputs = {
    hostedZoneId: hostedZone.zoneId,
    hostedZoneName: hostedZone.name,
    nameServers: hostedZone.nameServers,
    recordNames: {
      frontend: pulumi.output(config.domain),
      api: pulumi.output(`api.${config.domain}`),
      graphql: pulumi.output(`graphql.${config.domain}`),
      sse: pulumi.output(`sse.${config.domain}`),
      provider: pulumi.output(`provider.${config.domain}`),
    },
  };

  // Create CloudFront alias for frontend (apex or subdomain)
  if (config.cloudfrontDnsName && config.cloudfrontZoneId) {
    createCloudFrontAliasRecord(config, hostedZone, '');
  }

  // Create ALB aliases for API services
  if (config.albDnsName && config.albZoneId) {
    createAlbAliasRecord(config, hostedZone, 'api');
    createAlbAliasRecord(config, hostedZone, 'graphql');
    createAlbAliasRecord(config, hostedZone, 'sse');
    createAlbAliasRecord(config, hostedZone, 'provider');
  }

  return outputs;
}

/**
 * Get health check configuration
 */
export function createHealthCheck(
  config: Route53Config,
  fqdn: string,
  path: string = '/health'
): aws.route53.HealthCheck {
  return new aws.route53.HealthCheck(`ohi-${config.environment}-healthcheck-${fqdn}`, {
    fqdn: fqdn,
    port: 443,
    type: 'HTTPS',
    resourcePath: path,
    failureThreshold: 3,
    requestInterval: 30,
    measureLatency: true,
    tags: { ...getResourceTags(config.environment, `healthcheck-${fqdn}`) },
  });
}

/**
 * Get TTL values for different record types
 */
export function getRecordTtl(recordType: string, environment: string): number {
  const ttlMap: Record<string, number> = {
    A: 300, // 5 minutes for alias records (not actually used, but for reference)
    CNAME: 300, // 5 minutes
    TXT: 300, // 5 minutes
    NS: 172800, // 2 days
    MX: 3600, // 1 hour
    SOA: 900, // 15 minutes
  };

  // Lower TTL for dev/staging for faster changes
  if (environment !== 'prod') {
    return Math.min(ttlMap[recordType] || 300, 300);
  }

  return ttlMap[recordType] || 300;
}

/**
 * Validate domain name format
 */
export function validateDomainName(domain: string): boolean {
  const domainRegex = /^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$/;
  return domainRegex.test(domain);
}

/**
 * Get Route 53 zone for environment
 */
export function getZoneIdForEnvironment(environment: string): string {
  // These would be set as Pulumi config or environment variables
  const config = new pulumi.Config();
  const zoneIdKey = `route53ZoneId${environment.charAt(0).toUpperCase() + environment.slice(1)}`;
  return config.require(zoneIdKey);
}
