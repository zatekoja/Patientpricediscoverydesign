# SSE Server and Cache Implementation

## Overview

This document describes the real-time Server-Sent Events (SSE) implementation and intelligent caching strategy for the Patient Price Discovery platform. The system provides real-time updates for facility capacity, wait times, urgent care availability, and service health while maintaining optimal performance through strategic caching.

## Architecture

### Components

1. **SSE Handler** (`internal/api/handlers/sse_handler.go`)
   - Manages SSE connections
   - Routes events to connected clients
   - Provides facility-specific and regional streams

2. **Event Bus** (`internal/adapters/events/redis_event_bus.go`)
   - Redis Pub/Sub-based event distribution
   - Supports multiple channels and subscriptions
   - Handles concurrent connections efficiently

3. **Cache Layer** (`internal/adapters/cache/redis_adapter.go`)
   - Redis-based HTTP response caching
   - Pattern-based cache invalidation
   - Configurable TTLs per endpoint

4. **Cache Invalidation Service** (`internal/application/services/cache_invalidation_service.go`)
   - Event-driven cache invalidation
   - Strategic invalidation patterns
   - Prevents cache stampede

## Real-Time Streaming (SSE)

### Endpoints

#### 1. Facility-Specific Stream
```
GET /api/stream/facilities/{id}
```

**Use Case**: Subscribe to updates for a specific facility

**Events**:
- `capacity_update` - Capacity status changes
- `wait_time_update` - Average wait time changes
- `urgent_care_update` - Urgent care availability changes
- `service_health_update` - Service health changes
- `heartbeat` - Keep-alive (every 30s)

**Example**:
```javascript
const eventSource = new EventSource('/api/stream/facilities/fac_001');

eventSource.addEventListener('capacity_update', (event) => {
  const data = JSON.parse(event.data);
  console.log('Capacity updated:', data.changed_fields.capacity_status);
});

eventSource.addEventListener('wait_time_update', (event) => {
  const data = JSON.parse(event.data);
  console.log('Wait time updated:', data.changed_fields.avg_wait_minutes);
});
```

#### 2. Regional Stream
```
GET /api/stream/facilities/region?lat=X&lon=Y&radius=Z
```

**Use Case**: Subscribe to updates for all facilities within a geographic region

**Parameters**:
- `lat` (required): Latitude
- `lon` (required): Longitude
- `radius` (optional): Radius in kilometers (default: 50km)

**Benefits**:
- Reduces bandwidth by filtering events client-side
- Ideal for map views showing multiple facilities
- Automatic geographic filtering on the server

**Example**:
```javascript
// Subscribe to facilities within 25km of user's location
const eventSource = new EventSource(
  '/api/stream/facilities/region?lat=6.5244&lon=3.3792&radius=25'
);

eventSource.addEventListener('capacity_update', (event) => {
  const data = JSON.parse(event.data);
  updateMapMarker(data.facility_id, data.changed_fields);
});
```

### Event Format

All events follow this structure:

```json
{
  "id": "20260207123045-a1b2c3d4",
  "facility_id": "fac_001",
  "event_type": "capacity_update",
  "timestamp": "2026-02-07T12:30:45Z",
  "location": {
    "latitude": 6.5244,
    "longitude": 3.3792
  },
  "changed_fields": {
    "capacity_status": "high",
    "avg_wait_minutes": 15
  }
}
```

### Connection Management

- **Heartbeat**: Sent every 30 seconds to keep connection alive
- **Auto-reconnect**: Clients should implement reconnection logic
- **Graceful shutdown**: Server notifies clients before closing
- **Concurrent connections**: Server efficiently handles multiple clients per facility

## Caching Strategy

### Cache Configuration

| Endpoint | TTL | Rationale |
|----------|-----|-----------|
| `/api/facilities/search` | 5 minutes | Balances freshness with performance |
| `/api/facilities/{id}` | 10 minutes | Individual facility data changes less frequently |
| `/api/facilities/suggest` | 3 minutes | Autocomplete needs fresh data |
| `/api/insurance-providers` | 30 minutes | Provider data is relatively static |
| `/api/procedures` | 30 minutes | Procedure catalog rarely changes |
| `/api/geocode` | 1 hour | Geocoding results are immutable |

### Cache Invalidation Strategy

**Option B: TTL-Based Expiration (Implemented)**

We use natural TTL expiration for search caches rather than aggressive invalidation.

**Rationale**:
1. **Performance**: Avoids cache stampede when facilities update
2. **Scalability**: Reduces cache invalidation overhead
3. **Real-time updates**: SSE provides immediate updates to connected clients
4. **Acceptable staleness**: 3-5 minute staleness is acceptable for search results
5. **Eventual consistency**: All caches refresh within their TTL window

**Invalidation Rules**:

1. **Facility Updates** → Invalidate only specific facility cache
   - Pattern: `http:cache:*facilities/{id}*`
   - Impact: Minimal (single facility)
   - Trigger: On any facility field update

2. **Search Caches** → Let TTL expire naturally
   - No invalidation on facility updates
   - Connected clients get real-time updates via SSE
   - Disconnected clients see updates within TTL window (3-5 min)

3. **Manual Invalidation** (Admin operations only)
   - Use `InvalidateSearchCaches()` for bulk updates
   - Use `InvalidateRegionalCaches()` for region-specific updates

### Cache Keys

Cache keys are generated using SHA-256 hashing:

```
http:cache:{hash of request}
```

Example:
```
GET /api/facilities/search?lat=6.5244&lon=3.3792&radius=10
→ http:cache:a1b2c3d4e5f6...
```

### Cache Middleware

The cache middleware:
1. Only caches GET requests
2. Only caches successful responses (200 OK)
3. Adds `X-Cache: HIT` or `X-Cache: MISS` headers
4. Captures and stores response body
5. Applies per-route TTL configuration

## Event Flow

### Facility Update Flow

```
┌─────────────────┐
│  Client sends   │
│  PATCH /api/    │
│  facilities/{id}│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Facility        │
│ Handler         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Facility        │
│ Service.Update()│
└────────┬────────┘
         │
         ├──────────────────────────┐
         │                          │
         ▼                          ▼
┌─────────────────┐       ┌─────────────────┐
│ Update Database │       │ Update Search   │
│ (PostgreSQL)    │       │ Index           │
└─────────────────┘       │ (Typesense)     │
                          └─────────────────┘
         │
         ▼
┌─────────────────┐
│ Publish Event   │
│ to Event Bus    │
│ (Redis Pub/Sub) │
└────────┬────────┘
         │
         ├─────────────────────────────────┐
         │                                 │
         ▼                                 ▼
┌─────────────────┐              ┌─────────────────┐
│ Facility-       │              │ Global Channel  │
│ Specific        │              │ (for regional   │
│ Channel         │              │ subscribers)    │
└────────┬────────┘              └────────┬────────┘
         │                                │
         ├────────────────┬───────────────┤
         │                │               │
         ▼                ▼               ▼
┌─────────────────┐ ┌──────────┐ ┌──────────────┐
│ SSE Clients     │ │ SSE      │ │ Cache        │
│ (Facility-      │ │ Clients  │ │ Invalidation │
│ specific)       │ │ (Regional)│ │ Service      │
└─────────────────┘ └──────────┘ └──────┬───────┘
                                         │
                                         ▼
                                  ┌──────────────┐
                                  │ Invalidate   │
                                  │ Facility     │
                                  │ Cache Only   │
                                  └──────────────┘
```

### Real-Time Update Flow

1. **Facility Update**: Admin updates facility capacity via API
2. **Database Update**: Changes persisted to PostgreSQL
3. **Search Index Update**: Typesense index updated
4. **Event Publishing**: Event published to Redis Pub/Sub
5. **Event Distribution**: 
   - Facility-specific subscribers receive event
   - Regional subscribers within radius receive event
   - Cache invalidation service receives event
6. **Cache Invalidation**: Only specific facility cache invalidated
7. **Client Updates**: Connected SSE clients receive instant updates
8. **Search Cache**: Expires naturally within 3-5 minutes

## Implementation Details

### Facility Service Updates

The `FacilityService.Update()` method detects changes and publishes appropriate events:

```go
func (s *FacilityService) Update(ctx context.Context, facility *entities.Facility) error {
    // Get existing facility
    existing, err := s.repo.GetByID(ctx, facility.ID)
    
    // Update database
    s.repo.Update(ctx, facility)
    
    // Update search index
    s.searchRepo.Index(ctx, facility)
    
    // Publish events if relevant fields changed
    if s.eventBus != nil {
        s.publishUpdateEvents(ctx, existing, facility)
    }
    
    return nil
}
```

Monitored fields:
- `capacity_status`
- `avg_wait_minutes`
- `urgent_care_available`

### Cache Invalidation Service

Runs as a background service:

```go
func (s *CacheInvalidationService) Start() error {
    // Subscribe to global facility updates
    eventChan, err := s.eventBus.Subscribe(s.ctx, providers.EventChannelFacilityUpdates)
    
    go s.processEvents(eventChan)
    return nil
}
```

Handles events:
- Invalidates specific facility cache
- Logs invalidation actions
- Does NOT invalidate search caches (TTL strategy)

## Configuration

### Environment Variables

```bash
# Redis Configuration (required for SSE and caching)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Typesense Configuration (required for search)
TYPESENSE_HOST=localhost
TYPESENSE_PORT=8108
TYPESENSE_API_KEY=your-api-key

# Database Configuration
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=patient_price_discovery
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
```

### Cache TTL Tuning

To adjust cache TTLs, modify `cache.go`:

```go
routeConfigs: map[string]CacheConfig{
    "/api/facilities/search":   {TTLSeconds: 300, Enabled: true},  // 5 minutes
    "/api/facilities/":         {TTLSeconds: 600, Enabled: true},  // 10 minutes
    "/api/facilities/suggest":  {TTLSeconds: 180, Enabled: true},  // 3 minutes
}
```

## Testing

### Manual Testing

#### 1. Test Facility-Specific SSE

```bash
# Terminal 1: Subscribe to facility updates
curl -N -H "Accept: text/event-stream" \
  http://localhost:8080/api/stream/facilities/fac_001

# Terminal 2: Update facility
curl -X PATCH http://localhost:8080/api/facilities/fac_001 \
  -H "Content-Type: application/json" \
  -d '{
    "capacity_status": "high",
    "avg_wait_minutes": 15
  }'

# Expected: Terminal 1 should receive capacity_update event
```

#### 2. Test Regional SSE

```bash
# Subscribe to regional updates (Lagos area)
curl -N -H "Accept: text/event-stream" \
  "http://localhost:8080/api/stream/facilities/region?lat=6.5244&lon=3.3792&radius=25"
```

#### 3. Test Caching

```bash
# First request (cache miss)
curl -v http://localhost:8080/api/facilities/search?lat=6.5244&lon=3.3792&radius=10
# Check for: X-Cache: MISS

# Second request (cache hit)
curl -v http://localhost:8080/api/facilities/search?lat=6.5244&lon=3.3792&radius=10
# Check for: X-Cache: HIT

# Update facility
curl -X PATCH http://localhost:8080/api/facilities/fac_001 \
  -H "Content-Type: application/json" \
  -d '{"capacity_status": "low"}'

# Verify facility cache invalidated
curl -v http://localhost:8080/api/facilities/fac_001
# Check for: X-Cache: MISS

# Verify search cache NOT invalidated (TTL strategy)
curl -v http://localhost:8080/api/facilities/search?lat=6.5244&lon=3.3792&radius=10
# Check for: X-Cache: HIT (should still be cached)
```

### Load Testing

Use [k6](https://k6.io/) for load testing:

```javascript
// sse-load-test.js
import { check } from 'k6';
import http from 'k6/http';

export let options = {
  stages: [
    { duration: '30s', target: 100 },  // Ramp up to 100 SSE connections
    { duration: '1m', target: 100 },   // Stay at 100
    { duration: '30s', target: 0 },    // Ramp down
  ],
};

export default function () {
  const response = http.get(
    'http://localhost:8080/api/stream/facilities/fac_001',
    {
      headers: { 'Accept': 'text/event-stream' },
    }
  );
  
  check(response, {
    'is status 200': (r) => r.status === 200,
  });
}
```

Run: `k6 run sse-load-test.js`

## Performance Metrics

### Expected Performance

- **SSE Connections**: 10,000+ concurrent connections per server
- **Event Latency**: < 50ms from publish to client receipt
- **Cache Hit Rate**: 80-90% for search endpoints
- **Cache Invalidation**: < 10ms per facility
- **Memory Usage**: ~100KB per active SSE connection

### Monitoring

Key metrics to monitor:

1. **SSE Metrics**:
   - Active connections count
   - Event publish rate
   - Event delivery latency
   - Connection duration

2. **Cache Metrics**:
   - Hit rate per endpoint
   - Miss rate per endpoint
   - Invalidation rate
   - Cache size

3. **Event Bus Metrics**:
   - Pub/Sub message rate
   - Channel subscription count
   - Message queue depth

## Best Practices

### For Frontend Developers

1. **Implement Reconnection Logic**:
```javascript
function connectSSE(facilityId) {
  let eventSource;
  let reconnectAttempts = 0;
  
  function connect() {
    eventSource = new EventSource(`/api/stream/facilities/${facilityId}`);
    
    eventSource.onopen = () => {
      reconnectAttempts = 0;
      console.log('SSE connected');
    };
    
    eventSource.onerror = () => {
      eventSource.close();
      reconnectAttempts++;
      const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
      setTimeout(connect, delay);
    };
    
    eventSource.addEventListener('capacity_update', handleCapacityUpdate);
  }
  
  connect();
  return eventSource;
}
```

2. **Use Regional Streams for Maps**:
```javascript
// For map view with multiple facilities
const eventSource = new EventSource(
  `/api/stream/facilities/region?lat=${userLat}&lon=${userLon}&radius=50`
);
```

3. **Clean Up Connections**:
```javascript
useEffect(() => {
  const eventSource = connectSSE(facilityId);
  
  return () => {
    eventSource.close();
  };
}, [facilityId]);
```

### For Backend Developers

1. **Always Publish Events**: When updating facilities, always publish events if event bus is available
2. **Use Context Timeouts**: Always use context with timeout for cache operations
3. **Log Invalidations**: Log all cache invalidations for debugging
4. **Monitor Event Bus**: Watch for event queue buildup
5. **Test Edge Cases**: Test reconnection, network issues, high load

## Troubleshooting

### SSE Connection Drops

**Symptom**: Clients frequently disconnect

**Solutions**:
- Check heartbeat interval (default: 30s)
- Verify proxy/load balancer timeout settings
- Ensure Redis connection is stable
- Check server resource usage

### Cache Not Invalidating

**Symptom**: Stale data after facility updates

**Solutions**:
- Verify cache invalidation service is running
- Check event bus subscription status
- Verify Redis connection
- Review cache key patterns

### High Cache Miss Rate

**Symptom**: Cache hit rate < 50%

**Solutions**:
- Check if TTLs are too short
- Verify cache key consistency
- Check for high update rate
- Review query parameter ordering

### Event Delivery Delays

**Symptom**: Events take > 1 second to reach clients

**Solutions**:
- Check Redis Pub/Sub latency
- Verify network connection
- Check server CPU usage
- Review event channel buffer sizes

## Future Enhancements

1. **Event Filtering**: Allow clients to filter events by type
2. **Event History**: Provide last N events on connection
3. **Compression**: Compress SSE events for bandwidth savings
4. **Metrics Dashboard**: Real-time monitoring dashboard
5. **A/B Testing**: Test different cache TTLs and invalidation strategies
6. **Multi-Region**: Geo-distributed event bus for global deployment
7. **WebSocket Alternative**: Offer WebSocket for bidirectional communication

## References

- [Server-Sent Events Spec](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [Redis Pub/Sub](https://redis.io/docs/manual/pubsub/)
- [HTTP Caching](https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching)
- [Cache Stampede Prevention](https://en.wikipedia.org/wiki/Cache_stampede)

## Conclusion

This implementation provides a robust, scalable real-time update system with intelligent caching. The TTL-based cache expiration strategy balances performance with data freshness, while SSE provides instant updates to connected clients. The system is designed to handle high concurrency and scale horizontally as needed.

