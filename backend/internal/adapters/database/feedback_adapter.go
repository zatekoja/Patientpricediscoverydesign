package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
)

// FeedbackAdapter implements feedback persistence in Postgres.
type FeedbackAdapter struct {
	client *postgres.Client
	db     *goqu.Database
}

// NewFeedbackAdapter creates a new feedback adapter.
func NewFeedbackAdapter(client *postgres.Client) repositories.FeedbackRepository {
	return &FeedbackAdapter{
		client: client,
		db:     goqu.New("postgres", client.DB()),
	}
}

// Create inserts a feedback record.
func (a *FeedbackAdapter) Create(ctx context.Context, feedback *entities.Feedback) error {
	if feedback == nil {
		return apperrors.NewInternalError("feedback is nil", fmt.Errorf("feedback is nil"))
	}

	record := goqu.Record{
		"id":         feedback.ID,
		"rating":     feedback.Rating,
		"message":    sql.NullString{String: feedback.Message, Valid: feedback.Message != ""},
		"email":      sql.NullString{String: feedback.Email, Valid: feedback.Email != ""},
		"page":       sql.NullString{String: feedback.Page, Valid: feedback.Page != ""},
		"user_agent": sql.NullString{String: feedback.UserAgent, Valid: feedback.UserAgent != ""},
		"created_at": feedback.CreatedAt,
	}

	query, args, err := a.db.Insert("feedback").Rows(record).ToSQL()
	if err != nil {
		return apperrors.NewInternalError("failed to build feedback insert query", err)
	}

	if _, err := a.client.DB().ExecContext(ctx, query, args...); err != nil {
		return apperrors.NewInternalError("failed to create feedback", err)
	}

	return nil
}
