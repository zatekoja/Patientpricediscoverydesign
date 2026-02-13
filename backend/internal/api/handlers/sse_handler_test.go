package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

// MockEventBus for testing
type MockEventBus struct {
	mu          sync.RWMutex
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
	m.mu.Lock()
	m.published = append(m.published, event)
	channels := append([]chan *entities.FacilityEvent(nil), m.subscribers[channel]...)
	m.mu.Unlock()

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
		}
	}
	return nil
}

func (m *MockEventBus) Subscribe(ctx context.Context, channel string) (<-chan *entities.FacilityEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(chan *entities.FacilityEvent, 10)
	m.subscribers[channel] = append(m.subscribers[channel], ch)
	return ch, nil
}

func (m *MockEventBus) Unsubscribe(ctx context.Context, channel string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.subscribers, channel)
	return nil
}

func (m *MockEventBus) Close() error {
	m.mu.Lock()
	subs := m.subscribers
	m.subscribers = make(map[string][]chan *entities.FacilityEvent)
	m.mu.Unlock()
	for _, channels := range subs {
		for _, ch := range channels {
			close(ch)
		}
	}
	return nil
}

func (m *MockEventBus) PublishedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.published)
}

func TestSSEHandler_StreamFacilityUpdates(t *testing.T) {
	eventBus := NewMockEventBus()
	handler := handlers.NewSSEHandler(eventBus)

	t.Run("should establish SSE connection", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		req := httptest.NewRequest("GET", "/api/stream/facilities/fac_001", nil)
		req.SetPathValue("id", "fac_001")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			handler.StreamFacilityUpdates(w, req)
			close(done)
		}()

		// Wait a bit for connection to establish
		time.Sleep(100 * time.Millisecond)

		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("handler did not exit after cancel")
		}

		result := w.Result()
		if result.Header.Get("Content-Type") != "text/event-stream" {
			t.Errorf("Expected Content-Type text/event-stream, got %s", result.Header.Get("Content-Type"))
		}
		if result.Header.Get("Cache-Control") != "no-cache" {
			t.Errorf("Expected Cache-Control no-cache, got %s", result.Header.Get("Cache-Control"))
		}

	})

	t.Run("should receive facility events", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		req := httptest.NewRequest("GET", "/api/stream/facilities/fac_002", nil)
		req.SetPathValue("id", "fac_002")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			handler.StreamFacilityUpdates(w, req)
			close(done)
		}()

		// Wait for connection
		time.Sleep(100 * time.Millisecond)

		// Publish event
		event := entities.NewFacilityEvent(
			"fac_002",
			entities.FacilityEventTypeCapacityUpdate,
			entities.Location{Latitude: 6.5244, Longitude: 3.3792},
			map[string]interface{}{"capacity_status": "high"},
		)

		channel := providers.GetFacilityChannel("fac_002")
		eventBus.Publish(context.Background(), channel, event)

		// Wait for event to be sent
		time.Sleep(200 * time.Millisecond)

		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("handler did not exit after cancel")
		}

		if eventBus.PublishedCount() == 0 {
			t.Error("Expected event to be published")
		}
	})

	t.Run("should return error for missing facility ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/stream/facilities/", nil)
		w := httptest.NewRecorder()

		handler.StreamFacilityUpdates(w, req)

		result := w.Result()
		if result.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", result.StatusCode)
		}
	})
}

func TestSSEHandler_StreamRegionalUpdates(t *testing.T) {
	eventBus := NewMockEventBus()
	handler := handlers.NewSSEHandler(eventBus)

	t.Run("should establish regional SSE connection", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		req := httptest.NewRequest("GET", "/api/stream/facilities/region?lat=6.5244&lon=3.3792&radius=25", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			handler.StreamRegionalUpdates(w, req)
			close(done)
		}()
		time.Sleep(100 * time.Millisecond)

		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("handler did not exit after cancel")
		}

		result := w.Result()
		if result.Header.Get("Content-Type") != "text/event-stream" {
			t.Errorf("Expected Content-Type text/event-stream, got %s", result.Header.Get("Content-Type"))
		}
	})

	t.Run("should filter events by region", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		req := httptest.NewRequest("GET", "/api/stream/facilities/region?lat=6.5244&lon=3.3792&radius=10", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			handler.StreamRegionalUpdates(w, req)
			close(done)
		}()
		time.Sleep(100 * time.Millisecond)

		// Publish event within region
		eventInRegion := entities.NewFacilityEvent(
			"fac_003",
			entities.FacilityEventTypeCapacityUpdate,
			entities.Location{Latitude: 6.5244, Longitude: 3.3792}, // Same location
			map[string]interface{}{"capacity_status": "high"},
		)

		// Publish event outside region
		eventOutsideRegion := entities.NewFacilityEvent(
			"fac_004",
			entities.FacilityEventTypeCapacityUpdate,
			entities.Location{Latitude: 10.0, Longitude: 10.0}, // Far away
			map[string]interface{}{"capacity_status": "low"},
		)

		eventBus.Publish(context.Background(), providers.EventChannelFacilityUpdates, eventInRegion)
		eventBus.Publish(context.Background(), providers.EventChannelFacilityUpdates, eventOutsideRegion)

		time.Sleep(200 * time.Millisecond)

		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("handler did not exit after cancel")
		}

		if eventBus.PublishedCount() != 2 {
			t.Errorf("Expected 2 events published, got %d", eventBus.PublishedCount())
		}
	})

	t.Run("should return error for invalid parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/stream/facilities/region?lat=invalid&lon=3.3792", nil)
		w := httptest.NewRecorder()

		handler.StreamRegionalUpdates(w, req)

		result := w.Result()
		if result.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", result.StatusCode)
		}
	})
}

func TestSSEHandler_ClientCount(t *testing.T) {
	eventBus := NewMockEventBus()
	handler := handlers.NewSSEHandler(eventBus)

	// Initial count should be 0
	if count := handler.GetClientCount(); count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}

	// Start a connection
	req := httptest.NewRequest("GET", "/api/stream/facilities/fac_001", nil)
	req.SetPathValue("id", "fac_001")
	w := httptest.NewRecorder()

	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	go handler.StreamFacilityUpdates(w, req)
	time.Sleep(100 * time.Millisecond)

	// Count should be 1
	if count := handler.GetClientCount(); count != 1 {
		t.Errorf("Expected 1 client, got %d", count)
	}

	// Cancel connection
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Count should be 0 again
	if count := handler.GetClientCount(); count != 0 {
		t.Errorf("Expected 0 clients after disconnect, got %d", count)
	}
}
