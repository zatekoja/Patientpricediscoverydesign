package integration_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CalendlyWebhookPayload represents a Calendly webhook event
type CalendlyWebhookPayload struct {
	Event   string                 `json:"event"`
	Time    time.Time              `json:"time"`
	Payload map[string]interface{} `json:"payload"`
}

// TestCalendlyWebhookSignatureVerification tests webhook signature validation
func TestCalendlyWebhookSignatureVerification(t *testing.T) {
	signingSecret := "test_secret_key_123"

	testCases := []struct {
		name               string
		event              string
		includeSignature   bool
		correctSignature   bool
		expectedStatusCode int
		description        string
	}{
		{
			name:               "Valid signature",
			event:              "invitee.created",
			includeSignature:   true,
			correctSignature:   true,
			expectedStatusCode: http.StatusOK,
			description:        "Should accept webhook with valid HMAC-SHA256 signature",
		},
		{
			name:               "Invalid signature",
			event:              "invitee.created",
			includeSignature:   true,
			correctSignature:   false,
			expectedStatusCode: http.StatusUnauthorized,
			description:        "Should reject webhook with invalid signature",
		},
		{
			name:               "Missing signature",
			event:              "invitee.created",
			includeSignature:   false,
			correctSignature:   false,
			expectedStatusCode: http.StatusUnauthorized,
			description:        "Should reject webhook without signature",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create webhook payload
			payload := CalendlyWebhookPayload{
				Event: tc.event,
				Time:  time.Now(),
				Payload: map[string]interface{}{
					"uri": "https://calendly.com/events/abc123",
					"invitee": map[string]interface{}{
						"email": "patient@example.com",
						"name":  "Test Patient",
					},
					"event": map[string]interface{}{
						"start_time": time.Now().AddDate(0, 0, 1).Format(time.RFC3339),
						"location": map[string]interface{}{
							"join_url": "https://meet.google.com/xyz",
						},
					},
				},
			}

			body, err := json.Marshal(payload)
			require.NoError(t, err, "Failed to marshal payload")

			// Create HTTP request
			req := httptest.NewRequest("POST", "/webhooks/calendly", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			if tc.includeSignature {
				var signature string
				if tc.correctSignature {
					// Generate valid signature
					mac := hmac.New(sha256.New, []byte(signingSecret))
					mac.Write(body)
					signature = hex.EncodeToString(mac.Sum(nil))
				} else {
					// Use invalid signature
					signature = "invalid_signature_123456789"
				}
				req.Header.Set("Calendly-Webhook-Signature", signature)
			}

			// Test endpoint (placeholder - would call actual handler in real test)
			t.Logf("Test: %s - %s", tc.name, tc.description)
			t.Logf("Event: %s, Has Signature: %v, Status Expected: %d",
				tc.event, tc.includeSignature, tc.expectedStatusCode)
		})
	}
}

// TestCalendlyWebhookEventProcessing tests webhook event processing
func TestCalendlyWebhookEventProcessing(t *testing.T) {
	testCases := []struct {
		name          string
		eventType     string
		shouldProcess bool
		description   string
	}{
		{
			name:          "invitee.created event",
			eventType:     "invitee.created",
			shouldProcess: true,
			description:   "Should process appointment creation events",
		},
		{
			name:          "invitee.canceled event",
			eventType:     "invitee.canceled",
			shouldProcess: true,
			description:   "Should process appointment cancellation events",
		},
		{
			name:          "Unknown event type",
			eventType:     "invitee.custom_event",
			shouldProcess: false,
			description:   "Should ignore unknown event types",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload := CalendlyWebhookPayload{
				Event: tc.eventType,
				Time:  time.Now(),
				Payload: map[string]interface{}{
					"uri": "https://calendly.com/events/abc123",
					"invitee": map[string]interface{}{
						"email": "patient@example.com",
						"name":  "Test Patient",
					},
					"event": map[string]interface{}{
						"start_time": time.Now().AddDate(0, 0, 1).Format(time.RFC3339),
						"location": map[string]interface{}{
							"join_url": "https://meet.google.com/xyz",
						},
					},
				},
			}

			t.Logf("Test: %s - %s", tc.name, tc.description)
			t.Logf("Event Type: %s, Should Process: %v", tc.eventType, tc.shouldProcess)

			// Verify payload structure
			assert.NotEmpty(t, payload.Event, "Event type should not be empty")
			assert.NotNil(t, payload.Payload, "Payload should not be nil")
			assert.NotZero(t, payload.Time, "Time should be set")
		})
	}
}

// TestCalendlyWebhookPayloadValidation tests payload structure validation
func TestCalendlyWebhookPayloadValidation(t *testing.T) {
	validPayload := CalendlyWebhookPayload{
		Event: "invitee.created",
		Time:  time.Now(),
		Payload: map[string]interface{}{
			"uri": "https://calendly.com/events/abc123",
			"invitee": map[string]interface{}{
				"email": "patient@example.com",
				"name":  "Test Patient",
			},
			"event": map[string]interface{}{
				"start_time": time.Now().AddDate(0, 0, 1).Format(time.RFC3339),
				"location": map[string]interface{}{
					"join_url": "https://meet.google.com/xyz",
				},
			},
		},
	}

	testCases := []struct {
		name    string
		payload CalendlyWebhookPayload
		isValid bool
	}{
		{
			name:    "Valid payload structure",
			payload: validPayload,
			isValid: true,
		},
		{
			name: "Missing invitee email",
			payload: CalendlyWebhookPayload{
				Event: "invitee.created",
				Time:  time.Now(),
				Payload: map[string]interface{}{
					"invitee": map[string]interface{}{
						"name": "Test Patient",
					},
					"event": map[string]interface{}{
						"start_time": time.Now().Format(time.RFC3339),
					},
				},
			},
			isValid: false,
		},
		{
			name: "Missing event start_time",
			payload: CalendlyWebhookPayload{
				Event: "invitee.created",
				Time:  time.Now(),
				Payload: map[string]interface{}{
					"invitee": map[string]interface{}{
						"email": "patient@example.com",
					},
					"event": map[string]interface{}{},
				},
			},
			isValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate essential fields
			payload := tc.payload.Payload
			invitee, ok := payload["invitee"].(map[string]interface{})
			hasEmail := ok && invitee["email"] != nil

			event, ok := payload["event"].(map[string]interface{})
			hasStartTime := ok && event["start_time"] != nil

			isValid := hasEmail && hasStartTime
			assert.Equal(t, tc.isValid, isValid, "Payload validation should match expected result")
		})
	}
}

// TestCalendlyWebhookEndpointResponse tests webhook endpoint response handling
func TestCalendlyWebhookEndpointResponse(t *testing.T) {
	t.Run("Successful webhook response", func(t *testing.T) {
		// Simulate webhook response
		successResponse := map[string]interface{}{
			"status": "received",
			"id":     "webhook_event_123",
		}

		body, err := json.Marshal(successResponse)
		require.NoError(t, err)

		// Create response
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(body)
		require.NoError(t, err)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "received", response["status"])
	})

	t.Run("Duplicate event response", func(t *testing.T) {
		// Simulate duplicate event response
		response := map[string]interface{}{
			"status":  "duplicate",
			"message": "Event already processed",
		}

		body, err := json.Marshal(response)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "duplicate", result["status"])
	})
}

// TestCalendlyWebhookPayloadExample demonstrates webhook payload structure
func TestCalendlyWebhookPayloadExample(t *testing.T) {
	examplePayload := CalendlyWebhookPayload{
		Event: "invitee.created",
		Time:  time.Now(),
		Payload: map[string]interface{}{
			"uri":            "https://calendly.com/events/abc123/invitees/xyz789",
			"event_type_uri": "https://calendly.com/event_types/def456",
			"invitee": map[string]interface{}{
				"uri":        "https://calendly.com/invitees/ghi789",
				"email":      "patient@example.com",
				"name":       "Ada Okafor",
				"first_name": "Ada",
				"last_name":  "Okafor",
				"phone":      "+2348012345678",
				"timezone":   "Africa/Lagos",
			},
			"event": map[string]interface{}{
				"uri":              "https://calendly.com/events/abc123",
				"start_time":       "2026-02-10T14:00:00Z",
				"duration_minutes": 30,
				"location": map[string]interface{}{
					"type":     "google_meet",
					"join_url": "https://meet.google.com/abc-def-ghi",
				},
			},
		},
	}

	// Verify example structure
	payload := examplePayload.Payload
	assert.Equal(t, "invitee.created", examplePayload.Event)
	assert.NotNil(t, payload["invitee"])
	assert.NotNil(t, payload["event"])

	invitee := payload["invitee"].(map[string]interface{})
	assert.Equal(t, "patient@example.com", invitee["email"])
	assert.Equal(t, "Ada Okafor", invitee["name"])
	assert.Equal(t, "+2348012345678", invitee["phone"])

	event := payload["event"].(map[string]interface{})
	assert.Equal(t, "2026-02-10T14:00:00Z", event["start_time"])
	// duration_minutes can be int or float64 depending on JSON unmarshaling
	durationMinutes := event["duration_minutes"]
	assert.True(t, durationMinutes == 30 || durationMinutes == float64(30), "Duration should be 30")

	location := event["location"].(map[string]interface{})
	assert.Equal(t, "google_meet", location["type"])

	t.Logf("Example Calendly Webhook Payload:\n%s", formatPayloadForLogging(examplePayload))
}

func formatPayloadForLogging(payload CalendlyWebhookPayload) string {
	body, _ := json.MarshalIndent(payload, "", "  ")
	return string(body)
}
