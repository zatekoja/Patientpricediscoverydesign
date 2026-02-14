#!/bin/bash
# run-migrations.sh - Run database migrations via ECS RunTask
#
# Usage: ./scripts/run-migrations.sh <environment> [direction]
# Example: ./scripts/run-migrations.sh prod up

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
MAX_WAIT_TIME=300  # 5 minutes
CHECK_INTERVAL=5   # Check every 5 seconds

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
Usage: $0 <environment> [direction]

Run database migrations via AWS ECS RunTask.

Arguments:
  environment   Target environment (dev, staging, prod)
  direction     Migration direction: up (apply), down (rollback), status (check) (default: up)

Environment Variables:
  AWS_REGION    AWS region (default: eu-west-1)
  DRY_RUN       Set to 'true' to preview without running
  TIMEOUT       Max wait time in seconds (default: 300)

Examples:
  # Run migrations in prod
  $0 prod up

  # Check migration status in staging
  $0 staging status

  # Rollback last migration in dev
  $0 dev down

  # Dry run (preview only)
  DRY_RUN=true $0 prod up
EOF
    exit 1
}

# Validate arguments
if [ $# -lt 1 ]; then
    log_error "Missing required arguments"
    usage
fi

ENVIRONMENT=$1
DIRECTION=${2:-up}

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

# Validate direction
case "$DIRECTION" in
    up|down|status|force|version)
        ;;
    *)
        log_error "Invalid direction: $DIRECTION"
        log_info "Valid directions: up, down, status, force, version"
        exit 1
        ;;
esac

# ECS configuration
CLUSTER_NAME="ohi-${ENVIRONMENT}"
TASK_DEFINITION="ohi-${ENVIRONMENT}-api"  # Migrations run from API container
CONTAINER_NAME="api"

log_info "Starting database migration..."
log_info "Environment: $ENVIRONMENT"
log_info "Direction: $DIRECTION"
log_info "Cluster: $CLUSTER_NAME"
log_info "Task Definition: $TASK_DEFINITION"
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

# Confirm production migrations
if [ "$ENVIRONMENT" = "prod" ] && [ "$DIRECTION" != "status" ]; then
    log_warning "You are about to run migrations in PRODUCTION!"
    log_info "Migration direction: $DIRECTION"
    
    if [ "${FORCE_YES:-false}" != "true" ]; then
        read -p "Are you sure you want to continue? (yes/no): " -r
        echo
        if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
            log_info "Migration cancelled"
            exit 0
        fi
    fi
fi

# Get VPC configuration for task
log_info "Fetching VPC configuration..."
VPC_ID=$(aws ec2 describe-vpcs \
    --filters "Name=tag:Environment,Values=$ENVIRONMENT" \
    --query 'Vpcs[0].VpcId' \
    --output text \
    --region "$AWS_REGION")

if [ -z "$VPC_ID" ] || [ "$VPC_ID" = "None" ]; then
    log_error "VPC not found for environment: $ENVIRONMENT"
    log_info "Make sure infrastructure is deployed first"
    exit 1
fi

# Get private subnets
SUBNET_IDS=$(aws ec2 describe-subnets \
    --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Type,Values=private" \
    --query 'Subnets[*].SubnetId' \
    --output text \
    --region "$AWS_REGION")

if [ -z "$SUBNET_IDS" ]; then
    log_error "No private subnets found in VPC: $VPC_ID"
    exit 1
fi

# Convert space-separated to comma-separated
SUBNET_IDS=$(echo "$SUBNET_IDS" | tr ' ' ',')

# Get security group for API
SECURITY_GROUP_ID=$(aws ec2 describe-security-groups \
    --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Name,Values=ohi-${ENVIRONMENT}-api-sg" \
    --query 'SecurityGroups[0].GroupId' \
    --output text \
    --region "$AWS_REGION")

if [ -z "$SECURITY_GROUP_ID" ] || [ "$SECURITY_GROUP_ID" = "None" ]; then
    log_warning "API security group not found, trying ECS service security group..."
    SECURITY_GROUP_ID=$(aws ec2 describe-security-groups \
        --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Name,Values=ohi-${ENVIRONMENT}-ecs-services-sg" \
        --query 'SecurityGroups[0].GroupId' \
        --output text \
        --region "$AWS_REGION")
fi

if [ -z "$SECURITY_GROUP_ID" ] || [ "$SECURITY_GROUP_ID" = "None" ]; then
    log_error "Security group not found for environment: $ENVIRONMENT"
    exit 1
fi

log_success "VPC configuration retrieved"
log_info "VPC ID: $VPC_ID"
log_info "Subnets: $SUBNET_IDS"
log_info "Security Group: $SECURITY_GROUP_ID"
echo ""

# Build migration command based on direction
# DATABASE_URL is constructed at runtime from the task definition environment variables
DB_URL_EXPR='postgres://${DATABASE_USER}:${DATABASE_PASSWORD}@${DATABASE_HOST}/${DATABASE_NAME}?sslmode=require'
case "$DIRECTION" in
    up)
        MIGRATION_CMD="./migrate -path /migrations -database ${DB_URL_EXPR} up"
        ;;
    down)
        MIGRATION_CMD="./migrate -path /migrations -database ${DB_URL_EXPR} down 1"
        ;;
    status)
        MIGRATION_CMD="./migrate -path /migrations -database ${DB_URL_EXPR} version"
        ;;
    force)
        log_error "Force requires a version number. Use: $0 $ENVIRONMENT force <version>"
        exit 1
        ;;
    version)
        MIGRATION_CMD="./migrate -path /migrations -database ${DB_URL_EXPR} version"
        ;;
esac

# Dry run check
if [ "${DRY_RUN:-false}" = "true" ]; then
    log_warning "DRY RUN: Would run ECS task with:"
    echo "  Cluster: $CLUSTER_NAME"
    echo "  Task Definition: $TASK_DEFINITION"
    echo "  Subnets: $SUBNET_IDS"
    echo "  Security Group: $SECURITY_GROUP_ID"
    echo "  Command: $MIGRATION_CMD"
    exit 0
fi

# Run migration task
log_info "Starting ECS task for migrations..."
TASK_ARN=$(aws ecs run-task \
    --cluster "$CLUSTER_NAME" \
    --task-definition "$TASK_DEFINITION" \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[$SUBNET_IDS],securityGroups=[$SECURITY_GROUP_ID],assignPublicIp=DISABLED}" \
    --overrides "{
        \"containerOverrides\": [{
            \"name\": \"$CONTAINER_NAME\",
            \"command\": [\"sh\", \"-c\", \"$MIGRATION_CMD\"]
        }]
    }" \
    --region "$AWS_REGION" \
    --query 'tasks[0].taskArn' \
    --output text)

if [ -z "$TASK_ARN" ] || [ "$TASK_ARN" = "None" ]; then
    log_error "Failed to start ECS task"
    exit 1
fi

log_success "ECS task started: $TASK_ARN"
echo ""

# Wait for task to complete
log_info "Waiting for migration task to complete (max ${MAX_WAIT_TIME}s)..."
ELAPSED=0

while [ $ELAPSED -lt $MAX_WAIT_TIME ]; do
    TASK_STATUS=$(aws ecs describe-tasks \
        --cluster "$CLUSTER_NAME" \
        --tasks "$TASK_ARN" \
        --region "$AWS_REGION" \
        --query 'tasks[0].lastStatus' \
        --output text)
    
    log_info "Task status: $TASK_STATUS (elapsed: ${ELAPSED}s)"
    
    if [ "$TASK_STATUS" = "STOPPED" ]; then
        # Get exit code
        EXIT_CODE=$(aws ecs describe-tasks \
            --cluster "$CLUSTER_NAME" \
            --tasks "$TASK_ARN" \
            --region "$AWS_REGION" \
            --query 'tasks[0].containers[0].exitCode' \
            --output text)
        
        # Get stop reason
        STOP_REASON=$(aws ecs describe-tasks \
            --cluster "$CLUSTER_NAME" \
            --tasks "$TASK_ARN" \
            --region "$AWS_REGION" \
            --query 'tasks[0].stoppedReason' \
            --output text)
        
        echo ""
        if [ "$EXIT_CODE" = "0" ]; then
            log_success "Migration completed successfully! 🎉"
            
            # Try to fetch logs
            log_info "Fetching migration logs..."
            LOG_STREAM=$(aws ecs describe-tasks \
                --cluster "$CLUSTER_NAME" \
                --tasks "$TASK_ARN" \
                --region "$AWS_REGION" \
                --query 'tasks[0].containers[0].name' \
                --output text)
            
            TASK_ID=$(echo "$TASK_ARN" | rev | cut -d'/' -f1 | rev)
            LOG_GROUP="/ecs/ohi-${ENVIRONMENT}-${CONTAINER_NAME}"
            LOG_STREAM_NAME="ecs/${CONTAINER_NAME}/${TASK_ID}"
            
            if aws logs get-log-events \
                --log-group-name "$LOG_GROUP" \
                --log-stream-name "$LOG_STREAM_NAME" \
                --region "$AWS_REGION" \
                --query 'events[*].message' \
                --output text 2>/dev/null; then
                echo ""
            else
                log_warning "Could not fetch logs (they may not be available yet)"
                log_info "View logs manually:"
                log_info "  aws logs tail $LOG_GROUP --follow"
            fi
            
            exit 0
        else
            log_error "Migration failed with exit code: $EXIT_CODE"
            log_info "Stop reason: $STOP_REASON"
            
            # Try to fetch error logs
            log_info "Fetching error logs..."
            TASK_ID=$(echo "$TASK_ARN" | rev | cut -d'/' -f1 | rev)
            LOG_GROUP="/ecs/ohi-${ENVIRONMENT}-${CONTAINER_NAME}"
            LOG_STREAM_NAME="ecs/${CONTAINER_NAME}/${TASK_ID}"
            
            aws logs get-log-events \
                --log-group-name "$LOG_GROUP" \
                --log-stream-name "$LOG_STREAM_NAME" \
                --region "$AWS_REGION" \
                --query 'events[-20:].message' \
                --output text 2>/dev/null || log_warning "Could not fetch logs"
            
            exit 1
        fi
    fi
    
    sleep $CHECK_INTERVAL
    ELAPSED=$((ELAPSED + CHECK_INTERVAL))
done

# Timeout reached
log_error "Migration timeout reached (${MAX_WAIT_TIME}s)"
log_info "Task may still be running. Check status with:"
log_info "  aws ecs describe-tasks --cluster $CLUSTER_NAME --tasks $TASK_ARN --region $AWS_REGION"
exit 1
