package evaluation

import (
	"context"
	"strings"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

type SearchResultProvider interface {
	Search(ctx context.Context, params repositories.SearchParams) ([]entities.FacilitySearchResult, int, error)
}

// Runner runs evaluation across a set of golden queries.
type Runner struct {
	searchService SearchResultProvider
}

func NewRunner(svc SearchResultProvider) *Runner {
	return &Runner{searchService: svc}
}

func (r *Runner) Run(ctx context.Context, queries []GoldenQuery) (*EvalSummary, error) {
	summary := &EvalSummary{
		TotalQueries: len(queries),
		ByIntent:     make(map[Intent]*IntentSummary),
	}

	for _, gq := range queries {
		start := time.Now()
		params := repositories.SearchParams{
			Query:    gq.Query,
			Limit:    10,
			RadiusKm: 50,
		}

		results, count, err := r.searchService.Search(ctx, params)
		duration := time.Since(start)

		if err != nil {
			continue
		}

		resMatchTerms := make([][]string, len(results))
		for i, res := range results {
			// Collect all terms that could satisfy a match
			terms := append([]string{}, res.Tags...)
			terms = append(terms, res.Services...)
			terms = append(terms, strings.ToLower(res.FacilityType))
			resMatchTerms[i] = terms
		}

		// Calculate Recall and MRR
		recall := recallMulti(gq.ExpectedTags, resMatchTerms, 10)
		mrr := mrrMulti(gq.ExpectedTags, resMatchTerms, 10)

		result := EvalResult{
			QueryID:       gq.ID,
			Query:         gq.Query,
			Intent:        gq.Intent,
			RecallAt10:    recall,
			MRRAt10:       mrr,
			ResultCount:   count,
			RetrievedTags: flatten(resMatchTerms),
			Latency:       duration,
		}

		r.updateSummary(summary, result)
	}

	r.finalizeSummary(summary)
	return summary, nil
}

func recallMulti(expected []string, retrievedTerms [][]string, k int) float64 {
	if len(expected) == 0 {
		return 0.0
	}
	foundCount := 0
	for _, exp := range expected {
		match := false
		for i := 0; i < len(retrievedTerms) && i < k; i++ {
			for _, term := range retrievedTerms[i] {
				if strings.Contains(strings.ToLower(term), strings.ToLower(exp)) {
					match = true
					break
				}
			}
			if match {
				break
			}
		}
		if match {
			foundCount++
		}
	}
	return float64(foundCount) / float64(len(expected))
}

func mrrMulti(expected []string, retrievedTerms [][]string, k int) float64 {
	if len(expected) == 0 || len(retrievedTerms) == 0 {
		return 0.0
	}
	for i := 0; i < len(retrievedTerms) && i < k; i++ {
		for _, term := range retrievedTerms[i] {
			for _, exp := range expected {
				if strings.Contains(strings.ToLower(term), strings.ToLower(exp)) {
					return 1.0 / float64(i+1)
				}
			}
		}
	}
	return 0.0
}

func flatten(slices [][]string) []string {
	var result []string
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

func (r *Runner) updateSummary(s *EvalSummary, res EvalResult) {
	s.AvgRecallAt10 += res.RecallAt10
	s.AvgMRRAt10 += res.MRRAt10
	s.AvgLatency += res.Latency
	if res.ResultCount > 0 {
		s.QueriesWithHits++
	}

	if _, ok := s.ByIntent[res.Intent]; !ok {
		s.ByIntent[res.Intent] = &IntentSummary{}
	}
	is := s.ByIntent[res.Intent]
	is.Count++
	is.AvgRecallAt10 += res.RecallAt10
	is.AvgMRRAt10 += res.MRRAt10
}

func (r *Runner) finalizeSummary(s *EvalSummary) {
	if s.TotalQueries > 0 {
		n := float64(s.TotalQueries)
		s.AvgRecallAt10 /= n
		s.AvgMRRAt10 /= n
		s.AvgLatency /= time.Duration(s.TotalQueries)
	}

	for _, is := range s.ByIntent {
		if is.Count > 0 {
			n := float64(is.Count)
			is.AvgRecallAt10 /= n
			is.AvgMRRAt10 /= n
		}
	}
}