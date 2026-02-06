package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

// Client represents a Redis client
type Client struct {
	client *redis.Client
}

// NewClient creates a new Redis client
func NewClient(cfg *config.RedisConfig) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test the connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{client: client}, nil
}

// Client returns the underlying Redis client
func (c *Client) Client() *redis.Client {
	return c.client
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// Ping verifies the connection to Redis
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
