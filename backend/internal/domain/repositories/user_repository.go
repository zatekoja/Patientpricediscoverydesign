package repositories

import (
	"context"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *entities.User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id string) (*entities.User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*entities.User, error)

	// Update updates a user
	Update(ctx context.Context, user *entities.User) error

	// Delete deletes a user
	Delete(ctx context.Context, id string) error
}

// ReviewRepository defines the interface for review operations
type ReviewRepository interface {
	// Create creates a new review
	Create(ctx context.Context, review *entities.Review) error

	// GetByID retrieves a review by ID
	GetByID(ctx context.Context, id string) (*entities.Review, error)

	// ListByFacility retrieves reviews for a facility
	ListByFacility(ctx context.Context, facilityID string, limit, offset int) ([]*entities.Review, error)

	// ListByUser retrieves reviews by a user
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*entities.Review, error)

	// Update updates a review
	Update(ctx context.Context, review *entities.Review) error

	// Delete deletes a review
	Delete(ctx context.Context, id string) error
}
