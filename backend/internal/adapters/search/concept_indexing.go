package search

import (
	"strings"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

const MaxIndexedTerms = 100

type ConceptFields struct {
	Concepts    []string
	Conditions  []string
	Symptoms    []string
	Specialties []string
}

func BuildConceptFields(enrichments []*entities.ProcedureEnrichment) ConceptFields {
	if len(enrichments) == 0 {
		return ConceptFields{}
	}

	condSet := make(map[string]struct{})
	symSet := make(map[string]struct{})
	specSet := make(map[string]struct{})
	conceptSet := make(map[string]struct{})

	for _, e := range enrichments {
		if e == nil || e.SearchConcepts == nil {
			continue
		}
		sc := e.SearchConcepts

		add(condSet, sc.Conditions...)
		add(symSet, sc.Symptoms...)
		add(specSet, sc.Specialties...)

		// Add all terms to general concepts bucket
		add(conceptSet, sc.Conditions...)
		add(conceptSet, sc.Symptoms...)
		add(conceptSet, sc.LayTerms...)
		add(conceptSet, sc.Synonyms...)
		add(conceptSet, sc.Specialties...)
		add(conceptSet, sc.FacilityTypes...)
		add(conceptSet, sc.IntentTags...)
	}

	return ConceptFields{
		Conditions:  toSlice(condSet, MaxIndexedTerms),
		Symptoms:    toSlice(symSet, MaxIndexedTerms),
		Specialties: toSlice(specSet, MaxIndexedTerms),
		Concepts:    toSlice(conceptSet, MaxIndexedTerms*2), // Allow more for general bag
	}
}

func add(set map[string]struct{}, terms ...string) {
	for _, t := range terms {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" {
			set[t] = struct{}{}
		}
	}
}

func toSlice(set map[string]struct{}, limit int) []string {
	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
		if len(result) >= limit {
			break
		}
	}
	return result
}
