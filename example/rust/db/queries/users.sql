-- ============================================================
-- :one — single row returns
-- ============================================================

-- name: GetUser :one
SELECT id, name, email, age, bio, is_active, score, balance, created_at, updated_at
FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, name, email, age, bio, is_active, score, balance, created_at, updated_at
FROM users WHERE email = $1;

-- ============================================================
-- :many — multiple row returns
-- ============================================================

-- name: ListUsers :many
SELECT id, name, email, age, bio, is_active, score, balance, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListActiveUsers :many
SELECT id, name, email, age, is_active, created_at
FROM users
WHERE is_active = TRUE
ORDER BY name ASC;

-- name: SearchUsersByName :many
SELECT id, name, email, age
FROM users
WHERE name ILIKE '%' || $1 || '%'
ORDER BY name;

-- name: ListUsersInAgeRange :many
SELECT id, name, email, age
FROM users
WHERE age >= $1 AND age <= $2;

-- ============================================================
-- :exec — no return value
-- ============================================================

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: DeactivateUser :exec
UPDATE users SET is_active = FALSE, updated_at = NOW() WHERE id = $1;

-- name: DeleteInactiveUsers :exec
DELETE FROM users WHERE is_active = FALSE;

-- ============================================================
-- :execresult — returns affected row count
-- ============================================================

-- name: UpdateUserName :execresult
UPDATE users SET name = $2, updated_at = NOW() WHERE id = $1;

-- name: BulkActivateUsers :execresult
UPDATE users SET is_active = TRUE, updated_at = NOW() WHERE is_active = FALSE;

-- ============================================================
-- :one — single scalar returns
-- ============================================================

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountActiveUsers :one
SELECT COUNT(*) FROM users WHERE is_active = TRUE;

-- name: UserExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE id = $1);

-- name: EmailTaken :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1);

-- ============================================================
-- :exec / :one with multiple params (>2 triggers params struct)
-- ============================================================

-- name: CreateUser :one
INSERT INTO users (name, email, age, bio, score, balance)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, email, age, bio, is_active, score, balance, created_at, updated_at;

-- name: UpdateUserFull :exec
UPDATE users
SET name = $2, email = $3, age = $4, bio = $5, score = $6, balance = $7, updated_at = NOW()
WHERE id = $1;

-- ============================================================
-- :one / :many with nullable columns
-- ============================================================

-- name: GetUserAge :one
SELECT age FROM users WHERE id = $1;

-- name: ListUsersWithBio :many
SELECT id, name, bio FROM users WHERE bio IS NOT NULL;

-- ============================================================
-- Aggregation queries (single table)
-- ============================================================

-- name: GetUserAgeStats :one
SELECT
    AVG(age)::INTEGER AS avg_age,
    MIN(age) AS min_age,
    MAX(age) AS max_age
FROM users
WHERE age IS NOT NULL;

-- name: GetUserScoreDistribution :many
SELECT
    CASE WHEN score < 3.0 THEN 'low' WHEN score < 7.0 THEN 'medium' ELSE 'high' END AS tier,
    COUNT(*) AS user_count
FROM users
GROUP BY tier
ORDER BY user_count DESC;

-- ============================================================
-- INSERT ... ON CONFLICT (upsert)
-- ============================================================

-- name: UpsertUser :one
INSERT INTO users (name, email, age, bio, score)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (email)
DO UPDATE SET name = EXCLUDED.name, age = EXCLUDED.age, bio = EXCLUDED.bio, score = EXCLUDED.score, updated_at = NOW()
RETURNING id, name, email, age, bio, is_active, score, balance, created_at, updated_at;
