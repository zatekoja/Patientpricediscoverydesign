# Procedure Enrichment During Ingestion - Quick Reference

## What Changed

Service enrichment now happens **during ingestion** instead of **on-demand**. This is more efficient since procedure descriptions rarely change.

## Key Files Modified

| File | Change |
|------|--------|
| `internal/application/services/provider_ingestion_service.go` | Added batch enrichment after ingestion completes |
| `internal/application/services/procedure_enrichment_service.go` | Changed to return only cached enrichment (no on-demand generation) |
| `cmd/api/main.go` | Pass enrichment dependencies to ingestion service |
| `tests/integration/provider_ingestion_integration_test.go` | Updated test to include enrichment adapter |

## How It Works

### 1. Ingestion Process
```
POST /api/provider/ingest

ProviderIngestionService.SyncCurrentData(providerID)
├─ Fetch procedures from provider
├─ Store in database
└─ enrichProceduresBatch()  ← NEW: runs after ingestion
    ├─ Find procedures without enrichment
    ├─ Call LLM for each procedure
    └─ Store enrichment data
```

### 2. Query Process
```
GET /api/procedures/{id}/enrichment

ProcedureHandler.GetEnrichment(procedureID)
└─ Return cached enrichment from database
```

## Ingestion Response

The ingestion endpoint now includes enrichment metrics:

```json
{
  "records_processed": 500,
  "facilities_created": 45,
  "procedures_created": 150,
  "procedure_enrichments_created": 145,
  "procedure_enrichments_failed": 5
}
```

- **procedure_enrichments_created**: Number of procedures successfully enriched
- **procedure_enrichments_failed**: Failed enrichment attempts (will be retried on next ingestion)

## Configuration

Enrichment requires LLM provider configuration:

```bash
# Set LLM provider (openai, claude, etc.)
export PROCEDURE_ENRICHMENT_PROVIDER=openai
export OPENAI_API_KEY=sk-...
export OPENAI_API_ENDPOINT=https://api.openai.com/v1/chat/completions
```

If not configured, enrichment is **skipped** (logged) and ingestion continues normally.

## Behavior Changes

### Before
```go
// On-demand generation
GET /procedures/{id}/enrichment
→ Check cache
→ If missing: call LLM API (wait time!)
→ Store and return
```

### After
```go
// Cached only
POST /provider/ingest
→ Ingestion completes
→ Batch enrich all procedures (async)
→ Store to database

GET /procedures/{id}/enrichment
→ Return from cache immediately
→ If not found: return error
```

## Error Handling

### Enrichment Not Found
```
{
  "error": "enrichment not found for procedure {id}",
  "message": "enrichment should be generated during ingestion"
}
```

**Solution**: Re-run ingestion to generate enrichment for all procedures.

### Enrichment Provider Error
- **Logged**: Failed enrichments are logged with procedure ID
- **Retry**: Run ingestion again to retry failed procedures
- **Continue**: Ingestion doesn't fail if enrichment fails (graceful degradation)

## Testing

Run integration test to verify enrichment works:

```bash
cd backend
go test -v -run TestProviderIngestionServiceIntegration ./tests/integration
```

Expected output:
```
procedure_enrichments_created: ~95-99% of procedures
procedure_enrichments_failed: ~1-5% (if any)
```

## Monitoring

Track enrichment success in logs:

```bash
# Look for these log messages
"batch enriched 145 procedures (5 failed)"

# Or check response
curl -s http://localhost:8080/api/provider/ingest?provider_id=file_price_list | jq .procedure_enrichments_created
```

## Rollback

If you need to revert to on-demand enrichment:

1. Comment out the enrichment call in `provider_ingestion_service.go`:
   ```go
   // enrichmentSummary := s.enrichProceduresBatch(ctx)
   ```

2. Restore `ProcedureEnrichmentService.GetEnrichment()` to generate on-demand

3. Rebuild and deploy

## Performance Impact

- **Ingestion time**: +10-30% (depends on LLM API latency)
- **Query time**: -95% (cached data only, no LLM calls)
- **LLM API calls**: -99% (called once per procedure instead of per request)

## Next Steps

1. Run ingestion to enrich existing procedures
2. Monitor enrichment success in logs
3. Adjust LLM provider/model if needed
4. Consider scheduling periodic re-enrichment if procedures are frequently updated
