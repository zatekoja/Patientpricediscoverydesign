package resolvers

import (
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/query/services"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	searchAdapter         services.SearchAdapter
	facilityRepo          repositories.FacilityRepository
	appointmentRepo       repositories.AppointmentRepository
	procedureRepo         repositories.ProcedureRepository
	facilityProcedureRepo repositories.FacilityProcedureRepository
	insuranceRepo         repositories.InsuranceRepository
	cache                 services.QueryCacheProvider
}

// NewResolver creates a new resolver with dependencies
func NewResolver(
	searchAdapter services.SearchAdapter,
	facilityRepo repositories.FacilityRepository,
	appointmentRepo repositories.AppointmentRepository,
	procedureRepo repositories.ProcedureRepository,
	facilityProcedureRepo repositories.FacilityProcedureRepository,
	insuranceRepo repositories.InsuranceRepository,
	cache services.QueryCacheProvider,
) *Resolver {
	return &Resolver{
		searchAdapter:         searchAdapter,
		facilityRepo:          facilityRepo,
		appointmentRepo:       appointmentRepo,
		procedureRepo:         procedureRepo,
		facilityProcedureRepo: facilityProcedureRepo,
		insuranceRepo:         insuranceRepo,
		cache:                 cache,
	}
}
