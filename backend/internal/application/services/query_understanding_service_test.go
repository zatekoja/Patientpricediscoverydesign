package services

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/evaluation"
)

func testConfigDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "config")
}

func newTestQueryService(t *testing.T) *QueryUnderstandingService {
	t.Helper()
	configDir := testConfigDir()
	svc, err := NewQueryUnderstandingService(
		filepath.Join(configDir, "concept_dictionary.json"),
		filepath.Join(configDir, "spelling_corrections.json"),
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	return svc
}

// --- Normalization tests ---

func TestNormalize_Lowercase(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("MALARIA")
	if result.NormalizedQuery != "malaria" {
		t.Errorf("expected 'malaria', got %q", result.NormalizedQuery)
	}
}

func TestNormalize_ExtraWhitespace(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("  ct   scan  ")
	if result.NormalizedQuery != "ct scan" {
		t.Errorf("expected 'ct scan', got %q", result.NormalizedQuery)
	}
}

func TestNormalize_SpecialCharacters(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("x-ray!")
	if result.NormalizedQuery != "x-ray" {
		t.Errorf("expected 'x-ray', got %q", result.NormalizedQuery)
	}
}

// --- Spell correction tests ---

func TestSpellCorrect_CommonMisspelling(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("malarya")
	if result.CorrectedQuery != "malaria" {
		t.Errorf("expected 'malaria', got %q", result.CorrectedQuery)
	}
}

func TestSpellCorrect_NoCorrection(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("malaria")
	// When no correction needed, CorrectedQuery should be empty
	if result.CorrectedQuery != "" {
		t.Errorf("expected empty corrected query, got %q", result.CorrectedQuery)
	}
}

func TestSpellCorrect_MultiWord(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("blood tets")
	if result.CorrectedQuery != "blood test" {
		t.Errorf("expected 'blood test', got %q", result.CorrectedQuery)
	}
}

// --- Intent detection tests ---

func TestDetectIntent_Condition(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("malaria")
	if result.DetectedIntent != evaluation.IntentCondition {
		t.Errorf("expected condition intent, got %s", result.DetectedIntent)
	}
}

func TestDetectIntent_Procedure(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("ct scan")
	if result.DetectedIntent != evaluation.IntentProcedure {
		t.Errorf("expected procedure intent, got %s", result.DetectedIntent)
	}
}

func TestDetectIntent_Facility(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("hospital near me")
	if result.DetectedIntent != evaluation.IntentFacility {
		t.Errorf("expected facility intent, got %s", result.DetectedIntent)
	}
}

func TestDetectIntent_Symptom(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("headache")
	if result.DetectedIntent != evaluation.IntentSymptom {
		t.Errorf("expected symptom intent, got %s", result.DetectedIntent)
	}
}

// --- Term mapping tests ---

func TestMapTerms_SingleTerm(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("malaria")
	if len(result.SearchTerms) == 0 {
		t.Fatal("expected non-empty search terms")
	}
	// Should include related terms like "malaria test", "blood smear", etc.
	if !containsStr(result.SearchTerms, "malaria test") {
		t.Errorf("expected 'malaria test' in search terms, got %v", result.SearchTerms)
	}
}

func TestMapTerms_MultiWordPhrase(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("chest pain")
	if result.MappedConcepts == nil {
		t.Fatal("expected non-nil mapped concepts")
	}
	// Should map to cardiology
	if !containsStr(result.MappedConcepts.Specialties, "cardiology") && !containsStr(result.MappedConcepts.Specialties, "emergency_medicine") {
		t.Errorf("expected cardiology or emergency_medicine in specialties, got %v", result.MappedConcepts.Specialties)
	}
}

func TestMapTerms_LayTerm(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("sugar test")
	if result.MappedConcepts == nil {
		t.Fatal("expected non-nil mapped concepts")
	}
	// Should map to lab/diagnostic
	if !containsStr(result.SearchTerms, "glucose test") && !containsStr(result.SearchTerms, "blood sugar") {
		t.Errorf("expected glucose/blood sugar terms in search terms, got %v", result.SearchTerms)
	}
}

func TestMapTerms_NigerianSlang(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("belle")
	if result.MappedConcepts == nil {
		t.Fatal("expected non-nil mapped concepts")
	}
	// "belle" is Nigerian slang for pregnancy
	found := containsStr(result.SearchTerms, "pregnancy") ||
		containsStr(result.SearchTerms, "antenatal") ||
		containsStr(result.SearchTerms, "maternity")
	if !found {
		t.Errorf("expected pregnancy-related terms, got %v", result.SearchTerms)
	}
}

func TestMapTerms_UnknownTerm_PassThrough(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("xyzabc123")
	// Unknown terms should pass through as-is in search terms
	if !containsStr(result.SearchTerms, "xyzabc123") {
		t.Errorf("expected unknown term to pass through, got %v", result.SearchTerms)
	}
}

// --- Full pipeline tests ---

func TestInterpret_EndToEnd_ToothAche(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("tooth ache")
	if result.DetectedIntent != evaluation.IntentSymptom {
		t.Errorf("expected symptom intent, got %s", result.DetectedIntent)
	}
	if result.MappedConcepts == nil {
		t.Fatal("expected non-nil mapped concepts")
	}
	hasDental := containsStr(result.MappedConcepts.Specialties, "dental")
	if !hasDental {
		t.Errorf("expected dental specialty, got %v", result.MappedConcepts.Specialties)
	}
}

func TestInterpret_EndToEnd_Baby(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("baby")
	if result.MappedConcepts == nil {
		t.Fatal("expected non-nil mapped concepts")
	}
	hasObstetrics := containsStr(result.MappedConcepts.Specialties, "obstetrics") ||
		containsStr(result.MappedConcepts.Specialties, "paediatrics")
	if !hasObstetrics {
		t.Errorf("expected obstetrics or paediatrics, got %v", result.MappedConcepts.Specialties)
	}
}

func TestInterpret_EndToEnd_KneePain(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("knee pain")
	if result.MappedConcepts == nil {
		t.Fatal("expected non-nil mapped concepts")
	}
	if !containsStr(result.MappedConcepts.Specialties, "orthopaedics") {
		t.Errorf("expected orthopaedics specialty, got %v", result.MappedConcepts.Specialties)
	}
}

func TestInterpret_EndToEnd_MRI(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("MRI")
	if result.DetectedIntent != evaluation.IntentProcedure {
		t.Errorf("expected procedure intent, got %s", result.DetectedIntent)
	}
	if !containsStr(result.SearchTerms, "magnetic resonance") {
		t.Errorf("expected 'magnetic resonance' in search terms, got %v", result.SearchTerms)
	}
}

func TestInterpret_EmptyQuery(t *testing.T) {
	svc := newTestQueryService(t)
	result := svc.Interpret("")
	if result.NormalizedQuery != "" {
		t.Errorf("expected empty normalized query, got %q", result.NormalizedQuery)
	}
	if len(result.SearchTerms) != 0 {
		t.Errorf("expected empty search terms, got %v", result.SearchTerms)
	}
}

func containsStr(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
