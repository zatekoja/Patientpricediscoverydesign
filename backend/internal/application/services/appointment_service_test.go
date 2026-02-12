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

type MockFacilityRepository struct {
	mock.Mock
}

func (m *MockFacilityRepository) Create(ctx context.Context, facility *entities.Facility) error {
	args := m.Called(ctx, facility)
	return args.Error(0)
}

func (m *MockFacilityRepository) GetByID(ctx context.Context, id string) (*entities.Facility, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Facility), args.Error(1)
}

func (m *MockFacilityRepository) GetByIDs(ctx context.Context, ids []string) ([]*entities.Facility, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Facility), args.Error(1)
}

func (m *MockFacilityRepository) Update(ctx context.Context, facility *entities.Facility) error {
	args := m.Called(ctx, facility)
	return args.Error(0)
}

func (m *MockFacilityRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFacilityRepository) List(ctx context.Context, filter repositories.FacilityFilter) ([]*entities.Facility, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Facility), args.Error(1)
}

func (m *MockFacilityRepository) Search(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Facility), args.Error(1)
}

type MockProcedureRepository struct {
	mock.Mock
}

func (m *MockProcedureRepository) Create(ctx context.Context, procedure *entities.Procedure) error {
	args := m.Called(ctx, procedure)
	return args.Error(0)
}

func (m *MockProcedureRepository) GetByID(ctx context.Context, id string) (*entities.Procedure, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Procedure), args.Error(1)
}

func (m *MockProcedureRepository) GetByCode(ctx context.Context, code string) (*entities.Procedure, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Procedure), args.Error(1)
}

func (m *MockProcedureRepository) GetByIDs(ctx context.Context, ids []string) ([]*entities.Procedure, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Procedure), args.Error(1)
}

func (m *MockProcedureRepository) Update(ctx context.Context, procedure *entities.Procedure) error {
	args := m.Called(ctx, procedure)
	return args.Error(0)
}

func (m *MockProcedureRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProcedureRepository) List(ctx context.Context, filter repositories.ProcedureFilter) ([]*entities.Procedure, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Procedure), args.Error(1)
}

// Tests

func TestAppointmentService_BookAppointment(t *testing.T) {
	t.Run("successfully books appointment", func(t *testing.T) {
		// Arrange
		repo := new(MockAppointmentRepository)
		facilityRepo := new(MockFacilityRepository)
		procedureRepo := new(MockProcedureRepository)
		provider := new(MockAppointmentProvider)
		service := services.NewAppointmentService(repo, facilityRepo, procedureRepo, provider, true, nil)

		appointment := &entities.Appointment{
			FacilityID:  "facility-1",
			ProcedureID: "proc-1",
			ScheduledAt: time.Now().Add(24 * time.Hour),
			PatientName: "John Doe",
		}

		facilityRepo.On("GetByID", mock.Anything, "facility-1").Return(&entities.Facility{
			ID:                   "facility-1",
			SchedulingExternalID: "consultation-30min",
		}, nil)

		// Expectations
		// 1. Provider is called to book external slot
		provider.On("CreateAppointment", mock.Anything, appointment).Return("ext-123", "http://meet.com/123", nil)

		// 2. Repository is called to save appointment with external details
		repo.On("Create", mock.Anything, mock.MatchedBy(func(a *entities.Appointment) bool {
			return a.Status == entities.AppointmentStatusPending &&
				a.PatientName == "John Doe" &&
				a.BookingMethod == entities.BookingMethodAPI
		})).Return(nil)

		// Act
		err := service.BookAppointment(context.Background(), appointment)

		// Assert
		assert.NoError(t, err)
		repo.AssertExpectations(t)
		provider.AssertExpectations(t)
		facilityRepo.AssertExpectations(t)
	})

	t.Run("fails when provider fails", func(t *testing.T) {
		// Arrange
		repo := new(MockAppointmentRepository)
		facilityRepo := new(MockFacilityRepository)
		procedureRepo := new(MockProcedureRepository)
		provider := new(MockAppointmentProvider)
		service := services.NewAppointmentService(repo, facilityRepo, procedureRepo, provider, true, nil)

		appointment := &entities.Appointment{
			FacilityID:  "facility-1",
			ScheduledAt: time.Now().Add(24 * time.Hour),
		}

		facilityRepo.On("GetByID", mock.Anything, "facility-1").Return(&entities.Facility{
			ID:                   "facility-1",
			SchedulingExternalID: "consultation-30min",
		}, nil)

		provider.On("CreateAppointment", mock.Anything, appointment).Return("", "", errors.New("provider error"))

		// Act
		err := service.BookAppointment(context.Background(), appointment)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider error")
		repo.AssertNotCalled(t, "Create")
		facilityRepo.AssertExpectations(t)
	})
}
