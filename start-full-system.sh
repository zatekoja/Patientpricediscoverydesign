#!/bin/bash
# Start Full System - Production-like End-to-End Testing
# This script starts all services needed for complete system testing

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${BLUE}🚀 Starting Full System - Production-like Testing${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check prerequisites
echo -e "${YELLOW}📋 Checking prerequisites...${NC}"

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    exit 1
fi

# Check Docker Compose
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed${NC}"
    exit 1
fi

# Check Node.js
if ! command -v node &> /dev/null; then
    echo -e "${RED}❌ Node.js is not installed${NC}"
    exit 1
fi

# Check Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ All prerequisites met${NC}"
echo ""

# Step 1: Start Infrastructure Services
echo -e "${YELLOW}Step 1: Starting infrastructure services (PostgreSQL, Redis, MongoDB, Typesense)...${NC}"
docker-compose up -d postgres redis mongo typesense

echo "Waiting for services to be ready..."
sleep 5

# Check services
echo -e "${BLUE}Checking service health...${NC}"
docker-compose ps postgres redis mongo typesense

echo -e "${GREEN}✅ Infrastructure services started${NC}"
echo ""

# Step 2: Start Core API
echo -e "${YELLOW}Step 2: Starting Core API (port 8080)...${NC}"
echo "This will run in the background. Check Terminal 2 for logs."

cd backend

# Set environment variables for Core API
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=patient_price_discovery
export DB_SSLMODE=disable
export REDIS_HOST=localhost
export REDIS_PORT=6379
export SERVER_PORT=8080
export SERVER_HOST=0.0.0.0

# Start Core API in background
echo "Starting Core API..."
go run cmd/api/main.go > /tmp/core-api.log 2>&1 &
CORE_API_PID=$!
echo $CORE_API_PID > /tmp/core-api.pid

# Wait for Core API to start
echo "Waiting for Core API to start..."
HEALTH_CHECK_PASSED=false
for i in {1..30}; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Core API is running on http://localhost:8080${NC}"
        HEALTH_CHECK_PASSED=true
        break
    fi
    sleep 1
done

# Exit if health check failed
if [ "$HEALTH_CHECK_PASSED" = false ]; then
    echo -e "${RED}❌ Core API failed to start within 30 seconds${NC}"
    echo "Check /tmp/core-api.log for errors"
    exit 1
fi

cd ..
echo ""

# Step 3: Start SSE Service
echo -e "${YELLOW}Step 3: Starting SSE Service (port 8081)...${NC}"
echo "This will run in the background. Check Terminal 3 for logs."

cd backend

# Set environment variables for SSE Service
export SSE_REDIS_HOST=localhost
export SSE_REDIS_PORT=6379
export SSE_PORT=8081
export SSE_HOST=0.0.0.0

# Start SSE Service in background
echo "Starting SSE Service..."
go run cmd/sse/main.go > /tmp/sse-service.log 2>&1 &
SSE_PID=$!
echo $SSE_PID > /tmp/sse-service.pid

# Wait for SSE Service to start
echo "Waiting for SSE Service to start..."
for i in {1..30}; do
    if curl -s http://localhost:8081/health > /dev/null 2>&1; then
        echo -e "${GREEN}✅ SSE Service is running on http://localhost:8081${NC}"
        break
    fi
    sleep 1
done

cd ..
echo ""

# Step 4: Start Provider API
echo -e "${YELLOW}Step 4: Starting Provider API (port 3001)...${NC}"
echo "This will run in the background. Check Terminal 4 for logs."

cd backend

# Set environment variables for Provider API
export PROVIDER_ADMIN_TOKEN=test-admin-token
export PROVIDER_PUBLIC_BASE_URL=http://localhost:3001
export PROVIDER_INGESTION_WEBHOOK_URL=http://localhost:8080/api/v1/ingestion/capacity
export PORT=3001
export MONGO_URI=mongodb://localhost:27017
export MONGO_DB=patient_price_discovery

# Source nvm and use Node 24
source ~/.nvm/nvm.sh 2>/dev/null || true
nvm use 24 2>/dev/null || true

# Start Provider API in background
echo "Starting Provider API..."
npm run dev > /tmp/provider-api.log 2>&1 &
PROVIDER_API_PID=$!
echo $PROVIDER_API_PID > /tmp/provider-api.pid

# Wait for Provider API to start
echo "Waiting for Provider API to start..."
for i in {1..30}; do
    if curl -s http://localhost:3001/api/v1/health > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Provider API is running on http://localhost:3001${NC}"
        break
    fi
    sleep 1
done

cd ..
echo ""

# Step 5: Start Frontend
echo -e "${YELLOW}Step 5: Starting Frontend (port 5173)...${NC}"
echo "This will run in the background. Check Terminal 5 for logs."

cd Frontend

# Set environment variables for Frontend
export VITE_API_BASE_URL=http://localhost:8080
export VITE_SSE_BASE_URL=http://localhost:8081
export VITE_GRAPHQL_BASE_URL=http://localhost:8081

# Start Frontend in background
echo "Starting Frontend..."
npm run dev > /tmp/frontend.log 2>&1 &
FRONTEND_PID=$!
echo $FRONTEND_PID > /tmp/frontend.pid

# Wait for Frontend to start
echo "Waiting for Frontend to start..."
for i in {1..30}; do
    if curl -s http://localhost:5173 > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Frontend is running on http://localhost:5173${NC}"
        break
    fi
    sleep 1
done

cd ..
echo ""

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}✅ All services started successfully!${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${BLUE}📋 Service URLs:${NC}"
echo "   • Frontend:        http://localhost:5173"
echo "   • Core API:        http://localhost:8080"
echo "   • SSE Service:     http://localhost:8081"
echo "   • Provider API:    http://localhost:3001"
echo ""
echo -e "${BLUE}📋 Log Files:${NC}"
echo "   • Core API:        /tmp/core-api.log"
echo "   • SSE Service:     /tmp/sse-service.log"
echo "   • Provider API:    /tmp/provider-api.log"
echo "   • Frontend:        /tmp/frontend.log"
echo ""
echo -e "${BLUE}📋 Process IDs:${NC}"
echo "   • Core API PID:    $CORE_API_PID"
echo "   • SSE Service PID: $SSE_PID"
echo "   • Provider API PID: $PROVIDER_API_PID"
echo "   • Frontend PID:    $FRONTEND_PID"
echo ""
echo -e "${YELLOW}💡 To stop all services, run: ./stop-full-system.sh${NC}"
echo ""
echo -e "${GREEN}🎉 System is ready for testing!${NC}"
echo ""
echo "Next steps:"
echo "1. Open http://localhost:5173 in your browser"
echo "2. Generate a capacity update token"
echo "3. Submit a capacity update form"
echo "4. Watch the frontend update in real-time!"
