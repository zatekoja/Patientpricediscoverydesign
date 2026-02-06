package repositories

import (
	"context"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// AppointmentRepository defines the interface for appointment data operations
type AppointmentRepository interface {
	// Create creates a new appointment
	Create(ctx context.Context, appointment *entities.Appointment) error

	// GetByID retrieves an appointment by ID
	GetByID(ctx context.Context, id string) (*entities.Appointment, error)

	// Update updates an appointment
	Update(ctx context.Context, appointment *entities.Appointment) error

	// Cancel cancels an appointment
	Cancel(ctx context.Context, id string) error

	// ListByUser retrieves appointments for a user
	ListByUser(ctx context.Context, userID string, filter AppointmentFilter) ([]*entities.Appointment, error)

	// ListByFacility retrieves appointments for a facility
	ListByFacility(ctx context.Context, facilityID string, filter AppointmentFilter) ([]*entities.Appointment, error)
}

// AppointmentFilter defines filters for listing appointments
type AppointmentFilter struct {
	Status entities.AppointmentStatus
	From   *time.Time
	To     *time.Time
	Limit  int
	Offset int
}

// AvailabilityRepository defines the interface for availability slot operations
type AvailabilityRepository interface {
	// Create creates a new availability slot
	Create(ctx context.Context, slot *entities.AvailabilitySlot) error

	// GetByID retrieves an availability slot by ID
	GetByID(ctx context.Context, id string) (*entities.AvailabilitySlot, error)

	// ListByFacility retrieves availability slots for a facility
	ListByFacility(ctx context.Context, facilityID string, from, to time.Time) ([]*entities.AvailabilitySlot, error)

	// Update updates an availability slot
	Update(ctx context.Context, slot *entities.AvailabilitySlot) error

	// Delete deletes an availability slot
	Delete(ctx context.Context, id string) error
}
