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

// ProcedureAdapter implements ProcedureRepository
type ProcedureAdapter struct {
	client *postgres.Client
	db     *goqu.Database
}

// NewProcedureAdapter creates a new procedure adapter
func NewProcedureAdapter(client *postgres.Client) repositories.ProcedureRepository {
	return &ProcedureAdapter{
		client: client,
		db:     goqu.New("postgres", client.DB()),
	}
}

// Create creates a new procedure
func (a *ProcedureAdapter) Create(ctx context.Context, procedure *entities.Procedure) error {
	record := goqu.Record{
		"id":          procedure.ID,
		"name":        procedure.Name,
		"code":        procedure.Code,
		"category":    sql.NullString{String: procedure.Category, Valid: procedure.Category != ""},
		"description": sql.NullString{String: procedure.Description, Valid: procedure.Description != ""},
		"is_active":   procedure.IsActive,
		"created_at":  procedure.CreatedAt,
		"updated_at":  procedure.UpdatedAt,
	}

	query, args, err := a.db.Insert("procedures").Rows(record).ToSQL()
	if err != nil {
		return apperrors.NewInternalError("failed to build insert query", err)
	}

	_, err = a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to create procedure", err)
	}

	return nil
}

// GetByID retrieves a procedure by ID
func (a *ProcedureAdapter) GetByID(ctx context.Context, id string) (*entities.Procedure, error) {
	return a.getByField(ctx, "id", id)
}

// GetByCode retrieves a procedure by code
func (a *ProcedureAdapter) GetByCode(ctx context.Context, code string) (*entities.Procedure, error) {
	return a.getByField(ctx, "code", code)
}

func (a *ProcedureAdapter) getByField(ctx context.Context, field, value string) (*entities.Procedure, error) {
	query, args, err := a.db.Select(
		"id", "name", "code", "category", "description",
		"is_active", "created_at", "updated_at",
	).From("procedures").
		Where(goqu.Ex{field: value}).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	procedure := &entities.Procedure{}
	var category, description sql.NullString

	err = a.client.DB().QueryRowContext(ctx, query, args...).Scan(
		&procedure.ID,
		&procedure.Name,
		&procedure.Code,
		&category,
		&description,
		&procedure.IsActive,
		&procedure.CreatedAt,
		&procedure.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, apperrors.NewNotFoundError(fmt.Sprintf("procedure with %s %s not found", field, value))
	}
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get procedure", err)
	}

	procedure.Category = category.String
	procedure.Description = description.String

	return procedure, nil
}

// Update updates a procedure
func (a *ProcedureAdapter) Update(ctx context.Context, procedure *entities.Procedure) error {
	procedure.UpdatedAt = time.Now()

	record := goqu.Record{
		"name":        procedure.Name,
		"code":        procedure.Code,
		"category":    sql.NullString{String: procedure.Category, Valid: procedure.Category != ""},
		"description": sql.NullString{String: procedure.Description, Valid: procedure.Description != ""},
		"is_active":   procedure.IsActive,
		"updated_at":  procedure.UpdatedAt,
	}

	query, args, err := a.db.Update("procedures").
		Set(record).
		Where(goqu.Ex{"id": procedure.ID}).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build update query", err)
	}

	result, err := a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to update procedure", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("procedure with id %s not found", procedure.ID))
	}

	return nil
}

// Delete deletes a procedure
func (a *ProcedureAdapter) Delete(ctx context.Context, id string) error {
	// Soft delete
	query, args, err := a.db.Update("procedures").
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
		return apperrors.NewInternalError("failed to delete procedure", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("procedure with id %s not found", id))
	}

	return nil
}

// List retrieves procedures with filters
func (a *ProcedureAdapter) List(ctx context.Context, filter repositories.ProcedureFilter) ([]*entities.Procedure, error) {
	ds := a.db.Select(
		"id", "name", "code", "category", "description",
		"is_active", "created_at", "updated_at",
	).From("procedures")

	if filter.Category != "" {
		ds = ds.Where(goqu.Ex{"category": filter.Category})
	}

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
		return nil, apperrors.NewInternalError("failed to list procedures", err)
	}
	defer rows.Close()

	var procedures []*entities.Procedure
	for rows.Next() {
		procedure := &entities.Procedure{}
		var category, description sql.NullString

		err := rows.Scan(
			&procedure.ID,
			&procedure.Name,
			&procedure.Code,
			&category,
			&description,
			&procedure.IsActive,
			&procedure.CreatedAt,
			&procedure.UpdatedAt,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to scan procedure", err)
		}

		procedure.Category = category.String
		procedure.Description = description.String

		procedures = append(procedures, procedure)
	}

	return procedures, nil
}

// FacilityProcedureAdapter implements FacilityProcedureRepository
type FacilityProcedureAdapter struct {
	client *postgres.Client
	db     *goqu.Database
}

// NewFacilityProcedureAdapter creates a new facility procedure adapter
func NewFacilityProcedureAdapter(client *postgres.Client) repositories.FacilityProcedureRepository {
	return &FacilityProcedureAdapter{
		client: client,
		db:     goqu.New("postgres", client.DB()),
	}
}

// Create creates a new facility procedure
func (a *FacilityProcedureAdapter) Create(ctx context.Context, fp *entities.FacilityProcedure) error {
	record := goqu.Record{
		"id":                 fp.ID,
		"facility_id":        fp.FacilityID,
		"procedure_id":       fp.ProcedureID,
		"price":              fp.Price,
		"currency":           fp.Currency,
		"estimated_duration": fp.EstimatedDuration,
		"is_available":       fp.IsAvailable,
		"created_at":         fp.CreatedAt,
		"updated_at":         fp.UpdatedAt,
	}

	query, args, err := a.db.Insert("facility_procedures").Rows(record).ToSQL()
	if err != nil {
		return apperrors.NewInternalError("failed to build insert query", err)
	}

	_, err = a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to create facility procedure", err)
	}

	return nil
}

// GetByID retrieves a facility procedure by ID
func (a *FacilityProcedureAdapter) GetByID(ctx context.Context, id string) (*entities.FacilityProcedure, error) {
	query, args, err := a.db.Select(
		"id", "facility_id", "procedure_id", "price", "currency",
		"estimated_duration", "is_available", "created_at", "updated_at",
	).From("facility_procedures").
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	return a.scanFacilityProcedure(ctx, query, args...)
}

// GetByFacilityAndProcedure retrieves pricing for a specific facility and procedure
func (a *FacilityProcedureAdapter) GetByFacilityAndProcedure(ctx context.Context, facilityID, procedureID string) (*entities.FacilityProcedure, error) {
	query, args, err := a.db.Select(
		"id", "facility_id", "procedure_id", "price", "currency",
		"estimated_duration", "is_available", "created_at", "updated_at",
	).From("facility_procedures").
		Where(goqu.Ex{
			"facility_id":  facilityID,
			"procedure_id": procedureID,
		}).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	return a.scanFacilityProcedure(ctx, query, args...)
}

func (a *FacilityProcedureAdapter) scanFacilityProcedure(ctx context.Context, query string, args ...interface{}) (*entities.FacilityProcedure, error) {
	fp := &entities.FacilityProcedure{}
	
	err := a.client.DB().QueryRowContext(ctx, query, args...).Scan(
		&fp.ID,
		&fp.FacilityID,
		&fp.ProcedureID,
		&fp.Price,
		&fp.Currency,
		&fp.EstimatedDuration,
		&fp.IsAvailable,
		&fp.CreatedAt,
		&fp.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, apperrors.NewNotFoundError("facility procedure not found")
	}
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get facility procedure", err)
	}

	return fp, nil
}

// ListByFacility retrieves all procedures for a facility
func (a *FacilityProcedureAdapter) ListByFacility(ctx context.Context, facilityID string) ([]*entities.FacilityProcedure, error) {
	query, args, err := a.db.Select(
		"id", "facility_id", "procedure_id", "price", "currency",
		"estimated_duration", "is_available", "created_at", "updated_at",
	).From("facility_procedures").
		Where(goqu.Ex{"facility_id": facilityID, "is_available": true}).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build list query", err)
	}

	rows, err := a.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list facility procedures", err)
	}
	defer rows.Close()

	var fps []*entities.FacilityProcedure
	for rows.Next() {
		fp := &entities.FacilityProcedure{}
		err := rows.Scan(
			&fp.ID,
			&fp.FacilityID,
			&fp.ProcedureID,
			&fp.Price,
			&fp.Currency,
			&fp.EstimatedDuration,
			&fp.IsAvailable,
			&fp.CreatedAt,
			&fp.UpdatedAt,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to scan facility procedure", err)
		}
		fps = append(fps, fp)
	}

	return fps, nil
}

// Update updates a facility procedure
func (a *FacilityProcedureAdapter) Update(ctx context.Context, fp *entities.FacilityProcedure) error {
	fp.UpdatedAt = time.Now()

	record := goqu.Record{
		"price":              fp.Price,
		"currency":           fp.Currency,
		"estimated_duration": fp.EstimatedDuration,
		"is_available":       fp.IsAvailable,
		"updated_at":         fp.UpdatedAt,
	}

	query, args, err := a.db.Update("facility_procedures").
		Set(record).
		Where(goqu.Ex{"id": fp.ID}).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build update query", err)
	}

	result, err := a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to update facility procedure", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("facility procedure with id %s not found", fp.ID))
	}

	return nil
}

// Delete deletes a facility procedure
func (a *FacilityProcedureAdapter) Delete(ctx context.Context, id string) error {
	query, args, err := a.db.Delete("facility_procedures").
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build delete query", err)
	}

	result, err := a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to delete facility procedure", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("facility procedure with id %s not found", id))
	}

	return nil
}
