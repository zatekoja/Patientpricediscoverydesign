package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// CachedFacilityAdapter wraps FacilityAdapter with caching
type CachedFacilityAdapter struct {
	adapter repositories.FacilityRepository
	cache   providers.CacheProvider
}

// NewCachedFacilityAdapter creates a new cached facility adapter
func NewCachedFacilityAdapter(adapter repositories.FacilityRepository, cache providers.CacheProvider) repositories.FacilityRepository {
	return &CachedFacilityAdapter{
		adapter: adapter,
		cache:   cache,
	}
}

// Cache TTLs (in seconds)
const (
	facilityByIDTTL   = 300 // 5 minutes for single facility
	facilitiesListTTL = 180 // 3 minutes for lists
	searchResultsTTL  = 120 // 2 minutes for search results
)

// Cache key generators
func facilityCacheKey(id string) string {
	return fmt.Sprintf("facility:%s", id)
}

func facilitiesListCacheKey(filter repositories.FacilityFilter) string {
	return fmt.Sprintf("facilities:list:%s:%d:%d", filter.FacilityType, filter.Limit, filter.Offset)
}

func facilitiesSearchCountCacheKey(params string) string {
	return fmt.Sprintf("facilities:search:count:%s", params)
}

type cachedSearchResult struct {
	Facilities []*entities.Facility `json:"facilities"`
	TotalCount int                  `json:"total_count"`
}

// GetByID retrieves a facility by ID with caching
func (a *CachedFacilityAdapter) GetByID(ctx context.Context, id string) (*entities.Facility, error) {
	cacheKey := facilityCacheKey(id)

	// Try to get from cache first
	if cached, err := a.cache.Get(ctx, cacheKey); err == nil {
		var facility entities.Facility
		if err := json.Unmarshal(cached, &facility); err == nil {
			return &facility, nil
		}
		// If unmarshal fails, continue to fetch from DB
		log.Printf("Failed to unmarshal cached facility %s: %v", id, err)
	}

	// Cache miss - fetch from database
	facility, err := a.adapter.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update cache asynchronously to avoid blocking the response
	go func() {
		bgCtx := context.Background()
		if data, err := json.Marshal(facility); err == nil {
			if err := a.cache.Set(bgCtx, cacheKey, data, facilityByIDTTL); err != nil {
				log.Printf("Failed to cache facility %s: %v", id, err)
			}
		}
	}()

	return facility, nil
}

// GetByIDs retrieves multiple facilities by IDs with batch caching
func (a *CachedFacilityAdapter) GetByIDs(ctx context.Context, ids []string) ([]*entities.Facility, error) {
	if len(ids) == 0 {
		return []*entities.Facility{}, nil
	}

	// Try to get all from cache first using batch operation
	cacheKeys := make([]string, len(ids))
	for i, id := range ids {
		cacheKeys[i] = facilityCacheKey(id)
	}

	cached, _ := a.cache.GetMulti(ctx, cacheKeys)

	var cachedFacilities []*entities.Facility
	missingIDs := make([]string, 0)

	// Build map of ID to cache key for lookup
	idToCacheKey := make(map[string]string)
	for i, id := range ids {
		idToCacheKey[id] = cacheKeys[i]
	}

	for _, id := range ids {
		cacheKey := idToCacheKey[id]
		if data, ok := cached[cacheKey]; ok {
			var facility entities.Facility
			if err := json.Unmarshal(data, &facility); err == nil {
				cachedFacilities = append(cachedFacilities, &facility)
				continue
			}
		}
		missingIDs = append(missingIDs, id)
	}

	// If all were cached, return them
	if len(missingIDs) == 0 {
		return cachedFacilities, nil
	}

	// Fetch missing facilities from database
	dbFacilities, err := a.adapter.GetByIDs(ctx, missingIDs)
	if err != nil {
		return nil, err
	}

	// Cache the missing facilities asynchronously using batch operation
	go func() {
		bgCtx := context.Background()
		items := make(map[string][]byte)
		for _, facility := range dbFacilities {
			if data, err := json.Marshal(facility); err == nil {
				items[facilityCacheKey(facility.ID)] = data
			}
		}
		if len(items) > 0 {
			if err := a.cache.SetMulti(bgCtx, items, facilityByIDTTL); err != nil {
				log.Printf("Failed to batch cache facilities: %v", err)
			}
		}
	}()

	// Combine cached and db results
	allFacilities := append(cachedFacilities, dbFacilities...)
	return allFacilities, nil
}

// List retrieves a list of facilities with caching
func (a *CachedFacilityAdapter) List(ctx context.Context, filter repositories.FacilityFilter) ([]*entities.Facility, error) {
	cacheKey := facilitiesListCacheKey(filter)

	// Try to get from cache first
	if cached, err := a.cache.Get(ctx, cacheKey); err == nil {
		var facilities []*entities.Facility
		if err := json.Unmarshal(cached, &facilities); err == nil {
			return facilities, nil
		}
		log.Printf("Failed to unmarshal cached facilities list: %v", err)
	}

	// Cache miss - fetch from database
	facilities, err := a.adapter.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Update cache asynchronously
	go func() {
		bgCtx := context.Background()
		if data, err := json.Marshal(facilities); err == nil {
			if err := a.cache.Set(bgCtx, cacheKey, data, facilitiesListTTL); err != nil {
				log.Printf("Failed to cache facilities list: %v", err)
			}
		}
	}()

	return facilities, nil
}

// Create creates a facility and invalidates related caches
func (a *CachedFacilityAdapter) Create(ctx context.Context, facility *entities.Facility) error {
	// Create in database
	err := a.adapter.Create(ctx, facility)
	if err != nil {
		return err
	}

	// Invalidate list caches asynchronously
	go func() {
		bgCtx := context.Background()
		if err := a.cache.DeletePattern(bgCtx, "facilities:list:*"); err != nil {
			log.Printf("Failed to invalidate facilities list cache: %v", err)
		}
		if err := a.cache.DeletePattern(bgCtx, "facilities:search:*"); err != nil {
			log.Printf("Failed to invalidate facilities search cache: %v", err)
		}
	}()

	return nil
}

// Update updates a facility and invalidates its cache
func (a *CachedFacilityAdapter) Update(ctx context.Context, facility *entities.Facility) error {
	// Update in database
	err := a.adapter.Update(ctx, facility)
	if err != nil {
		return err
	}

	// Invalidate caches asynchronously
	go func() {
		bgCtx := context.Background()

		// Delete specific facility cache
		cacheKey := facilityCacheKey(facility.ID)
		if err := a.cache.Delete(bgCtx, cacheKey); err != nil {
			log.Printf("Failed to invalidate facility cache %s: %v", facility.ID, err)
		}

		// Delete list and search caches
		if err := a.cache.DeletePattern(bgCtx, "facilities:list:*"); err != nil {
			log.Printf("Failed to invalidate facilities list cache: %v", err)
		}
		if err := a.cache.DeletePattern(bgCtx, "facilities:search:*"); err != nil {
			log.Printf("Failed to invalidate facilities search cache: %v", err)
		}
	}()

	return nil
}

// Delete deletes a facility and invalidates its cache
func (a *CachedFacilityAdapter) Delete(ctx context.Context, id string) error {
	// Delete from database
	err := a.adapter.Delete(ctx, id)
	if err != nil {
		return err
	}

	// Invalidate caches asynchronously
	go func() {
		bgCtx := context.Background()

		// Delete specific facility cache
		cacheKey := facilityCacheKey(id)
		if err := a.cache.Delete(bgCtx, cacheKey); err != nil {
			log.Printf("Failed to invalidate facility cache %s: %v", id, err)
		}

		// Delete list and search caches
		if err := a.cache.DeletePattern(bgCtx, "facilities:list:*"); err != nil {
			log.Printf("Failed to invalidate facilities list cache: %v", err)
		}
		if err := a.cache.DeletePattern(bgCtx, "facilities:search:*"); err != nil {
			log.Printf("Failed to invalidate facilities search cache: %v", err)
		}
	}()

	return nil
}

// Search searches for facilities with caching
func (a *CachedFacilityAdapter) Search(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, error) {
	facilities, _, err := a.SearchWithCount(ctx, params)
	return facilities, err
}

// SearchWithCount searches for facilities with caching and returns total count.
func (a *CachedFacilityAdapter) SearchWithCount(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, int, error) {
	// Generate cache key from search parameters
	paramsJSON, _ := json.Marshal(params)
	cacheKey := facilitiesSearchCountCacheKey(string(paramsJSON))

	// Try to get from cache first
	if cached, err := a.cache.Get(ctx, cacheKey); err == nil {
		var cachedResult cachedSearchResult
		if err := json.Unmarshal(cached, &cachedResult); err == nil {
			return cachedResult.Facilities, cachedResult.TotalCount, nil
		}
		log.Printf("Failed to unmarshal cached search results: %v", err)
	}

	// Cache miss - search in database or underlying adapter
	if adapterWithCount, ok := a.adapter.(interface {
		SearchWithCount(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, int, error)
	}); ok {
		facilities, totalCount, err := adapterWithCount.SearchWithCount(ctx, params)
		if err != nil {
			return nil, 0, err
		}

		// Update cache asynchronously
		go func() {
			bgCtx := context.Background()
			payload := cachedSearchResult{Facilities: facilities, TotalCount: totalCount}
			if data, err := json.Marshal(payload); err == nil {
				if err := a.cache.Set(bgCtx, cacheKey, data, searchResultsTTL); err != nil {
					log.Printf("Failed to cache search results: %v", err)
				}
			}
		}()

		return facilities, totalCount, nil
	}

	facilities, err := a.adapter.Search(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	// Update cache asynchronously
	go func() {
		bgCtx := context.Background()
		payload := cachedSearchResult{Facilities: facilities, TotalCount: len(facilities)}
		if data, err := json.Marshal(payload); err == nil {
			if err := a.cache.Set(bgCtx, cacheKey, data, searchResultsTTL); err != nil {
				log.Printf("Failed to cache search results: %v", err)
			}
		}
	}()

	return facilities, len(facilities), nil
}
