/**
 * RDS PostgreSQL Module
 * 
 * Implements RDS PostgreSQL as defined in V1_DEPLOYMENT_ARCHITECTURE.md
 * 
 * Features:
 * - PostgreSQL 15 engine
 * - Multi-AZ deployment (prod only)
 * - Read replicas (2 for prod, 1 for staging, 0 for dev)
 * - Encryption at rest and in transit
 * - Automated backups with 7-day retention
 * - Performance Insights enabled
 * - Enhanced monitoring (60-second interval)
 * - CloudWatch log exports
 */

import * as aws from '@pulumi/aws';
import * as pulumi from '@pulumi/pulumi';
import * as random from '@pulumi/random';
import { getResourceTags, getDatabaseTags } from '../tagging';

// RDS Configuration
export const RDS_CONFIG = {
  engine: 'postgres',
  engineVersion: '15',
  allocatedStorage: 100, // GB
  storageType: 'gp3',
  storageEncrypted: true,
  backupRetentionPeriod: 7, // days
  backupWindow: '03:00-04:00', // UTC
  maintenanceWindow: 'sun:04:00-sun:05:00', // UTC
  performanceInsightsEnabled: true,
  performanceInsightsRetentionPeriod: 7, // days
  monitoringInterval: 60, // seconds
  enabledCloudwatchLogsExports: ['postgresql'],
  publiclyAccessible: false,
  skipFinalSnapshot: false, // Always create final snapshot
};

// RDS Parameters for PostgreSQL 15
export const RDS_PARAMETERS = [
  {
    name: 'max_connections',
    value: '200',
  },
  {
    name: 'shared_buffers',
    value: '{DBInstanceClassMemory/32768}', // 25% of RAM
  },
  {
    name: 'effective_cache_size',
    value: '{DBInstanceClassMemory/16384}', // 75% of RAM
  },
  {
    name: 'log_statement',
    value: 'ddl', // Log all DDL statements
  },
  {
    name: 'log_min_duration_statement',
    value: '1000', // Log queries slower than 1 second
  },
];

/**
 * Get RDS instance class for environment
 */
export function getRdsInstanceClass(environment: string): string {
  switch (environment) {
    case 'dev':
      return 'db.t4g.micro';
    case 'staging':
      return 'db.t4g.small';
    case 'prod':
      return 'db.t4g.medium';
    default:
      throw new Error(`Unknown environment: ${environment}`);
  }
}

/**
 * Should enable Multi-AZ deployment
 */
export function shouldEnableMultiAz(environment: string): boolean {
  return environment === 'prod';
}

/**
 * Should enable deletion protection
 */
export function shouldEnableDeletionProtection(environment: string): boolean {
  return environment === 'prod';
}

/**
 * Get number of read replicas
 */
export function getReadReplicaCount(environment: string): number {
  switch (environment) {
    case 'dev':
      return 0;
    case 'staging':
      return 1;
    case 'prod':
      return 2;
    default:
      return 0;
  }
}

/**
 * Get RDS identifier
 */
export function getRdsIdentifier(environment: string, type: string): string {
  return `ohi-${environment}-postgres-${type}`;
}

/**
 * Get DB subnet group name
 */
export function getDbSubnetGroupName(environment: string): string {
  return `ohi-${environment}-db-subnet-group`;
}

/**
 * Get RDS secret name
 */
export function getRdsSecretName(environment: string): string {
  return `ohi-${environment}-rds-master-password`;
}

/**
 * Create RDS master password secret
 */
export function createRdsPasswordSecret(environment: string): aws.secretsmanager.Secret {
  const name = getRdsSecretName(environment);

  const secret = new aws.secretsmanager.Secret(name, {
    name,
    description: 'Master password for RDS PostgreSQL',
    tags: getResourceTags(environment, 'rds', {
      Name: name,
    }),
  });

  // Generate random password
  const password = new random.RandomPassword(`${name}-value`, {
    length: 32,
    special: true,
    overrideSpecial: '!#$%&*()-_=+[]{}<>:?',
  });

  // Store password in secret
  new aws.secretsmanager.SecretVersion(`${name}-version`, {
    secretId: secret.id,
    secretString: password.result,
  });

  return secret;
}

/**
 * Create DB subnet group
 */
export function createDbSubnetGroup(
  environment: string,
  subnetIds: pulumi.Input<string>[]
): aws.rds.SubnetGroup {
  const name = getDbSubnetGroupName(environment);

  return new aws.rds.SubnetGroup(name, {
    name,
    subnetIds,
    description: 'Subnet group for RDS database instances',
    tags: getResourceTags(environment, 'rds', {
      Name: name,
    }),
  });
}

/**
 * Create RDS parameter group
 */
export function createRdsParameterGroup(environment: string): aws.rds.ParameterGroup {
  const name = `ohi-${environment}-postgres15`;

  return new aws.rds.ParameterGroup(name, {
    name,
    family: 'postgres15',
    description: 'Parameter group for PostgreSQL 15',
    parameters: RDS_PARAMETERS,
    tags: getResourceTags(environment, 'rds', {
      Name: name,
    }),
  });
}

/**
 * Create monitoring IAM role
 */
function createMonitoringRole(environment: string): aws.iam.Role {
  const name = `ohi-${environment}-rds-monitoring-role`;

  const role = new aws.iam.Role(name, {
    name,
    assumeRolePolicy: JSON.stringify({
      Version: '2012-10-17',
      Statement: [
        {
          Action: 'sts:AssumeRole',
          Principal: {
            Service: 'monitoring.rds.amazonaws.com',
          },
          Effect: 'Allow',
        },
      ],
    }),
    tags: getResourceTags(environment, 'rds'),
  });

  new aws.iam.RolePolicyAttachment(`${name}-policy`, {
    role: role.name,
    policyArn: 'arn:aws:iam::aws:policy/service-role/AmazonRDSEnhancedMonitoringRole',
  });

  return role;
}

/**
 * Create RDS primary instance
 */
export function createRdsPrimaryInstance(
  environment: string,
  dbSubnetGroupName: pulumi.Input<string>,
  securityGroupId: pulumi.Input<string>,
  parameterGroupName: pulumi.Input<string>
): aws.rds.Instance {
  const identifier = getRdsIdentifier(environment, 'primary');
  const secret = createRdsPasswordSecret(environment);
  const monitoringRole = createMonitoringRole(environment);

  // Generate password using random provider
  const password = new random.RandomPassword(`${identifier}-password`, {
    length: 32,
    special: true,
    overrideSpecial: '!#$%&*()-_=+[]{}<>:?',
  });

  return new aws.rds.Instance(identifier, {
    identifier,
    engine: RDS_CONFIG.engine,
    engineVersion: RDS_CONFIG.engineVersion,
    instanceClass: getRdsInstanceClass(environment),
    allocatedStorage: RDS_CONFIG.allocatedStorage,
    storageType: RDS_CONFIG.storageType,
    storageEncrypted: RDS_CONFIG.storageEncrypted,
    
    // Database configuration
    dbName: 'ohealth',
    username: 'ohiadmin',
    password: password.result,
    
    // High availability
    multiAz: shouldEnableMultiAz(environment),
    
    // Backup configuration
    backupRetentionPeriod: RDS_CONFIG.backupRetentionPeriod,
    backupWindow: RDS_CONFIG.backupWindow,
    skipFinalSnapshot: RDS_CONFIG.skipFinalSnapshot,
    finalSnapshotIdentifier: `${identifier}-final-snapshot`,
    
    // Maintenance
    maintenanceWindow: RDS_CONFIG.maintenanceWindow,
    autoMinorVersionUpgrade: true,
    
    // Network
    dbSubnetGroupName,
    vpcSecurityGroupIds: [securityGroupId],
    publiclyAccessible: RDS_CONFIG.publiclyAccessible,
    
    // Parameters
    parameterGroupName,
    
    // Monitoring
    performanceInsightsEnabled: RDS_CONFIG.performanceInsightsEnabled,
    performanceInsightsRetentionPeriod: RDS_CONFIG.performanceInsightsRetentionPeriod,
    monitoringInterval: RDS_CONFIG.monitoringInterval,
    monitoringRoleArn: monitoringRole.arn,
    enabledCloudwatchLogsExports: RDS_CONFIG.enabledCloudwatchLogsExports,
    
    // Protection
    deletionProtection: shouldEnableDeletionProtection(environment),
    
    // Tags
    tags: {
      ...getDatabaseTags(environment, 'rds'),
      Name: identifier,
      Role: 'primary',
    },
  });
}

/**
 * Create RDS read replicas
 */
export function createRdsReadReplicas(
  environment: string,
  primaryInstanceId: pulumi.Input<string>,
  securityGroupId: pulumi.Input<string>
): aws.rds.Instance[] {
  const count = getReadReplicaCount(environment);
  const replicas: aws.rds.Instance[] = [];

  for (let i = 1; i <= count; i++) {
    const identifier = getRdsIdentifier(environment, `replica-${i}`);
    const monitoringRole = createMonitoringRole(`${environment}-replica-${i}`);

    const replica = new aws.rds.Instance(identifier, {
      identifier,
      replicateSourceDb: primaryInstanceId,
      instanceClass: getRdsInstanceClass(environment),
      
      // Network
      vpcSecurityGroupIds: [securityGroupId],
      publiclyAccessible: RDS_CONFIG.publiclyAccessible,
      
      // Monitoring
      performanceInsightsEnabled: RDS_CONFIG.performanceInsightsEnabled,
      performanceInsightsRetentionPeriod: RDS_CONFIG.performanceInsightsRetentionPeriod,
      monitoringInterval: RDS_CONFIG.monitoringInterval,
      monitoringRoleArn: monitoringRole.arn,
      
      // Protection
      deletionProtection: shouldEnableDeletionProtection(environment),
      skipFinalSnapshot: true, // Replicas don't need final snapshots
      
      // Tags
      tags: {
        ...getDatabaseTags(environment, 'rds'),
        Name: identifier,
        Role: `read-replica-${i}`,
      },
    });

    replicas.push(replica);
  }

  return replicas;
}
