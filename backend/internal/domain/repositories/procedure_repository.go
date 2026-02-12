package repositories

import (
	"context"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// ProcedureRepository defines the interface for procedure data operations
type ProcedureRepository interface {
	// Create creates a new procedure
	Create(ctx context.Context, procedure *entities.Procedure) error

	// GetByID retrieves a procedure by ID
	GetByID(ctx context.Context, id string) (*entities.Procedure, error)

	// GetByCode retrieves a procedure by code
	GetByCode(ctx context.Context, code string) (*entities.Procedure, error)

	// GetByIDs retrieves multiple procedures by their IDs
	GetByIDs(ctx context.Context, ids []string) ([]*entities.Procedure, error)

	// Update updates a procedure
	Update(ctx context.Context, procedure *entities.Procedure) error

	// Delete deletes a procedure
	Delete(ctx context.Context, id string) error

	// List retrieves procedures with filters
	List(ctx context.Context, filter ProcedureFilter) ([]*entities.Procedure, error)
}

// ProcedureFilter defines filters for listing procedures
type ProcedureFilter struct {
	Category string
	IsActive *bool
	Limit    int
	Offset   int
}

// FacilityProcedureRepository defines the interface for facility-procedure pricing
type FacilityProcedureRepository interface {
	// Create creates a new facility procedure
	Create(ctx context.Context, fp *entities.FacilityProcedure) error

	// GetByID retrieves a facility procedure by ID
	GetByID(ctx context.Context, id string) (*entities.FacilityProcedure, error)

	// GetByFacilityAndProcedure retrieves pricing for a specific facility and procedure
	GetByFacilityAndProcedure(ctx context.Context, facilityID, procedureID string) (*entities.FacilityProcedure, error)

	// ListByFacility retrieves all procedures for a facility
	ListByFacility(ctx context.Context, facilityID string) ([]*entities.FacilityProcedure, error)

	// ListByFacilityWithCount retrieves paginated procedures for a facility with total count
	// CRITICAL: Search is performed across ALL data first, then filtered, then paginated
	// This ensures search results are complete, not limited to current page
	ListByFacilityWithCount(ctx context.Context, facilityID string, filter FacilityProcedureFilter) ([]*entities.FacilityProcedure, int, error)

	// Update updates a facility procedure
	Update(ctx context.Context, fp *entities.FacilityProcedure) error

	// Delete deletes a facility procedure
	Delete(ctx context.Context, id string) error
}

// FacilityProcedureFilter defines filters for facility procedure queries
// All filters are applied BEFORE pagination to ensure search works across entire dataset
type FacilityProcedureFilter struct {
	// Search and filtering (applied to ALL data before pagination)
	Category    string   // Filter by procedure category
	MinPrice    *float64 // Minimum price filter
	MaxPrice    *float64 // Maximum price filter
	IsAvailable *bool    // Availability filter
	SearchQuery string   // Text search in procedure name/description (searches ALL data first)
	SearchTerms []string // Expanded search terms (including synonyms) to match against tags

	// Sorting (applied to filtered data before pagination)
	SortBy    string // "price", "name", "category", "updated_at"
	SortOrder string // "asc", "desc"

	// Pagination (applied LAST after search/filter/sort)
	Limit  int // Pagination limit (applied after filtering/searching)
	Offset int // Pagination offset (applied after filtering/searching)
}
