package handlers

import (
	"context"
	"net/http"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
)

// ProcedureEnrichmentService defines the handler dependency for enrichment.
type ProcedureEnrichmentService interface {
	GetEnrichment(ctx context.Context, procedureID string, refresh bool) (*entities.ProcedureEnrichment, error)
}

// ProcedureHandler handles procedure-related requests
type ProcedureHandler struct {
	repo              repositories.ProcedureRepository
	enrichmentService ProcedureEnrichmentService
}

// NewProcedureHandler creates a new procedure handler
func NewProcedureHandler(repo repositories.ProcedureRepository, enrichmentService ProcedureEnrichmentService) *ProcedureHandler {
	return &ProcedureHandler{
		repo:              repo,
		enrichmentService: enrichmentService,
	}
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

// GetProcedureEnrichment handles GET /api/procedures/{id}/enrichment
func (h *ProcedureHandler) GetProcedureEnrichment(w http.ResponseWriter, r *http.Request) {
	if h.enrichmentService == nil {
		respondWithError(w, http.StatusServiceUnavailable, "procedure enrichment is not configured")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "procedure ID is required")
		return
	}

	refresh := r.URL.Query().Get("refresh") == "true"
	enrichment, err := h.enrichmentService.GetEnrichment(r.Context(), id, refresh)
	if err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			switch appErr.Type {
			case apperrors.ErrorTypeNotFound:
				respondWithError(w, http.StatusNotFound, "procedure not found")
				return
			case apperrors.ErrorTypeValidation:
				respondWithError(w, http.StatusBadRequest, appErr.Message)
				return
			case apperrors.ErrorTypeExternal:
				respondWithError(w, http.StatusServiceUnavailable, "procedure enrichment unavailable")
				return
			default:
				respondWithError(w, http.StatusInternalServerError, "failed to enrich procedure")
				return
			}
		}
		respondWithError(w, http.StatusInternalServerError, "failed to enrich procedure")
		return
	}

	respondWithJSON(w, http.StatusOK, enrichment)
}
