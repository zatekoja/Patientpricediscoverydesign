package repositories

import (
	"context"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

type SearchAnalyticsRepository interface {
	LogEvent(ctx context.Context, event *entities.SearchEvent) error
	GetZeroResultQueries(ctx context.Context, limit int) ([]*entities.SearchEvent, error)
}
