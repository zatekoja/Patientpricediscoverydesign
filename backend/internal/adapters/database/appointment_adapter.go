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

// AppointmentAdapter implements the AppointmentRepository interface
type AppointmentAdapter struct {
	client *postgres.Client
	db     *goqu.Database
}

// NewAppointmentAdapter creates a new appointment adapter
func NewAppointmentAdapter(client *postgres.Client) repositories.AppointmentRepository {
	return &AppointmentAdapter{
		client: client,
		db:     goqu.New("postgres", client.DB()),
	}
}

// Create creates a new appointment
func (a *AppointmentAdapter) Create(ctx context.Context, appointment *entities.Appointment) error {
	record := goqu.Record{
		"id":                      appointment.ID,
		"user_id":                 appointment.UserID,
		"facility_id":             appointment.FacilityID,
		"procedure_id":            appointment.ProcedureID,
		"scheduled_at":            appointment.ScheduledAt,
		"status":                  appointment.Status,
		"patient_name":            appointment.PatientName,
		"patient_email":           appointment.PatientEmail,
		"patient_phone":           appointment.PatientPhone,
		"insurance_provider":      appointment.InsuranceProvider,
		"insurance_policy_number": appointment.InsurancePolicyNumber,
		"notes":                   appointment.Notes,
		"created_at":              appointment.CreatedAt,
		"updated_at":              appointment.UpdatedAt,
	}

	query, args, err := a.db.Insert("appointments").Rows(record).ToSQL()
	if err != nil {
		return apperrors.NewInternalError("failed to build insert query", err)
	}

	_, err = a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to create appointment", err)
	}

	return nil
}

// GetByID retrieves an appointment by ID
func (a *AppointmentAdapter) GetByID(ctx context.Context, id string) (*entities.Appointment, error) {
	query, args, err := a.db.Select(
		"id", "user_id", "facility_id", "procedure_id", "scheduled_at",
		"status", "patient_name", "patient_email", "patient_phone",
		"insurance_provider", "insurance_policy_number", "notes",
		"created_at", "updated_at",
	).From("appointments").
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		return nil, apperrors.NewInternalError("failed to build query", err)
	}

	appointment := &entities.Appointment{}
	var userID sql.NullString
	var patientPhone, insuranceProvider, insurancePolicyNumber, notes sql.NullString

	err = a.client.DB().QueryRowContext(ctx, query, args...).Scan(
		&appointment.ID,
		&userID,
		&appointment.FacilityID,
		&appointment.ProcedureID,
		&appointment.ScheduledAt,
		&appointment.Status,
		&appointment.PatientName,
		&appointment.PatientEmail,
		&patientPhone,
		&insuranceProvider,
		&insurancePolicyNumber,
		&notes,
		&appointment.CreatedAt,
		&appointment.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, apperrors.NewNotFoundError(fmt.Sprintf("appointment with id %s not found", id))
	}
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get appointment", err)
	}

	if userID.Valid {
		appointment.UserID = &userID.String
	}
	appointment.PatientPhone = patientPhone.String
	appointment.InsuranceProvider = insuranceProvider.String
	appointment.InsurancePolicyNumber = insurancePolicyNumber.String
	appointment.Notes = notes.String

	return appointment, nil
}

// Update updates an appointment
func (a *AppointmentAdapter) Update(ctx context.Context, appointment *entities.Appointment) error {
	appointment.UpdatedAt = time.Now()

	record := goqu.Record{
		"scheduled_at":            appointment.ScheduledAt,
		"status":                  appointment.Status,
		"patient_name":            appointment.PatientName,
		"patient_email":           appointment.PatientEmail,
		"patient_phone":           appointment.PatientPhone,
		"insurance_provider":      appointment.InsuranceProvider,
		"insurance_policy_number": appointment.InsurancePolicyNumber,
		"notes":                   appointment.Notes,
		"updated_at":              appointment.UpdatedAt,
	}

	query, args, err := a.db.Update("appointments").
		Set(record).
		Where(goqu.Ex{"id": appointment.ID}).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build update query", err)
	}

	result, err := a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to update appointment", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("appointment with id %s not found", appointment.ID))
	}

	return nil
}

// Cancel cancels an appointment
func (a *AppointmentAdapter) Cancel(ctx context.Context, id string) error {
	query, args, err := a.db.Update("appointments").
		Set(goqu.Record{
			"status":     entities.AppointmentStatusCancelled,
			"updated_at": time.Now(),
		}).
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		return apperrors.NewInternalError("failed to build cancel query", err)
	}

	result, err := a.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.NewInternalError("failed to cancel appointment", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return apperrors.NewNotFoundError(fmt.Sprintf("appointment with id %s not found", id))
	}

	return nil
}

// ListByUser retrieves appointments for a user
func (a *AppointmentAdapter) ListByUser(ctx context.Context, userID string, filter repositories.AppointmentFilter) ([]*entities.Appointment, error) {
	ds := a.db.Select(
		"id", "user_id", "facility_id", "procedure_id", "scheduled_at",
		"status", "patient_name", "patient_email", "patient_phone",
		"insurance_provider", "insurance_policy_number", "notes",
		"created_at", "updated_at",
	).From("appointments").
		Where(goqu.Ex{"user_id": userID})

	if filter.Status != "" {
		ds = ds.Where(goqu.Ex{"status": filter.Status})
	}

	if filter.From != nil {
		ds = ds.Where(goqu.C("scheduled_at").Gte(*filter.From))
	}

	if filter.To != nil {
		ds = ds.Where(goqu.C("scheduled_at").Lte(*filter.To))
	}

	ds = ds.Order(goqu.I("scheduled_at").Desc())

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
		return nil, apperrors.NewInternalError("failed to list appointments", err)
	}
	defer rows.Close()

	var appointments []*entities.Appointment
	for rows.Next() {
		appointment := &entities.Appointment{}
		var userID sql.NullString
		var patientPhone, insuranceProvider, insurancePolicyNumber, notes sql.NullString

		err := rows.Scan(
			&appointment.ID,
			&userID,
			&appointment.FacilityID,
			&appointment.ProcedureID,
			&appointment.ScheduledAt,
			&appointment.Status,
			&appointment.PatientName,
			&appointment.PatientEmail,
			&patientPhone,
			&insuranceProvider,
			&insurancePolicyNumber,
			&notes,
			&appointment.CreatedAt,
			&appointment.UpdatedAt,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to scan appointment", err)
		}

		if userID.Valid {
			appointment.UserID = &userID.String
		}
		appointment.PatientPhone = patientPhone.String
		appointment.InsuranceProvider = insuranceProvider.String
		appointment.InsurancePolicyNumber = insurancePolicyNumber.String
		appointment.Notes = notes.String

		appointments = append(appointments, appointment)
	}

	return appointments, nil
}

// ListByFacility retrieves appointments for a facility
func (a *AppointmentAdapter) ListByFacility(ctx context.Context, facilityID string, filter repositories.AppointmentFilter) ([]*entities.Appointment, error) {
	ds := a.db.Select(
		"id", "user_id", "facility_id", "procedure_id", "scheduled_at",
		"status", "patient_name", "patient_email", "patient_phone",
		"insurance_provider", "insurance_policy_number", "notes",
		"created_at", "updated_at",
	).From("appointments").
		Where(goqu.Ex{"facility_id": facilityID})

	if filter.Status != "" {
		ds = ds.Where(goqu.Ex{"status": filter.Status})
	}

	if filter.From != nil {
		ds = ds.Where(goqu.C("scheduled_at").Gte(*filter.From))
	}

	if filter.To != nil {
		ds = ds.Where(goqu.C("scheduled_at").Lte(*filter.To))
	}

	ds = ds.Order(goqu.I("scheduled_at").Desc())

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
		return nil, apperrors.NewInternalError("failed to list appointments", err)
	}
	defer rows.Close()

	var appointments []*entities.Appointment
	for rows.Next() {
		appointment := &entities.Appointment{}
		var userID sql.NullString
		var patientPhone, insuranceProvider, insurancePolicyNumber, notes sql.NullString

		err := rows.Scan(
			&appointment.ID,
			&userID,
			&appointment.FacilityID,
			&appointment.ProcedureID,
			&appointment.ScheduledAt,
			&appointment.Status,
			&appointment.PatientName,
			&appointment.PatientEmail,
			&patientPhone,
			&insuranceProvider,
			&insurancePolicyNumber,
			&notes,
			&appointment.CreatedAt,
			&appointment.UpdatedAt,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to scan appointment", err)
		}

		if userID.Valid {
			appointment.UserID = &userID.String
		}
		appointment.PatientPhone = patientPhone.String
		appointment.InsuranceProvider = insuranceProvider.String
		appointment.InsurancePolicyNumber = insurancePolicyNumber.String
		appointment.Notes = notes.String

		appointments = append(appointments, appointment)
	}

	return appointments, nil
}
