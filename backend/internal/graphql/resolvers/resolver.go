package resolvers

import (
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/providerapi"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/query/services"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	searchAdapter services.SearchAdapter
	facilityRepo  repositories.FacilityRepository
	cache         services.QueryCacheProvider
	providerClient providerapi.Client
}

// NewResolver creates a new resolver with dependencies
func NewResolver(
	searchAdapter services.SearchAdapter,
	facilityRepo repositories.FacilityRepository,
	cache services.QueryCacheProvider,
	providerClient providerapi.Client,
) *Resolver {
	return &Resolver{
		searchAdapter: searchAdapter,
		facilityRepo:  facilityRepo,
		cache:         cache,
		providerClient: providerClient,
	}
}
