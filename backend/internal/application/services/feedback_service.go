package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// FeedbackService handles feedback submissions.
type FeedbackService struct {
	repo repositories.FeedbackRepository
}

// NewFeedbackService creates a new feedback service.
func NewFeedbackService(repo repositories.FeedbackRepository) *FeedbackService {
	return &FeedbackService{repo: repo}
}

// Create stores feedback.
func (s *FeedbackService) Create(ctx context.Context, feedback *entities.Feedback) error {
	if feedback.ID == "" {
		feedback.ID = uuid.New().String()
	}
	if feedback.CreatedAt.IsZero() {
		feedback.CreatedAt = time.Now().UTC()
	}
	return s.repo.Create(ctx, feedback)
}
