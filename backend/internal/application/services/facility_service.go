package services

import (
	"context"
	"log"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// FacilityService handles business logic for facilities
type FacilityService struct {
	repo       repositories.FacilityRepository
	searchRepo repositories.FacilitySearchRepository
}

// NewFacilityService creates a new facility service
func NewFacilityService(repo repositories.FacilityRepository, searchRepo repositories.FacilitySearchRepository) *FacilityService {
	return &FacilityService{
		repo:       repo,
		searchRepo: searchRepo,
	}
}

// Create creates a new facility and indexes it
func (s *FacilityService) Create(ctx context.Context, facility *entities.Facility) error {
	// 1. Save to database
	if err := s.repo.Create(ctx, facility); err != nil {
		return err
	}

	// 2. Index in search engine
	if s.searchRepo != nil {
		if err := s.searchRepo.Index(ctx, facility); err != nil {
			// Log error but don't fail the request (eventual consistency)
			log.Printf("Warning: Failed to index facility %s: %v", facility.ID, err)
		}
	}

	return nil
}

// GetByID retrieves a facility by ID
func (s *FacilityService) GetByID(ctx context.Context, id string) (*entities.Facility, error) {
	return s.repo.GetByID(ctx, id)
}

// Update updates a facility and updates index
func (s *FacilityService) Update(ctx context.Context, facility *entities.Facility) error {
	// 1. Update in database
	if err := s.repo.Update(ctx, facility); err != nil {
		return err
	}

	// 2. Update index
	if s.searchRepo != nil {
		if err := s.searchRepo.Index(ctx, facility); err != nil {
			log.Printf("Warning: Failed to update facility index %s: %v", facility.ID, err)
		}
	}

	return nil
}

// Delete deletes a facility and removes from index
func (s *FacilityService) Delete(ctx context.Context, id string) error {
	// 1. Delete from database
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// 2. Delete from index
	if s.searchRepo != nil {
		if err := s.searchRepo.Delete(ctx, id); err != nil {
			log.Printf("Warning: Failed to delete facility from index %s: %v", id, err)
		}
	}

	return nil
}

// List retrieves facilities
func (s *FacilityService) List(ctx context.Context, filter repositories.FacilityFilter) ([]*entities.Facility, error) {
	return s.repo.List(ctx, filter)
}

// Search searches facilities using search engine if available, falling back to database
func (s *FacilityService) Search(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, error) {
	if s.searchRepo != nil {
		return s.searchRepo.Search(ctx, params)
	}
	return s.repo.Search(ctx, params)
}
