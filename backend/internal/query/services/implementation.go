package services

import (
	"context"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// SearchAdapter wraps Typesense adapter for search operations
type SearchAdapter interface {
	Search(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, error)
	SearchWithFacets(ctx context.Context, params repositories.SearchParams) (*repositories.EnhancedSearchResult, error)
	Index(ctx context.Context, facility *entities.Facility) error
	Delete(ctx context.Context, id string) error
}

// QueryCacheProvider interface for caching in query services
type QueryCacheProvider interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// FacilityQueryServiceImpl is the concrete implementation
type FacilityQueryServiceImpl struct {
	searchAdapter SearchAdapter
	dbRepo        repositories.FacilityRepository
	cache         QueryCacheProvider
}

// NewFacilityQueryServiceImpl creates a new facility query service
func NewFacilityQueryServiceImpl(
	searchAdapter SearchAdapter,
	dbRepo repositories.FacilityRepository,
	cache QueryCacheProvider,
) *FacilityQueryServiceImpl {
	return &FacilityQueryServiceImpl{
		searchAdapter: searchAdapter,
		dbRepo:        dbRepo,
		cache:         cache,
	}
}

// Search performs a facility search
func (s *FacilityQueryServiceImpl) Search(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, error) {
	// Try to search using Typesense
	results, err := s.searchAdapter.Search(ctx, params)
	if err != nil {
		// Fallback to database search
		return s.dbRepo.Search(ctx, params)
	}
	return results, nil
}

// GetByID retrieves a facility by ID with caching
func (s *FacilityQueryServiceImpl) GetByID(ctx context.Context, id string) (*entities.Facility, error) {
	cacheKey := "facility:" + id

	// Try cache first
	if s.cache != nil {
		cached, err := s.cache.Get(ctx, cacheKey)
		if err == nil && cached != nil {
			if facility, ok := cached.(*entities.Facility); ok {
				return facility, nil
			}
		}
	}

	// Fall back to database
	facility, err := s.dbRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache result
	if s.cache != nil {
		_ = s.cache.Set(ctx, cacheKey, facility, 5*time.Minute)
	}

	return facility, nil
}
