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
		"search_concepts",
		"provider",
		"model",
		"enrichment_status",
		"enrichment_version",
		"retry_count",
		"last_error",
		"created_at",
		"updated_at",
	).
		From("procedure_enrichments").
		Where(goqu.Ex{"procedure_id": procedureID}).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build enrichment query", err)
	}

	var prepRaw, risksRaw, recoveryRaw, conceptsRaw []byte
	var description, provider, model, enrichmentStatus, lastError sql.NullString
	var enrichmentVersion, retryCount sql.NullInt32
	enrichment := &entities.ProcedureEnrichment{}

	err = a.client.DB().QueryRowContext(ctx, query, args...).Scan(
		&enrichment.ID,
		&enrichment.ProcedureID,
		&description,
		&prepRaw,
		&risksRaw,
		&recoveryRaw,
		&conceptsRaw,
		&provider,
		&model,
		&enrichmentStatus,
		&enrichmentVersion,
		&retryCount,
		&lastError,
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
	enrichment.EnrichmentStatus = enrichmentStatus.String
	enrichment.EnrichmentVersion = int(enrichmentVersion.Int32)
	enrichment.RetryCount = int(retryCount.Int32)
	enrichment.LastError = lastError.String

	if len(prepRaw) > 0 {
		_ = json.Unmarshal(prepRaw, &enrichment.PrepSteps)
	}
	if len(risksRaw) > 0 {
		_ = json.Unmarshal(risksRaw, &enrichment.Risks)
	}
	if len(recoveryRaw) > 0 {
		_ = json.Unmarshal(recoveryRaw, &enrichment.Recovery)
	}
	if len(conceptsRaw) > 0 && string(conceptsRaw) != "{}" {
		var sc entities.SearchConcepts
		if json.Unmarshal(conceptsRaw, &sc) == nil {
			enrichment.SearchConcepts = &sc
		}
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

	conceptsBytes := []byte("{}")
	if enrichment.SearchConcepts != nil {
		if b, err := json.Marshal(enrichment.SearchConcepts); err == nil {
			conceptsBytes = b
		}
	}

	enrichmentStatus := enrichment.EnrichmentStatus
	if enrichmentStatus == "" {
		enrichmentStatus = "pending"
	}

	query := `
		INSERT INTO procedure_enrichments
			(id, procedure_id, description, prep_steps, risks, recovery, search_concepts, provider, model, enrichment_status, enrichment_version, retry_count, last_error, created_at, updated_at)
		VALUES
			($1, $2, $3, $4::jsonb, $5::jsonb, $6::jsonb, $7::jsonb, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (procedure_id)
		DO UPDATE SET
			description = EXCLUDED.description,
			prep_steps = EXCLUDED.prep_steps,
			risks = EXCLUDED.risks,
			recovery = EXCLUDED.recovery,
			search_concepts = EXCLUDED.search_concepts,
			provider = EXCLUDED.provider,
			model = EXCLUDED.model,
			enrichment_status = EXCLUDED.enrichment_status,
			enrichment_version = EXCLUDED.enrichment_version,
			retry_count = EXCLUDED.retry_count,
			last_error = EXCLUDED.last_error,
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
		string(conceptsBytes),
		enrichment.Provider,
		enrichment.Model,
		enrichmentStatus,
		enrichment.EnrichmentVersion,
		enrichment.RetryCount,
		enrichment.LastError,
		enrichment.CreatedAt,
		enrichment.UpdatedAt,
	)
	if err != nil {
		return apperrors.NewInternalError("failed to upsert procedure enrichment", err)
	}

	return nil
}

// ListByStatus returns enrichments filtered by enrichment_status.
func (a *ProcedureEnrichmentAdapter) ListByStatus(ctx context.Context, status string, limit int) ([]*entities.ProcedureEnrichment, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, procedure_id, description, prep_steps, risks, recovery, search_concepts,
		       provider, model, enrichment_status, enrichment_version, retry_count, last_error,
		       created_at, updated_at
		FROM procedure_enrichments
		WHERE enrichment_status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := a.client.DB().QueryContext(ctx, query, status, limit)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list enrichments by status", err)
	}
	defer rows.Close()

	return scanEnrichmentRows(rows)
}

// UpdateStatus updates enrichment_status, last_error, and increments retry_count.
func (a *ProcedureEnrichmentAdapter) UpdateStatus(ctx context.Context, id string, status string, errMsg string) error {
	query := `
		UPDATE procedure_enrichments
		SET enrichment_status = $1,
		    last_error = $2,
		    retry_count = retry_count + 1,
		    updated_at = NOW()
		WHERE id = $3
	`
	_, err := a.client.DB().ExecContext(ctx, query, status, errMsg, id)
	if err != nil {
		return apperrors.NewInternalError("failed to update enrichment status", err)
	}
	return nil
}

// ListProcedureIDsNeedingEnrichment returns procedure IDs that either lack an enrichment record
// or have enrichment_version < the target version.
func (a *ProcedureEnrichmentAdapter) ListProcedureIDsNeedingEnrichment(ctx context.Context, version int, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT p.id
		FROM procedures p
		LEFT JOIN procedure_enrichments pe ON pe.procedure_id = p.id
		WHERE p.is_active = true
		  AND (pe.id IS NULL
		       OR pe.enrichment_version < $1
		       OR pe.enrichment_status = 'failed')
		ORDER BY p.created_at ASC
		LIMIT $2
	`

	rows, err := a.client.DB().QueryContext(ctx, query, version, limit)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list procedure IDs needing enrichment", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, apperrors.NewInternalError("failed to scan procedure ID", err)
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func scanEnrichmentRows(rows *sql.Rows) ([]*entities.ProcedureEnrichment, error) {
	var results []*entities.ProcedureEnrichment

	for rows.Next() {
		var prepRaw, risksRaw, recoveryRaw, conceptsRaw []byte
		var description, provider, model, enrichmentStatus, lastError sql.NullString
		var enrichmentVersion, retryCount sql.NullInt32
		enrichment := &entities.ProcedureEnrichment{}

		err := rows.Scan(
			&enrichment.ID,
			&enrichment.ProcedureID,
			&description,
			&prepRaw,
			&risksRaw,
			&recoveryRaw,
			&conceptsRaw,
			&provider,
			&model,
			&enrichmentStatus,
			&enrichmentVersion,
			&retryCount,
			&lastError,
			&enrichment.CreatedAt,
			&enrichment.UpdatedAt,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to scan enrichment row", err)
		}

		enrichment.Description = description.String
		enrichment.Provider = provider.String
		enrichment.Model = model.String
		enrichment.EnrichmentStatus = enrichmentStatus.String
		enrichment.EnrichmentVersion = int(enrichmentVersion.Int32)
		enrichment.RetryCount = int(retryCount.Int32)
		enrichment.LastError = lastError.String

		if len(prepRaw) > 0 {
			_ = json.Unmarshal(prepRaw, &enrichment.PrepSteps)
		}
		if len(risksRaw) > 0 {
			_ = json.Unmarshal(risksRaw, &enrichment.Risks)
		}
		if len(recoveryRaw) > 0 {
			_ = json.Unmarshal(recoveryRaw, &enrichment.Recovery)
		}
		if len(conceptsRaw) > 0 && string(conceptsRaw) != "{}" {
			var sc entities.SearchConcepts
			if json.Unmarshal(conceptsRaw, &sc) == nil {
				enrichment.SearchConcepts = &sc
			}
		}

		results = append(results, enrichment)
	}

	return results, rows.Err()
}
