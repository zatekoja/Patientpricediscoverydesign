package scheduling

import (
	"context"
	"errors"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

// ErrMissingExternalID indicates the facility has no scheduling identifier configured.
var ErrMissingExternalID = errors.New("scheduling external id is required")

// AppointmentProviderConfig configures scheduling providers.
type AppointmentProviderConfig struct {
	CalendlyAPIKey         string
	AllowMockFallback      bool
	AllowMissingExternalID bool
}

// NewAppointmentProvider creates a resilient provider with optional mock fallback.
func NewAppointmentProvider(cfg AppointmentProviderConfig) providers.AppointmentProvider {
	if cfg.CalendlyAPIKey == "" {
		// No real provider configured; use mock provider for dev.
		return NewMockAdapter()
	}

	primary := NewCalendlyAdapter(cfg.CalendlyAPIKey)
	fallback := NewMockAdapter()

	return &FallbackProvider{
		primary:                primary,
		fallback:               fallback,
		allowFallback:          cfg.AllowMockFallback,
		allowMissingExternalID: cfg.AllowMissingExternalID,
	}
}

// FallbackProvider wraps a primary provider with optional mock fallback.
type FallbackProvider struct {
	primary                providers.AppointmentProvider
	fallback               providers.AppointmentProvider
	allowFallback          bool
	allowMissingExternalID bool
}

func (p *FallbackProvider) GetAvailableSlots(ctx context.Context, externalID string, from, to time.Time) ([]entities.AvailabilitySlot, error) {
	if externalID == "" {
		if p.allowMissingExternalID && p.fallback != nil {
			return p.fallback.GetAvailableSlots(ctx, externalID, from, to)
		}
		return nil, ErrMissingExternalID
	}

	if p.primary == nil {
		if p.fallback != nil {
			return p.fallback.GetAvailableSlots(ctx, externalID, from, to)
		}
		return nil, errors.New("scheduling provider not configured")
	}

	slots, err := p.primary.GetAvailableSlots(ctx, externalID, from, to)
	if err != nil && p.allowFallback && p.fallback != nil {
		return p.fallback.GetAvailableSlots(ctx, externalID, from, to)
	}
	return slots, err
}

func (p *FallbackProvider) CreateAppointment(ctx context.Context, appointment *entities.Appointment) (string, string, error) {
	if p.primary == nil {
		if p.fallback != nil {
			return p.fallback.CreateAppointment(ctx, appointment)
		}
		return "", "", errors.New("scheduling provider not configured")
	}

	id, link, err := p.primary.CreateAppointment(ctx, appointment)
	if err != nil && p.allowFallback && p.fallback != nil {
		return p.fallback.CreateAppointment(ctx, appointment)
	}
	return id, link, err
}

func (p *FallbackProvider) CancelAppointment(ctx context.Context, externalID string, reason string) error {
	if p.primary == nil {
		if p.fallback != nil {
			return p.fallback.CancelAppointment(ctx, externalID, reason)
		}
		return errors.New("scheduling provider not configured")
	}

	err := p.primary.CancelAppointment(ctx, externalID, reason)
	if err != nil && p.allowFallback && p.fallback != nil {
		return p.fallback.CancelAppointment(ctx, externalID, reason)
	}
	return err
}
