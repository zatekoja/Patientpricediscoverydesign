package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
)

// FacilityHandler handles facility-related HTTP requests
type FacilityHandler struct {
	facilityRepo repositories.FacilityRepository
}

// NewFacilityHandler creates a new facility handler
func NewFacilityHandler(facilityRepo repositories.FacilityRepository) *FacilityHandler {
	return &FacilityHandler{
		facilityRepo: facilityRepo,
	}
}

// GetFacility handles GET /api/facilities/:id
func (h *FacilityHandler) GetFacility(w http.ResponseWriter, r *http.Request) {
	// Extract facility ID from URL path
	facilityID := r.PathValue("id")
	if facilityID == "" {
		respondWithError(w, http.StatusBadRequest, "facility ID is required")
		return
	}

	// Get facility from repository
	facility, err := h.facilityRepo.GetByID(r.Context(), facilityID)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			switch appErr.Type {
			case apperrors.ErrorTypeNotFound:
				respondWithError(w, http.StatusNotFound, appErr.Message)
			default:
				respondWithError(w, http.StatusInternalServerError, "internal server error")
			}
			return
		}
		respondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respondWithJSON(w, http.StatusOK, facility)
}

// ListFacilities handles GET /api/facilities
func (h *FacilityHandler) ListFacilities(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := repositories.FacilityFilter{
		FacilityType: r.URL.Query().Get("type"),
		Limit:        30,
		Offset:       0,
	}

	// Get facilities from repository
	facilities, err := h.facilityRepo.List(r.Context(), filter)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to list facilities")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"facilities": facilities,
		"count":      len(facilities),
	})
}

// SearchFacilities handles GET /api/facilities/search
func (h *FacilityHandler) SearchFacilities(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
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

	radius := 10.0 // default radius
	if radiusStr := query.Get("radius"); radiusStr != "" {
		radius, err = strconv.ParseFloat(radiusStr, 64)
		if err != nil || radius <= 0 {
			respondWithError(w, http.StatusBadRequest, "invalid radius parameter")
			return
		}
	}

	limit := 30 // default limit
	if limitStr := query.Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 || parsedLimit > 100 {
			respondWithError(w, http.StatusBadRequest, "invalid limit parameter (must be 1-100)")
			return
		}
		limit = parsedLimit
	}

	offset := 0 // default offset
	if offsetStr := query.Get("offset"); offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			respondWithError(w, http.StatusBadRequest, "invalid offset parameter")
			return
		}
	}

	params := repositories.SearchParams{
		Latitude:  lat,
		Longitude: lon,
		RadiusKm:  radius,
		Limit:     limit,
		Offset:    offset,
	}

	// Search facilities
	facilities, err := h.facilityRepo.Search(r.Context(), params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to search facilities")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"facilities": facilities,
		"count":      len(facilities),
	})
}

// Helper functions
func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	respondWithJSON(w, statusCode, map[string]string{
		"error": message,
	})
}
