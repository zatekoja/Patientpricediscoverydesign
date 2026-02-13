/**
 * Configuration Module
 * 
 * Centralized configuration management for Pulumi stacks
 */

import * as pulumi from '@pulumi/pulumi';

const config = new pulumi.Config('ohi');
const awsConfig = new pulumi.Config('aws');

export interface StackConfig {
  // AWS
  region: string;
  
  // Project
  projectName: string;
  environment: string;
  
  // VPC
  vpcCidr: string;
  
  // General
  enableDeletionProtection: boolean;
  
  // RDS
  rdsInstanceClass: string;
  rdsAllocatedStorage: number;
  
  // ElastiCache
  redisNodeType: string;
  
  // ECS
  ecsTaskCpu: string;
  ecsTaskMemory: string;
  enableSpotInstances: boolean;
  
  // DNS
  domainName: string;
}

/**
 * Get stack configuration with defaults
 */
export function getConfig(): StackConfig {
  return {
    region: awsConfig.require('region'),
    projectName: config.get('projectName') || 'open-health-initiative',
    environment: config.require('environment'),
    vpcCidr: config.get('vpcCidr') || '10.0.0.0/16',
    enableDeletionProtection: config.getBoolean('enableDeletionProtection') || false,
    rdsInstanceClass: config.get('rdsInstanceClass') || 'db.t4g.micro',
    rdsAllocatedStorage: config.getNumber('rdsAllocatedStorage') || 20,
    redisNodeType: config.get('redisNodeType') || 'cache.t4g.micro',
    ecsTaskCpu: config.get('ecsTaskCpu') || '256',
    ecsTaskMemory: config.get('ecsTaskMemory') || '512',
    enableSpotInstances: config.getBoolean('enableSpotInstances') || false,
    domainName: config.get('domainName') || 'dev.ohealth-ng.com',
  };
}

/**
 * Get environment-specific VPC CIDR
 */
export function getVpcCidrForEnvironment(environment: string): string {
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
 * Check if running in production
 */
export function isProduction(): boolean {
  return getConfig().environment === 'prod';
}

/**
 * Check if running in development
 */
export function isDevelopment(): boolean {
  return getConfig().environment === 'dev';
}
