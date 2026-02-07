#!/bin/bash

# Deploy Infrastructure using Terraform
# This script deploys the infrastructure to GCP

set -e

# Configuration
ENVIRONMENT="${1:-dev}"
TERRAFORM_DIR="terraform/environments/${ENVIRONMENT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Deploying Infrastructure ===${NC}"
echo "Environment: $ENVIRONMENT"
echo "Terraform Directory: $TERRAFORM_DIR"

# Check if terraform directory exists
if [ ! -d "$TERRAFORM_DIR" ]; then
  echo -e "${RED}Error: Terraform directory not found: $TERRAFORM_DIR${NC}"
  exit 1
fi

# Navigate to terraform directory
cd "$TERRAFORM_DIR"

# Initialize Terraform
echo -e "\n${YELLOW}Initializing Terraform...${NC}"
terraform init

# Validate configuration
echo -e "\n${YELLOW}Validating Terraform configuration...${NC}"
terraform validate

# Plan deployment
echo -e "\n${YELLOW}Planning deployment...${NC}"
terraform plan -out=tfplan

# Ask for confirmation
echo -e "\n${YELLOW}Do you want to apply this plan? (yes/no)${NC}"
read -r CONFIRMATION

if [ "$CONFIRMATION" = "yes" ]; then
  echo -e "\n${YELLOW}Applying Terraform configuration...${NC}"
  terraform apply tfplan
  
  echo -e "\n${GREEN}=== Deployment complete! ===${NC}"
  
  # Display outputs
  echo -e "\n${YELLOW}Important Information:${NC}"
  terraform output
  
  echo -e "\n${YELLOW}Next Steps:${NC}"
  echo "1. In the parent DNS zone for your domain (e.g., ohealth-ng.com), add NS delegation records for the environment subdomain (e.g., dev.ohealth-ng.com) using the DNS nameservers shown above"
  echo "2. Wait for DNS propagation of the new NS records (can take up to 48 hours, usually < 2 hours)"
  echo "3. Wait for SSL certificate provisioning (can take up to 15 minutes after DNS propagation)"
  echo "4. Build and push Docker images using: ./scripts/build-and-push.sh"
  echo "5. Update Cloud Run services with new images"
else
  echo -e "${RED}Deployment cancelled${NC}"
  rm -f tfplan
  exit 1
fi
