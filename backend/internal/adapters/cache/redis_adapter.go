package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	redisclient "github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/redis"
)

// RedisAdapter implements the CacheProvider interface using Redis
type RedisAdapter struct {
	client *redisclient.Client
}

// NewRedisAdapter creates a new Redis cache adapter
func NewRedisAdapter(client *redisclient.Client) providers.CacheProvider {
	return &RedisAdapter{
		client: client,
	}
}

// Get retrieves a value from cache
func (a *RedisAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	result, err := a.client.Client().Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}
	return result, nil
}

// Set stores a value in cache with expiration
func (a *RedisAdapter) Set(ctx context.Context, key string, value []byte, expirationSeconds int) error {
	expiration := time.Duration(expirationSeconds) * time.Second
	if err := a.client.Client().Set(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set in cache: %w", err)
	}
	return nil
}

// Delete removes a value from cache
func (a *RedisAdapter) Delete(ctx context.Context, key string) error {
	if err := a.client.Client().Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete from cache: %w", err)
	}
	return nil
}

// Exists checks if a key exists in cache
func (a *RedisAdapter) Exists(ctx context.Context, key string) (bool, error) {
	result, err := a.client.Client().Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence in cache: %w", err)
	}
	return result > 0, nil
}
