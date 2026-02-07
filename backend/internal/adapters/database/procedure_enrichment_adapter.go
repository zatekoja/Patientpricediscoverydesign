package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/google/uuid"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
)

// ProcedureEnrichmentAdapter implements ProcedureEnrichmentRepository.
type ProcedureEnrichmentAdapter struct {
	client *postgres.Client
	db     *goqu.Database
}

// NewProcedureEnrichmentAdapter creates a new adapter.
func NewProcedureEnrichmentAdapter(client *postgres.Client) repositories.ProcedureEnrichmentRepository {
	return &ProcedureEnrichmentAdapter{
		client: client,
		db:     goqu.New("postgres", client.DB()),
	}
}

// GetByProcedureID retrieves enrichment by procedure ID.
func (a *ProcedureEnrichmentAdapter) GetByProcedureID(ctx context.Context, procedureID string) (*entities.ProcedureEnrichment, error) {
	query, args, err := a.db.Select(
		"id",
		"procedure_id",
		"description",
		"prep_steps",
		"risks",
		"recovery",
		"provider",
		"model",
		"created_at",
		"updated_at",
	).
		From("procedure_enrichments").
		Where(goqu.Ex{"procedure_id": procedureID}).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build enrichment query", err)
	}

	var prepRaw, risksRaw, recoveryRaw []byte
	var description, provider, model sql.NullString
	enrichment := &entities.ProcedureEnrichment{}

	err = a.client.DB().QueryRowContext(ctx, query, args...).Scan(
		&enrichment.ID,
		&enrichment.ProcedureID,
		&description,
		&prepRaw,
		&risksRaw,
		&recoveryRaw,
		&provider,
		&model,
		&enrichment.CreatedAt,
		&enrichment.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, apperrors.NewNotFoundError(fmt.Sprintf("procedure enrichment with procedure_id %s not found", procedureID))
	}
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get procedure enrichment", err)
	}

	enrichment.Description = description.String
	enrichment.Provider = provider.String
	enrichment.Model = model.String

	if len(prepRaw) > 0 {
		_ = json.Unmarshal(prepRaw, &enrichment.PrepSteps)
	}
	if len(risksRaw) > 0 {
		_ = json.Unmarshal(risksRaw, &enrichment.Risks)
	}
	if len(recoveryRaw) > 0 {
		_ = json.Unmarshal(recoveryRaw, &enrichment.Recovery)
	}

	return enrichment, nil
}

// Upsert inserts or updates enrichment.
func (a *ProcedureEnrichmentAdapter) Upsert(ctx context.Context, enrichment *entities.ProcedureEnrichment) error {
	if enrichment == nil {
		return apperrors.NewValidationError("enrichment is required")
	}
	if enrichment.ID == "" {
		enrichment.ID = uuid.New().String()
	}

	prepBytes, _ := json.Marshal(enrichment.PrepSteps)
	risksBytes, _ := json.Marshal(enrichment.Risks)
	recoveryBytes, _ := json.Marshal(enrichment.Recovery)

	query := `
		INSERT INTO procedure_enrichments
			(id, procedure_id, description, prep_steps, risks, recovery, provider, model, created_at, updated_at)
		VALUES
			($1, $2, $3, $4::jsonb, $5::jsonb, $6::jsonb, $7, $8, $9, $10)
		ON CONFLICT (procedure_id)
		DO UPDATE SET
			description = EXCLUDED.description,
			prep_steps = EXCLUDED.prep_steps,
			risks = EXCLUDED.risks,
			recovery = EXCLUDED.recovery,
			provider = EXCLUDED.provider,
			model = EXCLUDED.model,
			updated_at = EXCLUDED.updated_at
	`

	_, err := a.client.DB().ExecContext(
		ctx,
		query,
		enrichment.ID,
		enrichment.ProcedureID,
		enrichment.Description,
		string(prepBytes),
		string(risksBytes),
		string(recoveryBytes),
		enrichment.Provider,
		enrichment.Model,
		enrichment.CreatedAt,
		enrichment.UpdatedAt,
	)
	if err != nil {
		return apperrors.NewInternalError("failed to upsert procedure enrichment", err)
	}

	return nil
}
