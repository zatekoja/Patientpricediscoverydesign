# GCP Deployment Checklist

Use this checklist to ensure a smooth deployment to Google Cloud Platform.

## Pre-Deployment Checklist

### 1. GCP Account Setup
- [ ] GCP account created
- [ ] Billing enabled on account
- [ ] Project `open-health-index-dev` created
- [ ] You have Owner or Editor role on the project

### 2. Domain Setup
- [ ] Domain `ohealth-ng.com` purchased
- [ ] Access to domain registrar settings
- [ ] Able to modify nameservers

### 3. Local Tools Installed
- [ ] Terraform >= 1.0 installed
- [ ] gcloud CLI installed
- [ ] Docker installed and running
- [ ] Git installed

### 4. API Keys Ready
- [ ] Google Maps API key obtained
- [ ] Typesense API key generated (or will use default)
- [ ] OpenAI API key obtained (optional)
- [ ] Strong PostgreSQL password generated

### 5. Repository Cloned
- [ ] Repository cloned locally
- [ ] On the correct branch
- [ ] All files present

## Deployment Steps

### Phase 1: GCP Authentication (5 min)
```bash
# Authenticate
gcloud auth login

# Set project
gcloud config set project open-health-index-dev

# Enable Application Default Credentials
gcloud auth application-default login
```

**Verification:**
```bash
gcloud config get-value project
# Should output: open-health-index-dev
```

- [ ] GCP authentication completed
- [ ] Project set correctly

### Phase 2: Configure Terraform (5 min)

```bash
cd terraform/environments/dev
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars` with your values:
```hcl
project_id  = "open-health-index-dev"
region      = "us-central1"
environment = "dev"
domain_name = "ohealth-ng.com"

# Replace with your actual values
google_maps_api_key = "AIza..."
typesense_api_key   = "your-key"
openai_api_key      = "sk-..."
postgres_password   = "your-secure-password-32-chars"
```

**Verification:**
```bash
cat terraform.tfvars | grep -v "^#" | grep "="
# Should show all your configured values
```

- [ ] terraform.tfvars created
- [ ] All API keys configured
- [ ] Secure password set

### Phase 3: Deploy Infrastructure (20-30 min)

```bash
# From repository root
./scripts/deploy.sh dev
```

**During deployment:**
- Terraform will show a plan
- Review the resources to be created
- Type `yes` to confirm
- Wait for deployment (20-30 minutes)

**Expected output:**
```
Apply complete! Resources: XX added, 0 changed, 0 destroyed.

Outputs:
dns_nameservers = [...]
frontend_url = "https://dev.ohealth-ng.com"
api_url = "https://dev.api.ohealth-ng.com"
load_balancer_ip = "XX.XX.XX.XX"
```

- [ ] Infrastructure deployed successfully
- [ ] DNS nameservers noted
- [ ] Load balancer IP noted
- [ ] No errors in output

### Phase 4: Configure DNS (5 min + waiting time)

1. Copy DNS nameservers from Terraform output
2. **For subdomain delegation** (recommended approach):
   - Log in to your DNS management for the parent domain (ohealth-ng.com)
   - This may be at your domain registrar or your DNS provider
   - Navigate to DNS records for ohealth-ng.com
   - Create **NS records** for the subdomain (e.g., dev.ohealth-ng.com)
   - Point these NS records to the Google Cloud nameservers from Terraform output
   - Do **not** replace the registrar nameservers for ohealth-ng.com
3. Save changes

**Verification (wait 10-15 minutes):**
```bash
dig dev.ohealth-ng.com
nslookup dev.api.ohealth-ng.com
```

Or use: https://www.whatsmydns.net/

- [ ] Nameservers updated at registrar
- [ ] DNS propagation started
- [ ] Can verify DNS resolution

**Note:** DNS can take up to 48 hours but usually works in 1-2 hours.

### Phase 5: Build and Push Docker Images (15 min)

```bash
# Set environment variables
export GCP_PROJECT_ID=open-health-index-dev
export ENVIRONMENT=dev
export GOOGLE_MAPS_API_KEY=your-key

# Build and push
./scripts/build-and-push.sh
```

**Expected output:**
```
✅ All images built and pushed successfully!
Frontend: gcr.io/open-health-index-dev/dev-ppd-frontend:latest
API: gcr.io/open-health-index-dev/dev-ppd-api:latest
GraphQL: gcr.io/open-health-index-dev/dev-ppd-graphql:latest
SSE: gcr.io/open-health-index-dev/dev-ppd-sse:latest
```

- [ ] Frontend image built and pushed
- [ ] API image built and pushed
- [ ] GraphQL image built and pushed
- [ ] SSE image built and pushed
- [ ] No build errors

### Phase 6: Verify SSL Certificate (Wait 15-60 min)

Wait for SSL certificate to provision (happens automatically after DNS propagation).

```bash
gcloud compute ssl-certificates describe dev-ppd-ssl-cert \
  --global \
  --project open-health-index-dev
```

Look for `status: ACTIVE`

- [ ] SSL certificate status checked
- [ ] Certificate is ACTIVE
- [ ] All domains validated

### Phase 7: Test Deployment (5 min)

```bash
# Test Frontend
curl -I https://dev.ohealth-ng.com

# Test API
curl https://dev.api.ohealth-ng.com/health

# Test GraphQL
curl https://dev.api.ohealth-ng.com/graphql -H "Content-Type: application/json" -d '{"query":"{ __schema { types { name } } }"}'
```

**Browser tests:**
- Open https://dev.ohealth-ng.com - Should load frontend
- Open https://dev.api.ohealth-ng.com - Should show API response

- [ ] Frontend accessible via HTTPS
- [ ] API responding
- [ ] GraphQL responding
- [ ] No SSL warnings
- [ ] Application loads correctly

### Phase 8: Verify Services (5 min)

Check Cloud Run services:
```bash
gcloud run services list --platform managed --region us-central1
```

Check databases:
```bash
gcloud sql instances list
gcloud redis instances list --region us-central1
```

View logs:
```bash
gcloud run services logs read dev-ppd-api --region us-central1 --limit 50
```

- [ ] All Cloud Run services running
- [ ] PostgreSQL instance healthy
- [ ] Redis instance healthy
- [ ] No critical errors in logs

## Post-Deployment Tasks

### Monitoring Setup
- [ ] Configure uptime checks in Cloud Monitoring
- [ ] Set up alerting policies
- [ ] Create monitoring dashboard

### Security Review
- [ ] Verify firewall rules
- [ ] Check IAM permissions
- [ ] Review Secret Manager access
- [ ] Confirm SSL/TLS configuration

### Cost Management
- [ ] Set up billing alerts
- [ ] Configure budget notifications
- [ ] Review resource utilization

### Backup Strategy
- [ ] Verify automated database backups
- [ ] Test backup restoration process
- [ ] Document backup schedule

### CI/CD Setup
- [ ] Add GCP service account key to GitHub Secrets
- [ ] Test GitHub Actions workflow
- [ ] Configure branch protection rules

## Troubleshooting

### Infrastructure Deployment Failed

1. Check enabled APIs:
```bash
gcloud services list --enabled
```

2. Enable missing APIs:
```bash
gcloud services enable compute.googleapis.com run.googleapis.com sqladmin.googleapis.com
```

3. Check Terraform logs for specific errors

### DNS Not Resolving

1. Verify nameservers are updated at registrar
2. Wait for DNS propagation (up to 48 hours)
3. Use DNS checker: https://www.whatsmydns.net/
4. Check Cloud DNS records:
```bash
gcloud dns record-sets list --zone=dev-ohealth-ng-com
```

### SSL Certificate Not Provisioning

1. Ensure DNS is propagating
2. Wait 15-60 minutes after DNS propagation
3. Check certificate status:
```bash
gcloud compute ssl-certificates describe dev-ppd-ssl-cert --global
```

### Services Won't Start

1. Check service logs:
```bash
gcloud run services logs read SERVICE_NAME --region us-central1
```

2. Verify environment variables
3. Check VPC connector status
4. Verify database connectivity

### Build Failures

1. Check Docker daemon is running
2. Verify you're authenticated to GCR:
```bash
gcloud auth configure-docker
```

3. Check for syntax errors in Dockerfiles
4. Verify base images are accessible

## Rollback Procedure

If deployment fails and you need to rollback:

```bash
cd terraform/environments/dev

# Destroy all resources
terraform destroy

# Confirm with: yes
```

**Warning:** This will delete all data. Ensure backups exist first.

## Success Criteria

✅ All checklist items completed  
✅ Infrastructure deployed without errors  
✅ DNS resolving correctly  
✅ SSL certificates active  
✅ All services accessible via HTTPS  
✅ No critical errors in logs  
✅ Frontend loading and functional  
✅ API endpoints responding  

## Estimated Total Time

- **Minimum**: 60-90 minutes (if DNS propagates quickly)
- **Maximum**: 2-3 hours (including DNS propagation wait time)
- **Hands-on time**: ~45 minutes
- **Waiting time**: ~15-60 minutes (DNS + SSL)

## Support Resources

- [Infrastructure Setup Guide](INFRASTRUCTURE_SETUP.md)
- [GCP Deployment Documentation](docs/GCP_DEPLOYMENT.md)
- [Terraform Documentation](terraform/README.md)
- GCP Console: https://console.cloud.google.com
- Cloud Run: https://console.cloud.google.com/run
- Cloud SQL: https://console.cloud.google.com/sql

## Next Steps After Deployment

1. Set up monitoring and alerting
2. Configure autoscaling policies
3. Implement backup and disaster recovery
4. Set up staging environment
5. Configure CI/CD pipeline
6. Performance testing and optimization
7. Security hardening
8. Cost optimization

---

**Deployment Date**: _____________  
**Deployed By**: _____________  
**Environment**: dev  
**Status**: ☐ Success ☐ Failed ☐ In Progress
