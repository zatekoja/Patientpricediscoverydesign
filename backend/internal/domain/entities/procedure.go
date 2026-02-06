package entities

import (
	"time"
)

// Procedure represents a medical procedure/service
type Procedure struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Code        string    `json:"code" db:"code"` // CPT/HCPCS code
	Category    string    `json:"category" db:"category"`
	Description string    `json:"description" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// FacilityProcedure represents pricing for a procedure at a specific facility
type FacilityProcedure struct {
	ID                string    `json:"id" db:"id"`
	FacilityID        string    `json:"facility_id" db:"facility_id"`
	ProcedureID       string    `json:"procedure_id" db:"procedure_id"`
	Price             float64   `json:"price" db:"price"`
	Currency          string    `json:"currency" db:"currency"`
	EstimatedDuration int       `json:"estimated_duration" db:"estimated_duration"` // in minutes
	IsAvailable       bool      `json:"is_available" db:"is_available"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}
