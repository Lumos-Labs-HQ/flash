-- name: CheckisAdmin :one
SELECT isadmin FROM users WHERE id = $1 LIMIT 1;
