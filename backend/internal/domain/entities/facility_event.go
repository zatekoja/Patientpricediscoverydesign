package entities

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// FacilityEventType represents the type of facility event
type FacilityEventType string

const (
	FacilityEventTypeCapacityUpdate            FacilityEventType = "capacity_update"
	FacilityEventTypeWardCapacityUpdate        FacilityEventType = "ward_capacity_update"
	FacilityEventTypeWaitTimeUpdate            FacilityEventType = "wait_time_update"
	FacilityEventTypeUrgentCareUpdate          FacilityEventType = "urgent_care_update"
	FacilityEventTypeServiceHealthUpdate       FacilityEventType = "service_health_update"
	FacilityEventTypeServiceAvailabilityUpdate FacilityEventType = "service_availability_update"
)

// FacilityEvent represents a real-time update event for a facility
type FacilityEvent struct {
	ID            string                 `json:"id"`
	FacilityID    string                 `json:"facility_id"`
	EventType     FacilityEventType      `json:"event_type"`
	Timestamp     time.Time              `json:"timestamp"`
	Location      Location               `json:"location"`
	ChangedFields map[string]interface{} `json:"changed_fields"`
}

// NewFacilityEvent creates a new facility event
func NewFacilityEvent(facilityID string, eventType FacilityEventType, location Location, changedFields map[string]interface{}) *FacilityEvent {
	return &FacilityEvent{
		ID:            generateEventID(),
		FacilityID:    facilityID,
		EventType:     eventType,
		Timestamp:     time.Now(),
		Location:      location,
		ChangedFields: changedFields,
	}
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	bytes := make([]byte, length/2+1)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based if crypto/rand fails
		return time.Now().Format("150405.000")
	}
	return hex.EncodeToString(bytes)[:length]
}
