package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

// SSEHandler handles Server-Sent Events for real-time facility updates
type SSEHandler struct {
	eventBus providers.EventBus
	clients  map[string]map[chan *entities.FacilityEvent]bool // channel -> clients
	mu       sync.RWMutex
}

// NewSSEHandler creates a new SSE handler
func NewSSEHandler(eventBus providers.EventBus) *SSEHandler {
	return &SSEHandler{
		eventBus: eventBus,
		clients:  make(map[string]map[chan *entities.FacilityEvent]bool),
	}
}

// StreamFacilityUpdates handles SSE connections for facility-specific updates
// GET /api/stream/facilities/{id}
func (h *SSEHandler) StreamFacilityUpdates(w http.ResponseWriter, r *http.Request) {
	facilityID := r.PathValue("id")
	if facilityID == "" {
		respondWithError(w, http.StatusBadRequest, "facility ID is required")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondWithError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client channel
	clientChan := make(chan *entities.FacilityEvent, 10)
	channel := providers.GetFacilityChannel(facilityID)

	// Register client
	h.registerClient(channel, clientChan)
	defer h.unregisterClient(channel, clientChan)

	// Subscribe to events
	eventChan, err := h.eventBus.Subscribe(r.Context(), channel)
	if err != nil {
		log.Printf("Failed to subscribe to channel %s: %v", channel, err)
		return
	}

	// Send initial connection event
	h.sendEvent(w, "connected", map[string]interface{}{
		"facility_id": facilityID,
		"timestamp":   time.Now(),
	})

	// Flush to send the initial event
	flusher.Flush()

	// Start forwarding events
	go h.forwardEvents(r.Context(), eventChan, clientChan)

	// Keep connection alive and send events
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			log.Printf("Client disconnected from facility stream: %s", facilityID)
			return
		case <-ticker.C:
			// Send heartbeat
			h.sendEvent(w, "heartbeat", map[string]interface{}{
				"timestamp": time.Now(),
			})
			flusher.Flush()
		case event := <-clientChan:
			if event == nil {
				continue
			}
			// Send facility update
			h.sendEvent(w, string(event.EventType), event)
			flusher.Flush()
		}
	}
}

// StreamRegionalUpdates handles SSE connections for regional facility updates
// GET /api/stream/facilities/region?lat=X&lon=Y&radius=Z
func (h *SSEHandler) StreamRegionalUpdates(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	lat, err := strconv.ParseFloat(query.Get("lat"), 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid latitude parameter")
		return
	}

	lon, err := strconv.ParseFloat(query.Get("lon"), 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid longitude parameter")
		return
	}

	radius := 50 // default radius in km
	if r := query.Get("radius"); r != "" {
		if parsed, err := strconv.Atoi(r); err == nil {
			radius = parsed
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondWithError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client channel
	clientChan := make(chan *entities.FacilityEvent, 50)

	// Subscribe to global facility updates
	channel := providers.EventChannelFacilityUpdates
	h.registerClient(channel, clientChan)
	defer h.unregisterClient(channel, clientChan)

	eventChan, err := h.eventBus.Subscribe(r.Context(), channel)
	if err != nil {
		log.Printf("Failed to subscribe to channel %s: %v", channel, err)
		return
	}

	// Send initial connection event
	h.sendEvent(w, "connected", map[string]interface{}{
		"lat":       lat,
		"lon":       lon,
		"radius_km": radius,
		"timestamp": time.Now(),
	})

	flusher.Flush()

	// Filter events by region
	regionLat, regionLon, regionRadius := lat, lon, float64(radius)
	go h.forwardRegionalEvents(r.Context(), eventChan, clientChan, regionLat, regionLon, regionRadius)

	// Keep connection alive and send events
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			log.Printf("Client disconnected from regional stream: %.2f,%.2f (radius: %dkm)", lat, lon, radius)
			return
		case <-ticker.C:
			// Send heartbeat
			h.sendEvent(w, "heartbeat", map[string]interface{}{
				"timestamp": time.Now(),
			})
			flusher.Flush()
		case event := <-clientChan:
			if event == nil {
				continue
			}
			// Send facility update
			h.sendEvent(w, string(event.EventType), event)
			flusher.Flush()
		}
	}
}

// forwardEvents forwards events from the event bus to a client channel
func (h *SSEHandler) forwardEvents(ctx context.Context, eventChan <-chan *entities.FacilityEvent, clientChan chan<- *entities.FacilityEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eventChan:
			if !ok {
				return
			}
			select {
			case clientChan <- event:
			default:
				// Client channel full, skip event
			}
		}
	}
}

// forwardRegionalEvents forwards events within a specific region
func (h *SSEHandler) forwardRegionalEvents(ctx context.Context, eventChan <-chan *entities.FacilityEvent, clientChan chan<- *entities.FacilityEvent, lat, lon, radiusKm float64) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eventChan:
			if !ok {
				return
			}
			// Check if event is within region
			distance := haversineDistance(lat, lon, event.Location.Latitude, event.Location.Longitude)
			if distance <= radiusKm {
				select {
				case clientChan <- event:
				default:
					// Client channel full, skip event
				}
			}
		}
	}
}

// registerClient registers a client for a channel
func (h *SSEHandler) registerClient(channel string, clientChan chan *entities.FacilityEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[channel] == nil {
		h.clients[channel] = make(map[chan *entities.FacilityEvent]bool)
	}
	h.clients[channel][clientChan] = true
	log.Printf("Client registered for channel: %s (total: %d)", channel, len(h.clients[channel]))
}

// unregisterClient unregisters a client from a channel
func (h *SSEHandler) unregisterClient(channel string, clientChan chan *entities.FacilityEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, exists := h.clients[channel]; exists {
		delete(clients, clientChan)
		log.Printf("Client unregistered from channel: %s (remaining: %d)", channel, len(clients))

		// Clean up empty channel
		if len(clients) == 0 {
			delete(h.clients, channel)
		}
	}
}

// sendEvent sends an SSE event to the client
func (h *SSEHandler) sendEvent(w http.ResponseWriter, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal event data: %v", err)
		return
	}

	fmt.Fprintf(w, "event: %s\n", eventType)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
}

// haversineDistance calculates the distance between two points in kilometers
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := toRadians(lat2 - lat1)
	dLon := toRadians(lon2 - lon1)

	lat1Rad := toRadians(lat1)
	lat2Rad := toRadians(lat2)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// toRadians converts degrees to radians
func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// GetClientCount returns the number of connected clients for debugging
func (h *SSEHandler) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, clients := range h.clients {
		count += len(clients)
	}
	return count
}
