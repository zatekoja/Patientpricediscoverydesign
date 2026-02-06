package providers

import (
	"context"
)

// GeolocationProvider defines the interface for geolocation services
type GeolocationProvider interface {
	// Geocode converts an address to coordinates
	Geocode(ctx context.Context, address string) (*Coordinates, error)

	// ReverseGeocode converts coordinates to an address
	ReverseGeocode(ctx context.Context, lat, lon float64) (*GeocodedAddress, error)

	// CalculateDistance calculates the distance between two points in kilometers
	CalculateDistance(ctx context.Context, from, to Coordinates) (float64, error)

	// GetNearbyPlaces finds places within a radius
	GetNearbyPlaces(ctx context.Context, center Coordinates, radiusKm float64, placeType string) ([]*Place, error)
}

// Coordinates represents geographical coordinates
type Coordinates struct {
	Latitude  float64
	Longitude float64
}

// GeocodedAddress represents a geocoded address
type GeocodedAddress struct {
	FormattedAddress string
	Street           string
	City             string
	State            string
	ZipCode          string
	Country          string
	Coordinates      Coordinates
}

// Place represents a geographical place
type Place struct {
	ID          string
	Name        string
	Address     string
	Coordinates Coordinates
	PlaceType   string
}
