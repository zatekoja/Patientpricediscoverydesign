# Read Performance Optimization Strategy

## Executive Summary
This document outlines a comprehensive strategy to optimize read performance for this patient price discovery application, which follows a read-heavy workload pattern. The implementation focuses on caching, database optimization, read replicas, query optimization, and architectural patterns.

## Current Architecture Analysis

### ✅ Already Implemented
- Redis cache layer with basic Get/Set/Delete operations
- Typesense for search operations (separate read model)
- PostgreSQL as primary database
- CQRS pattern with separate query services
- GraphQL query service (port 8081) for reads
- REST API service (port 8080) for writes
- Docker containerization with health checks
- OpenTelemetry for observability

### 🔨 Performance Optimizations to Implement

---

## 1. Multi-Layer Caching Strategy

### 1.1 Application-Level Caching (Redis)

#### Enhanced Cache Implementation
- **Cache-Aside Pattern**: Check cache before database
- **Write-Through Pattern**: Update cache on writes
- **TTL Strategy**: Different TTLs for different data types
  - Hot data (facilities, procedures): 5-15 minutes
  - Warm data (search results): 2-5 minutes
  - Cold data (analytics): 30-60 minutes

#### Cache Key Strategy
```
facility:{id}                        # Single facility
facilities:list:{page}:{limit}       # Paginated lists
facilities:search:{hash(params)}     # Search results
procedures:{facility_id}             # Procedures by facility
insurance:accepted:{facility_id}     # Insurance by facility
```

### 1.2 Query Result Caching
- Cache complex aggregations
- Cache joined data (denormalized)
- Cache search results with pagination
- Implement cache warming for frequently accessed data

### 1.3 HTTP Response Caching
- Add ETag support for conditional requests
- Implement Cache-Control headers
- Add Last-Modified headers
- Support 304 Not Modified responses

---

## 2. Database Read Optimization

### 2.1 PostgreSQL Read Replicas

#### Benefits
- Distribute read load across multiple replicas
- Zero impact on write performance
- Geographic distribution for lower latency
- Improved availability and fault tolerance

#### Implementation Strategy
```yaml
Database Architecture:
┌─────────────────┐
│  Primary DB     │  ← All writes go here
│  (Master)       │
└────────┬────────┘
         │
         │ Streaming Replication
         ├────────────────┬─────────────
         ▼                ▼             ▼
┌────────────────┐ ┌────────────┐ ┌────────────┐
│ Read Replica 1 │ │ Replica 2  │ │ Replica 3  │
│ (Hot queries)  │ │ (Search)   │ │ (Reports)  │
└────────────────┘ └────────────┘ └────────────┘
```

#### Connection Routing
- Write operations → Primary database
- Simple reads → Round-robin across replicas
- Complex queries → Dedicated replica
- Search operations → Already using Typesense

### 2.2 Database Indexing Strategy

#### Critical Indexes to Add/Optimize
```sql
-- Composite indexes for common queries
CREATE INDEX idx_facilities_type_active_rating 
  ON facilities(facility_type, is_active, rating DESC);

-- Covering indexes (include frequently accessed columns)
CREATE INDEX idx_facilities_search 
  ON facilities(is_active, facility_type) 
  INCLUDE (name, rating, review_count);

-- Partial indexes for hot data
CREATE INDEX idx_active_facilities 
  ON facilities(id) 
  WHERE is_active = true;

-- GiST index for geospatial queries
CREATE INDEX idx_facilities_location_gist 
  ON facilities USING GIST(ST_MakePoint(longitude, latitude));

-- B-tree index for range queries on prices
CREATE INDEX idx_facility_procedures_price_range 
  ON facility_procedures(price, facility_id) 
  WHERE is_available = true;
```

### 2.3 Query Optimization

#### Use Connection Pooling
```go
// Optimize pool settings
MaxOpenConns: 100      // Based on replica count
MaxIdleConns: 25       // Keep connections warm
ConnMaxLifetime: 5min  // Rotate connections
ConnMaxIdleTime: 1min  // Close idle connections
```

#### Prepared Statements
- Use prepared statements for repeated queries
- Reduce parsing overhead
- Improve query plan caching

#### Query Patterns
- Use LIMIT/OFFSET pagination efficiently (or cursor-based)
- Avoid SELECT *; fetch only needed columns
- Use EXISTS instead of COUNT(*) for existence checks
- Leverage materialized views for complex aggregations

---

## 3. GraphQL/API Optimizations

### 3.1 DataLoader Pattern (N+1 Query Prevention)

```go
// Batch loading for related data
type DataLoaders struct {
    FacilityLoader    *dataloader.Loader
    ProcedureLoader   *dataloader.Loader
    InsuranceLoader   *dataloader.Loader
}

// Single query instead of N queries
facilities := batchLoadFacilities(ids)
```

### 3.2 Query Complexity Analysis
- Limit query depth (max 5 levels)
- Set maximum field count per query
- Implement query cost analysis
- Rate limiting based on query complexity

### 3.3 Field-Level Caching
- Cache individual resolver results
- Use cache hints in GraphQL schema
- Implement APQ (Automatic Persisted Queries)

---

## 4. Search Optimization (Typesense)

### 4.1 Denormalized Collections
```javascript
// Store joined data in Typesense for zero-join queries
{
  id: "fac-123",
  name: "General Hospital",
  location: [37.7749, -122.4194],
  procedures: [
    {id: "proc-1", name: "X-Ray", price: 150},
    {id: "proc-2", name: "CT Scan", price: 500}
  ],
  insurance_accepted: ["ins-1", "ins-2", "ins-3"],
  avg_rating: 4.5,
  total_reviews: 234
}
```

### 4.2 Search Indexing Strategy
- Update Typesense asynchronously after DB writes
- Use bulk indexing for initial data load
- Implement incremental updates via event stream
- Add caching layer in front of Typesense

---

## 5. Content Delivery Network (CDN)

### 5.1 Static Asset Caching
- Cache facility images
- Cache static maps/tiles
- Cache API responses for public data
- Use CloudFlare or AWS CloudFront

### 5.2 Edge Caching
- Deploy read replicas geographically
- Use edge functions for simple queries
- Cache at CDN level with appropriate TTLs

---

## 6. Response Compression

### 6.1 HTTP Compression
```go
// Gzip compression for all responses
middleware.Compress(5) // compression level
```

### 6.2 Response Size Optimization
- Implement field filtering (sparse fieldsets)
- Paginate large lists
- Use cursor-based pagination for better performance

---

## 7. Monitoring & Metrics

### 7.1 Key Metrics to Track
- **Cache Hit Ratio**: Target > 80%
- **Query Response Time**: p50, p95, p99
- **Database Connection Pool**: Utilization
- **Read Replica Lag**: Keep < 100ms
- **API Response Time**: Target < 200ms
- **Typesense Query Time**: Target < 50ms

### 7.2 Performance Testing
- Load test with realistic read patterns
- Benchmark common query patterns
- Monitor slow queries (> 100ms)
- Set up alerts for degradation

---

## 8. Implementation Priority

### Phase 1: Quick Wins (Week 1)
1. ✅ Add cache-aside pattern to all read handlers
2. ✅ Implement cache warming for hot data
3. ✅ Add database indexes (listed above)
4. ✅ Enable HTTP compression
5. ✅ Add ETag support for conditional requests
6. ✅ Optimize connection pool settings

### Phase 2: Architectural (Week 2-3)
1. ✅ Implement DataLoader pattern in GraphQL
2. ✅ Add read replicas for PostgreSQL
3. ✅ Implement connection routing logic
4. ✅ Add query complexity analysis
5. ✅ Enhance Typesense collections with denormalized data

### Phase 3: Advanced (Week 4+)
1. Add CDN for static assets
2. Implement materialized views for analytics
3. Add APQ (Automatic Persisted Queries)
4. Geographic read replica distribution
5. Implement rate limiting per user/IP

---

## 9. Code Implementation

### 9.1 Enhanced Cache Provider Interface
```go
type CacheProvider interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl int) error
    GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)
    SetMulti(ctx context.Context, items map[string][]byte, ttl int) error
    Delete(ctx context.Context, key string) error
    DeletePattern(ctx context.Context, pattern string) error
    Exists(ctx context.Context, key string) (bool, error)
    TTL(ctx context.Context, key string) (time.Duration, error)
}
```

### 9.2 Repository with Caching
```go
func (r *FacilityRepository) GetByID(ctx context.Context, id string) (*Facility, error) {
    // Try cache first
    cacheKey := fmt.Sprintf("facility:%s", id)
    if cached, err := r.cache.Get(ctx, cacheKey); err == nil {
        var facility Facility
        if json.Unmarshal(cached, &facility) == nil {
            return &facility, nil
        }
    }
    
    // Cache miss - fetch from DB
    facility, err := r.db.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // Update cache asynchronously
    go func() {
        if data, err := json.Marshal(facility); err == nil {
            r.cache.Set(context.Background(), cacheKey, data, 300) // 5 min TTL
        }
    }()
    
    return facility, nil
}
```

### 9.3 Database Read/Write Splitting
```go
type DBConnections struct {
    Primary      *sql.DB  // For writes
    ReadReplicas []*sql.DB // For reads
    rrIndex      int32    // Round-robin index
}

func (db *DBConnections) GetReadConnection() *sql.DB {
    if len(db.ReadReplicas) == 0 {
        return db.Primary // Fallback to primary
    }
    idx := atomic.AddInt32(&db.rrIndex, 1)
    return db.ReadReplicas[idx % int32(len(db.ReadReplicas))]
}
```

---

## 10. Expected Performance Improvements

### Before Optimization
- Average read latency: 200-500ms
- Cache hit ratio: ~40%
- Database load: High on single instance
- P95 response time: 800ms

### After Optimization
- Average read latency: 50-100ms (50-80% improvement)
- Cache hit ratio: >80% (100% improvement)
- Database load: Distributed across replicas (75% reduction per instance)
- P95 response time: 200ms (75% improvement)

### Cost-Benefit Analysis
- Redis cache: Low cost, high impact
- Database indexes: Zero cost, medium-high impact
- Read replicas: Medium cost, high impact
- CDN: Medium cost, medium impact (geographic)
- DataLoader: Zero cost, high impact (N+1 prevention)

---

## 11. Testing Strategy

### Performance Tests
```bash
# Load testing with k6
k6 run --vus 100 --duration 5m read-test.js

# Cache hit ratio measurement
redis-cli info stats | grep keyspace_hits

# Slow query analysis
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
WHERE mean_exec_time > 100 
ORDER BY mean_exec_time DESC;
```

### Benchmarks to Establish
- Baseline: Current performance metrics
- Per optimization: Measure impact
- Final: Compare against baseline

---

## 12. Rollout Plan

1. **Enable monitoring** - Before any changes
2. **Index creation** - During low-traffic period
3. **Cache implementation** - Gradual rollout with feature flags
4. **Read replicas** - Test with 10% traffic, then increase
5. **DataLoader** - Deploy to staging first
6. **Validation** - Monitor for 1 week before next phase

---

## 13. Rollback Strategy

- Keep monitoring dashboards visible
- Set automated alerts for regressions
- Implement feature flags for each optimization
- Have rollback procedures documented
- Keep database backups before index changes

---

## Conclusion

This read-heavy optimization strategy focuses on:
1. **Multi-layer caching** (biggest impact)
2. **Database optimization** (indexes, replicas, query tuning)
3. **Architectural patterns** (CQRS, DataLoader, denormalization)
4. **CDN/Edge caching** (geographic performance)

Expected overall improvement: **3-5x faster reads** with **80%+ cache hit ratio**.

Implementation timeline: **2-4 weeks** for phases 1-2, with continuous improvements in phase 3.

