# Patient Price Discovery - GCP Infrastructure

This directory contains Terraform configuration for deploying the Patient Price Discovery application to Google Cloud Platform.

## Directory Structure

```
terraform/
├── main.tf                    # Root module
├── variables.tf               # Root variables
├── versions.tf                # Terraform version constraints
├── modules/                   # Reusable infrastructure modules
│   ├── dns/                   # Cloud DNS configuration
│   ├── networking/            # VPC and networking
│   ├── databases/             # Cloud SQL and Redis
│   ├── cloud-run/             # Cloud Run services
│   └── load-balancer/         # Global Load Balancer
└── environments/              # Environment-specific configurations
    └── dev/                   # Development environment
        ├── main.tf
        ├── variables.tf
        └── terraform.tfvars.example
```

## Quick Start

### Prerequisites

- Terraform >= 1.0
- gcloud CLI installed and authenticated
- GCP project: `open-health-index-dev`
- Domain: `ohealth-ng.com`

### Initial Setup

1. **Authenticate with GCP**:
```bash
gcloud auth login
gcloud auth application-default login
gcloud config set project open-health-index-dev
```

2. **Prepare configuration**:
```bash
cd environments/dev
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values
```

3. **Initialize Terraform**:
```bash
terraform init
```

4. **Deploy**:
```bash
terraform plan
terraform apply
```

## Modules

### DNS Module
- Creates Cloud DNS hosted zone
- Configures DNS records for dev.ohealth-ng.com and dev.api.ohealth-ng.com
- Enables DNSSEC

### Networking Module
- Creates VPC network
- Configures subnets
- Sets up VPC Access Connector for Cloud Run
- Configures firewall rules
- Sets up private service connection

### Databases Module
- Provisions Cloud SQL PostgreSQL (HA)
- Provisions Memorystore Redis (HA)
- Stores credentials in Secret Manager

### Cloud Run Module
- Deploys Frontend service
- Deploys API service
- Deploys GraphQL service
- Deploys SSE service
- Configures IAM and secrets

### Load Balancer Module
- Creates global HTTPS load balancer
- Provisions SSL certificates
- Configures URL mapping and routing
- Sets up HTTP to HTTPS redirect

## Environment Variables

Create a `terraform.tfvars` file in the environment directory:

```hcl
project_id          = "open-health-index-dev"
region              = "us-central1"
environment         = "dev"
domain_name         = "ohealth-ng.com"
google_maps_api_key = "your-api-key"
typesense_api_key   = "your-api-key"
openai_api_key      = "your-api-key"
postgres_password   = "your-secure-password"
```

## Outputs

After deployment, Terraform outputs:

- `dns_nameservers`: Configure these in your domain registrar
- `frontend_url`: Frontend application URL
- `api_url`: API endpoint URL
- `load_balancer_ip`: Load balancer IP address
- `postgres_connection_name`: Cloud SQL connection name
- `redis_host`: Redis instance host

## State Management

### Local State (Default)
State is stored locally in `terraform.tfstate` file.

### Remote State (Recommended for Production)
Uncomment the backend configuration in `main.tf`:

```hcl
backend "gcs" {
  bucket = "open-health-index-dev-tfstate"
  prefix = "terraform/state"
}
```

Then run:
```bash
terraform init -migrate-state
```

## Best Practices

1. **Never commit `terraform.tfvars`** - Contains sensitive data
2. **Use workspaces** for multiple environments
3. **Review plans** before applying
4. **Tag resources** appropriately
5. **Enable deletion protection** for databases
6. **Use remote state** for team collaboration

## Maintenance

### Update Infrastructure
```bash
terraform plan
terraform apply
```

### View Current State
```bash
terraform show
terraform state list
```

### Destroy Resources (CAUTION)
```bash
terraform destroy
```

## Troubleshooting

### Error: API not enabled
Enable required APIs:
```bash
gcloud services enable compute.googleapis.com
gcloud services enable run.googleapis.com
gcloud services enable sqladmin.googleapis.com
```

### Error: Permission denied
Ensure service account has required roles:
- Compute Admin
- Cloud Run Admin
- Cloud SQL Admin
- DNS Administrator
- Secret Manager Admin

### SSL Certificate Issues
Wait 15-60 minutes for certificate provisioning after DNS propagation.

## Cost Optimization

- Use Cloud Run's auto-scaling
- Configure appropriate instance sizes
- Enable Cloud CDN for static content
- Use committed use discounts for databases
- Set up budget alerts

## Security

- All secrets stored in Secret Manager
- Databases accessible only via VPC
- SSL/TLS for all connections
- IAM least privilege access
- Firewall rules restrict traffic
- DNSSEC enabled

## Support

See main documentation: `docs/GCP_DEPLOYMENT.md`
