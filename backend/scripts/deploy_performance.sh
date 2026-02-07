#!/bin/bash

# Quick Start - Performance Optimization Implementation
# Run this script to deploy all performance improvements

set -e

echo "============================================"
echo "Performance Optimization Deployment"
echo "============================================"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Step 1: Rebuild services
echo "Step 1: Rebuilding services with performance optimizations..."
echo ""

docker-compose build api graphql

echo -e "${GREEN}✓ Services rebuilt successfully${NC}"
echo ""

# Step 2: Restart services
echo "Step 2: Restarting services..."
echo ""

docker-compose up -d api graphql

echo -e "${GREEN}✓ Services restarted${NC}"
echo ""

# Step 3: Wait for services to be ready
echo "Step 3: Waiting for services to be ready..."
echo ""

sleep 5

# Check API health
echo -n "Checking REST API health... "
if curl -s -f http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Running${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
fi

echo -n "Checking GraphQL API health... "
if curl -s -f http://localhost:8081/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Running${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
fi

echo ""

# Step 4: Check cache warming
echo "Step 4: Verifying cache warming..."
echo ""

echo "REST API logs (last 5 lines with 'cache'):"
docker-compose logs --tail=50 api | grep -i cache | tail -5 || echo "No cache logs yet"

echo ""
echo "GraphQL API logs (last 5 lines with 'cache'):"
docker-compose logs --tail=50 graphql | grep -i cache | tail -5 || echo "No cache logs yet"

echo ""

# Step 5: Verify Redis keys
echo "Step 5: Checking Redis cache..."
echo ""

if docker ps | grep -q ppd_redis; then
    KEYS=$(docker exec ppd_redis redis-cli DBSIZE 2>/dev/null | grep -oE '[0-9]+' || echo "0")
    echo "Total keys in cache: $KEYS"

    if [ "$KEYS" -gt 0 ]; then
        echo -e "${GREEN}✓ Cache is being populated${NC}"
        echo ""
        echo "Sample cached keys:"
        docker exec ppd_redis redis-cli --scan --pattern "facility:*" | head -5 || true
    else
        echo -e "${YELLOW}⚠ Cache is empty (will populate on first requests)${NC}"
    fi
else
    echo -e "${RED}✗ Redis not running${NC}"
fi

echo ""

# Step 6: Test compression
echo "Step 6: Testing HTTP compression..."
echo ""

SIZE_WITHOUT=$(curl -s -w '%{size_download}' -o /dev/null http://localhost:8080/api/facilities 2>/dev/null || echo "0")
SIZE_WITH=$(curl -s -H "Accept-Encoding: gzip" -w '%{size_download}' -o /dev/null http://localhost:8080/api/facilities 2>/dev/null || echo "0")

if [ "$SIZE_WITHOUT" -gt 0 ] && [ "$SIZE_WITH" -gt 0 ]; then
    echo "Response size without compression: ${SIZE_WITHOUT} bytes"
    echo "Response size with compression: ${SIZE_WITH} bytes"

    if [ "$SIZE_WITH" -lt "$SIZE_WITHOUT" ]; then
        RATIO=$(echo "scale=1; (1 - $SIZE_WITH / $SIZE_WITHOUT) * 100" | bc)
        echo -e "${GREEN}✓ Compression working: ${RATIO}% reduction${NC}"
    else
        echo -e "${YELLOW}⚠ Compression may not be working${NC}"
    fi
else
    echo -e "${YELLOW}⚠ Could not test compression (API may not be responding yet)${NC}"
fi

echo ""

# Step 7: Apply database indexes (optional)
echo "Step 7: Database indexes..."
echo ""

read -p "Apply performance indexes now? (y/n): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    ./scripts/apply_performance_indexes.sh
else
    echo "Skipped. You can run './scripts/apply_performance_indexes.sh' later"
fi

echo ""

# Summary
echo "============================================"
echo "Deployment Complete!"
echo "============================================"
echo ""
echo -e "${GREEN}✓ Performance optimizations deployed${NC}"
echo ""
echo "What was deployed:"
echo "  • Cached facility adapters (REST + GraphQL)"
echo "  • Cache warming service (5-minute refresh)"
echo "  • HTTP compression (gzip)"
echo "  • ETag support (304 responses)"
echo "  • Cache-Control headers"
echo ""
echo "Next steps:"
echo "  1. Wait 5 minutes for cache to warm up"
echo "  2. Run: ./scripts/test_performance.sh"
echo "  3. Monitor cache hit ratio"
echo "  4. Apply database indexes if not done"
echo ""
echo "Expected improvements:"
echo "  • 3-4x faster reads (after cache warm-up)"
echo "  • 80-90% smaller responses (compression)"
echo "  • 70-80%+ cache hit ratio"
echo ""
echo "Monitor with:"
echo "  docker-compose logs -f api graphql"
echo "  docker exec ppd_redis redis-cli INFO stats"
echo ""

