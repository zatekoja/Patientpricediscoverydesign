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

// GetMulti retrieves multiple values from cache in a single call
func (a *RedisAdapter) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// Use pipeline for efficient batch retrieval
	pipe := a.client.Client().Pipeline()
	cmds := make(map[string]*redis.StringCmd, len(keys))

	for _, key := range keys {
		cmds[key] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		// Ignore redis.Nil as some keys might not exist
	}

	result := make(map[string][]byte)
	for key, cmd := range cmds {
		if val, err := cmd.Bytes(); err == nil {
			result[key] = val
		}
		// Skip keys that don't exist (redis.Nil error)
	}

	return result, nil
}

// SetMulti stores multiple values in cache with expiration
func (a *RedisAdapter) SetMulti(ctx context.Context, items map[string][]byte, expirationSeconds int) error {
	if len(items) == 0 {
		return nil
	}

	expiration := time.Duration(expirationSeconds) * time.Second

	// Use pipeline for efficient batch storage
	pipe := a.client.Client().Pipeline()

	for key, value := range items {
		pipe.Set(ctx, key, value, expiration)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set multiple items in cache: %w", err)
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

// DeletePattern removes all keys matching a pattern from cache
func (a *RedisAdapter) DeletePattern(ctx context.Context, pattern string) error {
	var cursor uint64
	var deletedCount int

	for {
		// Scan for keys matching pattern
		keys, nextCursor, err := a.client.Client().Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan cache keys: %w", err)
		}

		// Delete matching keys
		if len(keys) > 0 {
			if err := a.client.Client().Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("failed to delete cache keys: %w", err)
			}
			deletedCount += len(keys)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// TTL returns the time-to-live for a key
func (a *RedisAdapter) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := a.client.Client().TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL from cache: %w", err)
	}
	return ttl, nil
}
