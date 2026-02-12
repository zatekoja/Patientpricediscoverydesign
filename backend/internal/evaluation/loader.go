package evaluation

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadGoldenQueries reads and parses a golden query set from a JSON file.
func LoadGoldenQueries(path string) ([]GoldenQuery, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read golden queries file: %w", err)
	}

	var queries []GoldenQuery
	if err := json.Unmarshal(data, &queries); err != nil {
		return nil, fmt.Errorf("failed to parse golden queries: %w", err)
	}

	return queries, nil
}

var validDifficulties = map[string]bool{
	"easy":   true,
	"medium": true,
	"hard":   true,
}

// ValidateGoldenQueries checks that all golden queries have required fields and valid values.
func ValidateGoldenQueries(queries []GoldenQuery) error {
	seen := make(map[string]struct{}, len(queries))

	for i, q := range queries {
		if q.ID == "" {
			return fmt.Errorf("query at index %d: missing id", i)
		}
		if _, dup := seen[q.ID]; dup {
			return fmt.Errorf("query at index %d: duplicate id %q", i, q.ID)
		}
		seen[q.ID] = struct{}{}

		if q.Query == "" {
			return fmt.Errorf("query %q: missing query text", q.ID)
		}
		if !q.Intent.IsValid() {
			return fmt.Errorf("query %q: invalid intent %q", q.ID, q.Intent)
		}
		if !validDifficulties[q.Difficulty] {
			return fmt.Errorf("query %q: invalid difficulty %q (must be easy/medium/hard)", q.ID, q.Difficulty)
		}
	}

	return nil
}
