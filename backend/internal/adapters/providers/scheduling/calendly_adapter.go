package scheduling

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

// CalendlyAdapter implements AppointmentProvider for Calendly
type CalendlyAdapter struct {
	apiKey  string
	client  *http.Client
	baseURL string
}

// CalendlyOption defines a configuration option for CalendlyAdapter
type CalendlyOption func(*CalendlyAdapter)

// WithBaseURL allows overriding the base URL (useful for testing)
func WithBaseURL(url string) CalendlyOption {
	return func(a *CalendlyAdapter) {
		a.baseURL = url
	}
}

// NewCalendlyAdapter creates a new Calendly adapter
func NewCalendlyAdapter(apiKey string, opts ...CalendlyOption) providers.AppointmentProvider {
	adapter := &CalendlyAdapter{
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 10 * time.Second},
		baseURL: "https://api.calendly.com",
	}

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter
}

// GetAvailableSlots returns available slots
func (a *CalendlyAdapter) GetAvailableSlots(ctx context.Context, externalID string, from, to time.Time) ([]entities.AvailabilitySlot, error) {
	// Note: externalID here refers to Calendly Event Type UUID

	url := fmt.Sprintf("%s/event_type_available_times?event_type=%s&start_time=%s&end_time=%s",
		a.baseURL,
		externalID,
		from.UTC().Format(time.RFC3339),
		to.UTC().Format(time.RFC3339))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	a.addHeaders(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("calendly api error: status %d", resp.StatusCode)
	}

	var result struct {
		Collection []struct {
			Status        string    `json:"status"`
			StartTime     time.Time `json:"start_time"`
			SchedulingURL string    `json:"scheduling_url"`
		} `json:"collection"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var slots []entities.AvailabilitySlot
	for _, item := range result.Collection {
		if item.Status == "available" {
			slots = append(slots, entities.AvailabilitySlot{
				StartTime: item.StartTime,
				EndTime:   item.StartTime.Add(30 * time.Minute), // Assuming 30 min slots for now
				IsBooked:  false,
			})
		}
	}

	return slots, nil
}

// CreateAppointment creates an appointment via Calendly Scheduling Link
// This implementation generates a one-time scheduling link for the patient
// Returns (scheduling_link, event_type_id, error)
func (a *CalendlyAdapter) CreateAppointment(ctx context.Context, appointment *entities.Appointment) (string, string, error) {
	eventTypeUUID := strings.TrimSpace(appointment.SchedulingExternalID)
	if eventTypeUUID == "" {
		eventTypeUUID = strings.TrimSpace(appointment.FacilityID)
	}
	if eventTypeUUID == "" {
		return "", "", errors.New("missing Calendly event type UUID (facility scheduling external id)")
	}

	orgURL := strings.TrimRight(strings.TrimSpace(os.Getenv("CALENDLY_ORG_URL")), "/")
	if orgURL == "" {
		return "", "", errors.New("CALENDLY_ORG_URL is not configured")
	}

	// Build scheduling link with pre-filled data
	schedulingLink := fmt.Sprintf("%s/%s?name=%s&email=%s",
		orgURL,
		eventTypeUUID,
		url.QueryEscape(strings.TrimSpace(appointment.PatientName)),
		url.QueryEscape(strings.TrimSpace(appointment.PatientEmail)),
	)

	// Return the scheduling link that the frontend can redirect to
	// The webhook will receive the actual event ID when booking completes
	return eventTypeUUID, schedulingLink, nil
}

// CancelAppointment cancels an appointment
func (a *CalendlyAdapter) CancelAppointment(ctx context.Context, externalID string, reason string) error {
	url := fmt.Sprintf("%s/scheduled_events/%s/cancellation", a.baseURL, externalID)

	payload := map[string]string{
		"reason": reason,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	a.addHeaders(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		// handle errors
		return fmt.Errorf("failed to cancel: %d", resp.StatusCode)
	}

	return nil
}

func (a *CalendlyAdapter) addHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.apiKey))
	req.Header.Set("Content-Type", "application/json")
}
