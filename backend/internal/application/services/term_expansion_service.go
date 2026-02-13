package services

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

// TermExpansionService handles expansion of search terms into synonyms and related concepts
type TermExpansionService struct {
	terms map[string][]string
	mu    sync.RWMutex
}

// NewTermExpansionService creates a new term expansion service
func NewTermExpansionService(configPath string) (*TermExpansionService, error) {
	s := &TermExpansionService{
		terms: make(map[string][]string),
	}
	if err := s.loadConfig(configPath); err != nil {
		return nil, err
	}
	return s, nil
}

// loadConfig loads the term mappings from a JSON file
func (s *TermExpansionService) loadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var mappings map[string][]string
	if err := json.Unmarshal(data, &mappings); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Normalize keys to lowercase for consistent lookup
	for k, v := range mappings {
		s.terms[strings.ToLower(k)] = v
	}
	return nil
}

// Expand expands a search query into a list of related terms including the original terms
func (s *TermExpansionService) Expand(query string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return []string{}
	}

	// Split by space to handle multi-word queries
	// TODO: Can be improved to handle multi-word concepts if needed (e.g. "heart attack")
	// For now, simple tokenization
	rawTerms := strings.Fields(query)

	var expanded []string
	seen := make(map[string]bool)

	// Add original terms
	for _, term := range rawTerms {
		if !seen[term] {
			expanded = append(expanded, term)
			seen[term] = true
		}

		// Add expansions
		if synonyms, ok := s.terms[term]; ok {
			for _, syn := range synonyms {
				if !seen[syn] {
					expanded = append(expanded, syn)
					seen[syn] = true
				}
			}
		}
	}

	return expanded
}
