package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/retry"
)

// Client represents a Redis client
type Client struct {
	client *redis.Client
}

// NewClient creates a new Redis client with exponential backoff retry
func NewClient(cfg *config.RedisConfig) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test the connection with retry
	retryConfig := retry.DefaultConfig()
	err := retry.DoWithLog(
		context.Background(),
		retryConfig,
		"Redis",
		func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return client.Ping(ctx).Err()
		},
		func(attempt int, err error, nextDelay time.Duration) {
			log.Printf("Redis connection attempt %d failed: %v. Retrying in %v...", attempt, err, nextDelay)
		},
	)

	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis after retries: %w", err)
	}

	log.Println("Successfully connected to Redis")
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
