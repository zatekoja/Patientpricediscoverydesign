# Open Health Initiative - AWS Infrastructure

Infrastructure as Code using Pulumi (TypeScript) for deploying the Open Health Initiative platform on AWS.

## Architecture

- **Primary Region:** eu-west-1 (Ireland)
- **Cloud Provider:** AWS
- **IaC Tool:** Pulumi (TypeScript)
- **Testing:** TDD approach with Pulumi testing framework

## Directory Structure

```
infrastructure/
├── pulumi/
│   ├── src/
│   │   ├── index.ts              # Main entry point
│   │   ├── config.ts             # Configuration management
│   │   ├── tagging.ts            # Tagging strategy
│   │   ├── networking/
│   │   │   ├── vpc.ts            # VPC, subnets, NAT gateways
│   │   │   ├── security-groups.ts # Security group definitions
│   │   │   └── dns.ts            # Route 53 hosted zones
│   │   ├── compute/
│   │   │   ├── ecs-cluster.ts    # ECS cluster
│   │   │   ├── ecs-services.ts   # ECS service definitions
│   │   │   └── ec2.ts            # EC2 instances (ClickHouse)
│   │   ├── database/
│   │   │   ├── rds.ts            # PostgreSQL RDS
│   │   │   ├── elasticache.ts   # Redis ElastiCache
│   │   │   └── external.ts      # MongoDB Atlas, Typesense Cloud configs
│   │   ├── storage/
│   │   │   ├── s3.ts             # S3 buckets
│   │   │   └── ecr.ts            # ECR repositories
│   │   ├── cdn/
│   │   │   ├── cloudfront.ts     # CloudFront distributions
│   │   │   └── acm.ts            # SSL certificates
│   │   ├── loadbalancing/
│   │   │   └── alb.ts            # Application Load Balancers
│   │   ├── secrets/
│   │   │   └── secrets-manager.ts # AWS Secrets Manager
│   │   └── monitoring/
│   │       ├── cloudwatch.ts     # CloudWatch logs/alarms
│   │       └── config.ts         # AWS Config rules
│   ├── tests/
│   │   ├── networking.test.ts    # VPC tests
│   │   ├── compute.test.ts       # ECS tests
│   │   ├── database.test.ts      # RDS/ElastiCache tests
│   │   ├── security.test.ts      # Security group tests
│   │   ├── tagging.test.ts       # Tag compliance tests
│   │   └── integration.test.ts   # End-to-end tests
│   ├── package.json
│   ├── tsconfig.json
│   ├── Pulumi.yaml
│   ├── Pulumi.dev.yaml
│   ├── Pulumi.staging.yaml
│   └── Pulumi.prod.yaml
└── scripts/
    ├── setup.sh                  # Initial setup script
    ├── deploy.sh                 # Deployment script
    └── destroy.sh                # Cleanup script
```

## Prerequisites

1. **AWS Account** with appropriate permissions
2. **Pulumi CLI** installed: `brew install pulumi` (macOS) or see https://www.pulumi.com/docs/install/
3. **Node.js 20+** installed
4. **AWS CLI** configured with credentials
5. **Pulumi Account** (free tier): https://app.pulumi.com/

## Setup

```bash
cd infrastructure/pulumi

# Install dependencies
npm install

# Login to Pulumi (one-time)
pulumi login

# Select stack (environment)
pulumi stack select dev

# Preview infrastructure changes
pulumi preview

# Deploy infrastructure
pulumi up
```

## Testing

We follow Test-Driven Development (TDD) for infrastructure:

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run specific test file
npm test -- networking.test.ts

# Run tests with coverage
npm run test:coverage
```

## Stacks (Environments)

- **dev**: Development environment (`Pulumi.dev.yaml`)
- **staging**: Pre-production environment (`Pulumi.staging.yaml`)
- **prod**: Production environment (`Pulumi.prod.yaml`)

## Configuration

Stack-specific configuration is in `Pulumi.<stack>.yaml`:

```yaml
config:
  aws:region: eu-west-1
  ohi:environment: dev
  ohi:vpcCidr: 10.0.0.0/16
  ohi:enableDeletionProtection: false
  ohi:rdsInstanceClass: db.t4g.small
  ohi:redisNodeType: cache.t4g.micro
```

## Deployment Workflow

### Development
```bash
pulumi stack select dev
pulumi up -y
```

### Staging
```bash
pulumi stack select staging
pulumi preview  # Review changes
pulumi up       # Requires manual confirmation
```

### Production
```bash
pulumi stack select prod
pulumi preview  # Review changes
pulumi up       # Requires manual confirmation + approval
```

## Cost Management

- All resources tagged with `Project`, `Environment`, `CostCenter`, `Service`
- AWS Config rules enforce tag compliance
- Cost Explorer budgets per environment
- Billing alarms at 80%, 100%, 120% thresholds

## Security

- All secrets stored in AWS Secrets Manager
- No hardcoded credentials in code
- IAM roles follow principle of least privilege
- Security groups deny all by default
- Encryption at rest for all databases
- TLS for all data in transit

## Monitoring

- CloudWatch Logs for all services
- CloudWatch Alarms for critical metrics
- VPC Flow Logs enabled
- AWS Config for compliance monitoring

## Troubleshooting

### Pulumi state issues
```bash
# Refresh state from AWS
pulumi refresh

# Export state
pulumi stack export > state-backup.json

# Import state
pulumi stack import < state-backup.json
```

### Resource conflicts
```bash
# Check for resource conflicts
pulumi preview --diff

# Force refresh
pulumi refresh --yes
```

### Clean up
```bash
# Destroy all resources (use with caution!)
pulumi destroy

# Remove stack
pulumi stack rm <stack-name>
```

## Documentation

- [Architecture Document](../V1_DEPLOYMENT_ARCHITECTURE.md)
- [Pulumi Documentation](https://www.pulumi.com/docs/)
- [AWS Best Practices](https://aws.amazon.com/architecture/well-architected/)

## Support

For issues or questions:
- Open an issue in GitHub
- Contact the platform team
- Check Pulumi community: https://slack.pulumi.com/
