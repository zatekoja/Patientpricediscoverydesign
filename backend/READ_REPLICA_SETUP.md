# Read Replica Configuration Guide

## Overview
This guide explains how to configure and use PostgreSQL read replicas for improved read performance in the Patient Price Discovery application.

## Architecture

```
┌─────────────────┐
│  Primary DB     │  ← All WRITE operations
│  (Port 5432)    │
└────────┬────────┘
         │ Streaming Replication
         ├────────────────┬─────────────
         ▼                ▼             ▼
┌────────────────┐ ┌────────────┐ ┌────────────┐
│ Read Replica 1 │ │ Replica 2  │ │ Replica N  │
│ (Port 5433)    │ │(Port 5434) │ │(Port 543N) │
└────────────────┘ └────────────┘ └────────────┘
         ▲                ▲             ▲
         │                │             │
         └────────────────┴─────────────┘
              All READ operations (round-robin)
```

## Local Development Setup

### 1. Docker Compose Configuration

The `docker-compose.yml` includes:
- **postgres**: Primary database (port 5432)
- **postgres-replica-1**: First read replica (port 5433)
- **postgres-replica-2**: Second read replica (port 5434)

### 2. Starting Services

```bash
# Start all services including replicas
docker-compose up -d

# Verify replication status on primary
docker exec ppd_postgres psql -U postgres -c "SELECT * FROM pg_stat_replication;"

# Check replica status
docker exec ppd_postgres_replica_1 psql -U postgres -c "SELECT pg_is_in_recovery();"
# Should return 't' (true) indicating it's a replica
```

### 3. Application Configuration

Set environment variables for replica connections:

```bash
# Primary database (writes)
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=patient_price_discovery

# Read replicas (reads)
export DB_READ_REPLICA_1_HOST=localhost
export DB_READ_REPLICA_1_PORT=5433

export DB_READ_REPLICA_2_HOST=localhost
export DB_READ_REPLICA_2_PORT=5434

# Connection pool settings
export DB_MAX_OPEN_CONNS=100
export DB_MAX_IDLE_CONNS=25
export DB_CONN_MAX_LIFETIME=5m
export DB_CONN_MAX_IDLE_TIME=1m
```

## Production Setup

### AWS RDS Configuration

```yaml
Primary Instance:
  - Instance: db.r6g.xlarge (4 vCPU, 32 GB RAM)
  - Multi-AZ: Yes (for high availability)
  - Backup retention: 7 days

Read Replicas:
  - Count: 2-3 depending on load
  - Instance: db.r6g.large (2 vCPU, 16 GB RAM)
  - Same region as primary (lower latency)
  - Can be in different availability zones

Connection Settings:
  - max_connections: 200 per instance
  - Application connection pool: 25-50 per instance
```

### Environment Variables (Production)

```bash
# Primary (from RDS endpoint)
DB_HOST=ppd-primary.xxxxx.us-east-1.rds.amazonaws.com
DB_PORT=5432

# Read Replicas (from RDS read replica endpoints)
DB_READ_REPLICA_1_HOST=ppd-replica-1.xxxxx.us-east-1.rds.amazonaws.com
DB_READ_REPLICA_1_PORT=5432

DB_READ_REPLICA_2_HOST=ppd-replica-2.xxxxx.us-east-1.rds.amazonaws.com
DB_READ_REPLICA_2_PORT=5432
```

## Application Code Usage

### Using MultiDBClient

```go
import "github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"

// Create multi-DB client configuration
config := postgres.MultiDBConfig{
    PrimaryConfig: postgres.Config{
        Host:     os.Getenv("DB_HOST"),
        Port:     5432,
        User:     os.Getenv("DB_USER"),
        Password: os.Getenv("DB_PASSWORD"),
        DBName:   os.Getenv("DB_NAME"),
        SSLMode:  "require",
    },
    ReplicaConfigs: []postgres.Config{
        {
            Host:     os.Getenv("DB_READ_REPLICA_1_HOST"),
            Port:     5433,
            User:     os.Getenv("DB_USER"),
            Password: os.Getenv("DB_PASSWORD"),
            DBName:   os.Getenv("DB_NAME"),
            SSLMode:  "require",
        },
        {
            Host:     os.Getenv("DB_READ_REPLICA_2_HOST"),
            Port:     5434,
            User:     os.Getenv("DB_USER"),
            Password: os.Getenv("DB_PASSWORD"),
            DBName:   os.Getenv("DB_NAME"),
            SSLMode:  "require",
        },
    },
    MaxOpenConns:    100,
    MaxIdleConns:    25,
    ConnMaxLifetime: 5 * time.Minute,
    ConnMaxIdleTime: 1 * time.Minute,
}

// Create client
dbClient, err := postgres.NewMultiDBClient(config)
if err != nil {
    log.Fatal(err)
}
defer dbClient.Close()

// Use primary for writes
_, err = dbClient.Primary().ExecContext(ctx, "INSERT INTO facilities ...")

// Use replicas for reads (automatic round-robin)
rows, err := dbClient.Read().QueryContext(ctx, "SELECT * FROM facilities ...")
```

## Monitoring

### Replication Lag

Monitor replication lag to ensure replicas are up-to-date:

```sql
-- On primary
SELECT 
    client_addr,
    application_name,
    state,
    sync_state,
    replay_lag
FROM pg_stat_replication;
```

**Target**: Keep replay_lag < 100ms for real-time applications

### Connection Pool Stats

```go
stats := dbClient.Stats()
fmt.Printf("Primary: OpenConns=%d, InUse=%d\n", 
    stats.Primary.OpenConnections, 
    stats.Primary.InUse)

for i, replica := range stats.Replicas {
    fmt.Printf("Replica %d: OpenConns=%d, InUse=%d\n", 
        i+1, 
        replica.OpenConnections, 
        replica.InUse)
}
```

### Key Metrics to Track

1. **Replication Lag**: < 100ms
2. **Connection Pool Utilization**: 50-80%
3. **Query Response Time**: p95 < 100ms
4. **Read/Write Ratio**: Should be 80%+ reads for this app
5. **Replica Health**: All replicas should be online

## Troubleshooting

### Replica Not Catching Up

```bash
# Check if replica is receiving WAL
docker exec ppd_postgres_replica_1 psql -U postgres -c \
  "SELECT pg_last_wal_receive_lsn(), pg_last_wal_replay_lsn();"

# Restart replica if stuck
docker restart ppd_postgres_replica_1
```

### High Replication Lag

**Causes**:
- Heavy write load on primary
- Network latency between primary and replica
- Insufficient replica resources

**Solutions**:
- Increase replica instance size
- Enable synchronous replication for critical replicas
- Optimize write queries on primary

### Connection Pool Exhaustion

**Symptoms**:
- "Too many connections" errors
- Slow query responses

**Solutions**:
- Increase `MaxOpenConns` in application
- Increase `max_connections` in PostgreSQL
- Add more read replicas
- Implement connection retry logic

## Best Practices

1. **Read Replica Placement**
   - Same region as application (lower latency)
   - Different availability zones (high availability)
   - Geographic distribution for global apps

2. **Connection Management**
   - Use connection pooling
   - Set appropriate timeouts
   - Implement health checks
   - Graceful failover to primary if all replicas down

3. **Query Routing**
   - Route ALL reads to replicas
   - Route ALL writes to primary
   - Use cache for hot data (reduce DB load)
   - Consider eventual consistency for some reads

4. **Scaling Strategy**
   - Start with 2 read replicas
   - Add more replicas based on load (horizontal scaling)
   - Monitor connection pool saturation
   - Consider read replica promotion for failover

## Performance Expectations

### Before Read Replicas
- All queries hit primary database
- Write operations compete with read operations
- Limited connection pool (100 connections)
- Single point of contention

### After Read Replicas
- 80%+ of queries hit replicas
- Write and read operations isolated
- 3x connection pool capacity (100 + 100 + 100)
- Load distributed across multiple instances

### Expected Improvements
- **Read Latency**: 40-60% reduction
- **Write Performance**: 30-50% improvement (less contention)
- **Throughput**: 3-5x increase in read capacity
- **Availability**: Higher (can lose replicas without impact)

## Cost Considerations

### AWS RDS Pricing Example (us-east-1)

- Primary (db.r6g.xlarge): ~$350/month
- Read Replica (db.r6g.large): ~$175/month each
- Total for 1 primary + 2 replicas: ~$700/month

**ROI**: 3-5x performance improvement for 2x cost = good value for read-heavy apps

### Cost Optimization Tips
1. Use smaller instance types for low-traffic replicas
2. Schedule replicas (dev/staging environments)
3. Use Reserved Instances for 1-3 year commitments (40-60% savings)
4. Monitor and right-size based on actual usage

## Next Steps

1. ✅ Configure read replicas in docker-compose
2. ✅ Implement MultiDBClient in application
3. ✅ Update repositories to use read replicas
4. ⬜ Deploy to staging and test
5. ⬜ Monitor replication lag and query performance
6. ⬜ Deploy to production with gradual rollout
7. ⬜ Set up alerts for replication issues

