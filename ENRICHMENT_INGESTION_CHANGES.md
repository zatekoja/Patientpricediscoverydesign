# Service Enrichment During Ingestion - Implementation Summary

## Overview
Moved service enrichment from on-demand generation (when a user queries) to batch generation during the ingestion process. This allows enrichment to be generated once and cached, rather than regenerating it every time a user accesses the data.

## Problem Statement
Previously, procedure enrichment (AI-generated descriptions, prep steps, risks, recovery info) was generated on-demand using the `GetEnrichment` endpoint. This meant:
- **Multiple generations**: Same procedure could be enriched multiple times
- **Latency**: Users experienced delays waiting for LLM enrichment
- **Cost**: Multiple LLM API calls for the same content
- **Inefficient**: Data rarely changes, but was re-enriched constantly

## Solution
Integrated enrichment directly into the provider ingestion pipeline:
1. During `ProviderIngestionService.SyncCurrentData()`, procedures are created/updated
2. After all ingestion completes, `enrichProceduresBatch()` enriches unenriched procedures
3. Enrichment is stored in the database once
4. On queries, `ProcedureEnrichmentService.GetEnrichment()` returns only cached data

## Changes Made

### 1. Updated `ProviderIngestionSummary` struct
**File**: `internal/application/services/provider_ingestion_service.go`

Added enrichment tracking to the ingestion summary:
```go
ProcedureEnrichmentsCreated int `json:"procedure_enrichments_created"`
ProcedureEnrichmentsFailed  int `json:"procedure_enrichments_failed"`
```

### 2. Extended `ProviderIngestionService`
**File**: `internal/application/services/provider_ingestion_service.go`

Added dependencies:
- `enrichmentRepo repositories.ProcedureEnrichmentRepository` - stores enrichments
- `enrichmentProvider providers.ProcedureEnrichmentProvider` - generates enrichments (OpenAI, Claude, etc.)

Updated constructor to accept these new parameters (optional - nil if enrichment disabled).

### 3. Integrated Batch Enrichment
**File**: `internal/application/services/provider_ingestion_service.go`

Added new method `enrichProceduresBatch()`:
- Runs after all facility/procedure ingestion completes
- Iterates through all procedures in the database
- Skips procedures that already have enrichment
- Calls the enrichment provider to generate content
- Stores results in the enrichment repository
- Returns summary (created count, failed count)

### 4. Modified `ProcedureEnrichmentService` to be Lazy
**File**: `internal/application/services/procedure_enrichment_service.go`

Changed `GetEnrichment()` behavior:
- **Before**: Generated enrichment on-demand if not cached
- **After**: Only returns cached enrichment; returns error if not found
- Added helpful error message: "enrichment should be generated during ingestion"

This ensures enrichment is ONLY created during ingestion, never on-demand.

### 5. Updated Initialization
**File**: `cmd/api/main.go`

Updated `NewProviderIngestionService` call to pass:
- `procedureEnrichmentAdapter` - database layer for enrichments
- `enrichmentProvider` - the LLM provider (OpenAI client)

### 6. Updated Integration Tests
**File**: `tests/integration/provider_ingestion_integration_test.go`

Modified test setup to:
- Create enrichment adapter: `enrichmentAdapter := database.NewProcedureEnrichmentAdapter(client)`
- Pass to ingestion service (provider is nil for tests, so enrichment is skipped)

## Benefits

1. **Performance**: 
   - No latency on enrichment queries (cached data only)
   - Enrichment happens asynchronously during ingestion

2. **Cost Efficiency**:
   - Each procedure enriched exactly once
   - No duplicate LLM calls
   - Batch operations can be optimized

3. **Reliability**:
   - Enrichment failures don't affect query performance
   - Failed enrichments logged for retry/monitoring
   - Graceful degradation (returns cached data or null)

4. **Maintainability**:
   - Clear separation: ingestion = enrichment, queries = retrieval
   - Easier to monitor enrichment completion
   - Can run enrichment on a schedule or manually if needed

## Behavior

### During Ingestion
```
ProviderIngestionService.SyncCurrentData()
  ├─ Fetch and store facilities & procedures
  └─ After all ingestion complete:
      └─ enrichProceduresBatch()
          ├─ List all procedures
          ├─ For each procedure without enrichment:
          │   ├─ Call enrichmentProvider.EnrichProcedure(procedure)
          │   └─ Store result in repository
          └─ Return summary { Created: N, Failed: M }
```

### On Query
```
ProcedureHandler.GetEnrichment(procedureID)
  └─ ProcedureEnrichmentService.GetEnrichment(procedureID)
      └─ Return from cache OR error if not found
```

## Migration Path

1. **Existing procedures without enrichment**:
   - Run ingestion again to trigger enrichment generation
   - OR manually call enrichment endpoint if needed

2. **New procedures**:
   - Automatically enriched during next ingestion

3. **Backward compatibility**:
   - Clients can handle empty enrichment gracefully
   - Error handling in frontend for missing enrichment

## Configuration

Enrichment is controlled by:
- `PROCEDURE_ENRICHMENT_PROVIDER` env var (e.g., "openai")
- LLM API credentials (OPENAI_API_KEY, etc.)

If not configured, enrichment is skipped with a log message.

## Monitoring

Enrichment results are tracked in the ingestion summary:
```json
{
  "records_processed": 500,
  "procedures_created": 150,
  "procedure_enrichments_created": 145,
  "procedure_enrichments_failed": 5
}
```

Monitor these metrics to ensure enrichment is working correctly.

## Future Enhancements

1. **Scheduled enrichment**: Run enrichment on a schedule (hourly/daily)
2. **Parallel enrichment**: Process multiple procedures concurrently with rate limiting
3. **Enrichment versioning**: Support updating enrichment as LLM models improve
4. **Partial enrichment**: Enrich only modified procedures between ingestions
5. **Multi-language**: Generate enrichments in different languages during batch process
