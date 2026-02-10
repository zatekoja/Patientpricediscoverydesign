package entities

import "time"

// FeeWaiver represents a 3rd-party sponsored service fee waiver
type FeeWaiver struct {
	ID             string     `json:"id" db:"id"`
	SponsorName    string     `json:"sponsor_name" db:"sponsor_name"`
	SponsorContact string     `json:"sponsor_contact,omitempty" db:"sponsor_contact"`
	FacilityID     *string    `json:"facility_id,omitempty" db:"facility_id"` // nil = all facilities
	WaiverType     string     `json:"waiver_type" db:"waiver_type"`           // "full" or "partial"
	WaiverAmount   *float64   `json:"waiver_amount,omitempty" db:"waiver_amount"`
	MaxUses        *int       `json:"max_uses,omitempty" db:"max_uses"`
	CurrentUses    int        `json:"current_uses" db:"current_uses"`
	ValidFrom      time.Time  `json:"valid_from" db:"valid_from"`
	ValidUntil     *time.Time `json:"valid_until,omitempty" db:"valid_until"`
	IsActive       bool       `json:"is_active" db:"is_active"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// IsValid checks if a waiver is currently valid and has remaining uses
func (w *FeeWaiver) IsValid() bool {
	if !w.IsActive {
		return false
	}
	now := time.Now()
	if now.Before(w.ValidFrom) {
		return false
	}
	if w.ValidUntil != nil && now.After(*w.ValidUntil) {
		return false
	}
	if w.MaxUses != nil && w.CurrentUses >= *w.MaxUses {
		return false
	}
	return true
}

// ApplyToServiceFee calculates the waived service fee amount
func (w *FeeWaiver) ApplyToServiceFee(serviceFee float64) float64 {
	if !w.IsValid() {
		return serviceFee
	}
	if w.WaiverType == "full" {
		return 0
	}
	if w.WaiverAmount != nil {
		reduced := serviceFee - *w.WaiverAmount
		if reduced < 0 {
			return 0
		}
		return reduced
	}
	return serviceFee
}
