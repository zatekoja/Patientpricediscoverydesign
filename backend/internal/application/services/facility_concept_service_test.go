package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// Mocks

type MockFacilityProcRepo struct {
	mock.Mock
}

func (m *MockFacilityProcRepo) Create(ctx context.Context, fp *entities.FacilityProcedure) error {
	return nil
}
func (m *MockFacilityProcRepo) GetByID(ctx context.Context, id string) (*entities.FacilityProcedure, error) {
	return nil, nil
}
func (m *MockFacilityProcRepo) GetByFacilityAndProcedure(ctx context.Context, facilityID, procedureID string) (*entities.FacilityProcedure, error) {
	return nil, nil
}
func (m *MockFacilityProcRepo) ListByFacility(ctx context.Context, facilityID string) ([]*entities.FacilityProcedure, error) {
	args := m.Called(ctx, facilityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.FacilityProcedure), args.Error(1)
}
func (m *MockFacilityProcRepo) ListByFacilityWithCount(ctx context.Context, facilityID string, filter repositories.FacilityProcedureFilter) ([]*entities.FacilityProcedure, int, error) {
	return nil, 0, nil
}
func (m *MockFacilityProcRepo) Update(ctx context.Context, fp *entities.FacilityProcedure) error {
	return nil
}
func (m *MockFacilityProcRepo) Delete(ctx context.Context, id string) error {
	return nil
}

type MockEnrichRepoForConcept struct {
	mock.Mock
}

func (m *MockEnrichRepoForConcept) GetByProcedureID(ctx context.Context, procedureID string) (*entities.ProcedureEnrichment, error) {
	args := m.Called(ctx, procedureID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.ProcedureEnrichment), args.Error(1)
}
func (m *MockEnrichRepoForConcept) Upsert(ctx context.Context, enrichment *entities.ProcedureEnrichment) error {
	return nil
}
func (m *MockEnrichRepoForConcept) ListByStatus(ctx context.Context, status string, limit int) ([]*entities.ProcedureEnrichment, error) {
	return nil, nil
}
func (m *MockEnrichRepoForConcept) UpdateStatus(ctx context.Context, id string, status string, errMsg string) error {
	return nil
}
func (m *MockEnrichRepoForConcept) ListProcedureIDsNeedingEnrichment(ctx context.Context, version int, limit int) ([]string, error) {
	return nil, nil
}

// Tests

func TestAggregate_MultipleProcedures(t *testing.T) {
	mockFPRepo := new(MockFacilityProcRepo)
	mockEnrichRepo := new(MockEnrichRepoForConcept)

	service := NewFacilityConceptService(mockFPRepo, mockEnrichRepo)

	facID := "fac-1"
	fps := []*entities.FacilityProcedure{
		{ProcedureID: "p1"},
		{ProcedureID: "p2"},
	}
	mockFPRepo.On("ListByFacility", mock.Anything, facID).Return(fps, nil)

	e1 := &entities.ProcedureEnrichment{
		ProcedureID: "p1",
		SearchConcepts: &entities.SearchConcepts{
			Conditions:  []string{"malaria"},
			Specialties: []string{"general_practice"},
		},
	}
	e2 := &entities.ProcedureEnrichment{
		ProcedureID: "p2",
		SearchConcepts: &entities.SearchConcepts{
			Conditions:  []string{"typhoid"},
			Specialties: []string{"general_practice"},
		},
	}

	mockEnrichRepo.On("GetByProcedureID", mock.Anything, "p1").Return(e1, nil)
	mockEnrichRepo.On("GetByProcedureID", mock.Anything, "p2").Return(e2, nil)

	concepts, err := service.AggregateConcepts(context.Background(), facID)

	assert.NoError(t, err)
	assert.ElementsMatch(t, concepts.Conditions, []string{"malaria", "typhoid"})
	assert.ElementsMatch(t, concepts.Specialties, []string{"general_practice"}) // deduped
}

func TestAggregate_NoProcedures(t *testing.T) {
	mockFPRepo := new(MockFacilityProcRepo)
	mockEnrichRepo := new(MockEnrichRepoForConcept)

	service := NewFacilityConceptService(mockFPRepo, mockEnrichRepo)

	facID := "fac-1"
	mockFPRepo.On("ListByFacility", mock.Anything, facID).Return([]*entities.FacilityProcedure{}, nil)

	concepts, err := service.AggregateConcepts(context.Background(), facID)

	assert.NoError(t, err)
	assert.Empty(t, concepts.Conditions)
}
