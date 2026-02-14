# Phase 8: CI/CD Pipeline - Complete AWS Deployment

## Status: ✅ Complete

## Overview

Implemented complete CI/CD pipeline for AWS deployment with ECR, ECS, S3, and CloudFront. This phase provides:
- **ECR repositories** for Docker images
- **Deployment scripts** for automated rollouts
- **GitHub Actions workflow** for CI/CD
- **Zero-downtime deployments** with health checks
- **Database migrations** via ECS RunTask

## Files Created/Updated

### 1. ECR Module - [src/compute/ecr.ts](../src/compute/ecr.ts) (355 lines)

**Purpose**: Docker image repository management

**Key Features**:
- 11 ECR repositories for all services
- Lifecycle policies (keep last 10 images, remove untagged after 1 day)
- Image scanning on push (prod only)
- Immutable tags by default
- AES256 encryption
- Cross-account access support
- Replication for DR (optional)

**Repositories Created**:
```typescript
const ECR_SERVICES = [
  'api',              // Main API service
  'graphql',          // GraphQL server
  'sse',              // Server-Sent Events
  'provider-api',     // Provider API
  'reindexer',        // Data reindexer
  'blnk-api',         // Blnk API
  'blnk-worker',      // Blnk worker
  'clickhouse',       // ClickHouse (custom image)
  'otel',             // OpenTelemetry collector
  'signoz-query',     // SigNoz query service
  'signoz-frontend',  // SigNoz frontend
];
```

**Functions Exported**:
- `createEcrRepository()` - Create repository with scanning, encryption
- `createLifecyclePolicy()` - Manage image retention
- `createEcrAccessPolicy()` - IAM policy for push/pull
- `createRepositoryPolicy()` - Cross-account access
- `createReplicationConfiguration()` - Multi-region replication
- `getRegistryUrl()` - Get ECR registry URL
- `getImageUri()` - Construct full image URI
- `createEcrInfrastructure()` - Complete ECR setup

**Outputs**:
```typescript
{
  repositoryArns: Record<EcrService, pulumi.Output<string>>;
  repositoryUrls: Record<EcrService, pulumi.Output<string>>;
  repositoryNames: Record<EcrService, pulumi.Output<string>>;
  registryId: pulumi.Output<string>;
}
```

### 2. ECS Deployment Script - [scripts/deploy-ecs.sh](../../scripts/deploy-ecs.sh) (275 lines)

**Purpose**: Zero-downtime ECS service deployments

**Features**:
- Validates image exists in ECR
- Creates new task definition revision
- Updates ECS service with rolling deployment
- Waits for deployment completion (10min max)
- Health check monitoring
- Rollback support on failure
- Dry-run mode for testing

**Usage**:
```bash
# Deploy API to prod
./scripts/deploy-ecs.sh prod api abc123f

# Deploy GraphQL to staging
./scripts/deploy-ecs.sh staging graphql latest

# Dry run (preview only)
DRY_RUN=true ./scripts/deploy-ecs.sh dev api test-tag

# Skip waiting for deployment
SKIP_WAIT=true ./scripts/deploy-ecs.sh dev sse latest
```

**Arguments**:
- `environment` - Target environment (dev, staging, prod)
- `service` - Service name (api, graphql, sse, provider-api, reindexer)
- `image-tag` - Docker image tag (commit SHA or 'latest')

**Environment Variables**:
- `AWS_REGION` - AWS region (default: eu-west-1)
- `DRY_RUN` - Preview without deploying (default: false)
- `SKIP_WAIT` - Skip deployment wait (default: false)

**Deployment Process**:
1. Validate AWS credentials
2. Check image exists in ECR
3. Fetch current task definition
4. Update container image
5. Register new task definition
6. Update ECS service (force new deployment)
7. Monitor deployment status
8. Wait for `COMPLETED` rollout state
9. Verify running tasks match desired count

### 3. Frontend Deployment Script - [scripts/deploy-frontend.sh](../../scripts/deploy-frontend.sh) (245 lines)

**Purpose**: Deploy static assets to S3 with CloudFront invalidation

**Features**:
- Builds frontend with environment-specific config
- Uploads to S3 with cache-control headers
- CloudFront distribution lookup
- Creates invalidation for cache refresh
- Waits for invalidation completion
- SPA routing support (HTML no-cache)

**Usage**:
```bash
# Deploy to prod (auto-lookup distribution)
./scripts/deploy-frontend.sh prod

# Deploy to staging with specific distribution ID
./scripts/deploy-frontend.sh staging E1234ABCD5678

# Dry run
DRY_RUN=true ./scripts/deploy-frontend.sh dev

# Deploy without rebuilding
SKIP_BUILD=true ./scripts/deploy-frontend.sh prod

# Skip CloudFront invalidation
SKIP_INVALIDATION=true ./scripts/deploy-frontend.sh dev
```

**Arguments**:
- `environment` - Target environment (dev, staging, prod)
- `distribution-id` - CloudFront distribution ID (optional)

**Environment Variables**:
- `AWS_REGION` - AWS region (default: eu-west-1)
- `BUILD_DIR` - Build output directory (default: dist)
- `DRY_RUN` - Preview without deploying
- `SKIP_BUILD` - Skip npm build step
- `SKIP_INVALIDATION` - Skip CloudFront invalidation

**Deployment Process**:
1. Build frontend (`npm run build`)
2. Set environment variables:
   - `VITE_API_URL` - API endpoint URL
   - `VITE_GRAPHQL_URL` - GraphQL endpoint URL
   - `VITE_SSE_URL` - SSE endpoint URL
   - `VITE_ENVIRONMENT` - Target environment
3. Sync files to S3:
   - Assets: `cache-control: public,max-age=31536000,immutable`
   - HTML/JSON: `cache-control: public,max-age=0,must-revalidate`
4. Lookup CloudFront distribution ID
5. Create invalidation (`/*`)
6. Wait for completion (10min max)

### 4. Database Migration Script - [scripts/run-migrations.sh](../../scripts/run-migrations.sh) (290 lines)

**Purpose**: Run database migrations via ECS RunTask

**Features**:
- Runs migrations in isolated ECS task
- Supports up, down, status, force, version
- Production confirmation prompt
- VPC and security group lookup
- CloudWatch logs fetching
- Exit code validation

**Usage**:
```bash
# Run migrations in prod
./scripts/run-migrations.sh prod up

# Check migration status
./scripts/run-migrations.sh staging status

# Rollback last migration
./scripts/run-migrations.sh dev down

# Dry run
DRY_RUN=true ./scripts/run-migrations.sh prod up

# Force approval (CI/CD)
FORCE_YES=true ./scripts/run-migrations.sh prod up
```

**Arguments**:
- `environment` - Target environment (dev, staging, prod)
- `direction` - Migration direction (up, down, status, force, version)

**Environment Variables**:
- `AWS_REGION` - AWS region (default: eu-west-1)
- `DRY_RUN` - Preview without running
- `TIMEOUT` - Max wait time in seconds (default: 300)
- `FORCE_YES` - Skip confirmation prompt

**Migration Process**:
1. Confirm production migrations (if applicable)
2. Fetch VPC configuration
3. Get private subnets
4. Get security group for API
5. Build migration command
6. Run ECS task with command override
7. Monitor task status
8. Fetch CloudWatch logs
9. Verify exit code

### 5. GitHub Actions Workflow - [.github/workflows/cd-deploy.yml](../../.github/workflows/cd-deploy.yml) (454 lines)

**Purpose**: Automated deployment pipeline

**Workflow Structure**:
1. **Plan Deployment** - Determine what to deploy based on changes or manual input
2. **Manual Approval** - Required for staging/prod (skipped for dev)
3. **Deploy Infrastructure** - Pulumi infrastructure updates
4. **Deploy Backend** - Docker build, push, migrations, ECS updates
5. **Deploy Frontend** - Build, S3 sync, CloudFront invalidation
6. **Deployment Summary** - Show results and URLs

**Trigger Conditions**:
- `workflow_call` - Called from main pipeline
- `workflow_dispatch` - Manual trigger from GitHub UI

**Inputs**:
- `environment` - dev, staging, prod
- `deployment-type` - auto, backend-only, frontend-only, infrastructure-only, full
- `skip-approval` - Skip manual approval (dev only)

**Updated Sections**:
- Domain changed from `ohealth-ng.com` → `ateru.ng`
- ECR repository naming: `ohi-{env}-{service}`
- Backend deployment uses `./scripts/deploy-ecs.sh`
- Frontend deployment uses `./scripts/deploy-frontend.sh`
- Database migrations run before ECS deployment
- CloudFront invalidation included in frontend deployment

**Environment Configuration**:
```yaml
env:
  GO_VERSION: '1.25.0'
  NODE_VERSION: '18'
  AWS_REGION: 'eu-west-1'
  DOMAIN: 'ateru.ng'
```

**Deployment Matrix**:
```yaml
# Backend services deployed
services: [api, graphql, sse, provider-api, reindexer]

# Frontend URLs
prod: https://ateru.ng
staging: https://staging.ateru.ng
dev: https://dev.ateru.ng
```

## Infrastructure Integration

### ECR Cost Estimation (Monthly)

- **Repositories**: 11 × $0.10 = $1.10
- **Storage**: ~10GB × $0.10/GB = $1.00 (assuming 10 images per repo)
- **Data Transfer**: $0.09/GB (first 10TB)
  - Assume 50GB/month = $4.50
- **Subtotal**: ~$6.60/month

### Deployment Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      GitHub Actions                          │
│  (Trigger: push to main, manual, or workflow_call)          │
└──────────────────┬──────────────────────────────────────────┘
                   │
        ┌──────────▼──────────┐
        │  Plan Deployment    │  Auto-detect changes or use manual input
        └──────────┬──────────┘
                   │
        ┌──────────▼──────────┐
        │  Manual Approval    │  Required for staging/prod
        └──────────┬──────────┘
                   │
     ┌─────────────┴─────────────┐
     │                           │
┌────▼────────┐         ┌───────▼────────┐
│Infrastructure│        │    Backend     │
│   (Pulumi)  │        │  (Docker+ECS)  │
└────┬────────┘         └───────┬────────┘
     │                           │
     │                  ┌────────▼────────┐
     │                  │ Build & Push    │
     │                  │ ECR Images      │
     │                  └────────┬────────┘
     │                           │
     │                  ┌────────▼────────┐
     │                  │ Run Migrations  │
     │                  │  (ECS RunTask)  │
     │                  └────────┬────────┘
     │                           │
     │                  ┌────────▼────────┐
     │                  │ Deploy to ECS   │
     │                  │ (Rolling Update)│
     │                  └────────┬────────┘
     │                           │
     └───────────┬───────────────┘
                 │
        ┌────────▼────────┐
        │    Frontend     │
        │ (S3+CloudFront) │
        └────────┬────────┘
                 │
        ┌────────▼────────┐
        │ Build & Upload  │
        │   to S3 Bucket  │
        └────────┬────────┘
                 │
        ┌────────▼────────┐
        │   Invalidate    │
        │   CloudFront    │
        └────────┬────────┘
                 │
        ┌────────▼────────┐
        │    Deployment   │
        │     Summary     │
        └─────────────────┘
```

## Deployment Examples

### Full Deployment (Production)

```bash
# Trigger from GitHub Actions UI
# 1. Select "cd-deploy" workflow
# 2. Choose environment: prod
# 3. Choose deployment-type: full
# 4. Click "Run workflow"
# 5. Approve deployment when prompted

# Or trigger via API
gh workflow run cd-deploy.yml \
  -f environment=prod \
  -f deployment-type=full
```

### Backend-Only Deployment

```bash
# Build and deploy API service manually
cd backend
IMAGE_TAG=$(git rev-parse --short HEAD)

# Build and push
docker build -t 123456789.dkr.ecr.eu-west-1.amazonaws.com/ohi-prod-api:$IMAGE_TAG .
aws ecr get-login-password --region eu-west-1 | \
  docker login --username AWS --password-stdin 123456789.dkr.ecr.eu-west-1.amazonaws.com
docker push 123456789.dkr.ecr.eu-west-1.amazonaws.com/ohi-prod-api:$IMAGE_TAG

# Deploy
cd ..
./scripts/deploy-ecs.sh prod api $IMAGE_TAG
```

### Frontend-Only Deployment

```bash
# Build and deploy frontend manually
./scripts/deploy-frontend.sh prod
```

### Database Migrations

```bash
# Check current migration version
./scripts/run-migrations.sh prod status

# Run pending migrations
./scripts/run-migrations.sh prod up

# Rollback last migration (if needed)
./scripts/run-migrations.sh prod down
```

## Testing Checklist

### Pre-Deployment Tests

- [ ] All unit tests passing (`npm test` in pulumi/)
- [ ] Infrastructure compiles (`npx tsc --noEmit`)
- [ ] Backend builds successfully (`cd backend && go build`)
- [ ] Frontend builds successfully (`npm run build`)
- [ ] Environment variables configured correctly
- [ ] AWS credentials valid

### Post-Deployment Tests

- [ ] ECR repositories created and accessible
- [ ] Docker images pushed successfully
- [ ] ECS services updated and running
- [ ] Database migrations applied
- [ ] Frontend accessible via CloudFront
- [ ] API endpoints responding
- [ ] GraphQL endpoint functional
- [ ] SSE endpoint streaming
- [ ] Health checks passing
- [ ] CloudWatch logs flowing
- [ ] Metrics appearing in SigNoz

### Rollback Plan

If deployment fails:

1. **Backend**: Revert ECS service to previous task definition
   ```bash
   aws ecs update-service \
     --cluster ohi-prod \
     --service ohi-prod-api \
     --task-definition ohi-prod-api:PREVIOUS_REVISION
   ```

2. **Frontend**: Revert S3 bucket (versioning enabled)
   ```bash
   aws s3 sync s3://ohi-prod-frontend-hosting/ s3://ohi-prod-frontend-backup/
   # Then restore from backup
   ```

3. **Database**: Rollback migrations
   ```bash
   ./scripts/run-migrations.sh prod down
   ```

4. **Infrastructure**: Revert Pulumi stack
   ```bash
   cd infrastructure/pulumi
   pulumi stack select prod
   pulumi history
   pulumi stack export --version N > stack-backup.json
   pulumi import < stack-backup.json
   ```

## Security Considerations

### Secrets Management

- AWS credentials stored as GitHub secrets
- Database passwords in AWS Secrets Manager
- JWT secrets in AWS Secrets Manager
- API keys in AWS Secrets Manager
- No secrets in code or logs

### IAM Permissions

**GitHub Actions needs**:
- `ecr:*` - Push Docker images
- `ecs:*` - Update services and run tasks
- `s3:*` - Upload frontend files
- `cloudfront:CreateInvalidation` - Invalidate cache
- `secretsmanager:GetSecretValue` - Fetch secrets
- `logs:GetLogEvents` - Fetch migration logs

**ECS Task Execution Role needs**:
- `ecr:GetAuthorizationToken` - Pull images
- `ecr:BatchCheckLayerAvailability` - Verify layers
- `ecr:GetDownloadUrlForLayer` - Download layers
- `ecr:BatchGetImage` - Get images
- `secretsmanager:GetSecretValue` - Access secrets
- `logs:CreateLogStream` - Create log streams
- `logs:PutLogEvents` - Write logs

### Network Security

- ECS tasks run in private subnets
- ALB in public subnets
- Security groups restrict traffic
- CloudFront provides DDoS protection
- S3 bucket private (CloudFront OAI only)

## Monitoring & Alerts

### CloudWatch Alarms

Create alarms for:
- ECS service deployment failures
- High error rate (5xx responses)
- Low healthy host count (ALB)
- CloudFront error rate > 5%
- Database connection errors
- Migration task failures

### SigNoz Dashboards

Monitor:
- Deployment duration trends
- Rollback frequency
- Error rates before/after deployment
- Response time changes
- Database query performance

## Cost Optimization

### ECR
- Use lifecycle policies (implemented)
- Remove old images automatically
- Only scan prod images

### ECS
- Use Fargate Spot for dev (70% savings)
- Right-size task definitions
- Scale down during off-hours

### CloudFront
- Cache static assets aggressively
- Use compression
- Regional edge caches

### Total Estimated Cost (Monthly)
- ECR: $6.60
- ECS: $2,534 (prod), $880 (staging), $310 (dev)
- S3: $3 (storage) + $5 (requests)
- CloudFront: $50-200 (varies with traffic)
- **Total**: ~$3,788-$3,938/month

## Next Steps

1. ✅ ECR module created
2. ✅ Deployment scripts created
3. ✅ GitHub Actions workflow updated
4. ⏳ Create Pulumi index.ts to wire everything together
5. ⏳ Test deployment to dev environment
6. ⏳ Configure GitHub secrets (AWS credentials)
7. ⏳ Request ACM certificates
8. ⏳ Update DNS records
9. ⏳ Deploy infrastructure (`pulumi up`)
10. ⏳ Deploy backend services
11. ⏳ Deploy frontend
12. ⏳ Smoke test end-to-end

## Troubleshooting

### ECR Image Push Fails

**Problem**: Authentication error
```bash
# Solution: Refresh ECR login
aws ecr get-login-password --region eu-west-1 | \
  docker login --username AWS --password-stdin 123456789.dkr.ecr.eu-west-1.amazonaws.com
```

### ECS Deployment Stuck

**Problem**: Service not reaching steady state
```bash
# Check service events
aws ecs describe-services \
  --cluster ohi-prod \
  --services ohi-prod-api \
  --query 'services[0].events[:10]'

# Check task stopped reason
aws ecs describe-tasks \
  --cluster ohi-prod \
  --tasks TASK_ID \
  --query 'tasks[0].stoppedReason'
```

### CloudFront Invalidation Slow

**Problem**: Cache not refreshing fast enough
```bash
# Check invalidation status
aws cloudfront get-invalidation \
  --distribution-id E1234 \
  --id INVALIDATION_ID

# Wait for completion
aws cloudfront wait invalidation-completed \
  --distribution-id E1234 \
  --id INVALIDATION_ID
```

### Database Migration Fails

**Problem**: Migration task exits with non-zero code
```bash
# Check task logs
TASK_ID=$(aws ecs list-tasks \
  --cluster ohi-prod \
  --service-name ohi-prod-api \
  --query 'taskArns[0]' \
  --output text | rev | cut -d'/' -f1 | rev)

aws logs tail /ecs/ohi-prod-api --follow \
  --log-stream-names ecs/api/$TASK_ID
```

## References

- [AWS ECR Documentation](https://docs.aws.amazon.com/ecr/)
- [AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)
- [CloudFront Invalidation](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/Invalidation.html)
- [GitHub Actions](https://docs.github.com/en/actions)
- [Pulumi AWS](https://www.pulumi.com/registry/packages/aws/)
