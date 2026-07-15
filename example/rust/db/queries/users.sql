-- ============================================================
-- USERS queries  (PostgreSQL)
-- Covers: SELECT one/many, INSERT, UPDATE partial/full,
--         DELETE, EXISTS, COUNT, LIKE, pagination, BETWEEN,
--         date filter, aggregates, JOIN, GROUP BY, HAVING,
--         CASE, CTE, UPSERT, NULL handling, COALESCE.
-- ============================================================


-- ─── SELECT ──────────────────────────────────────────────────

-- name: GetUser :one
SELECT id, name, email, age, is_active, created_at, updated_at
FROM users
WHERE id = $1
LIMIT 1;


-- name: GetUserByEmail :one
SELECT id, name, email, age, is_active, created_at, updated_at
FROM users
WHERE email = $1
LIMIT 1;


-- name: ListUsers :many
SELECT id, name, email, age, is_active, created_at, updated_at
FROM users
ORDER BY created_at DESC;


-- name: ListActiveUsers :many
SELECT id, name, email, age, is_active, created_at, updated_at
FROM users
WHERE is_active = TRUE
ORDER BY name ASC;


-- name: SearchUsersByName :many
SELECT id, name, email, age, is_active, created_at, updated_at
FROM users
WHERE name ILIKE '%' || $1 || '%'
ORDER BY name ASC;


-- name: ListUsersPaginated :many
SELECT id, name, email, age, is_active, created_at, updated_at
FROM users
ORDER BY id ASC
LIMIT $1
OFFSET $2;


-- name: ListUsersInAgeRange :many
SELECT id, name, email, age, is_active, created_at, updated_at
FROM users
WHERE age BETWEEN $1 AND $2
ORDER BY age ASC;


-- name: GetUsersCreatedAfter :many
SELECT id, name, email, age, is_active, created_at, updated_at
FROM users
WHERE created_at > $1
ORDER BY created_at ASC;


-- ─── AGGREGATES ──────────────────────────────────────────────

-- name: CountUsers :one
SELECT COUNT(*) AS count FROM users;


-- name: CountActiveUsers :one
SELECT COUNT(*) AS count FROM users WHERE is_active = TRUE;


-- name: GetUserAgeStats :one
SELECT
    COUNT(*)    AS total,
    AVG(age)    AS avg_age,
    MIN(age)    AS min_age,
    MAX(age)    AS max_age
FROM users
WHERE age IS NOT NULL;


-- ─── INSERT ──────────────────────────────────────────────────

-- name: CreateUser :one
INSERT INTO users (name, email, age)
VALUES ($1, $2, $3)
RETURNING id, name, email, age, is_active, created_at, updated_at;


-- ─── UPDATE ──────────────────────────────────────────────────

-- name: UpdateUserName :one
UPDATE users
SET name       = $1,
    updated_at = NOW()
WHERE id = $2
RETURNING id, name, email, age, is_active, created_at, updated_at;


-- name: UpdateUserEmail :one
UPDATE users
SET email      = $1,
    updated_at = NOW()
WHERE id = $2
RETURNING id, name, email, age, is_active, created_at, updated_at;


-- name: UpdateUserFull :one
UPDATE users
SET name       = $1,
    email      = $2,
    age        = $3,
    is_active  = $4,
    updated_at = NOW()
WHERE id = $5
RETURNING id, name, email, age, is_active, created_at, updated_at;


-- name: DeactivateUser :one
UPDATE users
SET is_active  = FALSE,
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, email, age, is_active, created_at, updated_at;


-- name: ActivateUser :one
UPDATE users
SET is_active  = TRUE,
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, email, age, is_active, created_at, updated_at;


-- ─── DELETE ──────────────────────────────────────────────────

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;


-- name: DeleteInactiveUsers :exec
DELETE FROM users WHERE is_active = FALSE;


-- ─── EXISTS / SUB-QUERY ──────────────────────────────────────

-- name: UserExists :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE id = $1
) AS exists;


-- name: EmailTaken :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE email = $1
) AS exists;


-- name: GetUsersWithPosts :many
SELECT id, name, email, age, is_active, created_at, updated_at
FROM users
WHERE id IN (SELECT DISTINCT user_id FROM posts)
ORDER BY name ASC;


-- name: GetUsersWithoutPosts :many
SELECT id, name, email, age, is_active, created_at, updated_at
FROM users
WHERE id NOT IN (SELECT DISTINCT user_id FROM posts)
ORDER BY name ASC;


-- ─── JOIN ────────────────────────────────────────────────────

-- name: GetUserWithPostCount :one
SELECT
    u.id,
    u.name,
    u.email,
    COUNT(p.id) AS post_count
FROM users u
LEFT JOIN posts p ON p.user_id = u.id
WHERE u.id = $1
GROUP BY u.id, u.name, u.email;


-- name: ListUsersWithPostCount :many
SELECT
    u.id,
    u.name,
    u.email,
    u.is_active,
    COUNT(p.id) AS post_count
FROM users u
LEFT JOIN posts p ON p.user_id = u.id
GROUP BY u.id, u.name, u.email, u.is_active
ORDER BY post_count DESC;


-- ─── GROUP BY / HAVING ───────────────────────────────────────

-- name: GetActiveUsersByAgeGroup :many
SELECT
    age,
    COUNT(*) AS user_count
FROM users
WHERE is_active = TRUE
  AND age IS NOT NULL
GROUP BY age
HAVING COUNT(*) > $1
ORDER BY age ASC;


-- ─── CASE expression ─────────────────────────────────────────

-- name: ListUsersWithActivityLabel :many
SELECT
    id,
    name,
    email,
    CASE
        WHEN is_active = TRUE THEN 'active'
        ELSE 'inactive'
    END AS status
FROM users
ORDER BY name ASC;


-- ─── CTE ─────────────────────────────────────────────────────

-- name: GetTopPosters :many
WITH post_counts AS (
    SELECT user_id, COUNT(*) AS post_count
    FROM posts
    WHERE published = TRUE
    GROUP BY user_id
)
SELECT
    u.id,
    u.name,
    u.email,
    pc.post_count
FROM users u
INNER JOIN post_counts pc ON pc.user_id = u.id
ORDER BY pc.post_count DESC
LIMIT $1;


-- name: GetRecentActiveUsers :many
WITH recent AS (
    SELECT DISTINCT user_id
    FROM posts
    WHERE created_at > $1
)
SELECT
    u.id,
    u.name,
    u.email,
    u.created_at
FROM users u
INNER JOIN recent r ON r.user_id = u.id
ORDER BY u.name ASC;


-- ─── UPSERT ──────────────────────────────────────────────────

-- name: UpsertUser :one
INSERT INTO users (name, email, age, is_active)
VALUES ($1, $2, $3, TRUE)
ON CONFLICT (email) DO UPDATE
    SET name       = EXCLUDED.name,
        age        = EXCLUDED.age,
        is_active  = TRUE,
        updated_at = NOW()
RETURNING id, name, email, age, is_active, created_at, updated_at;


-- ─── NULL handling ───────────────────────────────────────────

-- name: ListUsersWithoutAge :many
SELECT id, name, email, age, is_active, created_at, updated_at
FROM users
WHERE age IS NULL
ORDER BY name ASC;


-- name: ListUsersWithAge :many
SELECT id, name, email, age, is_active, created_at, updated_at
FROM users
WHERE age IS NOT NULL
ORDER BY age ASC;


-- name: GetUserAgeOrDefault :one
SELECT
    id,
    name,
    email,
    COALESCE(age, 0) AS age,
    is_active,
    created_at,
    updated_at
FROM users
WHERE id = $1;


-- ─── COMPLEX CTEs ────────────────────────────────────────────

-- name: GetUserActivityReport :many
-- Full activity breakdown per user: posts, comments, views,
-- last active date — all in one query using multiple CTEs.
WITH user_posts AS (
    SELECT
        user_id,
        COUNT(*)            AS post_count,
        SUM(views)          AS total_views,
        MAX(created_at)     AS last_post_at
    FROM posts
    WHERE published = TRUE
    GROUP BY user_id
),
user_comments AS (
    SELECT
        user_id,
        COUNT(*)            AS comment_count,
        MAX(created_at)     AS last_comment_at
    FROM comments
    GROUP BY user_id
),
user_activity AS (
    SELECT
        u.id,
        GREATEST(
            COALESCE(up.last_post_at,    u.created_at),
            COALESCE(uc.last_comment_at, u.created_at)
        ) AS last_active_at
    FROM users u
    LEFT JOIN user_posts    up ON up.user_id = u.id
    LEFT JOIN user_comments uc ON uc.user_id = u.id
)
SELECT
    u.id,
    u.name,
    u.email,
    u.age,
    u.is_active,
    COALESCE(up.post_count,    0) AS post_count,
    COALESCE(up.total_views,   0) AS total_views,
    COALESCE(uc.comment_count, 0) AS comment_count,
    ua.last_active_at
FROM users u
LEFT JOIN user_posts    up ON up.user_id = u.id
LEFT JOIN user_comments uc ON uc.user_id = u.id
INNER JOIN user_activity ua ON ua.id     = u.id
ORDER BY ua.last_active_at DESC;


-- name: GetDormantUsers :many
-- CTE + subquery: active users who have not posted or commented
-- in the last N days.
WITH last_action AS (
    SELECT user_id, MAX(created_at) AS last_at
    FROM (
        SELECT user_id, created_at FROM posts
        UNION ALL
        SELECT user_id, created_at FROM comments
    ) actions
    GROUP BY user_id
)
SELECT
    u.id,
    u.name,
    u.email,
    u.created_at,
    la.last_at AS last_action_at
FROM users u
INNER JOIN last_action la ON la.user_id = u.id
WHERE u.is_active = TRUE
  AND la.last_at < NOW() - ($1 || ' days')::INTERVAL
ORDER BY la.last_at ASC;


-- name: GetUserRetentionCohorts :many
-- Cohort analysis: group users by signup month, count how many
-- posted at least once, compute retention rate.
WITH cohorts AS (
    SELECT
        id,
        DATE_TRUNC('month', created_at) AS cohort_month
    FROM users
),
cohort_active AS (
    SELECT DISTINCT
        u.id,
        DATE_TRUNC('month', u.created_at) AS cohort_month
    FROM users u
    INNER JOIN posts p ON p.user_id = u.id
    WHERE p.published = TRUE
)
SELECT
    c.cohort_month,
    COUNT(c.id)                                           AS total_users,
    COUNT(ca.id)                                          AS active_users,
    ROUND(100.0 * COUNT(ca.id) / NULLIF(COUNT(c.id), 0), 2) AS retention_pct
FROM cohorts c
LEFT JOIN cohort_active ca
       ON ca.id           = c.id
      AND ca.cohort_month = c.cohort_month
GROUP BY c.cohort_month
ORDER BY c.cohort_month ASC;


-- ─── COMPLEX SUBQUERIES ──────────────────────────────────────

-- name: GetUsersAboveAvgPostCount :many
-- Scalar subquery in WHERE: users whose post count beats the
-- average number of posts per user.
SELECT
    u.id,
    u.name,
    u.email,
    COUNT(p.id) AS post_count
FROM users u
INNER JOIN posts p ON p.user_id = u.id
WHERE p.published = TRUE
GROUP BY u.id, u.name, u.email
HAVING COUNT(p.id) > (
    SELECT AVG(cnt)
    FROM (
        SELECT COUNT(*) AS cnt
        FROM posts
        WHERE published = TRUE
        GROUP BY user_id
    ) sub
)
ORDER BY post_count DESC;


-- name: GetUsersWithRecentHighViewPost :many
-- Correlated subquery: users who have at least one post
-- created within the last $1 days with more than $2 views.
SELECT
    u.id,
    u.name,
    u.email,
    u.created_at
FROM users u
WHERE EXISTS (
    SELECT 1
    FROM posts p
    WHERE p.user_id   = u.id
      AND p.published = TRUE
      AND p.views     > $2
      AND p.created_at > NOW() - ($1 || ' days')::INTERVAL
)
ORDER BY u.name ASC;


-- name: GetUserRankByViews :many
-- Subquery-derived rank: each user's rank by total post views,
-- with their percentile in the overall user base.
SELECT
    id,
    name,
    email,
    total_views,
    RANK() OVER (ORDER BY total_views DESC) AS rank,
    ROUND(
        100.0 * RANK() OVER (ORDER BY total_views DESC)
              / NULLIF(COUNT(*) OVER (), 0),
    2) AS percentile
FROM (
    SELECT
        u.id,
        u.name,
        u.email,
        COALESCE(SUM(p.views), 0) AS total_views
    FROM users u
    LEFT JOIN posts p ON p.user_id = u.id AND p.published = TRUE
    GROUP BY u.id, u.name, u.email
) ranked
ORDER BY rank ASC;
