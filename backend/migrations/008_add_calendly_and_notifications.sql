-- Migration: Add Calendly tracking and notification system
-- Date: 2026-02-08

-- =====================================================
-- 1. Add Calendly tracking fields to appointments table
-- =====================================================
ALTER TABLE appointments
ADD COLUMN IF NOT EXISTS calendly_event_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS calendly_event_uri TEXT,
ADD COLUMN IF NOT EXISTS calendly_invitee_uri TEXT,
ADD COLUMN IF NOT EXISTS meeting_link TEXT,
ADD COLUMN IF NOT EXISTS booking_method VARCHAR(50) DEFAULT 'manual';

-- Index for faster lookup by Calendly event ID
CREATE INDEX IF NOT EXISTS idx_appointments_calendly_event_id 
ON appointments(calendly_event_id) WHERE calendly_event_id IS NOT NULL;

-- Index for booking method filtering
CREATE INDEX IF NOT EXISTS idx_appointments_booking_method 
ON appointments(booking_method);

-- Add constraint to ensure calendly_event_id is unique when present
CREATE UNIQUE INDEX IF NOT EXISTS idx_appointments_calendly_event_id_unique 
ON appointments(calendly_event_id) WHERE calendly_event_id IS NOT NULL;

-- =====================================================
-- 2. Create notification preferences table
-- =====================================================
CREATE TABLE IF NOT EXISTS notification_preferences (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) REFERENCES users(id) ON DELETE CASCADE,
    phone VARCHAR(50),
    email VARCHAR(255),
    whatsapp_enabled BOOLEAN DEFAULT true,
    email_enabled BOOLEAN DEFAULT true,
    sms_enabled BOOLEAN DEFAULT false,
    reminder_24h_enabled BOOLEAN DEFAULT true,
    reminder_1h_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);

CREATE INDEX IF NOT EXISTS idx_notification_prefs_user 
ON notification_preferences(user_id);

CREATE INDEX IF NOT EXISTS idx_notification_prefs_phone 
ON notification_preferences(phone) WHERE phone IS NOT NULL;

-- =====================================================
-- 3. Create appointment notifications tracking table
-- =====================================================
CREATE TABLE IF NOT EXISTS appointment_notifications (
    id VARCHAR(255) PRIMARY KEY,
    appointment_id VARCHAR(255) NOT NULL REFERENCES appointments(id) ON DELETE CASCADE,
    notification_type VARCHAR(50) NOT NULL,
    channel VARCHAR(20) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL,
    message_id VARCHAR(255),
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    read_at TIMESTAMP,
    failed_at TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_notifications_appointment 
ON appointment_notifications(appointment_id);

CREATE INDEX IF NOT EXISTS idx_notifications_status 
ON appointment_notifications(status);

CREATE INDEX IF NOT EXISTS idx_notifications_type 
ON appointment_notifications(notification_type);

CREATE INDEX IF NOT EXISTS idx_notifications_channel 
ON appointment_notifications(channel);

CREATE INDEX IF NOT EXISTS idx_notifications_recipient 
ON appointment_notifications(recipient);

CREATE INDEX IF NOT EXISTS idx_notifications_sent_at 
ON appointment_notifications(sent_at) WHERE sent_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_notifications_failed 
ON appointment_notifications(status, retry_count) 
WHERE status = 'failed' AND retry_count < 3;

-- =====================================================
-- 4. Create webhook events table for idempotency
-- =====================================================
CREATE TABLE IF NOT EXISTS webhook_events (
    id VARCHAR(255) PRIMARY KEY,
    provider VARCHAR(50) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, id)
);

CREATE INDEX IF NOT EXISTS idx_webhook_events_provider 
ON webhook_events(provider);

CREATE INDEX IF NOT EXISTS idx_webhook_events_processed 
ON webhook_events(processed);

CREATE INDEX IF NOT EXISTS idx_webhook_events_created 
ON webhook_events(created_at);

-- =====================================================
-- 5. Create notification templates table
-- =====================================================
CREATE TABLE IF NOT EXISTS notification_templates (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    channel VARCHAR(20) NOT NULL,
    template_type VARCHAR(50) NOT NULL,
    subject VARCHAR(255),
    body TEXT NOT NULL,
    whatsapp_template_name VARCHAR(100),
    whatsapp_template_lang VARCHAR(10) DEFAULT 'en_US',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notification_templates_name 
ON notification_templates(name);

CREATE INDEX IF NOT EXISTS idx_notification_templates_channel 
ON notification_templates(channel);

CREATE INDEX IF NOT EXISTS idx_notification_templates_active 
ON notification_templates(is_active);

-- =====================================================
-- 6. Insert default notification templates
-- =====================================================
INSERT INTO notification_templates (id, name, channel, template_type, body, whatsapp_template_name)
VALUES 
(
    'tmpl_booking_confirm_wa',
    'booking_confirmation_whatsapp',
    'whatsapp',
    'booking_confirmation',
    '✅ Appointment Confirmed

📅 Date: {{scheduled_date}}
🕐 Time: {{scheduled_time}}
🏥 Facility: {{facility_name}}
📍 Location: {{facility_address}}

{{#if meeting_link}}
🔗 Join: {{meeting_link}}
{{/if}}

Need to cancel? Reply CANCEL or contact us.',
    'appointment_confirmation'
),
(
    'tmpl_reminder_24h_wa',
    'reminder_24h_whatsapp',
    'whatsapp',
    'reminder',
    '⏰ Appointment Reminder

Your appointment is tomorrow!

📅 {{scheduled_date}}
🕐 {{scheduled_time}}
🏥 {{facility_name}}

{{#if meeting_link}}
🔗 {{meeting_link}}
{{/if}}

See you soon!',
    'appointment_reminder_24h'
),
(
    'tmpl_cancellation_wa',
    'cancellation_whatsapp',
    'whatsapp',
    'cancellation',
    '❌ Appointment Canceled

Your appointment has been canceled:

📅 {{scheduled_date}}
🕐 {{scheduled_time}}
🏥 {{facility_name}}

Need to reschedule? Contact us or book a new appointment.',
    'appointment_canceled'
)
ON CONFLICT (name) DO NOTHING;
