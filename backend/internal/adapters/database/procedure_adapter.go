package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/lib/pq"
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
		"id":              procedure.ID,
		"name":            procedure.Name,
		"display_name":    procedure.DisplayName,
		"code":            procedure.Code,
		"category":        sql.NullString{String: procedure.Category, Valid: procedure.Category != ""},
		"description":     sql.NullString{String: procedure.Description, Valid: procedure.Description != ""},
		"normalized_tags": pq.Array(procedure.NormalizedTags),
		"is_active":       procedure.IsActive,
		"created_at":      procedure.CreatedAt,
		"updated_at":      procedure.UpdatedAt,
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

// GetByIDs retrieves multiple procedures by their IDs
func (a *ProcedureAdapter) GetByIDs(ctx context.Context, ids []string) ([]*entities.Procedure, error) {
	if len(ids) == 0 {
		return []*entities.Procedure{}, nil
	}

	query, args, err := a.db.Select(
		"id", "name", "display_name", "code", "category", "description",
		"normalized_tags", "is_active", "created_at", "updated_at",
	).From("procedures").
		Where(goqu.Ex{"id": ids}).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	rows, err := a.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get procedures by ids", err)
	}
	defer rows.Close()

	var procedures []*entities.Procedure
	for rows.Next() {
		procedure := &entities.Procedure{}
		var category, description sql.NullString

		err := rows.Scan(
			&procedure.ID,
			&procedure.Name,
			&procedure.DisplayName,
			&procedure.Code,
			&category,
			&description,
			pq.Array(&procedure.NormalizedTags),
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

func (a *ProcedureAdapter) getByField(ctx context.Context, field, value string) (*entities.Procedure, error) {
	query, args, err := a.db.Select(
		"id", "name", "display_name", "code", "category", "description",
		"normalized_tags", "is_active", "created_at", "updated_at",
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
		&procedure.DisplayName,
		&procedure.Code,
		&category,
		&description,
		pq.Array(&procedure.NormalizedTags),
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
		"name":            procedure.Name,
		"display_name":    procedure.DisplayName,
		"code":            procedure.Code,
		"category":        sql.NullString{String: procedure.Category, Valid: procedure.Category != ""},
		"description":     sql.NullString{String: procedure.Description, Valid: procedure.Description != ""},
		"normalized_tags": pq.Array(procedure.NormalizedTags),
		"is_active":       procedure.IsActive,
		"updated_at":      procedure.UpdatedAt,
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
		"id", "name", "display_name", "code", "category", "description",
		"normalized_tags", "is_active", "created_at", "updated_at",
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
			&procedure.DisplayName,
			&procedure.Code,
			&category,
			&description,
			pq.Array(&procedure.NormalizedTags),
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

// ListByFacility retrieves all procedures for a facility (including unavailable ones marked with isAvailable=false)
// Services with IsAvailable=false will be returned so they can be displayed as "grayed out" on the frontend
func (a *FacilityProcedureAdapter) ListByFacility(ctx context.Context, facilityID string) ([]*entities.FacilityProcedure, error) {
	query, args, err := a.db.Select(
		"id", "facility_id", "procedure_id", "price", "currency",
		"estimated_duration", "is_available", "created_at", "updated_at",
	).From("facility_procedures").
		Where(goqu.Ex{"facility_id": facilityID}).
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

// ListByFacilityWithCount implements TDD-driven search-first pagination
// CRITICAL: This method searches the ENTIRE dataset first, then applies pagination
// This ensures users can find all relevant results across all pages
func (a *FacilityProcedureAdapter) ListByFacilityWithCount(
	ctx context.Context,
	facilityID string,
	filter repositories.FacilityProcedureFilter,
) ([]*entities.FacilityProcedure, int, error) {

	// Step 1: Build base query with JOIN to procedures table for search capability
	baseQuery := a.db.From(
		a.db.Select(
			goqu.L("fp.*"),
			goqu.COALESCE(goqu.I("p.display_name"), goqu.I("p.name")).As("procedure_name"),
			goqu.I("p.display_name").As("procedure_display_name"),
			goqu.I("p.code").As("procedure_code"),
			goqu.I("p.category").As("category"),
			goqu.I("p.description").As("description"),
			goqu.I("p.normalized_tags").As("procedure_normalized_tags"),
		).
			From(goqu.T("facility_procedures").As("fp")).
			Join(goqu.T("procedures").As("p"), goqu.On(goqu.I("fp.procedure_id").Eq(goqu.I("p.id")))).
			Where(goqu.I("fp.facility_id").Eq(facilityID)).
			Where(goqu.I("p.is_active").Eq(true)).
			As("base_data"),
	)

	// Step 2: Apply ALL filters to entire dataset BEFORE pagination
	filteredQuery := baseQuery

	// Availability filter
	if filter.IsAvailable != nil {
		filteredQuery = filteredQuery.Where(goqu.I("is_available").Eq(*filter.IsAvailable))
	}

	// Category filter
	if filter.Category != "" {
		filteredQuery = filteredQuery.Where(goqu.I("category").ILike(fmt.Sprintf("%%%s%%", filter.Category)))
	}

	// Price range filters
	if filter.MinPrice != nil {
		filteredQuery = filteredQuery.Where(goqu.I("price").Gte(*filter.MinPrice))
	}
	if filter.MaxPrice != nil {
		filteredQuery = filteredQuery.Where(goqu.I("price").Lte(*filter.MaxPrice))
	}

	// CRITICAL: Search query applied to ENTIRE dataset first
	if filter.SearchQuery != "" {
		searchPattern := fmt.Sprintf("%%%s%%", filter.SearchQuery)
		
		orConditions := []goqu.Expression{
			goqu.I("procedure_name").ILike(searchPattern),
			goqu.I("description").ILike(searchPattern),
		}
		
		if len(filter.SearchTerms) > 0 {
			orConditions = append(orConditions, goqu.L("procedure_normalized_tags && ?", pq.Array(filter.SearchTerms)))
		}
		
		filteredQuery = filteredQuery.Where(goqu.Or(orConditions...))
	}

	// Step 3: Get total count of filtered results (before pagination)
	countQuery := filteredQuery.Select(goqu.COUNT("*"))
	countSQL, countArgs, err := countQuery.ToSQL()
	if err != nil {
		return nil, 0, apperrors.NewInternalError("failed to build count query", err)
	}

	var totalCount int
	err = a.client.DB().QueryRowContext(ctx, countSQL, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, apperrors.NewInternalError("failed to count filtered procedures", err)
	}

	// Step 4: Apply sorting to filtered data
	sortedQuery := filteredQuery.Select(
		"id", "facility_id", "procedure_id", "price", "currency",
		"estimated_duration", "is_available", "created_at", "updated_at",
		"procedure_name", "procedure_display_name", "procedure_code", "category", "description", "procedure_normalized_tags",
	)

	if filter.SortBy != "" {
		switch filter.SortBy {
		case "name":
			if filter.SortOrder == "desc" {
				sortedQuery = sortedQuery.Order(goqu.I("procedure_name").Desc())
			} else {
				sortedQuery = sortedQuery.Order(goqu.I("procedure_name").Asc())
			}
		case "category":
			if filter.SortOrder == "desc" {
				sortedQuery = sortedQuery.Order(goqu.I("category").Desc())
			} else {
				sortedQuery = sortedQuery.Order(goqu.I("category").Asc())
			}
		case "updated_at":
			if filter.SortOrder == "desc" {
				sortedQuery = sortedQuery.Order(goqu.I("updated_at").Desc())
			} else {
				sortedQuery = sortedQuery.Order(goqu.I("updated_at").Asc())
			}
		case "price":
			if filter.SortOrder == "desc" {
				sortedQuery = sortedQuery.Order(goqu.I("price").Desc())
			} else {
				sortedQuery = sortedQuery.Order(goqu.I("price").Asc())
			}
		default:
			// default to price ascending
			sortedQuery = sortedQuery.Order(goqu.I("price").Asc())
		}
	} else {
		// Default sort by price ascending
		sortedQuery = sortedQuery.Order(goqu.I("price").Asc())
	}

	// Step 5: Apply pagination LAST (after search, filter, sort)
	if filter.Limit > 0 {
		sortedQuery = sortedQuery.Limit(uint(filter.Limit))
	}
	if filter.Offset > 0 {
		sortedQuery = sortedQuery.Offset(uint(filter.Offset))
	}

	// Step 6: Execute final query
	finalSQL, finalArgs, err := sortedQuery.ToSQL()
	if err != nil {
		return nil, 0, apperrors.NewInternalError("failed to build final query", err)
	}

	rows, err := a.client.DB().QueryContext(ctx, finalSQL, finalArgs...)
	if err != nil {
		return nil, 0, apperrors.NewInternalError("failed to execute paginated query", err)
	}
	defer rows.Close()

	var procedures []*entities.FacilityProcedure
	for rows.Next() {
		fp := &entities.FacilityProcedure{}
		var procName, procDisplayName, procCode, procCategory, procDescription sql.NullString
		var procTags []string
		err := rows.Scan(
			&fp.ID, &fp.FacilityID, &fp.ProcedureID, &fp.Price, &fp.Currency,
			&fp.EstimatedDuration, &fp.IsAvailable, &fp.CreatedAt, &fp.UpdatedAt,
			&procName, &procDisplayName, &procCode, &procCategory, &procDescription, pq.Array(&procTags),
		)
		if err != nil {
			return nil, 0, apperrors.NewInternalError("failed to scan facility procedure", err)
		}
		fp.ProcedureName = procName.String
		fp.ProcedureDisplayName = procDisplayName.String
		fp.ProcedureCode = procCode.String
		fp.ProcedureCategory = procCategory.String
		fp.ProcedureDescription = procDescription.String
		fp.ProcedureTags = procTags
		procedures = append(procedures, fp)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, apperrors.NewInternalError("error iterating procedures", err)
	}

	// Audit log filtering behavior for debugging
	logFilteringAudit(facilityID, filter, totalCount, len(procedures))

	return procedures, totalCount, nil
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

// logFilteringAudit logs information about service filtering for debugging
// This helps understand why certain services are excluded from results
func logFilteringAudit(facilityID string, filter repositories.FacilityProcedureFilter, totalCount, returnedCount int) {
	auditLog := fmt.Sprintf(
		"FILTER_AUDIT [FacilityID=%s] Total matching: %d, Returned (after pagination): %d | "+
			"Category: %v, MinPrice: %v, MaxPrice: %v, IsAvailable: %v, Search: %q, "+
			"Sort: %s %s, Limit: %d, Offset: %d",
		facilityID,
		totalCount,
		returnedCount,
		filter.Category,
		filter.MinPrice,
		filter.MaxPrice,
		filter.IsAvailable,
		filter.SearchQuery,
		filter.SortBy,
		filter.SortOrder,
		filter.Limit,
		filter.Offset,
	)
	log.Println(auditLog)
}
