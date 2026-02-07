package providers

import (
	"context"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// ProcedureEnrichmentProvider defines a provider that can enrich a procedure.
type ProcedureEnrichmentProvider interface {
	EnrichProcedure(ctx context.Context, procedure *entities.Procedure) (*entities.ProcedureEnrichment, error)
}
