package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// FeeWaiverHandler handles fee waiver endpoints
type FeeWaiverHandler struct {
	repo repositories.FeeWaiverRepository
}

// NewFeeWaiverHandler creates a new fee waiver handler
func NewFeeWaiverHandler(repo repositories.FeeWaiverRepository) *FeeWaiverHandler {
	return &FeeWaiverHandler{repo: repo}
}

// GetFacilityFeeWaiver handles GET /api/facilities/{id}/fee-waiver
func (h *FeeWaiverHandler) GetFacilityFeeWaiver(w http.ResponseWriter, r *http.Request) {
	facilityID := r.PathValue("id")
	if facilityID == "" {
		respondWithError(w, http.StatusBadRequest, "facility ID is required")
		return
	}

	waiver, err := h.repo.GetActiveFacilityWaiver(r.Context(), facilityID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to check fee waivers")
		return
	}

	if waiver == nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"has_waiver": false,
		})
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"has_waiver":    true,
		"sponsor_name":  waiver.SponsorName,
		"waiver_type":   waiver.WaiverType,
		"waiver_amount": waiver.WaiverAmount,
	})
}

// CreateFeeWaiver handles POST /api/admin/fee-waivers
func (h *FeeWaiverHandler) CreateFeeWaiver(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SponsorName    string   `json:"sponsor_name"`
		SponsorContact string   `json:"sponsor_contact"`
		FacilityID     *string  `json:"facility_id"`
		WaiverType     string   `json:"waiver_type"`
		WaiverAmount   *float64 `json:"waiver_amount"`
		MaxUses        *int     `json:"max_uses"`
		ValidUntil     *string  `json:"valid_until"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SponsorName == "" {
		respondWithError(w, http.StatusBadRequest, "sponsor_name is required")
		return
	}

	waiverType := req.WaiverType
	if waiverType == "" {
		waiverType = "full"
	}
	if waiverType != "full" && waiverType != "partial" {
		respondWithError(w, http.StatusBadRequest, "waiver_type must be 'full' or 'partial'")
		return
	}

	if waiverType == "partial" && req.WaiverAmount == nil {
		respondWithError(w, http.StatusBadRequest, "waiver_amount is required for partial waivers")
		return
	}

	now := time.Now()
	waiver := &entities.FeeWaiver{
		ID:             generateID(),
		SponsorName:    req.SponsorName,
		SponsorContact: req.SponsorContact,
		FacilityID:     req.FacilityID,
		WaiverType:     waiverType,
		WaiverAmount:   req.WaiverAmount,
		MaxUses:        req.MaxUses,
		CurrentUses:    0,
		ValidFrom:      now,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if req.ValidUntil != nil {
		t, err := time.Parse(time.RFC3339, *req.ValidUntil)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid valid_until format (use RFC3339)")
			return
		}
		waiver.ValidUntil = &t
	}

	if err := h.repo.Create(r.Context(), waiver); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to create fee waiver")
		return
	}

	respondWithJSON(w, http.StatusCreated, waiver)
}

func generateID() string {
	return uuid.New().String()
}
