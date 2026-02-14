/**
 * VPC Networking Module
 * 
 * Implements VPC infrastructure as defined in V1_DEPLOYMENT_ARCHITECTURE.md:
 * - 3-tier subnet architecture (Public, Private, Database)
 * - 3 Availability Zones in eu-west-1
 * - NAT Gateways with Elastic IPs (one per AZ)
 * - Internet Gateway
 * - VPC Flow Logs
 * - VPC Endpoints (S3, ECR, Secrets Manager)
 */

import * as aws from '@pulumi/aws';
import * as pulumi from '@pulumi/pulumi';
import { getResourceTags, generateResourceName } from '../tagging';

// AWS Region Configuration
export const AVAILABILITY_ZONES = ['eu-west-1a', 'eu-west-1b', 'eu-west-1c'];

// VPC Configuration
export const VPC_CONFIG = {
  enableDnsHostnames: true,
  enableDnsSupport: true,
};

// Subnet Configuration
export const PUBLIC_SUBNET_CONFIG = {
  mapPublicIpOnLaunch: true,
};

export const PRIVATE_SUBNET_CONFIG = {
  mapPublicIpOnLaunch: false,
};

export const DATABASE_SUBNET_CONFIG = {
  mapPublicIpOnLaunch: false,
};

// Route Configuration
export const PUBLIC_ROUTE_CONFIG = {
  destinationCidrBlock: '0.0.0.0/0',
};

export const PRIVATE_ROUTE_CONFIG = {
  destinationCidrBlock: '0.0.0.0/0',
};

export const DATABASE_ROUTE_CONFIG = {
  hasInternetRoute: false,
};

// VPC Flow Logs Configuration
export const VPC_FLOW_LOGS_CONFIG = {
  destination: 'cloud-watch-logs',
  trafficType: 'ALL',
};

// VPC Endpoints
export const REQUIRED_INTERFACE_ENDPOINTS = [
  'ecr.api',
  'ecr.dkr',
  'secretsmanager',
];

// Environment Isolation
export const VPC_PEERING_ENABLED = false;
export const USE_CUSTOM_NACLS = false;

/**
 * Get VPC CIDR block for environment
 */
export function getVpcCidr(environment: string): string {
  switch (environment) {
    case 'dev':
      return '10.0.0.0/16';
    case 'staging':
      return '10.1.0.0/16';
    case 'prod':
      return '10.2.0.0/16';
    default:
      throw new Error(`Unknown environment: ${environment}`);
  }
}

/**
 * Get public subnet CIDR blocks
 */
export function getPublicSubnetCidrs(environment: string): string[] {
  const base = getVpcCidr(environment).split('.')[1];
  return [
    `10.${base}.0.0/24`,
    `10.${base}.1.0/24`,
    `10.${base}.2.0/24`,
  ];
}

/**
 * Get private subnet CIDR blocks
 */
export function getPrivateSubnetCidrs(environment: string): string[] {
  const base = getVpcCidr(environment).split('.')[1];
  return [
    `10.${base}.10.0/24`,
    `10.${base}.11.0/24`,
    `10.${base}.12.0/24`,
  ];
}

/**
 * Get database subnet CIDR blocks
 */
export function getDatabaseSubnetCidrs(environment: string): string[] {
  const base = getVpcCidr(environment).split('.')[1];
  return [
    `10.${base}.20.0/24`,
    `10.${base}.21.0/24`,
    `10.${base}.22.0/24`,
  ];
}

/**
 * Get subnet name
 */
export function getSubnetName(environment: string, type: string, index: number): string {
  const azSuffix = ['a', 'b', 'c'][index];
  return generateResourceName(environment, type, `subnet-${azSuffix}`);
}

/**
 * Get Internet Gateway name
 */
export function getIgwName(environment: string): string {
  return `ohi-${environment}-igw`;
}

/**
 * Get NAT Gateway name
 */
export function getNatGatewayName(environment: string, index: number): string {
  const azSuffix = ['a', 'b', 'c'][index];
  return generateResourceName(environment, 'nat', azSuffix);
}

/**
 * Create VPC
 */
export function createVpc(environment: string): aws.ec2.Vpc {
  const vpcCidr = getVpcCidr(environment);
  const name = generateResourceName(environment, 'vpc', 'main');

  return new aws.ec2.Vpc(name, {
    cidrBlock: vpcCidr,
    enableDnsHostnames: VPC_CONFIG.enableDnsHostnames,
    enableDnsSupport: VPC_CONFIG.enableDnsSupport,
    tags: getResourceTags(environment, 'vpc', {
      Name: name,
    }),
  });
}

/**
 * Create public subnets
 */
export function createPublicSubnets(environment: string, vpcId: pulumi.Input<string>): aws.ec2.Subnet[] {
  const cidrs = getPublicSubnetCidrs(environment);
  
  return cidrs.map((cidr, index) => {
    const name = getSubnetName(environment, 'public', index);
    
    return new aws.ec2.Subnet(name, {
      vpcId,
      cidrBlock: cidr,
      availabilityZone: AVAILABILITY_ZONES[index],
      mapPublicIpOnLaunch: PUBLIC_SUBNET_CONFIG.mapPublicIpOnLaunch,
      tags: getResourceTags(environment, 'vpc', {
        Name: name,
        Type: 'public',
      }),
    });
  });
}

/**
 * Create private subnets
 */
export function createPrivateSubnets(environment: string, vpcId: pulumi.Input<string>): aws.ec2.Subnet[] {
  const cidrs = getPrivateSubnetCidrs(environment);
  
  return cidrs.map((cidr, index) => {
    const name = getSubnetName(environment, 'private', index);
    
    return new aws.ec2.Subnet(name, {
      vpcId,
      cidrBlock: cidr,
      availabilityZone: AVAILABILITY_ZONES[index],
      mapPublicIpOnLaunch: PRIVATE_SUBNET_CONFIG.mapPublicIpOnLaunch,
      tags: getResourceTags(environment, 'vpc', {
        Name: name,
        Type: 'private',
      }),
    });
  });
}

/**
 * Create database subnets
 */
export function createDatabaseSubnets(environment: string, vpcId: pulumi.Input<string>): aws.ec2.Subnet[] {
  const cidrs = getDatabaseSubnetCidrs(environment);
  
  return cidrs.map((cidr, index) => {
    const name = getSubnetName(environment, 'database', index);
    
    return new aws.ec2.Subnet(name, {
      vpcId,
      cidrBlock: cidr,
      availabilityZone: AVAILABILITY_ZONES[index],
      mapPublicIpOnLaunch: DATABASE_SUBNET_CONFIG.mapPublicIpOnLaunch,
      tags: getResourceTags(environment, 'vpc', {
        Name: name,
        Type: 'database',
      }),
    });
  });
}

/**
 * Create Internet Gateway
 */
export function createInternetGateway(environment: string, vpcId: pulumi.Input<string>): aws.ec2.InternetGateway {
  const name = getIgwName(environment);
  
  return new aws.ec2.InternetGateway(name, {
    vpcId,
    tags: getResourceTags(environment, 'vpc', {
      Name: name,
    }),
  });
}

/**
 * Create Elastic IPs for NAT Gateways
 */
export function createElasticIps(environment: string): aws.ec2.Eip[] {
  return AVAILABILITY_ZONES.map((_, index) => {
    const name = generateResourceName(environment, 'nat', `eip-${['a', 'b', 'c'][index]}`);
    
    return new aws.ec2.Eip(name, {
      vpc: true,
      tags: getResourceTags(environment, 'nat-gateway', {
        Name: name,
      }),
    });
  });
}

/**
 * Create NAT Gateways
 */
export function createNatGateways(
  environment: string,
  subnetIds: pulumi.Input<string>[]
): aws.ec2.NatGateway[] {
  const eips = createElasticIps(environment);
  
  return subnetIds.map((subnetId, index) => {
    const name = getNatGatewayName(environment, index);
    
    return new aws.ec2.NatGateway(name, {
      subnetId,
      allocationId: eips[index].id,
      tags: getResourceTags(environment, 'nat-gateway', {
        Name: name,
      }),
    });
  });
}

/**
 * Create public route table
 */
export function createPublicRouteTable(
  environment: string,
  vpcId: pulumi.Input<string>,
  igwId: pulumi.Input<string>
): aws.ec2.RouteTable {
  const name = generateResourceName(environment, 'public', 'rt');
  
  const routeTable = new aws.ec2.RouteTable(name, {
    vpcId,
    tags: getResourceTags(environment, 'vpc', {
      Name: name,
      Type: 'public',
    }),
  });

  // Route to Internet Gateway
  new aws.ec2.Route(`${name}-igw-route`, {
    routeTableId: routeTable.id,
    destinationCidrBlock: PUBLIC_ROUTE_CONFIG.destinationCidrBlock,
    gatewayId: igwId,
  });

  return routeTable;
}

/**
 * Create private route tables (one per AZ with NAT Gateway)
 */
export function createPrivateRouteTables(
  environment: string,
  vpcId: pulumi.Input<string>,
  natGatewayIds: pulumi.Input<string>[]
): aws.ec2.RouteTable[] {
  return natGatewayIds.map((natGatewayId, index) => {
    const azSuffix = ['a', 'b', 'c'][index];
    const name = generateResourceName(environment, 'private', `rt-${azSuffix}`);
    
    const routeTable = new aws.ec2.RouteTable(name, {
      vpcId,
      tags: getResourceTags(environment, 'vpc', {
        Name: name,
        Type: 'private',
      }),
    });

    // Route to NAT Gateway
    new aws.ec2.Route(`${name}-nat-route`, {
      routeTableId: routeTable.id,
      destinationCidrBlock: PRIVATE_ROUTE_CONFIG.destinationCidrBlock,
      natGatewayId,
    });

    return routeTable;
  });
}

/**
 * Create database route table (no internet access)
 */
export function createDatabaseRouteTable(
  environment: string,
  vpcId: pulumi.Input<string>
): aws.ec2.RouteTable {
  const name = generateResourceName(environment, 'database', 'rt');
  
  return new aws.ec2.RouteTable(name, {
    vpcId,
    tags: getResourceTags(environment, 'vpc', {
      Name: name,
      Type: 'database',
    }),
  });
  // No routes - completely isolated
}

/**
 * Create VPC Flow Logs
 */
export function createVpcFlowLogs(
  environment: string,
  vpcId: pulumi.Input<string>
): aws.ec2.FlowLog {
  const name = generateResourceName(environment, 'vpc', 'flow-logs');
  
  // Create CloudWatch Log Group
  const logGroup = new aws.cloudwatch.LogGroup(`${name}-log-group`, {
    namePrefix: `/aws/vpc/${environment}/`,
    retentionInDays: 30,
    tags: getResourceTags(environment, 'vpc', {
      Name: `${name}-log-group`,
    }),
  });

  // Create IAM role for Flow Logs
  const role = new aws.iam.Role(`${name}-role`, {
    assumeRolePolicy: JSON.stringify({
      Version: '2012-10-17',
      Statement: [{
        Action: 'sts:AssumeRole',
        Principal: {
          Service: 'vpc-flow-logs.amazonaws.com',
        },
        Effect: 'Allow',
      }],
    }),
    tags: getResourceTags(environment, 'vpc'),
  });

  // Attach policy to role
  new aws.iam.RolePolicy(`${name}-policy`, {
    role: role.id,
    policy: JSON.stringify({
      Version: '2012-10-17',
      Statement: [{
        Effect: 'Allow',
        Action: [
          'logs:CreateLogGroup',
          'logs:CreateLogStream',
          'logs:PutLogEvents',
          'logs:DescribeLogGroups',
          'logs:DescribeLogStreams',
        ],
        Resource: '*',
      }],
    }),
  });

  return new aws.ec2.FlowLog(name, {
    vpcId,
    trafficType: VPC_FLOW_LOGS_CONFIG.trafficType,
    logDestinationType: VPC_FLOW_LOGS_CONFIG.destination,
    logDestination: logGroup.arn,
    iamRoleArn: role.arn,
    tags: getResourceTags(environment, 'vpc', {
      Name: name,
    }),
  });
}

/**
 * Create S3 Gateway Endpoint
 */
export function createS3Endpoint(
  environment: string,
  vpcId: pulumi.Input<string>,
  routeTableIds: pulumi.Input<string>[]
): aws.ec2.VpcEndpoint {
  const name = generateResourceName(environment, 's3', 'endpoint');
  
  return new aws.ec2.VpcEndpoint(name, {
    vpcId,
    serviceName: 'com.amazonaws.eu-west-1.s3',
    vpcEndpointType: 'Gateway',
    routeTableIds,
    tags: getResourceTags(environment, 'vpc', {
      Name: name,
    }),
  });
}

/**
 * Create Interface VPC Endpoints
 */
export function createInterfaceEndpoints(
  environment: string,
  vpcId: pulumi.Input<string>,
  subnetIds: pulumi.Input<string>[]
): aws.ec2.VpcEndpoint[] {
  const services = [
    'com.amazonaws.eu-west-1.ecr.api',
    'com.amazonaws.eu-west-1.ecr.dkr',
    'com.amazonaws.eu-west-1.secretsmanager',
  ];

  return services.map((serviceName) => {
    const shortName = serviceName.split('.').pop() || 'endpoint';
    const name = generateResourceName(environment, shortName, 'endpoint');
    
    return new aws.ec2.VpcEndpoint(name, {
      vpcId,
      serviceName,
      vpcEndpointType: 'Interface',
      subnetIds,
      privateDnsEnabled: true,
      tags: getResourceTags(environment, name, {
        Name: name,
      }),
    });
  });
}

/**
 * Create Route Table Associations
 */
export function createRouteTableAssociations(
  environment: string,
  publicSubnets: aws.ec2.Subnet[],
  privateSubnets: aws.ec2.Subnet[],
  databaseSubnets: aws.ec2.Subnet[],
  publicRouteTable: aws.ec2.RouteTable,
  privateRouteTables: aws.ec2.RouteTable[],
  databaseRouteTable: aws.ec2.RouteTable
): void {
  // Public Associations
  publicSubnets.forEach((subnet, index) => {
    const name = generateResourceName(environment, 'public', `rta-${index}`);
    new aws.ec2.RouteTableAssociation(name, {
      subnetId: subnet.id,
      routeTableId: publicRouteTable.id,
    });
  });

  // Private Associations
  privateSubnets.forEach((subnet, index) => {
    const name = generateResourceName(environment, 'private', `rta-${index}`);
    new aws.ec2.RouteTableAssociation(name, {
      subnetId: subnet.id,
      routeTableId: privateRouteTables[index].id,
    });
  });

  // Database Associations
  databaseSubnets.forEach((subnet, index) => {
    const name = generateResourceName(environment, 'database', `rta-${index}`);
    new aws.ec2.RouteTableAssociation(name, {
      subnetId: subnet.id,
      routeTableId: databaseRouteTable.id,
    });
  });
}

