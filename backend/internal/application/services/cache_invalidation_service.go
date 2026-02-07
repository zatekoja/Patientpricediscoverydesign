package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

// CacheInvalidationService handles cache invalidation based on events
type CacheInvalidationService struct {
	cache    providers.CacheProvider
	eventBus providers.EventBus
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewCacheInvalidationService creates a new cache invalidation service
func NewCacheInvalidationService(cache providers.CacheProvider, eventBus providers.EventBus) *CacheInvalidationService {
	ctx, cancel := context.WithCancel(context.Background())
	return &CacheInvalidationService{
		cache:    cache,
		eventBus: eventBus,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins listening for events and invalidating cache
func (s *CacheInvalidationService) Start() error {
	// Subscribe to global facility updates
	eventChan, err := s.eventBus.Subscribe(s.ctx, providers.EventChannelFacilityUpdates)
	if err != nil {
		return fmt.Errorf("failed to subscribe to facility updates: %w", err)
	}

	go s.processEvents(eventChan)
	log.Println("Cache invalidation service started")
	return nil
}

// Stop stops the cache invalidation service
func (s *CacheInvalidationService) Stop() {
	s.cancel()
	log.Println("Cache invalidation service stopped")
}

// processEvents processes facility events and invalidates cache accordingly
func (s *CacheInvalidationService) processEvents(eventChan <-chan *entities.FacilityEvent) {
	for {
		select {
		case <-s.ctx.Done():
			return
		case event := <-eventChan:
			if event == nil {
				continue
			}
			s.handleEvent(event)
		}
	}
}

// handleEvent handles a single facility event
func (s *CacheInvalidationService) handleEvent(event *entities.FacilityEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("Processing cache invalidation for event: %s (facility: %s, type: %s)",
		event.ID, event.FacilityID, event.EventType)

	// Strategy: Let TTL expire naturally (Option B - performant)
	// We don't invalidate search caches on facility updates to maintain performance
	// Search results have shorter TTLs (3-5 minutes) and will refresh naturally

	// Only invalidate specific facility cache for immediate consistency
	facilityPattern := fmt.Sprintf("http:cache:*facilities/%s*", event.FacilityID)
	if err := s.cache.DeletePattern(ctx, facilityPattern); err != nil {
		log.Printf("Warning: Failed to invalidate facility cache for %s: %v", event.FacilityID, err)
	} else {
		log.Printf("Invalidated facility cache for %s", event.FacilityID)
	}

	// Note: We intentionally DO NOT invalidate search caches here
	// Rationale:
	// - Search results have short TTLs (3-5 minutes)
	// - Invalidating all search results would cause cache stampede
	// - Real-time updates are delivered via SSE for connected clients
	// - Disconnected clients will see updates within TTL window
}

// InvalidateSearchCaches invalidates all search-related caches
// This should only be called during maintenance or major data updates
func (s *CacheInvalidationService) InvalidateSearchCaches(ctx context.Context) error {
	patterns := []string{
		"http:cache:*search*",
		"http:cache:*suggest*",
	}

	for _, pattern := range patterns {
		if err := s.cache.DeletePattern(ctx, pattern); err != nil {
			return fmt.Errorf("failed to invalidate pattern %s: %w", pattern, err)
		}
		log.Printf("Invalidated cache pattern: %s", pattern)
	}

	return nil
}

// InvalidateFacilityCache invalidates cache for a specific facility
func (s *CacheInvalidationService) InvalidateFacilityCache(ctx context.Context, facilityID string) error {
	pattern := fmt.Sprintf("http:cache:*facilities/%s*", facilityID)
	if err := s.cache.DeletePattern(ctx, pattern); err != nil {
		return fmt.Errorf("failed to invalidate facility cache: %w", err)
	}
	log.Printf("Invalidated facility cache for %s", facilityID)
	return nil
}

// InvalidateRegionalCaches invalidates caches for a specific region
// This is useful when multiple facilities in a region are updated
func (s *CacheInvalidationService) InvalidateRegionalCaches(ctx context.Context, lat, lon, radiusKm float64) error {
	// For regional invalidation, we clear search caches that might include the region
	// This is a heavier operation and should be used sparingly
	pattern := "http:cache:*search*"
	if err := s.cache.DeletePattern(ctx, pattern); err != nil {
		return fmt.Errorf("failed to invalidate regional caches: %w", err)
	}
	log.Printf("Invalidated regional caches for %.2f,%.2f (radius: %.0fkm)", lat, lon, radiusKm)
	return nil
}
