#!/bin/bash
# deploy-ecs.sh - Deploy services to AWS ECS with zero-downtime
#
# Usage: ./scripts/deploy-ecs.sh <environment> <service> <image-tag>
# Example: ./scripts/deploy-ecs.sh prod api abc123f

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AWS_REGION="${AWS_REGION:-eu-west-1}"
MAX_WAIT_TIME=600  # 10 minutes
CHECK_INTERVAL=10   # Check every 10 seconds

# Function to print colored output
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to show usage
usage() {
    cat << EOF
Usage: $0 <environment> <service> <image-tag>

Deploy a service to AWS ECS with zero-downtime rolling update.

Arguments:
  environment   Target environment (dev, staging, prod)
  service       Service name (api, graphql, sse, provider-api, reindexer, blnk-api, blnk-worker)
  image-tag     Docker image tag to deploy (e.g., commit SHA or 'latest')

Environment Variables:
  AWS_REGION    AWS region (default: eu-west-1)
  DRY_RUN       Set to 'true' to preview without deploying
  SKIP_WAIT     Set to 'true' to skip waiting for deployment completion

Examples:
  # Deploy API service to prod with specific commit
  $0 prod api abc123f

  # Deploy GraphQL to staging with latest tag
  $0 staging graphql latest

  # Dry run (preview only)
  DRY_RUN=true $0 dev api test-tag
EOF
    exit 1
}

# Validate arguments
if [ $# -lt 3 ]; then
    log_error "Missing required arguments"
    usage
fi

ENVIRONMENT=$1
SERVICE=$2
IMAGE_TAG=$3

# Validate environment
case "$ENVIRONMENT" in
    dev|staging|prod)
        ;;
    *)
        log_error "Invalid environment: $ENVIRONMENT"
        log_info "Valid environments: dev, staging, prod"
        exit 1
        ;;
esac

# Validate service
VALID_SERVICES=(api graphql sse provider-api reindexer blnk-api blnk-worker)
if [[ ! " ${VALID_SERVICES[@]} " =~ " ${SERVICE} " ]]; then
    log_error "Invalid service: $SERVICE"
    log_info "Valid services: ${VALID_SERVICES[*]}"
    exit 1
fi

# ECS configuration
CLUSTER_NAME="ohi-${ENVIRONMENT}"
SERVICE_NAME="ohi-${ENVIRONMENT}-${SERVICE}"
TASK_FAMILY="ohi-${ENVIRONMENT}-${SERVICE}"

log_info "Starting deployment..."
log_info "Environment: $ENVIRONMENT"
log_info "Service: $SERVICE"
log_info "Image Tag: $IMAGE_TAG"
log_info "Cluster: $CLUSTER_NAME"
log_info "ECS Service: $SERVICE_NAME"
log_info "AWS Region: $AWS_REGION"
echo ""

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    log_error "AWS CLI not found. Please install it first."
    exit 1
fi

# Check AWS credentials
log_info "Checking AWS credentials..."
if ! aws sts get-caller-identity &> /dev/null; then
    log_error "AWS credentials not configured or invalid"
    exit 1
fi
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
log_success "AWS credentials valid (Account: $ACCOUNT_ID)"
echo ""

# Construct ECR image URI
ECR_REGISTRY="${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
ECR_REPO="ohi-${ENVIRONMENT}-${SERVICE}"
IMAGE_URI="${ECR_REGISTRY}/${ECR_REPO}:${IMAGE_TAG}"

log_info "Image URI: $IMAGE_URI"
echo ""

# Check if image exists in ECR
log_info "Checking if image exists in ECR..."
if aws ecr describe-images \
    --repository-name "$ECR_REPO" \
    --image-ids imageTag="$IMAGE_TAG" \
    --region "$AWS_REGION" &> /dev/null; then
    log_success "Image found in ECR"
else
    log_error "Image not found in ECR: $IMAGE_URI"
    log_info "Make sure to build and push the image first:"
    log_info "  docker build -t $IMAGE_URI ."
    log_info "  aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $ECR_REGISTRY"
    log_info "  docker push $IMAGE_URI"
    exit 1
fi
echo ""

# Get current task definition
log_info "Fetching current task definition..."
CURRENT_TASK_DEF=$(aws ecs describe-task-definition \
    --task-definition "$TASK_FAMILY" \
    --region "$AWS_REGION" \
    --query 'taskDefinition' \
    --output json)

if [ -z "$CURRENT_TASK_DEF" ] || [ "$CURRENT_TASK_DEF" = "null" ]; then
    log_error "Task definition not found: $TASK_FAMILY"
    log_info "Make sure infrastructure is deployed first (Pulumi)"
    exit 1
fi
log_success "Current task definition retrieved"
echo ""

# Update container image in task definition
log_info "Creating new task definition revision..."
NEW_TASK_DEF=$(echo "$CURRENT_TASK_DEF" | jq --arg IMAGE "$IMAGE_URI" '
    .containerDefinitions[0].image = $IMAGE |
    del(.taskDefinitionArn, .revision, .status, .requiresAttributes, .compatibilities, .registeredAt, .registeredBy)
')

# Register new task definition
if [ "${DRY_RUN:-false}" = "true" ]; then
    log_warning "DRY RUN: Would register new task definition:"
    echo "$NEW_TASK_DEF" | jq '.containerDefinitions[0] | {name, image}'
    log_warning "DRY RUN: Would update ECS service: $SERVICE_NAME"
    exit 0
fi

NEW_TASK_ARN=$(aws ecs register-task-definition \
    --region "$AWS_REGION" \
    --cli-input-json "$NEW_TASK_DEF" \
    --query 'taskDefinition.taskDefinitionArn' \
    --output text)

if [ -z "$NEW_TASK_ARN" ]; then
    log_error "Failed to register new task definition"
    exit 1
fi
log_success "New task definition registered: $NEW_TASK_ARN"
echo ""

# Update ECS service with new task definition
log_info "Updating ECS service..."
UPDATE_RESULT=$(aws ecs update-service \
    --cluster "$CLUSTER_NAME" \
    --service "$SERVICE_NAME" \
    --task-definition "$NEW_TASK_ARN" \
    --force-new-deployment \
    --region "$AWS_REGION" \
    --query 'service.{serviceName:serviceName,taskDefinition:taskDefinition,desiredCount:desiredCount}' \
    --output json)

log_success "ECS service update initiated"
echo "$UPDATE_RESULT" | jq '.'
echo ""

# Wait for deployment to complete (unless skipped)
if [ "${SKIP_WAIT:-false}" = "true" ]; then
    log_warning "Skipping deployment wait (SKIP_WAIT=true)"
    log_info "You can check deployment status with:"
    log_info "  aws ecs describe-services --cluster $CLUSTER_NAME --services $SERVICE_NAME --region $AWS_REGION"
    exit 0
fi

log_info "Waiting for deployment to complete (max ${MAX_WAIT_TIME}s)..."
ELAPSED=0
DEPLOYMENT_ID=""

while [ $ELAPSED -lt $MAX_WAIT_TIME ]; do
    # Get service status
    SERVICE_STATUS=$(aws ecs describe-services \
        --cluster "$CLUSTER_NAME" \
        --services "$SERVICE_NAME" \
        --region "$AWS_REGION" \
        --query 'services[0]' \
        --output json)

    # Get deployment status
    RUNNING_COUNT=$(echo "$SERVICE_STATUS" | jq -r '.runningCount')
    DESIRED_COUNT=$(echo "$SERVICE_STATUS" | jq -r '.desiredCount')
    DEPLOYMENTS_COUNT=$(echo "$SERVICE_STATUS" | jq -r '.deployments | length')
    PRIMARY_DEPLOYMENT=$(echo "$SERVICE_STATUS" | jq -r '.deployments[] | select(.status == "PRIMARY")')
    ROLLOUT_STATE=$(echo "$PRIMARY_DEPLOYMENT" | jq -r '.rolloutState // "IN_PROGRESS"')

    log_info "Status: Running=$RUNNING_COUNT/$DESIRED_COUNT, Deployments=$DEPLOYMENTS_COUNT, Rollout=$ROLLOUT_STATE"

    # Check if deployment is complete
    if [ "$ROLLOUT_STATE" = "COMPLETED" ] && [ "$RUNNING_COUNT" = "$DESIRED_COUNT" ]; then
        log_success "Deployment completed successfully! 🎉"
        
        # Show deployment summary
        echo ""
        echo "===== Deployment Summary ====="
        echo "Service: $SERVICE_NAME"
        echo "Task Definition: $NEW_TASK_ARN"
        echo "Running Tasks: $RUNNING_COUNT"
        echo "Image: $IMAGE_URI"
        echo "=============================="
        exit 0
    fi

    # Check for failed deployment
    if [ "$ROLLOUT_STATE" = "FAILED" ]; then
        log_error "Deployment failed!"
        log_info "Checking events for details..."
        aws ecs describe-services \
            --cluster "$CLUSTER_NAME" \
            --services "$SERVICE_NAME" \
            --region "$AWS_REGION" \
            --query 'services[0].events[:5]' \
            --output table
        exit 1
    fi

    sleep $CHECK_INTERVAL
    ELAPSED=$((ELAPSED + CHECK_INTERVAL))
done

# Timeout reached
log_warning "Deployment timeout reached (${MAX_WAIT_TIME}s)"
log_info "Deployment may still be in progress. Check status with:"
log_info "  aws ecs describe-services --cluster $CLUSTER_NAME --services $SERVICE_NAME --region $AWS_REGION"
exit 1
