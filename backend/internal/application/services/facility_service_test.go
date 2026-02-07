package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/tests/mocks"
)

func TestFacilityService_SearchResults_EnrichesServicesAndPrice(t *testing.T) {
	ctx := context.Background()

	facilityRepo := mocks.NewMockFacilityRepository(t)
	facilityProcedureRepo := mocks.NewMockFacilityProcedureRepository(t)
	procedureRepo := mocks.NewMockProcedureRepository(t)

	service := services.NewFacilityService(facilityRepo, nil, facilityProcedureRepo, procedureRepo, nil)

	params := repositories.SearchParams{
		Latitude:  6.5244,
		Longitude: 3.3792,
		RadiusKm:  10,
		Limit:     1,
		Offset:    0,
	}

	facility := &entities.Facility{
		ID:           "fac-1",
		Name:         "Test Hospital",
		FacilityType: "Hospital",
		Location:     entities.Location{Latitude: 6.525, Longitude: 3.38},
		UpdatedAt:    time.Now(),
	}

	facilityRepo.EXPECT().
		Search(mock.Anything, params).
		Return([]*entities.Facility{facility}, nil)

	facilityProcedures := []*entities.FacilityProcedure{
		{
			ID:          "fp-1",
			FacilityID:  "fac-1",
			ProcedureID: "proc-1",
			Price:       20000,
			Currency:    "NGN",
			IsAvailable: true,
		},
		{
			ID:          "fp-2",
			FacilityID:  "fac-1",
			ProcedureID: "proc-2",
			Price:       15000,
			Currency:    "NGN",
			IsAvailable: true,
		},
	}

	facilityProcedureRepo.EXPECT().
		ListByFacility(mock.Anything, "fac-1").
		Return(facilityProcedures, nil)

	procedureRepo.EXPECT().
		GetByID(mock.Anything, "proc-2").
		Return(&entities.Procedure{Name: "CT Scan"}, nil)
	procedureRepo.EXPECT().
		GetByID(mock.Anything, "proc-1").
		Return(&entities.Procedure{Name: "MRI"}, nil)

	results, err := service.SearchResults(ctx, params)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, []string{"CT Scan", "MRI"}, results[0].Services)
	assert.Len(t, results[0].ServicePrices, 2)
	assert.Equal(t, "CT Scan", results[0].ServicePrices[0].Name)
	assert.Equal(t, 15000.0, results[0].ServicePrices[0].Price)
	assert.Equal(t, "NGN", results[0].ServicePrices[0].Currency)
	if assert.NotNil(t, results[0].Price) {
		assert.Equal(t, 15000.0, results[0].Price.Min)
		assert.Equal(t, 20000.0, results[0].Price.Max)
		assert.Equal(t, "NGN", results[0].Price.Currency)
	}
}
