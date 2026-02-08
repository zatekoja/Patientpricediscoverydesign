package entities

import (
	"time"
)

// FacilityWard represents a ward/department within a healthcare facility
type FacilityWard struct {
	ID                   string     `json:"id" db:"id"`
	FacilityID           string     `json:"facility_id" db:"facility_id"`
	WardName             string     `json:"ward_name" db:"ward_name"`
	WardType             *string    `json:"ward_type,omitempty" db:"ward_type"`
	CapacityStatus       *string    `json:"capacity_status,omitempty" db:"capacity_status"`
	AvgWaitMinutes       *int       `json:"avg_wait_minutes,omitempty" db:"avg_wait_minutes"`
	UrgentCareAvailable  *bool      `json:"urgent_care_available,omitempty" db:"urgent_care_available"`
	LastUpdated          time.Time  `json:"last_updated" db:"last_updated"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
}
