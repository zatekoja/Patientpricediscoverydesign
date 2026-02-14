/**
 * Observability Module - ClickHouse + Zookeeper + Fluent Bit
 * 
 * Provides observability infrastructure for SigNoz:
 * - ClickHouse EC2 instances for telemetry storage
 * - Zookeeper ECS services for ClickHouse coordination  
 * - Fluent Bit configuration for log aggregation
 */

import * as aws from '@pulumi/aws';
import * as pulumi from '@pulumi/pulumi';
import { getResourceTags } from '../tagging';

export type Environment = 'dev' | 'staging' | 'prod';

/**
 * Configuration for observability infrastructure
 */
export interface ObservabilityConfig {
  environment: Environment;
  vpcId: pulumi.Input<string>;
  privateSubnetIds: pulumi.Input<string>[];
  ecsClusterId: pulumi.Input<string>;
  taskExecutionRoleArn: pulumi.Input<string>;
  clickhouseSecurityGroupId: pulumi.Input<string>;
  zookeeperSecurityGroupId: pulumi.Input<string>;
  clickhouseVolumeSize?: number;
  useInstanceStore?: boolean;
}

/**
 * ClickHouse EC2 configuration
 */
export interface ClickHouseConfig {
  environment: Environment;
  vpcId: pulumi.Input<string>;
  privateSubnetIds: pulumi.Input<string>[];
  clickhouseSecurityGroupId: pulumi.Input<string>;
  volumeSize?: number;
  useInstanceStore?: boolean;
}

/**
 * Observability infrastructure outputs
 */
export interface ObservabilityOutputs {
  clickhouseInstanceId: pulumi.Output<string>;
  clickhousePrivateIp: pulumi.Output<string>;
  clickhousePublicIp?: pulumi.Output<string>;
  dataVolumeId?: pulumi.Output<string>;
  zookeeperServiceArn: pulumi.Output<string>;
  zookeeperServiceDiscoveryEndpoint: pulumi.Output<string>;
  securityGroupId: pulumi.Output<string>;
}

/**
 * Fluent Bit configuration options
 */
export interface FluentBitConfig {
  environment: string;
  service: string;
  otelCollectorEndpoint: string;
}

/**
 * Get appropriate ClickHouse instance type based on environment
 */
export function getClickHouseInstanceType(environment: Environment): string {
  switch (environment) {
    case 'prod':
      return 'i3en.large'; // Instance store NVMe, 2 vCPU, 16 GB RAM, 1.25 TB NVMe
    case 'staging':
      return 't3.large'; // 2 vCPU, 8 GB RAM
    case 'dev':
      return 't3.medium'; // 2 vCPU, 4 GB RAM
    default:
      return 't3.medium';
  }
}

/**
 * Get storage size for ClickHouse based on environment
 */
export function getClickHouseStorageSize(environment: Environment): number {
  switch (environment) {
    case 'prod':
      return 500; // 500 GB
    case 'staging':
      return 200; // 200 GB
    case 'dev':
      return 100; // 100 GB
    default:
      return 100;
  }
}

/**
 * Generate ClickHouse user data script
 */
export function generateClickHouseUserData(
  environment: string,
  zookeeperHosts: string
): string {
  return `#!/bin/bash
set -e

# Update system
apt-get update
apt-get install -y apt-transport-https ca-certificates dirmngr

# Add ClickHouse repository
apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv 8919F6BD2B48D754
echo "deb https://packages.clickhouse.com/deb stable main" | tee /etc/apt/sources.list.d/clickhouse.list

# Install ClickHouse
apt-get update
apt-get install -y clickhouse-server clickhouse-client

# Create data directory
mkdir -p /var/lib/clickhouse
chown -R clickhouse:clickhouse /var/lib/clickhouse

# Configure ClickHouse
cat > /etc/clickhouse-server/config.d/network.xml <<EOF
<clickhouse>
  <listen_host>0.0.0.0</listen_host>
  <http_port>8123</http_port>
  <tcp_port>9000</tcp_port>
  <interserver_http_port>9009</interserver_http_port>
</clickhouse>
EOF

# Configure Zookeeper integration
cat > /etc/clickhouse-server/config.d/zookeeper.xml <<EOF
<clickhouse>
  <zookeeper>
${zookeeperHosts.split(',').map((host, idx) => `    <node index="${idx + 1}">
      <host>${host.split(':')[0]}</host>
      <port>${host.split(':')[1] || '2181'}</port>
    </node>`).join('\n')}
  </zookeeper>
</clickhouse>
EOF

# Configure data paths
cat > /etc/clickhouse-server/config.d/storage.xml <<EOF
<clickhouse>
  <path>/var/lib/clickhouse/</path>
  <tmp_path>/var/lib/clickhouse/tmp/</tmp_path>
  <user_files_path>/var/lib/clickhouse/user_files/</user_files_path>
  <format_schema_path>/var/lib/clickhouse/format_schemas/</format_schema_path>
</clickhouse>
EOF

# Configure logging
cat > /etc/clickhouse-server/config.d/logging.xml <<EOF
<clickhouse>
  <logger>
    <level>information</level>
    <log>/var/log/clickhouse-server/clickhouse-server.log</log>
    <errorlog>/var/log/clickhouse-server/clickhouse-server.err.log</errorlog>
    <size>1000M</size>
    <count>10</count>
  </logger>
</clickhouse>
EOF

# Start ClickHouse
systemctl enable clickhouse-server
systemctl start clickhouse-server

# Wait for ClickHouse to start
sleep 10

# Create SigNoz databases
clickhouse-client --query "CREATE DATABASE IF NOT EXISTS signoz_traces"
clickhouse-client --query "CREATE DATABASE IF NOT EXISTS signoz_metrics"
clickhouse-client --query "CREATE DATABASE IF NOT EXISTS signoz_logs"

echo "ClickHouse installation complete"
`;
}

/**
 * Create ClickHouse EC2 instance
 */
export function createClickHouseInstance(config: ClickHouseConfig): aws.ec2.Instance {
  const { environment, privateSubnetIds, clickhouseSecurityGroupId } = config;

  const instanceType = getClickHouseInstanceType(environment);
  const useInstanceStore = config.useInstanceStore ?? (environment === 'prod');

  // Get latest Ubuntu 22.04 AMI
  const ami = aws.ec2.getAmi({
    mostRecent: true,
    owners: ['099720109477'], // Canonical
    filters: [
      { name: 'name', values: ['ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*'] },
      { name: 'virtualization-type', values: ['hvm'] },
    ],
  });

  // Build Zookeeper hosts from service discovery DNS names
  const zookeeperCount = environment === 'prod' ? 3 : 1;
  const zookeeperHosts = Array.from(
    { length: zookeeperCount },
    (_, i) => `zookeeper-${i}.ohi-${environment}.local:2181`
  ).join(',');
  const userData = generateClickHouseUserData(environment, zookeeperHosts);

  const instance = new aws.ec2.Instance(
    `ohi-${environment}-clickhouse`,
    {
      instanceType,
      ami: ami.then(a => a.id),
      subnetId: pulumi.output(privateSubnetIds[0]),
      vpcSecurityGroupIds: [clickhouseSecurityGroupId],
      userData,
      tags: {
        ...getResourceTags(environment, 'clickhouse'),
        Name: `ohi-${environment}-clickhouse`,
        DataClassification: 'internal',
        BackupPolicy: environment === 'prod' ? 'daily' : 'weekly',
      },
      // Instance store volumes (i3en.large has 1.25 TB NVMe)
      ...(useInstanceStore ? {
        ephemeralBlockDevices: [{
          deviceName: '/dev/sdb',
          virtualName: 'ephemeral0',
        }],
      } : {}),
    }
  );

  return instance;
}

/**
 * Create EBS volume for ClickHouse data (when not using instance store)
 */
export function createClickHouseVolume(
  environment: Environment,
  size: number,
  availabilityZone: pulumi.Output<string>
): aws.ebs.Volume {
  const volume = new aws.ebs.Volume(
    `ohi-${environment}-clickhouse-data`,
    {
      availabilityZone,
      size,
      type: 'gp3',
      iops: 3000,
      throughput: 125,
      encrypted: true,
      tags: {
        ...getResourceTags(environment, 'clickhouse'),
        Name: `ohi-${environment}-clickhouse-data`,
        VolumeType: 'data',
      },
    }
  );

  return volume;
}

/**
 * Attach EBS volume to EC2 instance
 */
export function attachVolume(
  environment: Environment,
  instanceId: pulumi.Output<string>,
  volumeId: pulumi.Output<string>
): aws.ec2.VolumeAttachment {
  const attachment = new aws.ec2.VolumeAttachment(
    `ohi-${environment}-clickhouse-volume-attachment`,
    {
      instanceId,
      volumeId,
      deviceName: '/dev/sdf',
    }
  );

  return attachment;
}

/**
 * Generate Fluent Bit configuration
 */
export function generateFluentBitConfig(config: FluentBitConfig): string {
  const { environment, service, otelCollectorEndpoint } = config;

  return `[SERVICE]
    Flush        5
    Daemon       Off
    Log_Level    info
    Parsers_File parsers.conf

[INPUT]
    Name              forward
    Listen            0.0.0.0
    Port              24224
    Buffer_Chunk_Size 1M
    Buffer_Max_Size   6M

[FILTER]
    Name    record_modifier
    Match   *
    Record  service ${service}
    Record  environment ${environment}
    Record  source fluent-bit

[FILTER]
    Name    grep
    Match   *
    Exclude log ^\\s*$

[OUTPUT]
    Name        http
    Match       *
    Host        ${otelCollectorEndpoint.split(':')[0]}
    Port        ${otelCollectorEndpoint.split(':')[1] || '4318'}
    URI         /v1/logs
    Format      json
    Json_date_key    timestamp
    Json_date_format iso8601
    Retry_Limit 5

[OUTPUT]
    Name        stdout
    Match       *
    Format      json_lines
`;
}

/**
 * Create Fluent Bit sidecar container definition
 */
export function createFluentBitSidecar(config: FluentBitConfig): any {
  const fluentBitConfig = generateFluentBitConfig(config);

  return {
    name: 'fluent-bit',
    image: 'fluent/fluent-bit:2.2',
    essential: false,
    environment: [
      { name: 'SERVICE_NAME', value: config.service },
      { name: 'ENVIRONMENT', value: config.environment },
    ],
    logConfiguration: {
      logDriver: 'awslogs',
      options: {
        'awslogs-group': `/ecs/ohi-${config.environment}-fluent-bit`,
        'awslogs-region': 'eu-west-1',
        'awslogs-stream-prefix': 'fluent-bit',
      },
    },
    firelensConfiguration: {
      type: 'fluentbit',
      options: {
        'config-file-type': 'file',
        'config-file-value': '/fluent-bit/etc/fluent-bit.conf',
      },
    },
    mountPoints: [
      {
        sourceVolume: 'logs',
        containerPath: '/var/log',
        readOnly: false,
      },
    ],
  };
}

/**
 * Calculate observability infrastructure cost
 */
export function calculateObservabilityCost(
  environment: Environment,
  options: {
    clickhouseInstanceType: string;
    zookeeperInstances: number;
    storageGB: number;
  }
): { clickhouse: number; zookeeper: number; storage: number; total: number } {
  // Instance pricing (eu-west-1)
  const instancePricing: Record<string, number> = {
    'i3en.large': 0.226 * 730, // $165/month
    't3.large': 0.0832 * 730, // $61/month
    't3.medium': 0.0416 * 730, // $30/month
  };

  // Fargate pricing for Zookeeper (512 MB, 0.25 vCPU)
  const zookeeperCostPerInstance = 15; // ~$15/month per instance

  // EBS gp3 pricing
  const storageCost = options.storageGB * 0.08; // $0.08/GB/month

  const clickhouseCost = instancePricing[options.clickhouseInstanceType] || 0;
  const zookeeperCost = options.zookeeperInstances * zookeeperCostPerInstance;

  return {
    clickhouse: Math.round(clickhouseCost),
    zookeeper: Math.round(zookeeperCost),
    storage: Math.round(storageCost),
    total: Math.round(clickhouseCost + zookeeperCost + storageCost),
  };
}

/**
 * Create complete observability infrastructure
 */
export function createObservabilityInfrastructure(
  config: ObservabilityConfig
): ObservabilityOutputs {
  const { environment, privateSubnetIds } = config;

  const volumeSize = config.clickhouseVolumeSize || getClickHouseStorageSize(environment);
  const useInstanceStore = config.useInstanceStore ?? (environment === 'prod');

  // Create ClickHouse instance
  const clickhouseInstance = createClickHouseInstance({
    environment,
    vpcId: config.vpcId,
    privateSubnetIds,
    clickhouseSecurityGroupId: config.clickhouseSecurityGroupId,
    volumeSize,
    useInstanceStore,
  });

  // Create EBS volume if not using instance store
  let dataVolume: aws.ebs.Volume | undefined;
  let volumeAttachment: aws.ec2.VolumeAttachment | undefined;

  if (!useInstanceStore) {
    dataVolume = createClickHouseVolume(
      environment,
      volumeSize,
      clickhouseInstance.availabilityZone
    );

    volumeAttachment = attachVolume(
      environment,
      clickhouseInstance.id,
      dataVolume.id
    );
  }

  // Zookeeper service discovery endpoint (will be created by ECS module)
  const zookeeperEndpoint = pulumi.output(
    `zookeeper.ohi-${environment}.local:2181`
  );

  // Placeholder outputs for Zookeeper (actual service created in ECS module)
  const zookeeperServiceArn = pulumi.output(`arn:aws:ecs:eu-west-1:123:service/ohi-${environment}/zookeeper`);

  return {
    clickhouseInstanceId: clickhouseInstance.id,
    clickhousePrivateIp: clickhouseInstance.privateIp,
    clickhousePublicIp: clickhouseInstance.publicIp,
    dataVolumeId: dataVolume?.id,
    zookeeperServiceArn,
    zookeeperServiceDiscoveryEndpoint: zookeeperEndpoint,
    securityGroupId: pulumi.output(config.clickhouseSecurityGroupId),
  };
}

/**
 * Export helpers for testing and external usage
 */
export const helpers = {
  getClickHouseInstanceType,
  getZookeeperCount: (environment: Environment) => {
    return environment === 'prod' ? 3 : 1; // Quorum requires 3 nodes
  },
  getClickHouseStorageSize,
};
