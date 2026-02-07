//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/providers/scheduling"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

func TestCalendlyAdapterIntegration(t *testing.T) {
	// Mock Calendly API Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authentication
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/event_type_available_times":
			if r.Method != "GET" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			// Verify parameters
			query := r.URL.Query()
			if query.Get("event_type") == "" || query.Get("start_time") == "" || query.Get("end_time") == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Return mock available times
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"collection": []map[string]interface{}{
					{
						"status":         "available",
						"start_time":     "2025-01-01T10:00:00Z",
						"scheduling_url": "https://calendly.com/booking/slot1",
					},
					{
						"status":         "available",
						"start_time":     "2025-01-01T14:00:00Z",
						"scheduling_url": "https://calendly.com/booking/slot2",
					},
				},
			})

		case "/scheduled_events/ext-event-1/cancellation":
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.WriteHeader(http.StatusCreated) // Calendly returns 201 or 200

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Initialize Adapter with Test Server URL
	adapter := scheduling.NewCalendlyAdapter("test-api-key", scheduling.WithBaseURL(server.URL))

	ctx := context.Background()

	t.Run("GetAvailableSlots", func(t *testing.T) {
		from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

		slots, err := adapter.GetAvailableSlots(ctx, "event-type-uuid", from, to)
		require.NoError(t, err)
		assert.Len(t, slots, 2)
		assert.Equal(t, "2025-01-01T10:00:00Z", slots[0].StartTime.Format(time.RFC3339))
		assert.False(t, slots[0].IsBooked)
	})

	t.Run("CreateAppointment", func(t *testing.T) {
		appt := &entities.Appointment{
			FacilityID: "facility-1",
		}
		id, link, err := adapter.CreateAppointment(ctx, appt)
		require.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.Contains(t, link, "calendly.com")
	})

	t.Run("CancelAppointment", func(t *testing.T) {
		err := adapter.CancelAppointment(ctx, "ext-event-1", "Changed mind")
		require.NoError(t, err)
	})
	
	t.Run("AuthenticationError", func(t *testing.T) {
		// Test with wrong key
		badAdapter := scheduling.NewCalendlyAdapter("wrong-key", scheduling.WithBaseURL(server.URL))
		from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
		
		_, err := badAdapter.GetAvailableSlots(ctx, "uuid", from, to)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})
}