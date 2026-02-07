package handlers

import (
	"net/http"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// ProcedureHandler handles procedure-related requests
type ProcedureHandler struct {
	repo repositories.ProcedureRepository
}

// NewProcedureHandler creates a new procedure handler
func NewProcedureHandler(repo repositories.ProcedureRepository) *ProcedureHandler {
	return &ProcedureHandler{repo: repo}
}

// ListProcedures handles GET /api/procedures
func (h *ProcedureHandler) ListProcedures(w http.ResponseWriter, r *http.Request) {
	filter := repositories.ProcedureFilter{
		Category: r.URL.Query().Get("category"),
	}

	procedures, err := h.repo.List(r.Context(), filter)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to list procedures")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"procedures": procedures,
		"count":      len(procedures),
	})
}

// GetProcedure handles GET /api/procedures/{id}
func (h *ProcedureHandler) GetProcedure(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "procedure ID is required")
		return
	}

	procedure, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "procedure not found")
		return
	}

	respondWithJSON(w, http.StatusOK, procedure)
}
