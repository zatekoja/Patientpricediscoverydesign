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

// FeeWaiverAdapter implements FeeWaiverRepository
type FeeWaiverAdapter struct {
	client *postgres.Client
	db     *goqu.Database
}

var _ repositories.FeeWaiverRepository = (*FeeWaiverAdapter)(nil)

// NewFeeWaiverAdapter creates a new fee waiver adapter
func NewFeeWaiverAdapter(client *postgres.Client) *FeeWaiverAdapter {
	return &FeeWaiverAdapter{
		client: client,
		db:     goqu.New("postgres", client.DB()),
	}
}

// Create creates a new fee waiver
func (a *FeeWaiverAdapter) Create(ctx context.Context, waiver *entities.FeeWaiver) error {
	record := goqu.Record{
		"id":              waiver.ID,
		"sponsor_name":    waiver.SponsorName,
		"sponsor_contact": sql.NullString{String: waiver.SponsorContact, Valid: waiver.SponsorContact != ""},
		"facility_id":     waiver.FacilityID,
		"waiver_type":     waiver.WaiverType,
		"waiver_amount":   waiver.WaiverAmount,
		"max_uses":        waiver.MaxUses,
		"current_uses":    waiver.CurrentUses,
		"valid_from":      waiver.ValidFrom,
		"valid_until":     waiver.ValidUntil,
		"is_active":       waiver.IsActive,
		"created_at":      waiver.CreatedAt,
		"updated_at":      waiver.UpdatedAt,
	}

	query, args, err := a.db.Insert("fee_waivers").Rows(record).ToSQL()
	if err != nil {
		return apperrors.NewInternalError("failed to build insert query", err)
	}

	_, err = a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to create fee waiver", err)
	}

	return nil
}

// GetByID retrieves a fee waiver by ID
func (a *FeeWaiverAdapter) GetByID(ctx context.Context, id string) (*entities.FeeWaiver, error) {
	query, args, err := a.db.Select("*").From("fee_waivers").
		Where(goqu.Ex{"id": id}).
		ToSQL()
	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	return a.scanFeeWaiver(ctx, query, args...)
}

// GetActiveFacilityWaiver retrieves the active waiver for a facility (or global waiver)
func (a *FeeWaiverAdapter) GetActiveFacilityWaiver(ctx context.Context, facilityID string) (*entities.FeeWaiver, error) {
	now := time.Now()

	query, args, err := a.db.Select("*").From("fee_waivers").
		Where(
			goqu.Ex{"is_active": true},
			goqu.Or(
				goqu.Ex{"facility_id": facilityID},
				goqu.Ex{"facility_id": nil},
			),
			goqu.I("valid_from").Lte(now),
			goqu.Or(
				goqu.Ex{"valid_until": nil},
				goqu.I("valid_until").Gte(now),
			),
			goqu.Or(
				goqu.Ex{"max_uses": nil},
				goqu.L("current_uses < max_uses"),
			),
		).
		// Prefer facility-specific waivers over global ones
		Order(goqu.I("facility_id").Desc().NullsLast()).
		Limit(1).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	waiver, err := a.scanFeeWaiver(ctx, query, args...)
	if err != nil {
		// Not found is expected — no waiver available
		var appErr *apperrors.AppError
		if ok := isNotFoundError(err, &appErr); ok {
			return nil, nil
		}
		return nil, err
	}

	return waiver, nil
}

// IncrementUsage atomically increments the current_uses counter
func (a *FeeWaiverAdapter) IncrementUsage(ctx context.Context, id string) error {
	query := "UPDATE fee_waivers SET current_uses = current_uses + 1, updated_at = $1 WHERE id = $2"
	_, err := a.client.DB().ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return apperrors.NewInternalError("failed to increment waiver usage", err)
	}
	return nil
}

// Update updates a fee waiver
func (a *FeeWaiverAdapter) Update(ctx context.Context, waiver *entities.FeeWaiver) error {
	waiver.UpdatedAt = time.Now()

	record := goqu.Record{
		"sponsor_name":    waiver.SponsorName,
		"sponsor_contact": sql.NullString{String: waiver.SponsorContact, Valid: waiver.SponsorContact != ""},
		"facility_id":     waiver.FacilityID,
		"waiver_type":     waiver.WaiverType,
		"waiver_amount":   waiver.WaiverAmount,
		"max_uses":        waiver.MaxUses,
		"is_active":       waiver.IsActive,
		"valid_from":      waiver.ValidFrom,
		"valid_until":     waiver.ValidUntil,
		"updated_at":      waiver.UpdatedAt,
	}

	query, args, err := a.db.Update("fee_waivers").
		Set(record).
		Where(goqu.Ex{"id": waiver.ID}).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build update query", err)
	}

	result, err := a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to update fee waiver", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("fee waiver with id %s not found", waiver.ID))
	}

	return nil
}

func (a *FeeWaiverAdapter) scanFeeWaiver(ctx context.Context, query string, args ...interface{}) (*entities.FeeWaiver, error) {
	w := &entities.FeeWaiver{}
	var sponsorContact sql.NullString
	var facilityID sql.NullString
	var waiverAmount sql.NullFloat64
	var maxUses sql.NullInt32
	var validUntil sql.NullTime

	err := a.client.DB().QueryRowContext(ctx, query, args...).Scan(
		&w.ID,
		&w.SponsorName,
		&sponsorContact,
		&facilityID,
		&w.WaiverType,
		&waiverAmount,
		&maxUses,
		&w.CurrentUses,
		&w.ValidFrom,
		&validUntil,
		&w.IsActive,
		&w.CreatedAt,
		&w.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, apperrors.NewNotFoundError("fee waiver not found")
	}
	if err != nil {
		return nil, apperrors.NewInternalError("failed to scan fee waiver", err)
	}

	w.SponsorContact = sponsorContact.String
	if facilityID.Valid {
		w.FacilityID = &facilityID.String
	}
	if waiverAmount.Valid {
		w.WaiverAmount = &waiverAmount.Float64
	}
	if maxUses.Valid {
		val := int(maxUses.Int32)
		w.MaxUses = &val
	}
	if validUntil.Valid {
		w.ValidUntil = &validUntil.Time
	}

	return w, nil
}

func isNotFoundError(err error, target **apperrors.AppError) bool {
	if err == nil {
		return false
	}
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		return false
	}
	*target = appErr
	return appErr.Type == apperrors.ErrorTypeNotFound
}
