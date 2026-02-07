package providers

import (
	"context"
	"fmt"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// EventBus defines the interface for publishing and subscribing to events
type EventBus interface {
	// Publish publishes an event to all subscribers
	Publish(ctx context.Context, channel string, event *entities.FacilityEvent) error

	// Subscribe subscribes to events on a channel
	Subscribe(ctx context.Context, channel string) (<-chan *entities.FacilityEvent, error)

	// Unsubscribe unsubscribes from a channel
	Unsubscribe(ctx context.Context, channel string) error

	// Close closes the event bus and all subscriptions
	Close() error
}

// EventChannel constants for different event types
const (
	// EventChannelFacilityUpdates is the channel for all facility updates
	EventChannelFacilityUpdates = "facility:updates"

	// EventChannelFacilityPrefix is the prefix for facility-specific channels
	EventChannelFacilityPrefix = "facility:"

	// EventChannelRegionalPrefix is the prefix for regional channels
	EventChannelRegionalPrefix = "region:"
)

// GetFacilityChannel returns the channel name for a specific facility
func GetFacilityChannel(facilityID string) string {
	return EventChannelFacilityPrefix + facilityID
}

// GetRegionalChannel returns the channel name for a specific region
func GetRegionalChannel(lat, lon float64, radiusKm int) string {
	return EventChannelRegionalPrefix + formatRegion(lat, lon, radiusKm)
}

// formatRegion formats a region identifier
func formatRegion(lat, lon float64, radiusKm int) string {
	// Round to 2 decimal places for reasonable grouping
	return fmt.Sprintf("%.2f:%.2f:%d", lat, lon, radiusKm)
}
