package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

func TestRank_SortsCorrectly(t *testing.T) {
	svc := NewSearchRankingService()

	// Interp: user wants "malaria" (Intent: condition)
	interp := &QueryInterpretation{
		OriginalQuery:  "malaria",
		DetectedIntent: "condition",
		SearchTerms:    []string{"malaria"},
		MappedConcepts: &entities.SearchConcepts{
			Conditions: []string{"malaria"},
		},
	}

	f1 := &entities.Facility{ID: "f1", Name: "Malaria Clinic", Tags: []string{"malaria"}}
	f2 := &entities.Facility{ID: "f2", Name: "General Hospital"} // No match

	results := svc.Rank([]*entities.Facility{f2, f1}, interp, 0, 0)

	assert.Equal(t, 2, len(results))
	assert.Equal(t, "f1", results[0].Facility.ID) // f1 should be first
	assert.Greater(t, results[0].Score, results[1].Score)
}

func TestScore_GeoProximityBoost(t *testing.T) {
	svc := NewSearchRankingService()
	interp := &QueryInterpretation{OriginalQuery: "clinic"}

	// Both match "clinic", but f1 is closer
	f1 := &entities.Facility{ID: "f1", Location: entities.Location{Latitude: 10.0, Longitude: 10.0}}
	f2 := &entities.Facility{ID: "f2", Location: entities.Location{Latitude: 10.1, Longitude: 10.1}}

	// User at 10.0, 10.0
	results := svc.Rank([]*entities.Facility{f1, f2}, interp, 10.0, 10.0)

	assert.Equal(t, "f1", results[0].Facility.ID)
}

func TestRank_EmptyResults(t *testing.T) {
	svc := NewSearchRankingService()
	interp := &QueryInterpretation{OriginalQuery: "test"}
	results := svc.Rank([]*entities.Facility{}, interp, 0, 0)
	assert.Empty(t, results)
}
