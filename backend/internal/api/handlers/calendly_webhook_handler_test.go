package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	db := sqlx.NewDb(mockDB, "postgres")
	return db, mock
}

type mockNotificationService struct {
	sendBookingConfirmationCalled bool
	sendCancellationCalled        bool
	lastAppointment               *entities.Appointment
	returnError                   error
}

func (m *mockNotificationService) SendBookingConfirmation(ctx context.Context, appointment *entities.Appointment, facility *entities.Facility, procedure *entities.Procedure) error {
	m.sendBookingConfirmationCalled = true
	m.lastAppointment = appointment
	return m.returnError
}

func (m *mockNotificationService) SendCancellationNotice(ctx context.Context, appointment *entities.Appointment, facility *entities.Facility, procedure *entities.Procedure) error {
	m.sendCancellationCalled = true
	m.lastAppointment = appointment
	return m.returnError
}

func (m *mockNotificationService) SendReminder(ctx context.Context, appointment *entities.Appointment, facility *entities.Facility, procedure *entities.Procedure, reminderType entities.NotificationType) error {
	return m.returnError
}

func TestCalendlyWebhookHandler_HandleWebhook(t *testing.T) {
	tests := []struct {
		name               string
		eventPayload       CalendlyWebhookEvent
		signingSecret      string
		signRequest        bool
		setupMocks         func(sqlmock.Sqlmock, *mockNotificationService)
		expectedStatusCode int
		expectNotification bool
	}{
		{
			name: "Valid invitee.created event",
			eventPayload: CalendlyWebhookEvent{
				Event: "invitee.created",
				Time:  time.Now(),
				Payload: map[string]interface{}{
					"uri": "https://calendly.com/events/test-123",
					"invitee": map[string]interface{}{
						"email": "test@example.com",
						"name":  "Test Patient",
					},
					"event": map[string]interface{}{
						"start_time": "2026-02-10T14:00:00Z",
						"location": map[string]interface{}{
							"join_url": "https://meet.google.com/abc",
						},
					},
				},
			},
			signingSecret: "test_secret",
			signRequest:   true,
			setupMocks: func(m sqlmock.Sqlmock, ns *mockNotificationService) {
				// Check if event already processed
				m.ExpectQuery("SELECT COUNT\\(\\*\\) FROM webhook_events").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				// Store webhook event
				m.ExpectExec("INSERT INTO webhook_events").
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Find appointment by email and date
				m.ExpectQuery("SELECT \\* FROM appointments").
					WithArgs("test@example.com", "2026-02-10T14:00:00Z").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "user_id", "facility_id", "procedure_id", "scheduled_at",
						"status", "patient_name", "patient_email", "patient_phone",
						"insurance_provider", "insurance_policy_number", "notes",
						"calendly_event_id", "calendly_event_uri", "calendly_invitee_uri",
						"meeting_link", "booking_method", "created_at", "updated_at",
					}).AddRow(
						"appt_123", nil, "facility_123", "procedure_123",
						time.Now(), "pending", "Test Patient", "test@example.com",
						"+2348001234567", "", "", "", nil, nil, nil, nil, "manual",
						time.Now(), time.Now(),
					))

				// Update appointment
				m.ExpectExec("UPDATE appointments").
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Get facility
				m.ExpectQuery("SELECT \\* FROM facilities").
					WithArgs("facility_123").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "phone_number", "whatsapp_number", "email", "website",
						"description", "facility_type", "scheduling_external_id", "rating",
						"review_count", "capacity_status", "ward_statuses", "avg_wait_minutes",
						"urgent_care_available", "is_active", "created_at", "updated_at",
					}).AddRow(
						"facility_123", "Test Hospital", "+2348001234567", "+2348001234567", "info@test.com", "test.com",
						"", "general", "", 4.5,
						0, nil, []byte(`{}`), nil,
						nil, true, time.Now(), time.Now(),
					))

				// Get procedure
				m.ExpectQuery("SELECT \\* FROM procedures").
					WithArgs("procedure_123").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "description", "category", "created_at", "updated_at",
					}).AddRow(
						"procedure_123", "X-Ray", "Chest X-Ray", "Imaging",
						time.Now(), time.Now(),
					))

				// Mark event as processed
				m.ExpectExec("UPDATE webhook_events SET processed").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatusCode: http.StatusOK,
			expectNotification: true,
		},
		{
			name: "Duplicate event (already processed)",
			eventPayload: CalendlyWebhookEvent{
				Event: "invitee.created",
				Time:  time.Now(),
				Payload: map[string]interface{}{
					"uri": "https://calendly.com/events/duplicate-123",
				},
			},
			signingSecret: "",
			signRequest:   false,
			setupMocks: func(m sqlmock.Sqlmock, ns *mockNotificationService) {
				// Event already processed
				m.ExpectQuery("SELECT COUNT\\(\\*\\) FROM webhook_events").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			expectedStatusCode: http.StatusOK,
			expectNotification: false,
		},
		{
			name: "Invalid signature",
			eventPayload: CalendlyWebhookEvent{
				Event: "invitee.created",
				Time:  time.Now(),
				Payload: map[string]interface{}{
					"uri": "https://calendly.com/events/test-123",
				},
			},
			signingSecret:      "test_secret",
			signRequest:        false, // Don't sign the request
			setupMocks:         func(m sqlmock.Sqlmock, ns *mockNotificationService) {},
			expectedStatusCode: http.StatusUnauthorized,
			expectNotification: false,
		},
		{
			name: "Valid invitee.canceled event",
			eventPayload: CalendlyWebhookEvent{
				Event: "invitee.canceled",
				Time:  time.Now(),
				Payload: map[string]interface{}{
					"uri": "https://calendly.com/events/canceled-123",
				},
			},
			signingSecret: "",
			signRequest:   false,
			setupMocks: func(m sqlmock.Sqlmock, ns *mockNotificationService) {
				// Check if event already processed
				m.ExpectQuery("SELECT COUNT\\(\\*\\) FROM webhook_events").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				// Store webhook event
				m.ExpectExec("INSERT INTO webhook_events").
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Find appointment by Calendly URI
				eventURI := "https://calendly.com/events/canceled-123"
				m.ExpectQuery("SELECT \\* FROM appointments WHERE calendly_event_uri").
					WithArgs(eventURI).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "user_id", "facility_id", "procedure_id", "scheduled_at",
						"status", "patient_name", "patient_email", "patient_phone",
						"insurance_provider", "insurance_policy_number", "notes",
						"calendly_event_id", "calendly_event_uri", "calendly_invitee_uri",
						"meeting_link", "booking_method", "created_at", "updated_at",
					}).AddRow(
						"appt_456", nil, "facility_123", "procedure_123",
						time.Now(), "confirmed", "Test Patient", "test@example.com",
						"+2348001234567", "", "", "", "event_123", eventURI, nil,
						nil, "calendly", time.Now(), time.Now(),
					))

				// Update appointment status to cancelled
				m.ExpectExec("UPDATE appointments").
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Get facility
				m.ExpectQuery("SELECT \\* FROM facilities").
					WithArgs("facility_123").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "phone_number", "whatsapp_number", "email", "website",
						"description", "facility_type", "scheduling_external_id", "rating",
						"review_count", "capacity_status", "ward_statuses", "avg_wait_minutes",
						"urgent_care_available", "is_active", "created_at", "updated_at",
					}).AddRow(
						"facility_123", "Test Hospital", "+2348001234567", "+2348001234567", "info@test.com", "test.com",
						"", "general", "", 4.5,
						0, nil, []byte(`{}`), nil,
						nil, true, time.Now(), time.Now(),
					))

				// Get procedure
				m.ExpectQuery("SELECT \\* FROM procedures").
					WithArgs("procedure_123").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "description", "category", "created_at", "updated_at",
					}).AddRow(
						"procedure_123", "X-Ray", "Chest X-Ray", "Imaging",
						time.Now(), time.Now(),
					))

				// Mark event as processed
				m.ExpectExec("UPDATE webhook_events SET processed").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatusCode: http.StatusOK,
			expectNotification: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock expectations
			db, mock := setupMockDB(t)
			defer db.Close()

			mockNotifService := &mockNotificationService{}
			tt.setupMocks(mock, mockNotifService)

			handler := &CalendlyWebhookHandler{
				db:                  db,
				notificationService: mockNotifService,
				signingSecret:       tt.signingSecret,
			}

			// Prepare request
			body, _ := json.Marshal(tt.eventPayload)
			req := httptest.NewRequest("POST", "/webhooks/calendly", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Add signature if needed
			if tt.signRequest && tt.signingSecret != "" {
				mac := hmac.New(sha256.New, []byte(tt.signingSecret))
				mac.Write(body)
				signature := hex.EncodeToString(mac.Sum(nil))
				req.Header.Set("Calendly-Webhook-Signature", signature)
			}

			// Execute request
			rr := httptest.NewRecorder()
			handler.HandleWebhook(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatusCode {
				t.Errorf("HandleWebhook() status = %v, want %v", rr.Code, tt.expectedStatusCode)
			}

			// Check if notification was sent
			if tt.expectNotification {
				if tt.eventPayload.Event == "invitee.created" && !mockNotifService.sendBookingConfirmationCalled {
					t.Error("Expected SendBookingConfirmation to be called")
				}
				if tt.eventPayload.Event == "invitee.canceled" && !mockNotifService.sendCancellationCalled {
					t.Error("Expected SendCancellationNotice to be called")
				}
			}

			// Verify all expectations were met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestCalendlyWebhookHandler_ExtractEventID(t *testing.T) {
	handler := &CalendlyWebhookHandler{}

	tests := []struct {
		name    string
		payload map[string]interface{}
		want    string
	}{
		{
			name: "URI in top level",
			payload: map[string]interface{}{
				"uri": "https://calendly.com/events/test-123",
			},
			want: "https://calendly.com/events/test-123",
		},
		{
			name: "URI in event object",
			payload: map[string]interface{}{
				"event": map[string]interface{}{
					"uri": "https://calendly.com/events/test-456",
				},
			},
			want: "https://calendly.com/events/test-456",
		},
		{
			name:    "No URI",
			payload: map[string]interface{}{},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.extractEventID(tt.payload)
			if got != tt.want {
				t.Errorf("extractEventID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalendlyWebhookHandler_VerifySignature(t *testing.T) {
	secret := "test_secret"
	handler := &CalendlyWebhookHandler{
		signingSecret: secret,
	}

	body := []byte(`{"event":"invitee.created"}`)

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	validSignature := hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		signature string
		body      []byte
		want      bool
	}{
		{
			name:      "Valid signature",
			signature: validSignature,
			body:      body,
			want:      true,
		},
		{
			name:      "Invalid signature",
			signature: "invalid_signature",
			body:      body,
			want:      false,
		},
		{
			name:      "Missing signature",
			signature: "",
			body:      body,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/webhooks/calendly", bytes.NewBuffer(tt.body))
			if tt.signature != "" {
				req.Header.Set("Calendly-Webhook-Signature", tt.signature)
			}

			got := handler.verifySignature(req)
			if got != tt.want {
				t.Errorf("verifySignature() = %v, want %v", got, tt.want)
			}
		})
	}
}
