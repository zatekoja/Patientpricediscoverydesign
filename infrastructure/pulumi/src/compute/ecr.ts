/**
 * ECR (Elastic Container Registry) Module
 * 
 * Creates and manages Docker image repositories for all containerized services.
 * Includes lifecycle policies, image scanning, and tag mutability controls.
 */

import * as aws from '@pulumi/aws';
import * as pulumi from '@pulumi/pulumi';
import { getResourceTags } from '../tagging';

export type Environment = 'dev' | 'staging' | 'prod';

/**
 * Configuration for ECR repository creation
 */
export interface EcrConfig {
  environment: Environment;
  enableImageScanning?: boolean;     // Default: true for prod, false otherwise
  enableTagMutability?: boolean;     // Default: false (immutable tags)
  lifecycleMaxImages?: number;       // Default: 10
  enableReplication?: boolean;       // Default: false
}

/**
 * Service names for ECR repositories
 */
export const ECR_SERVICES = [
  'api',              // Main API service
  'graphql',          // GraphQL server
  'sse',              // Server-Sent Events server
  'provider-api',     // Provider API service
  'reindexer',        // Data reindexer service
  'blnk-api',         // Blnk API
  'blnk-worker',      // Blnk worker
  'clickhouse',       // ClickHouse (if custom image)
  'otel',             // OpenTelemetry collector
  'signoz-query',     // SigNoz query service
  'signoz-frontend',  // SigNoz frontend
] as const;

export type EcrService = typeof ECR_SERVICES[number];

/**
 * ECR infrastructure outputs
 */
export interface EcrOutputs {
  repositoryArns: Record<EcrService, pulumi.Output<string>>;
  repositoryUrls: Record<EcrService, pulumi.Output<string>>;
  repositoryNames: Record<EcrService, pulumi.Output<string>>;
  registryId: pulumi.Output<string>;
}

/**
 * Creates an ECR repository for a service
 */
export function createEcrRepository(
  config: EcrConfig,
  service: EcrService
): aws.ecr.Repository {
  const { environment } = config;
  const repoName = `ohi-${environment}-${service}`;

  // Determine scanning and mutability based on environment
  const enableScanning = config.enableImageScanning ?? (environment === 'prod');
  const tagMutability = config.enableTagMutability ? 'MUTABLE' : 'IMMUTABLE';

  const repository = new aws.ecr.Repository(
    `${repoName}-repo`,
    {
      name: repoName,
      imageScanningConfiguration: {
        scanOnPush: enableScanning,
      },
      imageTagMutability: tagMutability,
      encryptionConfigurations: [{
        encryptionType: 'AES256', // Use AWS managed key for encryption
      }],
      tags: {
        ...getResourceTags(environment, service),
        Repository: repoName,
      },
    }
  );

  return repository;
}

/**
 * Creates a lifecycle policy for ECR repository
 * Keeps only the most recent N images, removes older ones
 */
export function createLifecyclePolicy(
  config: EcrConfig,
  repository: aws.ecr.Repository,
  service: EcrService
): aws.ecr.LifecyclePolicy {
  const { environment } = config;
  const maxImages = config.lifecycleMaxImages ?? 10;

  const policyDocument = {
    rules: [
      {
        rulePriority: 1,
        description: `Keep last ${maxImages} images`,
        selection: {
          tagStatus: 'any',
          countType: 'imageCountMoreThan',
          countNumber: maxImages,
        },
        action: {
          type: 'expire',
        },
      },
      {
        rulePriority: 2,
        description: 'Remove untagged images after 1 day',
        selection: {
          tagStatus: 'untagged',
          countType: 'sinceImagePushed',
          countUnit: 'days',
          countNumber: 1,
        },
        action: {
          type: 'expire',
        },
      },
    ],
  };

  const policy = new aws.ecr.LifecyclePolicy(
    `ohi-${environment}-${service}-lifecycle`,
    {
      repository: repository.name,
      policy: JSON.stringify(policyDocument),
    }
  );

  return policy;
}

/**
 * Creates an IAM policy for ECR access
 * Allows CI/CD to push images and ECS to pull images
 */
export function createEcrAccessPolicy(
  config: EcrConfig,
  repositories: Record<EcrService, aws.ecr.Repository>
): aws.iam.Policy {
  const { environment } = config;

  const repositoryArns = Object.values(repositories).map(repo => repo.arn);

  const policy = new aws.iam.Policy(
    `ohi-${environment}-ecr-access-policy`,
    {
      name: `ohi-${environment}-ecr-access`,
      description: `ECR access policy for ${environment} environment`,
      policy: pulumi.all(repositoryArns).apply(arns =>
        JSON.stringify({
          Version: '2012-10-17',
          Statement: [
            {
              Sid: 'AllowPushPull',
              Effect: 'Allow',
              Action: [
                'ecr:GetDownloadUrlForLayer',
                'ecr:BatchGetImage',
                'ecr:BatchCheckLayerAvailability',
                'ecr:PutImage',
                'ecr:InitiateLayerUpload',
                'ecr:UploadLayerPart',
                'ecr:CompleteLayerUpload',
              ],
              Resource: arns,
            },
            {
              Sid: 'AllowGetAuthToken',
              Effect: 'Allow',
              Action: [
                'ecr:GetAuthorizationToken',
              ],
              Resource: '*',
            },
          ],
        })
      ),
      tags: getResourceTags(environment, 'ecr'),
    }
  );

  return policy;
}

/**
 * Creates repository policies for cross-account access (if needed)
 */
export function createRepositoryPolicy(
  config: EcrConfig,
  repository: aws.ecr.Repository,
  service: EcrService,
  allowedAccountIds: string[]
): aws.ecr.RepositoryPolicy {
  const { environment } = config;

  const policyDocument = {
    Version: '2012-10-17',
    Statement: [
      {
        Sid: 'AllowCrossAccountPull',
        Effect: 'Allow',
        Principal: {
          AWS: allowedAccountIds.map(id => `arn:aws:iam::${id}:root`),
        },
        Action: [
          'ecr:GetDownloadUrlForLayer',
          'ecr:BatchGetImage',
          'ecr:BatchCheckLayerAvailability',
        ],
      },
    ],
  };

  const policy = new aws.ecr.RepositoryPolicy(
    `ohi-${environment}-${service}-repo-policy`,
    {
      repository: repository.name,
      policy: JSON.stringify(policyDocument),
    }
  );

  return policy;
}

/**
 * Creates replication configuration for ECR
 * Useful for disaster recovery or multi-region deployments
 */
export function createReplicationConfiguration(
  config: EcrConfig,
  targetRegions: string[]
): aws.ecr.ReplicationConfiguration {
  if (targetRegions.length === 0) {
    throw new Error('At least one target region must be specified for replication');
  }

  const replicationConfig = new aws.ecr.ReplicationConfiguration(
    `ohi-${config.environment}-replication`,
    {
      replicationConfiguration: {
        rules: targetRegions.map((region, index) => ({
          destinations: [
            {
              region: region,
              registryId: pulumi.output(
                aws.getCallerIdentity().then(identity => identity.accountId)
              ),
            },
          ],
        })),
      },
    }
  );

  return replicationConfig;
}

/**
 * Gets the ECR registry URL for the current AWS account
 */
export function getRegistryUrl(region?: string): pulumi.Output<string> {
  const actualRegion = region || aws.config.requireRegion();
  
  return pulumi.output(
    aws.getCallerIdentity().then(identity => 
      `${identity.accountId}.dkr.ecr.${actualRegion}.amazonaws.com`
    )
  );
}

/**
 * Gets the full image URI for a service
 */
export function getImageUri(
  config: EcrConfig,
  service: EcrService,
  imageTag: string = 'latest'
): pulumi.Output<string> {
  const repoName = `ohi-${config.environment}-${service}`;
  return getRegistryUrl().apply(registryUrl => 
    `${registryUrl}/${repoName}:${imageTag}`
  );
}

/**
 * Creates all ECR repositories and related resources
 */
export function createEcrInfrastructure(config: EcrConfig): EcrOutputs {
  const repositories: Partial<Record<EcrService, aws.ecr.Repository>> = {};
  const lifecyclePolicies: Partial<Record<EcrService, aws.ecr.LifecyclePolicy>> = {};

  // Create repositories for all services
  for (const service of ECR_SERVICES) {
    const repository = createEcrRepository(config, service);
    repositories[service] = repository;

    // Create lifecycle policy
    const lifecyclePolicy = createLifecyclePolicy(config, repository, service);
    lifecyclePolicies[service] = lifecyclePolicy;
  }

  // Create ECR access policy
  createEcrAccessPolicy(config, repositories as Record<EcrService, aws.ecr.Repository>);

  // Get registry ID
  const registryId = pulumi.output(
    aws.getCallerIdentity().then(identity => identity.accountId)
  );

  // Prepare outputs
  const repositoryArns: Partial<Record<EcrService, pulumi.Output<string>>> = {};
  const repositoryUrls: Partial<Record<EcrService, pulumi.Output<string>>> = {};
  const repositoryNames: Partial<Record<EcrService, pulumi.Output<string>>> = {};

  for (const service of ECR_SERVICES) {
    const repo = repositories[service]!;
    repositoryArns[service] = repo.arn;
    repositoryUrls[service] = repo.repositoryUrl;
    repositoryNames[service] = repo.name;
  }

  return {
    repositoryArns: repositoryArns as Record<EcrService, pulumi.Output<string>>,
    repositoryUrls: repositoryUrls as Record<EcrService, pulumi.Output<string>>,
    repositoryNames: repositoryNames as Record<EcrService, pulumi.Output<string>>,
    registryId,
  };
}

/**
 * Export helpers for testing and external usage
 */
export const helpers = {
  getRepositoryName: (environment: Environment, service: EcrService) => 
    `ohi-${environment}-${service}`,
  getDefaultImageTag: () => 'latest',
  getImageTagFromCommit: (commitSha: string) => commitSha.substring(0, 7),
};
