# Phase 7: DNS and Secrets Management

## Status: ✅ Complete

## Overview

Implemented DNS management via Route 53, SSL/TLS certificates via ACM, and consolidated secrets management via AWS Secrets Manager. This phase provides:
- **DNS routing** for custom domains
- **SSL/TLS certificates** for HTTPS
- **Secrets consolidation** for credentials
- **IAM policies** for secure access

## Files Created

### 1. Route 53 Module - [src/networking/route53.ts](../src/networking/route53.ts) (262 lines)

**Purpose**: DNS management for custom domains

**Key Features**:
- Hosted zone creation for staging/dev subdomains
- A records (alias) for ALB integration
- A records (alias) for CloudFront integration
- CNAME records for custom configurations
- TXT records for domain verification
- NS records for subdomain delegation
- Health checks for endpoint monitoring
- TTL optimization (5 min for dev, standard for prod)

**Functions Exported**:
- `getOrCreateHostedZone()` - Create or reference hosted zone
- `getExistingHostedZone()` - Get existing zone by name
- `createAlbAliasRecord()` - A record for ALB (evaluates health)
- `createCloudFrontAliasRecord()` - A record for CloudFront
- `createCnameRecord()` - CNAME for custom routing
- `createTxtRecord()` - TXT for verification
- `createNsRecord()` - NS for subdomain delegation
- `createRoute53Infrastructure()` - Complete DNS setup
- `createHealthCheck()` - Route 53 health checks
- `getDomainForEnvironment()` - Helper for domain mapping
- `getServiceSubdomain()` - Helper for service subdomains
- `validateDomainName()` - Domain format validation

**Outputs**:
```typescript
{
  hostedZoneId: pulumi.Output<string>;
  hostedZoneName: pulumi.Output<string>;
  nameServers: pulumi.Output<string[]>;
  recordNames: {
    frontend: pulumi.Output<string>;    // ateru.ng
    api: pulumi.Output<string>;         // api.ateru.ng
    graphql: pulumi.Output<string>;     // graphql.ateru.ng
    sse: pulumi.Output<string>;         // sse.ateru.ng
    provider: pulumi.Output<string>;    // provider.ateru.ng
  };
}
```

### 2. ACM Module - [src/security/acm.ts](../src/security/acm.ts) (281 lines)

**Purpose**: SSL/TLS certificate management

**Key Features**:
- Wildcard certificate requests (`*.ateru.ng`)
- DNS validation (automated via Route 53)
- Region-specific certificates:
  - `eu-west-1` for ALB
  - `us-east-1` for CloudFront (required)
- Certificate validation tracking
- Manual validation instructions
- SSM Parameter Store export

**Functions Exported**:
- `requestCertificate()` - Request ACM certificate
- `createValidationRecords()` - Create Route 53 validation records
- `createCertificateValidation()` - Wait for validation
- `requestWildcardCertificate()` - Request `*.domain` cert
- `getExistingCertificateArn()` - Get ARN from existing cert
- `createAcmInfrastructure()` - Create both ALB and CloudFront certs
- `getValidationInstructions()` - Manual DNS validation guide
- `isCertificateIssued()` - Check validation status
- `getCertificateDetails()` - Get cert metadata
- `validateCertificateConfig()` - Config validation
- `exportCertificateArnToSsm()` - Export to SSM for reference

**Outputs**:
```typescript
{
  albCertificateArn: pulumi.Output<string>;        // eu-west-1
  cloudfrontCertificateArn: pulumi.Output<string>; // us-east-1
}
```

### 3. Secrets Module - [src/security/secrets.ts](../src/security/secrets.ts) (316 lines)

**Purpose**: Consolidated secrets management

**Key Features**:
- Master secret with all credentials
- Individual secrets for each credential type
- JWT secret generation (64-character random)
- API keys management
- IAM policies for ECS task access
- KMS encryption (default AWS managed key)
- Recovery window (30 days prod, 7 days dev/staging)
- Rotation schedules (prod only, 90 days)
- SSM Parameter Store export

**Functions Exported**:
- `createMasterSecret()` - Consolidated secret container
- `createMasterSecretVersion()` - JSON with all credentials
- `createDatabasePasswordSecret()` - Individual DB password
- `createRedisAuthTokenSecret()` - Individual Redis tokens
- `generateJwtSecret()` - Random 64-char JWT secret
- `createJwtSecret()` - JWT secret in Secrets Manager
- `createSecretsAccessPolicy()` - IAM policy for ECS tasks
- `attachSecretsPolicyToRole()` - Attach to ECS execution role
- `createSecretsInfrastructure()` - Complete secrets setup
- `getSecretValue()` - Retrieve secret value
- `getSecretArn()` - Get secret ARN
- `createSecretRotation()` - Automatic rotation (prod)
- `exportSecretArnToSsm()` - Export to SSM
- `createApiKeysSecret()` - API keys secret
- `validateSecretsConfig()` - Config validation

**Outputs**:
```typescript
{
  masterSecretArn: pulumi.Output<string>;
  masterSecretName: pulumi.Output<string>;
  databasePasswordArn: pulumi.Output<string>;
  redisAuthTokenArn: pulumi.Output<string>;
  blnkRedisAuthTokenArn: pulumi.Output<string>;
  jwtSecretArn?: pulumi.Output<string>;
}
```

## DNS Architecture

### Domain Structure

**Production (`ateru.ng`)**:
```
ateru.ng                    → CloudFront → S3 (Frontend)
api.ateru.ng                → ALB → ECS (API Service)
graphql.ateru.ng            → ALB → ECS (GraphQL Service)
sse.ateru.ng                → ALB → ECS (SSE Service)
provider.ateru.ng           → ALB → ECS (Provider API)
```

**Staging (`staging.ateru.ng`)**:
```
staging.ateru.ng            → CloudFront → S3 (Frontend)
api.staging.ateru.ng        → ALB → ECS (API Service)
graphql.staging.ateru.ng    → ALB → ECS (GraphQL Service)
sse.staging.ateru.ng        → ALB → ECS (SSE Service)
provider.staging.ateru.ng   → ALB → ECS (Provider API)
```

**Development (`dev.ateru.ng`)**:
```
dev.ateru.ng                → CloudFront → S3 (Frontend)
api.dev.ateru.ng            → ALB → ECS (API Service)
graphql.dev.ateru.ng        → ALB → ECS (GraphQL Service)
sse.dev.ateru.ng            → ALB → ECS (SSE Service)
provider.dev.ateru.ng       → ALB → ECS (Provider API)
```

### Route 53 Hosted Zones

1. **Production Zone** (ateru.ng):
   - Managed manually or via Pulumi
   - Contains A records for all prod subdomains
   - NS records for staging/dev delegation

2. **Staging Zone** (staging.ateru.ng):
   - Created by Pulumi
   - Delegated from prod via NS records
   - Contains all staging subdomains

3. **Development Zone** (dev.ateru.ng):
   - Created by Pulumi
   - Delegated from prod via NS records
   - Contains all dev subdomains

### Squarespace DNS Delegation

Add these NS records in Squarespace DNS:

```
# Main domain (if not already delegated)
ateru.ng               NS   ns-123.awsdns-12.com
                       NS   ns-456.awsdns-34.org
                       NS   ns-789.awsdns-56.net
                       NS   ns-012.awsdns-78.co.uk

# Staging subdomain delegation
staging.ateru.ng       NS   ns-xxx.awsdns-xx.com
                       NS   ns-yyy.awsdns-yy.org
                       NS   ns-zzz.awsdns-zz.net
                       NS   ns-www.awsdns-ww.co.uk

# Dev subdomain delegation
dev.ateru.ng           NS   ns-aaa.awsdns-aa.com
                       NS   ns-bbb.awsdns-bb.org
                       NS   ns-ccc.awsdns-cc.net
                       NS   ns-ddd.awsdns-dd.co.uk
```

## SSL/TLS Certificates

### Certificate Requirements

1. **ALB Certificate** (eu-west-1):
   - Domain: `*.ateru.ng`
   - Subject Alternative Names: `ateru.ng`
   - Region: `eu-west-1` (same as ALB)
   - Usage: ALB HTTPS listeners

2. **CloudFront Certificate** (us-east-1):
   - Domain: `*.ateru.ng`
   - Subject Alternative Names: `ateru.ng`
   - Region: `us-east-1` (CloudFront requirement)
   - Usage: CloudFront custom domains

### Certificate Creation Process

```bash
# 1. Request certificates via Pulumi
pulumi up

# 2. Get DNS validation records
pulumi stack output albCertValidationRecords
pulumi stack output cloudfrontCertValidationRecords

# 3. Add CNAME records to Route 53 (automated) or Squarespace (manual)
# Example:
_abc123.ateru.ng CNAME _xyz789.acm-validations.aws.

# 4. Wait for validation (5-30 minutes)
aws acm describe-certificate --certificate-arn arn:aws:acm:eu-west-1:123:certificate/abc --region eu-west-1

# 5. Verify status = ISSUED
aws acm list-certificates --region eu-west-1
aws acm list-certificates --region us-east-1
```

### Manual Validation (if needed)

If Pulumi doesn't automatically create validation records:

```typescript
// Get validation instructions
const cert = requestWildcardCertificate('prod', 'ateru.ng', 'eu-west-1');
const instructions = getValidationInstructions(cert);

// Output will show:
// Domain 1: *.ateru.ng
//   Record Type: CNAME
//   Record Name: _abc123.ateru.ng
//   Record Value: _xyz789.acm-validations.aws.
```

## Secrets Management

### Master Secret Structure

The master secret (`ohi-{environment}-master`) contains:

```json
{
  "DATABASE_PASSWORD": "secure-password-123",
  "REDIS_AUTH_TOKEN": "secure-token-456",
  "BLNK_REDIS_AUTH_TOKEN": "secure-token-789",
  "JWT_SECRET": "64-character-random-string",
  "API_KEY_PROVIDER1": "provider1-key",
  "API_KEY_PROVIDER2": "provider2-key"
}
```

### Individual Secrets

1. **Database Password** (`ohi-{env}-db-password`):
   - Plain text password
   - Used by RDS connection string
   - Rotation: 90 days (prod)

2. **Redis Auth Token** (`ohi-{env}-redis-token`):
   - Plain text token
   - Used by Redis connection
   - Rotation: 90 days (prod)

3. **Blnk Redis Auth Token** (`ohi-{env}-blnk-redis-token`):
   - Plain text token
   - Used by Blnk Redis connection
   - Rotation: 90 days (prod)

4. **JWT Secret** (`ohi-{env}-jwt-secret`):
   - 64-character random string
   - Used for JWT signing/verification
   - Rotation: Manual (requires coordination)

5. **API Keys** (`ohi-{env}-api-keys`):
   - JSON object with multiple keys
   - Used for external service integration
   - Rotation: Manual

### ECS Task Access

ECS tasks access secrets via environment variables:

```yaml
# Task definition
containerDefinitions:
  - name: api
    secrets:
      - name: DATABASE_PASSWORD
        valueFrom: arn:aws:secretsmanager:eu-west-1:123:secret:ohi-prod-db-password
      - name: REDIS_AUTH_TOKEN
        valueFrom: arn:aws:secretsmanager:eu-west-1:123:secret:ohi-prod-redis-token
```

IAM policy allows:
- `secretsmanager:GetSecretValue`
- `secretsmanager:DescribeSecret`
- `kms:Decrypt` (via Secrets Manager KMS key)

### Secrets Rotation (Prod)

Automatic rotation enabled for prod:
- **Frequency**: 90 days
- **Lambda**: Custom rotation function (to be created)
- **Process**:
  1. Lambda generates new password
  2. Updates database user password
  3. Updates secret in Secrets Manager
  4. Tests connection
  5. Marks secret as valid

## Health Checks

Route 53 health checks monitor endpoints:

```typescript
createHealthCheck(config, 'api.ateru.ng', '/health');
```

**Configuration**:
- **Protocol**: HTTPS
- **Port**: 443
- **Path**: `/health`
- **Interval**: 30 seconds
- **Failure Threshold**: 3 consecutive failures
- **Measure Latency**: Yes

**Actions on Failure**:
- CloudWatch alarm triggered
- SNS notification sent
- Failover to backup region (future)

## Cost Estimation (Monthly)

### Route 53
- **Hosted Zones**: 3 zones × $0.50 = **$1.50**
- **Queries**: First 1 billion = $0.40/million
  - Assume 10M queries/month: 10 × $0.40 = **$4.00**
- **Health Checks**: 3 checks × $0.50 = **$1.50**
- **Subtotal**: **$7/month**

### ACM Certificates
- **Public Certificates**: **$0** (free)
- **Private CA**: Not used
- **Subtotal**: **$0/month**

### Secrets Manager
- **Secrets**: 6 secrets × $0.40 = **$2.40**
- **API Calls**: 10,000 × $0.05/10,000 = **$0.05**
- **Subtotal**: **$2.45/month**

### Total Phase 7 Cost
**$9.45/month** across all environments

## Integration Example

Complete infrastructure setup:

```typescript
import { createRoute53Infrastructure } from './networking/route53';
import { createAcmInfrastructure } from './security/acm';
import { createSecretsInfrastructure } from './security/secrets';
import { createAlbInfrastructure } from './compute/alb';
import { createCloudFrontInfrastructure } from './compute/cloudfront';

// 1. Create ACM certificates
const certs = createAcmInfrastructure('prod', 'ateru.ng');

// 2. Create ALB with certificate
const albConfig: AlbConfig = {
  environment: 'prod',
  vpcId: vpcOutputs.vpcId,
  publicSubnetIds: vpcOutputs.publicSubnetIds,
  albSecurityGroupId: sgOutputs.albSecurityGroupId,
  certificateArn: certs.albCertificateArn,
};
const albOutputs = createAlbInfrastructure(albConfig);

// 3. Create CloudFront with certificate
const cloudfrontConfig: CloudFrontConfig = {
  environment: 'prod',
  s3BucketDomainName: bucketOutputs.domainName,
  s3BucketArn: bucketOutputs.arn,
  certificateArn: certs.cloudfrontCertificateArn,
  domainAliases: ['ateru.ng'],
};
const cloudfrontOutputs = createCloudFrontInfrastructure(cloudfrontConfig);

// 4. Create Route 53 DNS records
const route53Config: Route53Config = {
  environment: 'prod',
  domain: 'ateru.ng',
  createHostedZone: false, // Use existing
  albDnsName: albOutputs.albDnsName,
  albZoneId: albOutputs.albZoneId,
  cloudfrontDnsName: cloudfrontOutputs.distributionDomainName,
  cloudfrontZoneId: cloudfrontOutputs.distributionHostedZoneId,
};
const route53Outputs = createRoute53Infrastructure(route53Config);

// 5. Create secrets
const secretsConfig: SecretsConfig = {
  environment: 'prod',
  databasePassword: dbPasswordOutput,
  redisAuthToken: redisTokenOutput,
  blnkRedisAuthToken: blnkRedisTokenOutput,
  jwtSecret: generateJwtSecret('prod').result,
};
const secretsOutputs = createSecretsInfrastructure(secretsConfig);

// 6. Export outputs
export const nameServers = route53Outputs.nameServers;
export const albCertArn = certs.albCertificateArn;
export const cloudfrontCertArn = certs.cloudfrontCertificateArn;
export const masterSecretArn = secretsOutputs.masterSecretArn;
```

## Testing

### DNS Resolution Testing

```bash
# Test DNS resolution
dig ateru.ng
dig api.ateru.ng
dig graphql.ateru.ng

# Test with specific nameserver
dig @ns-123.awsdns-12.com ateru.ng

# Check TTL values
dig ateru.ng +noall +answer +ttlid
```

### SSL/TLS Testing

```bash
# Test certificate validity
openssl s_client -connect api.ateru.ng:443 -servername api.ateru.ng

# Check certificate details
curl -vvI https://api.ateru.ng

# Verify certificate chain
ssl-cert-check -c api.ateru.ng
```

### Secrets Testing

```bash
# Get secret value
aws secretsmanager get-secret-value \
  --secret-id ohi-prod-master \
  --region eu-west-1 \
  --query 'SecretString' \
  --output text | jq

# List all secrets
aws secretsmanager list-secrets \
  --region eu-west-1 \
  --query 'SecretList[?contains(Name, `ohi-prod`)].Name'

# Test ECS task access (from within container)
aws secretsmanager get-secret-value \
  --secret-id ohi-prod-db-password \
  --region eu-west-1
```

## Troubleshooting

### DNS Issues

**Problem**: Domain not resolving
- **Check**: NS records in Squarespace match Route 53 nameservers
- **Command**: `dig NS ateru.ng`
- **Fix**: Update Squarespace DNS with correct NS records

**Problem**: Stale DNS cache
- **Check**: TTL values
- **Command**: `dig ateru.ng +ttlid`
- **Fix**: Wait for TTL to expire or flush DNS cache

### Certificate Issues

**Problem**: Certificate validation pending
- **Check**: CNAME validation records exist
- **Command**: `aws acm describe-certificate --certificate-arn ARN --region eu-west-1`
- **Fix**: Add missing CNAME records to Route 53

**Problem**: CloudFront certificate not found
- **Check**: Certificate in us-east-1 region
- **Command**: `aws acm list-certificates --region us-east-1`
- **Fix**: Request certificate in us-east-1

### Secrets Issues

**Problem**: ECS task can't access secret
- **Check**: IAM policy attached to task execution role
- **Command**: `aws iam list-attached-role-policies --role-name ohi-prod-ecs-task-execution`
- **Fix**: Attach secrets access policy

**Problem**: Secret not found
- **Check**: Secret exists in correct region
- **Command**: `aws secretsmanager list-secrets --region eu-west-1`
- **Fix**: Create secret or update ARN

## Next Steps

1. ✅ Route 53 module created
2. ✅ ACM module created
3. ✅ Secrets module created
4. ⏳ Request ACM certificates (manual DNS validation)
5. ⏳ Update Squarespace DNS with NS records
6. ⏳ Update ECS module to reference secrets
7. ⏳ Create main Pulumi index file
8. ⏳ Deploy to dev environment for testing

## Security Best Practices

1. **Secrets Rotation**: Enable for prod, 90-day cycle
2. **KMS Encryption**: Use customer-managed keys for prod
3. **Least Privilege**: ECS tasks only access required secrets
4. **Audit Logging**: CloudTrail logs all secret access
5. **Recovery Window**: 30 days for prod, 7 days for dev/staging
6. **DNS Security**: DNSSEC for domain validation (future)
7. **Certificate Pinning**: Consider for mobile apps (future)

## References

- [Route 53 Documentation](https://docs.aws.amazon.com/route53/)
- [ACM Documentation](https://docs.aws.amazon.com/acm/)
- [Secrets Manager Documentation](https://docs.aws.amazon.com/secretsmanager/)
- [Pulumi Route 53](https://www.pulumi.com/registry/packages/aws/api-docs/route53/)
- [Pulumi ACM](https://www.pulumi.com/registry/packages/aws/api-docs/acm/)
- [Pulumi Secrets Manager](https://www.pulumi.com/registry/packages/aws/api-docs/secretsmanager/)
