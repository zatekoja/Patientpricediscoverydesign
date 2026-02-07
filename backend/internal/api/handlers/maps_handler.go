package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

const (
	staticMapURL          = "https://maps.googleapis.com/maps/api/staticmap"
	defaultStaticMapZoom  = "13"
	defaultStaticMapSize  = "640x360"
	defaultStaticMapScale = "1"
	staticMapCacheTTL     = 60 * 60 * 24 * 7
)

// MapsHandler handles map-related endpoints.
type MapsHandler struct {
	apiKey  string
	cache   providers.CacheProvider
	client  *http.Client
	baseURL string
}

// NewMapsHandler creates a new maps handler.
func NewMapsHandler(apiKey string, cache providers.CacheProvider) *MapsHandler {
	return NewMapsHandlerWithOptions(apiKey, cache, staticMapURL, nil)
}

// NewMapsHandlerWithOptions allows overriding base URL and HTTP client (used for tests).
func NewMapsHandlerWithOptions(apiKey string, cache providers.CacheProvider, baseURL string, client *http.Client) *MapsHandler {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = staticMapURL
	}
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	return &MapsHandler{
		apiKey:  apiKey,
		cache:   cache,
		client:  client,
		baseURL: baseURL,
	}
}

// GetStaticMap proxies Google Static Maps and caches responses.
func (h *MapsHandler) GetStaticMap(w http.ResponseWriter, r *http.Request) {
	if h.apiKey == "" {
		placeholder := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="640" height="360" viewBox="0 0 640 360">
  <defs>
    <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0%" stop-color="#f3f4f6"/>
      <stop offset="100%" stop-color="#e5e7eb"/>
    </linearGradient>
  </defs>
  <rect width="640" height="360" fill="url(#bg)"/>
  <rect x="16" y="16" width="608" height="328" rx="18" fill="#ffffff" stroke="#e5e7eb"/>
  <circle cx="320" cy="180" r="14" fill="#2563eb"/>
  <text x="320" y="220" text-anchor="middle" font-family="Arial, sans-serif" font-size="14" fill="#374151">
    Map preview unavailable (API key not configured)
  </text>
</svg>`)
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(placeholder)
		return
	}

	query := r.URL.Query()
	center := strings.TrimSpace(query.Get("center"))
	if center == "" {
		lat := strings.TrimSpace(query.Get("lat"))
		lon := strings.TrimSpace(query.Get("lon"))
		if lat == "" || lon == "" {
			respondWithError(w, http.StatusBadRequest, "center or lat/lon required")
			return
		}
		center = fmt.Sprintf("%s,%s", lat, lon)
	}

	zoom := strings.TrimSpace(query.Get("zoom"))
	if zoom == "" {
		zoom = defaultStaticMapZoom
	}
	size := strings.TrimSpace(query.Get("size"))
	if size == "" {
		size = defaultStaticMapSize
	}
	scale := strings.TrimSpace(query.Get("scale"))
	if scale == "" {
		scale = defaultStaticMapScale
	}

	markers := normalizeMarkers(query.Get("markers"))

	cacheKey := buildStaticMapCacheKey(center, zoom, size, scale, markers)
	if h.cache != nil {
		if cached, err := h.cache.Get(r.Context(), cacheKey); err == nil && len(cached) > 0 {
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(cached)
			return
		}
	}

	values := url.Values{}
	values.Set("center", center)
	values.Set("zoom", zoom)
	values.Set("size", size)
	values.Set("scale", scale)
	for _, marker := range markers {
		values.Add("markers", marker)
	}
	values.Set("key", h.apiKey)

	mapURL := fmt.Sprintf("%s?%s", h.baseURL, values.Encode())
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, mapURL, nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to build map request")
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		respondWithError(w, http.StatusBadGateway, "failed to fetch map image")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respondWithError(w, http.StatusBadGateway, "map provider returned an error")
		return
	}

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to read map image")
		return
	}

	if h.cache != nil {
		_ = h.cache.Set(r.Context(), cacheKey, imageBytes, staticMapCacheTTL)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(imageBytes)
}

func normalizeMarkers(markersParam string) []string {
	if strings.TrimSpace(markersParam) == "" {
		return nil
	}

	raw := strings.Split(markersParam, "|")
	clean := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		clean = append(clean, item)
	}
	return clean
}

func buildStaticMapCacheKey(center, zoom, size, scale string, markers []string) string {
	values := url.Values{}
	values.Set("center", center)
	values.Set("zoom", zoom)
	values.Set("size", size)
	values.Set("scale", scale)
	for _, marker := range markers {
		values.Add("markers", marker)
	}
	return "maps:static:" + hashString(values.Encode())
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
