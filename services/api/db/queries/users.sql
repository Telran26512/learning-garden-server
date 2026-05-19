-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, handle, display_name)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email, password_hash, handle, display_name, role, status, created_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, handle, display_name, role, status, created_at
FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT id, email, password_hash, handle, display_name, role, status, created_at
FROM users
WHERE id = $1 AND deleted_at IS NULL;
