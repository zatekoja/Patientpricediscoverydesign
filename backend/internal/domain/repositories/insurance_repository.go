package repositories

import (
	"context"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// InsuranceRepository defines the interface for insurance provider operations
type InsuranceRepository interface {
	// Create creates a new insurance provider
	Create(ctx context.Context, insurance *entities.InsuranceProvider) error

	// GetByID retrieves an insurance provider by ID
	GetByID(ctx context.Context, id string) (*entities.InsuranceProvider, error)

	// GetByCode retrieves an insurance provider by code
	GetByCode(ctx context.Context, code string) (*entities.InsuranceProvider, error)

	// Update updates an insurance provider
	Update(ctx context.Context, insurance *entities.InsuranceProvider) error

	// Delete deletes an insurance provider
	Delete(ctx context.Context, id string) error

	// List retrieves insurance providers
	List(ctx context.Context, filter InsuranceFilter) ([]*entities.InsuranceProvider, error)

	// GetFacilityInsurance retrieves accepted insurance for a facility
	GetFacilityInsurance(ctx context.Context, facilityID string) ([]*entities.InsuranceProvider, error)
}

// InsuranceFilter defines filters for listing insurance providers
type InsuranceFilter struct {
	IsActive *bool
	Limit    int
	Offset   int
}
