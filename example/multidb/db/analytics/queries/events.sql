-- name: GetRecentEvents :many
SELECT id, event_name, user_id, payload, created_at FROM events ORDER BY created_at DESC LIMIT $1;

-- name: TrackEvent :one
INSERT INTO events (event_name, user_id, payload) VALUES ($1, $2, $3) RETURNING id, event_name, user_id, payload, created_at;

-- name: GetMetricsByName :many
SELECT id, metric, value, recorded_at FROM metrics WHERE metric = $1 ORDER BY recorded_at DESC LIMIT $2;
