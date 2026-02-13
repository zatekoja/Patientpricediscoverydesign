/**
 * Security Groups Module
 * 
 * Implements 16 security groups as defined in V1_DEPLOYMENT_ARCHITECTURE.md
 * Section 6: Network Architecture & Security
 * 
 * Implements least privilege access:
 * - ALB accepts internet traffic (HTTP/HTTPS)
 * - Services accept traffic only from ALB or specific service SGs
 * - Databases accept traffic only from application service SGs
 * - No direct internet access to databases
 */

import * as aws from '@pulumi/aws';
import * as pulumi from '@pulumi/pulumi';
import { getResourceTags } from '../tagging';

// Port Configurations
export const ALB_PORTS = {
  http: 80,
  https: 443,
};

export const SERVICE_PORTS = {
  api: 8080,
  graphql: 8081,
  sse: 8082,
  providerApi: 3000,
  blnkApi: 5001,
};

export const DATABASE_PORTS = {
  postgres: 5432,
  redis: 6379,
};

export const OBSERVABILITY_PORTS = {
  clickhouse: 9000,
  otlpGrpc: 4317,
  otlpHttp: 4318,
  signozQuery: 8080,
  signozFrontend: 3301,
};

export const VPC_ENDPOINT_PORTS = {
  https: 443,
};

// Egress rule allowing all outbound traffic
const ALLOW_ALL_EGRESS = [
  {
    fromPort: 0,
    toPort: 0,
    protocol: '-1',
    cidrBlocks: ['0.0.0.0/0'],
    description: 'Allow all outbound traffic',
  },
];

/**
 * Get security group name following convention
 */
export function getSecurityGroupName(environment: string, service: string): string {
  return `ohi-${environment}-${service}-sg`;
}

/**
 * Create ALB Security Group
 * Allows HTTP (80) and HTTPS (443) from internet
 */
export function createAlbSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'alb');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for Application Load Balancer',
    ingress: [
      {
        fromPort: ALB_PORTS.http,
        toPort: ALB_PORTS.http,
        protocol: 'tcp',
        cidrBlocks: ['0.0.0.0/0'],
        description: 'Allow HTTP from internet',
      },
      {
        fromPort: ALB_PORTS.https,
        toPort: ALB_PORTS.https,
        protocol: 'tcp',
        cidrBlocks: ['0.0.0.0/0'],
        description: 'Allow HTTPS from internet',
      },
    ],
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'alb', {
      Name: name,
    }),
  });
}

/**
 * Create API Security Group
 * Allows traffic from ALB on port 8080
 */
export function createApiSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>,
  albSecurityGroupId: pulumi.Input<string>
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'api');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for API service',
    ingress: [
      {
        fromPort: SERVICE_PORTS.api,
        toPort: SERVICE_PORTS.api,
        protocol: 'tcp',
        securityGroups: [albSecurityGroupId],
        description: 'Allow traffic from ALB',
      },
    ],
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'api', {
      Name: name,
    }),
  });
}

/**
 * Create GraphQL Security Group
 * Allows traffic from ALB on port 8081
 */
export function createGraphqlSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>,
  albSecurityGroupId: pulumi.Input<string>
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'graphql');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for GraphQL service',
    ingress: [
      {
        fromPort: SERVICE_PORTS.graphql,
        toPort: SERVICE_PORTS.graphql,
        protocol: 'tcp',
        securityGroups: [albSecurityGroupId],
        description: 'Allow traffic from ALB',
      },
    ],
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'graphql', {
      Name: name,
    }),
  });
}

/**
 * Create SSE Security Group
 * Allows traffic from ALB on port 8082
 */
export function createSseSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>,
  albSecurityGroupId: pulumi.Input<string>
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'sse');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for SSE service',
    ingress: [
      {
        fromPort: SERVICE_PORTS.sse,
        toPort: SERVICE_PORTS.sse,
        protocol: 'tcp',
        securityGroups: [albSecurityGroupId],
        description: 'Allow traffic from ALB',
      },
    ],
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'sse', {
      Name: name,
    }),
  });
}

/**
 * Create Provider API Security Group
 * Allows traffic from ALB on port 3000
 */
export function createProviderApiSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>,
  albSecurityGroupId: pulumi.Input<string>
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'provider-api');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for Provider API service',
    ingress: [
      {
        fromPort: SERVICE_PORTS.providerApi,
        toPort: SERVICE_PORTS.providerApi,
        protocol: 'tcp',
        securityGroups: [albSecurityGroupId],
        description: 'Allow traffic from ALB',
      },
    ],
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'provider-api', {
      Name: name,
    }),
  });
}

/**
 * Create Reindexer Security Group
 * Background job - no inbound HTTP traffic
 */
export function createReindexerSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'reindexer');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for Reindexer service',
    ingress: [], // No inbound traffic - background job
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'reindexer', {
      Name: name,
    }),
  });
}

/**
 * Create Blnk API Security Group
 * Allows traffic from API and GraphQL services on port 5001
 */
export function createBlnkApiSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>,
  sourceSecurityGroupIds: pulumi.Input<string>[]
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'blnk-api');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for Blnk API service',
    ingress: [
      {
        fromPort: SERVICE_PORTS.blnkApi,
        toPort: SERVICE_PORTS.blnkApi,
        protocol: 'tcp',
        securityGroups: sourceSecurityGroupIds,
        description: 'Allow traffic from API services',
      },
    ],
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'blnk-api', {
      Name: name,
    }),
  });
}

/**
 * Create Blnk Worker Security Group
 * Background worker - no inbound HTTP traffic
 */
export function createBlnkWorkerSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'blnk-worker');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for Blnk Worker service',
    ingress: [], // No inbound traffic - background worker
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'blnk-worker', {
      Name: name,
    }),
  });
}

/**
 * Create RDS Security Group
 * Allows PostgreSQL (5432) from application services
 */
export function createRdsSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>,
  sourceSecurityGroupIds: pulumi.Input<string>[]
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'rds');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for RDS PostgreSQL',
    ingress: [
      {
        fromPort: DATABASE_PORTS.postgres,
        toPort: DATABASE_PORTS.postgres,
        protocol: 'tcp',
        securityGroups: sourceSecurityGroupIds,
        description: 'Allow PostgreSQL from application services',
      },
    ],
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'rds', {
      Name: name,
    }),
  });
}

/**
 * Create ElastiCache Security Group
 * Allows Redis (6379) from application services
 */
export function createElastiCacheSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>,
  sourceSecurityGroupIds: pulumi.Input<string>[]
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'elasticache');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for ElastiCache Redis',
    ingress: [
      {
        fromPort: DATABASE_PORTS.redis,
        toPort: DATABASE_PORTS.redis,
        protocol: 'tcp',
        securityGroups: sourceSecurityGroupIds,
        description: 'Allow Redis from application services',
      },
    ],
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'elasticache', {
      Name: name,
    }),
  });
}

/**
 * Create ClickHouse Security Group
 * Allows ClickHouse (9000) from OTEL and SigNoz services
 */
export function createClickHouseSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>,
  sourceSecurityGroupIds: pulumi.Input<string>[]
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'clickhouse');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for ClickHouse',
    ingress: [
      {
        fromPort: OBSERVABILITY_PORTS.clickhouse,
        toPort: OBSERVABILITY_PORTS.clickhouse,
        protocol: 'tcp',
        securityGroups: sourceSecurityGroupIds,
        description: 'Allow ClickHouse from observability services',
      },
    ],
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'clickhouse', {
      Name: name,
    }),
  });
}

/**
 * Create OTEL Collector Security Group
 * Allows OTLP gRPC (4317) and HTTP (4318) from application services
 */
export function createOtelCollectorSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>,
  sourceSecurityGroupIds: pulumi.Input<string>[]
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'otel-collector');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for OTEL Collector',
    ingress: [
      {
        fromPort: OBSERVABILITY_PORTS.otlpGrpc,
        toPort: OBSERVABILITY_PORTS.otlpGrpc,
        protocol: 'tcp',
        securityGroups: sourceSecurityGroupIds,
        description: 'Allow OTLP gRPC from services',
      },
      {
        fromPort: OBSERVABILITY_PORTS.otlpHttp,
        toPort: OBSERVABILITY_PORTS.otlpHttp,
        protocol: 'tcp',
        securityGroups: sourceSecurityGroupIds,
        description: 'Allow OTLP HTTP from services',
      },
    ],
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'otel-collector', {
      Name: name,
    }),
  });
}

/**
 * Create SigNoz Query Service Security Group
 * Allows port 8080 from SigNoz Frontend
 */
export function createSigNozQuerySecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>,
  signozFrontendSecurityGroupId: pulumi.Input<string>
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'signoz-query');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for SigNoz Query Service',
    ingress: [
      {
        fromPort: OBSERVABILITY_PORTS.signozQuery,
        toPort: OBSERVABILITY_PORTS.signozQuery,
        protocol: 'tcp',
        securityGroups: [signozFrontendSecurityGroupId],
        description: 'Allow traffic from SigNoz Frontend',
      },
    ],
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'signoz-query', {
      Name: name,
    }),
  });
}

/**
 * Create SigNoz Frontend Security Group
 * Allows traffic from ALB on port 3301
 */
export function createSigNozFrontendSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>,
  albSecurityGroupId: pulumi.Input<string>
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'signoz-frontend');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for SigNoz Frontend',
    ingress: [
      {
        fromPort: OBSERVABILITY_PORTS.signozFrontend,
        toPort: OBSERVABILITY_PORTS.signozFrontend,
        protocol: 'tcp',
        securityGroups: [albSecurityGroupId],
        description: 'Allow traffic from ALB',
      },
    ],
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'signoz-frontend', {
      Name: name,
    }),
  });
}

/**
 * Create ECS Tasks Security Group
 * General security group for ECS tasks not covered by specific SGs
 */
export function createEcsTasksSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'ecs-tasks');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for ECS tasks',
    ingress: [], // Specific ingress rules added per service
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'ecs-tasks', {
      Name: name,
    }),
  });
}

/**
 * Create VPC Endpoints Security Group
 * Allows HTTPS (443) from VPC CIDR
 */
export function createVpcEndpointsSecurityGroup(
  environment: string,
  vpcId: pulumi.Input<string>,
  vpcCidr: string
): aws.ec2.SecurityGroup {
  const name = getSecurityGroupName(environment, 'vpc-endpoints');

  return new aws.ec2.SecurityGroup(name, {
    vpcId,
    description: 'Security group for VPC Endpoints',
    ingress: [
      {
        fromPort: VPC_ENDPOINT_PORTS.https,
        toPort: VPC_ENDPOINT_PORTS.https,
        protocol: 'tcp',
        cidrBlocks: [vpcCidr],
        description: 'Allow HTTPS from VPC',
      },
    ],
    egress: ALLOW_ALL_EGRESS,
    tags: getResourceTags(environment, 'vpc-endpoints', {
      Name: name,
    }),
  });
}
