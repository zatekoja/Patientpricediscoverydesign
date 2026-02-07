#!/bin/bash

# Performance Testing Script
# Tests cache performance, response times, and compression effectiveness

set -e

echo "============================================"
echo "Performance Optimization Test Suite"
echo "============================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
GRAPHQL_URL="${GRAPHQL_URL:-http://localhost:8081/graphql}"

echo "Testing APIs:"
echo "  REST API: $API_URL"
echo "  GraphQL:  $GRAPHQL_URL"
echo ""

# Function to print section header
print_section() {
    echo ""
    echo "============================================"
    echo "$1"
    echo "============================================"
    echo ""
}

# Test 1: Check if services are running
print_section "1. Service Health Check"

echo -n "REST API: "
if curl -s -f "${API_URL}/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Running${NC}"
else
    echo -e "${RED}✗ Not running${NC}"
fi

echo -n "GraphQL API: "
if curl -s -f "${GRAPHQL_URL%/graphql}/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Running${NC}"
else
    echo -e "${RED}✗ Not running${NC}"
fi

# Test 2: Redis Cache Statistics
print_section "2. Redis Cache Statistics"

if docker ps | grep -q ppd_redis; then
    echo "Cache Hit/Miss Ratio:"
    docker exec ppd_redis redis-cli INFO stats | grep -E "keyspace_hits|keyspace_misses"

    echo ""
    echo "Cache Memory Usage:"
    docker exec ppd_redis redis-cli INFO memory | grep -E "used_memory_human|maxmemory_human"

    echo ""
    echo "Total Keys Cached:"
    docker exec ppd_redis redis-cli DBSIZE

    echo ""
    echo "Sample Cached Keys (facility:*):"
    docker exec ppd_redis redis-cli --scan --pattern "facility:*" | head -5
else
    echo -e "${YELLOW}⚠ Redis container not running${NC}"
fi

# Test 3: Database Connection Statistics
print_section "3. Database Connection Statistics"

if docker ps | grep -q ppd_postgres; then
    echo "Active Connections:"
    docker exec ppd_postgres psql -U postgres -d patient_price_discovery \
        -c "SELECT count(*) as active_connections FROM pg_stat_activity WHERE datname='patient_price_discovery';" \
        2>/dev/null || echo "Could not fetch connection stats"

    echo ""
    echo "Index Usage (Top 10):"
    docker exec ppd_postgres psql -U postgres -d patient_price_discovery \
        -c "SELECT schemaname, tablename, indexname, idx_scan FROM pg_stat_user_indexes WHERE schemaname = 'public' ORDER BY idx_scan DESC LIMIT 10;" \
        2>/dev/null || echo "Could not fetch index stats"
else
    echo -e "${YELLOW}⚠ PostgreSQL container not running${NC}"
fi

# Test 4: HTTP Compression Test
print_section "4. HTTP Compression Test"

echo "Testing /api/facilities endpoint..."
echo ""

# Without compression
echo -n "Response size WITHOUT gzip: "
SIZE_UNCOMPRESSED=$(curl -s -w '%{size_download}' -o /dev/null "${API_URL}/api/facilities")
echo "${SIZE_UNCOMPRESSED} bytes"

# With compression
echo -n "Response size WITH gzip:    "
SIZE_COMPRESSED=$(curl -s -H "Accept-Encoding: gzip" -w '%{size_download}' -o /dev/null "${API_URL}/api/facilities")
echo "${SIZE_COMPRESSED} bytes"

# Calculate compression ratio
if [ "$SIZE_UNCOMPRESSED" -gt 0 ]; then
    COMPRESSION_RATIO=$(echo "scale=1; (1 - $SIZE_COMPRESSED / $SIZE_UNCOMPRESSED) * 100" | bc)
    echo ""
    echo -e "${GREEN}Compression Ratio: ${COMPRESSION_RATIO}% reduction${NC}"
else
    echo ""
    echo -e "${YELLOW}Could not calculate compression ratio${NC}"
fi

# Test 5: ETag Support Test
print_section "5. ETag Support Test"

echo "Testing ETag functionality..."
echo ""

# First request to get ETag
RESPONSE=$(curl -s -i "${API_URL}/api/facilities" 2>/dev/null | head -20)
ETAG=$(echo "$RESPONSE" | grep -i "ETag:" | cut -d: -f2 | tr -d ' \r\n')

if [ -n "$ETAG" ]; then
    echo -e "${GREEN}✓ ETag header found: $ETAG${NC}"
    echo ""
    echo "Testing 304 Not Modified response..."

    # Second request with If-None-Match
    STATUS=$(curl -s -o /dev/null -w '%{http_code}' -H "If-None-Match: $ETAG" "${API_URL}/api/facilities")

    if [ "$STATUS" = "304" ]; then
        echo -e "${GREEN}✓ 304 Not Modified response received (ETag working!)${NC}"
    else
        echo -e "${YELLOW}⚠ Got status $STATUS instead of 304${NC}"
    fi
else
    echo -e "${YELLOW}⚠ No ETag header found${NC}"
fi

# Test 6: Response Time Test
print_section "6. Response Time Test (10 requests)"

echo "Testing /api/facilities endpoint..."
echo ""

TOTAL_TIME=0
for i in {1..10}; do
    TIME=$(curl -s -o /dev/null -w '%{time_total}' "${API_URL}/api/facilities")
    TIME_MS=$(echo "$TIME * 1000" | bc | cut -d. -f1)
    TOTAL_TIME=$(echo "$TOTAL_TIME + $TIME_MS" | bc)
    echo "Request $i: ${TIME_MS}ms"
done

AVG_TIME=$(echo "scale=0; $TOTAL_TIME / 10" | bc)
echo ""
echo -e "${GREEN}Average Response Time: ${AVG_TIME}ms${NC}"

if [ "$AVG_TIME" -lt 100 ]; then
    echo -e "${GREEN}✓ Excellent (< 100ms)${NC}"
elif [ "$AVG_TIME" -lt 200 ]; then
    echo -e "${YELLOW}✓ Good (< 200ms)${NC}"
else
    echo -e "${RED}⚠ Needs optimization (> 200ms)${NC}"
fi

# Test 7: Cache Effectiveness Test
print_section "7. Cache Effectiveness Test"

if docker ps | grep -q ppd_redis; then
    echo "Testing cache warming..."
    echo ""

    # Get initial stats
    INITIAL_HITS=$(docker exec ppd_redis redis-cli INFO stats | grep keyspace_hits | cut -d: -f2 | tr -d '\r')
    INITIAL_MISSES=$(docker exec ppd_redis redis-cli INFO stats | grep keyspace_misses | cut -d: -f2 | tr -d '\r')

    # Make 10 requests to the same endpoint
    echo "Making 10 requests to /api/facilities..."
    for i in {1..10}; do
        curl -s "${API_URL}/api/facilities" > /dev/null
    done

    # Get final stats
    FINAL_HITS=$(docker exec ppd_redis redis-cli INFO stats | grep keyspace_hits | cut -d: -f2 | tr -d '\r')
    FINAL_MISSES=$(docker exec ppd_redis redis-cli INFO stats | grep keyspace_misses | cut -d: -f2 | tr -d '\r')

    # Calculate hit ratio
    HITS_DIFF=$((FINAL_HITS - INITIAL_HITS))
    MISSES_DIFF=$((FINAL_MISSES - INITIAL_MISSES))
    TOTAL_DIFF=$((HITS_DIFF + MISSES_DIFF))

    if [ "$TOTAL_DIFF" -gt 0 ]; then
        HIT_RATIO=$(echo "scale=1; $HITS_DIFF * 100 / $TOTAL_DIFF" | bc)
        echo ""
        echo "Cache Hits: $HITS_DIFF"
        echo "Cache Misses: $MISSES_DIFF"
        echo -e "${GREEN}Cache Hit Ratio: ${HIT_RATIO}%${NC}"

        if [ "$(echo "$HIT_RATIO > 80" | bc)" -eq 1 ]; then
            echo -e "${GREEN}✓ Excellent (> 80%)${NC}"
        elif [ "$(echo "$HIT_RATIO > 50" | bc)" -eq 1 ]; then
            echo -e "${YELLOW}✓ Good (> 50%)${NC}"
        else
            echo -e "${RED}⚠ Needs improvement (< 50%)${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ Could not measure cache hit ratio${NC}"
    fi
else
    echo -e "${YELLOW}⚠ Redis container not running${NC}"
fi

# Test 8: GraphQL Performance
print_section "8. GraphQL Performance Test"

echo "Testing GraphQL query performance..."
echo ""

QUERY='{"query":"{ __typename }"}'
TIME=$(curl -s -o /dev/null -w '%{time_total}' -X POST \
    -H "Content-Type: application/json" \
    -d "$QUERY" \
    "${GRAPHQL_URL}")
TIME_MS=$(echo "$TIME * 1000" | bc | cut -d. -f1)

echo -e "Simple query response time: ${TIME_MS}ms"

if [ "$TIME_MS" -lt 50 ]; then
    echo -e "${GREEN}✓ Excellent (< 50ms)${NC}"
elif [ "$TIME_MS" -lt 100 ]; then
    echo -e "${YELLOW}✓ Good (< 100ms)${NC}"
else
    echo -e "${RED}⚠ Needs optimization (> 100ms)${NC}"
fi

# Summary
print_section "Summary"

echo "Performance optimization verification complete!"
echo ""
echo "Key Metrics:"
echo "  - Average response time: ${AVG_TIME}ms"
if [ -n "$COMPRESSION_RATIO" ]; then
    echo "  - Compression ratio: ${COMPRESSION_RATIO}%"
fi
if [ -n "$HIT_RATIO" ]; then
    echo "  - Cache hit ratio: ${HIT_RATIO}%"
fi
echo ""
echo "Next Steps:"
echo "  1. Monitor these metrics over time"
echo "  2. Adjust cache TTLs if needed"
echo "  3. Run load tests with 'ab' or 'k6'"
echo "  4. Check application logs for cache warming messages"
echo ""

