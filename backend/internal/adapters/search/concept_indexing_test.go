package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

func TestBuildConceptFields_FromEnrichment(t *testing.T) {
	enrichment := &entities.ProcedureEnrichment{
		SearchConcepts: &entities.SearchConcepts{
			Conditions:    []string{"malaria", "fever"},
			Symptoms:      []string{"headache"},
			Specialties:   []string{"general_practice"},
			FacilityTypes: []string{"hospital"},
		},
	}

	fields := BuildConceptFields([]*entities.ProcedureEnrichment{enrichment})

	assert.Contains(t, fields.Conditions, "malaria")
	assert.Contains(t, fields.Conditions, "fever")
	assert.Contains(t, fields.Symptoms, "headache")
	assert.Contains(t, fields.Specialties, "general_practice")
	assert.Contains(t, fields.Concepts, "malaria") // Concepts should aggregate key terms
	assert.Contains(t, fields.Concepts, "headache")
}

func TestBuildConceptFields_MergesAcrossProcedures(t *testing.T) {
	e1 := &entities.ProcedureEnrichment{
		SearchConcepts: &entities.SearchConcepts{
			Conditions: []string{"malaria"},
		},
	}
	e2 := &entities.ProcedureEnrichment{
		SearchConcepts: &entities.SearchConcepts{
			Conditions: []string{"typhoid"},
		},
	}

	fields := BuildConceptFields([]*entities.ProcedureEnrichment{e1, e2})

	assert.ElementsMatch(t, fields.Conditions, []string{"malaria", "typhoid"})
}

func TestBuildConceptFields_Deduplication(t *testing.T) {
	e1 := &entities.ProcedureEnrichment{
		SearchConcepts: &entities.SearchConcepts{
			Conditions: []string{"malaria"},
		},
	}
	e2 := &entities.ProcedureEnrichment{
		SearchConcepts: &entities.SearchConcepts{
			Conditions: []string{"malaria"},
		},
	}

	fields := BuildConceptFields([]*entities.ProcedureEnrichment{e1, e2})

	assert.Equal(t, 1, len(fields.Conditions))
	assert.Equal(t, "malaria", fields.Conditions[0])
}

func TestBuildConceptFields_EmptyEnrichment(t *testing.T) {
	fields := BuildConceptFields([]*entities.ProcedureEnrichment{})
	assert.Empty(t, fields.Conditions)
	assert.Empty(t, fields.Symptoms)
}

func TestBuildConceptFields_NilSearchConcepts(t *testing.T) {
	e1 := &entities.ProcedureEnrichment{
		SearchConcepts: nil,
	}
	fields := BuildConceptFields([]*entities.ProcedureEnrichment{e1})
	assert.Empty(t, fields.Conditions)
}
