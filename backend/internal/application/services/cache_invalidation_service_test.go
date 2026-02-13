package services_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

// MockCacheProvider for testing
type MockCacheProvider struct {
	mu      sync.RWMutex
	data    map[string][]byte
	deleted []string
}

func NewMockCacheProvider() *MockCacheProvider {
	return &MockCacheProvider{
		data:    make(map[string][]byte),
		deleted: make([]string, 0),
	}
}

func (m *MockCacheProvider) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return nil, nil
}

func (m *MockCacheProvider) Set(ctx context.Context, key string, value []byte, expirationSeconds int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *MockCacheProvider) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string][]byte)
	for _, key := range keys {
		if val, ok := m.data[key]; ok {
			result[key] = val
		}
	}
	return result, nil
}

func (m *MockCacheProvider) SetMulti(ctx context.Context, items map[string][]byte, expirationSeconds int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, value := range items {
		m.data[key] = value
	}
	return nil
}

func (m *MockCacheProvider) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	m.deleted = append(m.deleted, key)
	return nil
}

func (m *MockCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[key]
	return ok, nil
}

func (m *MockCacheProvider) DeletePattern(ctx context.Context, pattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Simple pattern matching for tests
	for key := range m.data {
		// Mock implementation - just delete all keys for testing
		delete(m.data, key)
		m.deleted = append(m.deleted, key)
	}
	return nil
}

func (m *MockCacheProvider) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Mock implementation - return a default TTL if key exists
	if _, ok := m.data[key]; ok {
		return time.Minute * 5, nil
	}
	return 0, nil
}

func (m *MockCacheProvider) DeletedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.deleted)
}

// MockEventBus for testing
type MockEventBus struct {
	subscribers map[string][]chan *entities.FacilityEvent
	published   []*entities.FacilityEvent
}

func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		subscribers: make(map[string][]chan *entities.FacilityEvent),
		published:   make([]*entities.FacilityEvent, 0),
	}
}

func (m *MockEventBus) Publish(ctx context.Context, channel string, event *entities.FacilityEvent) error {
	m.published = append(m.published, event)
	if channels, ok := m.subscribers[channel]; ok {
		for _, ch := range channels {
			select {
			case ch <- event:
			default:
			}
		}
	}
	return nil
}

func (m *MockEventBus) Subscribe(ctx context.Context, channel string) (<-chan *entities.FacilityEvent, error) {
	ch := make(chan *entities.FacilityEvent, 10)
	m.subscribers[channel] = append(m.subscribers[channel], ch)
	return ch, nil
}

func (m *MockEventBus) Unsubscribe(ctx context.Context, channel string) error {
	if channels, ok := m.subscribers[channel]; ok {
		for _, ch := range channels {
			close(ch)
		}
		delete(m.subscribers, channel)
	}
	return nil
}

func (m *MockEventBus) Close() error {
	for _, channels := range m.subscribers {
		for _, ch := range channels {
			close(ch)
		}
	}
	return nil
}

func TestCacheInvalidationService_Start(t *testing.T) {
	cache := NewMockCacheProvider()
	eventBus := NewMockEventBus()
	service := services.NewCacheInvalidationService(cache, eventBus)

	err := service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	// Verify subscription was created
	if len(eventBus.subscribers) != 1 {
		t.Errorf("Expected 1 subscriber, got %d", len(eventBus.subscribers))
	}

	service.Stop()
}

func TestCacheInvalidationService_HandleEvent(t *testing.T) {
	cache := NewMockCacheProvider()
	eventBus := NewMockEventBus()
	service := services.NewCacheInvalidationService(cache, eventBus)

	// Start service
	err := service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer service.Stop()

	// Add some cache data
	if err := cache.Set(context.Background(), "http:cache:facilities/fac_001", []byte("data"), 300); err != nil {
		t.Fatalf("Failed to seed cache data: %v", err)
	}

	// Publish facility event
	event := entities.NewFacilityEvent(
		"fac_001",
		entities.FacilityEventTypeCapacityUpdate,
		entities.Location{Latitude: 6.5244, Longitude: 3.3792},
		map[string]interface{}{"capacity_status": "high"},
	)

	if err := eventBus.Publish(context.Background(), providers.EventChannelFacilityUpdates, event); err != nil {
		t.Fatalf("Failed to publish facility event: %v", err)
	}

	// Wait for event processing
	time.Sleep(200 * time.Millisecond)

	// Verify cache was invalidated
	if cache.DeletedCount() == 0 {
		t.Error("Expected cache to be invalidated")
	}
}

func TestCacheInvalidationService_InvalidateFacilityCache(t *testing.T) {
	cache := NewMockCacheProvider()
	eventBus := NewMockEventBus()
	service := services.NewCacheInvalidationService(cache, eventBus)

	// Add cache data
	if err := cache.Set(context.Background(), "http:cache:facilities/fac_001", []byte("data"), 300); err != nil {
		t.Fatalf("Failed to seed cache data: %v", err)
	}

	// Invalidate facility cache
	err := service.InvalidateFacilityCache(context.Background(), "fac_001")
	if err != nil {
		t.Fatalf("Failed to invalidate facility cache: %v", err)
	}

	// Verify cache was deleted
	if cache.DeletedCount() == 0 {
		t.Error("Expected cache keys to be deleted")
	}
}

func TestCacheInvalidationService_InvalidateSearchCaches(t *testing.T) {
	cache := NewMockCacheProvider()
	eventBus := NewMockEventBus()
	service := services.NewCacheInvalidationService(cache, eventBus)

	// Add cache data
	if err := cache.Set(context.Background(), "http:cache:search:1", []byte("data"), 300); err != nil {
		t.Fatalf("Failed to seed cache data: %v", err)
	}
	if err := cache.Set(context.Background(), "http:cache:search:2", []byte("data"), 300); err != nil {
		t.Fatalf("Failed to seed cache data: %v", err)
	}

	// Invalidate search caches
	err := service.InvalidateSearchCaches(context.Background())
	if err != nil {
		t.Fatalf("Failed to invalidate search caches: %v", err)
	}

	// Verify caches were deleted
	if cache.DeletedCount() == 0 {
		t.Error("Expected cache keys to be deleted")
	}
}

func TestCacheInvalidationService_InvalidateRegionalCaches(t *testing.T) {
	cache := NewMockCacheProvider()
	eventBus := NewMockEventBus()
	service := services.NewCacheInvalidationService(cache, eventBus)

	// Add cache data
	if err := cache.Set(context.Background(), "http:cache:search:region", []byte("data"), 300); err != nil {
		t.Fatalf("Failed to seed cache data: %v", err)
	}

	// Invalidate regional caches
	err := service.InvalidateRegionalCaches(context.Background(), 6.5244, 3.3792, 25.0)
	if err != nil {
		t.Fatalf("Failed to invalidate regional caches: %v", err)
	}

	// Verify caches were deleted
	if cache.DeletedCount() == 0 {
		t.Error("Expected cache keys to be deleted")
	}
}
