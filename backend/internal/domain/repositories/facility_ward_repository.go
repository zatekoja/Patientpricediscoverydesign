package repositories

import (
	"context"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// FacilityWardRepository defines the interface for facility ward data operations
type FacilityWardRepository interface {
	// Create creates a new facility ward
	Create(ctx context.Context, ward *entities.FacilityWard) error

	// GetByID retrieves a facility ward by ID
	GetByID(ctx context.Context, id string) (*entities.FacilityWard, error)

	// GetByFacilityID retrieves all wards for a facility
	GetByFacilityID(ctx context.Context, facilityID string) ([]*entities.FacilityWard, error)

	// GetByFacilityIDs retrieves all wards for multiple facilities in a single query
	GetByFacilityIDs(ctx context.Context, facilityIDs []string) (map[string][]*entities.FacilityWard, error)

	// GetByFacilityAndWard retrieves a specific ward by facility ID and ward name
	GetByFacilityAndWard(ctx context.Context, facilityID, wardName string) (*entities.FacilityWard, error)

	// Update updates a facility ward
	Update(ctx context.Context, ward *entities.FacilityWard) error

	// Upsert creates or updates a facility ward (inserts if not exists, updates if exists)
	Upsert(ctx context.Context, ward *entities.FacilityWard) error

	// Delete deletes a facility ward
	Delete(ctx context.Context, id string) error

	// DeleteByFacilityAndWard deletes a ward by facility ID and ward name
	DeleteByFacilityAndWard(ctx context.Context, facilityID, wardName string) error
}
