-- name: CreateUser :one
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING *;

-- name: CreateUserFull :one
INSERT INTO users (name, email, age, bio, preferences, tags, role)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;