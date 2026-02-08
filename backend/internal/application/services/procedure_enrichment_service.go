package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
)

var ErrEnrichmentUnavailable = errors.New("procedure enrichment provider unavailable")

// ProcedureEnrichmentService coordinates procedure enrichment with caching.
type ProcedureEnrichmentService struct {
	repo          repositories.ProcedureEnrichmentRepository
	procedureRepo repositories.ProcedureRepository
	provider      providers.ProcedureEnrichmentProvider
}

// NewProcedureEnrichmentService creates a new enrichment service.
func NewProcedureEnrichmentService(
	repo repositories.ProcedureEnrichmentRepository,
	procedureRepo repositories.ProcedureRepository,
	provider providers.ProcedureEnrichmentProvider,
) *ProcedureEnrichmentService {
	return &ProcedureEnrichmentService{
		repo:          repo,
		procedureRepo: procedureRepo,
		provider:      provider,
	}
}

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
		return nil, apperrors.NewNotFoundError(fmt.Sprintf("enrichment not found for procedure %s (enrichment should be generated during ingestion)", procedureID))
	}

	return nil, apperrors.NewNotFoundError(fmt.Sprintf("enrichment not found for procedure %s", procedureID))
}
