package database

import (
	"context"
	"github.com/google/uuid"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
	"time"
)

type SearchAnalyticsAdapter struct {
	client *postgres.Client
}

func NewSearchAnalyticsAdapter(client *postgres.Client) repositories.SearchAnalyticsRepository {
	return &SearchAnalyticsAdapter{client: client}
}

func (a *SearchAnalyticsAdapter) LogEvent(ctx context.Context, event *entities.SearchEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO search_analytics 
		(id, query, normalized_query, detected_intent, intent_confidence, result_count, latency_ms, user_latitude, user_longitude, session_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := a.client.DB().ExecContext(ctx, query,
		event.ID,
		event.Query,
		event.NormalizedQuery,
		event.DetectedIntent,
		event.IntentConfidence,
		event.ResultCount,
		event.LatencyMs,
		event.UserLatitude,
		event.UserLongitude,
		event.SessionID,
		event.CreatedAt,
	)

	if err != nil {
		return apperrors.NewInternalError("failed to log search event", err)
	}

	return nil
}

func (a *SearchAnalyticsAdapter) GetZeroResultQueries(ctx context.Context, limit int) ([]*entities.SearchEvent, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, query, normalized_query, detected_intent, intent_confidence, result_count, latency_ms, user_latitude, user_longitude, session_id, created_at
		FROM search_analytics
		WHERE result_count = 0
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := a.client.DB().QueryContext(ctx, query, limit)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get zero result queries", err)
	}
	defer rows.Close()

	var events []*entities.SearchEvent
	for rows.Next() {
		e := &entities.SearchEvent{}
		err := rows.Scan(
			&e.ID,
			&e.Query,
			&e.NormalizedQuery,
			&e.DetectedIntent,
			&e.IntentConfidence,
			&e.ResultCount,
			&e.LatencyMs,
			&e.UserLatitude,
			&e.UserLongitude,
			&e.SessionID,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to scan search event", err)
		}
		events = append(events, e)
	}

	return events, nil
}
