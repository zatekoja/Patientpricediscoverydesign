package providers

import (
	"context"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// AppointmentProvider defines the interface for external scheduling services (Calendly, Cal.com, etc.)
type AppointmentProvider interface {
	// GetAvailableSlots returns available time slots for a given facility/resource
	GetAvailableSlots(ctx context.Context, externalID string, from, to time.Time) ([]entities.AvailabilitySlot, error)

	// CreateAppointment books an appointment on the external provider
	CreateAppointment(ctx context.Context, appointment *entities.Appointment) (externalID string, meetingLink string, err error)

	// CancelAppointment cancels an appointment on the external provider
	CancelAppointment(ctx context.Context, externalID string, reason string) error
}
