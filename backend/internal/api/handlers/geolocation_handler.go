package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

// GeolocationHandler handles geolocation endpoints.
type GeolocationHandler struct {
	provider providers.GeolocationProvider
}

// NewGeolocationHandler creates a new geolocation handler.
func NewGeolocationHandler(provider providers.GeolocationProvider) *GeolocationHandler {
	return &GeolocationHandler{provider: provider}
}

// Geocode handles GET /api/geocode?address=...
func (h *GeolocationHandler) Geocode(w http.ResponseWriter, r *http.Request) {
	address := strings.TrimSpace(r.URL.Query().Get("address"))
	if address == "" {
		respondWithError(w, http.StatusBadRequest, "address parameter is required")
		return
	}

	coords, err := h.provider.Geocode(r.Context(), address)
	if err != nil {
		respondWithError(w, http.StatusBadGateway, "failed to geocode address")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"address": address,
		"lat":     coords.Latitude,
		"lon":     coords.Longitude,
	})
}

// ReverseGeocode handles GET /api/reverse-geocode?lat=...&lon=...
func (h *GeolocationHandler) ReverseGeocode(w http.ResponseWriter, r *http.Request) {
	latStr := strings.TrimSpace(r.URL.Query().Get("lat"))
	lonStr := strings.TrimSpace(r.URL.Query().Get("lon"))
	if latStr == "" || lonStr == "" {
		respondWithError(w, http.StatusBadRequest, "lat and lon parameters are required")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid lat parameter")
		return
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid lon parameter")
		return
	}

	address, err := h.provider.ReverseGeocode(r.Context(), lat, lon)
	if err != nil {
		log.Printf("ReverseGeocode error: %v", err)
		respondWithError(w, http.StatusBadGateway, "failed to reverse geocode")
		return
	}

	respondWithJSON(w, http.StatusOK, address)
}
