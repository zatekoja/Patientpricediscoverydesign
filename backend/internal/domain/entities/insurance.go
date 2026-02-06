package entities

import (
	"time"
)

// InsuranceProvider represents an insurance provider
type InsuranceProvider struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Code        string    `json:"code" db:"code"`
	PhoneNumber string    `json:"phone_number" db:"phone_number"`
	Website     string    `json:"website" db:"website"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// FacilityInsurance represents the relationship between a facility and insurance provider
type FacilityInsurance struct {
	ID                  string    `json:"id" db:"id"`
	FacilityID          string    `json:"facility_id" db:"facility_id"`
	InsuranceProviderID string    `json:"insurance_provider_id" db:"insurance_provider_id"`
	IsAccepted          bool      `json:"is_accepted" db:"is_accepted"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}
