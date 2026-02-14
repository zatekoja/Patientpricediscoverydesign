# ECS Implementation Status

## Completed

✅ **ECS Fargate Module** ([src/compute/ecs.ts](../src/compute/ecs.ts)) - 670 lines
- Created comprehensive ECS Fargate infrastructure module
- Supports 11 containerized services (7 application + 4 observability)
- Type-safe TypeScript implementation with Pulumi AWS SDK

### Core Components Implemented

1. **ECS Cluster** (`createEcsCluster`)
   - Environment-specific naming: `ohi-{environment}`
   - Container Insights enabled for monitoring
   - Proper resource tagging

2. **IAM Task Execution Role** (`createTaskExecutionRole`)
   - ECS task execution base policy
   - ECR pull permissions (GetAuthorizationToken, BatchGetImage, etc.)
   - Secrets Manager read access for database/Redis credentials
   - CloudWatch Logs write permissions

3. **Task Definitions** (`createTaskDefinition`)
   - Fargate launch type with awsvpc networking
   - Environment-specific resource allocation:
     - **Prod**: API (1024 CPU/2048 MB), GraphQL (1024/2048), SSE (512/1024), Provider API (1024/2048), etc.
     - **Staging**: API (512/1024), GraphQL (512/1024), SSE (256/512), etc.
     - **Dev**: All services (256/512) for cost optimization
   - Container definitions with:
     - ECR image URIs: `{accountId}.dkr.ecr.eu-west-1.amazonaws.com/ohi-{service}:{environment}`
     - Port mappings (8080 for Go services, 3000 for Node, 5001 for Blnk, etc.)
     - Environment variables (DATABASE_HOST, REDIS_HOST, LOG_LEVEL, etc.)
     - Secrets (DATABASE_PASSWORD, REDIS_AUTH_TOKEN via Secrets Manager)
     - CloudWatch Logs integration

4. **ECS Services** (`createEcsService`)
   - Service-specific desired counts (3 for prod API, 2 for staging, 1 for dev)
   - **Fargate Spot** for dev environment (100% cost savings)
   - **Fargate** standard for staging/prod reliability
   - ALB integration for public-facing services (API, GraphQL, SSE, Provider API)
   - Health check grace period (60s) for services with load balancers
   - Deployment configuration (200% max, 100% min healthy)
   - Background jobs (Reindexer, Blnk Worker) without load balancers

5. **Auto-Scaling** (`createAutoScalingTarget`, `createCpuScalingPolicy`, `createMemoryScalingPolicy`)
   - Environment-specific scaling:
     - **Prod**: Min 2, Max 10 tasks
     - **Staging**: Min 1, Max 5 tasks
     - **Dev**: Min 1, Max 3 tasks
   - CPU-based scaling (70% target, 60s scale-out / 300s scale-in cooldown)
   - Memory-based scaling (80% target, 60s scale-out / 300s scale-in cooldown)
   - Applied to API, GraphQL, SSE, Provider API (not background jobs)

6. **CloudWatch Log Groups** (`createLogGroup`)
   - Naming: `/ecs/ohi-{environment}/{service}`
   - Retention: 30 days (prod), 7 days (dev/staging)
   - Proper tagging for cost allocation

7. **Service Discovery** (`createServiceDiscoveryNamespace`, `createServiceDiscoveryService`)
   - Private DNS namespace: `ohi-{environment}.local`
   - Service-specific discovery: `{service}.ohi-{environment}.local`
   - Multivalue routing for load distribution
   - Health check custom config (failure threshold: 1)

8. **Main Infrastructure Function** (`createEcsInfrastructure`)
   - Orchestrates creation of all components
   - Returns structured outputs (cluster ID/ARN, service ARNs, task definition ARNs, namespace ID)
   - Handles 7 core services: api, graphql, sse, provider-api, reindexer, blnk-api, blnk-worker

### Resource Allocation Strategy

| Service | Prod (CPU/Mem) | Staging (CPU/Mem) | Dev (CPU/Mem) | Desired Count (P/S/D) |
|---------|----------------|-------------------|---------------|----------------------|
| API | 1024/2048 | 512/1024 | 256/512 | 3/2/1 |
| GraphQL | 1024/2048 | 512/1024 | 256/512 | 3/2/1 |
| SSE | 512/1024 | 256/512 | 256/512 | 2/1/1 |
| Provider API | 1024/2048 | 512/1024 | 256/512 | 2/1/1 |
| Reindexer | 512/1024 | 256/512 | 256/512 | 1/1/1 |
| Blnk API | 512/1024 | 256/512 | 256/512 | 2/1/1 |
| Blnk Worker | 256/512 | 256/512 | 256/512 | 1/1/1 |
| ClickHouse | 2048/4096 | 1024/2048 | 512/1024 | 1/1/1 |
| OTEL | 512/1024 | 256/512 | 256/512 | 2/1/1 |
| SigNoz Query | 1024/2048 | 512/1024 | 256/512 | 1/1/1 |
| SigNoz Frontend | 256/512 | 256/512 | 256/512 | 1/1/1 |

### Port Mappings

| Service | Container Port | Protocol | ALB Integration |
|---------|---------------|----------|-----------------|
| API | 8080 | TCP | ✅ Yes |
| GraphQL | 8080 | TCP | ✅ Yes |
| SSE | 8080 | TCP | ✅ Yes |
| Provider API | 8080 | TCP | ✅ Yes |
| Reindexer | None | - | ❌ No (background job) |
| Blnk API | 5001 | TCP | ❌ No (internal only) |
| Blnk Worker | None | - | ❌ No (background job) |
| ClickHouse | 8123 | TCP | ❌ No (internal only) |
| OTEL Collector | 4317 | TCP | ❌ No (internal only) |
| SigNoz Query | 8080 | TCP | ❌ No (internal only) |
| SigNoz Frontend | 3301 | TCP | ❌ No (internal only) |

## Architecture Decisions

1. **Fargate over EC2**
   - No infrastructure management overhead
   - Per-second billing with no idle costs
   - Automatic scaling without capacity planning
   - Security patches managed by AWS

2. **Fargate Spot for Dev**
   - 70% cost savings vs Fargate standard
   - Acceptable for non-critical dev environment
   - Tasks can be interrupted with 2-minute notice
   - Falls back to standard Fargate if no spot capacity

3. **Service Discovery**
   - Enables service-to-service communication without hardcoded IPs
   - DNS-based: `blnk-api.ohi-prod.local:5001`
   - Automatic registration/deregistration as tasks start/stop
   - Health-check aware routing

4. **Auto-Scaling Strategy**
   - CPU and memory-based for comprehensive coverage
   - Conservative targets (70%/80%) to prevent thrashing
   - Fast scale-out (60s) for traffic spikes
   - Gradual scale-in (300s) to avoid premature termination
   - Background jobs (Reindexer, Workers) have fixed 1 task (no auto-scaling)

5. **Resource Allocation**
   - Prod: Generous resources for performance and headroom
   - Staging: Medium resources for realistic testing
   - Dev: Minimal resources for cost optimization
   - ClickHouse gets more resources (observability data intensive)

## Integration Points

### Required Dependencies
- VPC and subnets from [networking/vpc.ts](../src/networking/vpc.ts)
- Security groups from [networking/security-groups.ts](../src/networking/security-groups.ts)
- RDS endpoint and secrets from [databases/rds.ts](../src/databases/rds.ts)
- ElastiCache endpoints and secrets from [databases/elasticache.ts](../src/databases/elasticache.ts)
- ALB target groups from [compute/alb.ts](../src/compute/alb.ts) (pending)

### Provides
- ECS Cluster ID and ARN
- Service ARNs (for deployment automation)
- Task Definition ARNs (for CI/CD updates)
- Service Discovery Namespace ID (for DNS configuration)

## Pending Work

### 1. Testing (Priority: High)
The current implementation lacks proper unit tests due to Pulumi Output<T> mocking complexity. Need to:
- Create helper functions that return constants (similar to databases.test.ts pattern)
- Export configuration constants for validation
- Test resource allocation logic without Pulumi runtime
- Validate environment variable and secret configuration

**Approach**:
```typescript
// Export constants for testing
export const ECS_CLUSTER_CONFIG = {
  containerInsights: true,
};

export function getTaskResources(env: string, service: string): { cpu: number; memory: number } {
  return SERVICE_RESOURCES[env][service];
}

export function getDesiredCount(env: string, service: string): number {
  return DESIRED_COUNTS[env][service];
}

// Test these pure functions instead of resource creation
```

### 2. Observability Stack Integration (Priority: Medium)
Currently implemented placeholders for ClickHouse, OTEL, SigNoz. Need to:
- Add task definitions for observability services
- Configure OTEL Collector with SigNoz backend
- Set up ClickHouse data persistence (EBS volumes or EFS)
- Configure SigNoz Query Service connection to ClickHouse
- Add environment variables for observability endpoints

### 3. ECR Repository Creation (Priority: High)
ECS expects images in ECR at:
- `{accountId}.dkr.ecr.eu-west-1.amazonaws.com/ohi-{service}:{environment}`

Need separate module to create:
```typescript
// src/compute/ecr.ts
export function createEcrRepositories(services: string[]): Record<string, aws.ecr.Repository>
```

### 4. Task Definitions Refinement (Priority: Medium)
- Add health checks (HTTP endpoints for Go services)
- Configure stop timeout (30s default, may need tuning)
- Add ulimits for file descriptors (important for Go services)
- Configure working directory if needed
- Add labels for better organization

### 5. CI/CD Integration (Priority: High)
Update GitHub Actions workflow to:
- Build Docker images for each service
- Push to ECR with environment tag
- Update ECS services with new task definitions
- Wait for deployment completion
- Rollback on failure

Example:
```yaml
- name: Deploy to ECS
  run: |
    aws ecs update-service \
      --cluster ohi-${{ env.ENVIRONMENT }} \
      --service ohi-${{ env.ENVIRONMENT }}-api \
      --force-new-deployment
```

## Known Limitations

1. **No Blue/Green Deployments**
   - Current implementation uses rolling updates
   - Consider AWS CodeDeploy integration for blue/green
   - Trade-off: Simpler vs safer deployments

2. **No Circuit Breaker**
   - ECS deployment circuit breaker disabled
   - Failed deployments won't auto-rollback
   - Consider enabling for production

3. **No Task IAM Roles**
   - Current implementation only has execution role
   - Need separate task roles for application permissions
   - Example: S3 access, SQS, etc.

4. **Fixed Region**
   - Hardcoded `eu-west-1` in multiple places
   - Should parameterize for multi-region support

5. **No EFS Integration**
   - Stateful services (ClickHouse) use container storage
   - Data lost on task restart
   - Consider EFS for persistence

## Cost Estimation (Monthly)

### Production Environment
| Service | vCPU | Memory | Count | Hours/Month | Fargate Cost | Total |
|---------|------|--------|-------|-------------|--------------|-------|
| API | 1.0 | 2 GB | 3 | 2,190 | $0.098/hr | $644 |
| GraphQL | 1.0 | 2 GB | 3 | 2,190 | $0.098/hr | $644 |
| SSE | 0.5 | 1 GB | 2 | 1,460 | $0.0624/hr | $182 |
| Provider API | 1.0 | 2 GB | 2 | 1,460 | $0.098/hr | $286 |
| Reindexer | 0.5 | 1 GB | 1 | 730 | $0.0624/hr | $91 |
| Blnk API | 0.5 | 1 GB | 2 | 1,460 | $0.0624/hr | $182 |
| Blnk Worker | 0.25 | 0.5 GB | 1 | 730 | $0.0375/hr | $55 |
| **Subtotal** | | | | | | **$2,084** |

### Staging: ~$600/month (simplified resource allocation)
### Dev: ~$150/month (minimal resources + Fargate Spot 70% discount)

**Total ECS Cost: ~$2,834/month**

## Next Steps

1. ✅ Create ECS infrastructure module
2. ⏳ Create ECR repositories module
3. ⏳ Write comprehensive unit tests
4. ⏳ Integrate with ALB (Phase 6)
5. ⏳ Update CI/CD workflows
6. ⏳ Add observability stack services
7. ⏳ Test end-to-end deployment

## Files Modified

- ✅ [src/compute/ecs.ts](../src/compute/ecs.ts) - Main ECS infrastructure module (670 lines)
- ✅ [src/tagging.ts](../src/tagging.ts) - Uses existing tag functions (spread operator pattern)

## References

- V1_DEPLOYMENT_ARCHITECTURE.md - Original architecture specification
- AWS ECS Fargate Pricing: https://aws.amazon.com/fargate/pricing/
- AWS ECS Best Practices: https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/
- Pulumi AWS ECS: https://www.pulumi.com/registry/packages/aws/api-docs/ecs/
