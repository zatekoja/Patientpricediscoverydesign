package repositories

import (
	"context"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

// ProcedureEnrichmentRepository defines the interface for procedure enrichment storage.
type ProcedureEnrichmentRepository interface {
	GetByProcedureID(ctx context.Context, procedureID string) (*entities.ProcedureEnrichment, error)
	Upsert(ctx context.Context, enrichment *entities.ProcedureEnrichment) error
	ListByStatus(ctx context.Context, status string, limit int) ([]*entities.ProcedureEnrichment, error)
	UpdateStatus(ctx context.Context, id string, status string, errMsg string) error
	ListProcedureIDsNeedingEnrichment(ctx context.Context, version int, limit int) ([]string, error)
}
