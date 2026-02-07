package geolocation

import (
	"context"
	"fmt"
	"math"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

// MockGeolocationProvider implements a mock geolocation provider for testing
type MockGeolocationProvider struct{}

// NewMockGeolocationProvider creates a new mock geolocation provider
func NewMockGeolocationProvider() providers.GeolocationProvider {
	return &MockGeolocationProvider{}
}

// Geocode converts an address to coordinates (mock implementation)
func (m *MockGeolocationProvider) Geocode(ctx context.Context, address string) (*providers.Coordinates, error) {
	// Return mock coordinates based on common US cities
	mockCoordinates := map[string]providers.Coordinates{
		"New York":    {Latitude: 40.7128, Longitude: -74.0060},
		"Los Angeles": {Latitude: 34.0522, Longitude: -118.2437},
		"Chicago":     {Latitude: 41.8781, Longitude: -87.6298},
		"Houston":     {Latitude: 29.7604, Longitude: -95.3698},
		"Phoenix":     {Latitude: 33.4484, Longitude: -112.0740},
		"Lagos":       {Latitude: 6.5244, Longitude: 3.3792},
		"Abuja":       {Latitude: 9.0765, Longitude: 7.3986},
	}

	// Simple address matching (in production, use a real geocoding service)
	for city, coords := range mockCoordinates {
		if contains(address, city) {
			return &coords, nil
		}
	}

	// Return a default coordinate
	return &providers.Coordinates{Latitude: 37.7749, Longitude: -122.4194}, nil
}

// ReverseGeocode converts coordinates to an address (mock implementation)
func (m *MockGeolocationProvider) ReverseGeocode(ctx context.Context, lat, lon float64) (*providers.GeocodedAddress, error) {
	return &providers.GeocodedAddress{
		FormattedAddress: fmt.Sprintf("%f, %f", lat, lon),
		Street:           "123 Main St",
		City:             "San Francisco",
		State:            "CA",
		ZipCode:          "94102",
		Country:          "USA",
		Coordinates: providers.Coordinates{
			Latitude:  lat,
			Longitude: lon,
		},
	}, nil
}

// CalculateDistance calculates the distance between two points using Haversine formula
func (m *MockGeolocationProvider) CalculateDistance(ctx context.Context, from, to providers.Coordinates) (float64, error) {
	const earthRadiusKm = 6371.0

	lat1Rad := toRadians(from.Latitude)
	lat2Rad := toRadians(to.Latitude)
	deltaLat := toRadians(to.Latitude - from.Latitude)
	deltaLon := toRadians(to.Longitude - from.Longitude)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := earthRadiusKm * c
	return distance, nil
}

// GetNearbyPlaces finds places within a radius (mock implementation)
func (m *MockGeolocationProvider) GetNearbyPlaces(ctx context.Context, center providers.Coordinates, radiusKm float64, placeType string) ([]*providers.Place, error) {
	// Return mock nearby places
	return []*providers.Place{
		{
			ID:          "1",
			Name:        "Mock Hospital 1",
			Address:     "123 Healthcare Blvd",
			Coordinates: providers.Coordinates{Latitude: center.Latitude + 0.01, Longitude: center.Longitude + 0.01},
			PlaceType:   placeType,
		},
		{
			ID:          "2",
			Name:        "Mock Clinic 2",
			Address:     "456 Medical Ave",
			Coordinates: providers.Coordinates{Latitude: center.Latitude - 0.01, Longitude: center.Longitude - 0.01},
			PlaceType:   placeType,
		},
	}, nil
}

func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(str) > len(substr) && indexOf(str, substr) >= 0)
}

func indexOf(str, substr string) int {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
