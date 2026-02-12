package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
)

// AppointmentService handles appointment booking logic
type AppointmentService struct {
	repo                   repositories.AppointmentRepository
	facilityRepo           repositories.FacilityRepository
	procedureRepo          repositories.ProcedureRepository
	provider               providers.AppointmentProvider
	allowMissingExternalID bool
	notificationService    *NotificationService
}

// NewAppointmentService creates a new appointment service
func NewAppointmentService(
	repo repositories.AppointmentRepository,
	facilityRepo repositories.FacilityRepository,
	procedureRepo repositories.ProcedureRepository,
	provider providers.AppointmentProvider,
	allowMissingExternalID bool,
	notificationService *NotificationService,
) *AppointmentService {
	return &AppointmentService{
		repo:                   repo,
		facilityRepo:           facilityRepo,
		procedureRepo:          procedureRepo,
		provider:               provider,
		allowMissingExternalID: allowMissingExternalID,
		notificationService:    notificationService,
	}
}

// BookAppointment books an appointment
func (s *AppointmentService) BookAppointment(ctx context.Context, appointment *entities.Appointment) error {
	// 1. Validate appointment (e.g., check if time is in future)
	if appointment.ScheduledAt.Before(time.Now()) {
		return fmt.Errorf("cannot book appointment in the past")
	}

	if s.facilityRepo == nil {
		return fmt.Errorf("facility repository is not configured")
	}

	facility, err := s.facilityRepo.GetByID(ctx, appointment.FacilityID)
	if err != nil {
		return fmt.Errorf("failed to load facility: %w", err)
	}
	if facility == nil {
		return fmt.Errorf("facility not found")
	}

	externalID := strings.TrimSpace(facility.SchedulingExternalID)
	if externalID == "" {
		return fmt.Errorf("facility has no scheduling external id configured")
	}
	appointment.SchedulingExternalID = externalID

	// 2. Call external provider to book slot
	providerExternalID, link, err := s.provider.CreateAppointment(ctx, appointment)
	if err != nil {
		return fmt.Errorf("failed to book with provider: %w", err)
	}

	// 3. Enrich appointment with external details
	if appointment.ID == "" {
		appointment.ID = uuid.New().String()
	}
	// Booking is pending until Calendly webhook (invitee.created) confirms attendance.
	appointment.Status = entities.AppointmentStatusPending
	appointment.BookingMethod = entities.BookingMethodAPI
	if providerExternalID != "" {
		appointment.CalendlyEventID = &providerExternalID
	}
	if link != "" {
		appointment.MeetingLink = &link
	}

	appointment.CreatedAt = time.Now()
	appointment.UpdatedAt = time.Now()

	// 4. Save to repository
	if err := s.repo.Create(ctx, appointment); err != nil {
		// Rollback external booking if DB fails? (Advanced: Saga pattern)
		// For now, return error
		return fmt.Errorf("failed to save appointment: %w", err)
	}

	// NOTE: confirmation notification is sent by Calendly webhook handler after invitee.created.

	return nil
}

// GetAvailableSlots returns available slots
func (s *AppointmentService) GetAvailableSlots(ctx context.Context, facilityID string, from, to time.Time) ([]entities.AvailabilitySlot, error) {
	if s.facilityRepo == nil {
		return nil, apperrors.NewInternalError("facility repository not configured", nil)
	}

	facility, err := s.facilityRepo.GetByID(ctx, facilityID)
	if err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok && appErr.Type == apperrors.ErrorTypeNotFound {
			return nil, err
		}
		return nil, apperrors.NewNotFoundError("facility not found")
	}

	externalID := strings.TrimSpace(facility.SchedulingExternalID)
	if externalID == "" && !s.allowMissingExternalID {
		return nil, apperrors.NewValidationError("facility has no scheduling external id configured")
	}

	slots, err := s.provider.GetAvailableSlots(ctx, externalID, from, to)
	if err != nil {
		return nil, apperrors.NewExternalError("failed to fetch availability", err)
	}
	return slots, nil
}
