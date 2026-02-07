package providers

import (
	"context"
	"time"
)

// CacheProvider defines the interface for caching operations
type CacheProvider interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in cache with expiration
	Set(ctx context.Context, key string, value []byte, expirationSeconds int) error

	// GetMulti retrieves multiple values from cache in a single call (batch operation)
	GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)

	// SetMulti stores multiple values in cache with expiration (batch operation)
	SetMulti(ctx context.Context, items map[string][]byte, expirationSeconds int) error

	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)

	// DeletePattern removes all keys matching a pattern from cache
	DeletePattern(ctx context.Context, pattern string) error

	// TTL returns the time-to-live for a key
	TTL(ctx context.Context, key string) (time.Duration, error)
}
