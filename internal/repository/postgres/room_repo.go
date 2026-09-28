package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gozaq/internal/domain"
	"gozaq/pkg/database"
)

type RoomRepository struct {
	db *pgxpool.Pool
}

func NewRoomRepository(db *pgxpool.Pool) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) Create(ctx context.Context, room *domain.Room) error {
	db := database.GetDBTX(ctx, r.db)
	query := `
		INSERT INTO rooms (id, name, slug, category, capacity, location, description, facilities, image_url, status, operational_hours, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := db.Exec(ctx, query,
		room.ID,
		room.Name,
		room.Slug,
		room.Category,
		room.Capacity,
		room.Location,
		room.Description,
		room.Facilities,
		room.ImageURL,
		room.Status,
		room.OperationalHours,
		room.CreatedAt,
		room.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("failed to create room: %w", err)
	}
	return nil
}

func (r *RoomRepository) GetByID(ctx context.Context, id string) (*domain.Room, error) {
	query := `
		SELECT id, name, slug, category, capacity, location, description, facilities, COALESCE(image_url, ''), status, COALESCE(operational_hours, ''), created_at, updated_at
		FROM rooms
		WHERE id = $1 OR slug = $1
	`
	var room domain.Room
	err := r.db.QueryRow(ctx, query, id).Scan(
		&room.ID,
		&room.Name,
		&room.Slug,
		&room.Category,
		&room.Capacity,
		&room.Location,
		&room.Description,
		&room.Facilities,
		&room.ImageURL,
		&room.Status,
		&room.OperationalHours,
		&room.CreatedAt,
		&room.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get room: %w", err)
	}
	return &room, nil
}

func (r *RoomRepository) List(ctx context.Context) ([]domain.Room, error) {
	query := `
		SELECT id, name, slug, category, capacity, location, description, facilities, COALESCE(image_url, ''), status, COALESCE(operational_hours, ''), created_at, updated_at
		FROM rooms
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list rooms: %w", err)
	}
	defer rows.Close()

	var rooms []domain.Room
	for rows.Next() {
		var room domain.Room
		if err := rows.Scan(
			&room.ID,
			&room.Name,
			&room.Slug,
			&room.Category,
			&room.Capacity,
			&room.Location,
			&room.Description,
			&room.Facilities,
			&room.ImageURL,
			&room.Status,
			&room.OperationalHours,
			&room.CreatedAt,
			&room.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan room: %w", err)
		}
		rooms = append(rooms, room)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rooms: %w", err)
	}
	return rooms, nil
}

func (r *RoomRepository) Update(ctx context.Context, id string, req *domain.UpdateRoomRequest) (*domain.Room, error) {
	room, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		room.Name = *req.Name
	}
	if req.Slug != nil {
		room.Slug = *req.Slug
	}
	if req.Category != nil {
		room.Category = *req.Category
	}
	if req.Capacity != nil {
		room.Capacity = *req.Capacity
	}
	if req.Location != nil {
		room.Location = *req.Location
	}
	if req.Description != nil {
		room.Description = *req.Description
	}
	if req.Facilities != nil {
		facBytes, err := json.Marshal(*req.Facilities)
		if err == nil {
			room.Facilities = facBytes
		}
	}
	if req.ImageURL != nil {
		room.ImageURL = *req.ImageURL
	}
	if req.Status != nil {
		room.Status = *req.Status
	}
	if req.OperationalHours != nil {
		room.OperationalHours = *req.OperationalHours
	}
	room.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE rooms
		SET name = $1, slug = $2, category = $3, capacity = $4, location = $5, description = $6, facilities = $7, image_url = $8, status = $9, operational_hours = $10, updated_at = $11
		WHERE id = $12
	`
	_, err = r.db.Exec(ctx, query,
		room.Name,
		room.Slug,
		room.Category,
		room.Capacity,
		room.Location,
		room.Description,
		room.Facilities,
		room.ImageURL,
		room.Status,
		room.OperationalHours,
		room.UpdatedAt,
		room.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update room: %w", err)
	}
	return room, nil
}

func (r *RoomRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM rooms WHERE id = $1`
	res, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete room: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
