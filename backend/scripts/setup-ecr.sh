#!/bin/bash
set -e

REGION="${1:-eu-west-1}"
PROJECT_NAME="patient-price"

echo "Setting up ECR repositories in region: $REGION"

# List of service suffixes corresponding to Dockerfile naming conventions
# Dockerfile -> api
# Dockerfile.provider -> provider
# Dockerfile.graphql-server -> graphql-server
# Dockerfile.sse-server -> sse-server
# Dockerfile.indexer -> indexer
# blnk.Dockerfile -> blnk

repos=(
  "api"
  "provider"
  "graphql-server"
  "sse-server"
  "indexer"
  "blnk"
)

for repo in "${repos[@]}"; do
  repo_name="${PROJECT_NAME}-${repo}"
  echo "Checking repository: $repo_name..."
  
  if aws ecr describe-repositories --repository-names "$repo_name" --region "$REGION" >/dev/null 2>&1; then
    echo "Repository $repo_name already exists."
  else
    echo "Creating repository $repo_name..."
    aws ecr create-repository \
      --repository-name "$repo_name" \
      --region "$REGION" \
      --image-scanning-configuration scanOnPush=true \
      --encryption-configuration encryptionType=AES256 \
      --tags Key=Project,Value=patient-price-discovery Key=ManagedBy,Value=script
    echo "Repository $repo_name created."
  fi
done

echo "ECR setup complete."
