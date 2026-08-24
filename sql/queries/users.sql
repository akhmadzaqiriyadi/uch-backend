-- name: CreateUser :exec
INSERT INTO users (id, name, email, password, role, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetUserByID :one
SELECT id, name, email, password, role, created_at, updated_at
FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, name, email, password, role, created_at, updated_at
FROM users
WHERE email = $1 LIMIT 1;

-- name: UpdateUser :exec
UPDATE users
SET name = $2, role = $3, updated_at = $4
WHERE id = $1;

-- name: UpdateUserRole :exec
UPDATE users
SET role = $2, updated_at = $3
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT id, name, email, role, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;
