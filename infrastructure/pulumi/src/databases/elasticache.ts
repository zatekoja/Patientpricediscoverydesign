/**
 * ElastiCache Redis Module
 * 
 * Implements ElastiCache Redis clusters as defined in V1_DEPLOYMENT_ARCHITECTURE.md
 * 
 * Features:
 * - Redis 7.0 engine
 * - Application cache cluster (general caching)
 * - Blnk cache cluster (financial transactions)
 * - Encryption at rest and in transit
 * - Automatic failover (prod only)
 * - cache.t4g.small (prod), cache.t4g.micro (dev/staging)
 */

import * as aws from '@pulumi/aws';
import * as pulumi from '@pulumi/pulumi';
import * as random from '@pulumi/random';
import { getResourceTags, getDatabaseTags } from '../tagging';

// ElastiCache Configuration
export const ELASTICACHE_CONFIG = {
  engine: 'redis',
  engineVersion: '7.0',
  port: 6379,
  atRestEncryptionEnabled: true,
  transitEncryptionEnabled: true,
  authTokenEnabled: true,
  snapshotRetentionLimit: 5, // days
  snapshotWindow: '03:00-05:00', // UTC
  maintenanceWindow: 'sun:05:00-sun:07:00', // UTC
  autoMinorVersionUpgrade: true,
};

// ElastiCache Parameters for Redis 7
export const ELASTICACHE_PARAMETERS = [
  {
    name: 'maxmemory-policy',
    value: 'allkeys-lru', // Evict least recently used keys
  },
  {
    name: 'timeout',
    value: '300', // Close idle connections after 5 minutes
  },
];

/**
 * Get ElastiCache node type for environment
 */
export function getElastiCacheNodeType(environment: string): string {
  switch (environment) {
    case 'dev':
      return 'cache.t4g.micro';
    case 'staging':
      return 'cache.t4g.micro';
    case 'prod':
      return 'cache.t4g.small';
    default:
      throw new Error(`Unknown environment: ${environment}`);
  }
}

/**
 * Should enable automatic failover
 */
export function shouldEnableAutomaticFailover(environment: string): boolean {
  return environment === 'prod';
}

/**
 * Get number of cache nodes
 */
export function getNumCacheNodes(environment: string): number {
  // Prod has 2 nodes (primary + replica), dev/staging have 1
  return environment === 'prod' ? 2 : 1;
}

/**
 * Get cache cluster identifier
 */
export function getCacheClusterIdentifier(environment: string, purpose: string): string {
  return `ohi-${environment}-${purpose}-cache`;
}

/**
 * Get cache subnet group name
 */
export function getCacheSubnetGroupName(environment: string): string {
  return `ohi-${environment}-cache-subnet-group`;
}

/**
 * Create ElastiCache auth token secret
 */
export function createElastiCacheAuthTokenSecret(environment: string): aws.secretsmanager.Secret {
  const name = `ohi-${environment}-elasticache-auth-token`;

  const secret = new aws.secretsmanager.Secret(name, {
    name,
    description: 'Auth token for ElastiCache Redis',
    tags: getResourceTags(environment, 'elasticache', {
      Name: name,
    }),
  });

  // Generate random auth token
  const token = new random.RandomPassword(`${name}-value`, {
    length: 64,
    special: false, // Redis auth token doesn't support special chars
  });

  // Store token in secret
  new aws.secretsmanager.SecretVersion(`${name}-version`, {
    secretId: secret.id,
    secretString: token.result,
  });

  return secret;
}

/**
 * Create cache subnet group
 */
export function createCacheSubnetGroup(
  environment: string,
  subnetIds: pulumi.Input<string>[]
): aws.elasticache.SubnetGroup {
  const name = getCacheSubnetGroupName(environment);

  return new aws.elasticache.SubnetGroup(name, {
    name,
    subnetIds,
    description: 'Subnet group for ElastiCache clusters',
    tags: getResourceTags(environment, 'elasticache', {
      Name: name,
    }),
  });
}

/**
 * Create ElastiCache parameter group
 */
export function createElastiCacheParameterGroup(environment: string): aws.elasticache.ParameterGroup {
  const name = `ohi-${environment}-redis7`;

  return new aws.elasticache.ParameterGroup(name, {
    name,
    family: 'redis7',
    description: 'Parameter group for Redis 7',
    parameters: ELASTICACHE_PARAMETERS,
    tags: getResourceTags(environment, 'elasticache', {
      Name: name,
    }),
  });
}

/**
 * Create application cache cluster
 */
export function createApplicationCacheCluster(
  environment: string,
  subnetGroupName: pulumi.Input<string>,
  securityGroupId: pulumi.Input<string>,
  parameterGroupName: pulumi.Input<string>
): aws.elasticache.ReplicationGroup {
  const identifier = getCacheClusterIdentifier(environment, 'app');
  const authToken = createElastiCacheAuthTokenSecret(environment);
  const numNodes = getNumCacheNodes(environment);

  // Generate auth token using random provider
  const token = new random.RandomPassword(`${identifier}-token`, {
    length: 64,
    special: false,
  });

  return new aws.elasticache.ReplicationGroup(identifier, {
    replicationGroupId: identifier,
    description: 'Application cache cluster',
    engine: ELASTICACHE_CONFIG.engine,
    engineVersion: ELASTICACHE_CONFIG.engineVersion,
    nodeType: getElastiCacheNodeType(environment),
    port: ELASTICACHE_CONFIG.port,
    
    // Cluster configuration
    numCacheClusters: numNodes,
    automaticFailoverEnabled: shouldEnableAutomaticFailover(environment),
    
    // Network
    subnetGroupName,
    securityGroupIds: [securityGroupId],
    
    // Parameters
    parameterGroupName,
    
    // Security
    atRestEncryptionEnabled: ELASTICACHE_CONFIG.atRestEncryptionEnabled,
    transitEncryptionEnabled: ELASTICACHE_CONFIG.transitEncryptionEnabled,
    authToken: token.result,
    
    // Backup
    snapshotRetentionLimit: ELASTICACHE_CONFIG.snapshotRetentionLimit,
    snapshotWindow: ELASTICACHE_CONFIG.snapshotWindow,
    
    // Maintenance
    maintenanceWindow: ELASTICACHE_CONFIG.maintenanceWindow,
    autoMinorVersionUpgrade: ELASTICACHE_CONFIG.autoMinorVersionUpgrade,
    
    // Notifications
    notificationTopicArn: undefined, // TODO: Add SNS topic for notifications
    
    // Tags
    tags: {
      ...getDatabaseTags(environment, 'elasticache'),
      Name: identifier,
      Purpose: 'application-cache',
    },
  });
}

/**
 * Create Blnk cache cluster (isolated for financial transactions)
 */
export function createBlnkCacheCluster(
  environment: string,
  subnetGroupName: pulumi.Input<string>,
  securityGroupId: pulumi.Input<string>,
  parameterGroupName: pulumi.Input<string>
): aws.elasticache.ReplicationGroup {
  const identifier = getCacheClusterIdentifier(environment, 'blnk');
  const authToken = createElastiCacheAuthTokenSecret(`${environment}-blnk`);
  const numNodes = getNumCacheNodes(environment);

  // Generate auth token using random provider
  const token = new random.RandomPassword(`${identifier}-token`, {
    length: 64,
    special: false,
  });

  return new aws.elasticache.ReplicationGroup(identifier, {
    replicationGroupId: identifier,
    description: 'Blnk financial transactions cache cluster',
    engine: ELASTICACHE_CONFIG.engine,
    engineVersion: ELASTICACHE_CONFIG.engineVersion,
    nodeType: getElastiCacheNodeType(environment),
    port: ELASTICACHE_CONFIG.port,
    
    // Cluster configuration
    numCacheClusters: numNodes,
    automaticFailoverEnabled: shouldEnableAutomaticFailover(environment),
    
    // Network
    subnetGroupName,
    securityGroupIds: [securityGroupId],
    
    // Parameters
    parameterGroupName,
    
    // Security
    atRestEncryptionEnabled: ELASTICACHE_CONFIG.atRestEncryptionEnabled,
    transitEncryptionEnabled: ELASTICACHE_CONFIG.transitEncryptionEnabled,
    authToken: token.result,
    
    // Backup
    snapshotRetentionLimit: ELASTICACHE_CONFIG.snapshotRetentionLimit,
    snapshotWindow: ELASTICACHE_CONFIG.snapshotWindow,
    
    // Maintenance
    maintenanceWindow: ELASTICACHE_CONFIG.maintenanceWindow,
    autoMinorVersionUpgrade: ELASTICACHE_CONFIG.autoMinorVersionUpgrade,
    
    // Tags
    tags: {
      ...getDatabaseTags(environment, 'elasticache'),
      Name: identifier,
      Purpose: 'blnk-cache',
    },
  });
}
