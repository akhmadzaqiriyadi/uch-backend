package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gozaq/internal/domain"
	"gozaq/pkg/database"
	"gozaq/pkg/pagination"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	db := database.GetDBTX(ctx, r.db)
	query := `
		INSERT INTO users (id, name, email, password, role, is_verified, verified_at, deleted_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := db.Exec(ctx, query,
		user.ID,
		user.Name,
		user.Email,
		user.Password,
		user.Role,
		user.IsVerified,
		user.VerifiedAt,
		user.DeletedAt,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Unique violation
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, name, email, password, role, is_verified, verified_at, deleted_at, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	var user domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.IsVerified,
		&user.VerifiedAt,
		&user.DeletedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password, role, is_verified, verified_at, deleted_at, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`
	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.IsVerified,
		&user.VerifiedAt,
		&user.DeletedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	db := database.GetDBTX(ctx, r.db)
	query := `
		UPDATE users
		SET name = $2, role = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := db.Exec(ctx, query,
		user.ID,
		user.Name,
		user.Role,
		user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error {
	db := database.GetDBTX(ctx, r.db)
	query := `
		UPDATE users
		SET password = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := db.Exec(ctx, query, id, hashedPassword, time.Now().UTC())
	return err
}

func (r *UserRepository) UpdateRole(ctx context.Context, id uuid.UUID, role string) error {
	db := database.GetDBTX(ctx, r.db)
	query := `
		UPDATE users
		SET role = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := db.Exec(ctx, query, id, role, time.Now().UTC())
	return err
}

func (r *UserRepository) SetEmailVerified(ctx context.Context, id uuid.UUID) error {
	db := database.GetDBTX(ctx, r.db)
	now := time.Now().UTC()
	query := `
		UPDATE users
		SET is_verified = TRUE, verified_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := db.Exec(ctx, query, id, now)
	return err
}

// Delete performs a safe soft-delete
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	query := `UPDATE users SET deleted_at = $2, updated_at = $2 WHERE id = $1 AND deleted_at IS NULL`
	cmdTag, err := r.db.Exec(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) List(ctx context.Context, p pagination.Params, search string) ([]*domain.User, int, error) {
	countQuery := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND (name ILIKE $1 OR email ILIKE $1)`
	searchPattern := "%" + search + "%"

	var totalItems int
	if err := r.db.QueryRow(ctx, countQuery, searchPattern).Scan(&totalItems); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	orderDir := "DESC"
	if p.Order == "asc" {
		orderDir = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, name, email, password, role, is_verified, verified_at, deleted_at, created_at, updated_at
		FROM users
		WHERE deleted_at IS NULL AND (name ILIKE $1 OR email ILIKE $1)
		ORDER BY %s %s
		LIMIT $2 OFFSET $3
	`, p.SortBy, orderDir)

	rows, err := r.db.Query(ctx, query, searchPattern, p.Limit(), p.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.Password,
			&u.Role,
			&u.IsVerified,
			&u.VerifiedAt,
			&u.DeletedAt,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &u)
	}

	return users, totalItems, nil
}
