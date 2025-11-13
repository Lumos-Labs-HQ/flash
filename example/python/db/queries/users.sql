-- name: GetUser :one
SELECT id, name, email, created_at, updated_at FROM users
WHERE id = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (name, email)
VALUES ($1, $2)
RETURNING id, name, email, created_at, updated_at;

-- name: ListUsers :many
SELECT id, name, email, created_at, updated_at FROM users
ORDER BY created_at DESC;

-- name: UpdateUser :one
UPDATE users
SET name = $2, email = $3, updated_at = NOW()
WHERE id = $1
RETURNING id, name, email, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;