# Code Changes Summary - Procedure Enrichment During Ingestion

## 1. ProviderIngestionService - Core Changes

### Added Fields to Struct
```go
type ProviderIngestionService struct {
    // ... existing fields ...
    enrichmentRepo       repositories.ProcedureEnrichmentRepository
    enrichmentProvider   providers.ProcedureEnrichmentProvider
}
```

### Updated Constructor Signature
```go
func NewProviderIngestionService(
    client providerapi.Client,
    facilityRepo repositories.FacilityRepository,
    facilityService *FacilityService,
    procedureRepo repositories.ProcedureRepository,
    facilityProcedureRepo repositories.FacilityProcedureRepository,
    enrichmentRepo repositories.ProcedureEnrichmentRepository,  // NEW
    enrichmentProvider providers.ProcedureEnrichmentProvider,   // NEW
    geolocationProvider providers.GeolocationProvider,
    cacheProvider providers.CacheProvider,
    pageSize int,
) *ProviderIngestionService
```

### Enhanced ProviderIngestionSummary
```go
type ProviderIngestionSummary struct {
    RecordsProcessed            int `json:"records_processed"`
    FacilitiesCreated           int `json:"facilities_created"`
    FacilitiesUpdated           int `json:"facilities_updated"`
    ProceduresCreated           int `json:"procedures_created"`
    ProceduresUpdated           int `json:"procedures_updated"`
    FacilityProceduresCreated   int `json:"facility_procedures_created"`
    FacilityProceduresUpdated   int `json:"facility_procedures_updated"`
    ProcedureEnrichmentsCreated int `json:"procedure_enrichments_created"`  // NEW
    ProcedureEnrichmentsFailed  int `json:"procedure_enrichments_failed"`   // NEW
}
```

### Enrichment Trigger in SyncCurrentData
```go
// At the end of SyncCurrentData() method:
s.invalidateSearchCaches(ctx)

// Enrich procedures during ingestion
enrichmentSummary := s.enrichProceduresBatch(ctx)
summary.ProcedureEnrichmentsCreated = enrichmentSummary.Created
summary.ProcedureEnrichmentsFailed = enrichmentSummary.Failed

return summary, nil
```

### New Batch Enrichment Method
```go
// enrichProceduresBatch enriches all procedures that don't have enrichment data yet.
// This runs after ingestion to populate enrichment data once for all procedures.
func (s *ProviderIngestionService) enrichProceduresBatch(ctx context.Context) *struct {
    Created int
    Failed  int
} {
    result := &struct {
        Created int
        Failed  int
    }{}

    if s.enrichmentProvider == nil || s.enrichmentRepo == nil {
        log.Println("procedure enrichment provider or repository not configured, skipping enrichment")
        return result
    }

    // Get all procedures
    procedures, err := s.procedureRepo.List(ctx, repositories.ProcedureFilter{})
    if err != nil {
        log.Printf("failed to list procedures for enrichment: %v", err)
        return result
    }

    if len(procedures) == 0 {
        return result
    }

    // Check which procedures need enrichment
    var proceduresToEnrich []*entities.Procedure
    for _, proc := range procedures {
        // Check if enrichment already exists
        existing, err := s.enrichmentRepo.GetByProcedureID(ctx, proc.ID)
        if err == nil && existing != nil {
            // Enrichment already exists, skip
            continue
        }
        proceduresToEnrich = append(proceduresToEnrich, proc)
    }

    if len(proceduresToEnrich) == 0 {
        return result
    }

    now := time.Now()
    for _, proc := range proceduresToEnrich {
        enriched, err := s.enrichmentProvider.EnrichProcedure(ctx, proc)
        if err != nil {
            log.Printf("failed to enrich procedure %s (%s): %v", proc.ID, proc.Name, err)
            result.Failed++
            continue
        }

        // Ensure required fields are populated
        if enriched.ID == "" {
            enriched.ID = fmt.Sprintf("enrich_%s_%d", proc.ID, now.UnixNano())
        }
        if enriched.ProcedureID == "" {
            enriched.ProcedureID = proc.ID
        }
        if enriched.CreatedAt.IsZero() {
            enriched.CreatedAt = now
        }
        if enriched.UpdatedAt.IsZero() {
            enriched.UpdatedAt = now
        }

        // Use procedure description if enrichment doesn't have one
        if enriched.Description == "" {
            enriched.Description = proc.Description
        }

        // Store enrichment
        if err := s.enrichmentRepo.Upsert(ctx, enriched); err != nil {
            log.Printf("failed to store enrichment for procedure %s: %v", proc.ID, err)
            result.Failed++
            continue
        }

        result.Created++
    }

    log.Printf("batch enriched %d procedures (%d failed)", result.Created, result.Failed)
    return result
}
```

## 2. ProcedureEnrichmentService - Lazy Loading

### Changed GetEnrichment Behavior

**Before:**
```go
// GetEnrichment returns cached enrichment or generates and stores it.
func (s *ProcedureEnrichmentService) GetEnrichment(ctx context.Context, procedureID string, refresh bool) (*entities.ProcedureEnrichment, error) {
    if !refresh {
        if cached, err := s.repo.GetByProcedureID(ctx, procedureID); err == nil && cached != nil {
            return cached, nil
        }
    }

    // If not cached, generate on-demand
    if s.provider == nil {
        return nil, apperrors.NewExternalError("procedure enrichment provider not configured", ErrEnrichmentUnavailable)
    }

    procedure, err := s.procedureRepo.GetByID(ctx, procedureID)
    if err != nil {
        return nil, err
    }

    enriched, err := s.provider.EnrichProcedure(ctx, procedure)
    // ... store and return ...
}
```

**After:**
```go
// GetEnrichment returns cached enrichment. It no longer generates enrichment on-demand.
// Enrichment is now generated during ingestion and cached. Use GetEnrichment to retrieve cached data.
// Returns error if no enrichment exists - enrichment should be generated during ingestion.
func (s *ProcedureEnrichmentService) GetEnrichment(ctx context.Context, procedureID string, refresh bool) (*entities.ProcedureEnrichment, error) {
    if procedureID == "" {
        return nil, apperrors.NewValidationError("procedure ID is required")
    }

    // Always fetch from cache (ignore refresh parameter for now as enrichment is from ingestion)
    cached, err := s.repo.GetByProcedureID(ctx, procedureID)
    if err == nil && cached != nil {
        return cached, nil
    }

    // If no cached enrichment exists, log it but don't generate on-demand
    // Enrichment should be generated during ingestion via ProviderIngestionService.enrichProceduresBatch()
    if err != nil {
        return nil, apperrors.NewNotFoundError(fmt.Sprintf("enrichment not found for procedure %s (enrichment should be generated during ingestion)", procedureID), err)
    }

    return nil, apperrors.NewNotFoundError(fmt.Sprintf("enrichment not found for procedure %s", procedureID), nil)
}
```

## 3. Main API Setup - Dependency Injection

### cmd/api/main.go Changes

**Before:**
```go
ingestionService := services.NewProviderIngestionService(
    providerClient,
    facilityAdapter,
    facilityService,
    procedureAdapter,
    facilityProcedureAdapter,
    geolocationProvider,
    cacheProvider,
    pageSize,
)
```

**After:**
```go
ingestionService := services.NewProviderIngestionService(
    providerClient,
    facilityAdapter,
    facilityService,
    procedureAdapter,
    facilityProcedureAdapter,
    procedureEnrichmentAdapter,        // NEW: pass enrichment repository
    enrichmentProvider,                // NEW: pass enrichment provider
    geolocationProvider,
    cacheProvider,
    pageSize,
)
```

(Note: `procedureEnrichmentAdapter` and `enrichmentProvider` are already initialized earlier in main.go)

## 4. Integration Tests - Updated Setup

### tests/integration/provider_ingestion_integration_test.go

**Before:**
```go
providerClient := providerapi.NewClient(baseURL)
ingestion := services.NewProviderIngestionService(
    providerClient,
    facilityRepo,
    facilityService,
    procedureRepo,
    facilityProcedureRepo,
    nil,
    nil,
    200,
)
```

**After:**
```go
providerClient := providerapi.NewClient(baseURL)
enrichmentAdapter := database.NewProcedureEnrichmentAdapter(client)
ingestion := services.NewProviderIngestionService(
    providerClient,
    facilityRepo,
    facilityService,
    procedureRepo,
    facilityProcedureRepo,
    enrichmentAdapter,                  // NEW: enrichment repository
    nil,                                // NEW: no enrichment provider for test
    nil,                                // geolocation provider
    nil,                                // cache provider
    200,
)
```

## Summary of Design

### Architecture Pattern
- **Separation of Concerns**: Enrichment creation (ingestion) vs. enrichment retrieval (query)
- **Batch Processing**: All procedures enriched together during ingestion
- **Optional Feature**: Gracefully degrades if enrichment provider not configured
- **Resilient**: Failed enrichments don't block ingestion; retried on next run

### Data Flow
```
Provider API
    ↓
ProviderIngestionService.SyncCurrentData()
    ├─ Create/update facilities & procedures
    └─ enrichProceduresBatch()
        ├─ Query procedures without enrichment
        ├─ Call LLM provider for each
        └─ Store in database
            ↓
Query Handler
    ↓
ProcedureEnrichmentService.GetEnrichment()
    └─ Return cached enrichment
```

### Key Benefits
1. **One-time enrichment** per procedure (cost efficient)
2. **No query latency** from LLM calls (cached data)
3. **Graceful degradation** (works without enrichment provider)
4. **Observable** (success metrics in ingestion response)
5. **Retryable** (failed enrichments on next ingestion)
