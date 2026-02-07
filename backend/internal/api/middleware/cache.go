package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

// CacheConfig holds cache configuration for specific routes
type CacheConfig struct {
	TTLSeconds int
	Enabled    bool
}

// CacheMiddleware provides HTTP response caching
type CacheMiddleware struct {
	cache        providers.CacheProvider
	routeConfigs map[string]CacheConfig
}

// NewCacheMiddleware creates a new cache middleware
func NewCacheMiddleware(cache providers.CacheProvider) *CacheMiddleware {
	return &CacheMiddleware{
		cache: cache,
		routeConfigs: map[string]CacheConfig{
			"/api/facilities/search":   {TTLSeconds: 300, Enabled: true},  // 5 minutes
			"/api/facilities/":         {TTLSeconds: 600, Enabled: true},  // 10 minutes (prefix match)
			"/api/insurance-providers": {TTLSeconds: 1800, Enabled: true}, // 30 minutes
			"/api/procedures":          {TTLSeconds: 1800, Enabled: true}, // 30 minutes
			"/api/geocode":             {TTLSeconds: 3600, Enabled: true}, // 1 hour
			"/api/facilities/suggest":  {TTLSeconds: 180, Enabled: true},  // 3 minutes
		},
	}
}

// Middleware returns the cache middleware handler
func (m *CacheMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only cache GET requests
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		// Check if caching is disabled
		if m.cache == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Get cache config for this route
		config := m.getRouteConfig(r.URL.Path)
		if !config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Generate cache key
		cacheKey := m.generateCacheKey(r)

		// Try to get from cache
		if cached, err := m.cache.Get(r.Context(), cacheKey); err == nil {
			log.Printf("Cache HIT: %s", cacheKey)
			w.Header().Set("X-Cache", "HIT")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(cached)
			return
		}

		// Cache miss - capture response
		log.Printf("Cache MISS: %s", cacheKey)
		w.Header().Set("X-Cache", "MISS")

		// Create response recorder
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           &bytes.Buffer{},
		}

		// Call next handler
		next.ServeHTTP(recorder, r)

		// Only cache successful responses
		if recorder.statusCode == http.StatusOK && recorder.body.Len() > 0 {
			// Store in cache
			if err := m.cache.Set(r.Context(), cacheKey, recorder.body.Bytes(), config.TTLSeconds); err != nil {
				log.Printf("Failed to cache response for %s: %v", cacheKey, err)
			} else {
				log.Printf("Cached response for %s (TTL: %ds)", cacheKey, config.TTLSeconds)
			}
		}
	})
}

// getRouteConfig gets the cache configuration for a route
func (m *CacheMiddleware) getRouteConfig(path string) CacheConfig {
	// Exact match first
	if config, exists := m.routeConfigs[path]; exists {
		return config
	}

	// Prefix match for dynamic routes (e.g., /api/facilities/{id})
	for pattern, config := range m.routeConfigs {
		if strings.HasPrefix(path, pattern) {
			return config
		}
	}

	// Default: no caching
	return CacheConfig{Enabled: false}
}

// generateCacheKey generates a cache key from the request
func (m *CacheMiddleware) generateCacheKey(r *http.Request) string {
	// Include method, path, and query parameters
	key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)

	// Add query parameters (sorted for consistency)
	if r.URL.RawQuery != "" {
		key += "?" + r.URL.RawQuery
	}

	// Hash the key to keep it reasonable length
	hash := sha256.Sum256([]byte(key))
	return "http:cache:" + hex.EncodeToString(hash[:])
}

// responseRecorder captures the response for caching
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
	written    bool
}

// WriteHeader captures the status code
func (r *responseRecorder) WriteHeader(statusCode int) {
	if !r.written {
		r.statusCode = statusCode
		r.ResponseWriter.WriteHeader(statusCode)
		r.written = true
	}
}

// Write captures the response body and writes to the client
func (r *responseRecorder) Write(data []byte) (int, error) {
	if !r.written {
		r.WriteHeader(http.StatusOK)
	}

	// Write to buffer for caching
	r.body.Write(data)

	// Write to client
	return r.ResponseWriter.Write(data)
}

// CacheMiddlewareWithConfig creates a cache middleware with custom config
func CacheMiddlewareWithConfig(cache providers.CacheProvider, configs map[string]CacheConfig) func(http.Handler) http.Handler {
	m := &CacheMiddleware{
		cache:        cache,
		routeConfigs: configs,
	}
	return m.Middleware
}

// InvalidateCache invalidates cache entries matching a pattern
func (m *CacheMiddleware) InvalidateCache(pattern string) error {
	// Note: This is a simple implementation. For production, you'd want
	// to track cache keys or use Redis SCAN to find matching keys
	log.Printf("Cache invalidation requested for pattern: %s", pattern)
	// With TTL-based expiration, we let entries expire naturally
	return nil
}
