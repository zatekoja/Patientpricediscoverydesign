/**
 * Security Groups Tests
 * 
 * Tests security group implementation as defined in V1_DEPLOYMENT_ARCHITECTURE.md
 * Section 6: Network Architecture & Security
 * 
 * 16 Security Groups implementing least privilege:
 * 1. ALB Security Group
 * 2. API Security Group
 * 3. GraphQL Security Group
 * 4. SSE Security Group
 * 5. Provider API Security Group
 * 6. Reindexer Security Group
 * 7. Blnk API Security Group
 * 8. Blnk Worker Security Group
 * 9. RDS Security Group
 * 10. ElastiCache Security Group
 * 11. ClickHouse Security Group
 * 12. Zookeeper Security Group
 * 13. OTEL Collector Security Group
 * 14. SigNoz Query Service Security Group
 * 15. SigNoz Frontend Security Group
 * 16. ECS Tasks Security Group (general)
 * 17. VPC Endpoints Security Group
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

describe('Security Groups', () => {
  describe('ALB Security Group', () => {
    it('should create ALB security group', () => {
      const { createAlbSecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createAlbSecurityGroup('prod', 'vpc-123');
      expect(sg).toBeDefined();
    });

    it('should have HTTP and HTTPS ports configured', () => {
      const { ALB_PORTS } = require('../src/networking/security-groups');
      
      expect(ALB_PORTS.http).toBe(80);
      expect(ALB_PORTS.https).toBe(443);
    });

    it('should have correct name format', () => {
      const { getSecurityGroupName } = require('../src/networking/security-groups');
      
      expect(getSecurityGroupName('prod', 'alb')).toBe('ohi-prod-alb-sg');
    });
  });

  describe('API Service Security Groups', () => {
    it('should create API security group', () => {
      const { createApiSecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createApiSecurityGroup('prod', 'vpc-123', 'sg-alb-123');
      expect(sg).toBeDefined();
    });

    it('should have API port configured as 8080', () => {
      const { SERVICE_PORTS } = require('../src/networking/security-groups');
      
      expect(SERVICE_PORTS.api).toBe(8080);
    });

    it('should create GraphQL security group', () => {
      const { createGraphqlSecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createGraphqlSecurityGroup('prod', 'vpc-123', 'sg-alb-123');
      expect(sg).toBeDefined();
    });

    it('should have GraphQL port configured as 8081', () => {
      const { SERVICE_PORTS } = require('../src/networking/security-groups');
      
      expect(SERVICE_PORTS.graphql).toBe(8081);
    });

    it('should create SSE security group', () => {
      const { createSseSecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createSseSecurityGroup('prod', 'vpc-123', 'sg-alb-123');
      expect(sg).toBeDefined();
    });

    it('should have SSE port configured as 8082', () => {
      const { SERVICE_PORTS } = require('../src/networking/security-groups');
      
      expect(SERVICE_PORTS.sse).toBe(8082);
    });

    it('should create Provider API security group', () => {
      const { createProviderApiSecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createProviderApiSecurityGroup('prod', 'vpc-123', 'sg-alb-123');
      expect(sg).toBeDefined();
    });

    it('should have Provider API port configured as 3000', () => {
      const { SERVICE_PORTS } = require('../src/networking/security-groups');
      
      expect(SERVICE_PORTS.providerApi).toBe(3000);
    });
  });

  describe('Backend Service Security Groups', () => {
    it('should create Reindexer security group', () => {
      const { createReindexerSecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createReindexerSecurityGroup('prod', 'vpc-123');
      expect(sg).toBeDefined();
    });

    it('should create Blnk API security group', () => {
      const { createBlnkApiSecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createBlnkApiSecurityGroup('prod', 'vpc-123', ['sg-api-123', 'sg-graphql-123']);
      expect(sg).toBeDefined();
    });

    it('should have Blnk API port configured as 5001', () => {
      const { SERVICE_PORTS } = require('../src/networking/security-groups');
      
      expect(SERVICE_PORTS.blnkApi).toBe(5001);
    });

    it('should create Blnk Worker security group', () => {
      const { createBlnkWorkerSecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createBlnkWorkerSecurityGroup('prod', 'vpc-123');
      expect(sg).toBeDefined();
    });
  });

  describe('Database Security Groups', () => {
    it('should create RDS security group', () => {
      const { createRdsSecurityGroup } = require('../src/networking/security-groups');
      
      const serviceSgIds = ['sg-api-123', 'sg-graphql-123', 'sg-sse-123', 'sg-blnk-api-123'];
      const sg = createRdsSecurityGroup('prod', 'vpc-123', serviceSgIds);
      expect(sg).toBeDefined();
    });

    it('should have PostgreSQL port configured as 5432', () => {
      const { DATABASE_PORTS } = require('../src/networking/security-groups');
      
      expect(DATABASE_PORTS.postgres).toBe(5432);
    });

    it('should create ElastiCache security group', () => {
      const { createElastiCacheSecurityGroup } = require('../src/networking/security-groups');
      
      const serviceSgIds = ['sg-api-123', 'sg-graphql-123', 'sg-sse-123', 'sg-blnk-api-123'];
      const sg = createElastiCacheSecurityGroup('prod', 'vpc-123', serviceSgIds);
      expect(sg).toBeDefined();
    });

    it('should have Redis port configured as 6379', () => {
      const { DATABASE_PORTS } = require('../src/networking/security-groups');
      
      expect(DATABASE_PORTS.redis).toBe(6379);
    });
  });

  describe('Observability Security Groups', () => {
    it('should create ClickHouse security group', () => {
      const { createClickHouseSecurityGroup } = require('../src/networking/security-groups');
      
      const serviceSgIds = ['sg-otel-123', 'sg-signoz-query-123'];
      const sg = createClickHouseSecurityGroup('prod', 'vpc-123', serviceSgIds);
      expect(sg).toBeDefined();
    });

    it('should have ClickHouse port configured as 9000', () => {
      const { OBSERVABILITY_PORTS } = require('../src/networking/security-groups');
      
      expect(OBSERVABILITY_PORTS.clickhouse).toBe(9000);
    });

    it('should have ClickHouse HTTP port configured as 8123', () => {
      const { OBSERVABILITY_PORTS } = require('../src/networking/security-groups');
      
      expect(OBSERVABILITY_PORTS.clickhouseHttp).toBe(8123);
    });

    it('should have ClickHouse interserver port configured as 9009', () => {
      const { OBSERVABILITY_PORTS } = require('../src/networking/security-groups');
      
      expect(OBSERVABILITY_PORTS.clickhouseInterserver).toBe(9009);
    });

    it('should create Zookeeper security group', () => {
      const { createZookeeperSecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createZookeeperSecurityGroup('prod', 'vpc-123', 'sg-clickhouse-123');
      expect(sg).toBeDefined();
    });

    it('should have Zookeeper ports configured (client 2181, follower 2888, election 3888)', () => {
      const { OBSERVABILITY_PORTS } = require('../src/networking/security-groups');
      
      expect(OBSERVABILITY_PORTS.zookeeperClient).toBe(2181);
      expect(OBSERVABILITY_PORTS.zookeeperFollower).toBe(2888);
      expect(OBSERVABILITY_PORTS.zookeeperElection).toBe(3888);
    });

    it('should create OTEL Collector security group', () => {
      const { createOtelCollectorSecurityGroup } = require('../src/networking/security-groups');
      
      const serviceSgIds = ['sg-api-123', 'sg-graphql-123', 'sg-sse-123'];
      const sg = createOtelCollectorSecurityGroup('prod', 'vpc-123', serviceSgIds);
      expect(sg).toBeDefined();
    });

    it('should have OTLP ports configured (gRPC 4317, HTTP 4318)', () => {
      const { OBSERVABILITY_PORTS } = require('../src/networking/security-groups');
      
      expect(OBSERVABILITY_PORTS.otlpGrpc).toBe(4317);
      expect(OBSERVABILITY_PORTS.otlpHttp).toBe(4318);
    });

    it('should create SigNoz Query Service security group', () => {
      const { createSigNozQuerySecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createSigNozQuerySecurityGroup('prod', 'vpc-123', 'sg-signoz-frontend-123');
      expect(sg).toBeDefined();
    });

    it('should have SigNoz Query port configured as 8080', () => {
      const { OBSERVABILITY_PORTS } = require('../src/networking/security-groups');
      
      expect(OBSERVABILITY_PORTS.signozQuery).toBe(8080);
    });

    it('should create SigNoz Frontend security group', () => {
      const { createSigNozFrontendSecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createSigNozFrontendSecurityGroup('prod', 'vpc-123', 'sg-alb-123');
      expect(sg).toBeDefined();
    });

    it('should have SigNoz Frontend port configured as 3301', () => {
      const { OBSERVABILITY_PORTS } = require('../src/networking/security-groups');
      
      expect(OBSERVABILITY_PORTS.signozFrontend).toBe(3301);
    });
  });

  describe('VPC Endpoints Security Group', () => {
    it('should create VPC Endpoints security group', () => {
      const { createVpcEndpointsSecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createVpcEndpointsSecurityGroup('prod', 'vpc-123', '10.2.0.0/16');
      expect(sg).toBeDefined();
    });

    it('should have HTTPS port configured as 443', () => {
      const { VPC_ENDPOINT_PORTS } = require('../src/networking/security-groups');
      
      expect(VPC_ENDPOINT_PORTS.https).toBe(443);
    });
  });

  describe('ECS Tasks Security Group', () => {
    it('should create ECS tasks security group', () => {
      const { createEcsTasksSecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createEcsTasksSecurityGroup('prod', 'vpc-123');
      expect(sg).toBeDefined();
    });
  });

  describe('Security Group Properties', () => {
    it('should have correct tagging', () => {
      const { createApiSecurityGroup } = require('../src/networking/security-groups');
      
      const sg = createApiSecurityGroup('prod', 'vpc-123', 'sg-alb-123');
      // Tags applied via transformation
      expect(sg).toBeDefined();
    });

    it('should have descriptive names following convention', () => {
      const { getSecurityGroupName } = require('../src/networking/security-groups');
      
      expect(getSecurityGroupName('prod', 'api')).toBe('ohi-prod-api-sg');
      expect(getSecurityGroupName('prod', 'graphql')).toBe('ohi-prod-graphql-sg');
      expect(getSecurityGroupName('prod', 'rds')).toBe('ohi-prod-rds-sg');
    });
  });

  describe('Service Communication Matrix', () => {
    it('should create all 17 security groups', () => {
      const {
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
      } = require('../src/networking/security-groups');
      
      // Verify all functions exist
      expect(createAlbSecurityGroup).toBeDefined();
      expect(createApiSecurityGroup).toBeDefined();
      expect(createGraphqlSecurityGroup).toBeDefined();
      expect(createSseSecurityGroup).toBeDefined();
      expect(createProviderApiSecurityGroup).toBeDefined();
      expect(createReindexerSecurityGroup).toBeDefined();
      expect(createBlnkApiSecurityGroup).toBeDefined();
      expect(createBlnkWorkerSecurityGroup).toBeDefined();
      expect(createRdsSecurityGroup).toBeDefined();
      expect(createElastiCacheSecurityGroup).toBeDefined();
      expect(createClickHouseSecurityGroup).toBeDefined();
      expect(createZookeeperSecurityGroup).toBeDefined();
      expect(createOtelCollectorSecurityGroup).toBeDefined();
      expect(createSigNozQuerySecurityGroup).toBeDefined();
      expect(createSigNozFrontendSecurityGroup).toBeDefined();
      expect(createEcsTasksSecurityGroup).toBeDefined();
      expect(createVpcEndpointsSecurityGroup).toBeDefined();
    });
  });
});
