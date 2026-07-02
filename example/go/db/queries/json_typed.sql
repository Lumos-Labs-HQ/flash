-- ══════════════════════════════════════════════════════════════════════════════
-- JSON typed queries — tests @json annotation with import and inline formats
-- ══════════════════════════════════════════════════════════════════════════════

-- name: GetUserPrefs :one
-- @json import user_preferences.json as preferences
SELECT id, name, email, preferences FROM users WHERE id = $1;

-- name: SetUserPrefs :one
-- @json import user_preferences.json as preferences
UPDATE users SET preferences = $1, updated_at = NOW() WHERE id = $2
RETURNING id, name, preferences, updated_at;

-- name: GetPostMeta :one
-- @json import post_metadata.json as metadata
SELECT id, title, status, metadata, user_id, created_at
FROM posts WHERE id = $1;

-- name: ListUsersWithPrefs :many
-- @json import user_preferences.json as preferences
SELECT id, name, email, preferences, created_at
FROM users
ORDER BY created_at DESC
LIMIT $1;

-- name: ListPostsWithMeta :many
-- @json metadata {"view_count": "int", "shares": "int", "bookmarks": "int", "avg_read_time": "float"}
SELECT id, title, status, view_count, metadata, created_at
FROM posts WHERE user_id = $1 AND status = 'published'
ORDER BY created_at DESC;
