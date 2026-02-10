package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/notifications"
)

// NotificationService handles sending notifications
type NotificationService struct {
	db             *sqlx.DB
	whatsappSender *notifications.WhatsAppCloudSender
}

// NewNotificationService creates a new notification service
func NewNotificationService(db *sqlx.DB) (*NotificationService, error) {
	whatsappSender, err := notifications.NewWhatsAppCloudSender()
	if err != nil {
		return nil, fmt.Errorf("failed to create WhatsApp sender: %w", err)
	}

	return &NotificationService{
		db:             db,
		whatsappSender: whatsappSender,
	}, nil
}

// NotificationContext contains all data needed for notification rendering
type NotificationContext struct {
	AppointmentID     string
	PatientName       string
	PatientEmail      string
	PatientPhone      string
	FacilityName      string
	FacilityAddress   string
	ProcedureName     string
	ScheduledDate     string
	ScheduledTime     string
	MeetingLink       *string
	InsuranceProvider string
	Notes             string
}

// SendBookingConfirmation sends a booking confirmation notification
func (n *NotificationService) SendBookingConfirmation(ctx context.Context, appointment *entities.Appointment, facility *entities.Facility, procedure *entities.Procedure) error {
	// Get notification preferences
	prefs, err := n.getNotificationPreferences(ctx, appointment.PatientPhone)
	if err != nil {
		// If no preferences, use defaults (WhatsApp enabled)
		prefs = &entities.NotificationPreference{
			Phone:           &appointment.PatientPhone,
			Email:           &appointment.PatientEmail,
			WhatsAppEnabled: true,
			EmailEnabled:    true,
		}
	}

	// Build notification context
	notifCtx := &NotificationContext{
		AppointmentID:     appointment.ID,
		PatientName:       appointment.PatientName,
		PatientEmail:      appointment.PatientEmail,
		PatientPhone:      appointment.PatientPhone,
		FacilityName:      facility.Name,
		FacilityAddress:   fmt.Sprintf("%s, %s", facility.Address.Street, facility.Address.City),
		ProcedureName:     procedure.Name,
		ScheduledDate:     appointment.ScheduledAt.Format("Monday, January 2, 2006"),
		ScheduledTime:     appointment.ScheduledAt.Format("3:04 PM"),
		MeetingLink:       appointment.MeetingLink,
		InsuranceProvider: appointment.InsuranceProvider,
		Notes:             appointment.Notes,
	}

	// Send WhatsApp notification if enabled
	if prefs.WhatsAppEnabled && prefs.Phone != nil && *prefs.Phone != "" {
		if err := n.sendWhatsAppNotification(ctx, entities.NotificationBookingConfirmation, notifCtx); err != nil {
			// Log error but don't fail - try other channels
			fmt.Printf("Failed to send WhatsApp notification: %v\n", err)
		}
	}

	return nil
}

// SendCancellationNotice sends a cancellation notice
func (n *NotificationService) SendCancellationNotice(ctx context.Context, appointment *entities.Appointment, facility *entities.Facility, procedure *entities.Procedure) error {
	prefs, err := n.getNotificationPreferences(ctx, appointment.PatientPhone)
	if err != nil {
		prefs = &entities.NotificationPreference{
			Phone:           &appointment.PatientPhone,
			WhatsAppEnabled: true,
		}
	}

	notifCtx := &NotificationContext{
		AppointmentID: appointment.ID,
		PatientName:   appointment.PatientName,
		PatientPhone:  appointment.PatientPhone,
		FacilityName:  facility.Name,
		ProcedureName: procedure.Name,
		ScheduledDate: appointment.ScheduledAt.Format("Monday, January 2, 2006"),
		ScheduledTime: appointment.ScheduledAt.Format("3:04 PM"),
	}

	if prefs.WhatsAppEnabled && prefs.Phone != nil {
		if err := n.sendWhatsAppNotification(ctx, entities.NotificationCancellation, notifCtx); err != nil {
			fmt.Printf("Failed to send WhatsApp cancellation: %v\n", err)
		}
	}

	return nil
}

// SendReminder sends a reminder notification
func (n *NotificationService) SendReminder(ctx context.Context, appointment *entities.Appointment, facility *entities.Facility, procedure *entities.Procedure, reminderType entities.NotificationType) error {
	prefs, err := n.getNotificationPreferences(ctx, appointment.PatientPhone)
	if err != nil {
		prefs = &entities.NotificationPreference{
			Phone:              &appointment.PatientPhone,
			WhatsAppEnabled:    true,
			Reminder24hEnabled: true,
			Reminder1hEnabled:  true,
		}
	}

	// Check if this reminder type is enabled
	if reminderType == entities.NotificationReminder24h && !prefs.Reminder24hEnabled {
		return nil
	}
	if reminderType == entities.NotificationReminder1h && !prefs.Reminder1hEnabled {
		return nil
	}

	notifCtx := &NotificationContext{
		AppointmentID:   appointment.ID,
		PatientName:     appointment.PatientName,
		PatientPhone:    appointment.PatientPhone,
		FacilityName:    facility.Name,
		FacilityAddress: fmt.Sprintf("%s, %s", facility.Address.Street, facility.Address.City),
		ProcedureName:   procedure.Name,
		ScheduledDate:   appointment.ScheduledAt.Format("Monday, January 2, 2006"),
		ScheduledTime:   appointment.ScheduledAt.Format("3:04 PM"),
		MeetingLink:     appointment.MeetingLink,
	}

	if prefs.WhatsAppEnabled && prefs.Phone != nil {
		if err := n.sendWhatsAppNotification(ctx, reminderType, notifCtx); err != nil {
			fmt.Printf("Failed to send WhatsApp reminder: %v\n", err)
		}
	}

	return nil
}

// sendWhatsAppNotification sends a WhatsApp notification
func (n *NotificationService) sendWhatsAppNotification(ctx context.Context, notifType entities.NotificationType, notifCtx *NotificationContext) error {
	// Get template
	template, err := n.getTemplate(ctx, entities.ChannelWhatsApp, notifType)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// Render message body
	body := n.renderTemplate(template.Body, notifCtx)

	// Create notification record
	notification := &entities.AppointmentNotification{
		ID:               uuid.New().String(),
		AppointmentID:    notifCtx.AppointmentID,
		NotificationType: notifType,
		Channel:          entities.ChannelWhatsApp,
		Recipient:        notifCtx.PatientPhone,
		Status:           entities.NotificationStatusPending,
		RetryCount:       0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Save notification record
	if err := n.createNotification(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification record: %w", err)
	}

	// Send via WhatsApp
	var messageID string
	var sendErr error

	if template.WhatsAppTemplateName != nil && *template.WhatsAppTemplateName != "" {
		// Use approved template
		parameters := n.extractTemplateParameters(notifCtx)
		messageID, sendErr = n.whatsappSender.SendTemplate(
			notifCtx.PatientPhone,
			*template.WhatsAppTemplateName,
			template.WhatsAppTemplateLang,
			parameters,
		)
	} else {
		// Use freeform text (for testing or if template not approved)
		messageID, sendErr = n.whatsappSender.SendText(notifCtx.PatientPhone, body)
	}

	// Update notification status
	if sendErr != nil {
		now := time.Now()
		errMsg := sendErr.Error()
		notification.Status = entities.NotificationStatusFailed
		notification.FailedAt = &now
		notification.ErrorMessage = &errMsg
		notification.UpdatedAt = now
	} else {
		now := time.Now()
		notification.Status = entities.NotificationStatusSent
		notification.MessageID = &messageID
		notification.SentAt = &now
		notification.UpdatedAt = now
	}

	if err := n.updateNotification(ctx, notification); err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}

	return sendErr
}

// renderTemplate replaces placeholders in template
func (n *NotificationService) renderTemplate(template string, ctx *NotificationContext) string {
	replacements := map[string]string{
		"{{patient_name}}":       ctx.PatientName,
		"{{facility_name}}":      ctx.FacilityName,
		"{{facility_address}}":   ctx.FacilityAddress,
		"{{procedure_name}}":     ctx.ProcedureName,
		"{{scheduled_date}}":     ctx.ScheduledDate,
		"{{scheduled_time}}":     ctx.ScheduledTime,
		"{{insurance_provider}}": ctx.InsuranceProvider,
		"{{notes}}":              ctx.Notes,
	}

	// Handle meeting link conditionally
	if ctx.MeetingLink != nil && *ctx.MeetingLink != "" {
		replacements["{{meeting_link}}"] = *ctx.MeetingLink
		template = strings.ReplaceAll(template, "{{#if meeting_link}}", "")
		template = strings.ReplaceAll(template, "{{/if}}", "")
	} else {
		// Remove conditional section
		start := strings.Index(template, "{{#if meeting_link}}")
		if start >= 0 {
			end := strings.Index(template[start:], "{{/if}}")
			if end >= 0 {
				template = template[:start] + template[start+end+7:]
			}
		}
	}

	result := template
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// extractTemplateParameters extracts parameters for WhatsApp template
func (n *NotificationService) extractTemplateParameters(ctx *NotificationContext) []string {
	params := []string{
		ctx.ScheduledDate,
		ctx.ScheduledTime,
		ctx.FacilityName,
	}

	if ctx.FacilityAddress != "" {
		params = append(params, ctx.FacilityAddress)
	}

	if ctx.MeetingLink != nil && *ctx.MeetingLink != "" {
		params = append(params, *ctx.MeetingLink)
	}

	return params
}

// Database operations
func (n *NotificationService) getNotificationPreferences(ctx context.Context, phone string) (*entities.NotificationPreference, error) {
	var prefs entities.NotificationPreference
	query := `SELECT * FROM notification_preferences WHERE phone = $1 LIMIT 1`
	err := n.db.GetContext(ctx, &prefs, query, phone)
	if err != nil {
		return nil, err
	}
	return &prefs, nil
}

func (n *NotificationService) getTemplate(ctx context.Context, channel entities.NotificationChannel, notifType entities.NotificationType) (*entities.NotificationTemplate, error) {
	var template entities.NotificationTemplate
	query := `SELECT * FROM notification_templates WHERE channel = $1 AND template_type = $2 AND is_active = true LIMIT 1`
	err := n.db.GetContext(ctx, &template, query, string(channel), string(notifType))
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (n *NotificationService) createNotification(ctx context.Context, notification *entities.AppointmentNotification) error {
	metadata, _ := json.Marshal(notification.Metadata)
	query := `
		INSERT INTO appointment_notifications 
		(id, appointment_id, notification_type, channel, recipient, status, message_id, 
		 sent_at, delivered_at, read_at, failed_at, error_message, retry_count, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	_, err := n.db.ExecContext(ctx, query,
		notification.ID, notification.AppointmentID, notification.NotificationType, notification.Channel,
		notification.Recipient, notification.Status, notification.MessageID, notification.SentAt,
		notification.DeliveredAt, notification.ReadAt, notification.FailedAt, notification.ErrorMessage,
		notification.RetryCount, metadata, notification.CreatedAt, notification.UpdatedAt,
	)
	return err
}

func (n *NotificationService) updateNotification(ctx context.Context, notification *entities.AppointmentNotification) error {
	metadata, _ := json.Marshal(notification.Metadata)
	query := `
		UPDATE appointment_notifications 
		SET status = $1, message_id = $2, sent_at = $3, delivered_at = $4, read_at = $5,
		    failed_at = $6, error_message = $7, retry_count = $8, metadata = $9, updated_at = $10
		WHERE id = $11
	`
	_, err := n.db.ExecContext(ctx, query,
		notification.Status, notification.MessageID, notification.SentAt, notification.DeliveredAt,
		notification.ReadAt, notification.FailedAt, notification.ErrorMessage, notification.RetryCount,
		metadata, notification.UpdatedAt, notification.ID,
	)
	return err
}
