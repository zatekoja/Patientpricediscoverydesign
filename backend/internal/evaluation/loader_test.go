package evaluation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGoldenQueries_ValidFile(t *testing.T) {
	content := `[
		{"id": "q1", "query": "malaria", "intent": "condition", "expected_tags": ["laboratory", "diagnostic"], "expected_facility_types": ["hospital", "diagnostic_lab"], "difficulty": "easy"},
		{"id": "q2", "query": "ct scan", "intent": "procedure", "expected_tags": ["imaging"], "expected_facility_types": ["hospital", "imaging_center"], "difficulty": "easy"}
	]`
	path := writeTempFile(t, content)

	queries, err := LoadGoldenQueries(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(queries))
	}
	if queries[0].ID != "q1" {
		t.Errorf("expected id q1, got %s", queries[0].ID)
	}
	if queries[0].Intent != IntentCondition {
		t.Errorf("expected intent condition, got %s", queries[0].Intent)
	}
	if len(queries[0].ExpectedTags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(queries[0].ExpectedTags))
	}
	if queries[1].Query != "ct scan" {
		t.Errorf("expected query 'ct scan', got %s", queries[1].Query)
	}
}

func TestLoadGoldenQueries_InvalidFile(t *testing.T) {
	_, err := LoadGoldenQueries("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadGoldenQueries_InvalidJSON(t *testing.T) {
	path := writeTempFile(t, `not valid json`)
	_, err := LoadGoldenQueries(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadGoldenQueries_EmptyArray(t *testing.T) {
	path := writeTempFile(t, `[]`)
	queries, err := LoadGoldenQueries(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 0 {
		t.Errorf("expected 0 queries, got %d", len(queries))
	}
}

func TestGoldenQuery_IntentValidation(t *testing.T) {
	tests := []struct {
		intent Intent
		valid  bool
	}{
		{IntentCondition, true},
		{IntentProcedure, true},
		{IntentFacility, true},
		{IntentSymptom, true},
		{Intent("unknown"), false},
		{Intent(""), false},
	}
	for _, tt := range tests {
		got := tt.intent.IsValid()
		if got != tt.valid {
			t.Errorf("Intent(%q).IsValid() = %v, want %v", tt.intent, got, tt.valid)
		}
	}
}

func TestValidateGoldenQueries_MissingID(t *testing.T) {
	queries := []GoldenQuery{
		{ID: "", Query: "test", Intent: IntentCondition, Difficulty: "easy"},
	}
	err := ValidateGoldenQueries(queries)
	if err == nil {
		t.Error("expected validation error for missing ID")
	}
}

func TestValidateGoldenQueries_MissingQuery(t *testing.T) {
	queries := []GoldenQuery{
		{ID: "q1", Query: "", Intent: IntentCondition, Difficulty: "easy"},
	}
	err := ValidateGoldenQueries(queries)
	if err == nil {
		t.Error("expected validation error for missing query")
	}
}

func TestValidateGoldenQueries_InvalidIntent(t *testing.T) {
	queries := []GoldenQuery{
		{ID: "q1", Query: "test", Intent: Intent("bad"), Difficulty: "easy"},
	}
	err := ValidateGoldenQueries(queries)
	if err == nil {
		t.Error("expected validation error for invalid intent")
	}
}

func TestValidateGoldenQueries_InvalidDifficulty(t *testing.T) {
	queries := []GoldenQuery{
		{ID: "q1", Query: "test", Intent: IntentCondition, Difficulty: "impossible"},
	}
	err := ValidateGoldenQueries(queries)
	if err == nil {
		t.Error("expected validation error for invalid difficulty")
	}
}

func TestValidateGoldenQueries_DuplicateIDs(t *testing.T) {
	queries := []GoldenQuery{
		{ID: "q1", Query: "malaria", Intent: IntentCondition, Difficulty: "easy"},
		{ID: "q1", Query: "diabetes", Intent: IntentCondition, Difficulty: "easy"},
	}
	err := ValidateGoldenQueries(queries)
	if err == nil {
		t.Error("expected validation error for duplicate IDs")
	}
}

func TestValidateGoldenQueries_Valid(t *testing.T) {
	queries := []GoldenQuery{
		{ID: "q1", Query: "malaria", Intent: IntentCondition, ExpectedTags: []string{"lab"}, Difficulty: "easy"},
		{ID: "q2", Query: "hospital", Intent: IntentFacility, ExpectedFacTypes: []string{"hospital"}, Difficulty: "medium"},
	}
	err := ValidateGoldenQueries(queries)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}
