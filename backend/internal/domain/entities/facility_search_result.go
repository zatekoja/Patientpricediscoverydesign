package entities

import "time"

// FacilitySearchResult represents the enriched search payload returned to the UI.
type FacilitySearchResult struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	FacilityType        string              `json:"facility_type"`
	Address             Address             `json:"address"`
	Location            Location            `json:"location"`
	PhoneNumber         string              `json:"phone_number,omitempty"`
	WhatsAppNumber      string              `json:"whatsapp_number,omitempty"`
	Email               string              `json:"email,omitempty"`
	Website             string              `json:"website,omitempty"`
	Rating              float64             `json:"rating"`
	ReviewCount         int                 `json:"review_count"`
	DistanceKm          float64             `json:"distance_km"`
	Price               *FacilityPriceRange `json:"price,omitempty"`
	Services            []string            `json:"services"`
	ServicePrices       []ServicePrice      `json:"service_prices"`
	Tags                []string            `json:"tags,omitempty"`
	AcceptedInsurance   []string            `json:"accepted_insurance"`
	NextAvailableAt     *time.Time          `json:"next_available_at,omitempty"`
	AvgWaitMinutes      *int                `json:"avg_wait_minutes,omitempty"`
	CapacityStatus      string              `json:"capacity_status,omitempty"`
	WardStatuses        interface{}         `json:"ward_statuses,omitempty"`
	UrgentCareAvailable *bool               `json:"urgent_care_available,omitempty"`
	UpdatedAt           time.Time           `json:"updated_at"`
}

// FacilityPriceRange summarizes price ranges for a facility.
type FacilityPriceRange struct {
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Currency string  `json:"currency"`
}

// ServicePrice represents a priced service at a facility.
type ServicePrice struct {
	ProcedureID       string  `json:"procedure_id"`
	Name              string  `json:"name"`
	Price             float64 `json:"price"`
	Currency          string  `json:"currency"`
	Description       string  `json:"description,omitempty"`
	Category          string  `json:"category,omitempty"`
	Code              string  `json:"code,omitempty"`
	EstimatedDuration int     `json:"estimated_duration,omitempty"`
}
