package entities

import "time"

// NotificationPreference represents user notification settings
type NotificationPreference struct {
	ID                 string    `json:"id" db:"id"`
	UserID             string    `json:"user_id" db:"user_id"`
	Phone              *string   `json:"phone,omitempty" db:"phone"`
	Email              *string   `json:"email,omitempty" db:"email"`
	WhatsAppEnabled    bool      `json:"whatsapp_enabled" db:"whatsapp_enabled"`
	EmailEnabled       bool      `json:"email_enabled" db:"email_enabled"`
	SMSEnabled         bool      `json:"sms_enabled" db:"sms_enabled"`
	Reminder24hEnabled bool      `json:"reminder_24h_enabled" db:"reminder_24h_enabled"`
	Reminder1hEnabled  bool      `json:"reminder_1h_enabled" db:"reminder_1h_enabled"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// NotificationChannel represents the delivery channel
type NotificationChannel string

const (
	ChannelWhatsApp NotificationChannel = "whatsapp"
	ChannelEmail    NotificationChannel = "email"
	ChannelSMS      NotificationChannel = "sms"
)

// NotificationType represents the notification purpose
type NotificationType string

const (
	NotificationBookingConfirmation NotificationType = "booking_confirmation"
	NotificationReminder24h         NotificationType = "reminder_24h"
	NotificationReminder1h          NotificationType = "reminder_1h"
	NotificationCancellation        NotificationType = "cancellation"
	NotificationRescheduled         NotificationType = "rescheduled"
)

// NotificationStatus represents the delivery status
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusDelivered NotificationStatus = "delivered"
	NotificationStatusRead      NotificationStatus = "read"
	NotificationStatusFailed    NotificationStatus = "failed"
)

// AppointmentNotification tracks sent notifications
type AppointmentNotification struct {
	ID               string                 `json:"id" db:"id"`
	AppointmentID    string                 `json:"appointment_id" db:"appointment_id"`
	NotificationType NotificationType       `json:"notification_type" db:"notification_type"`
	Channel          NotificationChannel    `json:"channel" db:"channel"`
	Recipient        string                 `json:"recipient" db:"recipient"`
	Status           NotificationStatus     `json:"status" db:"status"`
	MessageID        *string                `json:"message_id,omitempty" db:"message_id"`
	SentAt           *time.Time             `json:"sent_at,omitempty" db:"sent_at"`
	DeliveredAt      *time.Time             `json:"delivered_at,omitempty" db:"delivered_at"`
	ReadAt           *time.Time             `json:"read_at,omitempty" db:"read_at"`
	FailedAt         *time.Time             `json:"failed_at,omitempty" db:"failed_at"`
	ErrorMessage     *string                `json:"error_message,omitempty" db:"error_message"`
	RetryCount       int                    `json:"retry_count" db:"retry_count"`
	Metadata         map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

// WebhookEvent stores received webhook events for idempotency
type WebhookEvent struct {
	ID           string                 `json:"id" db:"id"`
	Provider     string                 `json:"provider" db:"provider"`
	EventType    string                 `json:"event_type" db:"event_type"`
	Payload      map[string]interface{} `json:"payload" db:"payload"`
	Processed    bool                   `json:"processed" db:"processed"`
	ProcessedAt  *time.Time             `json:"processed_at,omitempty" db:"processed_at"`
	ErrorMessage *string                `json:"error_message,omitempty" db:"error_message"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
}

// NotificationTemplate defines reusable message templates
type NotificationTemplate struct {
	ID                   string              `json:"id" db:"id"`
	Name                 string              `json:"name" db:"name"`
	Channel              NotificationChannel `json:"channel" db:"channel"`
	TemplateType         string              `json:"template_type" db:"template_type"`
	Subject              *string             `json:"subject,omitempty" db:"subject"`
	Body                 string              `json:"body" db:"body"`
	WhatsAppTemplateName *string             `json:"whatsapp_template_name,omitempty" db:"whatsapp_template_name"`
	WhatsAppTemplateLang string              `json:"whatsapp_template_lang" db:"whatsapp_template_lang"`
	IsActive             bool                `json:"is_active" db:"is_active"`
	CreatedAt            time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at" db:"updated_at"`
}
