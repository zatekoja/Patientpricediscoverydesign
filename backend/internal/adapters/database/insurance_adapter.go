package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
)

// InsuranceAdapter implements InsuranceRepository
type InsuranceAdapter struct {
	client *postgres.Client
	db     *goqu.Database
}

// NewInsuranceAdapter creates a new insurance adapter
func NewInsuranceAdapter(client *postgres.Client) repositories.InsuranceRepository {
	return &InsuranceAdapter{
		client: client,
		db:     goqu.New("postgres", client.DB()),
	}
}

// Create creates a new insurance provider
func (a *InsuranceAdapter) Create(ctx context.Context, insurance *entities.InsuranceProvider) error {
	record := goqu.Record{
		"id":           insurance.ID,
		"name":         insurance.Name,
		"code":         insurance.Code,
		"phone_number": sql.NullString{String: insurance.PhoneNumber, Valid: insurance.PhoneNumber != ""},
		"website":      sql.NullString{String: insurance.Website, Valid: insurance.Website != ""},
		"is_active":    insurance.IsActive,
		"created_at":   insurance.CreatedAt,
		"updated_at":   insurance.UpdatedAt,
	}

	query, args, err := a.db.Insert("insurance_providers").Rows(record).ToSQL()
	if err != nil {
		return apperrors.NewInternalError("failed to build insert query", err)
	}

	_, err = a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to create insurance provider", err)
	}

	return nil
}

// GetByID retrieves an insurance provider by ID
func (a *InsuranceAdapter) GetByID(ctx context.Context, id string) (*entities.InsuranceProvider, error) {
	return a.getByField(ctx, "id", id)
}

// GetByCode retrieves an insurance provider by code
func (a *InsuranceAdapter) GetByCode(ctx context.Context, code string) (*entities.InsuranceProvider, error) {
	return a.getByField(ctx, "code", code)
}

func (a *InsuranceAdapter) getByField(ctx context.Context, field, value string) (*entities.InsuranceProvider, error) {
	query, args, err := a.db.Select(
		"id", "name", "code", "phone_number", "website",
		"is_active", "created_at", "updated_at",
	).From("insurance_providers").
		Where(goqu.Ex{field: value}).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	provider := &entities.InsuranceProvider{}
	var phone, website sql.NullString

	err = a.client.DB().QueryRowContext(ctx, query, args...).Scan(
		&provider.ID,
		&provider.Name,
		&provider.Code,
		&phone,
		&website,
		&provider.IsActive,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, apperrors.NewNotFoundError(fmt.Sprintf("insurance provider with %s %s not found", field, value))
	}
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get insurance provider", err)
	}

	provider.PhoneNumber = phone.String
	provider.Website = website.String

	return provider, nil
}

// Update updates an insurance provider
func (a *InsuranceAdapter) Update(ctx context.Context, insurance *entities.InsuranceProvider) error {
	insurance.UpdatedAt = time.Now()

	record := goqu.Record{
		"name":         insurance.Name,
		"code":         insurance.Code,
		"phone_number": sql.NullString{String: insurance.PhoneNumber, Valid: insurance.PhoneNumber != ""},
		"website":      sql.NullString{String: insurance.Website, Valid: insurance.Website != ""},
		"is_active":    insurance.IsActive,
		"updated_at":   insurance.UpdatedAt,
	}

	query, args, err := a.db.Update("insurance_providers").
		Set(record).
		Where(goqu.Ex{"id": insurance.ID}).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build update query", err)
	}

	result, err := a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to update insurance provider", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("insurance provider with id %s not found", insurance.ID))
	}

	return nil
}

// Delete deletes an insurance provider
func (a *InsuranceAdapter) Delete(ctx context.Context, id string) error {
	// Soft delete
	query, args, err := a.db.Update("insurance_providers").
		Set(goqu.Record{
			"is_active":  false,
			"updated_at": time.Now(),
		}).
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build delete query", err)
	}

	result, err := a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to delete insurance provider", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("insurance provider with id %s not found", id))
	}

	return nil
}

// List retrieves insurance providers
func (a *InsuranceAdapter) List(ctx context.Context, filter repositories.InsuranceFilter) ([]*entities.InsuranceProvider, error) {
	ds := a.db.Select(
		"id", "name", "code", "phone_number", "website",
		"is_active", "created_at", "updated_at",
	).From("insurance_providers")

	if filter.IsActive != nil {
		ds = ds.Where(goqu.Ex{"is_active": *filter.IsActive})
	}

	ds = ds.Order(goqu.I("name").Asc())

	if filter.Limit > 0 {
		ds = ds.Limit(uint(filter.Limit))
	}

	if filter.Offset > 0 {
		ds = ds.Offset(uint(filter.Offset))
	}

	query, args, err := ds.ToSQL()
	if err != nil {
		return nil, apperrors.NewInternalError("failed to build list query", err)
	}

	rows, err := a.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list insurance providers", err)
	}
	defer rows.Close()

	var providers []*entities.InsuranceProvider
	for rows.Next() {
		provider := &entities.InsuranceProvider{}
		var phone, website sql.NullString

		err := rows.Scan(
			&provider.ID,
			&provider.Name,
			&provider.Code,
			&phone,
			&website,
			&provider.IsActive,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to scan insurance provider", err)
		}

		provider.PhoneNumber = phone.String
		provider.Website = website.String

		providers = append(providers, provider)
	}

	return providers, nil
}

// GetFacilityInsurance retrieves accepted insurance for a facility
func (a *InsuranceAdapter) GetFacilityInsurance(ctx context.Context, facilityID string) ([]*entities.InsuranceProvider, error) {
	// Join facility_insurance with insurance_providers
	query, args, err := a.db.Select(
		"i.id", "i.name", "i.code", "i.phone_number", "i.website",
		"i.is_active", "i.created_at", "i.updated_at",
	).From(goqu.T("insurance_providers").As("i")).
		Join(
			goqu.T("facility_insurance").As("fi"),
			goqu.On(goqu.I("i.id").Eq(goqu.I("fi.insurance_provider_id"))),
		).
		Where(goqu.Ex{
			"fi.facility_id": facilityID,
			"fi.is_accepted": true,
			"i.is_active":    true,
		}).
		Order(goqu.I("i.name").Asc()).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	rows, err := a.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get facility insurance", err)
	}
	defer rows.Close()

	var providers []*entities.InsuranceProvider
	for rows.Next() {
		provider := &entities.InsuranceProvider{}
		var phone, website sql.NullString

		err := rows.Scan(
			&provider.ID,
			&provider.Name,
			&provider.Code,
			&phone,
			&website,
			&provider.IsActive,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to scan insurance provider", err)
		}

		provider.PhoneNumber = phone.String
		provider.Website = website.String

		providers = append(providers, provider)
	}

	return providers, nil
}
