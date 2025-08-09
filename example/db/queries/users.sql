
-- Add your SQL queries here

-- name: GetUser :one
SELECT id, name, email, created_at, updated_at FROM users
WHERE id = $1 LIMIT 1;


-- name: CreateUser :one
INSERT INTO users (name, email)
VALUES ($1, $2)
RETURNING id, name, email, created_at, updated_at;
