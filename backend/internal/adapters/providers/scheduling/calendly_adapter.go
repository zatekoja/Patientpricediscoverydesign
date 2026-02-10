package scheduling

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	// For Calendly Standard tier: Generate a scheduling link
	// The patient will use this link to complete the booking
	// We will receive confirmation via webhook (invitee.created)

	// Get the event type for this facility/procedure
	// In production, this would be mapped in configuration
	eventTypeUUID := a.getEventTypeForFacility(appointment.FacilityID)

	// Build scheduling link with pre-filled data
	schedulingLink := fmt.Sprintf("%s/%s?name=%s&email=%s",
		"https://calendly.com/your-org", // Replace with actual org URL
		eventTypeUUID,
		appointment.PatientName,
		appointment.PatientEmail,
	)

	// Return the scheduling link that the frontend can redirect to
	// The webhook will receive the actual event ID when booking completes
	return schedulingLink, eventTypeUUID, nil
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

// getEventTypeForFacility maps facility ID to Calendly event type UUID
// In production, this would be stored in database configuration
func (a *CalendlyAdapter) getEventTypeForFacility(facilityID string) string {
	// TODO: Implement database lookup
	// For now, return a placeholder that should be configured per facility
	return "event-type-uuid-for-" + facilityID
}

func (a *CalendlyAdapter) addHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.apiKey))
	req.Header.Set("Content-Type", "application/json")
}
