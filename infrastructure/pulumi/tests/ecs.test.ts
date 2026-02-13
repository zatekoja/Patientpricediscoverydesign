/**
 * ECS Services Tests
 * 
 * Tests ECS Fargate implementation as defined in V1_DEPLOYMENT_ARCHITECTURE.md
 * 
 * Services to deploy:
 * - API (Go) - Port 8080
 * - GraphQL (Go) - Port 8081
 * - SSE (Go) - Port 8082
 * - Provider API (Node.js) - Port 3000
 * - Reindexer (Go) - Background job
 * - Blnk API (Go) - Port 5001
 * - Blnk Worker (Go) - Background worker
 * 
 * Observability Stack:
 * - ClickHouse - Port 9000
 * - OTEL Collector - Ports 4317/4318
 * - SigNoz Query Service - Port 8080
 * - SigNoz Frontend - Port 3301
 */

import * as pulumi from '@pulumi/pulumi';

// Set up Pulumi mocks
pulumi.runtime.setMocks({
  newResource: (args: pulumi.runtime.MockResourceArgs): { id: string; state: any } => {
    return {
      id: `${args.name}_id`,
      state: args.inputs,
    };
  },
  call: (args: pulumi.runtime.MockCallArgs) => {
    return args.inputs;
  },
});

describe('ECS Services', () => {
  describe('ECS Cluster', () => {
    it('should create ECS cluster', () => {
      const { createEcsCluster } = require('../src/compute/ecs');
      
      const cluster = createEcsCluster('prod');
      expect(cluster).toBeDefined();
    });

    it('should have correct cluster name', () => {
      const { getEcsClusterName } = require('../src/compute/ecs');
      
      expect(getEcsClusterName('prod')).toBe('ohi-prod-cluster');
    });

    it('should enable container insights', () => {
      const { ECS_CLUSTER_CONFIG } = require('../src/compute/ecs');
      
      expect(ECS_CLUSTER_CONFIG.containerInsights).toBe(true);
    });
  });

  describe('Task Execution Role', () => {
    it('should create task execution role', () => {
      const { createTaskExecutionRole } = require('../src/compute/ecs');
      
      const role = createTaskExecutionRole('prod');
      expect(role).toBeDefined();
    });

    it('should have ECR pull permissions', () => {
      const { TASK_EXECUTION_POLICIES } = require('../src/compute/ecs');
      
      expect(TASK_EXECUTION_POLICIES).toContain('arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy');
    });

    it('should have Secrets Manager permissions', () => {
      const { TASK_EXECUTION_POLICIES } = require('../src/compute/ecs');
      
      expect(TASK_EXECUTION_POLICIES).toContain('arn:aws:iam::aws:policy/SecretsManagerReadWrite');
    });
  });

  describe('Task Definitions', () => {
    it('should create API task definition', () => {
      const { createApiTaskDefinition } = require('../src/compute/ecs');
      
      const taskDef = createApiTaskDefinition('prod', 'exec-role-arn');
      expect(taskDef).toBeDefined();
    });

    it('should configure correct CPU and memory for API', () => {
      const { getTaskResources } = require('../src/compute/ecs');
      
      const resources = getTaskResources('prod', 'api');
      expect(resources.cpu).toBe('512'); // 0.5 vCPU
      expect(resources.memory).toBe('1024'); // 1 GB
    });

    it('should create GraphQL task definition', () => {
      const { createGraphqlTaskDefinition } = require('../src/compute/ecs');
      
      const taskDef = createGraphqlTaskDefinition('prod', 'exec-role-arn');
      expect(taskDef).toBeDefined();
    });

    it('should create SSE task definition', () => {
      const { createSseTaskDefinition } = require('../src/compute/ecs');
      
      const taskDef = createSseTaskDefinition('prod', 'exec-role-arn');
      expect(taskDef).toBeDefined();
    });

    it('should create Provider API task definition', () => {
      const { createProviderApiTaskDefinition } = require('../src/compute/ecs');
      
      const taskDef = createProviderApiTaskDefinition('prod', 'exec-role-arn');
      expect(taskDef).toBeDefined();
    });

    it('should use Fargate launch type', () => {
      const { TASK_DEFINITION_CONFIG } = require('../src/compute/ecs');
      
      expect(TASK_DEFINITION_CONFIG.requiresCompatibilities).toContain('FARGATE');
      expect(TASK_DEFINITION_CONFIG.networkMode).toBe('awsvpc');
    });
  });

  describe('ECS Services', () => {
    it('should create API service', () => {
      const { createApiService } = require('../src/compute/ecs');
      
      const service = createApiService('prod', 'cluster-id', 'task-def-arn', ['subnet-1'], 'sg-id', 'tg-arn');
      expect(service).toBeDefined();
    });

    it('should have correct desired count per environment', () => {
      const { getDesiredCount } = require('../src/compute/ecs');
      
      expect(getDesiredCount('dev', 'api')).toBe(1);
      expect(getDesiredCount('staging', 'api')).toBe(2);
      expect(getDesiredCount('prod', 'api')).toBe(3);
    });

    it('should enable Fargate Spot for dev', () => {
      const { shouldUseFargateSpot } = require('../src/compute/ecs');
      
      expect(shouldUseFargateSpot('dev')).toBe(true);
      expect(shouldUseFargateSpot('prod')).toBe(false);
    });

    it('should create GraphQL service', () => {
      const { createGraphqlService } = require('../src/compute/ecs');
      
      const service = createGraphqlService('prod', 'cluster-id', 'task-def-arn', ['subnet-1'], 'sg-id', 'tg-arn');
      expect(service).toBeDefined();
    });

    it('should create SSE service', () => {
      const { createSseService } = require('../src/compute/ecs');
      
      const service = createSseService('prod', 'cluster-id', 'task-def-arn', ['subnet-1'], 'sg-id', 'tg-arn');
      expect(service).toBeDefined();
    });

    it('should configure health check grace period', () => {
      const { ECS_SERVICE_CONFIG } = require('../src/compute/ecs');
      
      expect(ECS_SERVICE_CONFIG.healthCheckGracePeriodSeconds).toBe(60);
    });
  });

  describe('Background Jobs', () => {
    it('should create Reindexer task definition', () => {
      const { createReindexerTaskDefinition } = require('../src/compute/ecs');
      
      const taskDef = createReindexerTaskDefinition('prod', 'exec-role-arn');
      expect(taskDef).toBeDefined();
    });

    it('should create Blnk Worker task definition', () => {
      const { createBlnkWorkerTaskDefinition } = require('../src/compute/ecs');
      
      const taskDef = createBlnkWorkerTaskDefinition('prod', 'exec-role-arn');
      expect(taskDef).toBeDefined();
    });

    it('should create Reindexer service without load balancer', () => {
      const { createReindexerService } = require('../src/compute/ecs');
      
      const service = createReindexerService('prod', 'cluster-id', 'task-def-arn', ['subnet-1'], 'sg-id');
      expect(service).toBeDefined();
    });

    it('should have desired count of 1 for background jobs', () => {
      const { getDesiredCount } = require('../src/compute/ecs');
      
      expect(getDesiredCount('prod', 'reindexer')).toBe(1);
      expect(getDesiredCount('prod', 'blnk-worker')).toBe(1);
    });
  });

  describe('Auto Scaling', () => {
    it('should create auto scaling target', () => {
      const { createAutoScalingTarget } = require('../src/compute/ecs');
      
      const target = createAutoScalingTarget('prod', 'cluster-name', 'service-name', 1, 10);
      expect(target).toBeDefined();
    });

    it('should configure min and max capacity per environment', () => {
      const { getAutoScalingConfig } = require('../src/compute/ecs');
      
      const devConfig = getAutoScalingConfig('dev', 'api');
      expect(devConfig.minCapacity).toBe(1);
      expect(devConfig.maxCapacity).toBe(2);
      
      const prodConfig = getAutoScalingConfig('prod', 'api');
      expect(prodConfig.minCapacity).toBe(3);
      expect(prodConfig.maxCapacity).toBe(10);
    });

    it('should create CPU-based scaling policy', () => {
      const { createCpuScalingPolicy } = require('../src/compute/ecs');
      
      const policy = createCpuScalingPolicy('prod', 'target-id');
      expect(policy).toBeDefined();
    });

    it('should configure CPU target at 70%', () => {
      const { AUTO_SCALING_CONFIG } = require('../src/compute/ecs');
      
      expect(AUTO_SCALING_CONFIG.cpuTargetValue).toBe(70);
    });

    it('should create memory-based scaling policy', () => {
      const { createMemoryScalingPolicy } = require('../src/compute/ecs');
      
      const policy = createMemoryScalingPolicy('prod', 'target-id');
      expect(policy).toBeDefined();
    });

    it('should configure memory target at 80%', () => {
      const { AUTO_SCALING_CONFIG } = require('../src/compute/ecs');
      
      expect(AUTO_SCALING_CONFIG.memoryTargetValue).toBe(80);
    });
  });

  describe('CloudWatch Log Groups', () => {
    it('should create log group for each service', () => {
      const { createLogGroup } = require('../src/compute/ecs');
      
      const logGroup = createLogGroup('prod', 'api');
      expect(logGroup).toBeDefined();
    });

    it('should have correct log group name format', () => {
      const { getLogGroupName } = require('../src/compute/ecs');
      
      expect(getLogGroupName('prod', 'api')).toBe('/aws/ecs/ohi-prod-api');
    });

    it('should configure log retention', () => {
      const { LOG_CONFIG } = require('../src/compute/ecs');
      
      expect(LOG_CONFIG.retentionInDays).toBe(30);
    });
  });

  describe('Service Discovery', () => {
    it('should create service discovery namespace', () => {
      const { createServiceDiscoveryNamespace } = require('../src/compute/ecs');
      
      const namespace = createServiceDiscoveryNamespace('prod', 'vpc-id');
      expect(namespace).toBeDefined();
    });

    it('should have correct namespace name', () => {
      const { getServiceDiscoveryNamespace } = require('../src/compute/ecs');
      
      expect(getServiceDiscoveryNamespace('prod')).toBe('ohi-prod.local');
    });

    it('should create service discovery service', () => {
      const { createServiceDiscoveryService } = require('../src/compute/ecs');
      
      const service = createServiceDiscoveryService('prod', 'namespace-id', 'api', 8080);
      expect(service).toBeDefined();
    });
  });

  describe('Environment Variables', () => {
    it('should configure environment variables for API', () => {
      const { getApiEnvironmentVariables } = require('../src/compute/ecs');
      
      const envVars = getApiEnvironmentVariables('prod');
      expect(envVars).toContainEqual(
        expect.objectContaining({ name: 'ENVIRONMENT' })
      );
    });

    it('should configure secrets for database credentials', () => {
      const { getApiSecrets } = require('../src/compute/ecs');
      
      const secrets = getApiSecrets('prod', 'db-secret-arn', 'redis-secret-arn');
      expect(secrets.length).toBeGreaterThan(0);
    });
  });

  describe('Observability Stack', () => {
    it('should create ClickHouse task definition', () => {
      const { createClickHouseTaskDefinition } = require('../src/compute/ecs');
      
      const taskDef = createClickHouseTaskDefinition('prod', 'exec-role-arn');
      expect(taskDef).toBeDefined();
    });

    it('should create OTEL Collector task definition', () => {
      const { createOtelCollectorTaskDefinition } = require('../src/compute/ecs');
      
      const taskDef = createOtelCollectorTaskDefinition('prod', 'exec-role-arn');
      expect(taskDef).toBeDefined();
    });

    it('should create SigNoz Query task definition', () => {
      const { createSigNozQueryTaskDefinition } = require('../src/compute/ecs');
      
      const taskDef = createSigNozQueryTaskDefinition('prod', 'exec-role-arn');
      expect(taskDef).toBeDefined();
    });

    it('should create SigNoz Frontend task definition', () => {
      const { createSigNozFrontendTaskDefinition } = require('../src/compute/ecs');
      
      const taskDef = createSigNozFrontendTaskDefinition('prod', 'exec-role-arn');
      expect(taskDef).toBeDefined();
    });
  });

  describe('Container Configuration', () => {
    it('should configure correct image URIs', () => {
      const { getImageUri } = require('../src/compute/ecs');
      
      // ECR format: <account-id>.dkr.ecr.<region>.amazonaws.com/<repo>:<tag>
      const imageUri = getImageUri('prod', 'api');
      expect(imageUri).toContain('.dkr.ecr.');
      expect(imageUri).toContain('/ohi-api:');
    });

    it('should configure essential containers', () => {
      const { CONTAINER_CONFIG } = require('../src/compute/ecs');
      
      expect(CONTAINER_CONFIG.essential).toBe(true);
    });

    it('should configure port mappings', () => {
      const { getPortMappings } = require('../src/compute/ecs');
      
      const apiPorts = getPortMappings('api');
      expect(apiPorts).toContainEqual(
        expect.objectContaining({ containerPort: 8080 })
      );
    });
  });

  describe('Resource Allocation', () => {
    it('should allocate more resources for prod', () => {
      const { getTaskResources } = require('../src/compute/ecs');
      
      const devResources = getTaskResources('dev', 'api');
      const prodResources = getTaskResources('prod', 'api');
      
      expect(parseInt(prodResources.cpu)).toBeGreaterThanOrEqual(parseInt(devResources.cpu));
    });

    it('should configure different resources per service', () => {
      const { getTaskResources } = require('../src/compute/ecs');
      
      const apiResources = getTaskResources('prod', 'api');
      const clickhouseResources = getTaskResources('prod', 'clickhouse');
      
      // ClickHouse needs more resources than API
      expect(parseInt(clickhouseResources.memory)).toBeGreaterThan(parseInt(apiResources.memory));
    });
  });
});
