package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// CacheWarmingService handles cache warming for frequently accessed data
type CacheWarmingService struct {
	facilityRepo repositories.FacilityRepository
	cache        providers.CacheProvider
}

// NewCacheWarmingService creates a new cache warming service
func NewCacheWarmingService(
	facilityRepo repositories.FacilityRepository,
	cache providers.CacheProvider,
) *CacheWarmingService {
	return &CacheWarmingService{
		facilityRepo: facilityRepo,
		cache:        cache,
	}
}

// WarmCache warms the cache with frequently accessed data
func (s *CacheWarmingService) WarmCache(ctx context.Context) error {
	log.Println("Starting cache warming...")

	// Warm top facilities (by rating)
	if err := s.warmTopFacilities(ctx); err != nil {
		log.Printf("Failed to warm top facilities: %v", err)
	}

	// Warm facilities list (first page)
	if err := s.warmFacilitiesList(ctx); err != nil {
		log.Printf("Failed to warm facilities list: %v", err)
	}

	log.Println("Cache warming completed")
	return nil
}

// warmTopFacilities caches the top-rated facilities
func (s *CacheWarmingService) warmTopFacilities(ctx context.Context) error {
	// Get top 50 facilities by rating
	activeFilter := true
	facilities, err := s.facilityRepo.List(ctx, repositories.FacilityFilter{
		IsActive: &activeFilter,
		Limit:    50,
		Offset:   0,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch top facilities: %w", err)
	}

	// Cache each facility individually using batch operation
	items := make(map[string][]byte)
	for _, facility := range facilities {
		data, err := json.Marshal(facility)
		if err != nil {
			log.Printf("Failed to marshal facility %s: %v", facility.ID, err)
			continue
		}
		key := fmt.Sprintf("facility:%s", facility.ID)
		items[key] = data
	}

	// Batch set to cache with 5 minute TTL
	if len(items) > 0 {
		if err := s.cache.SetMulti(ctx, items, 300); err != nil {
			return fmt.Errorf("failed to cache top facilities: %w", err)
		}
		log.Printf("Warmed cache with %d top facilities", len(items))
	}

	return nil
}

// warmFacilitiesList caches the first few pages of facilities
func (s *CacheWarmingService) warmFacilitiesList(ctx context.Context) error {
	// Warm first 3 pages (assuming 20 per page)
	activeFilter := true
	for page := 0; page < 3; page++ {
		facilities, err := s.facilityRepo.List(ctx, repositories.FacilityFilter{
			IsActive: &activeFilter,
			Limit:    20,
			Offset:   page * 20,
		})
		if err != nil {
			log.Printf("Failed to fetch facilities page %d: %v", page, err)
			continue
		}

		data, err := json.Marshal(facilities)
		if err != nil {
			log.Printf("Failed to marshal facilities list page %d: %v", page, err)
			continue
		}

		key := fmt.Sprintf("facilities:list::%d:%d", 20, page*20)
		if err := s.cache.Set(ctx, key, data, 180); err != nil {
			log.Printf("Failed to cache facilities list page %d: %v", page, err)
		}
	}

	log.Println("Warmed cache with facility lists")
	return nil
}

// StartPeriodicWarming starts a background goroutine that periodically warms the cache
func (s *CacheWarmingService) StartPeriodicWarming(ctx context.Context, interval time.Duration) {
	// Initial warming
	if err := s.WarmCache(ctx); err != nil {
		log.Printf("Initial cache warming failed: %v", err)
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("Stopping cache warming service")
				return
			case <-ticker.C:
				if err := s.WarmCache(context.Background()); err != nil {
					log.Printf("Periodic cache warming failed: %v", err)
				}
			}
		}
	}()
	log.Printf("Started periodic cache warming every %v", interval)
}

// WarmSpecificFacility warms cache for a specific facility
func (s *CacheWarmingService) WarmSpecificFacility(ctx context.Context, facilityID string) error {
	// Get facility
	facility, err := s.facilityRepo.GetByID(ctx, facilityID)
	if err != nil {
		return fmt.Errorf("failed to fetch facility: %w", err)
	}

	// Cache main facility
	facilityData, err := json.Marshal(facility)
	if err != nil {
		return fmt.Errorf("failed to marshal facility: %w", err)
	}

	items := make(map[string][]byte)
	items[fmt.Sprintf("facility:%s", facilityID)] = facilityData

	// Batch set with 5 minute TTL
	if err := s.cache.SetMulti(ctx, items, 300); err != nil {
		return fmt.Errorf("failed to cache facility data: %w", err)
	}

	log.Printf("Warmed cache for facility %s", facilityID)
	return nil
}

// InvalidateCache invalidates all cached data (useful after bulk updates)
func (s *CacheWarmingService) InvalidateCache(ctx context.Context) error {
	patterns := []string{
		"facility:*",
		"facilities:*",
		"procedures:*",
		"insurance:*",
	}

	for _, pattern := range patterns {
		if err := s.cache.DeletePattern(ctx, pattern); err != nil {
			log.Printf("Failed to invalidate cache pattern %s: %v", pattern, err)
		}
	}

	log.Println("Cache invalidated")
	return nil
}

// GetCacheStats returns cache statistics (if available)
func (s *CacheWarmingService) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Check a few sample keys to see if they're cached
	sampleKeys := []string{
		"facilities:list::20:0",
	}

	cachedCount := 0
	for _, key := range sampleKeys {
		exists, err := s.cache.Exists(ctx, key)
		if err != nil {
			continue
		}
		if exists {
			cachedCount++

			// Get TTL for this key
			if ttl, err := s.cache.TTL(ctx, key); err == nil {
				stats[fmt.Sprintf("%s_ttl", key)] = ttl.Seconds()
			}
		}
	}

	stats["cached_sample_keys"] = cachedCount
	stats["total_sample_keys"] = len(sampleKeys)
	if len(sampleKeys) > 0 {
		stats["sample_cache_hit_rate"] = float64(cachedCount) / float64(len(sampleKeys))
	}

	return stats, nil
}
