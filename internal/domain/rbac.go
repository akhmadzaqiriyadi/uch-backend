package domain

import (
	"context"
	"time"
)

type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Permission struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Module      string    `json:"module"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type RoleWithPermissions struct {
	Role
	Permissions []Permission `json:"permissions"`
}

type RBACRepository interface {
	GetPermissionsByRole(ctx context.Context, roleID string) ([]string, error)
	ListRoles(ctx context.Context) ([]RoleWithPermissions, error)
	ListPermissions(ctx context.Context) ([]Permission, error)
	AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error
	RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error
	CreateRole(ctx context.Context, id, name, description string) error
	DeleteRole(ctx context.Context, id string) error
}

