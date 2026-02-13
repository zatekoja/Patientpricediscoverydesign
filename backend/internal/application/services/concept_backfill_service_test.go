package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// Mocks

type MockProcedureRepo struct {
	mock.Mock
}

func (m *MockProcedureRepo) Create(ctx context.Context, procedure *entities.Procedure) error {
	args := m.Called(ctx, procedure)
	return args.Error(0)
}

func (m *MockProcedureRepo) GetByID(ctx context.Context, id string) (*entities.Procedure, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Procedure), args.Error(1)
}

func (m *MockProcedureRepo) GetByCode(ctx context.Context, code string) (*entities.Procedure, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Procedure), args.Error(1)
}

func (m *MockProcedureRepo) GetByIDs(ctx context.Context, ids []string) ([]*entities.Procedure, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Procedure), args.Error(1)
}

func (m *MockProcedureRepo) Update(ctx context.Context, procedure *entities.Procedure) error {
	args := m.Called(ctx, procedure)
	return args.Error(0)
}

func (m *MockProcedureRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProcedureRepo) List(ctx context.Context, filter repositories.ProcedureFilter) ([]*entities.Procedure, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Procedure), args.Error(1)
}

type MockEnrichmentRepo struct {
	mock.Mock
}

func (m *MockEnrichmentRepo) GetByProcedureID(ctx context.Context, procedureID string) (*entities.ProcedureEnrichment, error) {
	args := m.Called(ctx, procedureID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.ProcedureEnrichment), args.Error(1)
}

func (m *MockEnrichmentRepo) Upsert(ctx context.Context, enrichment *entities.ProcedureEnrichment) error {
	args := m.Called(ctx, enrichment)
	return args.Error(0)
}

func (m *MockEnrichmentRepo) ListByStatus(ctx context.Context, status string, limit int) ([]*entities.ProcedureEnrichment, error) {
	args := m.Called(ctx, status, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.ProcedureEnrichment), args.Error(1)
}

func (m *MockEnrichmentRepo) UpdateStatus(ctx context.Context, id string, status string, errMsg string) error {
	args := m.Called(ctx, id, status, errMsg)
	return args.Error(0)
}

func (m *MockEnrichmentRepo) ListProcedureIDsNeedingEnrichment(ctx context.Context, version int, limit int) ([]string, error) {
	args := m.Called(ctx, version, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) EnrichProcedure(ctx context.Context, procedure *entities.Procedure) (*entities.ProcedureEnrichment, error) {
	args := m.Called(ctx, procedure)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.ProcedureEnrichment), args.Error(1)
}

// Tests

func TestBackfillAll_EnrichesUnenriched(t *testing.T) {
	mockProcRepo := new(MockProcedureRepo)
	mockEnrichRepo := new(MockEnrichmentRepo)
	mockProvider := new(MockProvider)

	service := NewConceptBackfillService(mockProcRepo, mockEnrichRepo, mockProvider, 1, 3)

	// Setup expectations
	procID := "proc-1"
	mockEnrichRepo.On("ListProcedureIDsNeedingEnrichment", mock.Anything, 1, 100).Return([]string{procID}, nil).Once()

	procedure := &entities.Procedure{ID: procID, Name: "Test Procedure"}
	mockProcRepo.On("GetByID", mock.Anything, procID).Return(procedure, nil)

	enriched := &entities.ProcedureEnrichment{
		ProcedureID:    procID,
		SearchConcepts: &entities.SearchConcepts{Conditions: []string{"test"}},
	}
	mockProvider.On("EnrichProcedure", mock.Anything, procedure).Return(enriched, nil)
	
	// Expect Upsert with status "completed"
	mockEnrichRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(e *entities.ProcedureEnrichment) bool {
		return e.ProcedureID == procID && e.EnrichmentStatus == "completed" && e.EnrichmentVersion == 1
	})).Return(nil)

	summary, err := service.BackfillAll(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 1, summary.TotalProcessed)
	assert.Equal(t, 1, summary.SuccessCount)
	assert.Equal(t, 0, summary.FailureCount)
	
	mockProcRepo.AssertExpectations(t)
	mockEnrichRepo.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

func TestBackfillSingle_Success(t *testing.T) {
	mockProcRepo := new(MockProcedureRepo)
	mockEnrichRepo := new(MockEnrichmentRepo)
	mockProvider := new(MockProvider)

	service := NewConceptBackfillService(mockProcRepo, mockEnrichRepo, mockProvider, 1, 3)

	procID := "proc-1"
	procedure := &entities.Procedure{ID: procID, Name: "Test Procedure"}
	
	mockProcRepo.On("GetByID", mock.Anything, procID).Return(procedure, nil)

	enriched := &entities.ProcedureEnrichment{
		ProcedureID:    procID,
		SearchConcepts: &entities.SearchConcepts{Conditions: []string{"test"}},
	}
	mockProvider.On("EnrichProcedure", mock.Anything, procedure).Return(enriched, nil)

	mockEnrichRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(e *entities.ProcedureEnrichment) bool {
		return e.ProcedureID == procID && e.EnrichmentStatus == "completed"
	})).Return(nil)

	err := service.BackfillSingle(context.Background(), procID)

	assert.NoError(t, err)
	mockProcRepo.AssertExpectations(t)
	mockEnrichRepo.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

func TestBackfillSingle_ProviderError(t *testing.T) {
	mockProcRepo := new(MockProcedureRepo)
	mockEnrichRepo := new(MockEnrichmentRepo)
	mockProvider := new(MockProvider)

	service := NewConceptBackfillService(mockProcRepo, mockEnrichRepo, mockProvider, 1, 3)

	procID := "proc-1"
	procedure := &entities.Procedure{ID: procID, Name: "Test Procedure"}
	
	mockProcRepo.On("GetByID", mock.Anything, procID).Return(procedure, nil)

	mockProvider.On("EnrichProcedure", mock.Anything, procedure).Return(nil, errors.New("provider error"))

	// Simulate existing record
	existingEnrichment := &entities.ProcedureEnrichment{
		ID:          "enrich-1",
		ProcedureID: procID,
		RetryCount:  0,
	}
	mockEnrichRepo.On("GetByProcedureID", mock.Anything, procID).Return(existingEnrichment, nil)

	// Should update status to "pending" (retry)
	mockEnrichRepo.On("UpdateStatus", mock.Anything, "enrich-1", "pending", "provider error").Return(nil)

	err := service.BackfillSingle(context.Background(), procID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider error")
	mockEnrichRepo.AssertExpectations(t)
}

func TestBackfillAll_MaxRetries_MovesToAbandoned(t *testing.T) {
	mockProcRepo := new(MockProcedureRepo)
	mockEnrichRepo := new(MockEnrichmentRepo)
	mockProvider := new(MockProvider)

	// Max retries = 1
	service := NewConceptBackfillService(mockProcRepo, mockEnrichRepo, mockProvider, 1, 1)

	procID := "proc-1"
	// Mock returns ID needing enrichment
	mockEnrichRepo.On("ListProcedureIDsNeedingEnrichment", mock.Anything, 1, 100).Return([]string{procID}, nil).Once()

	procedure := &entities.Procedure{ID: procID, Name: "Test Procedure"}
	mockProcRepo.On("GetByID", mock.Anything, procID).Return(procedure, nil)

	// Provider fails
	mockProvider.On("EnrichProcedure", mock.Anything, procedure).Return(nil, errors.New("provider error"))

	// Retrieve existing enrichment to check retry count
	existingEnrichment := &entities.ProcedureEnrichment{
		ID:          "enrich-1",
		ProcedureID: procID,
		RetryCount:  1, // Already at max retries (1)
	}
	mockEnrichRepo.On("GetByProcedureID", mock.Anything, procID).Return(existingEnrichment, nil)

	// Since retry count >= max retries, should update to ABANDONED
	mockEnrichRepo.On("UpdateStatus", mock.Anything, "enrich-1", "abandoned", "provider error").Return(nil)

	summary, err := service.BackfillAll(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 1, summary.FailureCount)
	mockEnrichRepo.AssertExpectations(t)
}