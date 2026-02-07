package repositories

import (
	"context"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// FacilityRepository defines the interface for facility data operations
type FacilityRepository interface {
	// Create creates a new facility
	Create(ctx context.Context, facility *entities.Facility) error

	// GetByID retrieves a facility by ID
	GetByID(ctx context.Context, id string) (*entities.Facility, error)

	// GetByIDs retrieves multiple facilities by their IDs
	GetByIDs(ctx context.Context, ids []string) ([]*entities.Facility, error)

	// Update updates a facility
	Update(ctx context.Context, facility *entities.Facility) error

	// Delete deletes a facility
	Delete(ctx context.Context, id string) error

	// List retrieves facilities with filters
	List(ctx context.Context, filter FacilityFilter) ([]*entities.Facility, error)

	// Search searches facilities by location and criteria

	Search(ctx context.Context, params SearchParams) ([]*entities.Facility, error)
}

// FacilitySearchRepository defines the interface for facility search operations (e.g. Typesense)

type FacilitySearchRepository interface {

	// Search searches facilities

	Search(ctx context.Context, params SearchParams) ([]*entities.Facility, error)

	// SearchWithFacets searches facilities and returns facets and metadata
	SearchWithFacets(ctx context.Context, params SearchParams) (*EnhancedSearchResult, error)

	// Index indexes a facility

	Index(ctx context.Context, facility *entities.Facility) error

	// Delete removes a facility from index

	Delete(ctx context.Context, id string) error
}

// FacilityFilter defines filters for listing facilities

type FacilityFilter struct {
	FacilityType string
	IsActive     *bool
	Limit        int
	Offset       int
}

// SearchParams defines parameters for facility search
type SearchParams struct {
	Query             string
	Latitude          float64
	Longitude         float64
	RadiusKm          float64
	ProcedureID       string
	InsuranceProvider string
	MinPrice          *float64
	MaxPrice          *float64
	Limit             int
	Offset            int
	IncludeFacets     bool // Whether to include facet counts in results
}

// EnhancedSearchResult contains search results with facets and metadata
type EnhancedSearchResult struct {
	Facilities []*entities.Facility
	Facets     *entities.SearchFacets
	TotalCount int
	SearchTime float64 // in milliseconds
}
