package entities

import (
	"testing"
)

func TestSearchConcepts_Validate_Valid(t *testing.T) {
	sc := &SearchConcepts{
		Conditions:    []string{"malaria", "fever"},
		Symptoms:      []string{"headache", "chills"},
		LayTerms:      []string{"malaria test"},
		Synonyms:      []string{"mp test"},
		Specialties:   []string{"internal_medicine"},
		FacilityTypes: []string{"hospital", "diagnostic_lab"},
		IntentTags:    []string{"diagnostic"},
	}
	if err := sc.Validate(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestSearchConcepts_Validate_DuplicateRemoval(t *testing.T) {
	sc := &SearchConcepts{
		Conditions: []string{"malaria", "MALARIA", "Malaria", "malaria"},
		Symptoms:   []string{"headache", "Headache"},
	}
	if err := sc.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// After validation, duplicates should be removed (case-insensitive)
	if len(sc.Conditions) != 1 {
		t.Errorf("expected 1 condition after dedup, got %d: %v", len(sc.Conditions), sc.Conditions)
	}
	if sc.Conditions[0] != "malaria" {
		t.Errorf("expected 'malaria', got %q", sc.Conditions[0])
	}
	if len(sc.Symptoms) != 1 {
		t.Errorf("expected 1 symptom after dedup, got %d", len(sc.Symptoms))
	}
}

func TestSearchConcepts_Validate_LengthLimits(t *testing.T) {
	// More than MaxTermsPerField items
	conditions := make([]string, 15)
	for i := range conditions {
		conditions[i] = "condition"
	}
	sc := &SearchConcepts{
		Conditions: conditions,
	}
	if err := sc.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(sc.Conditions) > MaxTermsPerField {
		t.Errorf("expected at most %d conditions, got %d", MaxTermsPerField, len(sc.Conditions))
	}
}

func TestSearchConcepts_Validate_TermMaxLength(t *testing.T) {
	longTerm := make([]byte, MaxTermLength+20)
	for i := range longTerm {
		longTerm[i] = 'a'
	}
	sc := &SearchConcepts{
		Conditions: []string{string(longTerm)},
	}
	if err := sc.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Long term should be truncated
	if len(sc.Conditions[0]) > MaxTermLength {
		t.Errorf("expected term truncated to %d, got %d", MaxTermLength, len(sc.Conditions[0]))
	}
}

func TestSearchConcepts_Validate_LowercaseAndTrim(t *testing.T) {
	sc := &SearchConcepts{
		Conditions: []string{"  MALARIA  ", " Typhoid Fever "},
	}
	if err := sc.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if sc.Conditions[0] != "malaria" {
		t.Errorf("expected 'malaria', got %q", sc.Conditions[0])
	}
	if sc.Conditions[1] != "typhoid fever" {
		t.Errorf("expected 'typhoid fever', got %q", sc.Conditions[1])
	}
}

func TestSearchConcepts_Validate_EmptyTermsRemoved(t *testing.T) {
	sc := &SearchConcepts{
		Conditions: []string{"malaria", "", "  ", "typhoid"},
	}
	if err := sc.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(sc.Conditions) != 2 {
		t.Errorf("expected 2 conditions (empty removed), got %d: %v", len(sc.Conditions), sc.Conditions)
	}
}

func TestSearchConcepts_AllTerms_Flattened(t *testing.T) {
	sc := &SearchConcepts{
		Conditions:    []string{"malaria"},
		Symptoms:      []string{"fever", "chills"},
		LayTerms:      []string{"malaria test"},
		Synonyms:      []string{"mp test"},
		Specialties:   []string{"internal_medicine"},
		FacilityTypes: []string{"hospital"},
		IntentTags:    []string{"diagnostic"},
	}
	terms := sc.AllTerms()
	if len(terms) != 8 {
		t.Errorf("expected 8 unique terms, got %d: %v", len(terms), terms)
	}
}

func TestSearchConcepts_AllTerms_Deduplicated(t *testing.T) {
	sc := &SearchConcepts{
		Conditions: []string{"malaria"},
		Symptoms:   []string{"malaria"}, // duplicate across fields
		LayTerms:   []string{"test"},
	}
	terms := sc.AllTerms()
	if len(terms) != 2 {
		t.Errorf("expected 2 unique terms (dedup across fields), got %d: %v", len(terms), terms)
	}
}

func TestSearchConcepts_AllTerms_Nil(t *testing.T) {
	var sc *SearchConcepts
	terms := sc.AllTerms()
	if len(terms) != 0 {
		t.Errorf("expected 0 terms for nil, got %d", len(terms))
	}
}

func TestSearchConcepts_Merge_TwoConcepts(t *testing.T) {
	a := &SearchConcepts{
		Conditions: []string{"malaria"},
		Symptoms:   []string{"fever"},
	}
	b := &SearchConcepts{
		Conditions: []string{"typhoid"},
		Symptoms:   []string{"chills"},
		LayTerms:   []string{"worm test"},
	}
	merged := MergeSearchConcepts(a, b)
	if len(merged.Conditions) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(merged.Conditions))
	}
	if len(merged.Symptoms) != 2 {
		t.Errorf("expected 2 symptoms, got %d", len(merged.Symptoms))
	}
	if len(merged.LayTerms) != 1 {
		t.Errorf("expected 1 lay term, got %d", len(merged.LayTerms))
	}
}

func TestSearchConcepts_Merge_NilHandling(t *testing.T) {
	a := &SearchConcepts{Conditions: []string{"malaria"}}
	merged := MergeSearchConcepts(a, nil)
	if len(merged.Conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(merged.Conditions))
	}

	merged2 := MergeSearchConcepts(nil, a)
	if len(merged2.Conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(merged2.Conditions))
	}

	merged3 := MergeSearchConcepts(nil, nil)
	if merged3 == nil {
		t.Error("expected non-nil empty result")
	}
}

func TestSearchConcepts_Merge_Deduplication(t *testing.T) {
	a := &SearchConcepts{
		Conditions: []string{"malaria", "typhoid"},
	}
	b := &SearchConcepts{
		Conditions: []string{"malaria", "cholera"},
	}
	merged := MergeSearchConcepts(a, b)
	if len(merged.Conditions) != 3 {
		t.Errorf("expected 3 conditions (deduped), got %d: %v", len(merged.Conditions), merged.Conditions)
	}
}

func TestProcedureEnrichment_HasSearchConcepts(t *testing.T) {
	e := &ProcedureEnrichment{
		ID:          "test",
		ProcedureID: "proc1",
		SearchConcepts: &SearchConcepts{
			Conditions: []string{"malaria"},
		},
	}
	if e.SearchConcepts == nil {
		t.Error("expected non-nil search concepts")
	}
	if len(e.SearchConcepts.Conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(e.SearchConcepts.Conditions))
	}
}

func TestProcedureEnrichment_NilSearchConcepts(t *testing.T) {
	e := &ProcedureEnrichment{
		ID:          "test",
		ProcedureID: "proc1",
	}
	if e.SearchConcepts != nil {
		t.Error("expected nil search concepts")
	}
}
