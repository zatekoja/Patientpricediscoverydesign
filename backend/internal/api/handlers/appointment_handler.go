package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
)

// AppointmentService defines the interface for appointment operations
type AppointmentService interface {
	BookAppointment(ctx context.Context, appointment *entities.Appointment) error
	GetAvailableSlots(ctx context.Context, facilityID string, from, to time.Time) ([]entities.AvailabilitySlot, error)
}

// AppointmentHandler handles appointment requests
type AppointmentHandler struct {
	service AppointmentService
}

// NewAppointmentHandler creates a new appointment handler
func NewAppointmentHandler(service AppointmentService) *AppointmentHandler {
	return &AppointmentHandler{
		service: service,
	}
}

// BookAppointment handles POST /api/appointments
func (h *AppointmentHandler) BookAppointment(w http.ResponseWriter, r *http.Request) {
	var appointment entities.Appointment
	if err := json.NewDecoder(r.Body).Decode(&appointment); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.service.BookAppointment(r.Context(), &appointment); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, appointment)
}

// GetAvailability handles GET /api/facilities/:id/availability
func (h *AppointmentHandler) GetAvailability(w http.ResponseWriter, r *http.Request) {
	facilityID := r.PathValue("id")
	if facilityID == "" {
		respondWithError(w, http.StatusBadRequest, "facility ID is required")
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if fromStr == "" || toStr == "" {
		respondWithError(w, http.StatusBadRequest, "from and to query parameters are required")
		return
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid from date format (use RFC3339)")
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid to date format (use RFC3339)")
		return
	}

	slots, err := h.service.GetAvailableSlots(r.Context(), facilityID, from, to)
	if err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			switch appErr.Type {
			case apperrors.ErrorTypeNotFound:
				respondWithError(w, http.StatusNotFound, appErr.Message)
				return
			case apperrors.ErrorTypeValidation:
				respondWithError(w, http.StatusBadRequest, appErr.Message)
				return
			case apperrors.ErrorTypeExternal:
				respondWithError(w, http.StatusBadGateway, appErr.Message)
				return
			default:
				respondWithError(w, http.StatusInternalServerError, appErr.Message)
				return
			}
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"slots": slots,
	})
}
