package repositories

import (
	"context"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// FeeWaiverRepository defines operations for fee waiver storage
type FeeWaiverRepository interface {
	Create(ctx context.Context, waiver *entities.FeeWaiver) error
	GetByID(ctx context.Context, id string) (*entities.FeeWaiver, error)
	GetActiveFacilityWaiver(ctx context.Context, facilityID string) (*entities.FeeWaiver, error)
	IncrementUsage(ctx context.Context, id string) error
	Update(ctx context.Context, waiver *entities.FeeWaiver) error
}
