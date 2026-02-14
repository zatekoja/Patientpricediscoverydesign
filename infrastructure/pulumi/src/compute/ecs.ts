import * as pulumi from '@pulumi/pulumi';
import * as aws from '@pulumi/aws';
import { getResourceTags } from '../tagging';

export interface EcsConfig {
  environment: string;
  vpcId: pulumi.Output<string>;
  privateSubnetIds: pulumi.Output<string>[];
  securityGroupIds: {
    api: pulumi.Output<string>;
    graphql: pulumi.Output<string>;
    sse: pulumi.Output<string>;
    providerApi: pulumi.Output<string>;
    reindexer: pulumi.Output<string>;
    blnkApi: pulumi.Output<string>;
    blnkWorker: pulumi.Output<string>;
    clickhouse: pulumi.Output<string>;
    otel: pulumi.Output<string>;
    signoz: pulumi.Output<string>;
  };
  albTargetGroupArns?: {
    api: pulumi.Output<string>;
    graphql: pulumi.Output<string>;
    sse: pulumi.Output<string>;
    providerApi: pulumi.Output<string>;
  };
  databaseEndpoint: pulumi.Output<string>;
  databasePasswordSecretArn: pulumi.Output<string>;
  redisEndpoint: pulumi.Output<string>;
  redisAuthTokenSecretArn: pulumi.Output<string>;
  blnkRedisEndpoint: pulumi.Output<string>;
  blnkRedisAuthTokenSecretArn: pulumi.Output<string>;
}

interface ServiceResources {
  cpu: number;
  memory: number;
}

interface AutoScalingConfig {
  min: number;
  max: number;
  cpuTarget: number;
  memoryTarget: number;
}

// Resource allocation per environment and service
const SERVICE_RESOURCES: Record<string, Record<string, ServiceResources>> = {
  prod: {
    api: { cpu: 1024, memory: 2048 },
    graphql: { cpu: 1024, memory: 2048 },
    sse: { cpu: 512, memory: 1024 },
    'provider-api': { cpu: 1024, memory: 2048 },
    reindexer: { cpu: 512, memory: 1024 },
    'blnk-api': { cpu: 512, memory: 1024 },
    'blnk-worker': { cpu: 256, memory: 512 },
    // ClickHouse runs on EC2, not ECS Fargate (see clickhouse.ts)
    otel: { cpu: 512, memory: 1024 },
    'signoz-query': { cpu: 512, memory: 1024 },
    'signoz-frontend': { cpu: 256, memory: 512 },
  },
  staging: {
    api: { cpu: 512, memory: 1024 },
    graphql: { cpu: 512, memory: 1024 },
    sse: { cpu: 256, memory: 512 },
    'provider-api': { cpu: 512, memory: 1024 },
    reindexer: { cpu: 256, memory: 512 },
    'blnk-api': { cpu: 256, memory: 512 },
    'blnk-worker': { cpu: 256, memory: 512 },
    otel: { cpu: 256, memory: 512 },
    'signoz-query': { cpu: 512, memory: 1024 },
    'signoz-frontend': { cpu: 256, memory: 512 },
  },
  dev: {
    api: { cpu: 256, memory: 512 },
    graphql: { cpu: 256, memory: 512 },
    sse: { cpu: 256, memory: 512 },
    'provider-api': { cpu: 256, memory: 512 },
    reindexer: { cpu: 256, memory: 512 },
    'blnk-api': { cpu: 256, memory: 512 },
    'blnk-worker': { cpu: 256, memory: 512 },
    otel: { cpu: 256, memory: 512 },
    'signoz-query': { cpu: 256, memory: 512 },
    'signoz-frontend': { cpu: 256, memory: 512 },
  },
};

const DESIRED_COUNTS: Record<string, Record<string, number>> = {
  prod: {
    api: 3,
    graphql: 3,
    sse: 2,
    'provider-api': 2,
    reindexer: 1,
    'blnk-api': 2,
    'blnk-worker': 1,
    // ClickHouse runs on EC2, not ECS Fargate
    otel: 2,
    'signoz-query': 1,
    'signoz-frontend': 1,
  },
  staging: {
    api: 2,
    graphql: 2,
    sse: 1,
    'provider-api': 1,
    reindexer: 1,
    'blnk-api': 1,
    'blnk-worker': 1,
    otel: 1,
    'signoz-query': 1,
    'signoz-frontend': 1,
  },
  dev: {
    api: 1,
    graphql: 1,
    sse: 1,
    'provider-api': 1,
    reindexer: 1,
    'blnk-api': 1,
    'blnk-worker': 1,
    otel: 1,
    'signoz-query': 1,
    'signoz-frontend': 1,
  },
};

const AUTO_SCALING_CONFIGS: Record<string, AutoScalingConfig> = {
  prod: {
    min: 2,
    max: 10,
    cpuTarget: 70,
    memoryTarget: 80,
  },
  staging: {
    min: 1,
    max: 5,
    cpuTarget: 70,
    memoryTarget: 80,
  },
  dev: {
    min: 1,
    max: 3,
    cpuTarget: 70,
    memoryTarget: 80,
  },
};

export function createEcsCluster(config: EcsConfig): aws.ecs.Cluster {
  const cluster = new aws.ecs.Cluster(`ohi-${config.environment}`, {
    name: `ohi-${config.environment}`,
    settings: [
      {
        name: 'containerInsights',
        value: 'enabled',
      },
    ],
    tags: { ...getResourceTags(config.environment, 'ecs-cluster') },
  });

  return cluster;
}

export function createTaskExecutionRole(config: EcsConfig): aws.iam.Role {
  const role = new aws.iam.Role(`ohi-${config.environment}-ecs-task-execution`, {
    name: `ohi-${config.environment}-ecs-task-execution`,
    assumeRolePolicy: JSON.stringify({
      Version: '2012-10-17',
      Statement: [
        {
          Effect: 'Allow',
          Principal: {
            Service: 'ecs-tasks.amazonaws.com',
          },
          Action: 'sts:AssumeRole',
        },
      ],
    }),
    tags: { ...getResourceTags(config.environment, 'ecs-task-execution-role') },
  });

  // Attach AWS managed policy for ECS task execution
  new aws.iam.RolePolicyAttachment(`ohi-${config.environment}-ecs-task-execution-policy`, {
    role: role.name,
    policyArn: 'arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy',
  });

  // Additional policy for ECR and Secrets Manager
  const policy = new aws.iam.Policy(`ohi-${config.environment}-ecs-task-execution-additional`, {
    name: `ohi-${config.environment}-ecs-task-execution-additional`,
    policy: JSON.stringify({
      Version: '2012-10-17',
      Statement: [
        {
          Effect: 'Allow',
          Action: [
            'ecr:GetAuthorizationToken',
            'ecr:BatchCheckLayerAvailability',
            'ecr:GetDownloadUrlForLayer',
            'ecr:BatchGetImage',
          ],
          Resource: '*',
        },
        {
          Effect: 'Allow',
          Action: ['secretsmanager:GetSecretValue'],
          Resource: [
            config.databasePasswordSecretArn,
            config.redisAuthTokenSecretArn,
            config.blnkRedisAuthTokenSecretArn,
          ],
        },
        {
          Effect: 'Allow',
          Action: ['logs:CreateLogGroup', 'logs:CreateLogStream', 'logs:PutLogEvents'],
          Resource: '*',
        },
      ],
    }),
    tags: { ...getResourceTags(config.environment, 'ecs-task-execution-policy') },
  });

  new aws.iam.RolePolicyAttachment(`ohi-${config.environment}-ecs-task-execution-additional-attach`, {
    role: role.name,
    policyArn: policy.arn,
  });

  return role;
}

export function createLogGroup(config: EcsConfig, serviceName: string): aws.cloudwatch.LogGroup {
  return new aws.cloudwatch.LogGroup(`ohi-${config.environment}-${serviceName}`, {
    name: `/ecs/ohi-${config.environment}/${serviceName}`,
    retentionInDays: config.environment === 'prod' ? 30 : 7,
    tags: { ...getResourceTags(config.environment, `${serviceName}-log-group`) },
  });
}

export function createServiceDiscoveryNamespace(config: EcsConfig): aws.servicediscovery.PrivateDnsNamespace {
  return new aws.servicediscovery.PrivateDnsNamespace(`ohi-${config.environment}`, {
    name: `ohi-${config.environment}.local`,
    description: `Service discovery namespace for ohi-${config.environment}`,
    vpc: config.vpcId,
    tags: { ...getResourceTags(config.environment, 'service-discovery-namespace') },
  });
}

export function createServiceDiscoveryService(
  config: EcsConfig,
  serviceName: string,
  namespace: aws.servicediscovery.PrivateDnsNamespace
): aws.servicediscovery.Service {
  return new aws.servicediscovery.Service(`ohi-${config.environment}-${serviceName}`, {
    name: serviceName,
    dnsConfig: {
      namespaceId: namespace.id,
      dnsRecords: [
        {
          ttl: 10,
          type: 'A',
        },
      ],
      routingPolicy: 'MULTIVALUE',
    },
    healthCheckCustomConfig: {
      failureThreshold: 1,
    },
    tags: { ...getResourceTags(config.environment, `${serviceName}-discovery`) },
  });
}

function getEnvironmentVariables(
  config: EcsConfig,
  serviceName: string
): { name: string; value: string }[] {
  const commonEnvVars: { name: string; value: string }[] = [
    { name: 'ENVIRONMENT', value: config.environment },
    { name: 'LOG_LEVEL', value: config.environment === 'prod' ? 'info' : 'debug' },
    { name: 'DATABASE_HOST', value: pulumi.interpolate`${config.databaseEndpoint}` as any },
    { name: 'DATABASE_PORT', value: '5432' },
    { name: 'DATABASE_NAME', value: 'ohi' },
    { name: 'DATABASE_USER', value: 'ohi_admin' },
    { name: 'REDIS_HOST', value: pulumi.interpolate`${config.redisEndpoint}` as any },
    { name: 'REDIS_PORT', value: '6379' },
  ];

  if (serviceName === 'blnk-api' || serviceName === 'blnk-worker') {
    return [
      ...commonEnvVars,
      { name: 'BLNK_REDIS_HOST', value: pulumi.interpolate`${config.blnkRedisEndpoint}` as any },
      { name: 'BLNK_REDIS_PORT', value: '6379' },
    ];
  }

  return commonEnvVars;
}

function getSecrets(
  config: EcsConfig,
  serviceName: string
): { name: string; valueFrom: string }[] {
  const secrets: { name: string; valueFrom: string }[] = [
    {
      name: 'DATABASE_PASSWORD',
      valueFrom: pulumi.interpolate`${config.databasePasswordSecretArn}` as any,
    },
    {
      name: 'REDIS_AUTH_TOKEN',
      valueFrom: pulumi.interpolate`${config.redisAuthTokenSecretArn}` as any,
    },
  ];

  if (serviceName === 'blnk-api' || serviceName === 'blnk-worker') {
    secrets.push({
      name: 'BLNK_REDIS_AUTH_TOKEN',
      valueFrom: pulumi.interpolate`${config.blnkRedisAuthTokenSecretArn}` as any,
    });
  }

  return secrets;
}

export function createTaskDefinition(
  config: EcsConfig,
  serviceName: string,
  executionRole: aws.iam.Role,
  logGroup: aws.cloudwatch.LogGroup,
  containerPort?: number
): aws.ecs.TaskDefinition {
  const resources = SERVICE_RESOURCES[config.environment][serviceName];
  const accountId = aws.getCallerIdentityOutput().accountId;
  const imageUri = pulumi.interpolate`${accountId}.dkr.ecr.eu-west-1.amazonaws.com/ohi-${serviceName}:${config.environment}`;

  const portMappings: { containerPort: number; protocol: string }[] = containerPort
    ? [
        {
          containerPort: containerPort,
          protocol: 'tcp',
        },
      ]
    : [];

  return new aws.ecs.TaskDefinition(`ohi-${config.environment}-${serviceName}`, {
    family: `ohi-${config.environment}-${serviceName}`,
    cpu: resources.cpu.toString(),
    memory: resources.memory.toString(),
    networkMode: 'awsvpc',
    requiresCompatibilities: ['FARGATE'],
    executionRoleArn: executionRole.arn,
    taskRoleArn: executionRole.arn,
    containerDefinitions: pulumi.jsonStringify([
      {
        name: serviceName,
        image: imageUri,
        portMappings: portMappings,
        environment: getEnvironmentVariables(config, serviceName),
        secrets: getSecrets(config, serviceName),
        logConfiguration: {
          logDriver: 'awslogs',
          options: {
            'awslogs-group': logGroup.name,
            'awslogs-region': 'eu-west-1',
            'awslogs-stream-prefix': 'ecs',
          },
        },
        essential: true,
      },
    ]),
    tags: { ...getResourceTags(config.environment, `${serviceName}-task-definition`) },
  });
}

export function createEcsService(
  config: EcsConfig,
  serviceName: string,
  cluster: aws.ecs.Cluster,
  taskDefinition: aws.ecs.TaskDefinition,
  securityGroupId: pulumi.Output<string>,
  targetGroupArn?: pulumi.Output<string>,
  discoveryService?: aws.servicediscovery.Service
): aws.ecs.Service {
  const desiredCount = DESIRED_COUNTS[config.environment][serviceName];
  const loadBalancers = targetGroupArn
    ? [
        {
          targetGroupArn: targetGroupArn,
          containerName: serviceName,
          containerPort: getContainerPort(serviceName),
        },
      ]
    : [];

  // Use Fargate Spot for dev environment
  const capacityProviderStrategies =
    config.environment === 'dev'
      ? [
          {
            capacityProvider: 'FARGATE_SPOT',
            weight: 100,
            base: 0,
          },
        ]
      : [
          {
            capacityProvider: 'FARGATE',
            weight: 100,
            base: 0,
          },
        ];

  const serviceConfig: any = {
    name: `ohi-${config.environment}-${serviceName}`,
    cluster: cluster.id,
    taskDefinition: taskDefinition.arn,
    desiredCount: desiredCount,
    launchType: undefined, // Use capacity provider strategy instead
    capacityProviderStrategies: capacityProviderStrategies,
    networkConfiguration: {
      subnets: config.privateSubnetIds,
      securityGroups: [securityGroupId],
      assignPublicIp: false,
    },
    loadBalancers: loadBalancers,
    healthCheckGracePeriodSeconds: targetGroupArn ? 60 : undefined,
    deploymentConfiguration: {
      maximumPercent: 200,
      minimumHealthyPercent: 100,
    },
    tags: { ...getResourceTags(config.environment, `${serviceName}-service`) },
  };

  if (discoveryService) {
    serviceConfig.serviceRegistries = [
      {
        registryArn: discoveryService.arn,
      },
    ];
  }

  return new aws.ecs.Service(`ohi-${config.environment}-${serviceName}`, serviceConfig);
}

function getContainerPort(serviceName: string): number {
  const portMap: Record<string, number> = {
    api: 8080,
    graphql: 8080,
    sse: 8080,
    'provider-api': 8080,
    'blnk-api': 5001,
    clickhouse: 8123,
    otel: 4317,
    'signoz-query': 8080,
    'signoz-frontend': 3301,
  };
  return portMap[serviceName] || 8080;
}

export function createAutoScalingTarget(
  config: EcsConfig,
  serviceName: string,
  cluster: aws.ecs.Cluster,
  service: aws.ecs.Service
): aws.appautoscaling.Target {
  const scalingConfig = AUTO_SCALING_CONFIGS[config.environment];

  return new aws.appautoscaling.Target(`ohi-${config.environment}-${serviceName}`, {
    serviceNamespace: 'ecs',
    resourceId: pulumi.interpolate`service/${cluster.name}/${service.name}`,
    scalableDimension: 'ecs:service:DesiredCount',
    minCapacity: scalingConfig.min,
    maxCapacity: scalingConfig.max,
  });
}

export function createCpuScalingPolicy(
  config: EcsConfig,
  serviceName: string,
  target: aws.appautoscaling.Target
): aws.appautoscaling.Policy {
  const scalingConfig = AUTO_SCALING_CONFIGS[config.environment];

  return new aws.appautoscaling.Policy(`ohi-${config.environment}-${serviceName}-cpu`, {
    name: `ohi-${config.environment}-${serviceName}-cpu`,
    policyType: 'TargetTrackingScaling',
    serviceNamespace: target.serviceNamespace,
    resourceId: target.resourceId,
    scalableDimension: target.scalableDimension,
    targetTrackingScalingPolicyConfiguration: {
      targetValue: scalingConfig.cpuTarget,
      predefinedMetricSpecification: {
        predefinedMetricType: 'ECSServiceAverageCPUUtilization',
      },
      scaleInCooldown: 300,
      scaleOutCooldown: 60,
    },
  });
}

export function createMemoryScalingPolicy(
  config: EcsConfig,
  serviceName: string,
  target: aws.appautoscaling.Target
): aws.appautoscaling.Policy {
  const scalingConfig = AUTO_SCALING_CONFIGS[config.environment];

  return new aws.appautoscaling.Policy(`ohi-${config.environment}-${serviceName}-memory`, {
    name: `ohi-${config.environment}-${serviceName}-memory`,
    policyType: 'TargetTrackingScaling',
    serviceNamespace: target.serviceNamespace,
    resourceId: target.resourceId,
    scalableDimension: target.scalableDimension,
    targetTrackingScalingPolicyConfiguration: {
      targetValue: scalingConfig.memoryTarget,
      predefinedMetricSpecification: {
        predefinedMetricType: 'ECSServiceAverageMemoryUtilization',
      },
      scaleInCooldown: 300,
      scaleOutCooldown: 60,
    },
  });
}

export interface EcsOutputs {
  clusterId: pulumi.Output<string>;
  clusterArn: pulumi.Output<string>;
  serviceArns: Record<string, pulumi.Output<string>>;
  taskDefinitionArns: Record<string, pulumi.Output<string>>;
  serviceDiscoveryNamespaceId: pulumi.Output<string>;
  taskExecutionRoleArn: pulumi.Output<string>;
}

export function createEcsInfrastructure(config: EcsConfig): EcsOutputs {
  // Create ECS cluster
  const cluster = createEcsCluster(config);

  // Create task execution role
  const executionRole = createTaskExecutionRole(config);

  // Create service discovery namespace
  const namespace = createServiceDiscoveryNamespace(config);

  // Create log groups
  const logGroups: Record<string, aws.cloudwatch.LogGroup> = {};
  const services = ['api', 'graphql', 'sse', 'provider-api', 'reindexer', 'blnk-api', 'blnk-worker'];
  services.forEach((service) => {
    logGroups[service] = createLogGroup(config, service);
  });

  // Create task definitions
  const taskDefinitions: Record<string, aws.ecs.TaskDefinition> = {};
  taskDefinitions['api'] = createTaskDefinition(config, 'api', executionRole, logGroups['api'], 8080);
  taskDefinitions['graphql'] = createTaskDefinition(config, 'graphql', executionRole, logGroups['graphql'], 8080);
  taskDefinitions['sse'] = createTaskDefinition(config, 'sse', executionRole, logGroups['sse'], 8080);
  taskDefinitions['provider-api'] = createTaskDefinition(
    config,
    'provider-api',
    executionRole,
    logGroups['provider-api'],
    8080
  );
  taskDefinitions['reindexer'] = createTaskDefinition(config, 'reindexer', executionRole, logGroups['reindexer']);
  taskDefinitions['blnk-api'] = createTaskDefinition(config, 'blnk-api', executionRole, logGroups['blnk-api'], 5001);
  taskDefinitions['blnk-worker'] = createTaskDefinition(config, 'blnk-worker', executionRole, logGroups['blnk-worker']);

  // Create service discovery services
  const discoveryServices: Record<string, aws.servicediscovery.Service> = {};
  services.forEach((service) => {
    discoveryServices[service] = createServiceDiscoveryService(config, service, namespace);
  });

  // Create ECS services
  const ecsServices: Record<string, aws.ecs.Service> = {};
  ecsServices['api'] = createEcsService(
    config,
    'api',
    cluster,
    taskDefinitions['api'],
    config.securityGroupIds.api,
    config.albTargetGroupArns?.api,
    discoveryServices['api']
  );
  ecsServices['graphql'] = createEcsService(
    config,
    'graphql',
    cluster,
    taskDefinitions['graphql'],
    config.securityGroupIds.graphql,
    config.albTargetGroupArns?.graphql,
    discoveryServices['graphql']
  );
  ecsServices['sse'] = createEcsService(
    config,
    'sse',
    cluster,
    taskDefinitions['sse'],
    config.securityGroupIds.sse,
    config.albTargetGroupArns?.sse,
    discoveryServices['sse']
  );
  ecsServices['provider-api'] = createEcsService(
    config,
    'provider-api',
    cluster,
    taskDefinitions['provider-api'],
    config.securityGroupIds.providerApi,
    config.albTargetGroupArns?.providerApi,
    discoveryServices['provider-api']
  );
  ecsServices['reindexer'] = createEcsService(
    config,
    'reindexer',
    cluster,
    taskDefinitions['reindexer'],
    config.securityGroupIds.reindexer,
    undefined,
    discoveryServices['reindexer']
  );
  ecsServices['blnk-api'] = createEcsService(
    config,
    'blnk-api',
    cluster,
    taskDefinitions['blnk-api'],
    config.securityGroupIds.blnkApi,
    undefined,
    discoveryServices['blnk-api']
  );
  ecsServices['blnk-worker'] = createEcsService(
    config,
    'blnk-worker',
    cluster,
    taskDefinitions['blnk-worker'],
    config.securityGroupIds.blnkWorker,
    undefined,
    discoveryServices['blnk-worker']
  );

  // Create auto-scaling for services that need it (not background jobs)
  const scalableServices = ['api', 'graphql', 'sse', 'provider-api'];
  scalableServices.forEach((service) => {
    const target = createAutoScalingTarget(config, service, cluster, ecsServices[service]);
    createCpuScalingPolicy(config, service, target);
    createMemoryScalingPolicy(config, service, target);
  });

  return {
    clusterId: cluster.id,
    clusterArn: cluster.arn,
    serviceArns: {
      api: ecsServices['api'].id,
      graphql: ecsServices['graphql'].id,
      sse: ecsServices['sse'].id,
      providerApi: ecsServices['provider-api'].id,
      reindexer: ecsServices['reindexer'].id,
      blnkApi: ecsServices['blnk-api'].id,
      blnkWorker: ecsServices['blnk-worker'].id,
    },
    taskDefinitionArns: {
      api: taskDefinitions['api'].arn,
      graphql: taskDefinitions['graphql'].arn,
      sse: taskDefinitions['sse'].arn,
      providerApi: taskDefinitions['provider-api'].arn,
      reindexer: taskDefinitions['reindexer'].arn,
      blnkApi: taskDefinitions['blnk-api'].arn,
      blnkWorker: taskDefinitions['blnk-worker'].arn,
    },
    serviceDiscoveryNamespaceId: namespace.id,
    taskExecutionRoleArn: executionRole.arn,
  };
}

/**
 * Create Zookeeper task definition for ClickHouse coordination
 */
export function createZookeeperTaskDefinition(
  environment: string,
  taskExecutionRoleArn: pulumi.Input<string>,
  zookeeperIndex: number
): aws.ecs.TaskDefinition {
  const taskDef = new aws.ecs.TaskDefinition(
    `ohi-${environment}-zookeeper-${zookeeperIndex}`,
    {
      family: `ohi-${environment}-zookeeper-${zookeeperIndex}`,
      networkMode: 'awsvpc',
      requiresCompatibilities: ['FARGATE'],
      cpu: '256',
      memory: '512',
      executionRoleArn: taskExecutionRoleArn,
      containerDefinitions: JSON.stringify([
        {
          name: 'zookeeper',
          image: 'zookeeper:3.8.4',
          essential: true,
          environment: [
            { name: 'ZOO_MY_ID', value: String(zookeeperIndex + 1) },
            { name: 'ZOO_SERVERS', value: `server.1=zookeeper-1.ohi-${environment}.local:2888:3888 server.2=zookeeper-2.ohi-${environment}.local:2888:3888 server.3=zookeeper-3.ohi-${environment}.local:2888:3888` },
            { name: 'ZOO_4LW_COMMANDS_WHITELIST', value: 'srvr,mntr,ruok' },
            { name: 'ZOO_CFG_EXTRA', value: 'metricsProvider.className=org.apache.zookeeper.metrics.prometheus.PrometheusMetricsProvider metricsProvider.httpPort=7000' },
          ],
          portMappings: [
            { containerPort: 2181, protocol: 'tcp' }, // Client port
            { containerPort: 2888, protocol: 'tcp' }, // Follower port
            { containerPort: 3888, protocol: 'tcp' }, // Election port
            { containerPort: 7000, protocol: 'tcp' }, // Metrics port
          ],
          logConfiguration: {
            logDriver: 'awslogs',
            options: {
              'awslogs-group': `/ecs/ohi-${environment}-zookeeper`,
              'awslogs-region': 'eu-west-1',
              'awslogs-stream-prefix': 'zookeeper',
            },
          },
          healthCheck: {
            command: ['CMD-SHELL', 'echo ruok | nc localhost 2181 | grep imok || exit 1'],
            interval: 30,
            timeout: 5,
            retries: 3,
            startPeriod: 60,
          },
        },
      ]),
      tags: {
        ...getResourceTags(environment as any, 'zookeeper'),
        Name: `ohi-${environment}-zookeeper-${zookeeperIndex}`,
        Service: 'zookeeper',
        Component: 'observability',
      },
    }
  );

  return taskDef;
}

/**
 * Create Zookeeper ECS service for ClickHouse coordination
 */
export function createZookeeperService(
  environment: string,
  clusterId: pulumi.Input<string>,
  taskDefinitionArn: pulumi.Input<string>,
  privateSubnetIds: pulumi.Input<string>[],
  securityGroupId: pulumi.Input<string>,
  namespaceId: pulumi.Input<string>,
  zookeeperIndex: number
): aws.ecs.Service {
  // Create CloudWatch log group
  const logGroup = new aws.cloudwatch.LogGroup(
    `ohi-${environment}-zookeeper-logs`,
    {
      name: `/ecs/ohi-${environment}-zookeeper`,
      retentionInDays: environment === 'prod' ? 30 : 7,
      tags: {
        ...getResourceTags(environment as any, 'zookeeper'),
        Component: 'observability',
      },
    }
  );

  // Create service discovery
  const serviceDiscovery = new aws.servicediscovery.Service(
    `ohi-${environment}-zookeeper-${zookeeperIndex}-discovery`,
    {
      name: `zookeeper-${zookeeperIndex}`,
      dnsConfig: {
        namespaceId,
        dnsRecords: [{ ttl: 10, type: 'A' }],
        routingPolicy: 'MULTIVALUE',
      },
      healthCheckCustomConfig: {
        failureThreshold: 1,
      },
      tags: {
        ...getResourceTags(environment as any, 'zookeeper'),
        Service: 'zookeeper',
        Component: 'observability',
      },
    }
  );

  const serviceArgs: any = {
    name: `ohi-${environment}-zookeeper-${zookeeperIndex}`,
    cluster: clusterId,
    taskDefinition: taskDefinitionArn,
    desiredCount: 1, // Each service runs 1 instance
    launchType: 'FARGATE',
    networkConfiguration: {
      subnets: privateSubnetIds,
      securityGroups: [securityGroupId],
      assignPublicIp: false,
    },
    serviceRegistries: [{ registryArn: serviceDiscovery.arn }],
    enableExecuteCommand: environment !== 'prod',
    tags: {
      ...getResourceTags(environment as any, 'zookeeper'),
      Name: `ohi-${environment}-zookeeper-${zookeeperIndex}`,
      Service: 'zookeeper',
      Component: 'observability',
    },
  };

  const service = new aws.ecs.Service(
    `ohi-${environment}-zookeeper-${zookeeperIndex}`,
    serviceArgs,
    { dependsOn: [logGroup] }
  );

  return service;
}
