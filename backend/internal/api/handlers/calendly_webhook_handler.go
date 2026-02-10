package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// CalendlyWebhookHandler handles Calendly webhook events
type CalendlyWebhookHandler struct {
	db                  *sqlx.DB
	notificationService *services.NotificationService
	signingSecret       string
}

// NewCalendlyWebhookHandler creates a new webhook handler
func NewCalendlyWebhookHandler(db *sqlx.DB, notificationService *services.NotificationService) *CalendlyWebhookHandler {
	return &CalendlyWebhookHandler{
		db:                  db,
		notificationService: notificationService,
		signingSecret:       os.Getenv("CALENDLY_WEBHOOK_SECRET"),
	}
}

// CalendlyWebhookEvent represents the incoming webhook event
type CalendlyWebhookEvent struct {
	Event   string                 `json:"event"`
	Time    time.Time              `json:"time"`
	Payload map[string]interface{} `json:"payload"`
}

// HandleWebhook processes incoming Calendly webhooks
func (h *CalendlyWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify webhook signature
	if h.signingSecret != "" {
		if !h.verifySignature(r) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// Parse webhook event
	var event CalendlyWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Extract event ID for idempotency
	eventID := h.extractEventID(event.Payload)
	if eventID == "" {
		eventID = uuid.New().String()
	}

	// Check for duplicate event (idempotency)
	if h.isEventProcessed(ctx, eventID) {
		// Already processed, return success
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "already_processed"})
		return
	}

	// Store webhook event
	if err := h.storeWebhookEvent(ctx, eventID, &event); err != nil {
		fmt.Printf("Failed to store webhook event: %v\n", err)
	}

	// Process event based on type
	switch event.Event {
	case "invitee.created":
		if err := h.handleInviteeCreated(ctx, event.Payload); err != nil {
			h.markEventFailed(ctx, eventID, err)
			http.Error(w, fmt.Sprintf("Processing error: %v", err), http.StatusInternalServerError)
			return
		}
	case "invitee.canceled":
		if err := h.handleInviteeCanceled(ctx, event.Payload); err != nil {
			h.markEventFailed(ctx, eventID, err)
			http.Error(w, fmt.Sprintf("Processing error: %v", err), http.StatusInternalServerError)
			return
		}
	default:
		fmt.Printf("Unhandled Calendly event type: %s\n", event.Event)
	}

	// Mark event as processed
	h.markEventProcessed(ctx, eventID)

	// Return success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "processed"})
}

// handleInviteeCreated processes invitee.created events
func (h *CalendlyWebhookHandler) handleInviteeCreated(ctx context.Context, payload map[string]interface{}) error {
	// Extract event details
	eventURI, _ := payload["uri"].(string)

	// Extract invitee info
	invitee, ok := payload["invitee"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing invitee in payload")
	}

	email, _ := invitee["email"].(string)

	// Extract event info
	eventInfo, ok := payload["event"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing event in payload")
	}

	startTime, _ := eventInfo["start_time"].(string)
	meetingLink, _ := eventInfo["location"].(map[string]interface{})["join_url"].(string)

	// Find appointment by email and scheduled time
	appointment, err := h.findAppointmentByEmail(ctx, email, startTime)
	if err != nil {
		return fmt.Errorf("failed to find appointment: %w", err)
	}

	if appointment == nil {
		return fmt.Errorf("no matching appointment found for email %s", email)
	}

	// Update appointment with Calendly info
	appointment.CalendlyEventURI = &eventURI
	appointment.MeetingLink = &meetingLink
	appointment.BookingMethod = entities.BookingMethodCalendly
	appointment.Status = entities.AppointmentStatusConfirmed
	appointment.UpdatedAt = time.Now()

	if err := h.updateAppointment(ctx, appointment); err != nil {
		return fmt.Errorf("failed to update appointment: %w", err)
	}

	// Get facility and procedure info for notification
	facility, err := h.getFacility(ctx, appointment.FacilityID)
	if err != nil {
		fmt.Printf("Failed to get facility: %v\n", err)
		return nil // Don't fail webhook processing
	}

	procedure, err := h.getProcedure(ctx, appointment.ProcedureID)
	if err != nil {
		fmt.Printf("Failed to get procedure: %v\n", err)
		return nil
	}

	// Send booking confirmation
	if err := h.notificationService.SendBookingConfirmation(ctx, appointment, facility, procedure); err != nil {
		fmt.Printf("Failed to send booking confirmation: %v\n", err)
		// Don't fail the webhook - notification failure is not critical
	}

	return nil
}

// handleInviteeCanceled processes invitee.canceled events
func (h *CalendlyWebhookHandler) handleInviteeCanceled(ctx context.Context, payload map[string]interface{}) error {
	eventURI, _ := payload["uri"].(string)

	// Find appointment by Calendly event URI
	appointment, err := h.findAppointmentByCalendlyURI(ctx, eventURI)
	if err != nil {
		return fmt.Errorf("failed to find appointment: %w", err)
	}

	if appointment == nil {
		return fmt.Errorf("no appointment found for event URI %s", eventURI)
	}

	// Update appointment status
	appointment.Status = entities.AppointmentStatusCancelled
	appointment.UpdatedAt = time.Now()

	if err := h.updateAppointment(ctx, appointment); err != nil {
		return fmt.Errorf("failed to update appointment: %w", err)
	}

	// Get facility and procedure info
	facility, err := h.getFacility(ctx, appointment.FacilityID)
	if err != nil {
		fmt.Printf("Failed to get facility: %v\n", err)
		return nil
	}

	procedure, err := h.getProcedure(ctx, appointment.ProcedureID)
	if err != nil {
		fmt.Printf("Failed to get procedure: %v\n", err)
		return nil
	}

	// Send cancellation notice
	if err := h.notificationService.SendCancellationNotice(ctx, appointment, facility, procedure); err != nil {
		fmt.Printf("Failed to send cancellation notice: %v\n", err)
	}

	return nil
}

// verifySignature verifies the webhook signature
func (h *CalendlyWebhookHandler) verifySignature(r *http.Request) bool {
	signature := r.Header.Get("Calendly-Webhook-Signature")
	if signature == "" {
		return false
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}

	// Reset body for later reading
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Compute HMAC
	mac := hmac.New(sha256.New, []byte(h.signingSecret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// Database operations
func (h *CalendlyWebhookHandler) isEventProcessed(ctx context.Context, eventID string) bool {
	var count int
	query := `SELECT COUNT(*) FROM webhook_events WHERE id = $1 AND provider = 'calendly' AND processed = true`
	h.db.GetContext(ctx, &count, query, eventID)
	return count > 0
}

func (h *CalendlyWebhookHandler) storeWebhookEvent(ctx context.Context, eventID string, event *CalendlyWebhookEvent) error {
	payload, _ := json.Marshal(event.Payload)
	query := `
		INSERT INTO webhook_events (id, provider, event_type, payload, processed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (provider, id) DO NOTHING
	`
	_, err := h.db.ExecContext(ctx, query, eventID, "calendly", event.Event, payload, false, time.Now())
	return err
}

func (h *CalendlyWebhookHandler) markEventProcessed(ctx context.Context, eventID string) error {
	query := `UPDATE webhook_events SET processed = true, processed_at = $1 WHERE id = $2 AND provider = 'calendly'`
	_, err := h.db.ExecContext(ctx, query, time.Now(), eventID)
	return err
}

func (h *CalendlyWebhookHandler) markEventFailed(ctx context.Context, eventID string, err error) {
	errMsg := err.Error()
	query := `UPDATE webhook_events SET error_message = $1 WHERE id = $2 AND provider = 'calendly'`
	h.db.ExecContext(ctx, query, errMsg, eventID)
}

func (h *CalendlyWebhookHandler) findAppointmentByEmail(ctx context.Context, email, startTime string) (*entities.Appointment, error) {
	var appointment entities.Appointment
	query := `
		SELECT * FROM appointments 
		WHERE patient_email = $1 
		AND scheduled_at::date = $2::date
		AND status = 'pending'
		LIMIT 1
	`
	err := h.db.GetContext(ctx, &appointment, query, email, startTime)
	if err != nil {
		return nil, err
	}
	return &appointment, nil
}

func (h *CalendlyWebhookHandler) findAppointmentByCalendlyURI(ctx context.Context, uri string) (*entities.Appointment, error) {
	var appointment entities.Appointment
	query := `SELECT * FROM appointments WHERE calendly_event_uri = $1 LIMIT 1`
	err := h.db.GetContext(ctx, &appointment, query, uri)
	if err != nil {
		return nil, err
	}
	return &appointment, nil
}

func (h *CalendlyWebhookHandler) updateAppointment(ctx context.Context, appointment *entities.Appointment) error {
	query := `
		UPDATE appointments 
		SET status = $1, calendly_event_uri = $2, meeting_link = $3, 
		    booking_method = $4, updated_at = $5
		WHERE id = $6
	`
	_, err := h.db.ExecContext(ctx, query,
		appointment.Status, appointment.CalendlyEventURI, appointment.MeetingLink,
		appointment.BookingMethod, appointment.UpdatedAt, appointment.ID,
	)
	return err
}

func (h *CalendlyWebhookHandler) getFacility(ctx context.Context, facilityID string) (*entities.Facility, error) {
	var facility entities.Facility
	query := `SELECT * FROM facilities WHERE id = $1 LIMIT 1`
	err := h.db.GetContext(ctx, &facility, query, facilityID)
	if err != nil {
		return nil, err
	}
	return &facility, nil
}

func (h *CalendlyWebhookHandler) getProcedure(ctx context.Context, procedureID string) (*entities.Procedure, error) {
	var procedure entities.Procedure
	query := `SELECT * FROM procedures WHERE id = $1 LIMIT 1`
	err := h.db.GetContext(ctx, &procedure, query, procedureID)
	if err != nil {
		return nil, err
	}
	return &procedure, nil
}

func (h *CalendlyWebhookHandler) extractEventID(payload map[string]interface{}) string {
	if uri, ok := payload["uri"].(string); ok {
		return uri
	}
	if event, ok := payload["event"].(map[string]interface{}); ok {
		if uri, ok := event["uri"].(string); ok {
			return uri
		}
	}
	return ""
}
