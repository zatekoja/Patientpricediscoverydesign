# Observability Implementation Guide

This document describes the comprehensive observability setup for the Patient Price Discovery system, including metrics, traces, logs, and dashboards.

## Overview

The observability stack consists of:
- **OpenTelemetry**: Unified telemetry collection (traces, metrics, logs)
- **SigNoz**: Observability backend (stores and visualizes telemetry data)
- **Fluent Bit**: Log aggregation and forwarding
- **Database Exporters**: Metrics from PostgreSQL, Redis, MongoDB
- **Structured Logging**: Zerolog (Go) and Pino (TypeScript)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Application Services                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │   API    │  │ GraphQL  │  │   SSE    │  │ Provider │   │
│  │   (Go)   │  │   (Go)   │  │   (Go)   │  │   (TS)   │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
│       │             │             │             │           │
│       └─────────────┴─────────────┴─────────────┘           │
│                         │                                    │
│                    OTLP (gRPC/HTTP)                         │
└─────────────────────────┼──────────────────────────────────┘
                          │
┌─────────────────────────▼──────────────────────────────────┐
│              SigNoz OTEL Collector                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │   Traces     │  │   Metrics    │  │    Logs      │   │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘   │
└─────────┼──────────────────┼──────────────────┼──────────┘
          │                  │                  │
          └──────────────────┴──────────────────┘
                          │
┌─────────────────────────▼──────────────────────────────────┐
│                     ClickHouse                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │signoz_traces │  │signoz_metrics│  │signoz_logs   │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
└────────────────────────────────────────────────────────────┘
          │
┌─────────▼──────────────────────────────────────────────────┐
│              SigNoz Query Service & Frontend                │
└────────────────────────────────────────────────────────────┘
```

## Components

### 1. Go Services (API, GraphQL, SSE)

**Instrumentation:**
- OpenTelemetry SDK with gRPC exporter
- Runtime metrics collection (memory, goroutines, GC)
- Structured logging with zerolog
- Custom application metrics

**Metrics Collected:**
- HTTP request rate, latency, status codes
- Database query duration
- Cache hit/miss ratio
- Go runtime: memory allocation, heap objects, GC pauses, goroutines
- Custom business metrics

**Traces:**
- Automatic HTTP request tracing
- Manual span creation for business logic
- Trace context propagation across services

**Logs:**
- Structured JSON logs (production)
- Pretty console logs (development)
- Trace ID correlation for log-trace linking

### 2. TypeScript Provider Service

**Instrumentation:**
- OpenTelemetry SDK with HTTP exporter
- Node.js auto-instrumentation
- Host metrics for Node.js runtime
- Structured logging with Pino

**Metrics Collected:**
- HTTP request metrics
- Provider sync operations
- LLM tag generation metrics
- Data quality metrics
- Node.js runtime: heap usage, event loop lag, GC duration
- MongoDB operation metrics

**Traces:**
- Automatic Express instrumentation
- MongoDB query tracing
- HTTP client tracing

**Logs:**
- Structured JSON logs with trace correlation
- Pretty printing in development

### 3. Database Monitoring

**PostgreSQL Exporter:**
- Active connections
- Transaction rate
- Query performance
- Cache hit ratio
- Database size
- Deadlocks

**Redis Exporter:**
- Connected clients
- Operations per second
- Memory usage
- Cache hit rate
- Evicted keys
- Key count

**MongoDB Exporter:**
- Active connections
- Operations per second
- Query execution time
- Document operations
- Memory usage
- Collection count

### 4. Log Aggregation (Fluent Bit)

**Log Sources:**
- Docker container logs
- Application stdout/stderr
- Database logs

**Processing:**
- JSON parsing
- Service tagging
- Component labeling
- Enrichment with metadata

**Output:**
- OTLP HTTP to SigNoz collector
- Stdout for debugging

## Configuration

### Environment Variables

#### Go Services
```bash
OTEL_ENABLED=true
OTEL_SERVICE_NAME=patient-price-discovery
OTEL_SERVICE_VERSION=1.0.0
OTEL_ENDPOINT=signoz-otel-collector:4317
ENV=production  # or development
LOG_LEVEL=info
```

#### TypeScript Services
```bash
OTEL_ENABLED=true
OTEL_SERVICE_NAME=patient-price-discovery-provider
OTEL_SERVICE_VERSION=1.0.0
OTEL_ENDPOINT=http://signoz-otel-collector:4318
NODE_ENV=production  # or development
LOG_LEVEL=info
```

## Dashboards

Pre-built dashboards are available in `vendor/dashboards/`:

### 1. API Service Dashboard (`api-service.json`)
- Request rate, error rate, latency
- Database performance
- Cache performance
- Go runtime metrics
- Endpoint-level metrics

### 2. Provider Service Dashboard (`provider-service.json`)
- Request metrics
- Provider sync operations
- LLM tag generation
- Node.js runtime metrics
- Data quality metrics
- Capacity management

### 3. Database Health Dashboard (`database-health.json`)
- PostgreSQL metrics
- Redis metrics
- MongoDB metrics
- System resource utilization

### 4. Infrastructure Overview (`infrastructure-overview.json`)
- Service status (up/down)
- Request overview across all services
- Error budget tracking
- Resource utilization
- Error tracking
- Application-specific metrics

## Importing Dashboards

To import dashboards into SigNoz:

1. Open SigNoz UI at `http://localhost:3301`
2. Navigate to "Dashboards"
3. Click "Import Dashboard"
4. Upload the JSON file from `vendor/dashboards/`
5. Save and configure

## Metrics Reference

### HTTP Metrics
- `http.server.request.count` - Total HTTP requests
- `http.server.request.duration` - Request latency histogram
- `http.server.active_requests` - Currently active requests

### SSE Metrics
- `sse.active_connections` - Number of active SSE connections

### Database Metrics
- `db.query.duration` - Database query latency
- `cache.hit.count` - Cache hits
- `cache.miss.count` - Cache misses

### Go Runtime Metrics
- `go_memstats_alloc_bytes` - Allocated memory
- `go_goroutines` - Number of goroutines
- `go_gc_duration_seconds` - GC pause duration
- `go_memstats_heap_objects` - Heap objects

### Provider Metrics
- `provider.sync.count` - Provider sync operations
- `provider.sync.duration_ms` - Sync duration
- `provider.records.processed` - Records processed
- `provider.tag_generation.count` - Tag generation operations
- `provider.data.freshness_days` - Data freshness

### Node.js Runtime Metrics
- `process_resident_memory_bytes` - Process memory
- `nodejs_eventloop_lag_seconds` - Event loop lag
- `nodejs_active_handles` - Active handles
- `nodejs_gc_duration_seconds` - GC duration

## Accessing Observability Tools

### SigNoz
- URL: `http://localhost:3301`
- Features: Traces, Metrics, Logs, Dashboards, Alerts

### Database Exporters
- PostgreSQL: `http://localhost:9187/metrics`
- Redis: `http://localhost:9121/metrics`
- MongoDB: `http://localhost:9216/metrics`

## Best Practices

### Logging
- Use structured logging (zerolog/pino)
- Include trace IDs for correlation
- Log at appropriate levels (debug, info, warn, error)
- Avoid logging sensitive data (PII, credentials)

### Metrics
- Use descriptive metric names
- Add relevant labels (service, endpoint, status)
- Use histograms for latency measurements
- Use counters for cumulative values
- Use gauges for point-in-time values

### Tracing
- Create spans for significant operations
- Add relevant attributes to spans
- Propagate context across service boundaries
- Sample appropriately in production

### Dashboards
- Group related metrics together
- Use appropriate visualizations
- Set meaningful thresholds
- Add descriptions to panels
- Use variables for filtering

## Alerting

Create alerts in SigNoz for:
- High error rates (>1% of requests)
- High latency (p95 > 1000ms)
- Service downtime
- Database connection issues
- Memory pressure
- Disk space issues

## Troubleshooting

### No Metrics Appearing
1. Check `OTEL_ENABLED=true` is set
2. Verify OTEL endpoint connectivity
3. Check SigNoz collector logs: `docker logs ppd_signoz_otel_collector`
4. Verify metric exporters are running

### Missing Logs
1. Check Fluent Bit logs: `docker logs ppd_fluent_bit`
2. Verify log format is JSON
3. Check SigNoz log ingestion pipeline

### High Memory Usage
1. Check Go heap profiles
2. Monitor goroutine count
3. Review Node.js heap snapshots
4. Check for memory leaks in services

## Development vs Production

### Development
- Console logging with pretty formatting
- Full sampling (all traces)
- Shorter metric intervals
- Debug log level

### Production
- JSON logging
- Sampling (1% or adaptive)
- Standard metric intervals (60s)
- Info/warn log level
- Log retention policies
- Alert rules active

## Performance Impact

Expected overhead:
- Metrics collection: ~1-2% CPU
- Trace sampling (1%): <1% CPU
- Structured logging: ~2-3% CPU
- Total: ~3-6% CPU overhead

## Further Reading

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [SigNoz Documentation](https://signoz.io/docs/)
- [Fluent Bit Documentation](https://docs.fluentbit.io/)
- [Zerolog Documentation](https://github.com/rs/zerolog)
- [Pino Documentation](https://getpino.io/)

