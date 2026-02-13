package entities

import (
	"time"
)

// SearchEvent represents a single search interaction for analytics.
type SearchEvent struct {
	ID               string    `json:"id" db:"id"`
	Query            string    `json:"query" db:"query"`
	NormalizedQuery  string    `json:"normalized_query" db:"normalized_query"`
	DetectedIntent   string    `json:"detected_intent" db:"detected_intent"`
	IntentConfidence float64   `json:"intent_confidence" db:"intent_confidence"`
	ResultCount      int       `json:"result_count" db:"result_count"`
	LatencyMs        int       `json:"latency_ms" db:"latency_ms"`
	UserLatitude     float64   `json:"user_latitude" db:"user_latitude"`
	UserLongitude    float64   `json:"user_longitude" db:"user_longitude"`
	SessionID        string    `json:"session_id,omitempty" db:"session_id"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}
