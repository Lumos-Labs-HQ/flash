-- name: IsadminUser :one
SELECT isadmin FROM users WHERE id = $1 LIMIT 1;
