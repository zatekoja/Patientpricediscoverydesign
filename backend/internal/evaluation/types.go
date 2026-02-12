package evaluation

import "time"

// Intent represents the detected search intent category.
type Intent string

const (
	IntentCondition Intent = "condition" // e.g., "malaria", "diabetes"
	IntentProcedure Intent = "procedure" // e.g., "ct scan", "blood test"
	IntentFacility  Intent = "facility"  // e.g., "hospital near me", "pharmacy"
	IntentSymptom   Intent = "symptom"   // e.g., "headache", "chest pain"
)

// ValidIntents returns all valid intent values.
func ValidIntents() []Intent {
	return []Intent{IntentCondition, IntentProcedure, IntentFacility, IntentSymptom}
}

// IsValid checks if the intent value is one of the defined constants.
func (i Intent) IsValid() bool {
	switch i {
	case IntentCondition, IntentProcedure, IntentFacility, IntentSymptom:
		return true
	}
	return false
}

// GoldenQuery represents a labeled test query with expected outcomes.
type GoldenQuery struct {
	ID               string   `json:"id"`
	Query            string   `json:"query"`
	Intent           Intent   `json:"intent"`
	ExpectedTags     []string `json:"expected_tags"`
	ExpectedFacTypes []string `json:"expected_facility_types"`
	Difficulty       string   `json:"difficulty"` // easy, medium, hard
}

// EvalResult holds the evaluation outcome for a single query.
type EvalResult struct {
	QueryID       string
	Query         string
	Intent        Intent
	RecallAt10    float64
	MRRAt10       float64
	ResultCount   int
	RetrievedTags []string
	Latency       time.Duration
}

// EvalSummary holds aggregate metrics across all golden queries.
type EvalSummary struct {
	TotalQueries    int
	AvgRecallAt10   float64
	AvgMRRAt10      float64
	AvgLatency      time.Duration
	QueriesWithHits int // queries that returned at least 1 result
	ByIntent        map[Intent]*IntentSummary
}

// IntentSummary holds metrics grouped by intent type.
type IntentSummary struct {
	Count         int
	AvgRecallAt10 float64
	AvgMRRAt10    float64
}
