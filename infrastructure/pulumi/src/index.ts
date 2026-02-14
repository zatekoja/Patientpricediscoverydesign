/**
 * Open Health Initiative - AWS Infrastructure
 * Main Pulumi Program Entry Point
 * 
 * This infrastructure follows Test-Driven Development (TDD):
 * 1. Write tests first (RED) - Define what infrastructure should do
 * 2. Implement code (GREEN) - Make tests pass
 * 3. Refactor (REFACTOR) - Improve code while keeping tests green
 * 
 * Current Implementation Status:
 * ✅ Tagging Strategy (14/14 tests passing)
 * 🔄 VPC Networking (tests written, implementation pending)
 * 🔄 Security Groups (pending)
 * 🔄 RDS/ElastiCache (pending)
 * 🔄 ECS Cluster & Services (pending)
 * 🔄 ALB & CloudFront (pending)
 */

import * as pulumi from '@pulumi/pulumi';
import * as aws from '@pulumi/aws';
import { getConfig } from './config';
import { applyDefaultTags } from './tagging';

// Load stack configuration
const config = getConfig();

// Register resource transformation for automatic tagging
pulumi.runtime.registerStackTransformation(
  applyDefaultTags(config.environment)
);

// Export configuration for reference
export const stackConfig = {
  region: config.region,
  environment: config.environment,
  projectName: config.projectName,
  vpcCidr: config.vpcCidr,
};

// TODO: Implement VPC networking (next TDD cycle)
// TODO: Implement Security Groups (next TDD cycle)
// TODO: Implement RDS/ElastiCache (next TDD cycle)
// TODO: Implement ECS Cluster (next TDD cycle)

pulumi.log.info(`🚀 Deploying Open Health Initiative to ${config.region} (${config.environment})`);
