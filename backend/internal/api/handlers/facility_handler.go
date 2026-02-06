package handlers

import (
	"encoding/json"
	"net/http"

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
		if appErr, ok := err.(*apperrors.AppError); ok {
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
	// In production, properly parse and validate these parameters
	params := repositories.SearchParams{
		Latitude:  37.7749,  // Would come from query params
		Longitude: -122.4194, // Would come from query params
		RadiusKm:  10.0,     // Would come from query params
		Limit:     30,
		Offset:    0,
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
