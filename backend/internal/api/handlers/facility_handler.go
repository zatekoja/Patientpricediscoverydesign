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
	SearchResultsWithCount(ctx context.Context, params repositories.SearchParams) ([]entities.FacilitySearchResult, int, error)
	Suggest(ctx context.Context, query string, lat, lon float64, limit int) ([]*entities.Facility, error)
	Update(ctx context.Context, facility *entities.Facility) error
	UpdateServiceAvailability(ctx context.Context, facilityID, procedureID string, isAvailable bool) (*entities.FacilityProcedure, error)
}

// FacilityProcedureService defines the facility procedure operations
type FacilityProcedureService interface {
	ListByFacilityWithCount(ctx context.Context, facilityID string, filter repositories.FacilityProcedureFilter) ([]*entities.FacilityProcedure, int, error)
	GetByID(ctx context.Context, id string) (*entities.FacilityProcedure, error)
}

// FacilityHandler handles facility-related HTTP requests
type FacilityHandler struct {
	service                  FacilityService
	facilityProcedureService FacilityProcedureService
}

// NewFacilityHandler creates a new facility handler
func NewFacilityHandler(service FacilityService) *FacilityHandler {
	return &FacilityHandler{
		service: service,
	}
}

// NewFacilityHandlerWithServices creates a new facility handler with procedure service
func NewFacilityHandlerWithServices(service FacilityService, procedureService FacilityProcedureService) *FacilityHandler {
	return &FacilityHandler{
		service:                  service,
		facilityProcedureService: procedureService,
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

// UpdateFacility handles PATCH /api/facilities/:id
// Updates facility real-time data (capacity, wait times, urgent care availability)
func (h *FacilityHandler) UpdateFacility(w http.ResponseWriter, r *http.Request) {
	// Extract facility ID from URL path
	facilityID := r.PathValue("id")
	if facilityID == "" {
		respondWithError(w, http.StatusBadRequest, "facility ID is required")
		return
	}

	// Get existing facility
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

	// Parse update request
	var updateReq struct {
		CapacityStatus      *string `json:"capacity_status,omitempty"`
		AvgWaitMinutes      *int    `json:"avg_wait_minutes,omitempty"`
		UrgentCareAvailable *bool   `json:"urgent_care_available,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Apply updates
	updated := false
	if updateReq.CapacityStatus != nil {
		facility.CapacityStatus = updateReq.CapacityStatus
		updated = true
	}
	if updateReq.AvgWaitMinutes != nil {
		facility.AvgWaitMinutes = updateReq.AvgWaitMinutes
		updated = true
	}
	if updateReq.UrgentCareAvailable != nil {
		facility.UrgentCareAvailable = updateReq.UrgentCareAvailable
		updated = true
	}

	if !updated {
		respondWithError(w, http.StatusBadRequest, "no valid fields to update")
		return
	}

	// Update facility
	if err := h.service.Update(r.Context(), facility); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to update facility")
		return
	}

	respondWithJSON(w, http.StatusOK, facility)
}

// GetFacilityServices handles GET /api/facilities/:id/services
// Implements TDD-driven search that operates on ENTIRE dataset before pagination
func (h *FacilityHandler) GetFacilityServices(w http.ResponseWriter, r *http.Request) {
	facilityID := r.PathValue("id")
	if facilityID == "" {
		respondWithError(w, http.StatusBadRequest, "facility ID is required")
		return
	}

	if h.facilityProcedureService == nil {
		respondWithError(w, http.StatusInternalServerError, "facility procedure service not configured")
		return
	}

	// Parse query parameters for filtering and pagination
	query := r.URL.Query()
	filter := repositories.FacilityProcedureFilter{
		SearchQuery: strings.TrimSpace(query.Get("search")),
		Category:    strings.TrimSpace(query.Get("category")),
		SortBy:      query.Get("sort"),
		SortOrder:   query.Get("order"),
		Limit:       parseIntDefault(query.Get("limit"), 20),
		Offset:      parseIntDefault(query.Get("offset"), 0),
	}

	// Parse availability filter
	if availStr := query.Get("available"); availStr != "" {
		if avail, err := strconv.ParseBool(availStr); err == nil {
			filter.IsAvailable = &avail
		}
	}

	// Parse price filters
	if minPriceStr := query.Get("min_price"); minPriceStr != "" {
		if val, err := strconv.ParseFloat(minPriceStr, 64); err == nil && val >= 0 {
			filter.MinPrice = &val
		}
	}
	if maxPriceStr := query.Get("max_price"); maxPriceStr != "" {
		if val, err := strconv.ParseFloat(maxPriceStr, 64); err == nil && val >= 0 {
			filter.MaxPrice = &val
		}
	}

	// Validate pagination parameters
	if filter.Limit <= 0 || filter.Limit > 100 {
		respondWithError(w, http.StatusBadRequest, "limit must be between 1 and 100")
		return
	}
	if filter.Offset < 0 {
		respondWithError(w, http.StatusBadRequest, "offset must be non-negative")
		return
	}

	// Call service with enhanced filter - this searches ALL data first, then paginates
	services, totalCount, err := h.facilityProcedureService.ListByFacilityWithCount(r.Context(), facilityID, filter)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to fetch facility services")
		return
	}

	// Calculate pagination metadata
	currentPage := (filter.Offset / filter.Limit) + 1
	totalPages := 0
	if filter.Limit > 0 {
		totalPages = (totalCount + filter.Limit - 1) / filter.Limit
	}

	response := map[string]interface{}{
		"services":     services,
		"total_count":  totalCount,
		"current_page": currentPage,
		"total_pages":  totalPages,
		"page_size":    filter.Limit,
		"has_next":     filter.Offset+filter.Limit < totalCount,
		"has_prev":     filter.Offset > 0,
		"filters_applied": map[string]interface{}{
			"search":     filter.SearchQuery,
			"category":   filter.Category,
			"min_price":  filter.MinPrice,
			"max_price":  filter.MaxPrice,
			"available":  filter.IsAvailable,
			"sort_by":    filter.SortBy,
			"sort_order": filter.SortOrder,
		},
	}

	respondWithJSON(w, http.StatusOK, response)
}

// UpdateServiceAvailability handles PATCH /api/facilities/:id/services/:procedureId
func (h *FacilityHandler) UpdateServiceAvailability(w http.ResponseWriter, r *http.Request) {
	facilityID := r.PathValue("id")
	if facilityID == "" {
		respondWithError(w, http.StatusBadRequest, "facility ID is required")
		return
	}

	procedureID := r.PathValue("procedureId")
	if procedureID == "" {
		respondWithError(w, http.StatusBadRequest, "procedure ID is required")
		return
	}

	var updateReq struct {
		IsAvailable *bool `json:"is_available,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if updateReq.IsAvailable == nil {
		respondWithError(w, http.StatusBadRequest, "is_available is required")
		return
	}

	fp, err := h.service.UpdateServiceAvailability(r.Context(), facilityID, procedureID, *updateReq.IsAvailable)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			switch appErr.Type {
			case apperrors.ErrorTypeNotFound:
				respondWithError(w, http.StatusNotFound, appErr.Message)
				return
			case apperrors.ErrorTypeValidation:
				respondWithError(w, http.StatusBadRequest, appErr.Message)
				return
			default:
				respondWithError(w, http.StatusInternalServerError, "failed to update service availability")
				return
			}
		}
		respondWithError(w, http.StatusInternalServerError, "failed to update service availability")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"facility_id":  facilityID,
		"procedure_id": procedureID,
		"is_available": fp.IsAvailable,
		"updated_at":   fp.UpdatedAt,
	})
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
	facilities, totalCount, err := h.service.SearchResultsWithCount(r.Context(), params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to search facilities")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"facilities": facilities,
		"count":      totalCount,
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
		suggestions = append(suggestions, FacilitySuggestionFromSearchResult(result, query))
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"suggestions": suggestions,
		"count":       len(suggestions),
	})
}

// FacilitySuggestion is a lightweight suggestion payload.
type FacilitySuggestion struct {
	ID                  string                       `json:"id"`
	Name                string                       `json:"name"`
	FacilityType        string                       `json:"facility_type"`
	Address             entities.Address             `json:"address"`
	Location            entities.Location            `json:"location"`
	Rating              float64                      `json:"rating"`
	Price               *entities.FacilityPriceRange `json:"price,omitempty"`
	ServicePrices       []entities.ServicePrice      `json:"service_prices,omitempty"`
	MatchedServicePrice *entities.ServicePrice       `json:"matched_service_price,omitempty"`
	Tags                []string                     `json:"tags,omitempty"`
	MatchedTag          string                       `json:"matched_tag,omitempty"`
}

// FacilitySuggestionFromSearchResult maps a search result to a suggestion.
func FacilitySuggestionFromSearchResult(result entities.FacilitySearchResult, query string) FacilitySuggestion {
	suggestion := FacilitySuggestion{
		ID:            result.ID,
		Name:          result.Name,
		FacilityType:  result.FacilityType,
		Address:       result.Address,
		Location:      result.Location,
		Rating:        result.Rating,
		Price:         result.Price,
		ServicePrices: trimServicePrices(result.ServicePrices, 3),
		Tags:          trimTags(result.Tags, 5),
	}

	if matched := matchServicePrice(query, result.ServicePrices); matched != nil {
		suggestion.MatchedServicePrice = matched
	}
	if matched := matchTag(query, result.Tags); matched != "" {
		suggestion.MatchedTag = matched
	}

	return suggestion
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

func trimTags(items []string, limit int) []string {
	if limit <= 0 || len(items) == 0 {
		return nil
	}
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func matchServicePrice(query string, items []entities.ServicePrice) *entities.ServicePrice {
	trimmed := strings.ToLower(strings.TrimSpace(query))
	if trimmed == "" || len(items) == 0 {
		return nil
	}

	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), trimmed) {
			matched := item
			return &matched
		}
	}

	return nil
}

func matchTag(query string, tags []string) string {
	trimmed := strings.ToLower(strings.TrimSpace(query))
	if trimmed == "" || len(tags) == 0 {
		return ""
	}
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), trimmed) {
			return tag
		}
	}
	return ""
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

func parseIntDefault(str string, defaultVal int) int {
	if str == "" {
		return defaultVal
	}
	if val, err := strconv.Atoi(str); err == nil && val >= 0 {
		return val
	}
	return defaultVal
}
