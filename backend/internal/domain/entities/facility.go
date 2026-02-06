package entities

import (
	"time"
)

// Facility represents a healthcare facility in the system
type Facility struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Address     Address   `json:"address" db:"-"`
	Location    Location  `json:"location" db:"-"`
	PhoneNumber string    `json:"phone_number" db:"phone_number"`
	Email       string    `json:"email" db:"email"`
	Website     string    `json:"website" db:"website"`
	Description string    `json:"description" db:"description"`
	FacilityType string   `json:"facility_type" db:"facility_type"`
	AcceptedInsurance []string `json:"accepted_insurance" db:"-"`
	Rating      float64   `json:"rating" db:"rating"`
	ReviewCount int       `json:"review_count" db:"review_count"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Address represents a physical address
type Address struct {
	Street     string `json:"street" db:"street"`
	City       string `json:"city" db:"city"`
	State      string `json:"state" db:"state"`
	ZipCode    string `json:"zip_code" db:"zip_code"`
	Country    string `json:"country" db:"country"`
}

// Location represents geographical coordinates
type Location struct {
	Latitude  float64 `json:"latitude" db:"latitude"`
	Longitude float64 `json:"longitude" db:"longitude"`
}
