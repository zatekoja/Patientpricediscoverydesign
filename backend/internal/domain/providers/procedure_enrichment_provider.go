package providers

import (
	"context"
	"errors"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// ErrProcedureEnrichmentUnauthorized indicates the enrichment provider rejected credentials.
// Callers may treat this as a misconfiguration and disable enrichment to avoid repeated failures.
var ErrProcedureEnrichmentUnauthorized = errors.New("procedure enrichment provider unauthorized")

// ProcedureEnrichmentProvider defines a provider that can enrich a procedure.
type ProcedureEnrichmentProvider interface {
	EnrichProcedure(ctx context.Context, procedure *entities.Procedure) (*entities.ProcedureEnrichment, error)
}
