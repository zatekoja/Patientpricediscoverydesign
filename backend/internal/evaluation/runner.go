package evaluation

import (
	"context"
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

		resIDs := make([]string, len(results))
		resTags := make([]string, 0)
		for i, res := range results {
			resIDs[i] = res.ID
			resTags = append(resTags, res.Tags...)
		}

		// Use tags as "relevance" if facility IDs aren't provided in golden queries
		// (Plan said ExpectedTags)
		recall := RecallAtK(gq.ExpectedTags, resTags, 10)
		mrr := MRRAtK(gq.ExpectedTags, resTags, 10)

		result := EvalResult{
			QueryID:       gq.ID,
			Query:         gq.Query,
			Intent:        gq.Intent,
			RecallAt10:    recall,
			MRRAt10:       mrr,
			ResultCount:   count,
			RetrievedTags: resTags,
			Latency:       duration,
		}

		r.updateSummary(summary, result)
	}

	r.finalizeSummary(summary)
	return summary, nil
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