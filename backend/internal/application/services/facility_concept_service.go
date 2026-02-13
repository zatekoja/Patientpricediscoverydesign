package services

import (
	"context"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

type FacilityConceptService struct {
	fpRepo     repositories.FacilityProcedureRepository
	enrichRepo repositories.ProcedureEnrichmentRepository
}

func NewFacilityConceptService(fpRepo repositories.FacilityProcedureRepository, enrichRepo repositories.ProcedureEnrichmentRepository) *FacilityConceptService {
	return &FacilityConceptService{
		fpRepo:     fpRepo,
		enrichRepo: enrichRepo,
	}
}

func (s *FacilityConceptService) AggregateConcepts(ctx context.Context, facilityID string) (*entities.SearchConcepts, error) {
	fps, err := s.fpRepo.ListByFacility(ctx, facilityID)
	if err != nil {
		return nil, err
	}

	aggregated := &entities.SearchConcepts{}
	seenProcIDs := make(map[string]struct{})

	for _, fp := range fps {
		if fp == nil {
			continue
		}
		if _, seen := seenProcIDs[fp.ProcedureID]; seen {
			continue
		}
		seenProcIDs[fp.ProcedureID] = struct{}{}

		enrich, err := s.enrichRepo.GetByProcedureID(ctx, fp.ProcedureID)
		if err == nil && enrich != nil && enrich.SearchConcepts != nil {
			aggregated = entities.MergeSearchConcepts(aggregated, enrich.SearchConcepts)
		}
	}

	return aggregated, nil
}
