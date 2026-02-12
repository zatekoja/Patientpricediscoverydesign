package openai

import (
	"encoding/json"
	"testing"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

func TestParseEnrichmentWithConcepts_ValidResponse(t *testing.T) {
	raw := `{
		"description": "A blood test for malaria parasites.",
		"prep_steps": ["Fast for 8 hours", "Bring previous results"],
		"risks": ["Minor bruising", "Slight discomfort"],
		"recovery": ["Apply pressure to puncture site", "Resume normal activities"],
		"search_concepts": {
			"conditions": ["malaria", "plasmodium infection"],
			"symptoms": ["fever", "chills", "headache", "body ache"],
			"lay_terms": ["malaria test", "blood smear"],
			"synonyms": ["mp test", "malaria parasite test"],
			"specialties": ["internal_medicine", "infectious_disease"],
			"facility_types": ["hospital", "diagnostic_lab"],
			"intent_tags": ["diagnostic"]
		}
	}`

	payload, err := parseEnrichmentPayloadWithConcepts([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload.Description != "A blood test for malaria parasites." {
		t.Errorf("wrong description: %s", payload.Description)
	}
	if len(payload.PrepSteps) != 2 {
		t.Errorf("expected 2 prep steps, got %d", len(payload.PrepSteps))
	}
	if payload.SearchConcepts == nil {
		t.Fatal("expected non-nil search concepts")
	}
	if len(payload.SearchConcepts.Conditions) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(payload.SearchConcepts.Conditions))
	}
	if len(payload.SearchConcepts.Symptoms) != 4 {
		t.Errorf("expected 4 symptoms, got %d", len(payload.SearchConcepts.Symptoms))
	}
	if payload.SearchConcepts.IntentTags[0] != "diagnostic" {
		t.Errorf("expected intent_tag 'diagnostic', got %q", payload.SearchConcepts.IntentTags[0])
	}
}

func TestParseEnrichmentWithConcepts_MissingConcepts_FallsBackGracefully(t *testing.T) {
	raw := `{
		"description": "A blood test.",
		"prep_steps": ["Fast for 8 hours"],
		"risks": ["Minor bruising"],
		"recovery": ["Rest"]
	}`

	payload, err := parseEnrichmentPayloadWithConcepts([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload.Description != "A blood test." {
		t.Errorf("wrong description: %s", payload.Description)
	}
	// SearchConcepts should be nil when not in the response
	if payload.SearchConcepts != nil {
		t.Error("expected nil search concepts when not in response")
	}
}

func TestParseEnrichmentWithConcepts_ExcessiveTerms_Truncated(t *testing.T) {
	terms := make([]string, 20)
	for i := range terms {
		terms[i] = "term"
	}
	sc := &entities.SearchConcepts{Conditions: terms}
	_ = sc.Validate()
	if len(sc.Conditions) > entities.MaxTermsPerField {
		t.Errorf("expected at most %d conditions after validate, got %d", entities.MaxTermsPerField, len(sc.Conditions))
	}
}

func TestParseEnrichmentWithConcepts_DuplicateTerms_Deduped(t *testing.T) {
	raw := `{
		"description": "Test",
		"prep_steps": [],
		"risks": [],
		"recovery": [],
		"search_concepts": {
			"conditions": ["malaria", "MALARIA", "Malaria"],
			"symptoms": ["fever", "FEVER"],
			"lay_terms": [],
			"synonyms": [],
			"specialties": [],
			"facility_types": [],
			"intent_tags": []
		}
	}`

	payload, err := parseEnrichmentPayloadWithConcepts([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload.SearchConcepts == nil {
		t.Fatal("expected non-nil concepts")
	}
	// After validation, duplicates should be removed
	_ = payload.SearchConcepts.Validate()
	if len(payload.SearchConcepts.Conditions) != 1 {
		t.Errorf("expected 1 condition after dedup, got %d: %v", len(payload.SearchConcepts.Conditions), payload.SearchConcepts.Conditions)
	}
	if len(payload.SearchConcepts.Symptoms) != 1 {
		t.Errorf("expected 1 symptom after dedup, got %d", len(payload.SearchConcepts.Symptoms))
	}
}

func TestParseEnrichmentWithConcepts_InvalidJSON(t *testing.T) {
	_, err := parseEnrichmentPayloadWithConcepts([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestValidateConceptOutput_MaxLength(t *testing.T) {
	longTerm := make([]byte, 200)
	for i := range longTerm {
		longTerm[i] = 'x'
	}
	sc := &entities.SearchConcepts{
		Conditions: []string{string(longTerm)},
	}
	_ = sc.Validate()
	if len(sc.Conditions[0]) > entities.MaxTermLength {
		t.Errorf("term not truncated: len=%d", len(sc.Conditions[0]))
	}
}

func TestValidateConceptOutput_Lowercase(t *testing.T) {
	sc := &entities.SearchConcepts{
		Conditions: []string{"MALARIA", "Typhoid"},
	}
	_ = sc.Validate()
	if sc.Conditions[0] != "malaria" {
		t.Errorf("expected lowercase, got %q", sc.Conditions[0])
	}
	if sc.Conditions[1] != "typhoid" {
		t.Errorf("expected lowercase, got %q", sc.Conditions[1])
	}
}

func TestBuildSearchConceptPrompt_IncludesProcedureContext(t *testing.T) {
	prompt := buildSearchConceptUserPrompt("Malaria Parasite Test", "laboratory", "LAB001", "Blood test for malaria")
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	// Should include the procedure details
	for _, expected := range []string{"Malaria Parasite Test", "laboratory", "LAB001", "Blood test for malaria"} {
		if !contains(prompt, expected) {
			t.Errorf("prompt should contain %q, got: %s", expected, prompt)
		}
	}
}

func TestEnrichmentPayloadWithConcepts_JSONRoundTrip(t *testing.T) {
	original := enrichmentPayloadWithConcepts{
		enrichmentPayload: enrichmentPayload{
			Description: "Test procedure",
			PrepSteps:   []string{"Step 1"},
			Risks:       []string{"Risk 1"},
			Recovery:    []string{"Recovery 1"},
		},
		SearchConcepts: &entities.SearchConcepts{
			Conditions:    []string{"condition1"},
			Symptoms:      []string{"symptom1"},
			Specialties:   []string{"specialty1"},
			FacilityTypes: []string{"hospital"},
			IntentTags:    []string{"diagnostic"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	parsed, err := parseEnrichmentPayloadWithConcepts(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if parsed.Description != original.Description {
		t.Errorf("description mismatch")
	}
	if parsed.SearchConcepts == nil {
		t.Fatal("concepts should not be nil")
	}
	if len(parsed.SearchConcepts.Conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(parsed.SearchConcepts.Conditions))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
