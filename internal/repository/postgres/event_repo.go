package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gozaq/internal/domain"
	"gozaq/pkg/database"
)

type EventRepository struct {
	db *pgxpool.Pool
}

func NewEventRepository(db *pgxpool.Pool) *EventRepository {
	return &EventRepository{db: db}
}

const eventSelectColumns = `
	id, title, slug, description, COALESCE(long_description, ''),
	series_id, series_name, category_name, category_variant,
	TO_CHAR(event_date, 'YYYY-MM-DD') AS event_date,
	date_day, date_month, date_year, date_full_text,
	time, location_name, location_room, location_address, location_type,
	cover_image, quota_total, quota_filled, quota_status, quota_status_label,
	registration_url, featured, is_free, price, payment_info, requires_approval,
	custom_fields_schema, speakers, rundown, benefits, prerequisites,
	target_audience, registration_deadline, fee, contact_person, status,
	created_at, updated_at
`

func scanEvent(row pgx.Row) (*domain.Event, error) {
	var e domain.Event
	var seriesID, seriesName, locRoom, locAddr, regURL, targetAud, regDeadline *string

	err := row.Scan(
		&e.ID, &e.Title, &e.Slug, &e.Description, &e.LongDescription,
		&seriesID, &seriesName, &e.CategoryName, &e.CategoryVariant,
		&e.EventDate,
		&e.DateDay, &e.DateMonth, &e.DateYear, &e.DateFullText,
		&e.Time, &e.LocationName, &locRoom, &locAddr, &e.LocationType,
		&e.CoverImage, &e.QuotaTotal, &e.QuotaFilled, &e.QuotaStatus, &e.QuotaStatusLabel,
		&regURL, &e.Featured, &e.IsFree, &e.Price, &e.PaymentInfo, &e.RequiresApproval,
		&e.CustomFieldsSchema, &e.Speakers, &e.Rundown, &e.Benefits, &e.Prerequisites,
		&targetAud, &regDeadline, &e.Fee, &e.ContactPerson, &e.Status,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to scan event: %w", err)
	}

	e.SeriesID = seriesID
	e.SeriesName = seriesName
	e.LocationRoom = locRoom
	e.LocationAddress = locAddr
	e.RegistrationURL = regURL
	e.TargetAudience = targetAud
	e.RegistrationDeadline = regDeadline

	return &e, nil
}

func (r *EventRepository) Create(ctx context.Context, e *domain.Event) error {
	db := database.GetDBTX(ctx, r.db)
	query := `
		INSERT INTO events (
			id, title, slug, description, long_description,
			series_id, series_name, category_name, category_variant,
			event_date, date_day, date_month, date_year, date_full_text,
			time, location_name, location_room, location_address, location_type,
			cover_image, quota_total, quota_filled, quota_status, quota_status_label,
			registration_url, featured, is_free, price, payment_info, requires_approval,
			custom_fields_schema, speakers, rundown, benefits, prerequisites,
			target_audience, registration_deadline, fee, contact_person, status,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19,
			$20, $21, $22, $23, $24,
			$25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35,
			$36, $37, $38, $39, $40,
			$41, $42
		)
	`
	_, err := db.Exec(ctx, query,
		e.ID, e.Title, e.Slug, e.Description, e.LongDescription,
		e.SeriesID, e.SeriesName, e.CategoryName, e.CategoryVariant,
		e.EventDate, e.DateDay, e.DateMonth, e.DateYear, e.DateFullText,
		e.Time, e.LocationName, e.LocationRoom, e.LocationAddress, e.LocationType,
		e.CoverImage, e.QuotaTotal, e.QuotaFilled, e.QuotaStatus, e.QuotaStatusLabel,
		e.RegistrationURL, e.Featured, e.IsFree, e.Price, e.PaymentInfo, e.RequiresApproval,
		e.CustomFieldsSchema, e.Speakers, e.Rundown, e.Benefits, e.Prerequisites,
		e.TargetAudience, e.RegistrationDeadline, e.Fee, e.ContactPerson, e.Status,
		e.CreatedAt, e.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("failed to create event: %w", err)
	}
	return nil
}

func (r *EventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	query := fmt.Sprintf(`SELECT %s FROM events WHERE id = $1`, eventSelectColumns)
	return scanEvent(r.db.QueryRow(ctx, query, id))
}

func (r *EventRepository) GetBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	query := fmt.Sprintf(`SELECT %s FROM events WHERE slug = $1`, eventSelectColumns)
	return scanEvent(r.db.QueryRow(ctx, query, slug))
}

func (r *EventRepository) List(ctx context.Context, filter domain.EventFilter) ([]domain.Event, int, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category_name ILIKE $%d", idx))
		args = append(args, "%"+filter.Category+"%")
		idx++
	}

	if filter.SeriesID != "" {
		conditions = append(conditions, fmt.Sprintf("series_id = $%d", idx))
		args = append(args, filter.SeriesID)
		idx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, filter.Status)
		idx++
	}

	if filter.UpcomingOnly {
		conditions = append(conditions, "event_date >= CURRENT_DATE")
	} else if filter.PastOnly {
		conditions = append(conditions, "event_date < CURRENT_DATE")
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d OR location_name ILIKE $%d)", idx, idx, idx))
		args = append(args, "%"+filter.Search+"%")
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events %s", whereClause)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	orderDirection := "ASC"
	if filter.PastOnly {
		orderDirection = "DESC"
	}

	query := fmt.Sprintf(`
		SELECT %s FROM events
		%s
		ORDER BY event_date %s, created_at DESC
		LIMIT $%d OFFSET $%d
	`, eventSelectColumns, whereClause, orderDirection, idx, idx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.Event, 0)
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, *e)
	}

	return events, total, nil
}

func (r *EventRepository) Update(ctx context.Context, id string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Slug != nil {
		existing.Slug = *req.Slug
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.LongDescription != nil {
		existing.LongDescription = *req.LongDescription
	}
	if req.SeriesID != nil {
		existing.SeriesID = req.SeriesID
	}
	if req.SeriesName != nil {
		existing.SeriesName = req.SeriesName
	}
	if req.CategoryName != nil {
		existing.CategoryName = *req.CategoryName
	}
	if req.CategoryVariant != nil {
		existing.CategoryVariant = *req.CategoryVariant
	}
	if req.EventDate != nil {
		existing.EventDate = *req.EventDate
	}
	if req.DateDay != nil {
		existing.DateDay = *req.DateDay
	}
	if req.DateMonth != nil {
		existing.DateMonth = *req.DateMonth
	}
	if req.DateYear != nil {
		existing.DateYear = *req.DateYear
	}
	if req.DateFullText != nil {
		existing.DateFullText = *req.DateFullText
	}
	if req.Time != nil {
		existing.Time = *req.Time
	}
	if req.LocationName != nil {
		existing.LocationName = *req.LocationName
	}
	if req.LocationRoom != nil {
		existing.LocationRoom = req.LocationRoom
	}
	if req.LocationAddress != nil {
		existing.LocationAddress = req.LocationAddress
	}
	if req.LocationType != nil {
		existing.LocationType = *req.LocationType
	}
	if req.CoverImage != nil {
		existing.CoverImage = *req.CoverImage
	}
	if req.QuotaTotal != nil {
		existing.QuotaTotal = *req.QuotaTotal
	}
	if req.QuotaStatus != nil {
		existing.QuotaStatus = *req.QuotaStatus
	}
	if req.QuotaStatusLabel != nil {
		existing.QuotaStatusLabel = *req.QuotaStatusLabel
	}
	if req.Featured != nil {
		existing.Featured = *req.Featured
	}
	if req.IsFree != nil {
		existing.IsFree = *req.IsFree
	}
	if req.Price != nil {
		existing.Price = *req.Price
	}
	if req.PaymentInfo != nil {
		b, _ := json.Marshal(req.PaymentInfo)
		existing.PaymentInfo = b
	}
	if req.RequiresApproval != nil {
		existing.RequiresApproval = *req.RequiresApproval
	}
	if req.CustomFieldsSchema != nil {
		b, _ := json.Marshal(req.CustomFieldsSchema)
		existing.CustomFieldsSchema = b
	}
	if req.Speakers != nil {
		b, _ := json.Marshal(req.Speakers)
		existing.Speakers = b
	}
	if req.Rundown != nil {
		b, _ := json.Marshal(req.Rundown)
		existing.Rundown = b
	}
	if req.Benefits != nil {
		b, _ := json.Marshal(req.Benefits)
		existing.Benefits = b
	}
	if req.Prerequisites != nil {
		b, _ := json.Marshal(req.Prerequisites)
		existing.Prerequisites = b
	}
	if req.TargetAudience != nil {
		existing.TargetAudience = req.TargetAudience
	}
	if req.RegistrationDeadline != nil {
		existing.RegistrationDeadline = req.RegistrationDeadline
	}
	if req.Fee != nil {
		existing.Fee = *req.Fee
	}
	if req.ContactPerson != nil {
		b, _ := json.Marshal(req.ContactPerson)
		existing.ContactPerson = b
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}
	existing.UpdatedAt = time.Now()

	query := `
		UPDATE events SET
			title = $1, slug = $2, description = $3, long_description = $4,
			series_id = $5, series_name = $6, category_name = $7, category_variant = $8,
			event_date = $9, date_day = $10, date_month = $11, date_year = $12, date_full_text = $13,
			time = $14, location_name = $15, location_room = $16, location_address = $17, location_type = $18,
			cover_image = $19, quota_total = $20, quota_status = $21, quota_status_label = $22,
			featured = $23, is_free = $24, price = $25, payment_info = $26, requires_approval = $27,
			custom_fields_schema = $28, speakers = $29, rundown = $30, benefits = $31, prerequisites = $32,
			target_audience = $33, registration_deadline = $34, fee = $35, contact_person = $36, status = $37,
			updated_at = $38
		WHERE id = $39
	`
	_, err = r.db.Exec(ctx, query,
		existing.Title, existing.Slug, existing.Description, existing.LongDescription,
		existing.SeriesID, existing.SeriesName, existing.CategoryName, existing.CategoryVariant,
		existing.EventDate, existing.DateDay, existing.DateMonth, existing.DateYear, existing.DateFullText,
		existing.Time, existing.LocationName, existing.LocationRoom, existing.LocationAddress, existing.LocationType,
		existing.CoverImage, existing.QuotaTotal, existing.QuotaStatus, existing.QuotaStatusLabel,
		existing.Featured, existing.IsFree, existing.Price, existing.PaymentInfo, existing.RequiresApproval,
		existing.CustomFieldsSchema, existing.Speakers, existing.Rundown, existing.Benefits, existing.Prerequisites,
		existing.TargetAudience, existing.RegistrationDeadline, existing.Fee, existing.ContactPerson, existing.Status,
		existing.UpdatedAt, id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return existing, nil
}

func (r *EventRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM events WHERE id = $1`
	res, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *EventRepository) IncrementQuotaFilled(ctx context.Context, id string, delta int) error {
	query := `
		UPDATE events SET
			quota_filled = GREATEST(0, quota_filled + $2),
			quota_status = CASE
				WHEN quota_filled + $2 >= quota_total THEN 'full'
				WHEN quota_filled + $2 >= CAST(quota_total * 0.8 AS INT) THEN 'closing-soon'
				ELSE 'open'
			END,
			quota_status_label = CASE
				WHEN quota_filled + $2 >= quota_total THEN 'Kuota Penuh'
				WHEN quota_filled + $2 >= CAST(quota_total * 0.8 AS INT) THEN 'Slot Terbatas'
				ELSE 'Pendaftaran Dibuka'
			END,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, delta)
	if err != nil {
		return fmt.Errorf("failed to update quota: %w", err)
	}
	return nil
}

// ----------------- Registration Methods -----------------

const registrationSelectColumns = `
	r.id, r.registration_code, r.event_id,
	COALESCE(e.title, '') AS event_title,
	COALESCE(e.slug, '') AS event_slug,
	COALESCE(TO_CHAR(e.event_date, 'YYYY-MM-DD'), '') AS event_date,
	COALESCE(e.time, '') AS event_time,
	COALESCE(e.location_name, '') AS event_location,
	r.user_id, r.full_name, r.identity_number, r.institution,
	r.email, r.phone, r.notes, r.answers, r.uploaded_files,
	r.payment_proof_url, r.status, r.admin_notes,
	r.verified_by, r.verified_at, r.attended_at,
	r.created_at, r.updated_at
`

func scanRegistration(row pgx.Row) (*domain.EventRegistration, error) {
	var reg domain.EventRegistration
	var idNum, notes, proofURL, adminNotes *string
	var verifiedBy *uuid.UUID
	var verifiedAt, attendedAt *time.Time

	err := row.Scan(
		&reg.ID, &reg.RegistrationCode, &reg.EventID,
		&reg.EventTitle, &reg.EventSlug, &reg.EventDate, &reg.EventTime, &reg.EventLocation,
		&reg.UserID, &reg.FullName, &idNum, &reg.Institution,
		&reg.Email, &reg.Phone, &notes, &reg.Answers, &reg.UploadedFiles,
		&proofURL, &reg.Status, &adminNotes,
		&verifiedBy, &verifiedAt, &attendedAt,
		&reg.CreatedAt, &reg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to scan event registration: %w", err)
	}

	reg.IdentityNumber = idNum
	reg.Notes = notes
	reg.PaymentProofURL = proofURL
	reg.AdminNotes = adminNotes
	reg.VerifiedBy = verifiedBy
	reg.VerifiedAt = verifiedAt
	reg.AttendedAt = attendedAt

	return &reg, nil
}

func (r *EventRepository) Register(ctx context.Context, reg *domain.EventRegistration) error {
	db := database.GetDBTX(ctx, r.db)
	query := `
		INSERT INTO event_registrations (
			id, registration_code, event_id, user_id, full_name, identity_number,
			institution, email, phone, notes, answers, uploaded_files, payment_proof_url,
			status, admin_notes, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17
		)
	`
	_, err := db.Exec(ctx, query,
		reg.ID, reg.RegistrationCode, reg.EventID, reg.UserID, reg.FullName, reg.IdentityNumber,
		reg.Institution, reg.Email, reg.Phone, reg.Notes, reg.Answers, reg.UploadedFiles, reg.PaymentProofURL,
		reg.Status, reg.AdminNotes, reg.CreatedAt, reg.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("failed to save registration: %w", err)
	}
	return nil
}

func (r *EventRepository) GetRegistrationByID(ctx context.Context, id string) (*domain.EventRegistration, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM event_registrations r
		LEFT JOIN events e ON r.event_id = e.id
		WHERE r.id = $1
	`, registrationSelectColumns)
	return scanRegistration(r.db.QueryRow(ctx, query, id))
}

func (r *EventRepository) GetRegistrationByCode(ctx context.Context, code string) (*domain.EventRegistration, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM event_registrations r
		LEFT JOIN events e ON r.event_id = e.id
		WHERE r.registration_code = $1
	`, registrationSelectColumns)
	return scanRegistration(r.db.QueryRow(ctx, query, code))
}

func (r *EventRepository) GetUserRegistration(ctx context.Context, eventID string, userID uuid.UUID) (*domain.EventRegistration, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM event_registrations r
		LEFT JOIN events e ON r.event_id = e.id
		WHERE r.event_id = $1 AND r.user_id = $2
	`, registrationSelectColumns)
	return scanRegistration(r.db.QueryRow(ctx, query, eventID, userID))
}

func (r *EventRepository) ListRegistrations(ctx context.Context, filter domain.RegistrationFilter) ([]domain.EventRegistration, int, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	if filter.EventID != "" {
		conditions = append(conditions, fmt.Sprintf("r.event_id = $%d", idx))
		args = append(args, filter.EventID)
		idx++
	}

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("r.user_id = $%d", idx))
		args = append(args, *filter.UserID)
		idx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("r.status = $%d", idx))
		args = append(args, filter.Status)
		idx++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(r.full_name ILIKE $%d OR r.email ILIKE $%d OR r.registration_code ILIKE $%d OR r.institution ILIKE $%d)", idx, idx, idx, idx))
		args = append(args, "%"+filter.Search+"%")
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM event_registrations r %s", whereClause)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count registrations: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 25
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	query := fmt.Sprintf(`
		SELECT %s FROM event_registrations r
		LEFT JOIN events e ON r.event_id = e.id
		%s
		ORDER BY r.created_at DESC
		LIMIT $%d OFFSET $%d
	`, registrationSelectColumns, whereClause, idx, idx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list registrations: %w", err)
	}
	defer rows.Close()

	list := make([]domain.EventRegistration, 0)
	for rows.Next() {
		reg, err := scanRegistration(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *reg)
	}

	return list, total, nil
}

func (r *EventRepository) UpdateRegistrationStatus(ctx context.Context, id string, status string, notes *string, verifiedBy *uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE event_registrations SET
			status = $1,
			admin_notes = COALESCE($2, admin_notes),
			verified_by = $3,
			verified_at = $4,
			updated_at = $5
		WHERE id = $6
	`
	res, err := r.db.Exec(ctx, query, status, notes, verifiedBy, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to update registration status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *EventRepository) ReviseRegistration(ctx context.Context, id string, req *domain.ReviseRegistrationRequest) error {
	now := time.Now()
	var answersJSON, filesJSON []byte
	if req.Answers != nil {
		answersJSON, _ = json.Marshal(req.Answers)
	}
	if req.UploadedFiles != nil {
		filesJSON, _ = json.Marshal(req.UploadedFiles)
	}

	query := `
		UPDATE event_registrations SET
			status = 'pending_review',
			notes = COALESCE($1, notes),
			answers = CASE WHEN $2::jsonb IS NOT NULL THEN $2::jsonb ELSE answers END,
			uploaded_files = CASE WHEN $3::jsonb IS NOT NULL THEN $3::jsonb ELSE uploaded_files END,
			payment_proof_url = COALESCE($4, payment_proof_url),
			updated_at = $5
		WHERE id = $6
	`
	res, err := r.db.Exec(ctx, query, req.Notes, answersJSON, filesJSON, req.PaymentProofURL, now, id)
	if err != nil {
		return fmt.Errorf("failed to revise registration: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *EventRepository) CheckIn(ctx context.Context, code string) (*domain.EventRegistration, error) {
	now := time.Now()
	query := `
		UPDATE event_registrations SET
			status = 'attended',
			attended_at = $1,
			updated_at = $2
		WHERE registration_code = $3
		RETURNING id
	`
	var id string
	err := r.db.QueryRow(ctx, query, now, now, code).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to check in: %w", err)
	}

	return r.GetRegistrationByID(ctx, id)
}
