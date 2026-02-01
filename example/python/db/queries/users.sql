

-- name: ListUsers :many
SELECT id, name, email, created_at FROM users
ORDER BY created_at DESC;

-- name: UpdateUser :one
UPDATE users
SET name = $2, email = $3
WHERE id = $1
RETURNING id, name, email, created_at;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;