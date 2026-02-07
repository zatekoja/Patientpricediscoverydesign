package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/providerapi"
)

// ProviderPriceHandler proxies provider API data through the core REST API.
type ProviderPriceHandler struct {
	client providerapi.Client
}

func NewProviderPriceHandler(client providerapi.Client) *ProviderPriceHandler {
	return &ProviderPriceHandler{client: client}
}

func (h *ProviderPriceHandler) ensureClient(w http.ResponseWriter) bool {
	if h.client == nil {
		respondWithError(w, http.StatusServiceUnavailable, "provider api client not configured")
		return false
	}
	return true
}

// GetCurrentData handles GET /api/provider/prices/current
func (h *ProviderPriceHandler) GetCurrentData(w http.ResponseWriter, r *http.Request) {
	if !h.ensureClient(w) {
		return
	}

	req, ok := buildProviderDataRequest(r, 100)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "invalid limit or offset parameter")
		return
	}

	resp, err := h.client.GetCurrentData(r.Context(), req)
	if err != nil {
		respondWithError(w, http.StatusBadGateway, "failed to fetch provider data")
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}

// GetPreviousData handles GET /api/provider/prices/previous
func (h *ProviderPriceHandler) GetPreviousData(w http.ResponseWriter, r *http.Request) {
	if !h.ensureClient(w) {
		return
	}

	req, ok := buildProviderDataRequest(r, 100)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "invalid limit or offset parameter")
		return
	}

	resp, err := h.client.GetPreviousData(r.Context(), req)
	if err != nil {
		respondWithError(w, http.StatusBadGateway, "failed to fetch provider data")
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}

// GetHistoricalData handles GET /api/provider/prices/historical
func (h *ProviderPriceHandler) GetHistoricalData(w http.ResponseWriter, r *http.Request) {
	if !h.ensureClient(w) {
		return
	}

	req, ok := buildProviderHistoricalRequest(r)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "invalid historical query parameters")
		return
	}

	resp, err := h.client.GetHistoricalData(r.Context(), req)
	if err != nil {
		respondWithError(w, http.StatusBadGateway, "failed to fetch provider data")
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}

// GetProviderHealth handles GET /api/provider/health
func (h *ProviderPriceHandler) GetProviderHealth(w http.ResponseWriter, r *http.Request) {
	if !h.ensureClient(w) {
		return
	}

	providerID := strings.TrimSpace(r.URL.Query().Get("providerId"))
	resp, err := h.client.GetProviderHealth(r.Context(), providerID)
	if err != nil {
		respondWithError(w, http.StatusBadGateway, "failed to fetch provider health")
		return
	}
	respondWithJSON(w, http.StatusOK, resp)
}

// ListProviders handles GET /api/provider/list
func (h *ProviderPriceHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	if !h.ensureClient(w) {
		return
	}

	resp, err := h.client.ListProviders(r.Context())
	if err != nil {
		respondWithError(w, http.StatusBadGateway, "failed to list providers")
		return
	}
	respondWithJSON(w, http.StatusOK, resp)
}

// TriggerSync handles POST /api/provider/sync/trigger
func (h *ProviderPriceHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	if !h.ensureClient(w) {
		return
	}

	providerID := strings.TrimSpace(r.URL.Query().Get("providerId"))
	resp, err := h.client.TriggerSync(r.Context(), providerID)
	if err != nil {
		respondWithError(w, http.StatusBadGateway, "failed to trigger provider sync")
		return
	}
	respondWithJSON(w, http.StatusOK, resp)
}

// GetSyncStatus handles GET /api/provider/sync/status
func (h *ProviderPriceHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	if !h.ensureClient(w) {
		return
	}

	providerID := strings.TrimSpace(r.URL.Query().Get("providerId"))
	resp, err := h.client.GetSyncStatus(r.Context(), providerID)
	if err != nil {
		respondWithError(w, http.StatusBadGateway, "failed to fetch provider sync status")
		return
	}
	respondWithJSON(w, http.StatusOK, resp)
}

func buildProviderDataRequest(r *http.Request, defaultLimit int) (providerapi.CurrentDataRequest, bool) {
	query := r.URL.Query()
	req := providerapi.CurrentDataRequest{
		ProviderID: strings.TrimSpace(query.Get("providerId")),
		Limit:      defaultLimit,
		Offset:     0,
	}

	if limitStr := strings.TrimSpace(query.Get("limit")); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			return providerapi.CurrentDataRequest{}, false
		}
		req.Limit = limit
	}

	if offsetStr := strings.TrimSpace(query.Get("offset")); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return providerapi.CurrentDataRequest{}, false
		}
		req.Offset = offset
	}

	return req, true
}

func buildProviderHistoricalRequest(r *http.Request) (providerapi.HistoricalDataRequest, bool) {
	query := r.URL.Query()
	req := providerapi.HistoricalDataRequest{
		ProviderID: strings.TrimSpace(query.Get("providerId")),
	}

	if timeWindow := strings.TrimSpace(query.Get("timeWindow")); timeWindow != "" {
		req.TimeWindow = timeWindow
	}

	if start := strings.TrimSpace(query.Get("startDate")); start != "" {
		parsed, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return providerapi.HistoricalDataRequest{}, false
		}
		req.StartDate = &parsed
	}

	if end := strings.TrimSpace(query.Get("endDate")); end != "" {
		parsed, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return providerapi.HistoricalDataRequest{}, false
		}
		req.EndDate = &parsed
	}

	if req.TimeWindow == "" && (req.StartDate == nil || req.EndDate == nil) {
		return providerapi.HistoricalDataRequest{}, false
	}

	if limitStr := strings.TrimSpace(query.Get("limit")); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			return providerapi.HistoricalDataRequest{}, false
		}
		req.Limit = limit
	}

	if offsetStr := strings.TrimSpace(query.Get("offset")); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return providerapi.HistoricalDataRequest{}, false
		}
		req.Offset = offset
	}

	return req, true
}
