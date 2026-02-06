package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// AppointmentService handles appointment booking logic
type AppointmentService struct {
	repo     repositories.AppointmentRepository
	provider providers.AppointmentProvider
}

// NewAppointmentService creates a new appointment service
func NewAppointmentService(repo repositories.AppointmentRepository, provider providers.AppointmentProvider) *AppointmentService {
	return &AppointmentService{
		repo:     repo,
		provider: provider,
	}
}

// BookAppointment books an appointment
func (s *AppointmentService) BookAppointment(ctx context.Context, appointment *entities.Appointment) error {
	// 1. Validate appointment (e.g., check if time is in future)
	if appointment.ScheduledAt.Before(time.Now()) {
		return fmt.Errorf("cannot book appointment in the past")
	}

	// 2. Call external provider to book slot
	externalID, link, err := s.provider.CreateAppointment(ctx, appointment)
	if err != nil {
		return fmt.Errorf("failed to book with provider: %w", err)
	}

	// 3. Enrich appointment with external details
	if appointment.ID == "" {
		appointment.ID = uuid.New().String()
	}
	appointment.Status = entities.AppointmentStatusConfirmed
	// Note: In a real app, we would store ExternalID and MeetingLink in the entity
	// For now, we assume they are handled or logged
	// appointment.ExternalID = externalID
	// appointment.MeetingLink = link
	_ = externalID
	_ = link
	
	appointment.CreatedAt = time.Now()
	appointment.UpdatedAt = time.Now()

	// 4. Save to repository
	if err := s.repo.Create(ctx, appointment); err != nil {
		// Rollback external booking if DB fails? (Advanced: Saga pattern)
		// For now, return error
		return fmt.Errorf("failed to save appointment: %w", err)
	}

	return nil
}

// GetAvailableSlots returns available slots
func (s *AppointmentService) GetAvailableSlots(ctx context.Context, facilityID string, from, to time.Time) ([]entities.AvailabilitySlot, error) {
	// 1. Get facility external ID (mapping logic would go here)
	// For Phase 1, assume facilityID IS the external ID or we look it up
	externalID := facilityID 

	// 2. Call provider
	return s.provider.GetAvailableSlots(ctx, externalID, from, to)
}
