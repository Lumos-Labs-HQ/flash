-- name: GetUser :one
SELECT id, name, email, created_at FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at;

-- name: GetPostsByUser :many
SELECT id, title, content, created_at FROM posts WHERE user_id = $1 ORDER BY created_at DESC;
