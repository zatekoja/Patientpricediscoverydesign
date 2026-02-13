package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

const (
	TargetEnrichmentVersion = 1
	BatchSize               = 100
)

type BackfillSummary struct {
	TotalProcessed int
	SuccessCount   int
	FailureCount   int
}

type ConceptBackfillService struct {
	procedureRepo  repositories.ProcedureRepository
	enrichmentRepo repositories.ProcedureEnrichmentRepository
	provider       providers.ProcedureEnrichmentProvider
	workerCount    int
	maxRetries     int
}

func NewConceptBackfillService(
	procRepo repositories.ProcedureRepository,
	enrichRepo repositories.ProcedureEnrichmentRepository,
	provider providers.ProcedureEnrichmentProvider,
	workers int,
	maxRetries int,
) *ConceptBackfillService {
	if workers <= 0 {
		workers = 1
	}
	return &ConceptBackfillService{
		procedureRepo:  procRepo,
		enrichmentRepo: enrichRepo,
		provider:       provider,
		workerCount:    workers,
		maxRetries:     maxRetries,
	}
}

func (s *ConceptBackfillService) BackfillAll(ctx context.Context) (*BackfillSummary, error) {
	summary := &BackfillSummary{}
	var processed, success, failure int64

	// Channel to feed procedure IDs to workers
	idChan := make(chan string, BatchSize)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < s.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range idChan {
				err := s.BackfillSingle(ctx, id)
				atomic.AddInt64(&processed, 1)
				if err != nil {
					atomic.AddInt64(&failure, 1)
					log.Printf("Failed to backfill procedure %s: %v", id, err)
				} else {
					atomic.AddInt64(&success, 1)
				}
			}
		}()
	}

	// Producer loop
	offset := 0
	for {
		ids, err := s.enrichmentRepo.ListProcedureIDsNeedingEnrichment(ctx, TargetEnrichmentVersion, BatchSize)
		if err != nil {
			close(idChan)
			return nil, fmt.Errorf("failed to list procedures needing enrichment: %w", err)
		}

		if len(ids) == 0 {
			break
		}

		for _, id := range ids {
			select {
			case idChan <- id:
			case <-ctx.Done():
				close(idChan)
				return nil, ctx.Err()
			}
		}

		if len(ids) < BatchSize {
			break
		}
		offset += len(ids)
	}

	close(idChan)
	wg.Wait()

	summary.TotalProcessed = int(processed)
	summary.SuccessCount = int(success)
	summary.FailureCount = int(failure)

	return summary, nil
}

func (s *ConceptBackfillService) BackfillSingle(ctx context.Context, procedureID string) error {
	procedure, err := s.procedureRepo.GetByID(ctx, procedureID)
	if err != nil {
		return fmt.Errorf("failed to get procedure %s: %w", procedureID, err)
	}

	// 1. Enrich
	enrichment, err := s.provider.EnrichProcedure(ctx, procedure)
	if err != nil {
		// Handle failure
		return s.handleEnrichmentFailure(ctx, procedureID, err)
	}

	// 2. Success - update enrichment
	// Ensure we preserve existing ID/created_at if it exists, or Upsert handles it?
	// The adapter Upsert uses ON CONFLICT (procedure_id) DO UPDATE.
	// But we need to make sure we don't overwrite ID if we don't have it in the returned enrichment object?
	// provider.EnrichProcedure returns a new struct. It might not have ID set.
	// We should probably check if one exists first to get the ID, OR let Upsert handle it (if adapter generates ID on insert).
	// Adapter: if enrichment.ID == "", generate new UUID.
	// But if it exists, we want to update it.
	// Adapter uses ON CONFLICT (procedure_id), so ID isn't strictly needed for update logic, but good for consistency.
	
	enrichment.ProcedureID = procedureID
	enrichment.EnrichmentStatus = "completed"
	enrichment.EnrichmentVersion = TargetEnrichmentVersion
	// Reset retry count on success? The adapter upsert updates retry_count = EXCLUDED.retry_count.
	// We should probably set it to 0.
	enrichment.RetryCount = 0
	enrichment.LastError = ""

	if err := s.enrichmentRepo.Upsert(ctx, enrichment); err != nil {
		return fmt.Errorf("failed to upsert enrichment for %s: %w", procedureID, err)
	}

	return nil
}

func (s *ConceptBackfillService) handleEnrichmentFailure(ctx context.Context, procedureID string, err error) error {
	// Get existing to check retry count
	existing, getErr := s.enrichmentRepo.GetByProcedureID(ctx, procedureID)
	if getErr != nil {
		// If not found, it's a first-time failure.
		// We can't use UpdateStatus if we don't have an ID.
		// We need to create a record with status 'failed' or 'pending' (retry).
		// Since GetByProcedureID failed (likely not found), we construct a new one.
		// But wait, GetByProcedureID returns error if not found.
		// We should create a dummy record to track the failure.
		enrichment := &entities.ProcedureEnrichment{
			ProcedureID:      procedureID,
			EnrichmentStatus: "pending", // Default to pending to retry
			RetryCount:       1,
			LastError:        err.Error(),
		}
		if s.maxRetries > 0 && enrichment.RetryCount >= s.maxRetries {
			enrichment.EnrichmentStatus = "abandoned"
		}
		
		if upsertErr := s.enrichmentRepo.Upsert(ctx, enrichment); upsertErr != nil {
			return fmt.Errorf("failed to record failure for %s: %w", procedureID, upsertErr)
		}
		return err
	}

	// Existing record found
	newStatus := "pending"
	if s.maxRetries > 0 && existing.RetryCount+1 >= s.maxRetries {
		newStatus = "abandoned"
	}

	if updateErr := s.enrichmentRepo.UpdateStatus(ctx, existing.ID, newStatus, err.Error()); updateErr != nil {
		return fmt.Errorf("failed to update status for %s: %w", procedureID, updateErr)
	}

	return err
}
