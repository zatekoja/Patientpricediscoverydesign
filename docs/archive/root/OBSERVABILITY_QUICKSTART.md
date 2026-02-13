# Observability Quick Start Guide

This guide will help you quickly set up and start using the comprehensive observability stack.

## Prerequisites

- Docker and Docker Compose installed
- At least 4GB of free RAM
- Ports available: 3301 (SigNoz), 4317-4318 (OTLP), 9187, 9121, 9216 (exporters)

## Step 1: Start the Stack

```bash
# Start all services including observability components
docker-compose up -d

# Check that all services are running
docker-compose ps
```

Expected services:
- `ppd_signoz_otel_collector` - OTLP collector
- `ppd_clickhouse` - Time-series database
- `ppd_signoz_query_service` - Query service
- `ppd_signoz_frontend` - Web UI
- `ppd_fluent_bit` - Log aggregator
- `ppd_postgres_exporter` - PostgreSQL metrics
- `ppd_redis_exporter` - Redis metrics
- `ppd_mongodb_exporter` - MongoDB metrics

## Step 2: Access SigNoz UI

Open your browser and navigate to:
```
http://localhost:3301
```

First-time setup:
1. Create an admin account
2. Set organization name
3. Complete the onboarding wizard

## Step 3: Verify Data Collection

### Check Traces
1. In SigNoz UI, go to "Services"
2. You should see:
   - `patient-price-discovery` (API)
   - `patient-price-discovery-graphql`
   - `patient-price-discovery-sse`
   - `patient-price-discovery-provider`

### Check Metrics
1. Go to "Dashboards"
2. Click "Import Dashboard"
3. Import dashboards from `vendor/dashboards/`:
   - `api-service.json`
   - `provider-service.json`
   - `database-health.json`
   - `infrastructure-overview.json`

### Check Logs
1. Go to "Logs"
2. You should see logs from all services
3. Try filtering by service name or log level

## Step 4: Generate Traffic

Generate some traffic to populate the dashboards:

```bash
# Test API endpoints
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/facilities

# Test GraphQL
curl http://localhost:8081/health

# Test Provider API
curl http://localhost:3001/health
```

## Step 5: Explore Observability Features

### View Service Dependencies
1. Go to "Services" → Select a service
2. Click on "Dependency Graph"
3. See how services interact

### Analyze Traces
1. Go to "Traces"
2. Filter by service, operation, or status code
3. Click on a trace to see the flame graph
4. View logs correlated with traces

### Monitor Metrics
1. Go to "Dashboards"
2. Open "Infrastructure Overview"
3. Monitor:
   - Request rate
   - Error rate
   - Latency percentiles
   - Resource utilization

### Query Logs
1. Go to "Logs"
2. Use filters:
   ```
   service_name=patient-price-discovery AND level=error
   ```
3. Click on a log entry to see correlated traces

## Common Queries

### Find Slow Requests
In Traces, filter:
```
duration > 1000ms AND status_code=200
```

### Find Errors
In Logs, query:
```
level=error AND timestamp > now-1h
```

### Database Performance
In Metrics, query:
```
histogram_quantile(0.95, rate(db_query_duration_bucket[5m]))
```

### Cache Hit Ratio
In Metrics, query:
```
(sum(rate(cache_hit_count[5m])) / 
(sum(rate(cache_hit_count[5m])) + sum(rate(cache_miss_count[5m])))) * 100
```

## Troubleshooting

### No Data Appearing

1. **Check OTEL Configuration**
   ```bash
   docker logs ppd_api | grep -i otel
   docker logs ppd_provider_api | grep -i otel
   ```
   Look for "OpenTelemetry initialized successfully"

2. **Check Collector**
   ```bash
   docker logs ppd_signoz_otel_collector
   ```
   Should show received spans/metrics

3. **Check Service Environment Variables**
   ```bash
   docker exec ppd_api env | grep OTEL
   ```

### High Memory Usage

1. **Check ClickHouse**
   ```bash
   docker stats ppd_clickhouse
   ```
   Recommended: Increase Docker memory limit to 4GB+

2. **Adjust Retention**
   Edit retention policies in SigNoz UI

### Missing Logs

1. **Check Fluent Bit**
   ```bash
   docker logs ppd_fluent_bit
   ```
   Should show log forwarding activity

2. **Verify Log Format**
   Services should output JSON logs in production mode

### Database Exporter Issues

1. **Check Exporter Health**
   ```bash
   curl http://localhost:9187/metrics  # PostgreSQL
   curl http://localhost:9121/metrics  # Redis
   curl http://localhost:9216/metrics  # MongoDB
   ```

2. **Check Database Connectivity**
   ```bash
   docker logs ppd_postgres_exporter
   ```

## Performance Tuning

### Reduce Sampling Rate
In production, adjust trace sampling:

```yaml
# In signoz-otel-collector-config.yaml
processors:
  probabilistic_sampler:
    sampling_percentage: 1.0  # 1% sampling
```

### Adjust Metric Collection Interval
```yaml
# In docker-compose.yml for exporters
- '--collect-interval=60s'  # Collect every 60 seconds
```

### Configure Log Retention
In SigNoz UI:
1. Go to Settings → Retention
2. Set retention periods:
   - Traces: 7 days
   - Metrics: 30 days
   - Logs: 7 days

## Next Steps

1. **Create Alerts**
   - Go to "Alerts" in SigNoz
   - Create alert rules for critical metrics
   - Configure notification channels (Slack, email, PagerDuty)

2. **Create Custom Dashboards**
   - Start with provided dashboards
   - Customize for your specific needs
   - Add business-specific metrics

3. **Instrument Custom Code**
   - Add custom metrics for business logic
   - Create manual spans for important operations
   - Add structured log fields for better filtering

4. **Set Up Continuous Profiling**
   - Enable profiling for CPU-intensive operations
   - Analyze memory allocation patterns
   - Optimize hot code paths

## Reference

- Main documentation: [OBSERVABILITY.md](./OBSERVABILITY.md)
- Dashboard definitions: `vendor/dashboards/`
- Configuration files: `backend/signoz-otel-collector-config.yaml`, `backend/fluent-bit.conf`

## Support

For issues or questions:
1. Check logs: `docker-compose logs [service-name]`
2. Review configuration files
3. Consult [SigNoz documentation](https://signoz.io/docs/)
4. Check [OpenTelemetry documentation](https://opentelemetry.io/docs/)

