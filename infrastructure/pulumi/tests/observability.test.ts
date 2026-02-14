/**
 * Observability Infrastructure Tests
 * 
 * Tests for ClickHouse EC2, Zookeeper ECS, and Fluent Bit configuration
 */

import * as pulumi from '@pulumi/pulumi';

// Mock Pulumi runtime
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

describe('Observability Infrastructure', () => {
  const clickhouseModule = require('../src/observability/clickhouse');
  const ecsModule = require('../src/compute/ecs');

  describe('ClickHouse Helper Functions', () => {
    it('should return correct instance type for prod', () => {
      const type = clickhouseModule.getClickHouseInstanceType('prod');
      expect(type).toBe('i3en.large');
    });

    it('should return correct instance type for dev', () => {
      const type = clickhouseModule.getClickHouseInstanceType('dev');
      expect(type).toBe('t3.medium');
    });

    it('should return correct storage size for prod', () => {
      const size = clickhouseModule.getClickHouseStorageSize('prod');
      expect(size).toBe(500);
    });

    it('should return correct storage size for dev', () => {
      const size = clickhouseModule.getClickHouseStorageSize('dev');
      expect(size).toBe(100);
    });

    it('should expose helpers object', () => {
      expect(clickhouseModule.helpers).toBeDefined();
      expect(clickhouseModule.helpers.getClickHouseInstanceType).toBeInstanceOf(Function);
      expect(clickhouseModule.helpers.getZookeeperCount).toBeInstanceOf(Function);
    });

    it('should return 3 Zookeeper instances for prod', () => {
      const count = clickhouseModule.helpers.getZookeeperCount('prod');
      expect(count).toBe(3);
    });

    it('should return 1 Zookeeper instance for dev', () => {
      const count = clickhouseModule.helpers.getZookeeperCount('dev');
      expect(count).toBe(1);
    });
  });

  describe('ClickHouse User Data Script', () => {
    it('should generate user data with ClickHouse installation', () => {
      const userData = clickhouseModule.generateClickHouseUserData('prod', 'zk:2181');
      
      expect(userData).toContain('apt-get install');
      expect(userData).toContain('clickhouse-server');
      expect(userData).toContain('clickhouse-client');
    });

    it('should configure Zookeeper in user data', () => {
      const userData = clickhouseModule.generateClickHouseUserData('prod', 'zk1:2181,zk2:2181');
      
      expect(userData).toContain('<zookeeper>');
      expect(userData).toContain('2181');
    });

    it('should create SigNoz databases', () => {
      const userData = clickhouseModule.generateClickHouseUserData('prod', 'zk:2181');
      
      expect(userData).toContain('signoz_traces');
      expect(userData).toContain('signoz_metrics');
      expect(userData).toContain('signoz_logs');
    });
  });

  describe('ClickHouse EC2 Resources', () => {
    it('should create ClickHouse EC2 instance', () => {
      const instance = clickhouseModule.createClickHouseInstance({
        environment: 'prod',
        vpcId: pulumi.output('vpc-123'),
        privateSubnetIds: [pulumi.output('subnet-1')],
        clickhouseSecurityGroupId: pulumi.output('sg-clickhouse'),
      });

      expect(instance).toBeDefined();
    });

    it('should create EBS volume', () => {
      const volume = clickhouseModule.createClickHouseVolume(
        'dev',
        100,
        pulumi.output('eu-west-1a')
      );

      expect(volume).toBeDefined();
    });

    it('should attach EBS volume to instance', () => {
      const attachment = clickhouseModule.attachVolume(
        'dev',
        pulumi.output('i-123'),
        pulumi.output('vol-123')
      );

      expect(attachment).toBeDefined();
    });
  });

  describe('Fluent Bit Configuration', () => {
    it('should generate Fluent Bit config', () => {
      const config = clickhouseModule.generateFluentBitConfig({
        environment: 'prod',
        service: 'api',
        otelCollectorEndpoint: 'otel:4318',
      });

      expect(config).toContain('[SERVICE]');
      expect(config).toContain('[INPUT]');
      expect(config).toContain('[OUTPUT]');
      expect(config).toContain('otel');
    });

    it('should include service and environment in config', () => {
      const config = clickhouseModule.generateFluentBitConfig({
        environment: 'staging',
        service: 'graphql',
        otelCollectorEndpoint: 'otel:4318',
      });

      expect(config).toContain('service graphql');
      expect(config).toContain('environment staging');
    });

    it('should create Fluent Bit sidecar container', () => {
      const sidecar = clickhouseModule.createFluentBitSidecar({
        environment: 'prod',
        service: 'api',
        otelCollectorEndpoint: 'otel:4318',
      });

      expect(sidecar).toBeDefined();
      expect(sidecar.name).toBe('fluent-bit');
      expect(sidecar.image).toContain('fluent/fluent-bit');
    });

    it('should configure Fluent Bit with firelens', () => {
      const sidecar = clickhouseModule.createFluentBitSidecar({
        environment: 'prod',
        service: 'api',
        otelCollectorEndpoint: 'otel:4318',
      });

      expect(sidecar.firelensConfiguration).toBeDefined();
      expect(sidecar.firelensConfiguration.type).toBe('fluentbit');
    });
  });

  describe('Zookeeper ECS Resources', () => {
    it('should create Zookeeper task definition', () => {
      const taskDef = ecsModule.createZookeeperTaskDefinition(
        'prod',
        pulumi.output('arn:aws:iam::123:role/exec'),
        0
      );

      expect(taskDef).toBeDefined();
    });

    it('should create Zookeeper service', () => {
      const service = ecsModule.createZookeeperService(
        'prod',
        pulumi.output('cluster-1'),
        pulumi.output('task-def-arn'),
        [pulumi.output('subnet-1')],
        pulumi.output('sg-zk'),
        pulumi.output('namespace-id'),
        0
      );

      expect(service).toBeDefined();
    });
  });

  describe('Observability Infrastructure', () => {
    it('should create complete observability infrastructure', () => {
      const outputs = clickhouseModule.createObservabilityInfrastructure({
        environment: 'prod',
        vpcId: pulumi.output('vpc-123'),
        privateSubnetIds: [pulumi.output('subnet-1')],
        ecsClusterId: pulumi.output('cluster-1'),
        taskExecutionRoleArn: pulumi.output('arn:aws:iam::123:role/exec'),
        clickhouseSecurityGroupId: pulumi.output('sg-clickhouse'),
        zookeeperSecurityGroupId: pulumi.output('sg-zk'),
      });

      expect(outputs).toBeDefined();
      expect(outputs.clickhouseInstanceId).toBeDefined();
      expect(outputs.clickhousePrivateIp).toBeDefined();
      expect(outputs.zookeeperServiceArn).toBeDefined();
    });

    it('should include data volume for non-instance-store', () => {
      const outputs = clickhouseModule.createObservabilityInfrastructure({
        environment: 'dev',
        vpcId: pulumi.output('vpc-123'),
        privateSubnetIds: [pulumi.output('subnet-1')],
        ecsClusterId: pulumi.output('cluster-1'),
        taskExecutionRoleArn: pulumi.output('arn:aws:iam::123:role/exec'),
        clickhouseSecurityGroupId: pulumi.output('sg-clickhouse'),
        zookeeperSecurityGroupId: pulumi.output('sg-zk'),
        useInstanceStore: false,
      });

      expect(outputs.dataVolumeId).toBeDefined();
    });
  });

  describe('Cost Calculation', () => {
    it('should calculate cost for prod', () => {
      const cost = clickhouseModule.calculateObservabilityCost('prod', {
        clickhouseInstanceType: 'i3en.large',
        zookeeperInstances: 3,
        storageGB: 500,
      });

      expect(cost.clickhouse).toBeGreaterThan(0);
      expect(cost.zookeeper).toBeGreaterThan(0);
      expect(cost.total).toBeGreaterThan(0);
    });

    it('should calculate cost for dev', () => {
      const cost = clickhouseModule.calculateObservabilityCost('dev', {
        clickhouseInstanceType: 't3.medium',
        zookeeperInstances: 1,
        storageGB: 100,
      });

      expect(cost.total).toBeLessThan(100);
    });
  });
});
