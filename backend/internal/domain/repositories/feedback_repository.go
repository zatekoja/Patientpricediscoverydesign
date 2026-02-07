package repositories

import (
	"context"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// FeedbackRepository defines the interface for feedback operations.
type FeedbackRepository interface {
	Create(ctx context.Context, feedback *entities.Feedback) error
}
