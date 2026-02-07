# GCP Infrastructure Deployment Guide

## Overview

This guide provides comprehensive instructions for deploying the Patient Price Discovery application to Google Cloud Platform (GCP) using Infrastructure as Code (IaC) with Terraform.

## Architecture

The infrastructure consists of:

- **Cloud Run**: Serverless container platform for frontend and backend services
- **Cloud SQL (PostgreSQL)**: Managed PostgreSQL database
- **Memorystore (Redis)**: Managed Redis cache
- **Cloud DNS**: DNS management with hosted zones
- **Cloud Load Balancer**: Global HTTPS load balancer with SSL
- **VPC**: Virtual Private Cloud with private networking
- **Container Registry**: Docker image storage

## Domain Configuration

- **Domain**: ohealth-ng.com
- **Frontend**: dev.ohealth-ng.com
- **API**: dev.api.ohealth-ng.com

## Prerequisites

1. **Google Cloud Account**: Active GCP account with billing enabled
2. **GCP Project**: Project ID: `open-health-index-dev`
3. **Domain**: ohealth-ng.com purchased and accessible
4. **Local Tools**:
   - Terraform >= 1.0
   - gcloud CLI
   - Docker
   - git

## Setup Instructions

### Step 1: Install Required Tools

#### Install Terraform
```bash
# macOS
brew install terraform

# Linux
wget https://releases.hashicorp.com/terraform/1.7.0/terraform_1.7.0_linux_amd64.zip
unzip terraform_1.7.0_linux_amd64.zip
sudo mv terraform /usr/local/bin/
```

#### Install gcloud CLI
```bash
# macOS
brew install google-cloud-sdk

# Linux
curl https://sdk.cloud.google.com | bash
exec -l $SHELL
```

#### Authenticate with GCP
```bash
# Login to GCP
gcloud auth login

# Set project
gcloud config set project open-health-index-dev

# Enable Application Default Credentials
gcloud auth application-default login
```

### Step 2: Prepare Configuration

1. **Copy terraform variables template**:
```bash
cd terraform/environments/dev
cp terraform.tfvars.example terraform.tfvars
```

2. **Edit terraform.tfvars** with your values:
```hcl
project_id  = "open-health-index-dev"
region      = "us-central1"
environment = "dev"
domain_name = "ohealth-ng.com"

# Add your API keys
google_maps_api_key = "your-google-maps-api-key"
typesense_api_key   = "your-typesense-api-key"
openai_api_key      = "your-openai-api-key"
postgres_password   = "your-secure-postgres-password"
```

### Step 3: Deploy Infrastructure

Run the deployment script:
```bash
./scripts/deploy.sh dev
```

This will:
1. Initialize Terraform
2. Validate configuration
3. Create a deployment plan
4. Ask for confirmation
5. Deploy all infrastructure

Expected deployment time: **20-30 minutes**

### Step 4: Configure DNS

After deployment, you'll receive DNS nameservers. Configure your domain registrar (where you purchased ohealth-ng.com) with these nameservers.

### Step 5: Build and Push Docker Images

```bash
# Set environment variables
export GCP_PROJECT_ID=open-health-index-dev
export ENVIRONMENT=dev
export GOOGLE_MAPS_API_KEY=your-api-key

# Build and push images
./scripts/build-and-push.sh
```

### Step 6: Verify Deployment

```bash
# Test frontend
curl -I https://dev.ohealth-ng.com

# Test API
curl https://dev.api.ohealth-ng.com/health
```

## Infrastructure Components

### Cloud Run Services

| Service | Description | URL |
|---------|-------------|-----|
| Frontend | React application | dev.ohealth-ng.com |
| API | REST API | dev.api.ohealth-ng.com |
| GraphQL | GraphQL API | dev.api.ohealth-ng.com/graphql |
| SSE | Server-Sent Events | dev.api.ohealth-ng.com/sse |

For complete documentation, see the full guide in this file.
