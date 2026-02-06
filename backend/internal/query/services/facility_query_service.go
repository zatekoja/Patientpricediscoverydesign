package services

import (
	"context"
	"fmt"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// SearchRepository defines the interface for search operations
type SearchRepository interface {
	SearchWithFacets(ctx context.Context, params interface{}) ([]*entities.Facility, *Facets, int, error)
	Index(ctx context.Context, facility *entities.Facility) error
	Delete(ctx context.Context, id string) error
}

// FacilityRepository defines the interface for facility database operations
type FacilityRepository interface {
	GetByID(ctx context.Context, id string) (*entities.Facility, error)
	Create(ctx context.Context, facility *entities.Facility) error
	Update(ctx context.Context, facility *entities.Facility) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter interface{}) ([]*entities.Facility, error)
	Search(ctx context.Context, params interface{}) ([]*entities.Facility, error)
}

// CacheProvider defines the interface for cache operations
type CacheProvider interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// FacilityQueryService handles read-only facility operations
type FacilityQueryService struct {
	searchRepo SearchRepository
	dbRepo     FacilityRepository
	cache      CacheProvider
}

// NewFacilityQueryService creates a new facility query service
func NewFacilityQueryService(
	searchRepo SearchRepository,
	dbRepo FacilityRepository,
	cache CacheProvider,
) *FacilityQueryService {
	return &FacilityQueryService{
		searchRepo: searchRepo,
		dbRepo:     dbRepo,
		cache:      cache,
	}
}

// SearchParams defines parameters for facility search
type SearchParams struct {
	Query                string
	Latitude             float64
	Longitude            float64
	RadiusKm             float64
	FacilityTypes        []string
	InsuranceProviders   []string
	Specialties          []string
	Languages            []string
	MinRating            *float64
	MaxPrice             *float64
	MinPrice             *float64
	AcceptsNewPatients   *bool
	HasEmergency         *bool
	HasParking           *bool
	WheelchairAccessible *bool
	SortBy               string
	SortOrder            string
	Limit                int
	Offset               int
}

// SearchResult represents search results with metadata
type SearchResult struct {
	Facilities []*entities.Facility
	Facets     *Facets
	TotalCount int
	SearchTime float64 // milliseconds
}

// Facets represents aggregated search facets
type Facets struct {
	FacilityTypes      map[string]int
	InsuranceProviders map[string]int
	Cities             map[string]int
	States             map[string]int
	Specialties        map[string]int
	PriceRanges        []PriceRangeFacet
	RatingDistribution []RatingFacet
}

// PriceRangeFacet represents a price range bucket
type PriceRangeFacet struct {
	Min   float64
	Max   float64
	Count int
}

// RatingFacet represents a rating bucket
type RatingFacet struct {
	Rating float64
	Count  int
}

// FacilitySuggestion represents an autocomplete suggestion
type FacilitySuggestion struct {
	ID           string
	Name         string
	FacilityType string
	City         string
	State        string
	Distance     float64
	Rating       float64
}

// Search performs a facility search with facets
func (s *FacilityQueryService) Search(ctx context.Context, params SearchParams) (*SearchResult, error) {
	start := time.Now()

	// Build search parameters for the repository
	// In a real implementation, we'd convert SearchParams to the format expected by searchRepo
	searchParams := s.buildSearchParams(params)

	// Execute search with facets
	facilities, facets, total, err := s.searchRepo.SearchWithFacets(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	searchTime := float64(time.Since(start).Milliseconds())

	return &SearchResult{
		Facilities: facilities,
		Facets:     facets,
		TotalCount: total,
		SearchTime: searchTime,
	}, nil
}

// GetByID retrieves a facility by ID, using cache when available
func (s *FacilityQueryService) GetByID(ctx context.Context, id string) (*entities.Facility, error) {
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

	// Fall back to database for complete data
	facility, err := s.dbRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get facility: %w", err)
	}

	// Cache result
	if s.cache != nil {
		_ = s.cache.Set(ctx, cacheKey, facility, 5*time.Minute)
	}

	return facility, nil
}

// Suggest provides autocomplete suggestions for facility search
func (s *FacilityQueryService) Suggest(ctx context.Context, query string, lat, lon float64, limit int) ([]*FacilitySuggestion, error) {
	// TODO: Implement autocomplete using Typesense prefix search
	// For now, return nil to satisfy the interface
	return nil, nil
}

// buildSearchParams converts service params to repository params
func (s *FacilityQueryService) buildSearchParams(params SearchParams) interface{} {
	// This would convert to the format expected by the search repository
	// For now, we just pass through the params
	return params
}
