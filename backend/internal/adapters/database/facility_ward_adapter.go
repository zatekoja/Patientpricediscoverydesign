package database

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"strings"
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
	var wardTypeStr string
	var wardTypeValid bool
	if ward.WardType != nil {
		wardTypeStr = *ward.WardType
		wardTypeValid = *ward.WardType != ""
	}

	var capacityStatusStr string
	var capacityStatusValid bool
	if ward.CapacityStatus != nil {
		capacityStatusStr = *ward.CapacityStatus
		capacityStatusValid = *ward.CapacityStatus != ""
	}

	var avgWaitInt int64
	var avgWaitValid bool
	if ward.AvgWaitMinutes != nil {
		avgWaitInt = int64(*ward.AvgWaitMinutes)
		avgWaitValid = true
	}

	var urgentCareBool bool
	var urgentCareValid bool
	if ward.UrgentCareAvailable != nil {
		urgentCareBool = *ward.UrgentCareAvailable
		urgentCareValid = true
	}

	record := goqu.Record{
		"id":                     ward.ID,
		"facility_id":            ward.FacilityID,
		"ward_name":              ward.WardName,
		"ward_type":              sql.NullString{String: wardTypeStr, Valid: wardTypeValid},
		"capacity_status":        sql.NullString{String: capacityStatusStr, Valid: capacityStatusValid},
		"avg_wait_minutes":       sql.NullInt64{Int64: avgWaitInt, Valid: avgWaitValid},
		"urgent_care_available":  sql.NullBool{Bool: urgentCareBool, Valid: urgentCareValid},
		"last_updated":           ward.LastUpdated,
		"created_at":             ward.CreatedAt,
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

// GetByFacilityIDs retrieves all wards for multiple facilities in a single query
func (a *FacilityWardAdapter) GetByFacilityIDs(ctx context.Context, facilityIDs []string) (map[string][]*entities.FacilityWard, error) {
	if len(facilityIDs) == 0 {
		return make(map[string][]*entities.FacilityWard), nil
	}

	query, args, err := a.db.Select(
		"id", "facility_id", "ward_name", "ward_type",
		"capacity_status", "avg_wait_minutes", "urgent_care_available",
		"last_updated", "created_at",
	).From("facility_wards").
		Where(goqu.Ex{"facility_id": facilityIDs}).
		Order(goqu.I("facility_id").Asc(), goqu.I("ward_name").Asc()).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	rows, err := a.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to query facility wards", err)
	}
	defer rows.Close()

	// Use map to group wards by facility ID
	wardsByFacility := make(map[string][]*entities.FacilityWard)

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

		wardsByFacility[ward.FacilityID] = append(wardsByFacility[ward.FacilityID], ward)
	}

	return wardsByFacility, nil
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

	var wardTypeStr string
	var wardTypeValid bool
	if ward.WardType != nil {
		wardTypeStr = *ward.WardType
		wardTypeValid = *ward.WardType != ""
	}

	var capacityStatusStr string
	var capacityStatusValid bool
	if ward.CapacityStatus != nil {
		capacityStatusStr = *ward.CapacityStatus
		capacityStatusValid = *ward.CapacityStatus != ""
	}

	var avgWaitInt int64
	var avgWaitValid bool
	if ward.AvgWaitMinutes != nil {
		avgWaitInt = int64(*ward.AvgWaitMinutes)
		avgWaitValid = true
	}

	var urgentCareBool bool
	var urgentCareValid bool
	if ward.UrgentCareAvailable != nil {
		urgentCareBool = *ward.UrgentCareAvailable
		urgentCareValid = true
	}

	record := goqu.Record{
		"ward_name":              ward.WardName,
		"ward_type":              sql.NullString{String: wardTypeStr, Valid: wardTypeValid},
		"capacity_status":        sql.NullString{String: capacityStatusStr, Valid: capacityStatusValid},
		"avg_wait_minutes":       sql.NullInt64{Int64: avgWaitInt, Valid: avgWaitValid},
		"urgent_care_available":  sql.NullBool{Bool: urgentCareBool, Valid: urgentCareValid},
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

// hashString generates a short hash for use in IDs (same implementation as provider_ingestion_service)
func hashString(value string) string {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(value))
	return fmt.Sprintf("%x", hasher.Sum32())
}

// Upsert creates or updates a facility ward (inserts if not exists, updates if exists)
func (a *FacilityWardAdapter) Upsert(ctx context.Context, ward *entities.FacilityWard) error {
	// Ensure ID and timestamps are set for insert path
	if ward.ID == "" {
		// Generate ward ID using hash of full facilityID + normalized ward name
		// This ensures uniqueness regardless of facility ID length and avoids collisions
		normalizedWardName := strings.ToLower(strings.TrimSpace(ward.WardName))
		combinedKey := fmt.Sprintf("%s:%s", ward.FacilityID, normalizedWardName)
		fullHash := hashString(combinedKey)
		
		// Use a short readable prefix (first 20 chars) for debugging
		facilityIDPrefix := ward.FacilityID
		if len(ward.FacilityID) > 20 {
			facilityIDPrefix = ward.FacilityID[:20]
		}
		
		ward.ID = fmt.Sprintf("%s_%s", facilityIDPrefix, fullHash)
	}

	now := time.Now()
	if ward.CreatedAt.IsZero() {
		ward.CreatedAt = now
	}
	if ward.LastUpdated.IsZero() {
		ward.LastUpdated = now
	}

	var wardTypeStr string
	var wardTypeValid bool
	if ward.WardType != nil {
		wardTypeStr = *ward.WardType
		wardTypeValid = *ward.WardType != ""
	}

	var capacityStatusStr string
	var capacityStatusValid bool
	if ward.CapacityStatus != nil {
		capacityStatusStr = *ward.CapacityStatus
		capacityStatusValid = *ward.CapacityStatus != ""
	}

	var avgWaitInt int64
	var avgWaitValid bool
	if ward.AvgWaitMinutes != nil {
		avgWaitInt = int64(*ward.AvgWaitMinutes)
		avgWaitValid = true
	}

	var urgentCareBool bool
	var urgentCareValid bool
	if ward.UrgentCareAvailable != nil {
		urgentCareBool = *ward.UrgentCareAvailable
		urgentCareValid = true
	}

	wardTypeNull := sql.NullString{String: wardTypeStr, Valid: wardTypeValid}
	capacityStatusNull := sql.NullString{String: capacityStatusStr, Valid: capacityStatusValid}
	avgWaitNull := sql.NullInt64{Int64: avgWaitInt, Valid: avgWaitValid}
	urgentCareNull := sql.NullBool{Bool: urgentCareBool, Valid: urgentCareValid}

	// Build the insert record
	record := goqu.Record{
		"id":                    ward.ID,
		"facility_id":           ward.FacilityID,
		"ward_name":             ward.WardName,
		"ward_type":             wardTypeNull,
		"capacity_status":       capacityStatusNull,
		"avg_wait_minutes":      avgWaitNull,
		"urgent_care_available": urgentCareNull,
		"last_updated":          ward.LastUpdated,
		"created_at":            ward.CreatedAt,
	}

	// Build update record (exclude id, facility_id, ward_name, and created_at)
	updateRecord := goqu.Record{
		"ward_type":             wardTypeNull,
		"capacity_status":       capacityStatusNull,
		"avg_wait_minutes":      avgWaitNull,
		"urgent_care_available": urgentCareNull,
		"last_updated":          ward.LastUpdated,
	}

	// Use INSERT ... ON CONFLICT for atomic upsert
	// For composite unique key (facility_id, ward_name), pass columns as a slice of identifiers
	query, args, err := a.db.Insert("facility_wards").
		Rows(record).
		OnConflict(goqu.DoUpdate([]interface{}{goqu.I("facility_id"), goqu.I("ward_name")}, updateRecord)).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build upsert query", err)
	}

	_, err = a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to execute upsert", err)
	}

	return nil
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
