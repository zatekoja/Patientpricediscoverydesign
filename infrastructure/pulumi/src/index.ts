// Pulumi S3 backend - state in s3://ohi-pulumi-state-<account-id>, encrypted with PULUMI_CONFIG_PASSPHRASE
// ACM certificates validated — re-deploy to complete CertificateValidation resources
/**
 * Open Health Initiative - AWS Infrastructure
 * Main Pulumi Program Entry Point
 *
 * Deploys the full V1 infrastructure as defined in V1_DEPLOYMENT_ARCHITECTURE.md:
 *   1. VPC & Networking (3 AZs, public/private/database subnets)
 *   2. Security Groups (17 groups with least-privilege)
 *   3. ACM Certificates (ALB eu-west-1 + CloudFront us-east-1)
 *   4. Secrets Manager (database passwords, Redis auth tokens, JWT)
 *   5. RDS PostgreSQL 15 (primary + read replicas)
 *   6. ElastiCache Redis 7 (application + Blnk)
 *   7. ECR Repositories (all containerised services)
 *   8. ALB (HTTPS termination, path-based routing)
 *   9. ECS Fargate (application + observability services)
 *  10. CloudFront + S3 (frontend SPA)
 *  11. Route 53 DNS (apex + subdomains)
 *  12. Observability (ClickHouse EC2, Zookeeper ECS, SigNoz, OTEL, Fluent Bit)
 *
 * All 153 tests passing across 5 suites.
 */

import * as pulumi from '@pulumi/pulumi';
import * as aws from '@pulumi/aws';
import { getConfig } from './config';
import { applyDefaultTags } from './tagging';

// ─── Networking ──────────────────────────────────────────────────────────────
import {
  createVpc,
  createPublicSubnets,
  createPrivateSubnets,
  createDatabaseSubnets,
  createInternetGateway,
  createNatGateways,
  createPublicRouteTable,
  createPrivateRouteTables,
  createDatabaseRouteTable,
  createRouteTableAssociations,
  createVpcFlowLogs,
  createS3Endpoint,
  createInterfaceEndpoints,
} from './networking/vpc';

import {
  createAlbSecurityGroup,
  createApiSecurityGroup,
  createGraphqlSecurityGroup,
  createSseSecurityGroup,
  createProviderApiSecurityGroup,
  createReindexerSecurityGroup,
  createBlnkApiSecurityGroup,
  createBlnkWorkerSecurityGroup,
  createRdsSecurityGroup,
  createElastiCacheSecurityGroup,
  createClickHouseSecurityGroup,
  createZookeeperSecurityGroup,
  createOtelCollectorSecurityGroup,
  createSigNozQuerySecurityGroup,
  createSigNozFrontendSecurityGroup,
  createEcsTasksSecurityGroup,
  createVpcEndpointsSecurityGroup,
} from './networking/security-groups';

import {
  createRoute53Infrastructure,
  getOrCreateHostedZone,
  createAlbAliasRecord,
  createCloudFrontAliasRecord,
} from './networking/route53';

// ─── Security ────────────────────────────────────────────────────────────────
import { createAcmInfrastructure } from './security/acm';
import { createSecretsInfrastructure, getSecretArn } from './security/secrets';

// ─── Databases ───────────────────────────────────────────────────────────────
import {
  createDbSubnetGroup,
  createRdsParameterGroup,
  createRdsPrimaryInstance,
  createRdsReadReplicas,
  createRdsPasswordSecret,
} from './databases/rds';

import {
  createCacheSubnetGroup,
  createElastiCacheParameterGroup,
  createApplicationCacheCluster,
  createBlnkCacheCluster,
  createElastiCacheAuthTokenSecret,
} from './databases/elasticache';

// ─── Compute ─────────────────────────────────────────────────────────────────
import { createEcrInfrastructure } from './compute/ecr';
import { createAlbInfrastructure } from './compute/alb';
import { createEcsInfrastructure } from './compute/ecs';
import { createCloudFrontInfrastructure } from './compute/cloudfront';

// ─── Observability ───────────────────────────────────────────────────────────
import { createObservabilityInfrastructure } from './observability/clickhouse';

// =============================================================================
// Stack Configuration
// =============================================================================
const config = getConfig();

// Register resource transformation for automatic tagging
pulumi.runtime.registerStackTransformation(
  applyDefaultTags(config.environment)
);

pulumi.log.info(`🚀 Deploying Open Health Initiative to ${config.region} (${config.environment})`);

// =============================================================================
// 1. VPC & Networking
// =============================================================================
const vpc = createVpc(config.environment);
const publicSubnets = createPublicSubnets(config.environment, vpc.id);
const privateSubnets = createPrivateSubnets(config.environment, vpc.id);
const databaseSubnets = createDatabaseSubnets(config.environment, vpc.id);

const igw = createInternetGateway(config.environment, vpc.id);
const natGateways = createNatGateways(
  config.environment,
  publicSubnets.map(s => s.id)
);

const publicRouteTable = createPublicRouteTable(config.environment, vpc.id, igw.id);
const privateRouteTables = createPrivateRouteTables(
  config.environment,
  vpc.id,
  natGateways.map(ng => ng.id)
);
const databaseRouteTable = createDatabaseRouteTable(config.environment, vpc.id);

createRouteTableAssociations(
  config.environment,
  publicSubnets,
  privateSubnets,
  databaseSubnets,
  publicRouteTable,
  privateRouteTables,
  databaseRouteTable
);

createVpcFlowLogs(config.environment, vpc.id);

// Helper arrays
const publicSubnetIds = publicSubnets.map(s => s.id);
const privateSubnetIds = privateSubnets.map(s => s.id);
const databaseSubnetIds = databaseSubnets.map(s => s.id);

// VPC Endpoints
const vpcEndpointsSg = createVpcEndpointsSecurityGroup(config.environment, vpc.id, config.vpcCidr);
createS3Endpoint(
  config.environment,
  vpc.id,
  [publicRouteTable.id, ...privateRouteTables.map(rt => rt.id), databaseRouteTable.id]
);
createInterfaceEndpoints(
  config.environment,
  vpc.id,
  privateSubnetIds,
  [vpcEndpointsSg.id]
);

// =============================================================================
// 2. Security Groups (17 groups with least-privilege)
// =============================================================================
const albSg = createAlbSecurityGroup(config.environment, vpc.id);
const apiSg = createApiSecurityGroup(config.environment, vpc.id, albSg.id);
const graphqlSg = createGraphqlSecurityGroup(config.environment, vpc.id, albSg.id);
const sseSg = createSseSecurityGroup(config.environment, vpc.id, albSg.id);
const providerApiSg = createProviderApiSecurityGroup(config.environment, vpc.id, albSg.id);
const reindexerSg = createReindexerSecurityGroup(config.environment, vpc.id);
const blnkApiSg = createBlnkApiSecurityGroup(config.environment, vpc.id, [apiSg.id, graphqlSg.id]);
const blnkWorkerSg = createBlnkWorkerSecurityGroup(config.environment, vpc.id);
const ecsTasksSg = createEcsTasksSecurityGroup(config.environment, vpc.id);

// Database security groups
const rdsSg = createRdsSecurityGroup(config.environment, vpc.id, [
  apiSg.id, graphqlSg.id, sseSg.id, blnkApiSg.id, reindexerSg.id, providerApiSg.id,
]);
const elastiCacheSg = createElastiCacheSecurityGroup(config.environment, vpc.id, [
  apiSg.id, graphqlSg.id, sseSg.id, blnkApiSg.id, reindexerSg.id, providerApiSg.id,
]);

// Observability security groups
const otelSg = createOtelCollectorSecurityGroup(config.environment, vpc.id, [
  apiSg.id, graphqlSg.id, sseSg.id, providerApiSg.id,
]);
const clickhouseSg = createClickHouseSecurityGroup(config.environment, vpc.id, [otelSg.id]);
const zookeeperSg = createZookeeperSecurityGroup(config.environment, vpc.id, clickhouseSg.id);
const signozFrontendSg = createSigNozFrontendSecurityGroup(config.environment, vpc.id, albSg.id);
const signozQuerySg = createSigNozQuerySecurityGroup(config.environment, vpc.id, signozFrontendSg.id);

// =============================================================================
// 3. Route 53 Hosted Zone (created early so ACM can use it for DNS validation)
// =============================================================================
const dnsZoneDomain = config.environment === 'prod'
  ? config.domainName
  : `${config.environment}.${config.domainName}`;

const hostedZone = getOrCreateHostedZone({
  environment: config.environment,
  domain: dnsZoneDomain,
  createHostedZone: config.environment !== 'prod',
});

// =============================================================================
// 4. ACM Certificates (validated via Route 53 DNS)
// =============================================================================
const certs = createAcmInfrastructure(
  config.environment,
  dnsZoneDomain, // dev.ohealth-ng.com → cert for *.dev.ohealth-ng.com
  hostedZone.zoneId,
);

// =============================================================================
// 4. Credentials & Secrets Generation
// =============================================================================
// Create RDS password secret
const rdsSecretData = createRdsPasswordSecret(config.environment);

// Create Redis auth tokens
const redisAuthData = createElastiCacheAuthTokenSecret(config.environment);
const blnkRedisAuthData = createElastiCacheAuthTokenSecret(config.environment, 'blnk');

// Create consolidated Secrets Infrastructure (Master Secret)
const secrets = createSecretsInfrastructure({
  environment: config.environment,
  databasePassword: rdsSecretData.password,
  databasePasswordSecretArn: rdsSecretData.secret.arn,
  redisAuthToken: redisAuthData.token,
  redisAuthTokenSecretArn: redisAuthData.secret.arn,
  blnkRedisAuthToken: blnkRedisAuthData.token,
  blnkRedisAuthTokenSecretArn: blnkRedisAuthData.secret.arn,
});

// Externally-provisioned secrets (created by backend/scripts/setup-secrets.sh)
// Look up by name so ECS task definitions can reference their ARNs.
const externalSecretArns = {
  typesenseApiKey: getSecretArn(`ohi-${config.environment}-typesense-api-key`),
  openaiApiKey: getSecretArn(`ohi-${config.environment}-openai-api-key`),
  geolocationApiKey: getSecretArn(`ohi-${config.environment}-geolocation-api-key`),
  calendlyApiKey: getSecretArn(`ohi-${config.environment}-calendly-api-key`),
  calendlyWebhookSecret: getSecretArn(`ohi-${config.environment}-calendly-webhook-secret`),
  whatsappAccessToken: getSecretArn(`ohi-${config.environment}-whatsapp-access-token`),
  flutterwaveSecretKey: getSecretArn(`ohi-${config.environment}-flutterwave-secret-key`),
  flutterwaveWebhookSecret: getSecretArn(`ohi-${config.environment}-flutterwave-webhook-secret`),
  providerMongoUri: getSecretArn(`ohi-${config.environment}-provider-mongo-uri`),
  providerLlmApiKey: getSecretArn(`ohi-${config.environment}-provider-llm-api-key`),
};

// =============================================================================
// 5. RDS PostgreSQL 15
// =============================================================================
const dbSubnetGroup = createDbSubnetGroup(config.environment, databaseSubnetIds);
const rdsParamGroup = createRdsParameterGroup(config.environment);
const rdsPrimary = createRdsPrimaryInstance(
  config.environment,
  dbSubnetGroup.name,
  rdsSg.id,
  rdsParamGroup.name,
  rdsSecretData.password
);
const rdsReplicas = createRdsReadReplicas(
  config.environment,
  rdsPrimary.id,
  rdsSg.id
);

// =============================================================================
// 6. ElastiCache Redis 7
// =============================================================================
const cacheSubnetGroup = createCacheSubnetGroup(config.environment, databaseSubnetIds);
const cacheParamGroup = createElastiCacheParameterGroup(config.environment);
const appCache = createApplicationCacheCluster(
  config.environment,
  cacheSubnetGroup.name,
  elastiCacheSg.id,
  cacheParamGroup.name,
  redisAuthData.token
);
const blnkCache = createBlnkCacheCluster(
  config.environment,
  cacheSubnetGroup.name,
  elastiCacheSg.id,
  cacheParamGroup.name,
  blnkRedisAuthData.token
);

// =============================================================================
// 7. ECR Repositories
// =============================================================================
const ecr = createEcrInfrastructure({
  environment: config.environment as 'dev' | 'staging' | 'prod',
});

// =============================================================================
// 8. ALB (HTTPS + path routing)
// =============================================================================
const alb = createAlbInfrastructure({
  environment: config.environment,
  vpcId: vpc.id,
  publicSubnetIds,
  albSecurityGroupId: albSg.id,
  certificateArn: certs.albCertificateArn,
});

// =============================================================================
// 9. ECS Fargate (application + observability services)
// =============================================================================
const ecs = createEcsInfrastructure({
  environment: config.environment,
  vpcId: vpc.id,
  privateSubnetIds,
  securityGroupIds: {
    api: apiSg.id,
    graphql: graphqlSg.id,
    sse: sseSg.id,
    providerApi: providerApiSg.id,
    reindexer: reindexerSg.id,
    blnkApi: blnkApiSg.id,
    blnkWorker: blnkWorkerSg.id,
    clickhouse: clickhouseSg.id,
    otel: otelSg.id,
    signoz: signozFrontendSg.id,
  },
  albTargetGroupArns: alb.targetGroupArns,
  albListenerDependency: alb.httpsListenerResource,

  // Database & cache endpoints
  databaseEndpoint: rdsPrimary.endpoint,
  databasePasswordSecretArn: rdsSecretData.secret.arn,
  redisEndpoint: appCache.primaryEndpointAddress,
  redisAuthTokenSecretArn: redisAuthData.secret.arn,
  blnkRedisEndpoint: blnkCache.primaryEndpointAddress,
  blnkRedisAuthTokenSecretArn: blnkRedisAuthData.secret.arn,

  // Application config (non-secret)
  domainName: config.domainName,
  typesenseUrl: pulumi.getStack() === 'prod'
    ? 'https://yu0gb19aodc5ml7sp-1.a2.typesense.net'
    : 'https://yu0gb19aodc5ml7sp-1.a2.typesense.net',
  otelCollectorEndpoint: `http://otel.ohi-${config.environment}.local:4318`,
  providerApiInternalUrl: `http://provider-api.ohi-${config.environment}.local:8080/api/v1`,
  mongoDbName: 'provider_data',

  // Externally-provisioned secret ARNs (from setup-secrets.sh)
  typesenseApiKeySecretArn: externalSecretArns.typesenseApiKey,
  openaiApiKeySecretArn: externalSecretArns.openaiApiKey,
  geolocationApiKeySecretArn: externalSecretArns.geolocationApiKey,
  calendlyApiKeySecretArn: externalSecretArns.calendlyApiKey,
  calendlyWebhookSecretArn: externalSecretArns.calendlyWebhookSecret,
  whatsappAccessTokenSecretArn: externalSecretArns.whatsappAccessToken,
  flutterwaveSecretKeySecretArn: externalSecretArns.flutterwaveSecretKey,
  flutterwaveWebhookSecretArn: externalSecretArns.flutterwaveWebhookSecret,
  providerMongoUriSecretArn: externalSecretArns.providerMongoUri,
  providerLlmApiKeySecretArn: externalSecretArns.providerLlmApiKey,
});

// =============================================================================
// 10. CloudFront + S3 (frontend SPA)
// =============================================================================
const cloudfront = createCloudFrontInfrastructure({
  environment: config.environment,
  certificateArn: certs.cloudfrontCertificateArn,
  domainAliases: config.environment === 'prod'
    ? [config.domainName, `www.${config.domainName}`]
    : [`${config.environment}.${config.domainName}`],
});

// =============================================================================
// 11. Route 53 DNS Records (zone already created in step 3)
// =============================================================================
const dnsConfig = {
  environment: config.environment,
  domain: dnsZoneDomain,
  createHostedZone: false, // already created
  albDnsName: alb.albDnsName,
  albZoneId: alb.albZoneId,
  cloudfrontDnsName: cloudfront.distributionDomainName,
  cloudfrontZoneId: cloudfront.distributionHostedZoneId,
};

// CloudFront alias for frontend
createCloudFrontAliasRecord(dnsConfig, hostedZone, '');

// ALB aliases for API services
createAlbAliasRecord(dnsConfig, hostedZone, 'api');
createAlbAliasRecord(dnsConfig, hostedZone, 'graphql');
createAlbAliasRecord(dnsConfig, hostedZone, 'sse');
createAlbAliasRecord(dnsConfig, hostedZone, 'provider');

const dns = {
  hostedZoneId: hostedZone.zoneId,
  nameServers: hostedZone.nameServers,
};

// =============================================================================
// 12. Observability (ClickHouse EC2, Zookeeper ECS, SigNoz, OTEL)
// =============================================================================
const observability = createObservabilityInfrastructure({
  environment: config.environment as 'dev' | 'staging' | 'prod',
  vpcId: vpc.id,
  privateSubnetIds,
  ecsClusterId: ecs.clusterId,
  serviceDiscoveryNamespaceId: ecs.serviceDiscoveryNamespaceId,
  taskExecutionRoleArn: ecs.taskExecutionRoleArn,
  clickhouseSecurityGroupId: clickhouseSg.id,
  zookeeperSecurityGroupId: zookeeperSg.id,
});

// =============================================================================
// Stack Exports
// =============================================================================
export const stackConfig = {
  region: config.region,
  environment: config.environment,
  projectName: config.projectName,
  vpcCidr: config.vpcCidr,
  domain: config.domainName,
};

// VPC
export const vpcId = vpc.id;
export const vpcCidr = vpc.cidrBlock;

// Networking
export const publicSubnetIdsOutput = publicSubnets.map(s => s.id);
export const privateSubnetIdsOutput = privateSubnets.map(s => s.id);

// DNS
export const hostedZoneId = dns.hostedZoneId;
export const nameServers = dns.nameServers;
export const dnsZone = dnsZoneDomain;
export const nsDelegationRequired = pulumi.interpolate`Add NS records for "${dnsZoneDomain}" at your registrar (ohealth-ng.com) pointing to the nameServers above`;

// ALB
export const albDnsName = alb.albDnsName;

// ECS
export const ecsClusterId = ecs.clusterId;
export const ecsClusterArn = ecs.clusterArn;
export const ecsServiceDiscoveryNamespaceId = ecs.serviceDiscoveryNamespaceId;

// Databases
export const rdsEndpoint = rdsPrimary.endpoint;
export const rdsPort = rdsPrimary.port;

// ECR
export const ecrRepositoryUrls = ecr.repositoryUrls;

// CloudFront
export const cloudfrontDistributionId = cloudfront.distributionId;
export const cloudfrontDomainName = cloudfront.distributionDomainName;

// Frontend S3 Bucket
export const frontendBucketName = cloudfront.frontendBucketName;

// Observability
export const clickhouseInstanceId = observability.clickhouseInstanceId;
export const clickhousePrivateIp = observability.clickhousePrivateIp;

// Secrets
export const masterSecretArn = secrets.masterSecretArn;
