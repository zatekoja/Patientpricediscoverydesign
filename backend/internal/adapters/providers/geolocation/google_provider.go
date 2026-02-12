package geolocation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

const (
	googleGeocodeURL       = "https://maps.googleapis.com/maps/api/geocode/json"
	googlePlacesTextURL    = "https://maps.googleapis.com/maps/api/place/textsearch/json"
	defaultGeocodeCacheTTL = 60 * 60 * 24 * 30
	defaultReverseCacheTTL = 60 * 60 * 24 * 30
	defaultHTTPTimeout     = 8 * time.Second
)

// GoogleGeolocationProvider implements the GeolocationProvider using Google Maps APIs.
type GoogleGeolocationProvider struct {
	apiKey     string
	httpClient *http.Client
	cache      providers.CacheProvider
	baseURL    string
	placesURL  string
}

// NewGoogleGeolocationProvider creates a new Google geolocation provider.
func NewGoogleGeolocationProvider(apiKey string, cache providers.CacheProvider) providers.GeolocationProvider {
	return NewGoogleGeolocationProviderWithOptions(apiKey, cache, googleGeocodeURL, nil)
}

// NewGoogleGeolocationProviderWithOptions allows overriding base URL and HTTP client (used for tests).
func NewGoogleGeolocationProviderWithOptions(apiKey string, cache providers.CacheProvider, baseURL string, httpClient *http.Client) providers.GeolocationProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = googleGeocodeURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	placesURL := googlePlacesTextURL
	if baseURL != googleGeocodeURL {
		if strings.HasSuffix(baseURL, "/geocode") {
			placesURL = strings.TrimSuffix(baseURL, "/geocode") + "/place/textsearch"
		} else {
			placesURL = ""
		}
	}
	return &GoogleGeolocationProvider{
		apiKey:     apiKey,
		httpClient: httpClient,
		cache:      cache,
		baseURL:    baseURL,
		placesURL:  placesURL,
	}
}

// Geocode converts an address to a full geocoded address.
func (g *GoogleGeolocationProvider) Geocode(ctx context.Context, address string) (*providers.GeocodedAddress, error) {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return nil, fmt.Errorf("address is required")
	}

	cacheKey := "geo:v2:geocode:" + hashKey(strings.ToLower(trimmed))
	if g.cache != nil {
		if cached, err := g.cache.Get(ctx, cacheKey); err == nil && len(cached) > 0 {
			var addr providers.GeocodedAddress
			if err := json.Unmarshal(cached, &addr); err == nil && (addr.Coordinates.Latitude != 0 || addr.Coordinates.Longitude != 0) {
				return &addr, nil
			}
		}
	}

	if g.placesURL != "" {
		if addr, err := g.searchPlaceAddress(ctx, trimmed); err == nil && addr != nil {
			if g.cache != nil {
				if payload, err := json.Marshal(*addr); err == nil {
					_ = g.cache.Set(ctx, cacheKey, payload, defaultGeocodeCacheTTL)
				}
			}
			return addr, nil
		}
	}

	resp, err := g.doGeocodeRequest(ctx, url.Values{"address": []string{trimmed}})
	if err != nil {
		return nil, err
	}

	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("no results for address")
	}

	result := resp.Results[0]
	addr := providers.GeocodedAddress{
		FormattedAddress: result.FormattedAddress,
		Street:           buildStreet(result.AddressComponents),
		City:             component(result.AddressComponents, "locality", "administrative_area_level_2"),
		State:            component(result.AddressComponents, "administrative_area_level_1"),
		ZipCode:          component(result.AddressComponents, "postal_code"),
		Country:          component(result.AddressComponents, "country"),
		Coordinates: providers.Coordinates{
			Latitude:  result.Geometry.Location.Lat,
			Longitude: result.Geometry.Location.Lng,
		},
	}

	if g.cache != nil {
		if payload, err := json.Marshal(addr); err == nil {
			_ = g.cache.Set(ctx, cacheKey, payload, defaultGeocodeCacheTTL)
		}
	}

	return &addr, nil
}

// ReverseGeocode converts coordinates to an address.
func (g *GoogleGeolocationProvider) ReverseGeocode(ctx context.Context, lat, lon float64) (*providers.GeocodedAddress, error) {
	cacheKey := "geo:v2:reverse:" + hashKey(fmt.Sprintf("%.5f,%.5f", lat, lon))
	if g.cache != nil {
		if cached, err := g.cache.Get(ctx, cacheKey); err == nil && len(cached) > 0 {
			var address providers.GeocodedAddress
			if err := json.Unmarshal(cached, &address); err == nil && (address.Coordinates.Latitude != 0 || address.Coordinates.Longitude != 0) {
				return &address, nil
			}
		}
	}

	resp, err := g.doGeocodeRequest(ctx, url.Values{"latlng": []string{fmt.Sprintf("%f,%f", lat, lon)}})
	if err != nil {
		return nil, err
	}

	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("no results for coordinates")
	}

	result := resp.Results[0]
	address := providers.GeocodedAddress{
		FormattedAddress: result.FormattedAddress,
		Street:           buildStreet(result.AddressComponents),
		City:             component(result.AddressComponents, "locality", "administrative_area_level_2"),
		State:            component(result.AddressComponents, "administrative_area_level_1"),
		ZipCode:          component(result.AddressComponents, "postal_code"),
		Country:          component(result.AddressComponents, "country"),
		Coordinates: providers.Coordinates{
			Latitude:  result.Geometry.Location.Lat,
			Longitude: result.Geometry.Location.Lng,
		},
	}

	if g.cache != nil {
		if payload, err := json.Marshal(address); err == nil {
			_ = g.cache.Set(ctx, cacheKey, payload, defaultReverseCacheTTL)
		}
	}

	return &address, nil
}

// CalculateDistance calculates the distance between two points using the Haversine formula.
func (g *GoogleGeolocationProvider) CalculateDistance(ctx context.Context, from, to providers.Coordinates) (float64, error) {
	mock := MockGeolocationProvider{}
	return mock.CalculateDistance(ctx, from, to)
}

// GetNearbyPlaces returns an error for now since we are keeping the integration minimal.
func (g *GoogleGeolocationProvider) GetNearbyPlaces(ctx context.Context, center providers.Coordinates, radiusKm float64, placeType string) ([]*providers.Place, error) {
	return nil, fmt.Errorf("nearby places lookup not implemented for google provider")
}

func (g *GoogleGeolocationProvider) doGeocodeRequest(ctx context.Context, params url.Values) (*googleGeocodeResponse, error) {
	if g.apiKey == "" {
		return nil, fmt.Errorf("google maps api key is required")
	}

	params.Set("key", g.apiKey)
	reqURL := fmt.Sprintf("%s?%s", g.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build geocode request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("geocode request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("geocode request returned status %d", resp.StatusCode)
	}

	var payload googleGeocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode geocode response: %w", err)
	}

	if payload.Status != "OK" {
		if payload.ErrorMessage != "" {
			return nil, fmt.Errorf("geocode request failed: %s - %s", payload.Status, payload.ErrorMessage)
		}
		return nil, fmt.Errorf("geocode request failed: %s", payload.Status)
	}

	return &payload, nil
}

func (g *GoogleGeolocationProvider) searchPlaceAddress(ctx context.Context, query string) (*providers.GeocodedAddress, error) {
	resp, err := g.doPlacesTextSearch(ctx, query)
	if err != nil {
		return nil, err
	}
	if resp.Status == "ZERO_RESULTS" || len(resp.Results) == 0 {
		return nil, nil
	}
	if resp.Status != "OK" {
		if resp.ErrorMessage != "" {
			return nil, fmt.Errorf("places text search failed: %s - %s", resp.Status, resp.ErrorMessage)
		}
		return nil, fmt.Errorf("places text search failed: %s", resp.Status)
	}

	result := resp.Results[0]
	// Places API Text Search returns FormattedAddress.
	return &providers.GeocodedAddress{
		FormattedAddress: result.FormattedAddress,
		Coordinates: providers.Coordinates{
			Latitude:  result.Geometry.Location.Lat,
			Longitude: result.Geometry.Location.Lng,
		},
	}, nil
}

func (g *GoogleGeolocationProvider) doPlacesTextSearch(ctx context.Context, query string) (*googlePlacesTextSearchResponse, error) {
	if g.apiKey == "" {
		return nil, fmt.Errorf("google maps api key is required")
	}
	if strings.TrimSpace(g.placesURL) == "" {
		return nil, fmt.Errorf("places url is not configured")
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("region", "ng")
	params.Set("key", g.apiKey)

	reqURL := fmt.Sprintf("%s?%s", g.placesURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build places text search request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("places text search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("places text search returned status %d", resp.StatusCode)
	}

	var payload googlePlacesTextSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode places text search response: %w", err)
	}

	return &payload, nil
}

func hashKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func component(components []googleAddressComponent, primary string, fallback ...string) string {
	for _, comp := range components {
		if containsType(comp.Types, primary) {
			return comp.LongName
		}
	}
	for _, alt := range fallback {
		for _, comp := range components {
			if containsType(comp.Types, alt) {
				return comp.LongName
			}
		}
	}
	return ""
}

func buildStreet(components []googleAddressComponent) string {
	streetNumber := component(components, "street_number")
	route := component(components, "route")
	if streetNumber != "" && route != "" {
		return streetNumber + " " + route
	}
	if route != "" {
		return route
	}
	return streetNumber
}

func containsType(types []string, target string) bool {
	for _, t := range types {
		if t == target {
			return true
		}
	}
	return false
}

type googleGeocodeResponse struct {
	Status       string                `json:"status"`
	ErrorMessage string                `json:"error_message,omitempty"`
	Results      []googleGeocodeResult `json:"results"`
}

type googleGeocodeResult struct {
	FormattedAddress  string                   `json:"formatted_address"`
	AddressComponents []googleAddressComponent `json:"address_components"`
	Geometry          googleGeometry           `json:"geometry"`
}

type googleAddressComponent struct {
	LongName string   `json:"long_name"`
	Types    []string `json:"types"`
}

type googleGeometry struct {
	Location googleLocation `json:"location"`
}

type googleLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type googlePlacesTextSearchResponse struct {
	Status       string                         `json:"status"`
	ErrorMessage string                         `json:"error_message,omitempty"`
	Results      []googlePlacesTextSearchResult `json:"results"`
}

type googlePlacesTextSearchResult struct {
	FormattedAddress string         `json:"formatted_address"`
	PlaceID          string         `json:"place_id"`
	Name             string         `json:"name"`
	Geometry         googleGeometry `json:"geometry"`
}
