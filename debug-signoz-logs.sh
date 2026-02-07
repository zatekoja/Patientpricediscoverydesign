#!/bin/bash

# Debug script for SigNoz log collection
# This script helps verify that logs are properly flowing through the observability stack

set -e

echo "========================================"
echo "SigNoz Log Collection Debug Script"
echo "========================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
    fi
}

# Function to print info
print_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

echo "1. Checking if required containers are running..."
echo "------------------------------------------------"

# Check if containers are running
containers=("ppd_signoz_otel_collector" "ppd_clickhouse" "ppd_fluent_bit" "ppd_signoz_schema_migrator")
all_running=true

for container in "${containers[@]}"; do
    if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
        status=$(docker inspect -f '{{.State.Status}}' ${container})
        if [ "$status" = "running" ] || ([ "$container" = "ppd_signoz_schema_migrator" ] && [ "$status" = "exited" ]); then
            print_status 0 "${container} is ${status}"
        else
            print_status 1 "${container} is ${status}"
            all_running=false
        fi
    else
        print_status 1 "${container} is MISSING"
        all_running=false
    fi
done

if [ "$all_running" = false ]; then
    echo ""
    print_info "Some containers are not running. Start them with: docker-compose up -d"
    exit 1
fi

echo ""
echo "2. Checking OTEL Collector logs..."
echo "------------------------------------------------"
print_info "Last 20 lines of OTEL Collector logs:"
docker logs --tail 20 ppd_signoz_otel_collector 2>&1 | tail -20
echo ""

echo "3. Checking Fluent Bit logs..."
echo "------------------------------------------------"
print_info "Last 20 lines of Fluent Bit logs:"
docker logs --tail 20 ppd_fluent_bit 2>&1 | tail -20
echo ""

echo "4. Testing OTLP HTTP endpoint..."
echo "------------------------------------------------"
# Test if OTLP collector is accepting requests
if curl -s -X POST http://localhost:4318/v1/logs \
    -H "Content-Type: application/json" \
    -d '{"resourceLogs":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"test-service"}}]},"scopeLogs":[{"scope":{"name":"test"},"logRecords":[{"timeUnixNano":"1609459200000000000","severityNumber":9,"severityText":"INFO","body":{"stringValue":"Test log message"},"attributes":[{"key":"test","value":{"stringValue":"true"}}]}]}]}]}' \
    > /dev/null 2>&1; then
    print_status 0 "OTLP HTTP endpoint (4318) is accessible"
else
    print_status 1 "OTLP HTTP endpoint (4318) is NOT accessible"
fi

echo ""
echo "5. Testing OTLP gRPC endpoint..."
echo "------------------------------------------------"
if nc -z localhost 4317 2>/dev/null; then
    print_status 0 "OTLP gRPC endpoint (4317) is accessible"
else
    print_status 1 "OTLP gRPC endpoint (4317) is NOT accessible"
fi

echo ""
echo "6. Checking ClickHouse connectivity..."
echo "------------------------------------------------"
if docker exec ppd_signoz_otel_collector wget -q -O- http://clickhouse:8123/ping 2>/dev/null | grep -q "Ok"; then
    print_status 0 "ClickHouse is accessible from OTEL collector"
else
    print_status 1 "ClickHouse is NOT accessible from OTEL collector"
fi

echo ""
echo "7. Checking ClickHouse logs database..."
echo "------------------------------------------------"
# Check if logs database exists
if docker exec ppd_clickhouse clickhouse-client --user admin --password admin --query "SHOW DATABASES" 2>/dev/null | grep -q "signoz_logs"; then
    print_status 0 "signoz_logs database exists"

    # Check for tables
    print_info "Checking for log tables..."
    docker exec ppd_clickhouse clickhouse-client --user admin --password admin --query "SHOW TABLES FROM signoz_logs" 2>/dev/null || true
else
    print_status 1 "signoz_logs database does NOT exist"
    print_info "Creating signoz_logs database..."
    docker exec ppd_clickhouse clickhouse-client --user admin --password admin --query "CREATE DATABASE IF NOT EXISTS signoz_logs" 2>/dev/null || true
fi

echo ""
echo "8. Generating test logs..."
echo "------------------------------------------------"
# Generate some test logs from running services
print_info "Making test requests to generate logs..."

# Test API
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    print_status 0 "Generated logs from API service"
else
    print_status 1 "Could not reach API service"
fi

# Test GraphQL
if curl -s http://localhost:8081/health > /dev/null 2>&1; then
    print_status 0 "Generated logs from GraphQL service"
else
    print_status 1 "Could not reach GraphQL service"
fi

# Test Provider API
if curl -s http://localhost:3001/health > /dev/null 2>&1; then
    print_status 0 "Generated logs from Provider API service"
else
    print_status 1 "Could not reach Provider API service"
fi

echo ""
echo "9. Waiting for logs to be processed (10 seconds)..."
echo "------------------------------------------------"
sleep 10

echo ""
echo "10. Checking for recent logs in ClickHouse..."
echo "------------------------------------------------"
# Query recent logs
LOG_COUNT=$(docker exec ppd_clickhouse clickhouse-client --user admin --password admin --query "SELECT count() FROM signoz_logs.logs WHERE timestamp > now() - INTERVAL 5 MINUTE" 2>/dev/null || echo "0")
if [ "$LOG_COUNT" -gt 0 ]; then
    print_status 0 "Found $LOG_COUNT log entries in the last 5 minutes"

    print_info "Sample of recent logs:"
    docker exec ppd_clickhouse clickhouse-client --user admin --password admin --query "SELECT timestamp, severity_text, body FROM signoz_logs.logs ORDER BY timestamp DESC LIMIT 5 FORMAT Pretty" 2>/dev/null || true
else
    print_status 1 "No logs found in the last 5 minutes"
fi

echo ""
echo "11. Checking application log output..."
echo "------------------------------------------------"
print_info "Recent logs from API service:"
docker logs --tail 5 ppd_api 2>&1 | tail -5
echo ""

print_info "Recent logs from Provider API service:"
docker logs --tail 5 ppd_provider_api 2>&1 | tail -5
echo ""

echo "12. Environment variable check..."
echo "------------------------------------------------"
print_info "Checking OTEL configuration in API service:"
docker exec ppd_api env | grep OTEL || echo "No OTEL variables found"
echo ""

print_info "Checking OTEL configuration in Provider API:"
docker exec ppd_provider_api env | grep OTEL || echo "No OTEL variables found"
echo ""

echo "========================================"
echo "Debug Summary"
echo "========================================"
echo ""
echo "Access SigNoz UI at: http://localhost:3301"
echo ""
echo "If logs are not appearing in SigNoz:"
echo "1. Check that OTEL_ENABLED=true in your services"
echo "2. Restart services: docker-compose restart"
echo "3. Check Fluent Bit is collecting logs: docker logs ppd_fluent_bit"
echo "4. Check OTEL collector is receiving: docker logs ppd_signoz_otel_collector"
echo "5. Verify ClickHouse has signoz_logs database"
echo ""
echo "For detailed logs:"
echo "  docker logs ppd_fluent_bit -f          # Follow Fluent Bit logs"
echo "  docker logs ppd_signoz_otel_collector -f  # Follow OTEL collector"
echo ""

