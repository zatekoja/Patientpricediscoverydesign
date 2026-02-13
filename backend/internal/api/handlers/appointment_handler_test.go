package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// MockAppointmentService defines the mock service
type MockAppointmentService struct {
	mock.Mock
}

func (m *MockAppointmentService) BookAppointment(ctx context.Context, appointment *entities.Appointment) error {
	args := m.Called(ctx, appointment)
	return args.Error(0)
}

func (m *MockAppointmentService) GetAvailableSlots(ctx context.Context, facilityID string, from, to time.Time) ([]entities.AvailabilitySlot, error) {
	args := m.Called(ctx, facilityID, from, to)
	return args.Get(0).([]entities.AvailabilitySlot), args.Error(1)
}

// NOTE: We need to define the Service interface in the handler package or import it
// Since Go doesn't strict require interface implementation for mocks if we use duck typing or interface definition
// But for type safety, let's assume the handler accepts an interface.

func TestAppointmentHandler_BookAppointment(t *testing.T) {
	t.Run("successfully books appointment", func(t *testing.T) {
		mockService := new(MockAppointmentService)
		handler := handlers.NewAppointmentHandler(mockService)

		payload := map[string]interface{}{
			"facility_id":   "fac-1",
			"procedure_id":  "proc-1",
			"scheduled_at":  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"patient_name":  "John Doe",
			"patient_email": "john@example.com",
			"patient_phone": "555-1234",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/appointments", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		// Expectation
		mockService.On("BookAppointment", mock.Anything, mock.MatchedBy(func(a *entities.Appointment) bool {
			return a.FacilityID == "fac-1" && a.PatientName == "John Doe"
		})).Return(nil)

		// Act
		handler.BookAppointment(w, req)

		// Assert
		assert.Equal(t, http.StatusCreated, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("returns bad request for invalid payload", func(t *testing.T) {
		mockService := new(MockAppointmentService)
		handler := handlers.NewAppointmentHandler(mockService)

		req := httptest.NewRequest("POST", "/api/appointments", bytes.NewBufferString("invalid-json"))
		w := httptest.NewRecorder()

		handler.BookAppointment(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns internal error on service failure", func(t *testing.T) {
		mockService := new(MockAppointmentService)
		handler := handlers.NewAppointmentHandler(mockService)

		payload := map[string]interface{}{
			"facility_id": "fac-1",
			// ... required fields
			"scheduled_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/appointments", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		mockService.On("BookAppointment", mock.Anything, mock.Anything).Return(errors.New("service error"))

		handler.BookAppointment(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAppointmentHandler_GetAvailability(t *testing.T) {
	t.Run("successfully gets availability", func(t *testing.T) {
		mockService := new(MockAppointmentService)
		handler := handlers.NewAppointmentHandler(mockService)

		req := httptest.NewRequest("GET", "/api/facilities/fac-1/availability?from=2025-01-01T00:00:00Z&to=2025-01-02T00:00:00Z", nil)
		// Manually set PathValue for ServeMux behavior in test
		req.SetPathValue("id", "fac-1")

		w := httptest.NewRecorder()

		slots := []entities.AvailabilitySlot{
			{StartTime: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), IsBooked: false},
		}

		mockService.On("GetAvailableSlots", mock.Anything, "fac-1", mock.Anything, mock.Anything).Return(slots, nil)

		handler.GetAvailability(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
}
