import * as pulumi from '@pulumi/pulumi';
import * as aws from '@pulumi/aws';
import { getResourceTags } from '../tagging';

export interface AlbConfig {
  environment: string;
  vpcId: pulumi.Output<string>;
  publicSubnetIds: pulumi.Output<string>[];
  albSecurityGroupId: pulumi.Output<string>;
  certificateArn?: pulumi.Output<string>; // ACM certificate for HTTPS
}

export interface AlbOutputs {
  albArn: pulumi.Output<string>;
  albDnsName: pulumi.Output<string>;
  albZoneId: pulumi.Output<string>;
  targetGroupArns: {
    api: pulumi.Output<string>;
    graphql: pulumi.Output<string>;
    sse: pulumi.Output<string>;
    providerApi: pulumi.Output<string>;
  };
  httpsListenerArn?: pulumi.Output<string>;
  httpsListenerResource?: aws.lb.Listener;
  httpListenerArn: pulumi.Output<string>;
}

/**
 * Create Application Load Balancer for ECS services
 */
export function createApplicationLoadBalancer(config: AlbConfig): aws.lb.LoadBalancer {
  return new aws.lb.LoadBalancer(`ohi-${config.environment}`, {
    name: `ohi-${config.environment}`,
    internal: false,
    loadBalancerType: 'application',
    securityGroups: [config.albSecurityGroupId],
    subnets: config.publicSubnetIds,
    enableDeletionProtection: config.environment === 'prod',
    enableHttp2: true,
    enableCrossZoneLoadBalancing: true,
    idleTimeout: 60,
    tags: { ...getResourceTags(config.environment, 'alb') },
  });
}

/**
 * Create target group for a service
 */
export function createTargetGroup(
  config: AlbConfig,
  serviceName: string,
  port: number,
  healthCheckPath: string = '/health'
): aws.lb.TargetGroup {
  return new aws.lb.TargetGroup(`ohi-${config.environment}-${serviceName}`, {
    name: `ohi-${config.environment}-${serviceName}`,
    port: port,
    protocol: 'HTTP',
    vpcId: config.vpcId,
    targetType: 'ip', // Required for Fargate
    deregistrationDelay: 30,
    healthCheck: {
      enabled: true,
      path: healthCheckPath,
      protocol: 'HTTP',
      matcher: '200',
      interval: 30,
      timeout: 5,
      healthyThreshold: 2,
      unhealthyThreshold: 3,
    },
    stickiness: {
      enabled: true,
      type: 'lb_cookie',
      cookieDuration: 86400, // 24 hours
    },
    tags: { ...getResourceTags(config.environment, `${serviceName}-tg`) },
  });
}

/**
 * Create HTTP listener that redirects to HTTPS
 */
export function createHttpListener(
  config: AlbConfig,
  alb: aws.lb.LoadBalancer
): aws.lb.Listener {
  return new aws.lb.Listener(`ohi-${config.environment}-http`, {
    loadBalancerArn: alb.arn,
    port: 80,
    protocol: 'HTTP',
    defaultActions: [
      {
        type: 'redirect',
        redirect: {
          port: '443',
          protocol: 'HTTPS',
          statusCode: 'HTTP_301',
        },
      },
    ],
    tags: { ...getResourceTags(config.environment, 'http-listener') },
  });
}

/**
 * Create HTTPS listener with default action
 */
export function createHttpsListener(
  config: AlbConfig,
  alb: aws.lb.LoadBalancer,
  defaultTargetGroup: aws.lb.TargetGroup
): aws.lb.Listener {
  if (!config.certificateArn) {
    throw new Error('Certificate ARN required for HTTPS listener');
  }

  return new aws.lb.Listener(`ohi-${config.environment}-https`, {
    loadBalancerArn: alb.arn,
    port: 443,
    protocol: 'HTTPS',
    sslPolicy: 'ELBSecurityPolicy-TLS13-1-2-2021-06',
    certificateArn: config.certificateArn,
    defaultActions: [
      {
        type: 'forward',
        targetGroupArn: defaultTargetGroup.arn,
      },
    ],
    tags: { ...getResourceTags(config.environment, 'https-listener') },
  });
}

/**
 * Create listener rule for host-based routing
 */
export function createListenerRule(
  config: AlbConfig,
  listener: aws.lb.Listener,
  targetGroup: aws.lb.TargetGroup,
  hostHeader: string,
  priority: number
): aws.lb.ListenerRule {
  return new aws.lb.ListenerRule(
    `ohi-${config.environment}-${hostHeader.replace(/\./g, '-')}`,
    {
      listenerArn: listener.arn,
      priority: priority,
      conditions: [
        {
          hostHeader: {
            values: [hostHeader],
          },
        },
      ],
      actions: [
        {
          type: 'forward',
          targetGroupArn: targetGroup.arn,
        },
      ],
      tags: { ...getResourceTags(config.environment, `rule-${hostHeader}`) },
    }
  );
}

/**
 * Create complete ALB infrastructure
 */
export function createAlbInfrastructure(config: AlbConfig): AlbOutputs {
  // Create ALB
  const alb = createApplicationLoadBalancer(config);

  // Create target groups
  const apiTargetGroup = createTargetGroup(config, 'api', 8080, '/health');
  const graphqlTargetGroup = createTargetGroup(config, 'graphql', 8080, '/health');
  const sseTargetGroup = createTargetGroup(config, 'sse', 8080, '/health');
  const providerApiTargetGroup = createTargetGroup(config, 'provider-api', 8080, '/health');

  // Create HTTP listener (redirects to HTTPS)
  const httpListener = createHttpListener(config, alb);

  // Determine domain names based on environment
  const baseDomain =
    config.environment === 'prod'
      ? 'ohealth-ng.com'
      : config.environment === 'staging'
        ? 'staging.ohealth-ng.com'
        : 'dev.ohealth-ng.com';

  const outputs: AlbOutputs = {
    albArn: alb.arn,
    albDnsName: alb.dnsName,
    albZoneId: alb.zoneId,
    targetGroupArns: {
      api: apiTargetGroup.arn,
      graphql: graphqlTargetGroup.arn,
      sse: sseTargetGroup.arn,
      providerApi: providerApiTargetGroup.arn,
    },
    httpListenerArn: httpListener.arn,
  };

  // Create HTTPS listener and rules if certificate provided
  if (config.certificateArn) {
    const httpsListener = createHttpsListener(config, alb, apiTargetGroup);
    outputs.httpsListenerArn = httpsListener.arn;
    outputs.httpsListenerResource = httpsListener;

    // Create host-based routing rules
    createListenerRule(config, httpsListener, apiTargetGroup, `api.${baseDomain}`, 100);
    createListenerRule(config, httpsListener, graphqlTargetGroup, `graphql.${baseDomain}`, 200);
    createListenerRule(config, httpsListener, sseTargetGroup, `sse.${baseDomain}`, 300);
    createListenerRule(config, httpsListener, providerApiTargetGroup, `provider.${baseDomain}`, 400);
  }

  return outputs;
}

/**
 * Get health check configuration for a service
 */
export function getHealthCheckConfig(serviceName: string): {
  path: string;
  interval: number;
  timeout: number;
  healthyThreshold: number;
  unhealthyThreshold: number;
} {
  // Common health check for all services
  return {
    path: '/health',
    interval: 30,
    timeout: 5,
    healthyThreshold: 2,
    unhealthyThreshold: 3,
  };
}

/**
 * Get target group attributes for a service
 */
export function getTargetGroupAttributes(
  serviceName: string
): { deregistrationDelay: number; stickinessEnabled: boolean } {
  // SSE needs longer deregistration delay and sticky sessions
  if (serviceName === 'sse') {
    return {
      deregistrationDelay: 300, // 5 minutes for long-lived connections
      stickinessEnabled: true,
    };
  }

  return {
    deregistrationDelay: 30,
    stickinessEnabled: true,
  };
}

/**
 * Get ALB access logs configuration
 */
export function getAlbAccessLogsConfig(
  environment: string,
  s3BucketName: string
): {
  enabled: boolean;
  bucket: string;
  prefix: string;
} {
  return {
    enabled: environment === 'prod', // Only enable for prod to reduce costs
    bucket: s3BucketName,
    prefix: `alb-logs/${environment}`,
  };
}
