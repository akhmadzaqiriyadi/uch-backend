package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gozaq/internal/domain"
	"gozaq/pkg/database"
)

type BookingRepository struct {
	db *pgxpool.Pool
}

func NewBookingRepository(db *pgxpool.Pool) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) Create(ctx context.Context, b *domain.Booking) error {
	db := database.GetDBTX(ctx, r.db)
	query := `
		INSERT INTO bookings (
			id, user_id, room_id, room_name, applicant_name, applicant_role,
			id_number, prodi, purpose, audience, booking_date, start_time,
			end_time, status, admin_notes, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	_, err := db.Exec(ctx, query,
		b.ID,
		b.UserID,
		b.RoomID,
		b.RoomName,
		b.ApplicantName,
		b.ApplicantRole,
		b.IDNumber,
		b.Prodi,
		b.Purpose,
		b.Audience,
		b.BookingDate,
		b.StartTime,
		b.EndTime,
		b.Status,
		b.AdminNotes,
		b.CreatedAt,
		b.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("failed to create booking: %w", err)
	}
	return nil
}

func (r *BookingRepository) GetByID(ctx context.Context, id string) (*domain.Booking, error) {
	query := `
		SELECT 
			id, user_id, room_id, room_name, applicant_name, applicant_role,
			COALESCE(id_number, ''), COALESCE(prodi, ''), purpose, audience,
			to_char(booking_date, 'YYYY-MM-DD'), start_time, end_time, status,
			admin_notes, created_at, updated_at
		FROM bookings
		WHERE id = $1
	`
	var b domain.Booking
	err := r.db.QueryRow(ctx, query, id).Scan(
		&b.ID,
		&b.UserID,
		&b.RoomID,
		&b.RoomName,
		&b.ApplicantName,
		&b.ApplicantRole,
		&b.IDNumber,
		&b.Prodi,
		&b.Purpose,
		&b.Audience,
		&b.BookingDate,
		&b.StartTime,
		&b.EndTime,
		&b.Status,
		&b.AdminNotes,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}
	return &b, nil
}

func (r *BookingRepository) List(ctx context.Context, filter domain.BookingFilter) ([]domain.Booking, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.UserID != nil {
		where = append(where, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, *filter.UserID)
		argIdx++
	}

	if filter.RoomID != "" {
		where = append(where, fmt.Sprintf("room_id = $%d", argIdx))
		args = append(args, filter.RoomID)
		argIdx++
	}

	if filter.Status != "" && filter.Status != "all" {
		where = append(where, fmt.Sprintf("LOWER(status) = LOWER($%d)", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.Search != "" {
		where = append(where, fmt.Sprintf(
			"(id ILIKE $%d OR applicant_name ILIKE $%d OR room_name ILIKE $%d OR purpose ILIKE $%d)",
			argIdx, argIdx, argIdx, argIdx,
		))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// 1. Count total items
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM bookings WHERE %s", whereClause)
	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count bookings: %w", err)
	}

	// 2. Query items with pagination
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	query := fmt.Sprintf(`
		SELECT 
			id, user_id, room_id, room_name, applicant_name, applicant_role,
			COALESCE(id_number, ''), COALESCE(prodi, ''), purpose, audience,
			to_char(booking_date, 'YYYY-MM-DD'), start_time, end_time, status,
			admin_notes, created_at, updated_at
		FROM bookings
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list bookings: %w", err)
	}
	defer rows.Close()

	var bookings []domain.Booking
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(
			&b.ID,
			&b.UserID,
			&b.RoomID,
			&b.RoomName,
			&b.ApplicantName,
			&b.ApplicantRole,
			&b.IDNumber,
			&b.Prodi,
			&b.Purpose,
			&b.Audience,
			&b.BookingDate,
			&b.StartTime,
			&b.EndTime,
			&b.Status,
			&b.AdminNotes,
			&b.CreatedAt,
			&b.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan booking: %w", err)
		}
		bookings = append(bookings, b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating bookings: %w", err)
	}

	return bookings, total, nil
}

func (r *BookingRepository) UpdateStatus(ctx context.Context, id string, status string, notes *string) error {
	query := `
		UPDATE bookings
		SET status = $1, admin_notes = $2, updated_at = $3
		WHERE id = $4
	`
	res, err := r.db.Exec(ctx, query, status, notes, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to update booking status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *BookingRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM bookings WHERE id = $1`
	res, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete booking: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *BookingRepository) CheckConflict(
	ctx context.Context,
	roomID string,
	bookingDate string,
	startTime string,
	endTime string,
	excludeBookingID string,
) (*domain.Booking, error) {
	db := database.GetDBTX(ctx, r.db)
	query := `
		SELECT id, user_id, room_id, room_name, applicant_name, applicant_role,
		       id_number, prodi, purpose, audience, booking_date, start_time,
		       end_time, status, admin_notes, created_at, updated_at
		FROM bookings
		WHERE room_id = $1
		  AND booking_date = $2
		  AND status IN ('approved', 'completed')
		  AND start_time < $4
		  AND end_time > $3
		  AND ($5 = '' OR id != $5)
		LIMIT 1
	`
	var b domain.Booking
	err := db.QueryRow(ctx, query, roomID, bookingDate, startTime, endTime, excludeBookingID).Scan(
		&b.ID,
		&b.UserID,
		&b.RoomID,
		&b.RoomName,
		&b.ApplicantName,
		&b.ApplicantRole,
		&b.IDNumber,
		&b.Prodi,
		&b.Purpose,
		&b.Audience,
		&b.BookingDate,
		&b.StartTime,
		&b.EndTime,
		&b.Status,
		&b.AdminNotes,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No conflict
		}
		return nil, fmt.Errorf("failed to check booking conflict: %w", err)
	}
	return &b, nil
}

func (r *BookingRepository) GetOccupiedSlots(ctx context.Context, roomID string, date string) ([]domain.Booking, error) {
	db := database.GetDBTX(ctx, r.db)
	query := `
		SELECT id, user_id, room_id, room_name, applicant_name, applicant_role,
		       id_number, prodi, purpose, audience, booking_date, start_time,
		       end_time, status, admin_notes, created_at, updated_at
		FROM bookings
		WHERE room_id = $1
		  AND booking_date = $2
		  AND status IN ('approved', 'completed')
		ORDER BY start_time ASC
	`
	rows, err := db.Query(ctx, query, roomID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get occupied slots: %w", err)
	}
	defer rows.Close()

	bookings := make([]domain.Booking, 0)
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(
			&b.ID,
			&b.UserID,
			&b.RoomID,
			&b.RoomName,
			&b.ApplicantName,
			&b.ApplicantRole,
			&b.IDNumber,
			&b.Prodi,
			&b.Purpose,
			&b.Audience,
			&b.BookingDate,
			&b.StartTime,
			&b.EndTime,
			&b.Status,
			&b.AdminNotes,
			&b.CreatedAt,
			&b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan occupied slot: %w", err)
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}
