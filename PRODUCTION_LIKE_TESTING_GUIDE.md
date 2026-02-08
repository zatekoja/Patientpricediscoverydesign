# Production-Like End-to-End Testing Guide

## Overview
This guide helps you test the complete system exactly as it would work in production - with all services running and connected together. This is the comprehensive testing guide that covers everything from service startup to complete end-to-end testing.

## System Architecture

### Service Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐     ┌─────────────┐
│   Frontend  │────▶│   Core API   │────▶│  PostgreSQL │     │    Redis    │
│  (Port 5173)│     │  (Port 8080) │     │  (Port 5432)│     │  (Port 6379)│
└─────────────┘     └──────────────┘     └─────────────┘     └─────────────┘
      │                    │                      │                  │
      │                    │                      │                  │
      │                    ▼                      │                  │
      │            ┌──────────────┐               │                  │
      │            │ Provider API │               │                  │
      │            │  (Port 3001) │               │                  │
      │            └──────────────┘               │                  │
      │                    │                      │                  │
      └────────────────────┼──────────────────────┼──────────────────┘
                           │                      │
                           ▼                      ▼
                    ┌──────────────┐     ┌─────────────┐
                    │  SSE Service │     │   MongoDB   │
                    │  (Port 8081) │     │ (Port 27017)│
                    └──────────────┘     └─────────────┘
```

### Capacity Update Flow Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. Admin Triggers Capacity Request                              │
│    POST /api/v1/capacity/request                                │
│    { facilityId, channel: "email" }                             │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. CapacityRequestService                                       │
│    - Generates secure token (32 bytes, base64url)               │
│    - Stores token hash in MongoDB (with TTL)                    │
│    - Creates form link: /api/v1/capacity/form/{token}          │
│    - Sends email via SES or WhatsApp                            │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. Provider Receives Email/WhatsApp                             │
│    - Email contains link with embedded token                     │
│    - Link format: {PUBLIC_BASE_URL}/api/v1/capacity/form/{token}│
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. Provider Clicks Link                                          │
│    GET /api/v1/capacity/form/:token                             │
│    - Token validated (exists, not expired, not used)            │
│    - Returns HTML form with fields:                             │
│      • capacityStatus (dropdown: available/busy/full/closed)     │
│      • avgWaitMinutes (number input)                            │
│      • urgentCareAvailable (checkbox)                           │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. Provider Submits Form                                        │
│    POST /api/v1/capacity/submit                                 │
│    Body: {                                                       │
│      token: "...",                                               │
│      capacityStatus: "busy",                                     │
│      avgWaitMinutes: 45,                                         │
│      urgentCareAvailable: true                                   │
│    }                                                             │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│ 6. Token Consumption & Validation                               │
│    - Token hash looked up in MongoDB                             │
│    - Validated: not used, not expired                            │
│    - Marked as used (usedAt timestamp set)                       │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│ 7. Facility Profile Update (MongoDB)                            │
│    FacilityProfileService.updateStatus()                        │
│    - Updates facility profile in MongoDB                         │
│    - Fields: capacityStatus, avgWaitMinutes, urgentCareAvailable │
│    - Records metrics (OTEL)                                      │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│ 8. Ingestion Webhook Trigger                                    │
│    POST {PROVIDER_INGESTION_WEBHOOK_URL}                        │
│    Body: {                                                       │
│      facilityId: "...",                                         │
│      eventId: "...",                                             │
│      source: "capacity_update"                                   │
│    }                                                             │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│ 9. Core API Ingestion (Go Backend)                              │
│    POST /api/v1/ingestion/capacity                              │
│    - Syncs facility profile from MongoDB → PostgreSQL           │
│    - Updates facilities table with capacity fields              │
│    - Publishes SSE events via Redis Event Bus                   │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│ 10. Real-Time Frontend Updates                                  │
│     - SSE clients receive capacity_update events                │
│     - Frontend updates facility cards in real-time              │
│     - Cache invalidated for updated facility                    │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

Before starting, ensure:
1. **Docker Desktop is running** (for PostgreSQL, Redis, MongoDB, Typesense)
2. **Go is installed** and in PATH
3. **Node.js v24** is available (via nvm)
4. **Dependencies are installed** in both `backend/` and `Frontend/`

**Verify prerequisites:**
```bash
# Check Docker
docker ps

# Check Go
which go
go version

# Check Node.js
source ~/.nvm/nvm.sh
nvm use 24
node --version

# Install Frontend dependencies if needed
cd Frontend && npm install
```

---

## Quick Start: One Command (Optional)

```bash
./start-full-system.sh
```

**Note:** This script requires all prerequisites to be met. If it fails, use the manual setup below.

---

## Manual Setup (Step-by-Step)

### Step 1: Start Docker Desktop

**macOS:**
- Open Docker Desktop application
- Wait for it to fully start (whale icon in menu bar should be steady)

**Verify Docker is running:**
```bash
docker ps
```

### Step 2: Start Infrastructure Services

**Terminal 1:**
```bash
cd /Users/fehintola/Documents/GitHub/Patientpricediscoverydesign
docker-compose up -d postgres redis mongo typesense

# Wait for services to be ready
sleep 5

# Verify services are running
docker-compose ps
```

**Expected Output:**
- ✅ `ppd_postgres` - running
- ✅ `ppd_redis` - running
- ✅ `ppd_mongo` - running
- ✅ `ppd_typesense` - running

---

### Step 3: Start Core API (Port 8080)

**Terminal 2:**

```bash
cd backend

# Set environment variables
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

# Start Core API
go run cmd/api/main.go
```

**Verify:**
```bash
curl http://localhost:8080/health
# Expected: OK
```

---

### Step 4: Start SSE Service (Port 8081)

**Terminal 3:**

```bash
cd backend

# Set environment variables
export REDIS_HOST=localhost
export REDIS_PORT=6379
export SERVER_PORT=8081
export SERVER_HOST=0.0.0.0

# Start SSE Service
go run cmd/sse/main.go
```

**Verify:**
```bash
curl http://localhost:8081/health
# Expected: OK
```

---

### Step 5: Start Provider API (Port 3001)

**Terminal 4:**

```bash
cd backend

# Set environment variables
export PROVIDER_ADMIN_TOKEN=test-admin-token
export PROVIDER_PUBLIC_BASE_URL=http://localhost:3001
export PROVIDER_INGESTION_WEBHOOK_URL=http://localhost:8080/api/v1/ingestion/capacity
export PORT=3001
export MONGO_URI=mongodb://localhost:27017
export MONGO_DB=patient_price_discovery

# Source nvm and use Node 24
source ~/.nvm/nvm.sh
nvm use 24

# Start Provider API
npm start
# OR if you have issues with --watch flag:
# npx ts-node api/example-server.ts
```

**Verify:**
```bash
curl http://localhost:3001/api/v1/health
# Expected: {"status":"ok"}
```

---

### Step 6: Start Frontend (Port 5173)

**Terminal 5:**

```bash
cd Frontend

# Install dependencies if not already installed
npm install

# Set environment variables
export VITE_API_BASE_URL=http://localhost:8080
export VITE_SSE_BASE_URL=http://localhost:8081
export VITE_GRAPHQL_BASE_URL=http://localhost:8081

# Start Frontend
npm run dev
```

**Verify:**
- Open browser: http://localhost:5173
- Frontend should load

---

## Complete End-to-End Test Flow

### Test Scenario: Capacity Update → Frontend Display

#### Step 1: Generate Capacity Update Token

```bash
# In a new terminal
cd backend
source ~/.nvm/nvm.sh && nvm use 24

# Generate token for a test facility
curl -X POST http://localhost:3001/api/v1/capacity/request \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-admin-token" \
  -d '{
    "facilityId": "facility-test-1",
    "channel": "email"
  }'
```

**Note:** Check Provider API logs for the form URL, or use the test script:
```bash
npx ts-node scripts/test-form-manually.ts facility-test-1
```

#### Step 2: Open Form in Browser

1. Copy the form URL from the output (e.g., `http://localhost:3001/api/v1/capacity/form/{token}`)
2. Open it in your browser
3. You should see the capacity update form

#### Step 3: Submit Capacity Update

1. **Fill out the form:**
   - Capacity Status: `busy`
   - Average Wait Time: `45`
   - Urgent Care Available: ✓ (checked)

2. **Click "Submit Update"**

3. **Verify Success Page:**
   - Should see: "Thank You - Your capacity update has been recorded successfully"

#### Step 4: Verify Backend Processing

**Check Provider API logs (Terminal 4):**
```
POST /api/v1/capacity/submit
Token consumed successfully
Facility updated
Webhook triggered: http://localhost:8080/api/v1/ingestion/capacity
```

**Check Core API logs (Terminal 2):**
```
POST /api/v1/ingestion/capacity
Webhook received
Facility capacity updated in database
SSE event published
```

**Check SSE Service logs (Terminal 3):**
```
SSE event broadcasted: capacity_update
Clients notified: 1
```

#### Step 5: Verify Frontend Display

1. **Open Frontend:** http://localhost:5173

2. **Search for Facility:**
   - Search for "Test Facility" or facility ID: `facility-test-1`
   - Or navigate to the facility if it's already visible

3. **Verify Updates Appear:**
   - [ ] Capacity status badge shows "Limited capacity" (yellow badge)
   - [ ] Wait time shows "45 min"
   - [ ] Urgent care badge appears (blue badge)

4. **Open Facility Modal:**
   - [ ] Click on the facility card
   - [ ] In "Provider Health" section:
     - [ ] Capacity Status: `busy`
     - [ ] Avg Wait Time: `45 mins`
     - [ ] SSE connection shows "connected"

#### Step 6: Test Real-Time Updates

1. **Generate another token:**
   ```bash
   npx ts-node scripts/test-form-manually.ts facility-test-1
   ```

2. **Submit another update:**
   - Capacity Status: `available`
   - Average Wait Time: `15`
   - Submit form

3. **Watch Frontend (without refreshing):**
   - [ ] Facility card updates automatically
   - [ ] Badge changes from yellow to green
   - [ ] Wait time updates to "15 min"
   - [ ] No page refresh needed

---

## Service URLs Reference

| Service | URL | Purpose |
|---------|-----|---------|
| Frontend | http://localhost:5173 | User interface |
| Core API | http://localhost:8080 | Main REST API |
| SSE Service | http://localhost:8081 | Real-time updates |
| Provider API | http://localhost:3001 | Capacity form & provider data |
| PostgreSQL | localhost:5432 | Primary database |
| Redis | localhost:6379 | Cache & event bus |
| MongoDB | localhost:27017 | Provider data store |
| Typesense | localhost:8108 | Search engine |

---

## Health Checks

Verify all services are running:

```bash
# Core API
curl http://localhost:8080/health
# Expected: OK

# SSE Service
curl http://localhost:8081/health
# Expected: OK

# Provider API
curl http://localhost:3001/api/v1/health
# Expected: {"status":"ok"}

# Frontend
curl http://localhost:5173
# Expected: HTML page
```

---

## Troubleshooting

### Service Won't Start

**Check if port is already in use:**
```bash
# macOS
lsof -i :8080
lsof -i :8081
lsof -i :3001
lsof -i :5173

# Kill process if needed
kill -9 <PID>
```

### Database Connection Issues

**Check PostgreSQL is running:**
```bash
docker-compose ps postgres
docker-compose logs postgres
```

**Check database exists:**
```bash
psql -h localhost -U postgres -d patient_price_discovery -c "SELECT 1;"
```

### Frontend Not Updating

**Check SSE connection:**
1. Open browser DevTools (F12)
2. Check Console for SSE connection messages
3. Check Network tab for SSE connection

**Verify SSE Service is running:**
```bash
curl http://localhost:8081/health
```

**Check Redis connection:**
```bash
docker-compose logs redis
redis-cli -h localhost -p 6379 ping
# Expected: PONG
```

### Webhook Not Triggering

**Check Provider API logs:**
- Look for "Webhook triggered" message
- Check for webhook retry attempts

**Check Core API logs:**
- Look for "POST /api/v1/ingestion/capacity" requests
- Check for any errors

**Verify webhook URL:**
```bash
# Test webhook endpoint directly
curl -X POST http://localhost:8080/api/v1/ingestion/capacity \
  -H "Content-Type: application/json" \
  -d '{
    "facilityId": "test-facility",
    "source": "capacity_update",
    "timestamp": "2024-01-01T00:00:00Z"
  }'
```

---

## Stopping All Services

### Option 1: Use Stop Script

```bash
./stop-full-system.sh
```

### Option 2: Manual Stop

**Stop each service:**
- Press `Ctrl+C` in each terminal running a service

**Stop infrastructure:**
```bash
docker-compose stop postgres redis mongo typesense
```

**Or stop everything:**
```bash
docker-compose down
```

---

## Next Steps

1. ✅ All services started
2. ✅ Complete end-to-end test flow
3. ✅ Verify real-time updates work
4. ✅ Test error scenarios
5. ✅ Performance testing
6. ✅ User acceptance testing

---

---

## Complete Test Scenarios

### Scenario 1: Happy Path - Full Flow

**Steps:**
1. ✅ Generate token
2. ✅ Open form in browser
3. ✅ Verify form UI/UX
4. ✅ Submit form with: `busy`, `45 min`, `urgent care: yes`
5. ✅ Verify success page
6. ✅ Check backend logs (webhook triggered)
7. ✅ Open frontend
8. ✅ Search for facility
9. ✅ Verify capacity status shows "Limited capacity" (yellow)
10. ✅ Verify wait time shows "45 min"
11. ✅ Verify urgent care badge appears
12. ✅ Open facility modal
13. ✅ Verify detailed capacity info matches

**Expected Result:** ✅ All steps pass, updates visible in frontend

---

### Scenario 2: Real-Time Update Flow

**Steps:**
1. ✅ Open frontend and find facility
2. ✅ Note current capacity status (e.g., "Available")
3. ✅ Generate new token and submit form
4. ✅ Change capacity to "busy" with wait time "60 min"
5. ✅ Submit form
6. ✅ **Without refreshing frontend:**
   - [ ] Facility card updates automatically
   - [ ] Badge changes from green to yellow
   - [ ] Wait time updates to "60 min"
   - [ ] SSE connection shows "connected"

**Expected Result:** ✅ Frontend updates in real-time via SSE

---

### Scenario 3: Multiple Rapid Updates

**Steps:**
1. ✅ Submit update: `available`, `15 min`
2. ✅ Wait 2 seconds
3. ✅ Submit update: `busy`, `30 min`
4. ✅ Wait 2 seconds
5. ✅ Submit update: `full`, `0 min`
6. ✅ Watch frontend for each update

**Expected Result:** ✅ Frontend shows each update in sequence

---

### Scenario 4: Error Handling

**Test Invalid Token:**
1. ✅ Try to access form with expired token
2. ✅ Verify error page displays
3. ✅ Verify error is user-friendly

**Test Invalid Data:**
1. ✅ Submit form with invalid capacity status
2. ✅ Verify error page shows valid options
3. ✅ Verify can return to form

**Test Network Issues:**
1. ✅ Stop Core API (simulate webhook failure)
2. ✅ Submit form
3. ✅ Verify webhook retries (check logs)
4. ✅ Verify form still shows success (async processing)

---

## Frontend UI/UX Testing Checklist

### Search Results Component

**Capacity Status Display:**
- [ ] **Available** → Green badge with green dot
- [ ] **Busy** → Yellow badge with yellow dot
- [ ] **Full** → Gray badge with gray dot
- [ ] **Closed** → Gray badge with gray dot
- [ ] **Null/undefined** → Shows "Unknown" or default

**Wait Time Display:**
- [ ] Shows wait time in minutes: "45 min"
- [ ] Shows "N/A" or "--" when not provided
- [ ] Formatting is consistent

**Urgent Care Badge:**
- [ ] Blue badge appears when `urgentCareAvailable: true`
- [ ] Badge text: "Urgent care available"
- [ ] Badge is properly styled

**Real-Time Updates:**
- [ ] Badge updates without page refresh
- [ ] Wait time updates without page refresh
- [ ] Urgent care badge appears/disappears in real-time

---

### Facility Modal Component

**Provider Health Section:**
- [ ] Section is visible when provider health data exists
- [ ] Capacity Status card shows correct value
- [ ] Avg Wait Time card shows correct value
- [ ] Cards are properly styled (blue/purple backgrounds)

**Real-Time Updates:**
- [ ] Capacity status updates in modal without closing/reopening
- [ ] Wait time updates in modal
- [ ] SSE connection indicator shows status

**SSE Connection Status:**
- [ ] Shows "connected" when SSE is active
- [ ] Shows "connecting" when establishing connection
- [ ] Shows "error" when connection fails
- [ ] Last update timestamp is displayed

---

### Map View Component

**Capacity Indicators:**
- [ ] Facility markers show capacity status dots
- [ ] Green dot for "Available"
- [ ] Yellow dot for "Busy" or "Limited"
- [ ] Gray dot for other statuses

**Sidebar List:**
- [ ] Capacity status dots appear next to facility names
- [ ] Dots update in real-time

---

## Browser Compatibility Testing

### Desktop Browsers

**Chrome (Latest):**
- [ ] Form renders correctly
- [ ] Form submission works
- [ ] Frontend displays updates
- [ ] SSE updates work

**Firefox (Latest):**
- [ ] Form renders correctly
- [ ] Form submission works
- [ ] Frontend displays updates
- [ ] SSE updates work

**Safari (Latest):**
- [ ] Form renders correctly
- [ ] Form submission works
- [ ] Frontend displays updates
- [ ] SSE updates work

**Edge (Latest):**
- [ ] Form renders correctly
- [ ] Form submission works
- [ ] Frontend displays updates
- [ ] SSE updates work

---

### Mobile Browsers

**iOS Safari:**
- [ ] Form is responsive
- [ ] Form fields are tappable
- [ ] Frontend displays correctly
- [ ] SSE updates work

**Android Chrome:**
- [ ] Form is responsive
- [ ] Form fields are tappable
- [ ] Frontend displays correctly
- [ ] SSE updates work

---

## Performance Testing

### Form Performance
- [ ] Form loads in < 1 second
- [ ] Form submission responds quickly
- [ ] Success page loads immediately

### Frontend Update Performance
- [ ] SSE updates appear within 2 seconds of form submission
- [ ] No lag when updating multiple facilities
- [ ] UI remains responsive during updates

### Network Conditions
- [ ] Test on slow 3G connection
- [ ] Test on fast WiFi
- [ ] Verify graceful degradation

---

## Debugging Tips

### Check Backend Logs

**Provider API Logs:**
```bash
# Look for:
POST /api/v1/capacity/submit
Token consumed: ...
Facility updated: ...
Webhook triggered: ...
```

**Core API Logs:**
```bash
# Look for:
POST /api/v1/ingestion/capacity
Webhook received: ...
Facility updated in DB: ...
SSE event published: ...
```

**SSE Service Logs:**
```bash
# Look for:
SSE client connected: ...
Capacity update event: ...
Event broadcasted: ...
```

### Check Frontend Console

**Open Browser DevTools (F12):**
- Check Console for errors
- Check Network tab for SSE connection
- Check Network tab for API calls

**SSE Connection:**
```javascript
// In console, check:
// Should see: "SSE connected: /api/v1/facilities/stream"
// Should see: "capacity_update" events
```

### Check Database

**PostgreSQL:**
```sql
-- Check facility capacity status
SELECT id, name, capacity_status, avg_wait_minutes, urgent_care_available
FROM facilities
WHERE id = 'facility-test-1';
```

**MongoDB (Provider Data):**
```javascript
// Check capacity request tokens
db.capacity_tokens.find({ facilityId: "facility-test-1" })
```

---

## Common Issues & Solutions

### Issue: Form doesn't load
**Solution:**
- Check Provider API is running
- Check token is valid (not expired)
- Check browser console for errors
- Verify CORS is configured

### Issue: Form submission fails
**Solution:**
- Check backend logs for errors
- Verify token hasn't been used
- Check network tab for HTTP status
- Verify form data is valid

### Issue: Frontend doesn't update
**Solution:**
- Check SSE connection status
- Verify Core API received webhook
- Check PostgreSQL was updated
- Verify SSE service is broadcasting
- Check browser console for SSE errors

### Issue: Updates are delayed
**Solution:**
- Check webhook retry logs
- Verify network connectivity
- Check SSE service performance
- Verify database write performance

---

## Test Data Reference

### Valid Capacity Statuses
- `available` → Green badge "Available"
- `busy` → Yellow badge "Limited capacity"
- `full` → Gray badge "Full"
- `closed` → Gray badge "Closed"

### Test Facility IDs
- `facility-test-1` - Default test facility
- `facility-ttl-global` - TTL test facility
- `facility-ttl-override` - Per-facility TTL test

### Sample Form Submissions

**Test 1: Available Facility**
```json
{
  "capacityStatus": "available",
  "avgWaitMinutes": 15,
  "urgentCareAvailable": true
}
```

**Test 2: Busy Facility**
```json
{
  "capacityStatus": "busy",
  "avgWaitMinutes": 45,
  "urgentCareAvailable": false
}
```

**Test 3: Full Facility**
```json
{
  "capacityStatus": "full",
  "avgWaitMinutes": 0,
  "urgentCareAvailable": false
}
```

---

## Success Criteria

### Form UI/UX ✅
- [ ] Form is visually appealing and matches design system
- [ ] Form is responsive on all screen sizes
- [ ] Form is accessible (keyboard navigation, screen readers)
- [ ] Form validation works correctly
- [ ] Error messages are clear and helpful

### Backend Integration ✅
- [ ] Form submission is processed correctly
- [ ] Token security works (expiration, single-use)
- [ ] Webhook is triggered successfully
- [ ] Webhook retries on failure
- [ ] Database is updated correctly

### Frontend Display ✅
- [ ] Capacity status displays correctly
- [ ] Wait time displays correctly
- [ ] Urgent care badge appears/disappears correctly
- [ ] Updates appear in real-time via SSE
- [ ] UI remains responsive during updates

### End-to-End Flow ✅
- [ ] Complete flow works: Form → Backend → Webhook → Core API → SSE → Frontend
- [ ] Updates appear in frontend within 2 seconds
- [ ] Multiple rapid updates are handled correctly
- [ ] Error scenarios are handled gracefully

---

## Integration Points to Verify

When testing, verify these integration points:
- [ ] Provider API → Email/WhatsApp delivery
- [ ] Provider API → MongoDB (facility profiles)
- [ ] Provider API → MongoDB (tokens)
- [ ] Provider API → Core API webhook
- [ ] Core API → PostgreSQL sync
- [ ] Core API → Redis event bus
- [ ] SSE Server → Frontend clients

---

## Known Issues / Edge Cases

### Current Limitations
- [ ] No token refresh mechanism - provider must use link within TTL
- [ ] No audit log of capacity updates (only metrics)

### Implemented Enhancements ✅
- [x] **Token TTL configurable** - via environment variable or per-facility config
- [x] **Webhook retry mechanism** - automatic retry with exponential backoff
- [x] **Email template customization** - via environment variables
- [x] **Form branding** - improved styling matching frontend design
- [x] **Capacity status validation** - restricted to valid values

### Future Enhancements
- [ ] Add token refresh endpoint
- [ ] Add audit log table for capacity updates
- [ ] Add multi-language support for forms
- [ ] Add SMS as alternative channel

---

## Additional Resources

- **Form Testing Guide:** `CAPACITY_FORM_FRONTEND_TESTING_GUIDE.md` - Detailed form UI/UX testing
- **Implementation Details:** `CAPACITY_UPDATE_IMPLEMENTATION.md` - Technical implementation reference
