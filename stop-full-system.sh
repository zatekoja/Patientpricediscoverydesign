#!/bin/bash
# Stop Full System - Clean shutdown of all services

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${YELLOW}🛑 Stopping Full System${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Stop Frontend
if [ -f /tmp/frontend.pid ]; then
    FRONTEND_PID=$(cat /tmp/frontend.pid)
    if ps -p $FRONTEND_PID > /dev/null 2>&1; then
        echo "Stopping Frontend (PID: $FRONTEND_PID)..."
        kill $FRONTEND_PID 2>/dev/null || true
        rm /tmp/frontend.pid
        echo -e "${GREEN}✅ Frontend stopped${NC}"
    fi
fi

# Stop Provider API
if [ -f /tmp/provider-api.pid ]; then
    PROVIDER_API_PID=$(cat /tmp/provider-api.pid)
    if ps -p $PROVIDER_API_PID > /dev/null 2>&1; then
        echo "Stopping Provider API (PID: $PROVIDER_API_PID)..."
        kill $PROVIDER_API_PID 2>/dev/null || true
        rm /tmp/provider-api.pid
        echo -e "${GREEN}✅ Provider API stopped${NC}"
    fi
fi

# Stop SSE Service
if [ -f /tmp/sse-service.pid ]; then
    SSE_PID=$(cat /tmp/sse-service.pid)
    if ps -p $SSE_PID > /dev/null 2>&1; then
        echo "Stopping SSE Service (PID: $SSE_PID)..."
        kill $SSE_PID 2>/dev/null || true
        rm /tmp/sse-service.pid
        echo -e "${GREEN}✅ SSE Service stopped${NC}"
    fi
fi

# Stop Core API
if [ -f /tmp/core-api.pid ]; then
    CORE_API_PID=$(cat /tmp/core-api.pid)
    if ps -p $CORE_API_PID > /dev/null 2>&1; then
        echo "Stopping Core API (PID: $CORE_API_PID)..."
        kill $CORE_API_PID 2>/dev/null || true
        rm /tmp/core-api.pid
        echo -e "${GREEN}✅ Core API stopped${NC}"
    fi
fi

# Stop Infrastructure Services (optional - comment out if you want to keep them running)
echo ""
echo "Stopping infrastructure services..."
docker-compose stop postgres redis mongo typesense 2>/dev/null || true
echo -e "${GREEN}✅ Infrastructure services stopped${NC}"

echo ""
echo -e "${GREEN}✅ All services stopped${NC}"
echo ""
echo "Note: Infrastructure services (PostgreSQL, Redis, MongoDB, Typesense) are stopped."
echo "To stop them completely, run: docker-compose down"
