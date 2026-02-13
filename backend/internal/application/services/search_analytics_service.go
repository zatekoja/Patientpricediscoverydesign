package services

import (
	"context"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"log"
	"time"
)

type SearchAnalyticsService struct {
	repo repositories.SearchAnalyticsRepository
}

func NewSearchAnalyticsService(repo repositories.SearchAnalyticsRepository) *SearchAnalyticsService {
	return &SearchAnalyticsService{repo: repo}
}

func (s *SearchAnalyticsService) TrackSearch(ctx context.Context, event *entities.SearchEvent) {
	// Execute in background to not block the user request
	go func() {
		// Use a fresh context since the request context might be cancelled
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.repo.LogEvent(bgCtx, event); err != nil {
			log.Printf("Warning: failed to log search event: %v", err)
		}
	}()
}

func (s *SearchAnalyticsService) GetZeroResultQueries(ctx context.Context, limit int) ([]*entities.SearchEvent, error) {
	return s.repo.GetZeroResultQueries(ctx, limit)
}
