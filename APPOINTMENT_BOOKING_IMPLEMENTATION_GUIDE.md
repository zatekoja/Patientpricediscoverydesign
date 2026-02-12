# Appointment Booking + WhatsApp Integration - Implementation Guide

## ✅ Phase 1: Database Migration - COMPLETE

The database migration has been successfully applied with the following changes:

### New Database Structure
- **appointments table**: Added 5 new Calendly tracking columns
  - `calendly_event_id` (VARCHAR 255)
  - `calendly_event_uri` (TEXT)
  - `calendly_invitee_uri` (TEXT)
  - `meeting_link` (TEXT)
  - `booking_method` (VARCHAR 50) - 'manual', 'calendly', 'api'

- **notification_preferences table**: User communication preferences
  - Controls WhatsApp, Email, SMS channels
  - Per-user reminder settings (24h, 1h)

- **appointment_notifications table**: Notification audit trail
  - Tracks delivery status (pending, sent, delivered, read, failed)
  - Retry logic with retry_count
  - Error tracking
  - Metadata JSONB for extensibility

- **webhook_events table**: Idempotency for webhooks
  - Prevents duplicate processing
  - Stores raw payload for debugging
  - Error tracking

- **notification_templates table**: Reusable templates
  - 3 pre-populated WhatsApp templates (confirmation, reminder, cancellation)
  - Template parameter support
  - Multi-channel support

### Verification
```bash
# Check tables were created
docker compose exec postgres psql -U postgres -d patient_price_discovery -c "\dt notification_*"

# Check appointments table columns
docker compose exec postgres psql -U postgres -d patient_price_discovery -c "\d appointments"

# Check templates were inserted
docker compose exec postgres psql -U postgres -d patient_price_discovery -c "SELECT name, channel FROM notification_templates"
```

---

## ✅ Phase 2: Core Services - COMPLETE

### Files Created

1. **backend/internal/domain/entities/notification.go**
   - Domain entities for notification system
   - Enums: NotificationChannel, NotificationType, NotificationStatus
   - Structs: NotificationPreference, AppointmentNotification, WebhookEvent, NotificationTemplate

2. **backend/internal/domain/entities/appointment.go** (Updated)
   - Added BookingMethod enum
   - Added 5 Calendly tracking fields to Appointment struct

3. **backend/internal/infrastructure/notifications/whatsapp_sender.go**
   - WhatsApp Cloud API sender extracted from workflows
   - Supports template messages (approved templates)
   - Supports freeform text messages (for testing)
   - Full error handling and response parsing

4. **backend/internal/application/services/notification_service.go**
   - High-level notification orchestration
   - Methods:
     - `SendBookingConfirmation()` - Sends confirmation after booking
     - `SendCancellationNotice()` - Sends cancellation notice
     - `SendReminder()` - Sends 24h or 1h reminders
   - Template rendering with placeholder replacement
   - Database tracking of all notifications
   - Retry logic for failed sends

5. **backend/internal/api/handlers/calendly_webhook_handler.go**
   - Webhook endpoint: `POST /webhooks/calendly`
   - Signature verification (HMAC SHA256)
   - Idempotency via webhook_events table
   - Event handlers:
     - `invitee.created` - Booking confirmed
     - `invitee.canceled` - Booking canceled
   - Automatic notification triggering

6. **backend/internal/adapters/providers/scheduling/calendly_adapter.go** (Updated)
   - Updated CreateAppointment() to generate scheduling links
   - Added getEventTypeForFacility() mapping helper
   - Webhook-driven flow (link → user books → webhook → confirmation)

---

## 🔄 Phase 3: Backend Integration - IN PROGRESS

### What Needs to be Done

#### 3.1 Update Main Application (Dependency Injection)

**File**: `backend/cmd/api/main.go` or similar

```go
import (
    "backend/internal/application/services"
    "backend/internal/api/handlers"
)

// In main() or service initialization:

// Initialize notification service
notificationService, err := services.NewNotificationService(db)
if err != nil {
    log.Fatal("Failed to create notification service:", err)
}

// Initialize Calendly webhook handler
calendlyWebhookHandler := handlers.NewCalendlyWebhookHandler(db, notificationService)

// Register webhook route
router.HandleFunc("/webhooks/calendly", calendlyWebhookHandler.HandleWebhook).Methods("POST")
```

#### 3.2 Update Appointment Service

**File**: Find your existing appointment service (likely in `backend/internal/application/services/`)

Add notification sending after appointment creation:

```go
func (s *AppointmentService) CreateAppointment(ctx context.Context, req *CreateAppointmentRequest) (*entities.Appointment, error) {
    // ... existing appointment creation code ...
    
    // Generate Calendly link if enabled
    if req.UseCalendly {
        schedulingLink, eventTypeID, err := s.calendlyAdapter.CreateAppointment(ctx, appointment)
        if err != nil {
            return nil, fmt.Errorf("failed to create Calendly link: %w", err)
        }
        appointment.BookingMethod = entities.BookingMethodCalendly
        // Store scheduling link to return to frontend
        // Actual event ID will come via webhook
    }
    
    // Save appointment
    if err := s.appointmentRepo.Create(ctx, appointment); err != nil {
        return nil, err
    }
    
    // Send booking confirmation (async recommended)
    go func() {
        facility, _ := s.facilityRepo.GetByID(context.Background(), appointment.FacilityID)
        procedure, _ := s.procedureRepo.GetByID(context.Background(), appointment.ProcedureID)
        s.notificationService.SendBookingConfirmation(context.Background(), appointment, facility, procedure)
    }()
    
    return appointment, nil
}
```

#### 3.3 Environment Variables

Add to your `.env` or Vault configuration:

```bash
# WhatsApp Cloud API (already configured?)
WHATSAPP_ACCESS_TOKEN=your_token_here
WHATSAPP_PHONE_NUMBER_ID=your_phone_number_id_here
WHATSAPP_TEMPLATE_NAME=appointment_confirmation

# Calendly API
CALENDLY_API_KEY=your_calendly_api_key_here
CALENDLY_WEBHOOK_SECRET=your_webhook_secret_here

# Organization Calendly URL
CALENDLY_ORG_URL=https://calendly.com/your-org
```

---

## 🎨 Phase 4: Frontend Integration - PENDING

### What Needs to be Done

#### 4.1 Update Booking Modal

**File**: `Frontend/src/app/components/FacilityModal.tsx` or similar

Add phone number field and WhatsApp opt-in:

```tsx
const [bookingData, setBookingData] = useState({
  patientName: '',
  patientEmail: '',
  patientPhone: '',  // NEW
  whatsappOptIn: true,  // NEW
  scheduledDate: '',
  notes: ''
});

// In JSX:
<div className="form-group">
  <label>Phone Number (with country code)</label>
  <input
    type="tel"
    value={bookingData.patientPhone}
    onChange={(e) => setBookingData({...bookingData, patientPhone: e.target.value})}
    placeholder="+234 800 123 4567"
    required
  />
</div>

<div className="form-group">
  <label>
    <input
      type="checkbox"
      checked={bookingData.whatsappOptIn}
      onChange={(e) => setBookingData({...bookingData, whatsappOptIn: e.target.checked})}
    />
    Send appointment confirmations via WhatsApp
  </label>
</div>
```

#### 4.2 Update Booking API Call

```tsx
const handleBookAppointment = async () => {
  const response = await api.createAppointment({
    ...bookingData,
    facilityId: facility.id,
    procedureId: selectedProcedure.id,
  });
  
  // If Calendly link is returned, redirect user
  if (response.schedulingLink) {
    window.open(response.schedulingLink, '_blank');
    // Show message: "Complete your booking on Calendly. You'll receive confirmation via WhatsApp"
  }
};
```

---

## 📱 Phase 5: WhatsApp Template Approval - PENDING

### Steps to Submit Templates to Meta

1. **Login to Meta Business Suite**
   - Go to: https://business.facebook.com/
   - Navigate to: WhatsApp Manager → Message Templates

2. **Create Template: Booking Confirmation**
   - Name: `appointment_confirmation`
   - Category: TRANSACTIONAL
   - Language: English (US)
   - Body:
     ```
     ✅ Appointment Confirmed
     
     📅 Date: {{1}}
     🕐 Time: {{2}}
     🏥 Facility: {{3}}
     📍 Location: {{4}}
     🔗 Join: {{5}}
     
     Need to cancel? Reply CANCEL or contact us.
     ```
   - Parameters: 5 (date, time, facility, address, link)

3. **Create Template: 24h Reminder**
   - Name: `appointment_reminder_24h`
   - Category: TRANSACTIONAL
   - Body similar to confirmation

4. **Create Template: Cancellation**
   - Name: `appointment_canceled`
   - Category: TRANSACTIONAL

5. **Wait for Approval** (1-5 business days)

### Testing Before Approval

The notification service falls back to freeform text messages if templates aren't approved yet:

```go
// In notification_service.go
if template.WhatsAppTemplateName != nil && *template.WhatsAppTemplateName != "" {
    // Use approved template
    messageID, sendErr = n.whatsappSender.SendTemplate(...)
} else {
    // Use freeform text (works immediately)
    messageID, sendErr = n.whatsappSender.SendText(notifCtx.PatientPhone, body)
}
```

---

## 🔧 Phase 6: Calendly Configuration - PENDING

### Webhooks Setup

1. **Login to Calendly**
   - Go to: https://calendly.com/integrations/api_webhooks

2. **Create Webhook Subscription**
   - URL: `https://your-domain.com/webhooks/calendly`
   - Events to subscribe to:
     - `invitee.created`
     - `invitee.canceled`
   - Copy the signing secret

3. **Test Webhook**
   ```bash
   # Calendly will send test events
   # Check your logs for incoming webhooks
   ```

### Event Type Mapping

Map your facilities to Calendly event types in database:

```sql
-- Add configuration table
CREATE TABLE calendly_facility_mapping (
    facility_id VARCHAR(255) PRIMARY KEY REFERENCES facilities(id),
    calendly_event_type_uuid VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Example mapping
INSERT INTO calendly_facility_mapping VALUES 
('megalek-facility-id', 'your-calendly-event-type-uuid-here', NOW());
```

Update `calendly_adapter.go` to query this table instead of mock implementation.

---

## 🧪 Testing Guide

### 1. Test Database Migration
```bash
# Verify tables
docker compose exec postgres psql -U postgres -d patient_price_discovery -c "
SELECT table_name 
FROM information_schema.tables 
WHERE table_name LIKE 'notification%' OR table_name = 'webhook_events'
"

# Check appointments columns
docker compose exec postgres psql -U postgres -d patient_price_discovery -c "
SELECT column_name, data_type 
FROM information_schema.columns 
WHERE table_name = 'appointments' 
AND column_name LIKE 'calendly%'
"
```

### 2. Test Notification Service (Unit Test)
```bash
# Create test file: backend/internal/application/services/notification_service_test.go
cd backend
go test ./internal/application/services -v -run TestNotificationService
```

### 3. Test Webhook Handler (Integration Test)
```bash
# Send test webhook
curl -X POST http://localhost:8080/webhooks/calendly \
  -H "Content-Type: application/json" \
  -d '{
    "event": "invitee.created",
    "time": "2026-02-08T12:00:00Z",
    "payload": {
      "uri": "https://calendly.com/events/test-123",
      "invitee": {
        "email": "test@example.com",
        "name": "Test Patient"
      },
      "event": {
        "start_time": "2026-02-10T14:00:00Z",
        "location": {
          "join_url": "https://meet.google.com/abc-defg-hij"
        }
      }
    }
  }'
```

### 4. Test WhatsApp Sending
```bash
# Requires valid WHATSAPP_ACCESS_TOKEN
# Test freeform message
curl -X POST http://localhost:8080/api/test/send-whatsapp \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+2348001234567",
    "message": "Test notification from Patient Price Discovery"
  }'
```

---

## 📊 Monitoring & Observability

### Database Queries for Monitoring

```sql
-- Check notification success rate
SELECT 
    channel,
    notification_type,
    status,
    COUNT(*) as count
FROM appointment_notifications
WHERE created_at >= NOW() - INTERVAL '24 hours'
GROUP BY channel, notification_type, status;

-- Check failed notifications needing retry
SELECT 
    id,
    appointment_id,
    recipient,
    error_message,
    retry_count
FROM appointment_notifications
WHERE status = 'failed' AND retry_count < 3
ORDER BY created_at DESC;

-- Check webhook processing stats
SELECT 
    provider,
    event_type,
    processed,
    COUNT(*) as count
FROM webhook_events
GROUP BY provider, event_type, processed;
```

### Logging

Add structured logging in key places:

```go
log.Printf("[NOTIFICATION] Sent %s via %s to %s: message_id=%s", 
    notifType, channel, recipient, messageID)

log.Printf("[WEBHOOK] Processed %s event: %s (duplicate=%v)", 
    provider, eventType, isDuplicate)
```

---

## 🚀 Deployment Checklist

### Before Going Live

- [ ] Database migration applied successfully
- [ ] Environment variables configured (WhatsApp, Calendly)
- [ ] Webhook endpoint accessible publicly (not localhost)
- [ ] Calendly webhook subscription created and tested
- [ ] WhatsApp templates submitted and approved by Meta
- [ ] Frontend updated with phone number field
- [ ] Integration tests passing
- [ ] Load testing (can handle 100+ notifications/min?)
- [ ] Error alerting configured (Sentry, PagerDuty, etc.)
- [ ] Documentation updated for support team

### Post-Deployment Monitoring

- Monitor notification success rate (target: >95%)
- Monitor webhook processing latency (target: <2s)
- Monitor WhatsApp API rate limits
- Track user opt-out rates
- Gather user feedback on notifications

---

## 🐛 Common Issues & Solutions

### Issue: Notifications Not Sending

**Check**:
1. Environment variables set: `echo $WHATSAPP_ACCESS_TOKEN`
2. Phone number format: Must include country code (e.g., +234...)
3. WhatsApp API logs: Check Meta Business Suite → WhatsApp Manager → Logs
4. Database: Query appointment_notifications for error_message

### Issue: Webhook Not Received

**Check**:
1. Webhook URL is publicly accessible (not localhost)
2. Calendly webhook subscription is active
3. Signature verification not failing (check CALENDLY_WEBHOOK_SECRET)
4. Check webhook_events table for duplicates

### Issue: Template Message Rejected

**Check**:
1. Template approved in Meta Business Suite
2. Template name matches exactly (case-sensitive)
3. Parameter count matches template definition
4. Language code correct (en_US vs en)

---

## 📈 Future Enhancements

1. **Email Notifications**: Add email sender alongside WhatsApp
2. **SMS Fallback**: If WhatsApp fails, try SMS
3. **Reminder Scheduler**: Cron job to send 24h/1h reminders
4. **Two-Way Communication**: Handle WhatsApp replies (cancel, reschedule)
5. **Analytics Dashboard**: Track booking funnel, no-show rates
6. **Multi-Language**: Support for local languages (Yoruba, Hausa, Igbo)
7. **Calendar Integration**: Add to Google Calendar, iCal
8. **Payment Integration**: Collect deposits via Paystack/Flutterwave

---

## 📚 Additional Resources

- [Calendly API Docs](https://developer.calendly.com/api-docs)
- [WhatsApp Cloud API Docs](https://developers.facebook.com/docs/whatsapp/cloud-api)
- [WhatsApp Template Guidelines](https://developers.facebook.com/docs/whatsapp/message-templates/guidelines)
- [HMAC Signature Verification](https://webhooks.fyi/security/hmac)

---

## ✅ Summary

**Completed** (Phases 1-2):
- ✅ Database schema with Calendly tracking and notifications
- ✅ Notification domain entities
- ✅ WhatsApp sender service
- ✅ Notification orchestration service
- ✅ Calendly webhook handler
- ✅ Updated Calendly adapter

**Next Steps** (Phases 3-6):
- Wire services in main application (DI)
- Update appointment service to send notifications
- Update frontend with phone number field
- Submit WhatsApp templates for approval
- Configure Calendly webhooks
- Testing and deployment

**Estimated Time to Production**:
- Backend integration: 2-3 hours
- Frontend updates: 1-2 hours
- Template approval: 1-5 days (Meta review)
- Testing: 1-2 hours
- **Total**: 1-2 weeks (mostly waiting for template approval)
