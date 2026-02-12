package entities

import "time"

// ProcedureEnrichment stores AI-generated information for a procedure.
type ProcedureEnrichment struct {
	ID             string          `json:"id" db:"id"`
	ProcedureID    string          `json:"procedure_id" db:"procedure_id"`
	Description    string          `json:"description" db:"description"`
	PrepSteps      []string        `json:"prep_steps" db:"prep_steps"`
	Risks          []string        `json:"risks" db:"risks"`
	Recovery       []string        `json:"recovery" db:"recovery"`
	SearchConcepts *SearchConcepts `json:"search_concepts,omitempty" db:"-"`
	Provider       string          `json:"provider" db:"provider"`
	Model          string          `json:"model" db:"model"`

	// Enrichment status tracking for backfill
	EnrichmentStatus  string `json:"enrichment_status,omitempty" db:"enrichment_status"`
	EnrichmentVersion int    `json:"enrichment_version,omitempty" db:"enrichment_version"`
	RetryCount        int    `json:"retry_count,omitempty" db:"retry_count"`
	LastError         string `json:"last_error,omitempty" db:"last_error"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
