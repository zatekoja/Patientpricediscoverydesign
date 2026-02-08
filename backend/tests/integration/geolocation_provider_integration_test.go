//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/cache"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/providers/geolocation"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
)

func TestGoogleProviderAndStaticMapCaching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	redisClient := maybeTestRedisClient(t)
	if redisClient == nil {
		t.Skip("Redis not available for integration test")
	}
	defer redisClient.Close()

	ctx := context.Background()
	require.NoError(t, redisClient.Client().FlushDB(ctx).Err())

	var geocodeCalls int32
	var mapCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/geocode":
			atomic.AddInt32(&geocodeCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
  "status": "OK",
  "results": [{
    "formatted_address": "Lagos, Nigeria",
    "address_components": [
      {"long_name": "Lagos", "types": ["locality"]},
      {"long_name": "Lagos", "types": ["administrative_area_level_1"]},
      {"long_name": "Nigeria", "types": ["country"]}
    ],
    "geometry": { "location": { "lat": 6.5244, "lng": 3.3792 } }
  }]
}`))
		case "/staticmap":
			atomic.AddInt32(&mapCalls, 1)
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("PNGDATA"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cacheProvider := cache.NewRedisAdapter(redisClient)
	provider := geolocation.NewGoogleGeolocationProviderWithOptions(
		"test-key",
		cacheProvider,
		server.URL+"/geocode",
		server.Client(),
	)

	coords, err := provider.Geocode(ctx, "Lagos, Nigeria")
	require.NoError(t, err)
	require.Equal(t, 6.5244, coords.Latitude)
	require.Equal(t, 3.3792, coords.Longitude)

	coords, err = provider.Geocode(ctx, "Lagos, Nigeria")
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&geocodeCalls))

	mapsHandler := handlers.NewMapsHandlerWithOptions(
		"test-key",
		cacheProvider,
		server.URL+"/staticmap",
		server.Client(),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/maps/static?center=6.5244,3.3792&zoom=12&size=640x360", nil)
	rr := httptest.NewRecorder()
	mapsHandler.GetStaticMap(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "image/png", rr.Header().Get("Content-Type"))
	require.Equal(t, "PNGDATA", rr.Body.String())

	req = httptest.NewRequest(http.MethodGet, "/api/maps/static?center=6.5244,3.3792&zoom=12&size=640x360", nil)
	rr = httptest.NewRecorder()
	mapsHandler.GetStaticMap(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, int32(1), atomic.LoadInt32(&mapCalls))
}

func TestGoogleProviderPlacesFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	var geocodeCalls int32
	var placesCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/geocode":
			atomic.AddInt32(&geocodeCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
  "status": "ZERO_RESULTS",
  "results": []
}`))
		case "/place/textsearch":
			atomic.AddInt32(&placesCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
  "status": "OK",
  "results": [{
    "formatted_address": "Badagry, Lagos, Nigeria",
    "name": "General Hospital Badagry",
    "place_id": "test-place-id",
    "geometry": { "location": { "lat": 6.4152, "lng": 2.8866 } }
  }]
}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := geolocation.NewGoogleGeolocationProviderWithOptions(
		"test-key",
		nil,
		server.URL+"/geocode",
		server.Client(),
	)

	coords, err := provider.Geocode(context.Background(), "General Hospital Badagry, Nigeria")
	require.NoError(t, err)
	require.NotNil(t, coords)
	require.Equal(t, 6.4152, coords.Latitude)
	require.Equal(t, 2.8866, coords.Longitude)
	require.Equal(t, int32(0), atomic.LoadInt32(&geocodeCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&placesCalls))
}
