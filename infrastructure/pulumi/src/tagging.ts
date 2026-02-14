/**
 * Tagging Strategy Module
 * 
 * Implements super strict tagging for shared AWS account as defined in V1_DEPLOYMENT_ARCHITECTURE.md
 * All resources MUST have 11 required tags for cost allocation and compliance
 */

import * as pulumi from '@pulumi/pulumi';

/**
 * Required tags that MUST be present on all resources
 */
export const requiredTags = {
  Project: 'open-health-initiative',
  Environment: '', // Set per environment
  Owner: 'platform-team',
  CostCenter: 'ohi-infrastructure',
  Service: '', // Set per service
  ManagedBy: 'pulumi',
  CreatedBy: 'pulumi-cli', // Override in CI/CD
  CreatedDate: new Date().toISOString().split('T')[0], // YYYY-MM-DD
  DataClassification: 'internal', // Default, override as needed
  BackupPolicy: 'none', // Default, override for databases
  Compliance: 'gdpr', // Default for EU region
};

/**
 * Valid service tag values for all microservices and infrastructure
 */
export const serviceTags = [
  // Application microservices
  'api',
  'graphql',
  'sse',
  'provider-api',
  'reindexer',
  'blnk-api',
  'blnk-worker',
  'frontend',
  // Infrastructure services
  'postgres-primary',
  'postgres-read-replica',
  'redis-cache',
  'redis-blnk',
  'mongodb-provider',
  'clickhouse',
  'signoz-collector',
  'signoz-query',
  'signoz-frontend',
  // Network
  'alb',
  'vpc',
  'nat-gateway',
];

/**
 * Valid data classification values
 */
export const dataClassifications = [
  'public',
  'internal',
  'confidential',
  'pii',
];

/**
 * Valid backup policy values
 */
export const backupPolicies = [
  'daily',
  'weekly',
  'none',
];

/**
 * Valid compliance tag values
 */
export const complianceTags = [
  'hipaa',
  'gdpr',
  'none',
];

/**
 * Generate resource name following convention: {project}-{environment}-{service}-{resource-type}
 * 
 * @param environment - Environment name (dev, staging, prod)
 * @param service - Service name
 * @param resourceType - Type of resource (e.g., ecs-service, task, alb)
 * @returns Formatted resource name
 */
export function generateResourceName(
  environment: string,
  service: string,
  resourceType: string
): string {
  return `ohi-${environment}-${service}-${resourceType}`;
}

/**
 * Merge custom tags with default tags, giving precedence to default tags for required fields
 * 
 * @param defaultTags - Default tags (including required tags)
 * @param customTags - Custom tags to merge
 * @returns Merged tag object
 */
export function mergeTags(
  defaultTags: Record<string, string>,
  customTags: Record<string, string>
): Record<string, string> {
  // Custom tags first, then override with defaults (defaults win)
  return {
    ...customTags,
    ...defaultTags,
  };
}

/**
 * Create resource transformation function that applies default tags
 * 
 * @param environment - Environment name
 * @param service - Service name (optional, can be overridden per resource)
 * @returns Pulumi ResourceTransformation function
 */
export function applyDefaultTags(
  environment: string,
  service?: string
): pulumi.ResourceTransformation {
  // AWS resource types that do NOT support a top-level 'tags' property
  const untaggableTypes = new Set([
    'aws:cloudfront/originAccessIdentity:OriginAccessIdentity',
    'aws:iam/rolePolicyAttachment:RolePolicyAttachment',
    'aws:iam/instanceProfile:InstanceProfile',
    'aws:ec2/routeTableAssociation:RouteTableAssociation',
    'aws:ec2/route:Route',
    'aws:ec2/securityGroupRule:SecurityGroupRule',
    'aws:ecs/clusterCapacityProviders:ClusterCapacityProviders',
    'aws:rds/parameterGroup:ParameterGroup',
    'aws:elasticache/parameterGroup:ParameterGroup',
    'aws:acm/certificateValidation:CertificateValidation',
    'aws:route53/record:Record',
    'aws:lb/listener:Listener',
    'aws:lb/listenerRule:ListenerRule',
    'aws:appautoscaling/target:Target',
    'aws:appautoscaling/policy:Policy',
    'aws:servicediscovery/service:Service',
  ]);

  return (args: pulumi.ResourceTransformationArgs): pulumi.ResourceTransformationResult => {
    // Only tag AWS resources (skip providers, Pulumi internals, etc.)
    if (!args.type.startsWith('aws:') || untaggableTypes.has(args.type)) {
      return { props: args.props, opts: args.opts };
    }

    const tags = {
      ...requiredTags,
      Environment: environment,
      ...(service && { Service: service }),
    };

    // Merge with existing tags
    if (args.props.tags) {
      args.props.tags = mergeTags(tags, args.props.tags);
    } else {
      args.props.tags = tags;
    }

    return {
      props: args.props,
      opts: args.opts,
    };
  };
}

/**
 * Get tags for a specific resource with custom overrides
 * 
 * @param environment - Environment name
 * @param service - Service name
 * @param overrides - Custom tag overrides
 * @returns Complete tag object
 */
export function getResourceTags(
  environment: string,
  service: string,
  overrides: Partial<Record<string, string>> = {}
): Record<string, string> {
  return {
    ...requiredTags,
    Environment: environment,
    Service: service,
    ...overrides,
  };
}

/**
 * Validate that all required tags are present
 * 
 * @param tags - Tags object to validate
 * @throws Error if required tags are missing
 */
export function validateTags(tags: Record<string, string>): void {
  const requiredKeys = Object.keys(requiredTags);
  const missingTags = requiredKeys.filter((key) => !tags[key]);

  if (missingTags.length > 0) {
    throw new Error(
      `Missing required tags: ${missingTags.join(', ')}`
    );
  }
}

/**
 * Get database-specific tags (daily backup, PII classification)
 */
export function getDatabaseTags(
  environment: string,
  service: string
): Record<string, string> {
  return getResourceTags(environment, service, {
    DataClassification: 'pii',
    BackupPolicy: 'daily',
    Compliance: 'gdpr',
  });
}

/**
 * Get public-facing resource tags (frontend, CDN)
 */
export function getPublicTags(
  environment: string,
  service: string
): Record<string, string> {
  return getResourceTags(environment, service, {
    DataClassification: 'public',
    BackupPolicy: 'none',
  });
}
