package scheduling

import (
	"context"
	"errors"
	"fmt"
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
		return &UnavailableProvider{reason: "CALENDLY_API_KEY is not configured"}
	}

	primary := NewCalendlyAdapter(cfg.CalendlyAPIKey)
	var fallback providers.AppointmentProvider
	if cfg.AllowMockFallback {
		fallback = NewMockAdapter()
	}

	return &FallbackProvider{
		primary:                primary,
		fallback:               fallback,
		allowFallback:          cfg.AllowMockFallback,
		allowMissingExternalID: cfg.AllowMissingExternalID,
	}
}

// UnavailableProvider fails fast when scheduling is not configured.
type UnavailableProvider struct {
	reason string
}

func (p *UnavailableProvider) GetAvailableSlots(ctx context.Context, externalID string, from, to time.Time) ([]entities.AvailabilitySlot, error) {
	return nil, fmt.Errorf("scheduling provider unavailable: %s", p.reason)
}

func (p *UnavailableProvider) CreateAppointment(ctx context.Context, appointment *entities.Appointment) (string, string, error) {
	return "", "", fmt.Errorf("scheduling provider unavailable: %s", p.reason)
}

func (p *UnavailableProvider) CancelAppointment(ctx context.Context, externalID string, reason string) error {
	return fmt.Errorf("scheduling provider unavailable: %s", p.reason)
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
