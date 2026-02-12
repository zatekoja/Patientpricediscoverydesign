package entities

import (
	"time"
)

// Procedure represents a medical procedure/service
type Procedure struct {
	ID             string    `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	DisplayName    string    `json:"display_name" db:"display_name"` // Normalized human-readable name
	Code           string    `json:"code" db:"code"`                 // CPT/HCPCS code
	Category       string    `json:"category" db:"category"`
	Description    string    `json:"description" db:"description"`
	NormalizedTags []string  `json:"normalized_tags" db:"normalized_tags"` // Searchable tags from normalization
	IsActive       bool      `json:"is_active" db:"is_active"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`

	// GraphQL support fields (populated by resolvers when in context of a facility)
	FacilityID string  `json:"facility_id,omitempty" db:"-"`
	Price      float64 `json:"price,omitempty" db:"-"`
	Duration   int     `json:"duration,omitempty" db:"-"`
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

	// Enriched fields populated by JOIN queries (not stored in facility_procedures table)
	ProcedureName        string   `json:"name,omitempty" db:"-"`
	ProcedureDisplayName string   `json:"display_name,omitempty" db:"-"`
	ProcedureCode        string   `json:"code,omitempty" db:"-"`
	ProcedureCategory    string   `json:"category,omitempty" db:"-"`
	ProcedureDescription string   `json:"description,omitempty" db:"-"`
	ProcedureTags        []string `json:"normalized_tags,omitempty" db:"-"`
}
