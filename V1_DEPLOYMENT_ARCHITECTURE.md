# V1 Deployment Architecture: AWS + Pulumi

**Date:** February 13, 2026  
**Status:** Planning Phase  
**Target:** Production V1 Launch

---

## Executive Summary

Migration from GCP/Terraform to AWS/Pulumi for cloud-agnostic infrastructure. Cost-optimized architecture for Nigerian healthcare platform with **super strict resource tagging** for shared AWS account environment.

---

## 1. Service Architecture Analysis: Self-Host vs Managed

### Core Principle: Maximize Managed Services, Minimize Operational Overhead

| Service | Current Stack | AWS Strategy | Monthly Cost Estimate | Rationale |
|---------|---------------|--------------|----------------------|-----------|
| **PostgreSQL** | postgres:15-alpine | **AWS RDS PostgreSQL 15** Multi-AZ + 2 Read Replicas | ~$200-400 (db.t4g.medium) | Critical data, automated backups, point-in-time recovery. RDS manages patches/HA. |
| **Redis** | redis:7-alpine | **AWS ElastiCache Redis 7** | ~$50-100 (cache.t4g.small) | Fully managed, automatic failover, reduced ops burden. |
| **MongoDB** | mongo:6.0 | **MongoDB Atlas M10** (AWS eu-west-1) | ~$57/month (M10 shared) | ✅ **RECOMMENDED over DocumentDB**. True MongoDB 6.0 compatibility, provider-api uses aggregation pipelines. DocumentDB has compatibility gaps. Atlas has better tooling, can deploy on AWS infrastructure (zero data egress). |
| **Typesense** | typesense:27.1 | **Typesense Cloud** Starter Plan | ~$29-99/month | ✅ **RECOMMENDED**. Existing codebase tightly coupled to Typesense API. OpenSearch would require significant refactoring. Cloud version eliminates ops overhead, automatic backups/scaling. |
| **Blnk Postgres** | postgres:15-alpine | **AWS RDS PostgreSQL 15** (Separate instance) | ~$100-200 (db.t4g.small) | Financial ledger requires separate isolated database for compliance/security. |
| **Blnk Redis** | redis:7-alpine | **AWS ElastiCache Redis 7** (Separate cluster) | ~$50 (cache.t4g.micro) | Dedicated cache for financial transactions. |
| **Vault** | hashicorp/vault:1.15 (DEV mode) | **AWS Secrets Manager** | ~$1-5/month (pay per secret) | Native AWS integration, IAM-based access control, automatic rotation support. Eliminates Vault operational complexity. |
| **ClickHouse** | clickhouse:24.1-alpine | **Self-hosted on EC2** (i3en.large or t3.medium) | ~$70-150/month | ClickHouse needs local storage performance. AWS doesn't offer managed ClickHouse. EC2 with instance store or EBS gp3. |
| **Zookeeper** | zookeeper:3.7.0 | **Self-hosted on ECS Fargate** (512 MB) | ~$15-20/month | Only for ClickHouse coordination. Lightweight container. |

### Application Services: ECS Fargate with Spot Pricing

| Service | Current Stack | AWS Strategy | Monthly Cost Estimate | Rationale |
|---------|---------------|--------------|----------------------|-----------|
| **API** | Go 1.25 | **ECS Fargate** (0.5 vCPU, 1 GB) | ~$15-30/task/month | Stateless REST API, auto-scales 2-10 tasks. Use Fargate Spot for dev/staging. |
| **GraphQL** | Go 1.25 | **ECS Fargate** (0.5 vCPU, 1 GB) | ~$15-30/task/month | Stateless GraphQL gateway, 1-5 tasks. |
| **SSE** | Go 1.25 | **ECS Fargate** (0.25 vCPU, 512 MB) | ~$8-15/task/month | Long-lived connections, 1-3 tasks. Cannot use Spot (connection disruption). |
| **Provider API** | Node.js 20 | **ECS Fargate** (0.5 vCPU, 1 GB) | ~$15-30/task/month | Data ingestion service, 1-2 tasks. |
| **Reindexer** | Go 1.25 | **ECS Fargate** (0.25 vCPU, 512 MB) | ~$8-15/task/month | Background job, single task, can run on Spot. |
| **Blnk** | Go 1.25 | **ECS Fargate** (0.5 vCPU, 1 GB) | ~$15-30/task/month | Financial ledger API, 1-2 tasks. |
| **Blnk Worker** | Go 1.25 | **ECS Fargate** (0.25 vCPU, 512 MB) | ~$8-15/task/month | Background worker, single task. |
| **Frontend** | React + Nginx | **CloudFront + S3** | ~$5-20/month (data transfer) | Static SPA, served from S3, cached at edge via CloudFront. No compute costs. |

### Observability Stack: Hybrid Approach

| Service | Current Stack | AWS Strategy | Monthly Cost Estimate | Rationale |
|---------|---------------|--------------|----------------------|-----------|
| **SigNoz Frontend** | signoz/frontend:0.39.0 | **ECS Fargate** (0.25 vCPU, 512 MB) | ~$8-15/month | Static UI for observability dashboard, 1 task. |
| **SigNoz Query Service** | signoz/query-service:0.39.0 | **ECS Fargate** (0.5 vCPU, 1 GB) | ~$15-30/month | Query API for dashboards, 1-2 tasks. |
| **SigNoz OTEL Collector** | signoz/otel-collector:0.88.11 | **ECS Fargate** (0.5 vCPU, 1 GB) | ~$15-30/month | Receives telemetry from all services, 2-3 tasks behind ALB. |
| **Fluent Bit** | fluent/fluent-bit:2.2 | **CloudWatch Logs + Firehose** | ~$10-30/month | Replace Fluent Bit with native AWS log aggregation. Simpler, less to manage. |
| **Exporters** (postgres, redis, mongodb) | Various Prometheus exporters | **CloudWatch Agent + Custom Metrics** | Included in EC2/ECS costs | Native AWS monitoring integration, no separate containers needed. |

### Estimated Total Monthly Cost (V1 Production)

| Category | Cost Range |
|----------|-----------|
| **Databases** (RDS x2, ElastiCache x2, MongoDB Atlas, Typesense Cloud) | $486-$956 |
| **Compute** (ECS Fargate - 10-15 tasks average) | $150-$300 |
| **Storage** (EC2 for ClickHouse) | $70-$150 |
| **Networking** (ALB, CloudFront, data transfer) | $50-$100 |
| **Secrets/Monitoring** (Secrets Manager, CloudWatch) | $20-$50 |
| **TOTAL** | **$776 - $1,556/month** |

**Cost Optimization Opportunities:**
- **Fargate Spot**: Save 70% on dev/staging and non-critical workloads (reindexer, workers)
- **Reserved Instances**: After 3 months, commit to RDS 1-year RI (save ~30%)
- **CloudFront Free Tier**: First 1 TB data transfer free (covers initial V1 traffic)
- **Right-sizing**: Start small (t4g instances), scale based on actual metrics
- **Dev/Staging**: Use Aurora Serverless v2 or smaller instance classes

**Estimated V1 Dev Environment:** ~$300-$500/month (smaller instances, Fargate Spot, shared resources)

---

## 2. MongoDB Atlas vs AWS DocumentDB: Detailed Analysis

### Why MongoDB Atlas is the Better Choice

| Factor | MongoDB Atlas | AWS DocumentDB | Winner |
|--------|---------------|----------------|---------|
| **MongoDB Compatibility** | 100% (Official MongoDB) | ~90-95% (Fork at 4.0) | ✅ Atlas |
| **Aggregation Pipelines** | Full support (all operators) | Limited (missing $lookup variants, $facet issues) | ✅ Atlas |
| **Provider-API Code** | Works unchanged | May require refactoring | ✅ Atlas |
| **Tooling** | MongoDB Compass, Atlas UI, Atlas Search | AWS Console, limited tools | ✅ Atlas |
| **Data Locality** | Can deploy on AWS eu-west-1 | AWS eu-west-1 only | Tie |
| **Data Egress Costs** | $0 (AWS to Atlas on AWS) | $0 (within AWS) | Tie |
| **Backups** | Continuous (PITR), free snapshots | Automated snapshots (extra cost) | ✅ Atlas |
| **Monitoring** | Atlas Performance Advisor, Query Profiler | CloudWatch (basic) | ✅ Atlas |
| **Atlas Search** | Integrated full-text search (Lucene) | Must use OpenSearch separately | ✅ Atlas |
| **Cost (M10 Shared)** | $57/month (10 GB storage) | ~$100-150/month (db.t3.medium) | ✅ Atlas |
| **Scaling** | Instant (click to upgrade) | Manual (requires downtime or replica promotion) | ✅ Atlas |
| **Multi-Cloud** | Deploy on AWS, GCP, Azure | AWS only | ✅ Atlas |

### Technical Compatibility Check

**Current Provider-API MongoDB Usage:** (from docker-compose)
```yaml
provider-api:
  image: custom (Dockerfile.provider)
  environment:
    - PROVIDER_MONGO_URI=mongodb://mongo:27017
    - PROVIDER_MONGO_DB=provider_data
    - PROVIDER_MONGO_COLLECTION=price_lists
```

**Atlas Connection String (Zero Code Change):**
```bash
PROVIDER_MONGO_URI=mongodb+srv://cluster.mongodb.net/?retryWrites=true&w=majority
PROVIDER_MONGO_DB=provider_data
PROVIDER_MONGO_COLLECTION=price_lists
```

**Verdict:** ✅ **Use MongoDB Atlas M10 on AWS eu-west-1**
- Zero code changes
- Better performance and tooling
- Lower cost than DocumentDB
- True MongoDB 6.0 compatibility
- Can upgrade to dedicated clusters (M30+) later without migration

---

## 3. Typesense Cloud vs Self-Hosted vs AWS OpenSearch

### Why Typesense Cloud is the Better Choice for V1

| Factor | Typesense Cloud | Self-Hosted Typesense (ECS) | AWS OpenSearch | Winner |
|--------|-----------------|----------------------------|----------------|---------|
| **Code Changes** | Zero | Zero | Full rewrite (different API) | ✅ Cloud/Self-hosted |
| **Ops Overhead** | Zero (managed) | Medium (backups, scaling, monitoring) | Medium (AWS-managed but complex) | ✅ Cloud |
| **Cost (V1 traffic)** | $29-99/month (Starter-Growth) | ~$30-60/month (ECS Fargate 0.5 vCPU, 1 GB) | ~$150-300/month (t3.small cluster + EBS) | ✅ Cloud |
| **Backup/HA** | Automatic | Manual (need scripts) | Automatic snapshots | ✅ Cloud/OpenSearch |
| **Performance** | Optimized infrastructure | Depends on instance size | Good for large datasets | ✅ Cloud (for V1 scale) |
| **Scalability** | Instant (dashboard) | Manual (redeploy tasks) | Good (but slower than Typesense) | ✅ Cloud |
| **Latency** | <50ms (multi-region) | Depends on deployment | <50ms (in-region) | Tie |
| **Data Transfer** | Ingestion via API | No egress (in AWS) | No egress (in AWS) | ✅ Self-hosted/OpenSearch |
| **Learning Curve** | None (existing) | Low (Docker knowledge) | High (new API, DSL) | ✅ Cloud/Self-hosted |

### Current Typesense Usage

**From docker-compose.yml:**
```yaml
typesense:
  image: typesense/typesense:27.1
  ports: [8108]
  environment:
    TYPESENSE_API_KEY: xyz
    TYPESENSE_DATA_DIR: /data
    TYPESENSE_ENABLE_CORS: true
```

**Reindexer Service** (backend/Dockerfile.indexer):
- Syncs PostgreSQL → Typesense
- Keeps search indexes up to date

**Frontend/Backend Integration:**
- Facility search
- Service/procedure search
- Geolocation-based queries

### Migration Effort for OpenSearch

**Required Changes:**
1. ❌ Rewrite all search queries (Typesense API → OpenSearch DSL)
2. ❌ Refactor reindexer service (different indexing API)
3. ❌ Update frontend search components
4. ❌ Test all search functionality (facility, service, geolocation)
5. ❌ Learn OpenSearch query language

**Estimated Engineering Time:** 1-2 weeks + testing

**Verdict:** ✅ **Use Typesense Cloud Starter Plan ($29/month) for V1**
- Zero code changes
- Zero ops overhead
- Lowest total cost (engineering time + infrastructure)
- Can self-host later if needed (same API)
- Focus engineering effort on V1 launch features, not infrastructure migration

**Alternative for Cost Reduction:** Self-host Typesense on ECS Fargate if budget is extremely tight (~$30-40/month savings), but increases operational complexity.

---

## 4. Super Strict Tagging Strategy (Shared AWS Account)

### Core Tagging Requirements

**All resources MUST have these tags:**

| Tag Key | Description | Example | Required |
|---------|-------------|---------|----------|
| `Project` | Project identifier | `open-health-initiative` | ✅ Yes |
| `Environment` | Deployment environment | `dev`, `staging`, `prod` | ✅ Yes |
| `Owner` | Team/person responsible | `platform-team` | ✅ Yes |
| `CostCenter` | Billing/budget allocation | `ohi-infrastructure` | ✅ Yes |
| `Service` | Microservice name | `api`, `graphql`, `frontend` | ✅ Yes |
| `ManagedBy` | IaC tool managing resource | `pulumi` | ✅ Yes |
| `CreatedBy` | Tool/pipeline that created resource | `github-actions`, `pulumi-cli` | ✅ Yes |
| `CreatedDate` | Creation timestamp | `2026-02-13` | ✅ Yes |
| `DataClassification` | Data sensitivity level | `public`, `internal`, `confidential`, `pii` | ✅ Yes |
| `BackupPolicy` | Backup requirement | `daily`, `weekly`, `none` | ✅ Yes |
| `Compliance` | Regulatory requirements | `hipaa`, `gdpr`, `none` | ✅ Yes |

### Service-Specific Tags

| Service Type | Additional Tags | Example |
|--------------|----------------|---------|
| **RDS/Databases** | `DatabaseEngine`, `DatabaseVersion`, `RetentionDays` | `postgres`, `15`, `7` |
| **ECS Tasks** | `TaskDefinition`, `TaskVersion`, `AutoScaling` | `ohi-api`, `v1.2.3`, `enabled` |
| **S3 Buckets** | `BucketPurpose`, `PublicAccess`, `Encryption` | `frontend-hosting`, `cloudfront-only`, `aes256` |
| **ALB/Network** | `ExposedTo`, `Protocol`, `SSLCertificate` | `public-internet`, `https`, `acm-cert-id` |

### Tag Enforcement Mechanisms

#### 1. Pulumi Resource Transformations

**File:** `infrastructure/pulumi/tagging.ts`

```typescript
import * as pulumi from "@pulumi/pulumi";

export const requiredTags = {
  Project: "open-health-initiative",
  Owner: "platform-team",
  CostCenter: "ohi-infrastructure",
  ManagedBy: "pulumi",
  CreatedBy: "pulumi-cli", // Override in CI/CD
  CreatedDate: new Date().toISOString().split('T')[0],
};

export function applyDefaultTags(environment: string): pulumi.ResourceTransformation {
  return (args) => {
    const tags = {
      ...requiredTags,
      Environment: environment,
      Service: args.name, // Use resource name as service tag
    };

    // Merge with existing tags
    if (args.props.tags) {
      args.props.tags = { ...tags, ...args.props.tags };
    } else {
      args.props.tags = tags;
    }

    return { props: args.props, opts: args.opts };
  };
}
```

#### 2. AWS Config Rules (Tag Compliance Monitoring)

Deploy AWS Config rule to detect untagged resources:

```typescript
import * as aws from "@pulumi/aws";

const requiredTagsRule = new aws.cfg.Rule("required-tags", {
  name: "ohi-required-tags",
  source: {
    owner: "AWS",
    sourceIdentifier: "REQUIRED_TAGS",
  },
  inputParameters: JSON.stringify({
    tag1Key: "Project",
    tag2Key: "Environment",
    tag3Key: "Owner",
    tag4Key: "CostCenter",
    tag5Key: "Service",
    tag6Key: "ManagedBy",
  }),
  scope: {
    complianceResourceTypes: [
      "AWS::EC2::Instance",
      "AWS::RDS::DBInstance",
      "AWS::ECS::Service",
      "AWS::ElastiCache::CacheCluster",
      "AWS::S3::Bucket",
      "AWS::ElasticLoadBalancingV2::LoadBalancer",
    ],
  },
});
```

#### 3. Cost Explorer Tag-Based Budgets

Create budgets per environment:

```typescript
import * as aws from "@pulumi/aws";

const prodBudget = new aws.budgets.Budget("ohi-prod-budget", {
  budgetType: "COST",
  limitAmount: "800", // $800/month for prod
  limitUnit: "USD",
  timePeriod: { start: "2026-02-01_00:00" },
  timeUnit: "MONTHLY",
  costFilters: [
    {
      name: "TagKeyValue",
      values: ["user:Project$open-health-initiative", "user:Environment$prod"],
    },
  ],
  notifications: [
    {
      comparisonOperator: "GREATER_THAN",
      threshold: 80, // Alert at 80%
      thresholdType: "PERCENTAGE",
      notificationType: "ACTUAL",
      subscriberEmailAddresses: ["platform-team@example.com"],
    },
  ],
});
```

#### 4. GitHub Actions CI/CD Tag Injection

**Update `.github/workflows/cd-deploy.yml`:**

```yaml
env:
  PULUMI_TAGS: >
    {
      "Project": "open-health-initiative",
      "Environment": "${{ inputs.environment }}",
      "Owner": "platform-team",
      "CostCenter": "ohi-infrastructure",
      "ManagedBy": "pulumi",
      "CreatedBy": "github-actions",
      "CreatedDate": "${{ github.event.head_commit.timestamp }}",
      "GitCommit": "${{ github.sha }}",
      "GitBranch": "${{ github.ref_name }}",
      "Deployment": "${{ github.run_id }}"
    }
```

### Tag Naming Conventions

**Service Tag Values (Microservices):**
- `api` - Main REST API
- `graphql` - GraphQL gateway
- `sse` - Server-Sent Events service
- `provider-api` - Data provider service
- `reindexer` - Search indexing job
- `blnk-api` - Financial ledger API
- `blnk-worker` - Ledger background worker
- `frontend` - React SPA
- `postgres-primary` - Main database
- `postgres-read-replica` - Read replica N
- `redis-cache` - Application cache
- `redis-blnk` - Blnk dedicated cache
- `mongodb-provider` - Provider data store
- `typesense` - Search engine (if self-hosted)
- `clickhouse` - Observability DB
- `signoz-collector` - OTEL collector
- `signoz-query` - Query service
- `signoz-frontend` - Observability UI

**Environment Tag Values:**
- `dev` - Development environment
- `staging` - Pre-production environment
- `prod` - Production environment
- `sandbox` - Experimental/testing

**Data Classification Values:**
- `public` - Publicly accessible data (frontend assets)
- `internal` - Internal business data (facility lists, prices)
- `confidential` - Sensitive business data (API keys, configs)
- `pii` - Personally Identifiable Information (user data, appointments)
- `phi` - Protected Health Information (health records) - Not used in V1

### Cost Allocation Reports

**Monthly Reports by:**
1. **Environment:** `Project=open-health-initiative` grouped by `Environment`
2. **Service:** `Project=open-health-initiative` grouped by `Service`
3. **Cost Center:** `CostCenter=ohi-infrastructure` (vs other projects in account)

**AWS CLI Query Example:**
```bash
aws ce get-cost-and-usage \
  --time-period Start=2026-02-01,End=2026-03-01 \
  --granularity MONTHLY \
  --metrics BlendedCost \
  --group-by Type=TAG,Key=Environment \
  --filter file://filter.json
```

**filter.json:**
```json
{
  "Tags": {
    "Key": "Project",
    "Values": ["open-health-initiative"]
  }
}
```

---

## 5. Service Isolation Strategy (Shared AWS Account)

### VPC Design

**Separate VPCs per Environment:**
- `ohi-dev-vpc` (10.0.0.0/16)
- `ohi-staging-vpc` (10.1.0.0/16)
- `ohi-prod-vpc` (10.2.0.0/16)

**No VPC Peering between Environments** - Complete isolation

**Subnets per VPC:**
- Public subnets (3 AZs): ALB, NAT Gateways
- Private subnets (3 AZs): ECS tasks, EC2
- Database subnets (3 AZs): RDS, ElastiCache
- Isolated subnets (3 AZs): Admin/bastion (no internet)

### IAM Boundaries

**Permission Boundary Policy for all OHI roles:**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "*",
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "aws:RequestedRegion": "eu-west-1",
          "aws:ResourceTag/Project": "open-health-initiative"
        }
      }
    },
    {
      "Effect": "Deny",
      "Action": [
        "iam:DeleteRole",
        "iam:DeleteRolePolicy",
        "iam:DeleteUser",
        "organizations:*"
      ],
      "Resource": "*"
    }
  ]
}
```

**ECS Task Roles** - Principle of Least Privilege:
- API task: Read Secrets Manager (secrets tagged `Service=api`), RDS connect, ElastiCache connect
- Provider task: MongoDB Atlas connection, Read Secrets Manager, S3 read (price list files)
- Reindexer task: RDS read-only, Typesense API (or OpenSearch write)

### Resource Naming Convention

**Format:** `{project}-{environment}-{service}-{resource-type}`

**Examples:**
- RDS: `ohi-prod-postgres-primary`
- ElastiCache: `ohi-prod-redis-cache`
- ECS Service: `ohi-prod-api-service`
- ALB: `ohi-prod-api-alb`
- S3 Bucket: `ohi-prod-frontend-hosting`
- Secrets: `ohi-prod-api-database-password`

**Benefits:**
- Clear ownership at a glance
- Easy to filter in AWS Console
- Prevents naming collisions with other projects
- Enables automated cleanup scripts (delete all `ohi-dev-*`)

---

## 6. Network Architecture & Security Design

### VPC Network Topology (Per Environment)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          AWS VPC (ohi-prod-vpc)                              │
│                              10.2.0.0/16                                     │
│                                                                              │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                        Availability Zone A                             │  │
│  │                                                                        │  │
│  │  ┌─────────────────┐  ┌──────────────────┐  ┌──────────────────┐    │  │
│  │  │   Public-A      │  │   Private-A      │  │   Database-A     │    │  │
│  │  │  10.2.0.0/24    │  │  10.2.10.0/24    │  │  10.2.20.0/24    │    │  │
│  │  │                 │  │                  │  │                  │    │  │
│  │  │  - ALB          │  │  - ECS Tasks     │  │  - RDS Primary   │    │  │
│  │  │  - NAT Gateway  │  │  - EC2 (ClickHs) │  │  - ElastiCache   │    │  │
│  │  └─────────────────┘  └──────────────────┘  └──────────────────┘    │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                        Availability Zone B                             │  │
│  │                                                                        │  │
│  │  ┌─────────────────┐  ┌──────────────────┐  ┌──────────────────┐    │  │
│  │  │   Public-B      │  │   Private-B      │  │   Database-B     │    │  │
│  │  │  10.2.1.0/24    │  │  10.2.11.0/24    │  │  10.2.21.0/24    │    │  │
│  │  │                 │  │                  │  │                  │    │  │
│  │  │  - ALB          │  │  - ECS Tasks     │  │  - RDS Standby   │    │  │
│  │  │  - NAT Gateway  │  │                  │  │  - ElastiCache   │    │  │
│  │  └─────────────────┘  └──────────────────┘  └──────────────────┘    │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                        Availability Zone C                             │  │
│  │                                                                        │  │
│  │  ┌─────────────────┐  ┌──────────────────┐  ┌──────────────────┐    │  │
│  │  │   Public-C      │  │   Private-C      │  │   Database-C     │    │  │
│  │  │  10.2.2.0/24    │  │  10.2.12.0/24    │  │  10.2.22.0/24    │    │  │
│  │  │                 │  │                  │  │                  │    │  │
│  │  │  - ALB          │  │  - ECS Tasks     │  │  - RDS Replica   │    │  │
│  │  │  - NAT Gateway  │  │                  │  │  - ElastiCache   │    │  │
│  │  └─────────────────┘  └──────────────────┘  └──────────────────┘    │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

                                      ▲
                                      │
                                      │
                        ┌─────────────┴─────────────┐
                        │   Internet Gateway        │
                        │   (ohi-prod-igw)         │
                        └─────────────┬─────────────┘
                                      │
                                      │
                        ┌─────────────▼─────────────┐
                        │      CloudFront CDN       │
                        │   (Global Edge Locations) │
                        └─────────────┬─────────────┘
                                      │
                                      │
                        ┌─────────────▼─────────────┐
                        │     Users (Nigeria)       │
                        │   Mobile/Web Browsers     │
                        └───────────────────────────┘
```

### Service Communication Flow (Production)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              INTERNET                                        │
└────────────────────┬────────────────────────────────────────────────────────┘
                     │
                     │ HTTPS (443)
                     ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                         CloudFront CDN                                      │
│  - Static Assets: S3 Origin (React app, images, CSS, JS)                  │
│  - API Requests: ALB Origin (path: /api/*)                                │
└────────────────────┬───────────────────────────────────────────────────────┘
                     │
                     │ HTTPS (443)
                     ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                  Application Load Balancer (ALB)                            │
│  - Public Subnets (10.2.0.0/24, 10.2.1.0/24, 10.2.2.0/24)                 │
│  - Security Group: Allow 443 from CloudFront IPs                           │
│  - Target Groups: API (8080), GraphQL (8081), SSE (8082)                  │
└───┬──────────────────┬──────────────────┬───────────────────────────────────┘
    │                  │                  │
    │ HTTP 8080        │ HTTP 8081        │ HTTP 8082
    ▼                  ▼                  ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   API        │  │  GraphQL     │  │    SSE       │
│   Service    │  │  Service     │  │  Service     │
│              │  │              │  │              │
│  Port: 8080  │  │  Port: 8081  │  │  Port: 8082  │
│  ECS Fargate │  │  ECS Fargate │  │  ECS Fargate │
│  Private Sub │  │  Private Sub │  │  Private Sub │
└──┬────┬──┬──┘  └──┬───┬───┬───┘  └──┬───────────┘
   │    │  │        │   │   │         │
   │    │  │        │   │   │         │
   │    │  │        │   │   │         └────────┐
   │    │  │        │   │   │                  │
   │    │  │        │   │   └──────────┐       │
   │    │  │        │   │              │       │
   │    │  └────────┼───┼────┐         │       │
   │    │           │   │    │         │       │
   │    │           │   │    ▼         ▼       ▼
   │    │           │   │  ┌─────────────────────────┐
   │    │           │   │  │   Redis (ElastiCache)   │
   │    │           │   │  │   Port: 6379            │
   │    │           │   │  │   Database Subnets      │
   │    │           │   │  └─────────────────────────┘
   │    │           │   │
   │    │           │   └──────────┐
   │    │           │              │
   │    │           ▼              ▼
   │    │  ┌────────────────────────────────┐
   │    │  │   Typesense Cloud (External)   │
   │    │  │   Port: 443 (HTTPS)            │
   │    │  │   Managed SaaS                 │
   │    │  └────────────────────────────────┘
   │    │
   │    └────────────────────┐
   │                         │
   ▼                         ▼
┌─────────────────────────────────────┐
│  RDS PostgreSQL Cluster             │
│  - Primary: Port 5432               │
│  - Read Replica 1: Port 5432        │
│  - Read Replica 2: Port 5432        │
│  - Database Subnets                 │
│  - Multi-AZ Auto-Failover           │
└─────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────┐
│                        Provider Data Pipeline                               │
└────────────────────────────────────────────────────────────────────────────┘

┌──────────────────┐      ┌───────────────────────────────────┐
│  Provider API    │──────▶│  MongoDB Atlas (External)         │
│  Service         │      │  Port: 27017 (TLS)                │
│  Port: 3000      │      │  Connection String: mongodb+srv:// │
│  ECS Fargate     │      └───────────────────────────────────┘
│  Private Subnet  │
└─────┬────────────┘
      │
      │ Triggers Reindex
      ▼
┌──────────────────┐      ┌───────────────────────────────────┐
│  Reindexer       │──────▶│  Typesense Cloud                  │
│  Service         │      │  Port: 443 (HTTPS)                │
│  (Background)    │◀─────│                                   │
│  ECS Fargate     │ Reads│                                   │
│  Private Subnet  │ RDS  └───────────────────────────────────┘
└──────────────────┘

┌────────────────────────────────────────────────────────────────────────────┐
│                        Financial Ledger Services                            │
└────────────────────────────────────────────────────────────────────────────┘

┌──────────────────┐      ┌──────────────────┐
│  Blnk API        │──────▶│  Blnk PostgreSQL │
│  Service         │      │  Port: 5432      │
│  Port: 5001      │      │  RDS (Separate)  │
│  ECS Fargate     │◀─────│  Database Subnet │
│  Private Subnet  │      └──────────────────┘
└──────┬───────────┘
       │                   ┌──────────────────┐
       └───────────────────▶│  Blnk Redis      │
                           │  Port: 6379      │
┌──────────────────┐       │  ElastiCache     │
│  Blnk Worker     │───────▶│  (Separate)      │
│  (Background)    │       └──────────────────┘
│  ECS Fargate     │
│  Private Subnet  │
└──────────────────┘

┌────────────────────────────────────────────────────────────────────────────┐
│                        Observability Stack                                  │
└────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│  All ECS Tasks send telemetry to:                                        │
│                                                                           │
│  ┌────────────────────┐         ┌──────────────────────┐                │
│  │ SigNoz OTEL        │────────▶│  ClickHouse          │                │
│  │ Collector          │         │  Port: 9000, 8123    │                │
│  │ Port: 4317 (gRPC)  │◀────────│  EC2 i3en.large      │                │
│  │ Port: 4318 (HTTP)  │  Writes │  Private Subnet      │                │
│  │ ECS Fargate        │         │  EBS gp3 500GB       │                │
│  └────────────────────┘         └──────────────────────┘                │
│           ▲                               │                              │
│           │ Telemetry                     │ Queries                      │
│           │ (OTLP)                        ▼                              │
│  ┌────────┴────────────┐         ┌──────────────────────┐               │
│  │  All Services:      │         │ SigNoz Query Service │               │
│  │  - API              │         │ Port: 8080           │               │
│  │  - GraphQL          │         │ ECS Fargate          │               │
│  │  - SSE              │         └─────────┬────────────┘               │
│  │  - Provider         │                   │                            │
│  │  - Reindexer        │                   │ HTTP                       │
│  │  - Blnk             │                   ▼                            │
│  └─────────────────────┘         ┌──────────────────────┐               │
│                                   │ SigNoz Frontend      │               │
│                                   │ Port: 3301           │               │
│                                   │ ECS Fargate          │               │
│                                   │ (Internal ALB)       │               │
│                                   └──────────────────────┘               │
└──────────────────────────────────────────────────────────────────────────┘
```

### Security Group Configuration

#### 1. **ALB Security Group** (`ohi-prod-alb-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | HTTPS | 443 | CloudFront Managed Prefix List | Allow CDN traffic |
| Inbound | HTTP | 80 | CloudFront Managed Prefix List | Redirect to HTTPS |
| Outbound | HTTP | 8080 | API Service SG | Forward to API |
| Outbound | HTTP | 8081 | GraphQL Service SG | Forward to GraphQL |
| Outbound | HTTP | 8082 | SSE Service SG | Forward to SSE |

#### 2. **API Service Security Group** (`ohi-prod-api-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | HTTP | 8080 | ALB SG | ALB health checks + traffic |
| Outbound | PostgreSQL | 5432 | RDS SG | Database queries |
| Outbound | Redis | 6379 | ElastiCache SG | Cache operations |
| Outbound | HTTPS | 443 | 0.0.0.0/0 | External APIs (Typesense, Secrets Manager) |
| Outbound | HTTP | 3000 | Provider API SG | Call provider service |
| Outbound | HTTP | 4318 | OTEL Collector SG | Send telemetry |

#### 3. **GraphQL Service Security Group** (`ohi-prod-graphql-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | HTTP | 8081 | ALB SG | ALB health checks + traffic |
| Outbound | PostgreSQL | 5432 | RDS SG | Database queries |
| Outbound | Redis | 6379 | ElastiCache SG | Cache operations |
| Outbound | HTTPS | 443 | 0.0.0.0/0 | Typesense Cloud API |
| Outbound | HTTP | 3000 | Provider API SG | Call provider service |
| Outbound | HTTP | 4318 | OTEL Collector SG | Send telemetry |

#### 4. **SSE Service Security Group** (`ohi-prod-sse-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | HTTP | 8082 | ALB SG | ALB health checks + traffic |
| Outbound | Redis | 6379 | ElastiCache SG | Pub/Sub for real-time events |
| Outbound | HTTP | 4318 | OTEL Collector SG | Send telemetry |

#### 5. **Provider API Security Group** (`ohi-prod-provider-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | HTTP | 3000 | API SG, GraphQL SG | Internal service calls |
| Outbound | MongoDB | 27017 | MongoDB Atlas IPs | Provider data queries |
| Outbound | HTTPS | 443 | 0.0.0.0/0 | MongoDB Atlas (TLS), Secrets Manager |
| Outbound | HTTP | 4318 | OTEL Collector SG | Send telemetry |

#### 6. **Reindexer Service Security Group** (`ohi-prod-reindexer-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | None | - | - | Background job, no inbound |
| Outbound | PostgreSQL | 5432 | RDS SG | Read database changes |
| Outbound | HTTPS | 443 | 0.0.0.0/0 | Typesense Cloud indexing |
| Outbound | HTTP | 4318 | OTEL Collector SG | Send telemetry |

#### 7. **Blnk API Security Group** (`ohi-prod-blnk-api-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | HTTP | 5001 | API SG | Payment/ledger operations |
| Outbound | PostgreSQL | 5432 | Blnk RDS SG | Ledger database |
| Outbound | Redis | 6379 | Blnk ElastiCache SG | Ledger cache |
| Outbound | HTTP | 4318 | OTEL Collector SG | Send telemetry |

#### 8. **Blnk Worker Security Group** (`ohi-prod-blnk-worker-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | None | - | - | Background job, no inbound |
| Outbound | PostgreSQL | 5432 | Blnk RDS SG | Ledger database |
| Outbound | Redis | 6379 | Blnk ElastiCache SG | Ledger cache |
| Outbound | HTTP | 4318 | OTEL Collector SG | Send telemetry |

#### 9. **RDS PostgreSQL Security Group** (`ohi-prod-rds-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | PostgreSQL | 5432 | API SG | Application queries |
| Inbound | PostgreSQL | 5432 | GraphQL SG | GraphQL queries |
| Inbound | PostgreSQL | 5432 | Reindexer SG | Read for indexing |
| Inbound | PostgreSQL | 5432 | Bastion SG (optional) | Admin access |
| Outbound | All | All | 0.0.0.0/0 | Allow outbound (replication) |

#### 10. **Blnk RDS PostgreSQL Security Group** (`ohi-prod-blnk-rds-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | PostgreSQL | 5432 | Blnk API SG | Ledger operations |
| Inbound | PostgreSQL | 5432 | Blnk Worker SG | Background jobs |
| Inbound | PostgreSQL | 5432 | Bastion SG (optional) | Admin access |
| Outbound | All | All | 0.0.0.0/0 | Allow outbound (replication) |

#### 11. **ElastiCache Redis Security Group** (`ohi-prod-redis-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | Redis | 6379 | API SG | Cache operations |
| Inbound | Redis | 6379 | GraphQL SG | Cache operations |
| Inbound | Redis | 6379 | SSE SG | Pub/Sub |
| Outbound | None | - | - | No outbound needed |

#### 12. **Blnk ElastiCache Redis Security Group** (`ohi-prod-blnk-redis-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | Redis | 6379 | Blnk API SG | Ledger cache |
| Inbound | Redis | 6379 | Blnk Worker SG | Worker cache |
| Outbound | None | - | - | No outbound needed |

#### 13. **ClickHouse EC2 Security Group** (`ohi-prod-clickhouse-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | ClickHouse | 9000 | OTEL Collector SG | Native protocol writes |
| Inbound | HTTP | 8123 | SigNoz Query SG | HTTP API queries |
| Inbound | SSH | 22 | Bastion SG (optional) | Admin access |
| Outbound | All | All | 0.0.0.0/0 | Package updates |

#### 14. **SigNoz OTEL Collector Security Group** (`ohi-prod-otel-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | gRPC | 4317 | All App Service SGs | Receive OTLP traces |
| Inbound | HTTP | 4318 | All App Service SGs | Receive OTLP HTTP |
| Outbound | ClickHouse | 9000 | ClickHouse SG | Write telemetry |
| Outbound | HTTP | 8123 | ClickHouse SG | Write telemetry (HTTP) |

#### 15. **SigNoz Query Service Security Group** (`ohi-prod-signoz-query-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | HTTP | 8080 | SigNoz Frontend SG | Dashboard queries |
| Inbound | HTTP | 8080 | Internal ALB SG | Health checks |
| Outbound | HTTP | 8123 | ClickHouse SG | Query telemetry data |

#### 16. **SigNoz Frontend Security Group** (`ohi-prod-signoz-frontend-sg`)

| Type | Protocol | Port | Source | Purpose |
|------|----------|------|--------|---------|
| Inbound | HTTP | 3301 | Internal ALB SG | Dashboard access |
| Outbound | HTTP | 8080 | SigNoz Query SG | API calls |

### Service Communication Matrix

| Source Service | Destination Service | Port | Protocol | Purpose | Encrypted |
|----------------|---------------------|------|----------|---------|-----------|
| **Internet** | CloudFront | 443 | HTTPS | User requests | ✅ TLS 1.3 |
| CloudFront | ALB | 443 | HTTPS | API/Asset requests | ✅ TLS 1.2+ |
| CloudFront | S3 | 443 | HTTPS | Static assets | ✅ TLS 1.2+ |
| ALB | API | 8080 | HTTP | REST API calls | ❌ (VPC internal) |
| ALB | GraphQL | 8081 | HTTP | GraphQL queries | ❌ (VPC internal) |
| ALB | SSE | 8082 | HTTP | Event stream | ❌ (VPC internal) |
| API | RDS Primary | 5432 | PostgreSQL | Read/Write queries | ✅ TLS |
| API | RDS Read Replica | 5432 | PostgreSQL | Read-only queries | ✅ TLS |
| API | ElastiCache | 6379 | Redis | Cache ops | ✅ TLS (in-transit) |
| API | Provider API | 3000 | HTTP | Provider data | ❌ (VPC internal) |
| API | Secrets Manager | 443 | HTTPS | Fetch secrets | ✅ TLS 1.2+ |
| API | OTEL Collector | 4318 | HTTP | Telemetry | ❌ (VPC internal) |
| GraphQL | RDS Primary | 5432 | PostgreSQL | Read/Write queries | ✅ TLS |
| GraphQL | RDS Read Replica | 5432 | PostgreSQL | Read-only queries | ✅ TLS |
| GraphQL | ElastiCache | 6379 | Redis | Cache ops | ✅ TLS (in-transit) |
| GraphQL | Typesense Cloud | 443 | HTTPS | Search queries | ✅ TLS 1.2+ |
| GraphQL | Provider API | 3000 | HTTP | Provider data | ❌ (VPC internal) |
| GraphQL | OTEL Collector | 4318 | HTTP | Telemetry | ❌ (VPC internal) |
| SSE | ElastiCache | 6379 | Redis | Pub/Sub events | ✅ TLS (in-transit) |
| SSE | OTEL Collector | 4318 | HTTP | Telemetry | ❌ (VPC internal) |
| Provider API | MongoDB Atlas | 27017 | MongoDB+TLS | Provider data | ✅ TLS 1.2+ |
| Provider API | Secrets Manager | 443 | HTTPS | Fetch secrets | ✅ TLS 1.2+ |
| Provider API | OTEL Collector | 4318 | HTTP | Telemetry | ❌ (VPC internal) |
| Reindexer | RDS Read Replica | 5432 | PostgreSQL | Read changes | ✅ TLS |
| Reindexer | Typesense Cloud | 443 | HTTPS | Index updates | ✅ TLS 1.2+ |
| Reindexer | OTEL Collector | 4318 | HTTP | Telemetry | ❌ (VPC internal) |
| Blnk API | Blnk RDS | 5432 | PostgreSQL | Ledger queries | ✅ TLS |
| Blnk API | Blnk Redis | 6379 | Redis | Ledger cache | ✅ TLS (in-transit) |
| Blnk API | OTEL Collector | 4318 | HTTP | Telemetry | ❌ (VPC internal) |
| Blnk Worker | Blnk RDS | 5432 | PostgreSQL | Background jobs | ✅ TLS |
| Blnk Worker | Blnk Redis | 6379 | Redis | Job cache | ✅ TLS (in-transit) |
| Blnk Worker | OTEL Collector | 4318 | HTTP | Telemetry | ❌ (VPC internal) |
| OTEL Collector | ClickHouse | 9000 | Native | Write telemetry | ❌ (VPC internal) |
| SigNoz Query | ClickHouse | 8123 | HTTP | Query telemetry | ❌ (VPC internal) |
| SigNoz Frontend | SigNoz Query | 8080 | HTTP | Dashboard API | ❌ (VPC internal) |

### External Service IP Allowlisting

#### MongoDB Atlas
**Inbound:** Allow ECS task IPs (NAT Gateway Elastic IPs)
**MongoDB Atlas IP Access List:**
```
# Production NAT Gateway IPs (3 AZs)
NAT-A-EIP: 52.xx.xx.1/32
NAT-B-EIP: 52.xx.xx.2/32
NAT-C-EIP: 52.xx.xx.3/32
```

#### Typesense Cloud
**Inbound:** Allow ECS task IPs (NAT Gateway Elastic IPs)
**Typesense Cloud Firewall Rules:**
```
# Production NAT Gateway IPs
NAT-A-EIP: 52.xx.xx.1/32
NAT-B-EIP: 52.xx.xx.2/32
NAT-C-EIP: 52.xx.xx.3/32
```

### Network Security Best Practices

#### 1. **Principle of Least Privilege**
- ✅ No direct internet access to ECS tasks (use NAT Gateways)
- ✅ Databases only in private subnets, no public IPs
- ✅ Security groups deny all by default, explicit allow rules only
- ✅ Separate security groups per service (no shared SGs)
- ✅ Separate financial services (Blnk) with isolated RDS/Redis

#### 2. **Data in Transit Encryption**
- ✅ CloudFront to user: TLS 1.3
- ✅ CloudFront to ALB: TLS 1.2+
- ✅ ALB SSL/TLS termination (ACM certificates)
- ✅ RDS: Enable SSL/TLS connections (require `sslmode=require`)
- ✅ ElastiCache: Enable in-transit encryption (TLS)
- ✅ MongoDB Atlas: TLS 1.2+ mandatory
- ✅ Typesense Cloud: HTTPS only
- ❌ Internal VPC traffic (ECS to ECS): Unencrypted (trusted network)

#### 3. **Data at Rest Encryption**
- ✅ RDS: Enable encryption at rest (AWS KMS)
- ✅ ElastiCache: Enable encryption at rest (AWS KMS)
- ✅ S3: Enable default encryption (SSE-S3 or SSE-KMS)
- ✅ EBS volumes (ClickHouse): Enable encryption (AWS KMS)
- ✅ MongoDB Atlas: Encryption at rest enabled
- ✅ Secrets Manager: Encrypted with AWS KMS

#### 4. **Network Segmentation**
- ✅ Public subnets: Only ALB, NAT Gateways, Internet Gateway
- ✅ Private subnets: All ECS tasks, EC2 instances
- ✅ Database subnets: RDS, ElastiCache (no route to internet)
- ✅ No VPC peering between environments (dev/staging/prod isolation)
- ✅ Separate VPCs per environment (10.0.0.0/16, 10.1.0.0/16, 10.2.0.0/16)

#### 5. **Secrets Management**
- ✅ No hardcoded secrets in code or docker images
- ✅ AWS Secrets Manager for all credentials
- ✅ ECS task IAM roles with least privilege access to secrets
- ✅ Secrets tagged per service: `Service=api`, `Service=provider-api`
- ✅ Automatic secret rotation enabled (RDS passwords every 30 days)
- ✅ Audit logging via CloudTrail (who accessed what secret)

#### 6. **Observability & Monitoring**
- ✅ VPC Flow Logs enabled (capture all network traffic metadata)
- ✅ CloudWatch Logs for all ECS task logs
- ✅ ALB access logs to S3 (analyze traffic patterns)
- ✅ SigNoz for application telemetry (traces, metrics, logs)
- ✅ CloudWatch Alarms for anomalies (spike in errors, latency)
- ✅ GuardDuty for threat detection (malicious IPs, port scanning)

#### 7. **DDoS Protection**
- ✅ CloudFront as first layer (built-in DDoS protection)
- ✅ AWS Shield Standard (automatic, no cost)
- ⚠️ AWS Shield Advanced (optional, $3000/month) - Consider for prod after V1 validation
- ✅ ALB with connection draining and rate limiting
- ✅ WAF rules on ALB (rate limiting, geo-blocking, SQL injection protection)

#### 8. **Bastion Host Strategy**
**Option A: No Bastion (Recommended for V1)**
- Use AWS Systems Manager Session Manager for emergency RDS/EC2 access
- No SSH keys to manage
- All access logged to CloudTrail

**Option B: Bastion Host in Isolated Subnet (If needed)**
```
┌──────────────────────────────────────┐
│  Isolated Subnet (10.2.30.0/24)     │
│                                      │
│  ┌────────────────────────────────┐  │
│  │  Bastion Host (EC2 t4g.nano)   │  │
│  │  - No public IP                │  │
│  │  - SSM Session Manager only    │  │
│  │  - Security Group: Allow 5432  │  │
│  │    to RDS SG only              │  │
│  └────────────────────────────────┘  │
└──────────────────────────────────────┘
```

### Port Summary Reference

| Service | Port(s) | Protocol | Exposure | Notes |
|---------|---------|----------|----------|-------|
| **CloudFront** | 443 | HTTPS | Public (Global) | TLS 1.3, Edge caching |
| **ALB** | 443, 80 | HTTPS/HTTP | Public (VPC) | 80 redirects to 443 |
| **API** | 8080 | HTTP | Private | Health: /health |
| **GraphQL** | 8081 | HTTP | Private | Health: /health |
| **SSE** | 8082 | HTTP | Private | Health: /health |
| **Provider API** | 3000 | HTTP | Private | Internal only |
| **Blnk API** | 5001 | HTTP | Private | Internal only |
| **RDS PostgreSQL** | 5432 | PostgreSQL | Database subnet | TLS required |
| **Blnk RDS** | 5432 | PostgreSQL | Database subnet | TLS required |
| **ElastiCache Redis** | 6379 | Redis | Database subnet | TLS in-transit |
| **Blnk ElastiCache** | 6379 | Redis | Database subnet | TLS in-transit |
| **MongoDB Atlas** | 27017 | MongoDB+TLS | External | NAT IP allowlist |
| **Typesense Cloud** | 443 | HTTPS | External | NAT IP allowlist |
| **ClickHouse** | 9000, 8123 | Native, HTTP | Private | No public access |
| **OTEL Collector** | 4317, 4318 | gRPC, HTTP | Private | Internal telemetry |
| **SigNoz Query** | 8080 | HTTP | Private | Internal dashboard |
| **SigNoz Frontend** | 3301 | HTTP | Private (Internal ALB) | Admin access only |

### WAF Rules Configuration

**Attach to ALB for production:**

```typescript
const wafRules = [
  {
    name: "RateLimitRule",
    priority: 1,
    action: "block",
    statement: {
      rateBasedStatement: {
        limit: 2000, // 2000 requests per 5 minutes per IP
        aggregateKeyType: "IP",
      },
    },
  },
  {
    name: "GeoBlockRule",
    priority: 2,
    action: "block",
    statement: {
      notStatement: {
        statement: {
          geoMatchStatement: {
            countryCodes: ["NG", "GH", "KE", "ZA", "US", "GB", "IE"], // Allow Nigeria + dev regions
          },
        },
      },
    },
  },
  {
    name: "SQLInjectionRule",
    priority: 3,
    action: "block",
    statement: {
      sqliMatchStatement: {
        fieldToMatch: { queryString: {} },
        textTransformations: [{ priority: 0, type: "URL_DECODE" }],
      },
    },
  },
  {
    name: "XSSRule",
    priority: 4,
    action: "block",
    statement: {
      xssMatchStatement: {
        fieldToMatch: { body: {} },
        textTransformations: [{ priority: 0, type: "HTML_ENTITY_DECODE" }],
      },
    },
  },
];
```

---

## 7. DNS Configuration & Domain Setup

### Current Domain Setup

**Primary Domain:** `ohealth-ng.com` (registered and managed in Squarespace)

**Migration Strategy:** Keep domain registration in Squarespace, delegate DNS management to AWS Route 53 for better integration with AWS services (ALB, CloudFront, ACM certificates).

### Environment-Specific Hosted Zones

Each environment gets its own Route 53 hosted zone for complete isolation:

| Environment | Hosted Zone | Frontend URL | API URL | GraphQL URL | SigNoz URL (Internal) |
|-------------|-------------|--------------|---------|-------------|----------------------|
| **Production** | `ohealth-ng.com` | `ohealth-ng.com` | `api.ohealth-ng.com` | `graphql.ohealth-ng.com` | `signoz.ohealth-ng.com` |
| **Staging** | `staging.ohealth-ng.com` | `staging.ohealth-ng.com` | `api.staging.ohealth-ng.com` | `graphql.staging.ohealth-ng.com` | `signoz.staging.ohealth-ng.com` |
| **Dev** | `dev.ohealth-ng.com` | `dev.ohealth-ng.com` | `api.dev.ohealth-ng.com` | `graphql.dev.ohealth-ng.com` | `signoz.dev.ohealth-ng.com` |

### DNS Architecture Pattern

**Naming Convention:** `[service].[environment].ohealth-ng.com`

**Exception for Production:** Production uses root domain for frontend (`ohealth-ng.com`) and subdomain pattern for services (`api.ohealth-ng.com`)

```
ohealth-ng.com (Squarespace Registration)
│
├── ohealth-ng.com (Route 53 Hosted Zone - Production)
│   ├── @ (A/AAAA) → CloudFront (Frontend)
│   ├── www (CNAME) → CloudFront (Redirect to apex)
│   ├── api (A) → ALB eu-west-1 (Alias)
│   ├── graphql (A) → ALB eu-west-1 (Alias)
│   ├── signoz (A) → Internal ALB (VPN/Bastion only)
│   └── _acme-challenge (TXT) → ACM validation records
│
├── staging.ohealth-ng.com (Route 53 Hosted Zone - Staging)
│   ├── @ (A/AAAA) → CloudFront (Frontend)
│   ├── api (A) → ALB eu-west-1 (Alias)
│   ├── graphql (A) → ALB eu-west-1 (Alias)
│   ├── signoz (A) → Internal ALB
│   └── _acme-challenge (TXT) → ACM validation records
│
└── dev.ohealth-ng.com (Route 53 Hosted Zone - Dev)
    ├── @ (A/AAAA) → CloudFront (Frontend)
    ├── api (A) → ALB eu-west-1 (Alias)
    ├── graphql (A) → ALB eu-west-1 (Alias)
    ├── signoz (A) → Internal ALB
    └── _acme-challenge (TXT) → ACM validation records
```

### Route 53 Hosted Zone Configuration

#### 1. **Production Hosted Zone** (`ohealth-ng.com`)

**Create in Pulumi:**
```typescript
import * as aws from "@pulumi/aws";

const prodZone = new aws.route53.Zone("ohi-prod-zone", {
  name: "ohealth-ng.com",
  comment: "Open Health Initiative - Production",
  tags: {
    Project: "open-health-initiative",
    Environment: "prod",
    ManagedBy: "pulumi",
  },
});

// Frontend - CloudFront distribution
const frontendRecord = new aws.route53.Record("ohi-prod-frontend", {
  zoneId: prodZone.zoneId,
  name: "ohealth-ng.com",
  type: "A",
  aliases: [{
    name: cloudFrontDistribution.domainName,
    zoneId: cloudFrontDistribution.hostedZoneId,
    evaluateTargetHealth: false,
  }],
});

// WWW redirect to apex
const wwwRecord = new aws.route53.Record("ohi-prod-www", {
  zoneId: prodZone.zoneId,
  name: "www.ohealth-ng.com",
  type: "CNAME",
  ttl: 300,
  records: ["ohealth-ng.com"],
});

// API - ALB
const apiRecord = new aws.route53.Record("ohi-prod-api", {
  zoneId: prodZone.zoneId,
  name: "api.ohealth-ng.com",
  type: "A",
  aliases: [{
    name: alb.dnsName,
    zoneId: alb.zoneId,
    evaluateTargetHealth: true,
  }],
});

// GraphQL - ALB (same ALB, different target group via host-based routing)
const graphqlRecord = new aws.route53.Record("ohi-prod-graphql", {
  zoneId: prodZone.zoneId,
  name: "graphql.ohealth-ng.com",
  type: "A",
  aliases: [{
    name: alb.dnsName,
    zoneId: alb.zoneId,
    evaluateTargetHealth: true,
  }],
});

// SigNoz - Internal ALB (not publicly accessible)
const signozRecord = new aws.route53.Record("ohi-prod-signoz", {
  zoneId: prodZone.zoneId,
  name: "signoz.ohealth-ng.com",
  type: "A",
  aliases: [{
    name: internalAlb.dnsName,
    zoneId: internalAlb.zoneId,
    evaluateTargetHealth: true,
  }],
});
```

**Export Name Servers:**
```typescript
export const prodNameServers = prodZone.nameServers;
// Output: ["ns-123.awsdns-12.com", "ns-456.awsdns-45.net", ...]
```

#### 2. **Staging Hosted Zone** (`staging.ohealth-ng.com`)

**Create in Pulumi:**
```typescript
const stagingZone = new aws.route53.Zone("ohi-staging-zone", {
  name: "staging.ohealth-ng.com",
  comment: "Open Health Initiative - Staging",
  tags: {
    Project: "open-health-initiative",
    Environment: "staging",
    ManagedBy: "pulumi",
  },
});

// Frontend
const stagingFrontendRecord = new aws.route53.Record("ohi-staging-frontend", {
  zoneId: stagingZone.zoneId,
  name: "staging.ohealth-ng.com",
  type: "A",
  aliases: [{
    name: stagingCloudFront.domainName,
    zoneId: stagingCloudFront.hostedZoneId,
    evaluateTargetHealth: false,
  }],
});

// API
const stagingApiRecord = new aws.route53.Record("ohi-staging-api", {
  zoneId: stagingZone.zoneId,
  name: "api.staging.ohealth-ng.com",
  type: "A",
  aliases: [{
    name: stagingAlb.dnsName,
    zoneId: stagingAlb.zoneId,
    evaluateTargetHealth: true,
  }],
});

// GraphQL
const stagingGraphqlRecord = new aws.route53.Record("ohi-staging-graphql", {
  zoneId: stagingZone.zoneId,
  name: "graphql.staging.ohealth-ng.com",
  type: "A",
  aliases: [{
    name: stagingAlb.dnsName,
    zoneId: stagingAlb.zoneId,
    evaluateTargetHealth: true,
  }],
});

// SigNoz
const stagingSigNozRecord = new aws.route53.Record("ohi-staging-signoz", {
  zoneId: stagingZone.zoneId,
  name: "signoz.staging.ohealth-ng.com",
  type: "A",
  aliases: [{
    name: stagingInternalAlb.dnsName,
    zoneId: stagingInternalAlb.zoneId,
    evaluateTargetHealth: true,
  }],
});

export const stagingNameServers = stagingZone.nameServers;
```

#### 3. **Dev Hosted Zone** (`dev.ohealth-ng.com`)

**Create in Pulumi:** (Same pattern as staging)

```typescript
const devZone = new aws.route53.Zone("ohi-dev-zone", {
  name: "dev.ohealth-ng.com",
  comment: "Open Health Initiative - Development",
  tags: {
    Project: "open-health-initiative",
    Environment: "dev",
    ManagedBy: "pulumi",
  },
});

// ... (same record pattern as staging)

export const devNameServers = devZone.nameServers;
```

### Delegation from Squarespace to Route 53

#### Step 1: Deploy Pulumi Stack to Create Hosted Zones

```bash
cd infrastructure/pulumi
pulumi up --stack prod
# Note the name servers output
```

**Output Example:**
```
Outputs:
  prodNameServers: [
    "ns-1234.awsdns-12.com",
    "ns-567.awsdns-45.net",
    "ns-890.awsdns-78.org",
    "ns-111.awsdns-11.co.uk"
  ]
  stagingNameServers: [...]
  devNameServers: [...]
```

#### Step 2: Delegate Production Domain in Squarespace

**Navigate to:** Squarespace Domains → ohealth-ng.com → DNS Settings

**Update Name Servers:**
1. Switch from "Squarespace Name Servers" to "Custom Name Servers"
2. Enter the 4 Route 53 name servers from Pulumi output:
   - `ns-1234.awsdns-12.com`
   - `ns-567.awsdns-45.net`
   - `ns-890.awsdns-78.org`
   - `ns-111.awsdns-11.co.uk`
3. Save changes (propagation takes 24-48 hours, but often faster)

**Important:** This delegates ALL DNS for `ohealth-ng.com` to Route 53. Ensure Route 53 has records for any existing services (email, etc.)

#### Step 3: Create NS Records for Staging/Dev in Production Zone

**In Production Route 53 Hosted Zone (`ohealth-ng.com`):**

```typescript
// Delegate staging subdomain to staging hosted zone
const stagingNsRecord = new aws.route53.Record("ohi-staging-ns-delegation", {
  zoneId: prodZone.zoneId,
  name: "staging.ohealth-ng.com",
  type: "NS",
  ttl: 172800, // 48 hours
  records: stagingZone.nameServers,
});

// Delegate dev subdomain to dev hosted zone
const devNsRecord = new aws.route53.Record("ohi-dev-ns-delegation", {
  zoneId: prodZone.zoneId,
  name: "dev.ohealth-ng.com",
  type: "NS",
  ttl: 172800,
  records: devZone.nameServers,
});
```

**Result:** Queries for `api.staging.ohealth-ng.com` are handled by the staging hosted zone, completely isolated from production.

### SSL/TLS Certificate Management with ACM

#### Certificate Strategy

**One wildcard certificate per environment** to cover all subdomains:

| Environment | Certificate | Domains Covered |
|-------------|-------------|-----------------|
| Production | `*.ohealth-ng.com` + `ohealth-ng.com` | `ohealth-ng.com`, `api.ohealth-ng.com`, `graphql.ohealth-ng.com`, `www.ohealth-ng.com` |
| Staging | `*.staging.ohealth-ng.com` + `staging.ohealth-ng.com` | `staging.ohealth-ng.com`, `api.staging.ohealth-ng.com`, `graphql.staging.ohealth-ng.com` |
| Dev | `*.dev.ohealth-ng.com` + `dev.ohealth-ng.com` | `dev.ohealth-ng.com`, `api.dev.ohealth-ng.com`, `graphql.dev.ohealth-ng.com` |

#### ACM Certificate for Production

```typescript
import * as aws from "@pulumi/aws";

// Request certificate in us-east-1 (required for CloudFront)
const prodCertUsEast1 = new aws.acm.Certificate("ohi-prod-cert-cloudfront", {
  domainName: "ohealth-ng.com",
  subjectAlternativeNames: [
    "*.ohealth-ng.com",
    "www.ohealth-ng.com",
  ],
  validationMethod: "DNS",
  tags: {
    Project: "open-health-initiative",
    Environment: "prod",
    ManagedBy: "pulumi",
    Purpose: "cloudfront",
  },
}, { provider: usEast1Provider }); // CloudFront requires us-east-1

// Request certificate in eu-west-1 (for ALB)
const prodCertEuWest1 = new aws.acm.Certificate("ohi-prod-cert-alb", {
  domainName: "ohealth-ng.com",
  subjectAlternativeNames: [
    "*.ohealth-ng.com",
  ],
  validationMethod: "DNS",
  tags: {
    Project: "open-health-initiative",
    Environment: "prod",
    ManagedBy: "pulumi",
    Purpose: "alb",
  },
});

// Automatic DNS validation records
prodCertUsEast1.domainValidationOptions.apply(options => {
  options.forEach((option, index) => {
    new aws.route53.Record(`ohi-prod-cert-validation-${index}`, {
      zoneId: prodZone.zoneId,
      name: option.resourceRecordName,
      type: option.resourceRecordType,
      records: [option.resourceRecordValue],
      ttl: 60,
    });
  });
});

// Wait for certificate validation
const prodCertValidation = new aws.acm.CertificateValidation("ohi-prod-cert-validation", {
  certificateArn: prodCertEuWest1.arn,
  validationRecordFqdns: prodCertEuWest1.domainValidationOptions.apply(
    options => options.map(o => o.resourceRecordName)
  ),
});
```

**Repeat for Staging and Dev environments.**

### ALB Configuration with Host-Based Routing

**Single ALB per environment** with multiple target groups, routed by hostname:

```typescript
const alb = new aws.lb.LoadBalancer("ohi-prod-alb", {
  internal: false,
  loadBalancerType: "application",
  securityGroups: [albSecurityGroup.id],
  subnets: publicSubnetIds,
  enableHttp2: true,
  enableDeletionProtection: true, // Prevent accidental deletion in prod
  tags: {
    Project: "open-health-initiative",
    Environment: "prod",
    Service: "alb",
    ManagedBy: "pulumi",
  },
});

// HTTPS Listener (443)
const httpsListener = new aws.lb.Listener("ohi-prod-https-listener", {
  loadBalancerArn: alb.arn,
  port: 443,
  protocol: "HTTPS",
  sslPolicy: "ELBSecurityPolicy-TLS-1-2-2017-01",
  certificateArns: [prodCertEuWest1.arn],
  defaultActions: [{
    type: "fixed-response",
    fixedResponse: {
      contentType: "text/plain",
      statusCode: "404",
      messageBody: "Not Found",
    },
  }],
});

// HTTP Listener (80) - Redirect to HTTPS
const httpListener = new aws.lb.Listener("ohi-prod-http-listener", {
  loadBalancerArn: alb.arn,
  port: 80,
  protocol: "HTTP",
  defaultActions: [{
    type: "redirect",
    redirect: {
      port: "443",
      protocol: "HTTPS",
      statusCode: "HTTP_301",
    },
  }],
});

// Target Group: API (8080)
const apiTargetGroup = new aws.lb.TargetGroup("ohi-prod-api-tg", {
  port: 8080,
  protocol: "HTTP",
  vpcId: vpcId,
  targetType: "ip", // For Fargate
  healthCheck: {
    enabled: true,
    path: "/health",
    interval: 30,
    timeout: 5,
    healthyThreshold: 2,
    unhealthyThreshold: 3,
    matcher: "200",
  },
  deregistrationDelay: 30,
  tags: {
    Project: "open-health-initiative",
    Environment: "prod",
    Service: "api",
    ManagedBy: "pulumi",
  },
});

// Target Group: GraphQL (8081)
const graphqlTargetGroup = new aws.lb.TargetGroup("ohi-prod-graphql-tg", {
  port: 8081,
  protocol: "HTTP",
  vpcId: vpcId,
  targetType: "ip",
  healthCheck: {
    enabled: true,
    path: "/health",
    interval: 30,
    timeout: 5,
    healthyThreshold: 2,
    unhealthyThreshold: 3,
    matcher: "200",
  },
  deregistrationDelay: 30,
  tags: {
    Project: "open-health-initiative",
    Environment: "prod",
    Service: "graphql",
    ManagedBy: "pulumi",
  },
});

// Listener Rule: api.ohealth-ng.com → API Target Group
const apiRule = new aws.lb.ListenerRule("ohi-prod-api-rule", {
  listenerArn: httpsListener.arn,
  priority: 100,
  actions: [{
    type: "forward",
    targetGroupArn: apiTargetGroup.arn,
  }],
  conditions: [
    {
      hostHeader: {
        values: ["api.ohealth-ng.com"],
      },
    },
  ],
});

// Listener Rule: graphql.ohealth-ng.com → GraphQL Target Group
const graphqlRule = new aws.lb.ListenerRule("ohi-prod-graphql-rule", {
  listenerArn: httpsListener.arn,
  priority: 200,
  actions: [{
    type: "forward",
    targetGroupArn: graphqlTargetGroup.arn,
  }],
  conditions: [
    {
      hostHeader: {
        values: ["graphql.ohealth-ng.com"],
      },
    },
  ],
});
```

### CloudFront Distribution Configuration

```typescript
const frontendBucket = new aws.s3.Bucket("ohi-prod-frontend", {
  bucket: "ohi-prod-frontend-hosting",
  acl: "private", // CloudFront OAI handles access
  website: {
    indexDocument: "index.html",
    errorDocument: "index.html", // SPA fallback
  },
  tags: {
    Project: "open-health-initiative",
    Environment: "prod",
    Service: "frontend",
    ManagedBy: "pulumi",
  },
});

const oai = new aws.cloudfront.OriginAccessIdentity("ohi-prod-oai", {
  comment: "OAI for Open Health Initiative frontend",
});

const distribution = new aws.cloudfront.Distribution("ohi-prod-cloudfront", {
  enabled: true,
  isIpv6Enabled: true,
  defaultRootObject: "index.html",
  aliases: ["ohealth-ng.com", "www.ohealth-ng.com"],
  viewerCertificate: {
    acmCertificateArn: prodCertUsEast1.arn,
    sslSupportMethod: "sni-only",
    minimumProtocolVersion: "TLSv1.2_2021",
  },
  origins: [
    {
      originId: "s3-frontend",
      domainName: frontendBucket.bucketRegionalDomainName,
      s3OriginConfig: {
        originAccessIdentity: oai.cloudfrontAccessIdentityPath,
      },
    },
    {
      originId: "alb-api",
      domainName: alb.dnsName,
      customOriginConfig: {
        httpPort: 80,
        httpsPort: 443,
        originProtocolPolicy: "https-only",
        originSslProtocols: ["TLSv1.2"],
      },
    },
  ],
  defaultCacheBehavior: {
    targetOriginId: "s3-frontend",
    viewerProtocolPolicy: "redirect-to-https",
    allowedMethods: ["GET", "HEAD", "OPTIONS"],
    cachedMethods: ["GET", "HEAD"],
    forwardedValues: {
      queryString: false,
      cookies: { forward: "none" },
    },
    minTtl: 0,
    defaultTtl: 3600,
    maxTtl: 86400,
    compress: true,
  },
  orderedCacheBehaviors: [
    {
      pathPattern: "/api/*",
      targetOriginId: "alb-api",
      viewerProtocolPolicy: "https-only",
      allowedMethods: ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"],
      cachedMethods: ["GET", "HEAD"],
      forwardedValues: {
        queryString: true,
        headers: ["Authorization", "Host"],
        cookies: { forward: "all" },
      },
      minTtl: 0,
      defaultTtl: 0,
      maxTtl: 0, // No caching for API requests
    },
  ],
  customErrorResponses: [
    {
      errorCode: 404,
      responseCode: 200,
      responsePagePath: "/index.html", // SPA routing
    },
  ],
  restrictions: {
    geoRestriction: {
      restrictionType: "none",
    },
  },
  tags: {
    Project: "open-health-initiative",
    Environment: "prod",
    Service: "cloudfront",
    ManagedBy: "pulumi",
  },
});
```

### Environment Variable Configuration for Services

**Update ECS Task Definitions to use environment-specific URLs:**

```typescript
// Production API Task Definition
const apiTaskDefinition = new aws.ecs.TaskDefinition("ohi-prod-api-task", {
  family: "ohi-prod-api",
  containerDefinitions: pulumi.jsonStringify([
    {
      name: "api",
      image: apiImageUri,
      portMappings: [{ containerPort: 8080, protocol: "tcp" }],
      environment: [
        { name: "SERVER_HOST", value: "0.0.0.0" },
        { name: "SERVER_PORT", value: "8080" },
        { name: "DB_HOST", value: rdsEndpoint },
        { name: "REDIS_HOST", value: redisEndpoint },
        { name: "FRONTEND_URL", value: "https://ohealth-ng.com" },
        { name: "API_BASE_URL", value: "https://api.ohealth-ng.com" },
        { name: "GRAPHQL_URL", value: "https://graphql.ohealth-ng.com" },
        { name: "SSE_URL", value: "https://api.ohealth-ng.com/stream" }, // Via ALB path routing
        { name: "CORS_ALLOWED_ORIGINS", value: "https://ohealth-ng.com,https://www.ohealth-ng.com" },
      ],
      secrets: [
        { name: "DB_PASSWORD", valueFrom: dbPasswordSecretArn },
        // ... other secrets
      ],
      logConfiguration: {
        logDriver: "awslogs",
        options: {
          "awslogs-group": "/ecs/ohi-prod-api",
          "awslogs-region": "eu-west-1",
          "awslogs-stream-prefix": "api",
        },
      },
    },
  ]),
  // ... other task def properties
});
```

### DNS Propagation Verification

**After delegation, verify DNS resolution:**

```bash
# Check production
dig ohealth-ng.com +short
# Should return CloudFront IP

dig api.ohealth-ng.com +short
# Should return ALB IP

# Check staging
dig staging.ohealth-ng.com +short
dig api.staging.ohealth-ng.com +short

# Check dev
dig dev.ohealth-ng.com +short
dig api.dev.ohealth-ng.com +short

# Verify name server delegation
dig ohealth-ng.com NS +short
# Should return Route 53 name servers

dig staging.ohealth-ng.com NS +short
# Should return staging hosted zone name servers
```

### Squarespace Email Forwarding (If Needed)

**If Squarespace is handling email forwarding** (e.g., hello@ohealth-ng.com), you'll lose this when delegating to Route 53.

**Solution: Recreate email forwarding in Route 53 + SES:**

1. **Verify domain in SES:**
   ```typescript
   const sesDomain = new aws.ses.DomainIdentity("ohi-prod-ses", {
     domain: "ohealth-ng.com",
   });
   
   const sesVerificationRecord = new aws.route53.Record("ohi-prod-ses-verification", {
     zoneId: prodZone.zoneId,
     name: pulumi.interpolate`_amazonses.${sesDomain.domain}`,
     type: "TXT",
     ttl: 600,
     records: [sesDomain.verificationToken],
   });
   ```

2. **Create MX records:**
   ```typescript
   const mxRecord = new aws.route53.Record("ohi-prod-mx", {
     zoneId: prodZone.zoneId,
     name: "ohealth-ng.com",
     type: "MX",
     ttl: 300,
     records: [
       "10 inbound-smtp.eu-west-1.amazonaws.com",
     ],
   });
   ```

3. **Set up SES receipt rules** to forward to your actual email (e.g., Gmail)

**Alternative:** Use a third-party email service (Google Workspace, ProtonMail, etc.) and update MX records accordingly.

### Deployment Checklist

- [ ] Deploy Pulumi stack to create Route 53 hosted zones
- [ ] Export name servers from Pulumi outputs
- [ ] Update Squarespace name servers to Route 53 (production)
- [ ] Wait 24-48 hours for full DNS propagation (check with `dig`)
- [ ] Request ACM certificates in us-east-1 (CloudFront) and eu-west-1 (ALB)
- [ ] Wait for automatic DNS validation (Pulumi handles this)
- [ ] Deploy ALB with host-based routing rules
- [ ] Deploy CloudFront distribution with custom domain
- [ ] Deploy ECS services with environment-specific URLs
- [ ] Verify DNS resolution for all subdomains
- [ ] Test HTTPS on all domains (should see valid SSL cert)
- [ ] Verify HTTP→HTTPS redirect on ALB
- [ ] Test API endpoints: `curl https://api.ohealth-ng.com/health`
- [ ] Test GraphQL endpoint: `curl https://graphql.ohealth-ng.com/health`
- [ ] Test frontend: `curl -I https://ohealth-ng.com`
- [ ] (Optional) Set up email forwarding with SES if needed

### Cost Implications

| Resource | Monthly Cost | Notes |
|----------|-------------|-------|
| Route 53 Hosted Zones | $1.50 (3 zones × $0.50) | First 25 zones |
| Route 53 Queries | $0.40/million | First billion queries: $0.40/million |
| ACM Certificates | **FREE** | No charge for public certificates |
| CloudFront | $0.085/GB (first 10 TB) | Plus request charges (~$0.0075/10k) |
| ALB | $22.50/month (eu-west-1) | ~$0.0225/hour |
| ALB LCU | ~$5-20/month | Depends on traffic |

**Total DNS/CDN Cost:** ~$30-50/month for V1 traffic levels

---

## Next Steps

1. **Approve Architecture Decisions:**
   - ✅ MongoDB Atlas M10 on AWS eu-west-1 ($57/month)
   - ✅ Typesense Cloud Starter ($29/month)
   - ✅ Super strict tagging with AWS Config enforcement
   - ✅ Separate VPCs per environment
   - ✅ Cost optimization with Fargate Spot + right-sizing
   - ✅ Network security architecture with security groups per service
   - ✅ External service IP allowlisting via NAT Gateway Elastic IPs

2. **Proceed to Step 2:** Select AWS region (likely eu-west-1, validate latency)

3. **Proceed to Step 3:** Create Pulumi infrastructure code with:
   - Resource transformations for automatic tagging
   - VPC per environment (3 AZs, public/private/database subnets)
   - NAT Gateways with Elastic IPs for external service access
   - RDS + ElastiCache + MongoDB Atlas + Typesense Cloud integrations
   - ECS Fargate cluster with spot pricing config
   - ALB with HTTPS termination + WAF rules
   - CloudFront + S3 for frontend
   - Secrets Manager for all credentials
   - Security groups with least privilege rules
   - CloudWatch Logs + Alarms

4. **Cost Tracking Setup:**
   - Deploy AWS Config tag compliance rules
   - Create Cost Explorer budgets per environment
   - Set up weekly cost reports via email
   - Configure billing alarms (80%, 100%, 120% thresholds)

---

**Estimated Timeline:**
- Step 2 (Region selection): 1 day
- Step 3 (Pulumi infrastructure): 3-5 days
- Step 4 (CI/CD updates): 2-3 days
- Step 5 (Secrets + migrations): 2-3 days
- Step 6 (Observability): 2-3 days
- **Total:** 10-15 days to production-ready infrastructure

**Total V1 Monthly Cost:** ~$776-$1,556 (prod), ~$300-$500 (dev/staging)

---

## 8. AWS Region Selection & Latency Analysis

### Primary Target Market: Nigeria

**Geographic Considerations:**
- Primary users: Nigeria (Lagos, Abuja, Port Harcourt, Kano, Ibadan)
- Secondary markets: Ghana, Kenya, South Africa (future expansion)
- Platform criticality: Healthcare pricing discovery (near real-time search)
- Expected traffic pattern: 80% Nigeria, 15% West Africa, 5% diaspora/international

### AWS Global Infrastructure - Africa & Europe Proximity

**AWS does NOT have a region in Africa as of February 2026**

**Nearest AWS Regions to Nigeria:**

| Region Code | Region Name | Approx Distance from Lagos | Typical Latency | Service Availability | Compliance |
|------------|-------------|---------------------------|-----------------|---------------------|------------|
| **eu-west-1** | Ireland (Dublin) | ~5,000 km | **100-120ms** | ✅ Full (all services) | ✅ GDPR-compliant |
| **eu-south-1** | Italy (Milan) | ~4,500 km | **120-140ms** | ⚠️ Limited (no ElastiCache NodeJS SDK support) | ✅ GDPR-compliant |
| **me-south-1** | Bahrain (Manama) | ~5,500 km | **130-150ms** | ⚠️ Limited (opt-in required) | ⚠️ Middle East data residency |
| **eu-central-1** | Germany (Frankfurt) | ~5,200 km | **110-130ms** | ✅ Full (all services) | ✅ GDPR-compliant |
| **eu-west-2** | UK (London) | ~5,100 km | **105-125ms** | ✅ Full (all services) | ✅ GDPR-compliant |
| **af-south-1** | South Africa (Cape Town) | ~4,800 km | **80-100ms** | ⚠️ Limited (opt-in, higher cost) | ✅ African data residency |

### Latency Testing Methodology

**Real-World Latency Tests from Lagos, Nigeria:**

Run these tests from Nigerian ISPs to validate:

```bash
# Test from Lagos to various AWS regions
# Use CloudPing or custom script

# Ireland (eu-west-1)
ping dynamodb.eu-west-1.amazonaws.com
# Result: ~100-120ms average

# London (eu-west-2)
ping dynamodb.eu-west-2.amazonaws.com
# Result: ~105-125ms average

# Frankfurt (eu-central-1)
ping dynamodb.eu-central-1.amazonaws.com
# Result: ~110-130ms average

# Cape Town (af-south-1)
ping dynamodb.af-south-1.amazonaws.com
# Result: ~80-100ms average (BUT opt-in required, limited services)

# Milan (eu-south-1)
ping dynamodb.eu-south-1.amazonaws.com
# Result: ~120-140ms average

# Bahrain (me-south-1)
ping dynamodb.me-south-1.amazonaws.com
# Result: ~130-150ms average
```

**Expected Results:**
- **Best latency:** af-south-1 (80-100ms) - BUT requires opt-in and has limited service availability
- **Best practical latency:** eu-west-1 (100-120ms) - Full service availability
- **Alternative:** eu-west-2 (105-125ms) or eu-central-1 (110-130ms)

### Region Comparison Matrix

| Factor | eu-west-1 (Ireland) | eu-west-2 (London) | eu-central-1 (Frankfurt) | af-south-1 (Cape Town) | me-south-1 (Bahrain) |
|--------|--------------------|--------------------|-------------------------|------------------------|---------------------|
| **Latency from Lagos** | 100-120ms ✅ | 105-125ms ✅ | 110-130ms ⚠️ | 80-100ms ✅✅ | 130-150ms ❌ |
| **RDS Availability** | ✅ All engines | ✅ All engines | ✅ All engines | ⚠️ Limited instance types | ⚠️ Limited instances |
| **ElastiCache Availability** | ✅ Full | ✅ Full | ✅ Full | ✅ Full | ⚠️ Limited SDK support |
| **ECS Fargate Availability** | ✅ Full | ✅ Full | ✅ Full | ✅ Full | ✅ Full |
| **Graviton (ARM) Instances** | ✅ t4g, m7g | ✅ t4g, m7g | ✅ t4g, m7g | ⚠️ Limited | ⚠️ Limited |
| **Cost (Fargate)** | $0.04048/vCPU-hr | $0.04114/vCPU-hr | $0.04389/vCPU-hr | $0.04856/vCPU-hr (+20%) | $0.04653/vCPU-hr (+15%) |
| **Cost (RDS t4g.medium)** | $0.052/hr | $0.053/hr | $0.057/hr | $0.062/hr (+19%) | $0.060/hr (+15%) |
| **Data Transfer Out** | $0.09/GB | $0.09/GB | $0.09/GB | $0.154/GB (+71%) | $0.11/GB (+22%) |
| **Opt-in Required** | ❌ No | ❌ No | ❌ No | ✅ Yes | ✅ Yes |
| **GDPR Compliance** | ✅ Yes | ✅ Yes | ✅ Yes | ❌ No (African) | ❌ No (Middle East) |
| **Availability Zones** | 3 AZs | 3 AZs | 3 AZs | 3 AZs | 3 AZs |
| **MongoDB Atlas Colocation** | ✅ Available | ✅ Available | ✅ Available | ✅ Available | ✅ Available |
| **Typesense Cloud Colocation** | ✅ Available | ✅ Available | ✅ Available | ❌ Not available | ❌ Not available |

### Recommendation: **eu-west-1 (Ireland)**

**Primary Region:** `eu-west-1` (Ireland - Dublin)

**Rationale:**

#### 1. **Latency (100-120ms from Lagos)**
- ✅ Second-best latency after Cape Town
- ✅ Acceptable for healthcare pricing discovery (non-critical, read-heavy)
- ✅ CloudFront edge caching reduces perceived latency for static content to <50ms

#### 2. **Service Availability**
- ✅ **Full AWS service catalog** (all RDS engines, ElastiCache, Fargate, Graviton instances)
- ✅ **MongoDB Atlas presence** (zero data egress between AWS eu-west-1 and Atlas eu-west-1)
- ✅ **Typesense Cloud availability** (colocated in eu-west-1 for low latency)
- ✅ **No opt-in required** (ready to use)

#### 3. **Cost Efficiency**
- ✅ **Lowest Fargate pricing** in Europe ($0.04048/vCPU-hr)
- ✅ **Standard data transfer costs** ($0.09/GB)
- ✅ **Competitive RDS pricing** ($0.052/hr for t4g.medium)
- ✅ **No premium pricing** (unlike af-south-1 or me-south-1)

#### 4. **Data Residency & Compliance**
- ✅ **GDPR-compliant** (EU region)
- ✅ **Strong data protection laws** (relevant for health data)
- ✅ **Stable regulatory environment** (no geo-political risks)
- ⚠️ **Note:** If strict Nigerian data residency is required by future regulation, we can migrate to af-south-1 when available

#### 5. **Proven Track Record**
- ✅ **Most mature AWS region** in Europe (launched 2007)
- ✅ **Highest availability SLA history**
- ✅ **Large ecosystem** of AWS partners and support
- ✅ **Extensive documentation** and community resources

#### 6. **Multi-AZ High Availability**
- ✅ **3 Availability Zones** in Dublin
- ✅ RDS Multi-AZ automatic failover
- ✅ ElastiCache cluster mode with replicas
- ✅ ECS Fargate distributed across AZs

### Alternative Consideration: **af-south-1 (Cape Town)**

**Why NOT Cape Town (despite better latency)?**

| Factor | Analysis |
|--------|----------|
| **Latency** | ✅ Best latency (80-100ms) - 20-30ms better than Ireland |
| **Cost** | ❌ 15-20% higher for compute, **71% higher data transfer** ($0.154/GB vs $0.09/GB) |
| **Service Availability** | ⚠️ Limited instance types (no latest Graviton, fewer RDS options) |
| **Opt-in** | ❌ Requires AWS account opt-in (adds complexity) |
| **Typesense Cloud** | ❌ Not available in af-south-1 (must use eu-west-1, negating latency benefit) |
| **MongoDB Atlas** | ⚠️ Available but higher pricing than EU regions |
| **Data Egress** | ❌ Significantly higher costs for serving Nigeria (most traffic goes OUT) |
| **Maturity** | ⚠️ Newer region (2020), less proven for production workloads |

**Cost Impact Example (Monthly):**
```
V1 Production Estimate:
- Compute: +$100-150/month (higher instance costs)
- Data Transfer: +$200-400/month (5TB egress × $0.064 premium)
- Typesense Cloud: Must host in eu-west-1 anyway (+latency)
- Total Premium: +$300-550/month (~40% cost increase)
```

**Verdict:** 20-30ms latency improvement does NOT justify 40% cost increase + service limitations for V1.

**Future Migration Path:** Once V1 is validated and generating revenue, we can:
1. Evaluate actual traffic patterns and latency sensitivity
2. Test af-south-1 performance with real users
3. Migrate if 20-30ms matters for user experience
4. AWS supports cross-region replication for gradual migration

### CloudFront Edge Optimization

**Latency Mitigation Strategy:**

Even with eu-west-1 backend, **CloudFront edge caching** provides near-instant access to static content:

**CloudFront Edge Locations Near Nigeria:**
- Lagos, Nigeria (Direct POP)
- Johannesburg, South Africa
- Cairo, Egypt
- Nairobi, Kenya

**Perceived Latency for Users:**
- **Static content (HTML, CSS, JS, images):** <30ms (Lagos edge POP)
- **Cached API responses:** <50ms (Lagos edge POP with cache hit)
- **Dynamic API requests:** 100-120ms (must reach eu-west-1 origin)

**Cache Strategy:**
```typescript
// CloudFront cache behaviors
const cacheBehaviors = [
  {
    pathPattern: "/static/*",
    cacheDuration: 86400, // 24 hours
    viewerProtocolPolicy: "redirect-to-https",
  },
  {
    pathPattern: "/api/facilities/*", // Facility listings
    cacheDuration: 300, // 5 minutes (semi-static data)
    viewerProtocolPolicy: "https-only",
  },
  {
    pathPattern: "/api/services/*", // Service searches
    cacheDuration: 300, // 5 minutes
    viewerProtocolPolicy: "https-only",
  },
  {
    pathPattern: "/api/*", // Other API endpoints
    cacheDuration: 0, // No cache (dynamic)
    viewerProtocolPolicy: "https-only",
  },
];
```

**Result:** Most user interactions feel <50ms even with 100-120ms backend latency.

### Disaster Recovery & Multi-Region Strategy

**V1 Deployment:** Single region (eu-west-1) with multi-AZ within region

**Future V2+ Strategy (Post-Revenue):**

#### Phase 2: Active-Passive DR (3-6 months post-launch)
- **Primary:** eu-west-1 (Ireland)
- **DR:** eu-west-2 (London)
- **Failover:** Route 53 health checks + automatic DNS failover
- **RTO:** 15-30 minutes
- **RPO:** 5-15 minutes (RDS snapshots + S3 replication)

#### Phase 3: Multi-Region Active-Active (1 year post-launch)
- **Europe:** eu-west-1 (Ireland) - Primary for global traffic
- **Africa:** af-south-1 (Cape Town) - Primary for African traffic
- **Routing:** Route 53 geolocation-based routing
- **Data:** Aurora Global Database (cross-region replication <1 second)
- **Cost:** +60-80% infrastructure costs

**Rationale for Phased Approach:**
1. ✅ V1 validation first (does the product work? Do users care about 20ms latency?)
2. ✅ Prove revenue model before 40-80% cost increase
3. ✅ AWS continuously adds services to af-south-1 (wait for feature parity)
4. ✅ Potential AWS Africa expansion (Nigeria, Kenya rumored for 2027-2028)

### Network Connectivity Considerations

**Nigerian Internet Infrastructure:**
- **ISPs:** MTN, Airtel, Glo, 9mobile (all have international peering via undersea cables)
- **Undersea Cables:** MainOne, Glo-1, SAT-3/WASC, ACE (connect Lagos to Europe)
- **Latency to Europe:** 80-120ms typical (fiber optic via undersea cables)
- **Latency to South Africa:** 60-90ms (longer physical distance, fewer hops)

**Why Europe is Competitive with South Africa:**
- ✅ Lagos→Europe uses modern undersea cables (high bandwidth, low latency)
- ✅ Europe has more peering points with Nigerian ISPs
- ⚠️ Lagos→Cape Town may route through Europe anyway (depends on ISP peering)

**Validation Test:**
```bash
# Traceroute from Lagos to AWS regions
traceroute dynamodb.eu-west-1.amazonaws.com
# Expected: Lagos → London → Dublin (~10-15 hops)

traceroute dynamodb.af-south-1.amazonaws.com
# Expected: Lagos → ? → Cape Town (~12-18 hops, may route via Europe!)
```

### Final Region Decision Matrix

| Criteria | Weight | eu-west-1 | eu-west-2 | eu-central-1 | af-south-1 |
|----------|--------|-----------|-----------|--------------|------------|
| Latency | 30% | 9/10 | 8.5/10 | 8/10 | 10/10 |
| Service Availability | 25% | 10/10 | 10/10 | 10/10 | 6/10 |
| Cost | 20% | 10/10 | 9.5/10 | 9/10 | 5/10 |
| External Service Support | 15% | 10/10 | 10/10 | 10/10 | 4/10 |
| Compliance | 5% | 10/10 | 10/10 | 10/10 | 7/10 |
| Maturity/Stability | 5% | 10/10 | 9/10 | 9/10 | 6/10 |
| **Weighted Score** | | **9.55** | **9.28** | **9.05** | **6.95** |

**Winner: eu-west-1 (Ireland)** 🏆

### Region Configuration in Pulumi

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";

// Primary region provider (eu-west-1)
const primaryProvider = new aws.Provider("primary", {
  region: "eu-west-1",
  defaultTags: {
    tags: {
      Project: "open-health-initiative",
      ManagedBy: "pulumi",
      Region: "eu-west-1",
    },
  },
});

// US East 1 provider (for CloudFront certificates - required)
const usEast1Provider = new aws.Provider("us-east-1", {
  region: "us-east-1",
  defaultTags: {
    tags: {
      Project: "open-health-initiative",
      ManagedBy: "pulumi",
      Region: "us-east-1",
      Purpose: "cloudfront-certificates",
    },
  },
});

// Export region configuration
export const primaryRegion = "eu-west-1";
export const primaryRegionFullName = "Europe (Ireland)";
export const availabilityZones = ["eu-west-1a", "eu-west-1b", "eu-west-1c"];
```

### Migration Path to Multi-Region (Future)

**When to Consider:**

1. ✅ V1 revenue validated (>$10k MRR)
2. ✅ User latency complaints (>10% of support tickets)
3. ✅ African market expansion (Ghana, Kenya, South Africa)
4. ✅ af-south-1 service parity achieved (full RDS, ElastiCache options)
5. ✅ Budget allows 40-80% infrastructure cost increase

**Migration Steps:**

1. **Deploy DR in eu-west-2 (London)** - Passive standby
2. **Implement Aurora Global Database** - Cross-region replication
3. **Deploy af-south-1 read replicas** - Test latency improvements
4. **Gradually shift traffic** - Route 53 weighted routing (10% → 50% → 100%)
5. **Monitor costs vs. latency benefits** - Validate ROI

**Estimated Migration Timeline:** 4-6 weeks (infrastructure + testing)

### Summary

**Selected Region:** `eu-west-1` (Ireland - Dublin)

**Key Benefits:**
- ✅ Excellent latency to Nigeria (100-120ms)
- ✅ Full AWS service availability
- ✅ Lowest cost in Europe
- ✅ MongoDB Atlas + Typesense Cloud colocated
- ✅ GDPR-compliant
- ✅ CloudFront edge caching (<30ms for static content)

**V1 Architecture:**
- **Primary Region:** eu-west-1 (3 Availability Zones)
- **Multi-AZ:** RDS, ElastiCache, ECS Fargate
- **Global CDN:** CloudFront (Lagos edge POP)
- **DR Strategy:** Multi-AZ within region (V1), cross-region (V2+)

**Cost Impact:** No premium (baseline AWS pricing)

**Next Steps:**
- ✅ Proceed to Pulumi infrastructure implementation in eu-west-1
- ✅ Configure CloudFront with Lagos edge optimization
- ✅ Set up Route 53 with eu-west-1 endpoints
- ✅ Deploy MongoDB Atlas in eu-west-1
- ✅ Deploy Typesense Cloud in eu-west-1

---

## Next Steps
