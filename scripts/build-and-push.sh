#!/bin/bash

# Build and Push Docker Images to Google Container Registry
# This script builds all Docker images and pushes them to GCR

set -e

# Configuration
PROJECT_ID="${GCP_PROJECT_ID:-open-health-index-dev}"
REGION="${GCP_REGION:-us-central1}"
ENVIRONMENT="${ENVIRONMENT:-dev}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Building and Pushing Docker Images ===${NC}"
echo "Project ID: $PROJECT_ID"
echo "Region: $REGION"
echo "Environment: $ENVIRONMENT"

# Authenticate with GCP
echo -e "\n${YELLOW}Authenticating with GCP...${NC}"
gcloud auth configure-docker

# Build and push Frontend
echo -e "\n${YELLOW}Building Frontend...${NC}"
docker build -f Frontend/frontend.Dockerfile \
  --build-arg VITE_GOOGLE_MAPS_API_KEY="${GOOGLE_MAPS_API_KEY}" \
  -t gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-frontend:latest \
  -t gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-frontend:$(git rev-parse --short HEAD) \
  .

echo -e "${YELLOW}Pushing Frontend image...${NC}"
docker push gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-frontend:latest
docker push gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-frontend:$(git rev-parse --short HEAD)

# Build and push API
echo -e "\n${YELLOW}Building API...${NC}"
docker build -f backend/Dockerfile \
  -t gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-api:latest \
  -t gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-api:$(git rev-parse --short HEAD) \
  backend/

echo -e "${YELLOW}Pushing API image...${NC}"
docker push gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-api:latest
docker push gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-api:$(git rev-parse --short HEAD)

# Build and push GraphQL
echo -e "\n${YELLOW}Building GraphQL...${NC}"
docker build -f backend/Dockerfile.graphql-server \
  -t gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-graphql:latest \
  -t gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-graphql:$(git rev-parse --short HEAD) \
  backend/

echo -e "${YELLOW}Pushing GraphQL image...${NC}"
docker push gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-graphql:latest
docker push gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-graphql:$(git rev-parse --short HEAD)

# Build and push SSE
echo -e "\n${YELLOW}Building SSE...${NC}"
docker build -f backend/Dockerfile.sse-server \
  -t gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-sse:latest \
  -t gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-sse:$(git rev-parse --short HEAD) \
  backend/

echo -e "${YELLOW}Pushing SSE image...${NC}"
docker push gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-sse:latest
docker push gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-sse:$(git rev-parse --short HEAD)

echo -e "\n${GREEN}=== All images built and pushed successfully! ===${NC}"
echo "Frontend: gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-frontend:latest"
echo "API: gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-api:latest"
echo "GraphQL: gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-graphql:latest"
echo "SSE: gcr.io/${PROJECT_ID}/${ENVIRONMENT}-ppd-sse:latest"
