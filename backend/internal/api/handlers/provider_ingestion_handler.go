package handlers

import (
	"net/http"
	"strings"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
)

// ProviderIngestionHandler triggers provider -> core data sync.
type ProviderIngestionHandler struct {
	service *services.ProviderIngestionService
}

func NewProviderIngestionHandler(service *services.ProviderIngestionService) *ProviderIngestionHandler {
	return &ProviderIngestionHandler{service: service}
}

func (h *ProviderIngestionHandler) TriggerIngestion(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		respondWithError(w, http.StatusServiceUnavailable, "provider ingestion service not configured")
		return
	}

	providerID := strings.TrimSpace(r.URL.Query().Get("providerId"))
	summary, err := h.service.SyncCurrentData(r.Context(), providerID)
	if err != nil {
		respondWithError(w, http.StatusBadGateway, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, summary)
}
