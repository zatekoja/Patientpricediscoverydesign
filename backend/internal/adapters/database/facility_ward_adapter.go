package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
)

// FacilityWardAdapter implements the FacilityWardRepository interface
type FacilityWardAdapter struct {
	client *postgres.Client
	db     *goqu.Database
}

// NewFacilityWardAdapter creates a new facility ward adapter
func NewFacilityWardAdapter(client *postgres.Client) repositories.FacilityWardRepository {
	return &FacilityWardAdapter{
		client: client,
		db:     goqu.New("postgres", client.DB()),
	}
}

// Create creates a new facility ward
func (a *FacilityWardAdapter) Create(ctx context.Context, ward *entities.FacilityWard) error {
	record := goqu.Record{
		"id":                     ward.ID,
		"facility_id":            ward.FacilityID,
		"ward_name":              ward.WardName,
		"ward_type":              sql.NullString{String: func() string { if ward.WardType != nil { return *ward.WardType } return "" }(), Valid: ward.WardType != nil && *ward.WardType != ""},
		"capacity_status":        sql.NullString{String: func() string { if ward.CapacityStatus != nil { return *ward.CapacityStatus } return "" }(), Valid: ward.CapacityStatus != nil && *ward.CapacityStatus != ""},
		"avg_wait_minutes":       sql.NullInt64{Int64: func() int64 { if ward.AvgWaitMinutes != nil { return int64(*ward.AvgWaitMinutes) } return 0 }(), Valid: ward.AvgWaitMinutes != nil},
		"urgent_care_available":  sql.NullBool{Bool: func() bool { if ward.UrgentCareAvailable != nil { return *ward.UrgentCareAvailable } return false }(), Valid: ward.UrgentCareAvailable != nil},
		"last_updated":           ward.LastUpdated,
		"created_at":            ward.CreatedAt,
	}

	query, args, err := a.db.Insert("facility_wards").Rows(record).ToSQL()
	if err != nil {
		return apperrors.NewInternalError("failed to build insert query", err)
	}

	_, err = a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to create facility ward", err)
	}

	return nil
}

// GetByID retrieves a facility ward by ID
func (a *FacilityWardAdapter) GetByID(ctx context.Context, id string) (*entities.FacilityWard, error) {
	query, args, err := a.db.Select(
		"id", "facility_id", "ward_name", "ward_type",
		"capacity_status", "avg_wait_minutes", "urgent_care_available",
		"last_updated", "created_at",
	).From("facility_wards").
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	ward := &entities.FacilityWard{}
	var wardType, capacityStatus sql.NullString
	var avgWaitMinutes sql.NullInt64
	var urgentCareAvailable sql.NullBool

	err = a.client.DB().QueryRowContext(ctx, query, args...).Scan(
		&ward.ID,
		&ward.FacilityID,
		&ward.WardName,
		&wardType,
		&capacityStatus,
		&avgWaitMinutes,
		&urgentCareAvailable,
		&ward.LastUpdated,
		&ward.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, apperrors.NewNotFoundError(fmt.Sprintf("facility ward with id %s not found", id))
	}
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get facility ward", err)
	}

	// Map nullable fields
	if wardType.Valid {
		value := wardType.String
		ward.WardType = &value
	}
	if capacityStatus.Valid {
		value := capacityStatus.String
		ward.CapacityStatus = &value
	}
	if avgWaitMinutes.Valid {
		value := int(avgWaitMinutes.Int64)
		ward.AvgWaitMinutes = &value
	}
	if urgentCareAvailable.Valid {
		value := urgentCareAvailable.Bool
		ward.UrgentCareAvailable = &value
	}

	return ward, nil
}

// GetByFacilityID retrieves all wards for a facility
func (a *FacilityWardAdapter) GetByFacilityID(ctx context.Context, facilityID string) ([]*entities.FacilityWard, error) {
	query, args, err := a.db.Select(
		"id", "facility_id", "ward_name", "ward_type",
		"capacity_status", "avg_wait_minutes", "urgent_care_available",
		"last_updated", "created_at",
	).From("facility_wards").
		Where(goqu.Ex{"facility_id": facilityID}).
		Order(goqu.I("ward_name").Asc()).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	rows, err := a.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get facility wards", err)
	}
	defer rows.Close()

	var wards []*entities.FacilityWard
	for rows.Next() {
		ward := &entities.FacilityWard{}
		var wardType, capacityStatus sql.NullString
		var avgWaitMinutes sql.NullInt64
		var urgentCareAvailable sql.NullBool

		err := rows.Scan(
			&ward.ID,
			&ward.FacilityID,
			&ward.WardName,
			&wardType,
			&capacityStatus,
			&avgWaitMinutes,
			&urgentCareAvailable,
			&ward.LastUpdated,
			&ward.CreatedAt,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to scan facility ward", err)
		}

		// Map nullable fields
		if wardType.Valid {
			value := wardType.String
			ward.WardType = &value
		}
		if capacityStatus.Valid {
			value := capacityStatus.String
			ward.CapacityStatus = &value
		}
		if avgWaitMinutes.Valid {
			value := int(avgWaitMinutes.Int64)
			ward.AvgWaitMinutes = &value
		}
		if urgentCareAvailable.Valid {
			value := urgentCareAvailable.Bool
			ward.UrgentCareAvailable = &value
		}

		wards = append(wards, ward)
	}

	return wards, nil
}

// GetByFacilityAndWard retrieves a specific ward by facility ID and ward name
func (a *FacilityWardAdapter) GetByFacilityAndWard(ctx context.Context, facilityID, wardName string) (*entities.FacilityWard, error) {
	query, args, err := a.db.Select(
		"id", "facility_id", "ward_name", "ward_type",
		"capacity_status", "avg_wait_minutes", "urgent_care_available",
		"last_updated", "created_at",
	).From("facility_wards").
		Where(goqu.Ex{
			"facility_id": facilityID,
			"ward_name":  wardName,
		}).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	ward := &entities.FacilityWard{}
	var wardType, capacityStatus sql.NullString
	var avgWaitMinutes sql.NullInt64
	var urgentCareAvailable sql.NullBool

	err = a.client.DB().QueryRowContext(ctx, query, args...).Scan(
		&ward.ID,
		&ward.FacilityID,
		&ward.WardName,
		&wardType,
		&capacityStatus,
		&avgWaitMinutes,
		&urgentCareAvailable,
		&ward.LastUpdated,
		&ward.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, apperrors.NewNotFoundError(fmt.Sprintf("facility ward %s for facility %s not found", wardName, facilityID))
	}
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get facility ward", err)
	}

	// Map nullable fields
	if wardType.Valid {
		value := wardType.String
		ward.WardType = &value
	}
	if capacityStatus.Valid {
		value := capacityStatus.String
		ward.CapacityStatus = &value
	}
	if avgWaitMinutes.Valid {
		value := int(avgWaitMinutes.Int64)
		ward.AvgWaitMinutes = &value
	}
	if urgentCareAvailable.Valid {
		value := urgentCareAvailable.Bool
		ward.UrgentCareAvailable = &value
	}

	return ward, nil
}

// Update updates a facility ward
func (a *FacilityWardAdapter) Update(ctx context.Context, ward *entities.FacilityWard) error {
	ward.LastUpdated = time.Now()

	record := goqu.Record{
		"ward_name":              ward.WardName,
		"ward_type":              sql.NullString{String: func() string { if ward.WardType != nil { return *ward.WardType } return "" }(), Valid: ward.WardType != nil && *ward.WardType != ""},
		"capacity_status":        sql.NullString{String: func() string { if ward.CapacityStatus != nil { return *ward.CapacityStatus } return "" }(), Valid: ward.CapacityStatus != nil && *ward.CapacityStatus != ""},
		"avg_wait_minutes":       sql.NullInt64{Int64: func() int64 { if ward.AvgWaitMinutes != nil { return int64(*ward.AvgWaitMinutes) } return 0 }(), Valid: ward.AvgWaitMinutes != nil},
		"urgent_care_available":   sql.NullBool{Bool: func() bool { if ward.UrgentCareAvailable != nil { return *ward.UrgentCareAvailable } return false }(), Valid: ward.UrgentCareAvailable != nil},
		"last_updated":           ward.LastUpdated,
	}

	query, args, err := a.db.Update("facility_wards").
		Set(record).
		Where(goqu.Ex{"id": ward.ID}).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build update query", err)
	}

	result, err := a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to update facility ward", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("facility ward with id %s not found", ward.ID))
	}

	return nil
}

// Upsert creates or updates a facility ward (inserts if not exists, updates if exists)
func (a *FacilityWardAdapter) Upsert(ctx context.Context, ward *entities.FacilityWard) error {
	// Try to get existing ward by facility_id and ward_name
	existing, err := a.GetByFacilityAndWard(ctx, ward.FacilityID, ward.WardName)
	if err != nil {
		// If not found, create new
		if apperrors.IsNotFound(err) {
			// Generate ID if not provided
			if ward.ID == "" {
				ward.ID = fmt.Sprintf("%s-%s", ward.FacilityID, ward.WardName)
			}
			if ward.CreatedAt.IsZero() {
				ward.CreatedAt = time.Now()
			}
			if ward.LastUpdated.IsZero() {
				ward.LastUpdated = time.Now()
			}
			return a.Create(ctx, ward)
		}
		// Other error, return it
		return err
	}

	// Update existing ward
	ward.ID = existing.ID
	ward.CreatedAt = existing.CreatedAt
	return a.Update(ctx, ward)
}

// Delete deletes a facility ward
func (a *FacilityWardAdapter) Delete(ctx context.Context, id string) error {
	query, args, err := a.db.Delete("facility_wards").
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build delete query", err)
	}

	result, err := a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to delete facility ward", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("facility ward with id %s not found", id))
	}

	return nil
}

// DeleteByFacilityAndWard deletes a ward by facility ID and ward name
func (a *FacilityWardAdapter) DeleteByFacilityAndWard(ctx context.Context, facilityID, wardName string) error {
	query, args, err := a.db.Delete("facility_wards").
		Where(goqu.Ex{
			"facility_id": facilityID,
			"ward_name":  wardName,
		}).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build delete query", err)
	}

	result, err := a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to delete facility ward", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("facility ward %s for facility %s not found", wardName, facilityID))
	}

	return nil
}
