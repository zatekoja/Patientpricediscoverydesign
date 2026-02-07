package handlers

import (
	"net/http"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// InsuranceHandler handles insurance-related requests
type InsuranceHandler struct {
	repo repositories.InsuranceRepository
}

// NewInsuranceHandler creates a new insurance handler
func NewInsuranceHandler(repo repositories.InsuranceRepository) *InsuranceHandler {
	return &InsuranceHandler{repo: repo}
}

// ListInsuranceProviders handles GET /api/insurance-providers
func (h *InsuranceHandler) ListInsuranceProviders(w http.ResponseWriter, r *http.Request) {
	filter := repositories.InsuranceFilter{}

	providers, err := h.repo.List(r.Context(), filter)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to list insurance providers")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"providers": providers,
		"count":     len(providers),
	})
}

// GetInsuranceProvider handles GET /api/insurance-providers/{id}
func (h *InsuranceHandler) GetInsuranceProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "insurance provider ID is required")
		return
	}

	provider, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "insurance provider not found")
		return
	}

	respondWithJSON(w, http.StatusOK, provider)
}
