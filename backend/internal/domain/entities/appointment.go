package entities

import (
	"time"
)

// AppointmentStatus represents the status of an appointment
type AppointmentStatus string

const (
	AppointmentStatusPending   AppointmentStatus = "pending"
	AppointmentStatusConfirmed AppointmentStatus = "confirmed"
	AppointmentStatusCancelled AppointmentStatus = "cancelled"
	AppointmentStatusCompleted AppointmentStatus = "completed"
)

// Appointment represents a scheduled appointment
type Appointment struct {
	ID                  string            `json:"id" db:"id"`
	UserID              string            `json:"user_id" db:"user_id"`
	FacilityID          string            `json:"facility_id" db:"facility_id"`
	ProcedureID         string            `json:"procedure_id" db:"procedure_id"`
	ScheduledAt         time.Time         `json:"scheduled_at" db:"scheduled_at"`
	Status              AppointmentStatus `json:"status" db:"status"`
	PatientName         string            `json:"patient_name" db:"patient_name"`
	PatientEmail        string            `json:"patient_email" db:"patient_email"`
	PatientPhone        string            `json:"patient_phone" db:"patient_phone"`
	InsuranceProvider   string            `json:"insurance_provider" db:"insurance_provider"`
	InsurancePolicyNumber string          `json:"insurance_policy_number" db:"insurance_policy_number"`
	Notes               string            `json:"notes" db:"notes"`
	CreatedAt           time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at" db:"updated_at"`
}

// AvailabilitySlot represents an available time slot at a facility
type AvailabilitySlot struct {
	ID         string    `json:"id" db:"id"`
	FacilityID string    `json:"facility_id" db:"facility_id"`
	StartTime  time.Time `json:"start_time" db:"start_time"`
	EndTime    time.Time `json:"end_time" db:"end_time"`
	IsBooked   bool      `json:"is_booked" db:"is_booked"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}
