package loaders

import (
	"context"
	"fmt"

	"github.com/graph-gophers/dataloader/v7"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

type ctxKey string

const loadersKey ctxKey = "dataloaders"

// Loaders contains all the dataloaders for the application
type Loaders struct {
	FacilityLoader  *dataloader.Loader[string, *entities.Facility]
	ProcedureLoader *dataloader.Loader[string, *entities.Procedure]
}

// NewLoaders creates a new instance of Loaders
func NewLoaders(facilityRepo repositories.FacilityRepository, procedureRepo repositories.ProcedureRepository) *Loaders {
	return &Loaders{
		FacilityLoader: dataloader.NewBatchedLoader(func(ctx context.Context, keys []string) []*dataloader.Result[*entities.Facility] {
			results := make([]*dataloader.Result[*entities.Facility], len(keys))
			facilities, err := facilityRepo.GetByIDs(ctx, keys)

			facilityMap := make(map[string]*entities.Facility)
			if err == nil {
				for _, f := range facilities {
					facilityMap[f.ID] = f
				}
			}

			for i, key := range keys {
				if err != nil {
					results[i] = &dataloader.Result[*entities.Facility]{Error: err}
				} else if f, ok := facilityMap[key]; ok {
					results[i] = &dataloader.Result[*entities.Facility]{Data: f}
				} else {
					results[i] = &dataloader.Result[*entities.Facility]{Error: fmt.Errorf("facility %s not found", key)}
				}
			}
			return results
		}),
		ProcedureLoader: dataloader.NewBatchedLoader(func(ctx context.Context, keys []string) []*dataloader.Result[*entities.Procedure] {
			results := make([]*dataloader.Result[*entities.Procedure], len(keys))
			procs, err := procedureRepo.GetByIDs(ctx, keys)

			procMap := make(map[string]*entities.Procedure)
			if err == nil {
				for _, p := range procs {
					procMap[p.ID] = p
				}
			}

			for i, key := range keys {
				if err != nil {
					results[i] = &dataloader.Result[*entities.Procedure]{Error: err}
				} else if p, ok := procMap[key]; ok {
					results[i] = &dataloader.Result[*entities.Procedure]{Data: p}
				} else {
					results[i] = &dataloader.Result[*entities.Procedure]{Error: fmt.Errorf("procedure %s not found", key)}
				}
			}
			return results
		}),
	}
}

// For returns the loaders for a given context
func For(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}

// WithLoaders returns a new context with the loaders attached
func WithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, loadersKey, loaders)
}
