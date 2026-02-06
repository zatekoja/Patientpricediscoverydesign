package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
)

// FacilityAdapter implements the FacilityRepository interface
type FacilityAdapter struct {
	client *postgres.Client
}

// NewFacilityAdapter creates a new facility adapter
func NewFacilityAdapter(client *postgres.Client) repositories.FacilityRepository {
	return &FacilityAdapter{
		client: client,
	}
}

// Create creates a new facility
func (a *FacilityAdapter) Create(ctx context.Context, facility *entities.Facility) error {
	query := `
		INSERT INTO facilities (
			id, name, street, city, state, zip_code, country,
			latitude, longitude, phone_number, email, website,
			description, facility_type, rating, review_count,
			is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`

	_, err := a.client.DB().ExecContext(ctx, query,
		facility.ID,
		facility.Name,
		facility.Address.Street,
		facility.Address.City,
		facility.Address.State,
		facility.Address.ZipCode,
		facility.Address.Country,
		facility.Location.Latitude,
		facility.Location.Longitude,
		facility.PhoneNumber,
		facility.Email,
		facility.Website,
		facility.Description,
		facility.FacilityType,
		facility.Rating,
		facility.ReviewCount,
		facility.IsActive,
		facility.CreatedAt,
		facility.UpdatedAt,
	)

	if err != nil {
		return apperrors.NewInternalError("failed to create facility", err)
	}

	return nil
}

// GetByID retrieves a facility by ID
func (a *FacilityAdapter) GetByID(ctx context.Context, id string) (*entities.Facility, error) {
	query := `
		SELECT 
			id, name, street, city, state, zip_code, country,
			latitude, longitude, phone_number, email, website,
			description, facility_type, rating, review_count,
			is_active, created_at, updated_at
		FROM facilities
		WHERE id = $1 AND is_active = true
	`

	facility := &entities.Facility{}
	err := a.client.DB().QueryRowContext(ctx, query, id).Scan(
		&facility.ID,
		&facility.Name,
		&facility.Address.Street,
		&facility.Address.City,
		&facility.Address.State,
		&facility.Address.ZipCode,
		&facility.Address.Country,
		&facility.Location.Latitude,
		&facility.Location.Longitude,
		&facility.PhoneNumber,
		&facility.Email,
		&facility.Website,
		&facility.Description,
		&facility.FacilityType,
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

	return facility, nil
}

// Update updates a facility
func (a *FacilityAdapter) Update(ctx context.Context, facility *entities.Facility) error {
	query := `
		UPDATE facilities SET
			name = $2, street = $3, city = $4, state = $5, zip_code = $6, country = $7,
			latitude = $8, longitude = $9, phone_number = $10, email = $11, website = $12,
			description = $13, facility_type = $14, rating = $15, review_count = $16,
			is_active = $17, updated_at = $18
		WHERE id = $1
	`

	facility.UpdatedAt = time.Now()

	result, err := a.client.DB().ExecContext(ctx, query,
		facility.ID,
		facility.Name,
		facility.Address.Street,
		facility.Address.City,
		facility.Address.State,
		facility.Address.ZipCode,
		facility.Address.Country,
		facility.Location.Latitude,
		facility.Location.Longitude,
		facility.PhoneNumber,
		facility.Email,
		facility.Website,
		facility.Description,
		facility.FacilityType,
		facility.Rating,
		facility.ReviewCount,
		facility.IsActive,
		facility.UpdatedAt,
	)

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

// Delete deletes a facility (soft delete)
func (a *FacilityAdapter) Delete(ctx context.Context, id string) error {
	query := `UPDATE facilities SET is_active = false, updated_at = $2 WHERE id = $1`

	result, err := a.client.DB().ExecContext(ctx, query, id, time.Now())
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
	query := `
		SELECT 
			id, name, street, city, state, zip_code, country,
			latitude, longitude, phone_number, email, website,
			description, facility_type, rating, review_count,
			is_active, created_at, updated_at
		FROM facilities
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	if filter.FacilityType != "" {
		query += fmt.Sprintf(" AND facility_type = $%d", argCount)
		args = append(args, filter.FacilityType)
		argCount++
	}

	if filter.IsActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argCount)
		args = append(args, *filter.IsActive)
		argCount++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := a.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list facilities", err)
	}
	defer rows.Close()

	facilities := []*entities.Facility{}
	for rows.Next() {
		facility := &entities.Facility{}
		err := rows.Scan(
			&facility.ID,
			&facility.Name,
			&facility.Address.Street,
			&facility.Address.City,
			&facility.Address.State,
			&facility.Address.ZipCode,
			&facility.Address.Country,
			&facility.Location.Latitude,
			&facility.Location.Longitude,
			&facility.PhoneNumber,
			&facility.Email,
			&facility.Website,
			&facility.Description,
			&facility.FacilityType,
			&facility.Rating,
			&facility.ReviewCount,
			&facility.IsActive,
			&facility.CreatedAt,
			&facility.UpdatedAt,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to scan facility", err)
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
	// This is a simplified version. In production, you'd use PostGIS for spatial queries
	query := `
		SELECT 
			id, name, street, city, state, zip_code, country,
			latitude, longitude, phone_number, email, website,
			description, facility_type, rating, review_count,
			is_active, created_at, updated_at,
			(6371 * acos(cos(radians($1)) * cos(radians(latitude)) * 
			cos(radians(longitude) - radians($2)) + sin(radians($1)) * 
			sin(radians(latitude)))) AS distance
		FROM facilities
		WHERE is_active = true
		HAVING distance <= $3
		ORDER BY distance
	`

	args := []interface{}{params.Latitude, params.Longitude, params.RadiusKm}
	argCount := 4

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, params.Limit)
		argCount++
	}

	if params.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, params.Offset)
	}

	rows, err := a.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to search facilities", err)
	}
	defer rows.Close()

	facilities := []*entities.Facility{}
	for rows.Next() {
		facility := &entities.Facility{}
		var distance float64
		err := rows.Scan(
			&facility.ID,
			&facility.Name,
			&facility.Address.Street,
			&facility.Address.City,
			&facility.Address.State,
			&facility.Address.ZipCode,
			&facility.Address.Country,
			&facility.Location.Latitude,
			&facility.Location.Longitude,
			&facility.PhoneNumber,
			&facility.Email,
			&facility.Website,
			&facility.Description,
			&facility.FacilityType,
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
		facilities = append(facilities, facility)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternalError("error iterating facilities", err)
	}

	return facilities, nil
}
