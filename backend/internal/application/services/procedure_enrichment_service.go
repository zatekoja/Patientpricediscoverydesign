package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
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

// GetEnrichment returns cached enrichment or generates and stores it.
func (s *ProcedureEnrichmentService) GetEnrichment(ctx context.Context, procedureID string, refresh bool) (*entities.ProcedureEnrichment, error) {
	if procedureID == "" {
		return nil, apperrors.NewValidationError("procedure ID is required")
	}

	if !refresh {
		if cached, err := s.repo.GetByProcedureID(ctx, procedureID); err == nil && cached != nil {
			return cached, nil
		}
	}

	if s.provider == nil {
		return nil, apperrors.NewExternalError("procedure enrichment provider not configured", ErrEnrichmentUnavailable)
	}

	procedure, err := s.procedureRepo.GetByID(ctx, procedureID)
	if err != nil {
		return nil, err
	}

	enriched, err := s.provider.EnrichProcedure(ctx, procedure)
	if err != nil {
		return nil, apperrors.NewExternalError("failed to enrich procedure", err)
	}

	now := time.Now()
	enriched.ID = uuid.New().String()
	enriched.ProcedureID = procedure.ID
	enriched.CreatedAt = now
	enriched.UpdatedAt = now

	if enriched.Description == "" {
		enriched.Description = procedure.Description
	}

	if err := s.repo.Upsert(ctx, enriched); err != nil {
		return nil, err
	}

	return enriched, nil
}
