package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"gozaq/internal/domain"
	"gozaq/pkg/database"
)

type RBACRepository struct {
	db *pgxpool.Pool
}

func NewRBACRepository(db *pgxpool.Pool) *RBACRepository {
	return &RBACRepository{db: db}
}

func (r *RBACRepository) GetPermissionsByRole(ctx context.Context, roleID string) ([]string, error) {
	db := database.GetDBTX(ctx, r.db)
	query := `
		SELECT permission_id 
		FROM role_permissions 
		WHERE role_id = $1
		ORDER BY permission_id ASC
	`

	rows, err := db.Query(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query role permissions: %w", err)
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var perm string
		if err := rows.Scan(&perm); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, perm)
	}

	if permissions == nil {
		permissions = []string{}
	}

	return permissions, nil
}

func (r *RBACRepository) ListRoles(ctx context.Context) ([]domain.RoleWithPermissions, error) {
	db := database.GetDBTX(ctx, r.db)
	query := `
		SELECT id, name, description, created_at
		FROM roles
		ORDER BY id ASC
	`

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []domain.RoleWithPermissions
	for rows.Next() {
		var rwp domain.RoleWithPermissions
		if err := rows.Scan(&rwp.ID, &rwp.Name, &rwp.Description, &rwp.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}

		perms, err := r.GetPermissionsByRole(ctx, rwp.ID)
		if err != nil {
			return nil, err
		}

		for _, p := range perms {
			rwp.Permissions = append(rwp.Permissions, domain.Permission{ID: p, Name: p})
		}

		roles = append(roles, rwp)
	}

	return roles, nil
}

func (r *RBACRepository) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	db := database.GetDBTX(ctx, r.db)
	query := `
		SELECT id, name, module, description, created_at
		FROM permissions
		ORDER BY module ASC, id ASC
	`

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		var p domain.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Module, &p.Description, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		perms = append(perms, p)
	}

	return perms, nil
}

func (r *RBACRepository) AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	db := database.GetDBTX(ctx, r.db)
	query := `INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := db.Exec(ctx, query, roleID, permissionID)
	return err
}

func (r *RBACRepository) RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error {
	db := database.GetDBTX(ctx, r.db)
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	_, err := db.Exec(ctx, query, roleID, permissionID)
	return err
}

func (r *RBACRepository) CreateRole(ctx context.Context, id, name, description string) error {
	db := database.GetDBTX(ctx, r.db)
	query := `INSERT INTO roles (id, name, description, created_at) VALUES ($1, $2, $3, NOW())`
	_, err := db.Exec(ctx, query, id, name, description)
	return err
}

func (r *RBACRepository) DeleteRole(ctx context.Context, id string) error {
	db := database.GetDBTX(ctx, r.db)
	_, _ = db.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, id)
	_, err := db.Exec(ctx, `DELETE FROM roles WHERE id = $1`, id)
	return err
}
