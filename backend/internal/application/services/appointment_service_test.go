package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// Mocks

type MockAppointmentRepository struct {
	mock.Mock
}

func (m *MockAppointmentRepository) Create(ctx context.Context, appointment *entities.Appointment) error {
	args := m.Called(ctx, appointment)
	return args.Error(0)
}

func (m *MockAppointmentRepository) GetByID(ctx context.Context, id string) (*entities.Appointment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Appointment), args.Error(1)
}

// ... other repository methods mocked as needed ...
func (m *MockAppointmentRepository) Update(ctx context.Context, appointment *entities.Appointment) error {
	return nil
}
func (m *MockAppointmentRepository) Cancel(ctx context.Context, id string) error {
	return nil
}
func (m *MockAppointmentRepository) ListByUser(ctx context.Context, userID string, filter repositories.AppointmentFilter) ([]*entities.Appointment, error) {
	return nil, nil
}
func (m *MockAppointmentRepository) ListByFacility(ctx context.Context, facilityID string, filter repositories.AppointmentFilter) ([]*entities.Appointment, error) {
	return nil, nil
}

type MockAppointmentProvider struct {
	mock.Mock
}

func (m *MockAppointmentProvider) GetAvailableSlots(ctx context.Context, externalID string, from, to time.Time) ([]entities.AvailabilitySlot, error) {
	args := m.Called(ctx, externalID, from, to)
	return args.Get(0).([]entities.AvailabilitySlot), args.Error(1)
}

func (m *MockAppointmentProvider) CreateAppointment(ctx context.Context, appointment *entities.Appointment) (string, string, error) {
	args := m.Called(ctx, appointment)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockAppointmentProvider) CancelAppointment(ctx context.Context, externalID string, reason string) error {
	args := m.Called(ctx, externalID, reason)
	return args.Error(0)
}

// Tests

func TestAppointmentService_BookAppointment(t *testing.T) {
	t.Run("successfully books appointment", func(t *testing.T) {
		// Arrange
		repo := new(MockAppointmentRepository)
		provider := new(MockAppointmentProvider)
		service := services.NewAppointmentService(repo, provider)

		appointment := &entities.Appointment{
			FacilityID:  "facility-1",
			ProcedureID: "proc-1",
			ScheduledAt: time.Now().Add(24 * time.Hour),
			PatientName: "John Doe",
		}

		// Expectations
		// 1. Provider is called to book external slot
		provider.On("CreateAppointment", mock.Anything, appointment).Return("ext-123", "http://meet.com/123", nil)

		// 2. Repository is called to save appointment with external details
		repo.On("Create", mock.Anything, mock.MatchedBy(func(a *entities.Appointment) bool {
			return a.Status == entities.AppointmentStatusConfirmed && a.PatientName == "John Doe"
		})).Return(nil)

		// Act
		err := service.BookAppointment(context.Background(), appointment)

		// Assert
		assert.NoError(t, err)
		repo.AssertExpectations(t)
		provider.AssertExpectations(t)
	})

	t.Run("fails when provider fails", func(t *testing.T) {
		// Arrange
		repo := new(MockAppointmentRepository)
		provider := new(MockAppointmentProvider)
		service := services.NewAppointmentService(repo, provider)

		appointment := &entities.Appointment{
			FacilityID: "facility-1",
            ScheduledAt: time.Now().Add(24 * time.Hour),
		}

		provider.On("CreateAppointment", mock.Anything, appointment).Return("", "", errors.New("provider error"))

		// Act
		err := service.BookAppointment(context.Background(), appointment)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider error")
		repo.AssertNotCalled(t, "Create")
	})
}
