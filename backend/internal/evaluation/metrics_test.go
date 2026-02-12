package evaluation

import (
	"math"
	"testing"
)

const floatTolerance = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < floatTolerance
}

// --- RecallAtK tests ---

func TestRecallAtK_AllRelevantInTop10(t *testing.T) {
	relevant := []string{"a", "b", "c"}
	retrieved := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	got := RecallAtK(relevant, retrieved, 10)
	if !almostEqual(got, 1.0) {
		t.Errorf("expected 1.0, got %f", got)
	}
}

func TestRecallAtK_SomeRelevantMissing(t *testing.T) {
	relevant := []string{"a", "b", "c", "d"}
	retrieved := []string{"a", "b", "x", "y", "z", "w", "v", "u", "t", "s"}
	got := RecallAtK(relevant, retrieved, 10)
	// 2 of 4 relevant found
	if !almostEqual(got, 0.5) {
		t.Errorf("expected 0.5, got %f", got)
	}
}

func TestRecallAtK_EmptyResults(t *testing.T) {
	relevant := []string{"a", "b"}
	retrieved := []string{}
	got := RecallAtK(relevant, retrieved, 10)
	if !almostEqual(got, 0.0) {
		t.Errorf("expected 0.0, got %f", got)
	}
}

func TestRecallAtK_NoRelevantDocs(t *testing.T) {
	relevant := []string{}
	retrieved := []string{"a", "b", "c"}
	got := RecallAtK(relevant, retrieved, 10)
	// No relevant docs means recall is undefined; we return 0
	if !almostEqual(got, 0.0) {
		t.Errorf("expected 0.0, got %f", got)
	}
}

func TestRecallAtK_KSmallerThanRetrieved(t *testing.T) {
	relevant := []string{"a", "b", "c"}
	// "c" is at position 5 (index 4), but k=3 so we only look at first 3
	retrieved := []string{"a", "b", "x", "y", "c"}
	got := RecallAtK(relevant, retrieved, 3)
	// Only "a" and "b" in top 3
	if !almostEqual(got, 2.0/3.0) {
		t.Errorf("expected %f, got %f", 2.0/3.0, got)
	}
}

func TestRecallAtK_RetrievedShorterThanK(t *testing.T) {
	relevant := []string{"a", "b"}
	retrieved := []string{"a"} // only 1 result, k=10
	got := RecallAtK(relevant, retrieved, 10)
	if !almostEqual(got, 0.5) {
		t.Errorf("expected 0.5, got %f", got)
	}
}

// --- MRRAtK tests ---

func TestMRRAtK_FirstResultRelevant(t *testing.T) {
	relevant := []string{"a", "b"}
	retrieved := []string{"a", "x", "y", "z"}
	got := MRRAtK(relevant, retrieved, 10)
	// First relevant at rank 1, reciprocal = 1/1 = 1.0
	if !almostEqual(got, 1.0) {
		t.Errorf("expected 1.0, got %f", got)
	}
}

func TestMRRAtK_ThirdResultRelevant(t *testing.T) {
	relevant := []string{"a"}
	retrieved := []string{"x", "y", "a", "z"}
	got := MRRAtK(relevant, retrieved, 10)
	// First relevant at rank 3, reciprocal = 1/3
	if !almostEqual(got, 1.0/3.0) {
		t.Errorf("expected %f, got %f", 1.0/3.0, got)
	}
}

func TestMRRAtK_NoRelevantInTop10(t *testing.T) {
	relevant := []string{"a"}
	retrieved := []string{"x", "y", "z", "w", "v", "u", "t", "s", "r", "q", "a"}
	got := MRRAtK(relevant, retrieved, 10)
	// "a" is at rank 11, beyond k=10
	if !almostEqual(got, 0.0) {
		t.Errorf("expected 0.0, got %f", got)
	}
}

func TestMRRAtK_EmptyRelevant(t *testing.T) {
	relevant := []string{}
	retrieved := []string{"a", "b"}
	got := MRRAtK(relevant, retrieved, 10)
	if !almostEqual(got, 0.0) {
		t.Errorf("expected 0.0, got %f", got)
	}
}

func TestMRRAtK_EmptyRetrieved(t *testing.T) {
	relevant := []string{"a"}
	retrieved := []string{}
	got := MRRAtK(relevant, retrieved, 10)
	if !almostEqual(got, 0.0) {
		t.Errorf("expected 0.0, got %f", got)
	}
}

func TestMRRAtK_MultipleRelevant_ReturnsFirst(t *testing.T) {
	relevant := []string{"a", "b", "c"}
	retrieved := []string{"x", "b", "a", "c"}
	got := MRRAtK(relevant, retrieved, 10)
	// First relevant is "b" at rank 2, reciprocal = 1/2
	if !almostEqual(got, 0.5) {
		t.Errorf("expected 0.5, got %f", got)
	}
}
