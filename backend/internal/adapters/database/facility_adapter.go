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

// FacilityAdapter implements the FacilityRepository interface
type FacilityAdapter struct {
	client *postgres.Client
	db     *goqu.Database
}

// NewFacilityAdapter creates a new facility adapter
func NewFacilityAdapter(client *postgres.Client) repositories.FacilityRepository {
	return &FacilityAdapter{
		client: client,
		db:     goqu.New("postgres", client.DB()),
	}
}

// Create creates a new facility
func (a *FacilityAdapter) Create(ctx context.Context, facility *entities.Facility) error {
	record := goqu.Record{
		"id":            facility.ID,
		"name":          facility.Name,
		"street":        sql.NullString{String: facility.Address.Street, Valid: facility.Address.Street != ""},
		"city":          sql.NullString{String: facility.Address.City, Valid: facility.Address.City != ""},
		"state":         sql.NullString{String: facility.Address.State, Valid: facility.Address.State != ""},
		"zip_code":      sql.NullString{String: facility.Address.ZipCode, Valid: facility.Address.ZipCode != ""},
		"country":       sql.NullString{String: facility.Address.Country, Valid: facility.Address.Country != ""},
		"latitude":      facility.Location.Latitude,
		"longitude":     facility.Location.Longitude,
		"phone_number":  sql.NullString{String: facility.PhoneNumber, Valid: facility.PhoneNumber != ""},
		"email":         sql.NullString{String: facility.Email, Valid: facility.Email != ""},
		"website":       sql.NullString{String: facility.Website, Valid: facility.Website != ""},
		"description":   sql.NullString{String: facility.Description, Valid: facility.Description != ""},
		"facility_type": sql.NullString{String: facility.FacilityType, Valid: facility.FacilityType != ""},
		"capacity_status": sql.NullString{
			String: func() string {
				if facility.CapacityStatus != nil {
					return *facility.CapacityStatus
				}
				return ""
			}(),
			Valid: facility.CapacityStatus != nil && *facility.CapacityStatus != "",
		},
		"avg_wait_minutes": sql.NullInt64{
			Int64: func() int64 {
				if facility.AvgWaitMinutes != nil {
					return int64(*facility.AvgWaitMinutes)
				}
				return 0
			}(),
			Valid: facility.AvgWaitMinutes != nil,
		},
		"urgent_care_available": sql.NullBool{
			Bool: func() bool {
				if facility.UrgentCareAvailable != nil {
					return *facility.UrgentCareAvailable
				}
				return false
			}(),
			Valid: facility.UrgentCareAvailable != nil,
		},
		"rating":       facility.Rating,
		"review_count": facility.ReviewCount,
		"is_active":    facility.IsActive,
		"created_at":   facility.CreatedAt,
		"updated_at":   facility.UpdatedAt,
	}

	query, args, err := a.db.Insert("facilities").Rows(record).ToSQL()
	if err != nil {
		return apperrors.NewInternalError("failed to build insert query", err)
	}

	_, err = a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to create facility", err)
	}

	return nil
}

// GetByID retrieves a facility by ID
func (a *FacilityAdapter) GetByID(ctx context.Context, id string) (*entities.Facility, error) {
	query, args, err := a.db.Select(
		"id", "name", "street", "city", "state", "zip_code", "country",
		"latitude", "longitude", "phone_number", "email", "website",
		"description", "facility_type", "capacity_status", "avg_wait_minutes", "urgent_care_available", "rating", "review_count",
		"is_active", "created_at", "updated_at",
	).From("facilities").
		Where(goqu.Ex{"id": id, "is_active": true}).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	facility := &entities.Facility{}
	var street, city, state, zipCode, country sql.NullString
	var phoneNumber, email, website, description, facilityType sql.NullString
	var capacityStatus sql.NullString
	var avgWaitMinutes sql.NullInt64
	var urgentCareAvailable sql.NullBool

	err = a.client.DB().QueryRowContext(ctx, query, args...).Scan(
		&facility.ID,
		&facility.Name,
		&street,
		&city,
		&state,
		&zipCode,
		&country,
		&facility.Location.Latitude,
		&facility.Location.Longitude,
		&phoneNumber,
		&email,
		&website,
		&description,
		&facilityType,
		&capacityStatus,
		&avgWaitMinutes,
		&urgentCareAvailable,
		&facility.Rating,
		&facility.ReviewCount,
		&facility.IsActive,
		&facility.CreatedAt,
		&facility.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, apperrors.NewNotFoundError(fmt.Sprintf("facility with id %s not found", id))
	}
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get facility", err)
	}

	// Map nullable fields
	facility.Address.Street = street.String
	facility.Address.City = city.String
	facility.Address.State = state.String
	facility.Address.ZipCode = zipCode.String
	facility.Address.Country = country.String
	facility.PhoneNumber = phoneNumber.String
	facility.Email = email.String
	facility.Website = website.String
	facility.Description = description.String
	facility.FacilityType = facilityType.String
	if capacityStatus.Valid {
		value := capacityStatus.String
		facility.CapacityStatus = &value
	}
	if avgWaitMinutes.Valid {
		value := int(avgWaitMinutes.Int64)
		facility.AvgWaitMinutes = &value
	}
	if urgentCareAvailable.Valid {
		value := urgentCareAvailable.Bool
		facility.UrgentCareAvailable = &value
	}

	return facility, nil
}

// Update updates a facility
func (a *FacilityAdapter) Update(ctx context.Context, facility *entities.Facility) error {
	facility.UpdatedAt = time.Now()

	record := goqu.Record{
		"name":          facility.Name,
		"street":        sql.NullString{String: facility.Address.Street, Valid: facility.Address.Street != ""},
		"city":          sql.NullString{String: facility.Address.City, Valid: facility.Address.City != ""},
		"state":         sql.NullString{String: facility.Address.State, Valid: facility.Address.State != ""},
		"zip_code":      sql.NullString{String: facility.Address.ZipCode, Valid: facility.Address.ZipCode != ""},
		"country":       sql.NullString{String: facility.Address.Country, Valid: facility.Address.Country != ""},
		"latitude":      facility.Location.Latitude,
		"longitude":     facility.Location.Longitude,
		"phone_number":  sql.NullString{String: facility.PhoneNumber, Valid: facility.PhoneNumber != ""},
		"email":         sql.NullString{String: facility.Email, Valid: facility.Email != ""},
		"website":       sql.NullString{String: facility.Website, Valid: facility.Website != ""},
		"description":   sql.NullString{String: facility.Description, Valid: facility.Description != ""},
		"facility_type": sql.NullString{String: facility.FacilityType, Valid: facility.FacilityType != ""},
		"rating":        facility.Rating,
		"review_count":  facility.ReviewCount,
		"is_active":     facility.IsActive,
		"updated_at":    facility.UpdatedAt,
	}

	query, args, err := a.db.Update("facilities").
		Set(record).
		Where(goqu.Ex{"id": facility.ID}).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build update query", err)
	}

	result, err := a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to update facility", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("facility with id %s not found", facility.ID))
	}

	return nil
}

// GetByIDs retrieves multiple facilities by their IDs
func (a *FacilityAdapter) GetByIDs(ctx context.Context, ids []string) ([]*entities.Facility, error) {
	if len(ids) == 0 {
		return []*entities.Facility{}, nil
	}

	query, args, err := a.db.Select(
		"id", "name", "street", "city", "state", "zip_code", "country",
		"latitude", "longitude", "phone_number", "email", "website",
		"description", "facility_type", "capacity_status", "avg_wait_minutes", "urgent_care_available", "rating", "review_count",
		"is_active", "created_at", "updated_at",
	).From("facilities").
		Where(goqu.Ex{"id": ids, "is_active": true}).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	rows, err := a.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get facilities by ids", err)
	}
	defer rows.Close()

	facilities := []*entities.Facility{}
	for rows.Next() {
		facility := &entities.Facility{}
		var street, city, state, zipCode, country sql.NullString
		var phoneNumber, email, website, description, facilityType sql.NullString
		var capacityStatus sql.NullString
		var avgWaitMinutes sql.NullInt64
		var urgentCareAvailable sql.NullBool

		err := rows.Scan(
			&facility.ID,
			&facility.Name,
			&street,
			&city,
			&state,
			&zipCode,
			&country,
			&facility.Location.Latitude,
			&facility.Location.Longitude,
			&phoneNumber,
			&email,
			&website,
			&description,
			&facilityType,
			&capacityStatus,
			&avgWaitMinutes,
			&urgentCareAvailable,
			&facility.Rating,
			&facility.ReviewCount,
			&facility.IsActive,
			&facility.CreatedAt,
			&facility.UpdatedAt,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to scan facility", err)
		}

		// Map nullable fields
		facility.Address.Street = street.String
		facility.Address.City = city.String
		facility.Address.State = state.String
		facility.Address.ZipCode = zipCode.String
		facility.Address.Country = country.String
		facility.PhoneNumber = phoneNumber.String
		facility.Email = email.String
		facility.Website = website.String
		facility.Description = description.String
		facility.FacilityType = facilityType.String
		if capacityStatus.Valid {
			value := capacityStatus.String
			facility.CapacityStatus = &value
		}
		if avgWaitMinutes.Valid {
			value := int(avgWaitMinutes.Int64)
			facility.AvgWaitMinutes = &value
		}
		if urgentCareAvailable.Valid {
			value := urgentCareAvailable.Bool
			facility.UrgentCareAvailable = &value
		}

		facilities = append(facilities, facility)
	}

	return facilities, nil
}

// Delete deletes a facility (soft delete)
func (a *FacilityAdapter) Delete(ctx context.Context, id string) error {
	query, args, err := a.db.Update("facilities").
		Set(goqu.Record{"is_active": false, "updated_at": time.Now()}).
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build delete query", err)
	}

	result, err := a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to delete facility", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("facility with id %s not found", id))
	}

	return nil
}

// List retrieves facilities with filters
func (a *FacilityAdapter) List(ctx context.Context, filter repositories.FacilityFilter) ([]*entities.Facility, error) {
	ds := a.db.Select(
		"id", "name", "street", "city", "state", "zip_code", "country",
		"latitude", "longitude", "phone_number", "email", "website",
		"description", "facility_type", "capacity_status", "avg_wait_minutes", "urgent_care_available", "rating", "review_count",
		"is_active", "created_at", "updated_at",
	).From("facilities")

	if filter.FacilityType != "" {
		ds = ds.Where(goqu.Ex{"facility_type": filter.FacilityType})
	}

	if filter.IsActive != nil {
		ds = ds.Where(goqu.Ex{"is_active": *filter.IsActive})
	}

	ds = ds.Order(goqu.I("created_at").Desc())

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
		return nil, apperrors.NewInternalError("failed to list facilities", err)
	}
	defer rows.Close()

	facilities := []*entities.Facility{}
	for rows.Next() {
		facility := &entities.Facility{}
		var street, city, state, zipCode, country sql.NullString
		var phoneNumber, email, website, description, facilityType sql.NullString
		var capacityStatus sql.NullString
		var avgWaitMinutes sql.NullInt64
		var urgentCareAvailable sql.NullBool

		err := rows.Scan(
			&facility.ID,
			&facility.Name,
			&street,
			&city,
			&state,
			&zipCode,
			&country,
			&facility.Location.Latitude,
			&facility.Location.Longitude,
			&phoneNumber,
			&email,
			&website,
			&description,
			&facilityType,
			&capacityStatus,
			&avgWaitMinutes,
			&urgentCareAvailable,
			&facility.Rating,
			&facility.ReviewCount,
			&facility.IsActive,
			&facility.CreatedAt,
			&facility.UpdatedAt,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to scan facility", err)
		}

		// Map nullable fields
		facility.Address.Street = street.String
		facility.Address.City = city.String
		facility.Address.State = state.String
		facility.Address.ZipCode = zipCode.String
		facility.Address.Country = country.String
		facility.PhoneNumber = phoneNumber.String
		facility.Email = email.String
		facility.Website = website.String
		facility.Description = description.String
		facility.FacilityType = facilityType.String
		if capacityStatus.Valid {
			value := capacityStatus.String
			facility.CapacityStatus = &value
		}
		if avgWaitMinutes.Valid {
			value := int(avgWaitMinutes.Int64)
			facility.AvgWaitMinutes = &value
		}
		if urgentCareAvailable.Valid {
			value := urgentCareAvailable.Bool
			facility.UrgentCareAvailable = &value
		}

		facilities = append(facilities, facility)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternalError("error iterating facilities", err)
	}

	return facilities, nil
}

// Search searches facilities by location and criteria
func (a *FacilityAdapter) Search(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, error) {
	// Use a subquery to calculate distance, then filter in outer WHERE clause
	// This avoids the HAVING issue and is more efficient
	distanceExpr := goqu.L(
		"(6371 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude))))",
		params.Latitude, params.Longitude, params.Latitude,
	)

	ds := a.db.Select(
		"id", "name", "street", "city", "state", "zip_code", "country",
		"latitude", "longitude", "phone_number", "email", "website",
		"description", "facility_type", "capacity_status", "avg_wait_minutes", "urgent_care_available", "rating", "review_count",
		"is_active", "created_at", "updated_at",
		distanceExpr.As("distance"),
	).From("facilities").
		Where(goqu.Ex{"is_active": true}).
		Where(distanceExpr.Lte(params.RadiusKm)).
		Order(goqu.I("distance").Asc())

	if params.Limit > 0 {
		ds = ds.Limit(uint(params.Limit))
	}

	if params.Offset > 0 {
		ds = ds.Offset(uint(params.Offset))
	}

	query, args, err := ds.ToSQL()
	if err != nil {
		return nil, apperrors.NewInternalError("failed to build search query", err)
	}

	rows, err := a.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to search facilities", err)
	}
	defer rows.Close()

	facilities := []*entities.Facility{}
	for rows.Next() {
		facility := &entities.Facility{}
		var street, city, state, zipCode, country sql.NullString
		var phoneNumber, email, website, description, facilityType sql.NullString
		var capacityStatus sql.NullString
		var avgWaitMinutes sql.NullInt64
		var urgentCareAvailable sql.NullBool
		var distance float64

		err := rows.Scan(
			&facility.ID,
			&facility.Name,
			&street,
			&city,
			&state,
			&zipCode,
			&country,
			&facility.Location.Latitude,
			&facility.Location.Longitude,
			&phoneNumber,
			&email,
			&website,
			&description,
			&facilityType,
			&capacityStatus,
			&avgWaitMinutes,
			&urgentCareAvailable,
			&facility.Rating,
			&facility.ReviewCount,
			&facility.IsActive,
			&facility.CreatedAt,
			&facility.UpdatedAt,
			&distance,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to scan facility", err)
		}

		// Map nullable fields
		facility.Address.Street = street.String
		facility.Address.City = city.String
		facility.Address.State = state.String
		facility.Address.ZipCode = zipCode.String
		facility.Address.Country = country.String
		facility.PhoneNumber = phoneNumber.String
		facility.Email = email.String
		facility.Website = website.String
		facility.Description = description.String
		facility.FacilityType = facilityType.String
		if capacityStatus.Valid {
			value := capacityStatus.String
			facility.CapacityStatus = &value
		}
		if avgWaitMinutes.Valid {
			value := int(avgWaitMinutes.Int64)
			facility.AvgWaitMinutes = &value
		}
		if urgentCareAvailable.Valid {
			value := urgentCareAvailable.Bool
			facility.UrgentCareAvailable = &value
		}

		facilities = append(facilities, facility)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternalError("error iterating facilities", err)
	}

	return facilities, nil
}
