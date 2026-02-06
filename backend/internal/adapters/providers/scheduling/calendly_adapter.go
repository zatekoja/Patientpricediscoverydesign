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
	apiKey string
	client *http.Client
	baseURL string
}

// NewCalendlyAdapter creates a new Calendly adapter
func NewCalendlyAdapter(apiKey string) providers.AppointmentProvider {
	return &CalendlyAdapter{
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 10 * time.Second},
		baseURL: "https://api.calendly.com",
	}
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
			Status    string    `json:"status"`
			StartTime time.Time `json:"start_time"`
			SchedulingURL string `json:"scheduling_url"`
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

// CreateAppointment creates an appointment (Headless via Scheduling Link or direct API if supported)
// Note: Calendly "Create Event" usually requires generating a One-Off link or using the Embedding.
// For true headless booking, we might need to use their "Scheduling Link" flow or v2 API "scheduled_events" (which is restricted).
// For this adapter, we will simulate the "Booking" by returning the Scheduling URL if real booking is not exposed to PAT.
// However, search results indicated "Create Event Invitee endpoint" allows creating invitees.
func (a *CalendlyAdapter) CreateAppointment(ctx context.Context, appointment *entities.Appointment) (string, string, error) {
	// In a real implementation, we would POST to /scheduled_events/{uuid}/invitees
	// But that requires an existing event UUID.
	// For "Headless" flow, typically we:
	// 1. Get the slot UUID from Availability.
	// 2. POST to book it.
	
	// Since we don't have the specific UUID in the input (only time), this is complex with Calendly.
	// We will implement a simplified version that returns a booking link for now, 
	// OR if we assume we have the event URI, we try to book it.
	
	// Mock implementation for Phase 1 as we don't have a real Calendly Enterprise account to test Headless fully.
	// We will assume "externalID" in appointment (FacilityID map) is the Event Type.
	
	return "mock-calendly-id", fmt.Sprintf("https://calendly.com/booking/%s", appointment.FacilityID), nil
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
