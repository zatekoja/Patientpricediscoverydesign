package scheduling

import (
	"context"
	"fmt"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

// MockAdapter provides deterministic availability for local development.
type MockAdapter struct {
	slotDuration time.Duration
	maxSlots     int
}

// NewMockAdapter creates a mock scheduling provider.
func NewMockAdapter() providers.AppointmentProvider {
	return &MockAdapter{
		slotDuration: 30 * time.Minute,
		maxSlots:     6,
	}
}

// GetAvailableSlots returns sample slots within the requested range.
func (m *MockAdapter) GetAvailableSlots(ctx context.Context, externalID string, from, to time.Time) ([]entities.AvailabilitySlot, error) {
	if to.Before(from) {
		return nil, fmt.Errorf("invalid time range")
	}

	slots := make([]entities.AvailabilitySlot, 0, m.maxSlots)
	cursor := from.Truncate(time.Minute).Add(30 * time.Minute)
	for cursor.Before(to) && len(slots) < m.maxSlots {
		slots = append(slots, entities.AvailabilitySlot{
			StartTime: cursor,
			EndTime:   cursor.Add(m.slotDuration),
			IsBooked:  false,
		})
		cursor = cursor.Add(m.slotDuration)
	}

	return slots, nil
}

// CreateAppointment returns a mock booking reference.
func (m *MockAdapter) CreateAppointment(ctx context.Context, appointment *entities.Appointment) (string, string, error) {
	id := fmt.Sprintf("mock-%d", time.Now().UnixNano())
	return id, fmt.Sprintf("https://example.com/booking/%s", id), nil
}

// CancelAppointment is a no-op for the mock provider.
func (m *MockAdapter) CancelAppointment(ctx context.Context, externalID string, reason string) error {
	return nil
}
