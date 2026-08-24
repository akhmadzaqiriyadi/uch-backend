-- name: GetPermissionsByRole :many
SELECT permission_id
FROM role_permissions
WHERE role_id = $1
ORDER BY permission_id ASC;

-- name: ListRoles :many
SELECT id, name, description, created_at
FROM roles
ORDER BY id ASC;

-- name: ListPermissions :many
SELECT id, name, module, description, created_at
FROM permissions
ORDER BY module ASC, id ASC;

-- name: AssignPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RevokePermissionFromRole :exec
DELETE FROM role_permissions
WHERE role_id = $1 AND permission_id = $2;
