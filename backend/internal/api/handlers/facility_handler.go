package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
)

// FacilityService defines the facility operations used by the handler.
type FacilityService interface {
	GetByID(ctx context.Context, id string) (*entities.Facility, error)
	List(ctx context.Context, filter repositories.FacilityFilter) ([]*entities.Facility, error)
	Search(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, error)
	SearchResults(ctx context.Context, params repositories.SearchParams) ([]entities.FacilitySearchResult, error)
	Suggest(ctx context.Context, query string, lat, lon float64, limit int) ([]*entities.Facility, error)
}

// FacilityHandler handles facility-related HTTP requests
type FacilityHandler struct {
	service FacilityService
}

// NewFacilityHandler creates a new facility handler
func NewFacilityHandler(service FacilityService) *FacilityHandler {
	return &FacilityHandler{
		service: service,
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
	facility, err := h.service.GetByID(r.Context(), facilityID)
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
	facilities, err := h.service.List(r.Context(), filter)
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
		Query:     strings.TrimSpace(query.Get("query")),
		Latitude:  lat,
		Longitude: lon,
		RadiusKm:  radius,
		Limit:     limit,
		Offset:    offset,
	}

	if insuranceProvider := strings.TrimSpace(query.Get("insurance_provider")); insuranceProvider != "" {
		params.InsuranceProvider = insuranceProvider
	}

	if minPriceStr := strings.TrimSpace(query.Get("min_price")); minPriceStr != "" {
		minPrice, err := strconv.ParseFloat(minPriceStr, 64)
		if err != nil || minPrice < 0 {
			respondWithError(w, http.StatusBadRequest, "invalid min_price parameter")
			return
		}
		params.MinPrice = &minPrice
	}

	if maxPriceStr := strings.TrimSpace(query.Get("max_price")); maxPriceStr != "" {
		maxPrice, err := strconv.ParseFloat(maxPriceStr, 64)
		if err != nil || maxPrice < 0 {
			respondWithError(w, http.StatusBadRequest, "invalid max_price parameter")
			return
		}
		params.MaxPrice = &maxPrice
	}

	// Search facilities
	facilities, err := h.service.SearchResults(r.Context(), params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to search facilities")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"facilities": facilities,
		"count":      len(facilities),
	})
}

// SuggestFacilities handles GET /api/facilities/suggest
func (h *FacilityHandler) SuggestFacilities(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"suggestions": []FacilitySuggestion{},
			"count":       0,
		})
		return
	}

	lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid latitude parameter")
		return
	}

	lon, err := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid longitude parameter")
		return
	}

	limit := 6
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 || parsedLimit > 20 {
			respondWithError(w, http.StatusBadRequest, "invalid limit parameter (must be 1-20)")
			return
		}
		limit = parsedLimit
	}

	params := repositories.SearchParams{
		Query:     query,
		Latitude:  lat,
		Longitude: lon,
		RadiusKm:  50,
		Limit:     limit,
		Offset:    0,
	}

	results, err := h.service.SearchResults(r.Context(), params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to fetch suggestions")
		return
	}

	suggestions := make([]FacilitySuggestion, 0, len(results))
	for _, result := range results {
		suggestions = append(suggestions, FacilitySuggestionFromSearchResult(result))
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"suggestions": suggestions,
		"count":       len(suggestions),
	})
}

// FacilitySuggestion is a lightweight suggestion payload.
type FacilitySuggestion struct {
	ID            string                       `json:"id"`
	Name          string                       `json:"name"`
	FacilityType  string                       `json:"facility_type"`
	Address       entities.Address             `json:"address"`
	Location      entities.Location            `json:"location"`
	Rating        float64                      `json:"rating"`
	Price         *entities.FacilityPriceRange `json:"price,omitempty"`
	ServicePrices []entities.ServicePrice      `json:"service_prices,omitempty"`
}

// FacilitySuggestionFromSearchResult maps a search result to a suggestion.
func FacilitySuggestionFromSearchResult(result entities.FacilitySearchResult) FacilitySuggestion {
	return FacilitySuggestion{
		ID:            result.ID,
		Name:          result.Name,
		FacilityType:  result.FacilityType,
		Address:       result.Address,
		Location:      result.Location,
		Rating:        result.Rating,
		Price:         result.Price,
		ServicePrices: trimServicePrices(result.ServicePrices, 3),
	}
}

func trimServicePrices(items []entities.ServicePrice, limit int) []entities.ServicePrice {
	if limit <= 0 || len(items) == 0 {
		return nil
	}
	if len(items) <= limit {
		return items
	}
	return items[:limit]
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
