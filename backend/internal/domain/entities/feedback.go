package entities

import "time"

// Feedback captures quick product feedback from users.
type Feedback struct {
	ID        string    `json:"id" db:"id"`
	Rating    int       `json:"rating" db:"rating"`
	Message   string    `json:"message" db:"message"`
	Email     string    `json:"email" db:"email"`
	Page      string    `json:"page" db:"page"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
