# Read Performance Optimization - Implementation Checklist

## Quick Start Guide

This checklist helps you implement read performance optimizations systematically.

---

## Phase 1: Quick Wins (Week 1) ✅

### 1.1 Enhanced Caching ✅

**Files Modified/Created:**
- ✅ `internal/domain/providers/cache_provider.go` - Added GetMulti, SetMulti, TTL methods
- ✅ `internal/adapters/cache/redis_adapter.go` - Implemented batch operations with Redis pipelining
- ✅ `internal/adapters/database/cached_facility_adapter.go` - Cache decorator for FacilityRepository

**Benefits:**
- 50-80% latency reduction for cached data
- Reduced database load by 60-80%
- Automatic cache invalidation on writes

**Next Steps:**
```bash
# 1. Update your main.go to use CachedFacilityAdapter
# Wrap existing adapter with caching:
facilityAdapter := database.NewFacilityAdapter(dbClient)
cachedFacilityAdapter := database.NewCachedFacilityAdapter(facilityAdapter, cacheProvider)

# 2. Test caching
go test ./internal/adapters/cache/...
go test ./internal/adapters/database/...
```

### 1.2 Database Indexes ✅

**Files Created:**
- ✅ `migrations/005_performance_indexes.sql` - Comprehensive indexing strategy

**Indexes Added:**
- Composite indexes for filtered queries
- Covering indexes to reduce table heap access
- Partial indexes for active records only
- GiST indexes for geospatial queries
- B-tree indexes for sorting and range queries

**Next Steps:**
```bash
# Run the migration (do this during low-traffic period)
# The indexes use CONCURRENTLY to avoid table locks

# For development:
docker exec ppd_postgres psql -U postgres -d patient_price_discovery -f /migrations/005_performance_indexes.sql

# For production:
# Run during maintenance window or ensure CONCURRENTLY works with your PostgreSQL version
psql -h <host> -U <user> -d patient_price_discovery -f migrations/005_performance_indexes.sql

# Verify indexes were created:
docker exec ppd_postgres psql -U postgres -d patient_price_discovery -c "\di+"
```

### 1.3 HTTP Response Optimization ✅

**Files Created:**
- ✅ `internal/api/middleware/performance.go` - Compression, ETag, Cache-Control middleware

**Features:**
- Gzip compression (5-10x response size reduction)
- ETag support (304 Not Modified responses)
- Cache-Control headers (browser/CDN caching)
- Response pooling (reduced GC pressure)

**Next Steps:**
```bash
# Update your API routes to use the middleware
# In your main.go or routes setup:

import "github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/middleware"

// Apply to all routes
mux.Handle("/api/", middleware.ResponseOptimization(yourHandler))

# Or apply selectively:
mux.Handle("/api/facilities/", middleware.Compression(
    middleware.ETag(
        middleware.CacheControl(yourHandler)
    )
))
```

### 1.4 Cache Warming Service ✅

**Files Created:**
- ✅ `internal/application/services/cache_warming_service.go` - Automatic cache warming

**Features:**
- Warms top 50 facilities on startup
- Warms first 3 pages of facility lists
- Periodic cache refresh (configurable)
- Cache statistics and monitoring

**Next Steps:**
```bash
# In your main.go:

warmingService := services.NewCacheWarmingService(
    facilityRepo,
    procedureRepo,
    cacheProvider,
)

// Warm cache on startup
warmingService.WarmCache(context.Background())

// Start periodic warming (every 5 minutes)
warmingService.StartPeriodicWarming(context.Background(), 5*time.Minute)
```

---

## Phase 2: Architectural Improvements (Week 2-3)

### 2.1 Read Replicas ✅

**Files Modified/Created:**
- ✅ `docker-compose.yml` - Added postgres-replica-1 and postgres-replica-2
- ✅ `internal/infrastructure/clients/postgres/multi_db_client.go` - Multi-DB connection manager
- ✅ `READ_REPLICA_SETUP.md` - Comprehensive setup guide

**Benefits:**
- 3x read capacity (1 primary + 2 replicas)
- Isolated write and read operations
- Improved availability

**Next Steps:**
```bash
# 1. Start services with replicas
docker-compose up -d postgres postgres-replica-1 postgres-replica-2

# 2. Verify replication
docker exec ppd_postgres psql -U postgres -c "SELECT * FROM pg_stat_replication;"

# 3. Update application to use MultiDBClient (see READ_REPLICA_SETUP.md)

# 4. Monitor replication lag
docker exec ppd_postgres psql -U postgres -c "
  SELECT 
    client_addr,
    application_name,
    state,
    replay_lag
  FROM pg_stat_replication;
"
```

### 2.2 Connection Pool Optimization ⬜

**Todo:**
- Update postgres client configuration
- Set optimal pool sizes based on replica count
- Implement connection health checks

**Recommended Settings:**
```go
MaxOpenConns: 100      // Per database instance
MaxIdleConns: 25       // Keep connections warm
ConnMaxLifetime: 5min  // Rotate connections
ConnMaxIdleTime: 1min  // Close idle connections
```

### 2.3 DataLoader for GraphQL (N+1 Prevention) ⬜

**Todo:**
- Implement DataLoader pattern in GraphQL resolvers
- Batch load facilities, procedures, insurance
- Add per-request caching

**Files to Create:**
- `internal/graphql/loaders/facility_loader.go`
- `internal/graphql/loaders/procedure_loader.go`
- `internal/graphql/loaders/insurance_loader.go`

**Benefits:**
- Eliminates N+1 queries
- 10-100x improvement for nested GraphQL queries

### 2.4 Query Complexity Analysis ⬜

**Todo:**
- Add query depth limiting (max 5 levels)
- Implement cost-based rate limiting
- Add query timeout enforcement

---

## Phase 3: Advanced Optimizations (Week 4+)

### 3.1 CDN Integration ⬜

**Todo:**
- Set up CloudFlare or AWS CloudFront
- Cache static assets (images, maps)
- Cache API responses at edge locations
- Configure geo-routing

### 3.2 Materialized Views ⬜

**Todo:**
- Create materialized views for analytics
- Implement refresh strategy
- Add to cache warming service

**Example Views:**
- Popular procedures by region
- Top-rated facilities
- Insurance acceptance statistics

### 3.3 APQ (Automatic Persisted Queries) ⬜

**Todo:**
- Implement APQ in GraphQL server
- Cache query strings
- Send only query hash in requests

**Benefits:**
- Reduced request size (80-95%)
- Lower bandwidth costs
- Faster parsing

### 3.4 Geographic Distribution ⬜

**Todo:**
- Deploy read replicas in multiple regions
- Implement geo-routing
- Use edge functions for simple queries

---

## Testing & Validation

### Performance Benchmarks

**Baseline Metrics (Before Optimization):**
```bash
# Capture current performance
ab -n 1000 -c 10 http://localhost:8080/api/facilities/
# Note: Requests/sec, Time per request, Transfer rate

# Check current cache hit ratio
docker exec ppd_redis redis-cli info stats | grep keyspace
```

**Expected Improvements:**

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Avg Response Time | 200-500ms | 50-100ms | 50-80% |
| p95 Response Time | 800ms | 200ms | 75% |
| Cache Hit Ratio | 40% | 80%+ | 100% |
| Requests/sec | 100 | 400-500 | 4-5x |
| DB Connections Used | 80/100 | 30/100 (per instance) | 62% reduction |

### Load Testing

```bash
# Install k6 for load testing
brew install k6  # macOS
# or
apt-get install k6  # Linux

# Run load test
k6 run --vus 100 --duration 5m scripts/load_test.js

# Monitor during test
watch -n 1 'docker stats'
docker exec ppd_redis redis-cli info stats | grep keyspace_hits
```

### Monitoring Setup

**Metrics to Track:**

1. **Cache Performance**
   - Hit ratio (target: >80%)
   - Miss rate
   - Eviction rate
   - Memory usage

2. **Database Performance**
   - Query response time (p50, p95, p99)
   - Connection pool utilization
   - Slow query count (>100ms)
   - Replication lag (target: <100ms)

3. **API Performance**
   - Response time by endpoint
   - Error rate
   - Request rate
   - Payload size (before/after compression)

4. **System Resources**
   - CPU utilization
   - Memory usage
   - Network I/O
   - Disk I/O

---

## Rollout Strategy

### Development
1. ✅ Implement all changes
2. ⬜ Test locally with docker-compose
3. ⬜ Run performance benchmarks
4. ⬜ Verify cache hit ratios
5. ⬜ Test replication lag

### Staging
1. ⬜ Deploy with feature flags
2. ⬜ Run load tests
3. ⬜ Monitor for 48 hours
4. ⬜ Compare metrics with baseline
5. ⬜ Test failover scenarios

### Production
1. ⬜ Enable caching (10% traffic)
2. ⬜ Monitor for 24 hours
3. ⬜ Increase to 50% traffic
4. ⬜ Monitor for 24 hours
5. ⬜ Enable for 100% traffic
6. ⬜ Add read replicas
7. ⬜ Route reads to replicas (gradual rollout)

---

## Troubleshooting

### Cache Issues

**Problem:** Low cache hit ratio (<50%)
**Solutions:**
- Increase TTL for stable data
- Implement cache warming
- Check cache key patterns
- Monitor cache memory

**Problem:** Stale cache data
**Solutions:**
- Verify cache invalidation on writes
- Reduce TTL
- Implement cache versioning
- Add manual invalidation endpoint

### Database Issues

**Problem:** High replication lag (>1s)
**Solutions:**
- Check network latency
- Increase replica resources
- Optimize write queries
- Consider synchronous replication

**Problem:** Connection pool exhaustion
**Solutions:**
- Increase MaxOpenConns
- Add more replicas
- Implement connection retry logic
- Optimize slow queries

### API Issues

**Problem:** High response times despite caching
**Solutions:**
- Check database query performance
- Verify indexes are used (EXPLAIN ANALYZE)
- Enable compression
- Optimize serialization

---

## Success Criteria

✅ **Phase 1 Complete When:**
- [ ] Cache hit ratio >80%
- [ ] Average response time <100ms
- [ ] All indexes created and used
- [ ] Compression enabled on all responses

✅ **Phase 2 Complete When:**
- [ ] Read replicas operational
- [ ] Replication lag <100ms
- [ ] 80%+ reads hitting replicas
- [ ] Connection pools optimized

✅ **Phase 3 Complete When:**
- [ ] CDN caching operational
- [ ] Geographic distribution (if applicable)
- [ ] Materialized views created
- [ ] APQ implemented

---

## Next Actions (Priority Order)

1. **Immediate (Today)**
   - [ ] Test new cache methods with unit tests
   - [ ] Apply performance middleware to API routes
   - [ ] Run index migration on development database

2. **This Week**
   - [ ] Integrate CachedFacilityAdapter in main application
   - [ ] Start cache warming service
   - [ ] Set up monitoring dashboards
   - [ ] Run baseline performance benchmarks

3. **Next Week**
   - [ ] Deploy to staging environment
   - [ ] Enable read replicas
   - [ ] Run load tests
   - [ ] Compare metrics with baseline

4. **Following Weeks**
   - [ ] Implement DataLoader pattern
   - [ ] Add query complexity analysis
   - [ ] Plan CDN integration
   - [ ] Create materialized views

---

## Resources

- [READ_PERFORMANCE_OPTIMIZATION.md](./READ_PERFORMANCE_OPTIMIZATION.md) - Full strategy document
- [READ_REPLICA_SETUP.md](./READ_REPLICA_SETUP.md) - Read replica configuration guide
- [PostgreSQL Performance Tuning](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [Redis Best Practices](https://redis.io/docs/manual/patterns/)
- [Go Database SQL Tutorial](https://go.dev/doc/database/index)

---

## Questions or Issues?

If you encounter issues during implementation:
1. Check the troubleshooting sections above
2. Review the monitoring metrics
3. Check application logs for errors
4. Verify database and cache connectivity

Good luck with the optimization! 🚀

