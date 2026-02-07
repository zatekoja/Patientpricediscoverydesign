# Infrastructure Setup - Quick Start Guide

This guide will help you deploy the Patient Price Discovery application to Google Cloud Platform in under 30 minutes.

## Prerequisites Checklist

- [ ] GCP account with billing enabled
- [ ] Domain `ohealth-ng.com` purchased
- [ ] Terraform installed (v1.0+)
- [ ] gcloud CLI installed
- [ ] Docker installed
- [ ] Git installed

## Step-by-Step Setup

### 1. Authenticate with GCP (5 minutes)

```bash
# Login to GCP
gcloud auth login

# Set project
gcloud config set project open-health-index-dev

# Enable Application Default Credentials
gcloud auth application-default login
```

### 2. Configure Terraform Variables (3 minutes)

```bash
cd terraform/environments/dev
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars` with your API keys:
```hcl
google_maps_api_key = "YOUR_GOOGLE_MAPS_KEY"
typesense_api_key   = "YOUR_TYPESENSE_KEY"
openai_api_key      = "YOUR_OPENAI_KEY"
postgres_password   = "SECURE_PASSWORD"
```

### 3. Deploy Infrastructure (20-30 minutes)

From the repository root:
```bash
./scripts/deploy.sh dev
```

Type `yes` when prompted.

### 4. Configure DNS (Manual step - 2 minutes)

After deployment, you'll see output like:
```
dns_nameservers = [
  "ns-cloud-a1.googledomains.com",
  "ns-cloud-a2.googledomains.com",
  "ns-cloud-a3.googledomains.com",
  "ns-cloud-a4.googledomains.com",
]
```

**Action Required:**
1. Go to your domain registrar (where you bought ohealth-ng.com)
2. Update nameservers with the values from Terraform output
3. Wait for DNS propagation (15 minutes - 48 hours, usually < 2 hours)

### 5. Build and Deploy Applications (10-15 minutes)

Set environment variables:
```bash
export GCP_PROJECT_ID=open-health-index-dev
export ENVIRONMENT=dev
export GOOGLE_MAPS_API_KEY=your-key
```

Build and push Docker images:
```bash
./scripts/build-and-push.sh
```

### 6. Verify Deployment (2 minutes)

Check SSL certificate status:
```bash
gcloud compute ssl-certificates describe dev-ppd-ssl-cert \
  --global \
  --project open-health-index-dev
```

Test endpoints:
```bash
# Frontend
curl -I https://dev.ohealth-ng.com

# API
curl https://dev.api.ohealth-ng.com/health
```

Visit in browser:
- Frontend: https://dev.ohealth-ng.com
- API: https://dev.api.ohealth-ng.com

## Estimated Costs

Monthly costs for dev environment: **$450-750**

| Service | Cost/Month |
|---------|------------|
| Cloud Run | $50-150 |
| Cloud SQL (PostgreSQL) | $200-300 |
| Memorystore (Redis) | $150-200 |
| Load Balancer | $20-50 |
| Networking | $30-50 |
| DNS | $1-5 |

## Next Steps

- [ ] Set up monitoring alerts
- [ ] Configure CI/CD with GitHub Actions
- [ ] Set up backup strategy
- [ ] Create staging environment
- [ ] Configure autoscaling policies
- [ ] Set up cost alerts

## Troubleshooting

### SSL Certificate Not Active
Wait 15-60 minutes after DNS propagation. Check status:
```bash
gcloud compute ssl-certificates describe dev-ppd-ssl-cert --global
```

### Services Won't Start
Check Cloud Run logs:
```bash
gcloud run services logs read dev-ppd-api --region us-central1
```

### DNS Not Resolving
Verify DNS propagation:
```bash
dig dev.ohealth-ng.com
nslookup dev.api.ohealth-ng.com
```

Or use: https://www.whatsmydns.net/

## Support

For detailed documentation, see:
- `docs/GCP_DEPLOYMENT.md` - Full deployment guide
- `terraform/README.md` - Terraform documentation

## Cleanup

To remove all infrastructure:
```bash
cd terraform/environments/dev
terraform destroy
```

**Warning:** This deletes all data including databases. Back up first!

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Internet / DNS                            │
│              (ohealth-ng.com nameservers)                    │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
         ┌──────────────────────────────┐
         │   Cloud Load Balancer        │
         │   (SSL/TLS Termination)      │
         └──────────────────────────────┘
                        │
          ┌─────────────┴─────────────┐
          │                           │
          ▼                           ▼
┌─────────────────┐         ┌─────────────────┐
│ dev.ohealth-ng  │         │ dev.api.        │
│    .com         │         │ ohealth-ng.com  │
│                 │         │                 │
│  Frontend       │         │  API Services   │
│  (Cloud Run)    │         │  (Cloud Run)    │
│                 │         │  • REST API     │
│                 │         │  • GraphQL      │
│                 │         │  • SSE          │
└─────────────────┘         └─────────────────┘
                                      │
                    ┌─────────────────┴─────────────────┐
                    │          VPC Network               │
                    │                                    │
                    ▼                                    ▼
          ┌─────────────────┐              ┌─────────────────┐
          │  Cloud SQL      │              │  Memorystore    │
          │  (PostgreSQL)   │              │  (Redis)        │
          │  • HA Enabled   │              │  • HA Enabled   │
          │  • Backups      │              │  • 5GB          │
          └─────────────────┘              └─────────────────┘
```

## Summary

You now have a production-ready infrastructure on GCP with:

✅ Scalable Cloud Run services  
✅ High-availability databases  
✅ Global load balancing with SSL  
✅ Custom domain with DNS  
✅ Secure networking with VPC  
✅ Secret management  
✅ Monitoring and logging  
✅ Infrastructure as Code  

Your application is accessible at:
- **Frontend**: https://dev.ohealth-ng.com
- **API**: https://dev.api.ohealth-ng.com
