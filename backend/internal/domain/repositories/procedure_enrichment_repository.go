package repositories

import (
	"context"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// ProcedureEnrichmentRepository defines the interface for procedure enrichment storage.
type ProcedureEnrichmentRepository interface {
	GetByProcedureID(ctx context.Context, procedureID string) (*entities.ProcedureEnrichment, error)
	Upsert(ctx context.Context, enrichment *entities.ProcedureEnrichment) error
}
