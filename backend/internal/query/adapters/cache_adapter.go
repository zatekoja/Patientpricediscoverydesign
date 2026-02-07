package adapters

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

// QueryCacheAdapter wraps the domain CacheProvider to implement query services.QueryCacheProvider
type QueryCacheAdapter struct {
	provider providers.CacheProvider
}

// NewQueryCacheAdapter creates a new query cache adapter
func NewQueryCacheAdapter(provider providers.CacheProvider) *QueryCacheAdapter {
	return &QueryCacheAdapter{provider: provider}
}

// Get retrieves a value from cache and unmarshals it to interface{}
func (a *QueryCacheAdapter) Get(ctx context.Context, key string) (interface{}, error) {
	data, err := a.provider.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Set marshals the value to JSON and stores it in cache
func (a *QueryCacheAdapter) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return a.provider.Set(ctx, key, data, ttl)
}

// Delete removes a value from cache
func (a *QueryCacheAdapter) Delete(ctx context.Context, key string) error {
	return a.provider.Delete(ctx, key)
}
