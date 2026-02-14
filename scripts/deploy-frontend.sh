#!/bin/bash
# deploy-frontend.sh - Deploy frontend to S3 and invalidate CloudFront
#
# Usage: ./scripts/deploy-frontend.sh <environment> [distribution-id]
# Example: ./scripts/deploy-frontend.sh prod E1234ABCD5678

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
AWS_REGION="${AWS_REGION:-eu-west-1}"
BUILD_DIR="${BUILD_DIR:-dist}"

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
Usage: $0 <environment> [distribution-id]

Deploy frontend static assets to S3 and invalidate CloudFront cache.

Arguments:
  environment       Target environment (dev, staging, prod)
  distribution-id   CloudFront distribution ID (optional, will be looked up if not provided)

Environment Variables:
  AWS_REGION        AWS region (default: eu-west-1)
  BUILD_DIR         Build output directory (default: dist)
  DRY_RUN           Set to 'true' to preview without deploying
  SKIP_BUILD        Set to 'true' to skip npm build step
  SKIP_INVALIDATION Set to 'true' to skip CloudFront invalidation

Examples:
  # Deploy to prod with auto-lookup of distribution ID
  $0 prod

  # Deploy to staging with specific distribution ID
  $0 staging E1234ABCD5678

  # Dry run (preview only)
  DRY_RUN=true $0 dev

  # Deploy without rebuilding (use existing dist/)
  SKIP_BUILD=true $0 prod
EOF
    exit 1
}

# Validate arguments
if [ $# -lt 1 ]; then
    log_error "Missing required arguments"
    usage
fi

ENVIRONMENT=$1
DISTRIBUTION_ID=${2:-""}

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

# S3 and CloudFront configuration
BUCKET_NAME="ohi-${ENVIRONMENT}-frontend"

log_info "Starting frontend deployment..."
log_info "Environment: $ENVIRONMENT"
log_info "S3 Bucket: $BUCKET_NAME"
log_info "Build Directory: $BUILD_DIR"
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

# Build frontend (unless skipped)
if [ "${SKIP_BUILD:-false}" != "true" ]; then
    log_info "Building frontend..."
    cd "$PROJECT_ROOT"
    
    # Check if node_modules exists
    if [ ! -d "node_modules" ]; then
        log_info "Installing dependencies..."
        npm ci
    fi
    
    # Set environment-specific variables
    if [ "$ENVIRONMENT" = "prod" ]; then
        export VITE_API_URL="https://api.ohealth-ng.com"
        export VITE_GRAPHQL_URL="https://graphql.ohealth-ng.com"
        export VITE_SSE_URL="https://sse.ohealth-ng.com"
    else
        export VITE_API_URL="https://api.${ENVIRONMENT}.ohealth-ng.com"
        export VITE_GRAPHQL_URL="https://graphql.${ENVIRONMENT}.ohealth-ng.com"
        export VITE_SSE_URL="https://sse.${ENVIRONMENT}.ohealth-ng.com"
    fi
    export VITE_ENVIRONMENT="$ENVIRONMENT"
    
    log_info "Building with:"
    log_info "  VITE_API_URL=$VITE_API_URL"
    log_info "  VITE_GRAPHQL_URL=$VITE_GRAPHQL_URL"
    log_info "  VITE_SSE_URL=$VITE_SSE_URL"
    log_info "  VITE_ENVIRONMENT=$VITE_ENVIRONMENT"
    
    npm run build
    log_success "Frontend built successfully"
else
    log_warning "Skipping build (SKIP_BUILD=true)"
    cd "$PROJECT_ROOT"
fi
echo ""

# Verify build directory exists
if [ ! -d "$BUILD_DIR" ]; then
    log_error "Build directory not found: $BUILD_DIR"
    log_info "Run without SKIP_BUILD or check your build configuration"
    exit 1
fi

# Check if S3 bucket exists
log_info "Checking S3 bucket..."
if ! aws s3 ls "s3://$BUCKET_NAME" --region "$AWS_REGION" &> /dev/null; then
    log_error "S3 bucket not found: $BUCKET_NAME"
    log_info "Make sure infrastructure is deployed first (Pulumi)"
    exit 1
fi
log_success "S3 bucket found"
echo ""

# Dry run check
if [ "${DRY_RUN:-false}" = "true" ]; then
    log_warning "DRY RUN: Would sync files from $BUILD_DIR to s3://$BUCKET_NAME"
    log_info "Files to upload:"
    find "$BUILD_DIR" -type f | head -20
    FILE_COUNT=$(find "$BUILD_DIR" -type f | wc -l)
    log_info "Total files: $FILE_COUNT"
    exit 0
fi

# Upload files to S3
log_info "Uploading files to S3..."
log_info "This may take a few minutes..."

# Sync with specific cache-control headers
aws s3 sync "$BUILD_DIR/" "s3://$BUCKET_NAME/" \
    --region "$AWS_REGION" \
    --delete \
    --cache-control "public,max-age=31536000,immutable" \
    --exclude "*.html" \
    --exclude "*.json"

# Upload HTML files with shorter cache (for SPA routing)
aws s3 sync "$BUILD_DIR/" "s3://$BUCKET_NAME/" \
    --region "$AWS_REGION" \
    --cache-control "public,max-age=0,must-revalidate" \
    --exclude "*" \
    --include "*.html" \
    --include "*.json"

log_success "Files uploaded to S3"
echo ""

# Get or lookup CloudFront distribution ID
if [ -z "$DISTRIBUTION_ID" ]; then
    log_info "Looking up CloudFront distribution..."
    DISTRIBUTION_ID=$(aws cloudfront list-distributions \
        --region us-east-1 \
        --query "DistributionList.Items[?Comment=='OHI ${ENVIRONMENT} frontend distribution'].Id" \
        --output text)
    
    if [ -z "$DISTRIBUTION_ID" ] || [ "$DISTRIBUTION_ID" = "None" ]; then
        log_warning "CloudFront distribution not found"
        log_info "Trying alternative lookup by origin..."
        DISTRIBUTION_ID=$(aws cloudfront list-distributions \
            --region us-east-1 \
            --query "DistributionList.Items[?Origins.Items[?DomainName==\`${BUCKET_NAME}.s3.${AWS_REGION}.amazonaws.com\`]].Id" \
            --output text)
    fi
    
    if [ -z "$DISTRIBUTION_ID" ] || [ "$DISTRIBUTION_ID" = "None" ]; then
        log_error "Could not find CloudFront distribution for $ENVIRONMENT"
        log_info "Provide distribution ID manually or check infrastructure deployment"
        log_warning "Files uploaded to S3 but CloudFront not invalidated"
        exit 1
    fi
fi

log_success "CloudFront Distribution: $DISTRIBUTION_ID"
echo ""

# Skip invalidation if requested
if [ "${SKIP_INVALIDATION:-false}" = "true" ]; then
    log_warning "Skipping CloudFront invalidation (SKIP_INVALIDATION=true)"
    log_info "Remember to invalidate manually if needed:"
    log_info "  aws cloudfront create-invalidation --distribution-id $DISTRIBUTION_ID --paths '/*'"
    exit 0
fi

# Create CloudFront invalidation
log_info "Creating CloudFront invalidation..."
INVALIDATION_ID=$(aws cloudfront create-invalidation \
    --distribution-id "$DISTRIBUTION_ID" \
    --paths "/*" \
    --region us-east-1 \
    --query 'Invalidation.Id' \
    --output text)

if [ -z "$INVALIDATION_ID" ]; then
    log_error "Failed to create CloudFront invalidation"
    exit 1
fi

log_success "CloudFront invalidation created: $INVALIDATION_ID"
log_info "Waiting for invalidation to complete (this may take a few minutes)..."

# Wait for invalidation to complete
ELAPSED=0
MAX_WAIT=600  # 10 minutes

while [ $ELAPSED -lt $MAX_WAIT ]; do
    STATUS=$(aws cloudfront get-invalidation \
        --distribution-id "$DISTRIBUTION_ID" \
        --id "$INVALIDATION_ID" \
        --region us-east-1 \
        --query 'Invalidation.Status' \
        --output text)
    
    if [ "$STATUS" = "Completed" ]; then
        log_success "Invalidation completed! 🎉"
        break
    fi
    
    log_info "Invalidation status: $STATUS (elapsed: ${ELAPSED}s)"
    sleep 10
    ELAPSED=$((ELAPSED + 10))
done

if [ $ELAPSED -ge $MAX_WAIT ]; then
    log_warning "Invalidation timeout (still in progress)"
    log_info "Check status with:"
    log_info "  aws cloudfront get-invalidation --distribution-id $DISTRIBUTION_ID --id $INVALIDATION_ID"
fi

echo ""
echo "===== Deployment Summary ====="
echo "Environment: $ENVIRONMENT"
echo "S3 Bucket: s3://$BUCKET_NAME"
echo "CloudFront Distribution: $DISTRIBUTION_ID"
if [ "$ENVIRONMENT" = "prod" ]; then
    echo "Frontend URL: https://ohealth-ng.com"
else
    echo "Frontend URL: https://${ENVIRONMENT}.ohealth-ng.com"
fi
echo "=============================="
echo ""
log_success "Frontend deployment completed! 🚀"
