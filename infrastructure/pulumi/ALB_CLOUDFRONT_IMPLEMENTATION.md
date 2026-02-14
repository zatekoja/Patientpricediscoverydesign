# Phase 6: ALB and CloudFront Implementation

## Status: ✅ Complete

## Overview

Implemented Application Load Balancer (ALB) and CloudFront CDN infrastructure for the Open Health Initiative V1 deployment. This phase provides:
- **Load balancing** for ECS Fargate services
- **SSL/TLS termination** at the load balancer
- **Host-based routing** for microservices
- **CDN distribution** for frontend assets
- **Origin Access Identity** for secure S3 access

## Files Created

### 1. ALB Module - [src/compute/alb.ts](../src/compute/alb.ts) (242 lines)

**Purpose**: Application Load Balancer for routing traffic to ECS services

**Key Features**:
- Internet-facing ALB in public subnets
- Target groups for 4 public services (API, GraphQL, SSE, Provider API)
- HTTP → HTTPS redirect (301 permanent)
- HTTPS listener with TLS 1.3 policy
- Host-based routing rules (api.ateru.ng, graphql.ateru.ng, etc.)
- Health checks every 30 seconds with 5-second timeout
- Session stickiness (24-hour cookie duration)
- 30-second deregistration delay (300s for SSE long-lived connections)

**Functions Exported**:
- `createApplicationLoadBalancer()` - Main ALB resource
- `createTargetGroup()` - Target group with health checks
- `createHttpListener()` - HTTP listener (redirects to HTTPS)
- `createHttpsListener()` - HTTPS listener with ACM certificate
- `createListenerRule()` - Host-based routing rules
- `createAlbInfrastructure()` - Complete ALB setup
- `getHealthCheckConfig()` - Health check configuration helper
- `getTargetGroupAttributes()` - Service-specific attributes
- `getAlbAccessLogsConfig()` - Access logging configuration

**Outputs**:
```typescript
{
  albArn: pulumi.Output<string>;
  albDnsName: pulumi.Output<string>; // e.g., ohi-prod-123456789.eu-west-1.elb.amazonaws.com
  albZoneId: pulumi.Output<string>;
  targetGroupArns: {
    api: pulumi.Output<string>;
    graphql: pulumi.Output<string>;
    sse: pulumi.Output<string>;
    providerApi: pulumi.Output<string>;
  };
  httpsListenerArn?: pulumi.Output<string>;
  httpListenerArn: pulumi.Output<string>;
}
```

### 2. CloudFront Module - [src/compute/cloudfront.ts](../src/compute/cloudfront.ts) (268 lines)

**Purpose**: CloudFront CDN for frontend static assets

**Key Features**:
- S3 bucket for frontend hosting (React + Vite build)
- Origin Access Identity (OAI) for secure S3 access
- HTTP → HTTPS redirect
- Custom error pages (403/404 → index.html for SPA routing)
- Environment-specific caching:
  - **Prod**: 24-hour default, 1-year max
  - **Dev/Staging**: 5-minute default, 1-hour max
- Gzip/Brotli compression enabled
- IPv6 support
- Custom domain support (ateru.ng, staging.ateru.ng, dev.ateru.ng)
- Versioning enabled for prod
- CORS configuration for API calls
- Lifecycle rules (30-day expiration for dev/staging)

**Functions Exported**:
- `createOriginAccessIdentity()` - OAI for S3 access
- `createS3BucketPolicy()` - Allow CloudFront OAI
- `createCloudFrontDistribution()` - Main CDN distribution
- `createFrontendBucket()` - S3 bucket for static assets
- `createCloudFrontInfrastructure()` - Complete CloudFront setup
- `getApiCacheBehavior()` - Cache behavior for API routes (no caching)
- `getLoggingConfig()` - CloudFront access logging
- `createInvalidation()` - Helper for cache invalidation in CI/CD

**Outputs**:
```typescript
{
  distributionId: pulumi.Output<string>; // e.g., E1234ABCDEF
  distributionArn: pulumi.Output<string>;
  distributionDomainName: pulumi.Output<string>; // e.g., d123abc.cloudfront.net
  distributionHostedZoneId: pulumi.Output<string>; // Z2FDTNDATAQYW2 (CloudFront hosted zone)
}
```

## Architecture

### ALB Request Flow

```
Internet (HTTPS)
    ↓
  ALB (Port 443)
    ↓
Host-based Routing:
  - api.ateru.ng       → API Target Group       → ECS Service (api)       → Port 8080
  - graphql.ateru.ng   → GraphQL Target Group   → ECS Service (graphql)   → Port 8080
  - sse.ateru.ng       → SSE Target Group       → ECS Service (sse)       → Port 8080
  - provider.ateru.ng  → Provider API TG        → ECS Service (provider)  → Port 8080
```

### CloudFront Request Flow

```
Internet (HTTPS)
    ↓
  CloudFront (d123abc.cloudfront.net or ateru.ng)
    ↓
  Origin Access Identity (OAI)
    ↓
  S3 Bucket (ohi-prod-frontend)
    ↓
  React SPA (index.html, assets/)
```

## Domain Configuration

### Production (ateru.ng)
- **Frontend**: `ateru.ng` → CloudFront → S3
- **API**: `api.ateru.ng` → ALB → ECS
- **GraphQL**: `graphql.ateru.ng` → ALB → ECS
- **SSE**: `sse.ateru.ng` → ALB → ECS
- **Provider**: `provider.ateru.ng` → ALB → ECS

### Staging (staging.ateru.ng)
- **Frontend**: `staging.ateru.ng` → CloudFront → S3
- **API**: `api.staging.ateru.ng` → ALB → ECS
- **GraphQL**: `graphql.staging.ateru.ng` → ALB → ECS
- **SSE**: `sse.staging.ateru.ng` → ALB → ECS
- **Provider**: `provider.staging.ateru.ng` → ALB → ECS

### Development (dev.ateru.ng)
- **Frontend**: `dev.ateru.ng` → CloudFront → S3
- **API**: `api.dev.ateru.ng` → ALB → ECS
- **GraphQL**: `graphql.dev.ateru.ng` → ALB → ECS
- **SSE**: `sse.dev.ateru.ng` → ALB → ECS
- **Provider**: `provider.dev.ateru.ng` → ALB → ECS

## SSL/TLS Configuration

### Requirements
1. **ALB Certificate** (eu-west-1):
   - `*.ateru.ng` wildcard certificate
   - Must be in same region as ALB (eu-west-1)
   - Used for HTTPS listener on ALB

2. **CloudFront Certificate** (us-east-1):
   - `*.ateru.ng` wildcard certificate
   - **Must be in us-east-1** (CloudFront requirement)
   - Used for custom domain on CloudFront

### ACM Certificate Creation

```bash
# ALB Certificate (eu-west-1)
aws acm request-certificate \
  --domain-name "*.ateru.ng" \
  --subject-alternative-names "ateru.ng" \
  --validation-method DNS \
  --region eu-west-1

# CloudFront Certificate (us-east-1)
aws acm request-certificate \
  --domain-name "*.ateru.ng" \
  --subject-alternative-names "ateru.ng" \
  --validation-method DNS \
  --region us-east-1
```

### DNS Validation
Add CNAME records to Squarespace DNS (provided by ACM):
```
_acmvalidation.ateru.ng CNAME _abc123.acm-validations.aws
```

## Health Checks

All services use same health check configuration:
- **Path**: `/health`
- **Protocol**: HTTP
- **Interval**: 30 seconds
- **Timeout**: 5 seconds
- **Healthy Threshold**: 2 consecutive successes
- **Unhealthy Threshold**: 3 consecutive failures
- **Expected Response**: 200 OK

### Service Health Endpoints

Each service must implement:
```go
// Go services
func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
```

```typescript
// Node.js Provider API
app.get('/health', (req, res) => {
  res.json({ status: 'healthy' });
});
```

## Session Stickiness

### Configuration
- **Type**: Load Balancer Cookie (AWSALB)
- **Duration**: 86,400 seconds (24 hours)
- **Services**: All (API, GraphQL, SSE, Provider API)

### Why Stickiness?
- **SSE**: Critical for Server-Sent Events (long-lived connections)
- **API/GraphQL**: Improves cache locality, reduces cold starts
- **Provider API**: Maintains session state

## Cache Strategy

### CloudFront (Frontend Assets)

**Production**:
- **Default TTL**: 24 hours (86,400s)
- **Max TTL**: 1 year (31,536,000s)
- **Min TTL**: 0 (allows immediate invalidation)
- **Compression**: Gzip + Brotli enabled

**Dev/Staging**:
- **Default TTL**: 5 minutes (300s)
- **Max TTL**: 1 hour (3600s)
- **Rationale**: Faster iteration, immediate updates

### Cache Invalidation (CI/CD)

```bash
# Invalidate all files
aws cloudfront create-invalidation \
  --distribution-id $DISTRIBUTION_ID \
  --paths "/*"

# Invalidate specific files
aws cloudfront create-invalidation \
  --distribution-id $DISTRIBUTION_ID \
  --paths "/index.html" "/assets/*"
```

**Cost**: First 1,000 invalidations/month free, then $0.005 per path

## Load Balancer Configuration

### Timeouts
- **Idle Timeout**: 60 seconds (matches default ECS container timeout)
- **Deregistration Delay**: 30 seconds (standard), 300 seconds (SSE)

### Cross-Zone Load Balancing
- **Enabled**: Yes (distributes traffic evenly across all AZs)

### Deletion Protection
- **Prod**: Enabled (prevents accidental deletion)
- **Dev/Staging**: Disabled (allows easy cleanup)

### HTTP/2
- **Enabled**: Yes (faster multiplexed connections)

## Security

### ALB Security
- **Security Group**: Only allows inbound 80/443 from internet (0.0.0.0/0)
- **Outbound**: Only to ECS service security groups
- **TLS Policy**: `ELBSecurityPolicy-TLS13-1-2-2021-06` (TLS 1.3 + 1.2)
- **No direct access** to ECS tasks from internet

### CloudFront Security
- **S3 Bucket**: Private (no public access)
- **Access Method**: Origin Access Identity (OAI) only
- **Viewer Protocol**: HTTPS required (redirect from HTTP)
- **TLS Version**: TLSv1.2_2021 minimum

### Bucket Policy (S3)
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowCloudFrontOAI",
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::cloudfront:user/CloudFront Origin Access Identity E123ABC"
      },
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::ohi-prod-frontend/*"
    }
  ]
}
```

## Cost Estimation (Monthly)

### Application Load Balancer
- **Load Balancer Hours**: 730 hours × $0.0225/hour = **$16.43**
- **LCU (Load Balancer Capacity Units)**:
  - New connections/sec: ~50 × $0.008 = $0.40
  - Active connections: ~1,000 × $0.008 = $8.00
  - Processed bytes: ~10 GB/hr × $0.008 = $58.40
  - Rule evaluations: Included
- **Subtotal**: ~**$83/month** per environment

### CloudFront
- **Data Transfer Out** (to internet):
  - First 10 TB: $0.085/GB
  - Assume 1 TB/month: 1,000 GB × $0.085 = **$85**
- **HTTPS Requests**:
  - Assume 50M requests/month: 50M × $0.01/10,000 = **$50**
- **Invalidations**: First 1,000 free = **$0**
- **Subtotal**: ~**$135/month** per environment

### S3 Storage (Frontend Assets)
- **Storage**: ~500 MB × $0.023/GB = **$0.01**
- **PUT Requests**: ~100 × $0.005/1,000 = **$0.00**
- **GET Requests**: Covered by CloudFront
- **Subtotal**: **Negligible**

### Total
- **Prod**: ALB ($83) + CloudFront ($135) + S3 ($0.01) = **$218/month**
- **Staging**: ALB ($83) + CloudFront ($40) + S3 ($0.01) = **$123/month** (lower traffic)
- **Dev**: ALB ($83) + CloudFront ($15) + S3 ($0.01) = **$98/month** (minimal traffic)
- **Grand Total**: **$439/month** for ALB + CloudFront across all environments

## Integration with ECS

### ECS Service Configuration

Update ECS services to use ALB target groups:

```typescript
import { createAlbInfrastructure } from './compute/alb';
import { createEcsInfrastructure } from './compute/ecs';

// Create ALB first
const albOutputs = createAlbInfrastructure(albConfig);

// Pass target group ARNs to ECS
const ecsConfig: EcsConfig = {
  // ... other config
  albTargetGroupArns: albOutputs.targetGroupArns,
};

const ecsOutputs = createEcsInfrastructure(ecsConfig);
```

### Health Check Grace Period
ECS services with ALB integration have 60-second grace period:
- Allows container startup without marking unhealthy
- Prevents premature task termination
- Configured in [src/compute/ecs.ts](../src/compute/ecs.ts)

## Frontend Deployment

### Build and Upload to S3

```bash
# Build frontend (Vite + React)
cd Frontend
npm run build

# Upload to S3
aws s3 sync dist/ s3://ohi-prod-frontend/ \
  --delete \
  --cache-control "public,max-age=31536000,immutable" \
  --exclude "index.html"

# Upload index.html separately (no cache)
aws s3 cp dist/index.html s3://ohi-prod-frontend/index.html \
  --cache-control "public,max-age=0,must-revalidate"

# Invalidate CloudFront cache
aws cloudfront create-invalidation \
  --distribution-id $DISTRIBUTION_ID \
  --paths "/*"
```

### Cache Headers Strategy
- **Assets** (JS, CSS, images): `max-age=31536000,immutable` (1 year)
- **index.html**: `max-age=0,must-revalidate` (always fresh)
- **Service Worker**: `max-age=0` (if using PWA)

## Monitoring and Logging

### ALB Metrics (CloudWatch)
- `TargetResponseTime` - Latency
- `HTTPCode_Target_2XX_Count` - Success rate
- `HTTPCode_Target_4XX_Count` - Client errors
- `HTTPCode_Target_5XX_Count` - Server errors
- `HealthyHostCount` - Available targets
- `UnHealthyHostCount` - Failed health checks
- `RequestCount` - Total requests

### CloudFront Metrics
- `Requests` - Total requests
- `BytesDownloaded` - Data transfer
- `4xxErrorRate` - Client errors
- `5xxErrorRate` - Server errors
- `CacheHitRate` - Cache efficiency

### ALB Access Logs (Prod Only)
- **Format**: S3 bucket with prefixes
- **Location**: `s3://ohi-logs/alb-logs/prod/`
- **Fields**: Timestamp, client IP, target IP, request, response, processing time
- **Cost**: S3 storage only (~$0.023/GB/month)

## Next Steps

1. ✅ ALB module created
2. ✅ CloudFront module created
3. ⏳ Create ACM certificates (manual step)
4. ⏳ Update ECS module to use ALB target groups
5. ⏳ Create Route 53 hosted zones (Phase 7)
6. ⏳ Create DNS records (A/CNAME for ALB and CloudFront)
7. ⏳ Test end-to-end: Browser → CloudFront → S3
8. ⏳ Test end-to-end: Browser → ALB → ECS → Database

## Known Limitations

1. **ACM Certificates Not Automated**: Must be manually created and validated
   - Pulumi can request certificates but can't complete DNS validation
   - Validation records must be added to Squarespace DNS manually

2. **No WAF Integration**: Web Application Firewall not configured
   - Consider adding AWS WAF for production
   - Protects against SQL injection, XSS, DDoS

3. **No Rate Limiting**: ALB doesn't have native rate limiting
   - Consider AWS WAF rate-based rules
   - Or implement in application layer

4. **No Geoblocking**: CloudFront restriction set to "none"
   - Could restrict to specific countries if needed
   - `geoRestriction.restrictionType = "whitelist"`

5. **No Custom Error Pages**: CloudFront uses default S3 error pages
   - Could create custom 404.html, 500.html
   - Update CloudFront error responses

## References

- [AWS ALB Documentation](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/)
- [AWS CloudFront Documentation](https://docs.aws.amazon.com/cloudfront/)
- [Pulumi AWS ALB](https://www.pulumi.com/registry/packages/aws/api-docs/lb/loadbalancer/)
- [Pulumi AWS CloudFront](https://www.pulumi.com/registry/packages/aws/api-docs/cloudfront/distribution/)
- [TLS 1.3 Policy](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/create-https-listener.html#describe-ssl-policies)
