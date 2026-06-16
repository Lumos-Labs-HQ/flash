-- ===========================================================================
-- FlashORM Example Queries — PostgreSQL (comprehensive edge cases)
-- ===========================================================================
-- :one  = single row or null
-- :many = multiple rows (slice)
-- :exec = execute, no return
-- ===========================================================================

-- === BASIC CRUD ===

-- name: CreateUser :one
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING *;

-- name: CreateUserFull :one
INSERT INTO users (name, email, age, bio, preferences, tags, role)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUserName :one
UPDATE users SET name = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: UpdateUserRole :exec
UPDATE users SET role = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- === UPSERT / ON CONFLICT ===

-- name: UpsertUser :one
INSERT INTO users (name, email, role)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO UPDATE
SET name = EXCLUDED.name, updated_at = NOW()
RETURNING *;

-- name: UpsertUserWithCOALESCE :one
INSERT INTO users (name, email, bio)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO UPDATE
SET name = COALESCE(EXCLUDED.name, users.name),
    bio  = COALESCE(EXCLUDED.bio, users.bio),
    updated_at = NOW()
RETURNING *;

-- === NULL & COALESCE HANDLING ===

-- name: GetUsersWithNullAddress :many
SELECT id, name, email FROM users WHERE address IS NULL;

-- name: GetUsersWithBio :many
SELECT id, name, email, COALESCE(bio, '') AS bio FROM users WHERE bio IS NOT NULL;

-- name: GetUserDisplayInfo :one
SELECT id, name, email,
       COALESCE(address, 'No address provided') AS display_address,
       COALESCE(age, 0) AS age,
       COALESCE(bio, '') AS bio
FROM users WHERE id = $1;

-- name: SearchUsersWithCOALESCE :many
SELECT id, name, email, COALESCE(bio, 'No bio') AS bio_text
FROM users
WHERE (name ILIKE $1 OR $1 IS NULL)
  AND (email ILIKE $2 OR $2 IS NULL)
  AND COALESCE(age, 0) >= $3
ORDER BY name
LIMIT $4 OFFSET $5;

-- === BETWEEN & RANGE QUERIES ===

-- name: GetUsersCreatedBetween :many
SELECT id, name, email, created_at
FROM users
WHERE created_at >= $1 AND created_at <= $2
ORDER BY created_at DESC;

-- name: GetUsersByAgeRange :many
SELECT id, name, age, age_range
FROM users
WHERE age >= $1 AND age <= $2
ORDER BY age;

-- name: GetUsersByGeneratedRange :many
SELECT id, name, age, age_range
FROM users
WHERE age_range @> $1::integer;

-- name: GetRecentUsers :many
SELECT * FROM users WHERE created_at > $1 LIMIT $2 OFFSET $3;

-- === JSONB OPERATIONS ===

-- name: GetUserPreferences :one
SELECT id, name, preferences
FROM users WHERE id = $1;

-- name: UpdateUserPreferences :exec
UPDATE users SET preferences = preferences || $2, updated_at = NOW()
WHERE id = $1;

-- name: FindUsersByJsonKey :many
SELECT id, name, email, preferences
FROM users
WHERE preferences->>'theme' = $1;

-- name: FindUsersByJsonContains :many
SELECT id, name, email
FROM users
WHERE preferences @> $1::jsonb;

-- === ARRAY OPERATIONS ===

-- name: GetUsersWithTag :many
SELECT id, name, email, tags
FROM users
WHERE $1 = ANY(tags);

-- name: GetUsersWithAnyTag :many
SELECT id, name, email, tags
FROM users
WHERE tags && $1::text[];

-- name: AddUserTag :exec
UPDATE users SET tags = array_append(tags, $2), updated_at = NOW()
WHERE id = $1;

-- name: RemoveUserTag :exec
UPDATE users SET tags = array_remove(tags, $2), updated_at = NOW()
WHERE id = $1;

-- === COMPOSITE TYPE ACCESS ===

-- name: GetUserShippingAddress :one
SELECT id, name, shipping, (shipping).city AS shipping_city, (shipping).country AS shipping_country
FROM users WHERE id = $1;

-- name: UpdateUserShipping :exec
UPDATE users SET shipping = ROW($2, $3, $4, $5, $6), updated_at = NOW()
WHERE id = $1;

-- === CTE (Common Table Expression) EDGE CASES ===

-- name: GetComplexUserAnalytics :many
WITH user_post_stats AS (
    SELECT
        u.id AS user_id,
        u.name,
        u.email,
        u.role,
        u.isadmin,
        u.created_at AS user_created_at,
        COUNT(DISTINCT p.id) AS total_posts,
        COUNT(DISTINCT CASE WHEN p.status = 'published' THEN p.id END) AS published_posts,
        COUNT(DISTINCT CASE WHEN p.status = 'draft' THEN p.id END) AS draft_posts,
        MAX(p.created_at) AS last_post_date,
        AVG(LENGTH(p.content)) AS avg_post_length
    FROM users u
    LEFT JOIN posts p ON u.id = p.user_id
    GROUP BY u.id, u.name, u.email, u.role, u.isadmin, u.created_at
),
user_comment_stats AS (
    SELECT
        u.id AS user_id,
        COUNT(c.id) AS total_comments,
        COUNT(DISTINCT c.post_id) AS posts_commented_on,
        MAX(c.created_at) AS last_comment_date
    FROM users u
    LEFT JOIN comments c ON u.id = c.user_id
    GROUP BY u.id
),
category_engagement AS (
    SELECT
        p.user_id,
        COUNT(DISTINCT p.category_id) AS categories_used,
        STRING_AGG(DISTINCT cat.name, ', ' ORDER BY cat.name) AS category_names
    FROM posts p
    INNER JOIN categories cat ON p.category_id = cat.id
    GROUP BY p.user_id
)
SELECT
    ups.user_id AS id,
    ups.name,
    ups.email,
    ups.role,
    ups.isadmin,
    ups.user_created_at,
    COALESCE(ups.total_posts, 0) AS total_posts,
    COALESCE(ups.published_posts, 0) AS published_posts,
    COALESCE(ups.draft_posts, 0) AS draft_posts,
    COALESCE(ucs.total_comments, 0) AS total_comments,
    COALESCE(ucs.posts_commented_on, 0) AS posts_commented_on,
    COALESCE(ce.categories_used, 0) AS categories_used,
    COALESCE(ce.category_names, '') AS category_names,
    ups.last_post_date,
    ucs.last_comment_date,
    COALESCE(ups.avg_post_length, 0)::NUMERIC(10,2) AS avg_post_length,
    CASE
        WHEN ups.total_posts > 10 AND ucs.total_comments > 20 THEN 'highly_active'
        WHEN ups.total_posts > 5 OR ucs.total_comments > 10 THEN 'active'
        WHEN ups.total_posts > 0 OR ucs.total_comments > 0 THEN 'casual'
        ELSE 'inactive'
    END AS activity_level,
    (COALESCE(ups.total_posts, 0) + COALESCE(ucs.total_comments, 0)) AS engagement_score
FROM user_post_stats ups
LEFT JOIN user_comment_stats ucs ON ups.user_id = ucs.user_id
LEFT JOIN category_engagement ce ON ups.user_id = ce.user_id
WHERE ups.total_posts > $1 OR ucs.total_comments > $2
ORDER BY engagement_score DESC, ups.last_post_date DESC NULLS LAST
LIMIT $3;

-- name: GetPostWithActiveCommenters :many
WITH active_commenters AS (
    SELECT c.post_id, c.user_id, u.name AS commenter_name, c.created_at,
           ROW_NUMBER() OVER (PARTITION BY c.post_id ORDER BY c.created_at DESC) AS rn
    FROM comments c
    JOIN users u ON c.user_id = u.id
)
SELECT ac.commenter_name, ac.created_at AS last_comment_at
FROM active_commenters ac
WHERE ac.rn <= $2 AND ac.post_id = $1
ORDER BY ac.created_at DESC;

-- === WINDOW FUNCTIONS ===

-- name: GetUserPostRankings :many
SELECT
    u.id,
    u.name,
    COUNT(p.id) AS post_count,
    RANK() OVER (ORDER BY COUNT(p.id) DESC) AS post_rank,
    DENSE_RANK() OVER (ORDER BY COUNT(p.id) DESC) AS dense_post_rank,
    ROW_NUMBER() OVER (ORDER BY COUNT(p.id) DESC, u.name ASC) AS row_num
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
GROUP BY u.id, u.name
ORDER BY post_count DESC
LIMIT $1;

-- name: GetUserTrendingPosts :many
SELECT
    p.id,
    p.title,
    p.user_id,
    p.view_count,
    p.created_at,
    LAG(p.view_count) OVER (PARTITION BY p.user_id ORDER BY p.created_at) AS prev_view_count,
    LEAD(p.view_count) OVER (PARTITION BY p.user_id ORDER BY p.created_at) AS next_view_count,
    p.view_count - LAG(p.view_count) OVER (PARTITION BY p.user_id ORDER BY p.created_at) AS view_delta
FROM posts p
WHERE p.user_id = $1
ORDER BY p.created_at DESC
LIMIT $2;

-- === SUBQUERIES & CORRELATED SUBQUERIES ===

-- name: GetPostCountByUser :one
SELECT
    (SELECT COUNT(*) FROM posts WHERE user_id = $1) AS post_count,
    (SELECT COUNT(*) FROM comments WHERE user_id = $1) AS comment_count;

-- name: GetUsersWithManyPosts :many
SELECT id, name, email,
       (SELECT COUNT(*) FROM posts p WHERE p.user_id = u.id) AS total_posts
FROM users u
WHERE (SELECT COUNT(*) FROM posts WHERE user_id = u.id) > $1
ORDER BY total_posts DESC;

-- name: GetPostsWithCommentCount :many
SELECT p.id, p.title, p.created_at,
       (SELECT COUNT(*) FROM comments c WHERE c.post_id = p.id) AS comment_count,
       (SELECT COUNT(DISTINCT c2.user_id) FROM comments c2 WHERE c2.post_id = p.id) AS unique_commenters,
       (SELECT MAX(c3.created_at) FROM comments c3 WHERE c3.post_id = p.id) AS last_comment_at
FROM posts p
WHERE p.status = 'published'
ORDER BY comment_count DESC
LIMIT $1 OFFSET $2;

-- === CASE EXPRESSIONS ===

-- name: GetUsersWithActivityLevel :many
SELECT id, name, email, created_at,
       CASE
           WHEN created_at >= NOW() - INTERVAL '7 days' THEN 'new'
           WHEN created_at >= NOW() - INTERVAL '30 days' THEN 'recent'
           WHEN created_at >= NOW() - INTERVAL '1 year' THEN 'established'
           ELSE 'veteran'
       END AS account_age_category,
       CASE WHEN isadmin THEN 'administrator'
            ELSE role::TEXT
       END AS effective_role
FROM users
ORDER BY created_at DESC;

-- === JOIN VARIANTS ===

-- name: GetPostWithComments :many
SELECT p.id AS post_id, p.title, p.content, u.name AS author,
       c.content AS comment_text, cu.name AS commenter
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN comments c ON p.id = c.post_id
LEFT JOIN users cu ON c.user_id = cu.id
WHERE p.id = $1;

-- name: GetPostDetailsWithAllRelations :one
SELECT
    p.id, p.title, p.content, p.status, p.created_at, p.updated_at,
    u.id AS author_id, u.name AS author_name, u.email AS author_email, u.role AS author_role,
    u.isadmin AS author_is_admin,
    cat.id AS category_id, cat.name AS category_name,
    COUNT(DISTINCT c.id) AS comment_count,
    COUNT(DISTINCT c.user_id) AS unique_commenters,
    STRING_AGG(DISTINCT c.content, ' | ' ORDER BY c.content) AS all_comments,
    ARRAY_AGG(DISTINCT cu.name ORDER BY cu.name) AS commenter_names,
    MAX(c.created_at) AS last_comment_date,
    LENGTH(p.content) AS content_length,
    EXTRACT(EPOCH FROM (NOW() - p.created_at)) / 3600 AS hours_since_created
FROM posts p
INNER JOIN users u ON p.user_id = u.id
INNER JOIN categories cat ON p.category_id = cat.id
LEFT JOIN comments c ON p.id = c.post_id
LEFT JOIN users cu ON c.user_id = cu.id
WHERE p.id = $1
GROUP BY p.id, p.title, p.content, p.status, p.created_at, p.updated_at,
         u.id, u.name, u.email, u.role, u.isadmin, cat.id, cat.name;

-- === AGGREGATE & GROUPING EDGE CASES ===

-- name: CountUsersByRole :one
SELECT COUNT(*) FROM users WHERE role = $1;

-- name: CountUsers :one
SELECT COUNT(*) AS total_users,
       COUNT(CASE WHEN isadmin = TRUE THEN 1 END) AS admin_count,
       COUNT(CASE WHEN isadmin = FALSE THEN 1 END) AS regular_count
FROM users;

-- name: GetUserRoleCount :many
SELECT role, COUNT(*) AS count
FROM users
GROUP BY role
ORDER BY count DESC;

-- name: GetUserAgeStats :one
SELECT
    MIN(created_at) AS first_joined,
    MAX(created_at) AS last_joined,
    COUNT(*) AS total,
    AVG(COALESCE(age, 0))::NUMERIC(10,2) AS avg_age,
    AVG(LENGTH(COALESCE(name, '')))::NUMERIC(10,2) AS avg_name_length
FROM users;

-- name: GetPostsGroupedByStatus :many
SELECT status, COUNT(*) AS count, MIN(created_at) AS oldest, MAX(created_at) AS newest
FROM posts
GROUP BY status
HAVING COUNT(*) > $1
ORDER BY count DESC;

-- === DISTINCT & DISTINCT ON ===

-- name: GetDistinctCommenters :many
SELECT DISTINCT u.id, u.name, u.email
FROM users u
JOIN comments c ON u.id = c.user_id
ORDER BY u.name;

-- name: GetLatestPostPerUser :many
SELECT DISTINCT ON (user_id)
    user_id, id AS post_id, title, status, created_at
FROM posts
ORDER BY user_id, created_at DESC;

-- === TEXT SEARCH ===

-- name: SearchUsers :many
SELECT id, name, email
FROM users
WHERE name ILIKE $1 OR email ILIKE $2
ORDER BY name ASC
LIMIT $3 OFFSET $4;

-- name: SearchPostsByTitle :many
SELECT id, title, status, created_at
FROM posts
WHERE title ILIKE $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: FullTextSearchPosts :many
SELECT id, title, ts_rank(to_tsvector('english', title || ' ' || content),
                           plainto_tsquery('english', $1)) AS rank
FROM posts
WHERE to_tsvector('english', title || ' ' || content) @@ plainto_tsquery('english', $1)
ORDER BY rank DESC
LIMIT $2;

-- === EXTRACT / DATE FUNCTIONS ===

-- name: GetUserRegistrationStats :many
SELECT
    EXTRACT(YEAR FROM created_at)::INT AS year,
    EXTRACT(MONTH FROM created_at)::INT AS month,
    COUNT(*) AS signups
FROM users
GROUP BY EXTRACT(YEAR FROM created_at), EXTRACT(MONTH FROM created_at)
ORDER BY year DESC, month DESC;

-- name: GetWeeklyPostStats :many
SELECT
    DATE_TRUNC('week', created_at) AS week_start,
    COUNT(*) AS posts_created,
    SUM(view_count) AS total_views
FROM posts
WHERE created_at >= $1
GROUP BY DATE_TRUNC('week', created_at)
ORDER BY week_start DESC;

-- === IN / ANY / EXISTS ===

-- name: GetUsersInIds :many
SELECT * FROM users WHERE id = ANY($1::bigint[]);

-- name: GetUsersByNames :many
SELECT id, name, email FROM users WHERE name IN ($1, $2, $3);

-- name: GetUsersWhoCommented :many
SELECT id, name, email
FROM users u
WHERE EXISTS (SELECT 1 FROM comments c WHERE c.user_id = u.id)
ORDER BY u.name;

-- name: GetUsersWithNoPosts :many
SELECT id, name, email
FROM users u
WHERE NOT EXISTS (SELECT 1 FROM posts p WHERE p.user_id = u.id)
ORDER BY u.created_at DESC;

-- === UNION / UNION ALL ===

-- name: GetAllContentByUser :many
SELECT 'post' AS content_type, id::TEXT AS content_id, title AS content_summary, created_at
FROM posts WHERE user_id = $1
UNION ALL
SELECT 'comment' AS content_type, id::TEXT AS content_id, LEFT(content, 100) AS content_summary, created_at
FROM comments WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- === VIEW & MATERIALIZED VIEW ===

-- name: GetActiveUsers :many
SELECT * FROM active_users ORDER BY created_at DESC LIMIT $1;

-- name: GetUserActivitySummary :many
SELECT * FROM user_activity_summary WHERE post_count > $1 OR comment_count > $2 ORDER BY post_count DESC;

-- name: RefreshPostStats :exec
REFRESH MATERIALIZED VIEW CONCURRENTLY post_stats;

-- name: GetPostStats :many
SELECT * FROM post_stats ORDER BY comment_count DESC LIMIT $1;

-- === SUBSCRIPTION & ORDERS ===

-- name: GetUserSubscriptions :many
SELECT s.id, s.tier, s.started_at, s.expires_at, s.auto_renew
FROM subscriptions s
WHERE s.user_id = $1
ORDER BY s.started_at DESC;

-- name: CreateSubscription :one
INSERT INTO subscriptions (user_id, tier, expires_at, auto_renew)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetOrdersByUser :many
SELECT id, total_amount, discount_pct, state, shipping_addr, placed_at
FROM orders
WHERE user_id = $1
ORDER BY placed_at DESC
LIMIT $2;

-- name: GetOrdersInState :many
SELECT o.id, o.user_id, u.name AS user_name, o.total_amount, o.state, o.placed_at
FROM orders o
JOIN users u ON o.user_id = u.id
WHERE o.state = $1
ORDER BY o.placed_at DESC
LIMIT $2;

-- === TRIGGER-POPULATED DATA ===

-- name: GetAuditLogForUser :many
SELECT id, table_name, record_id, action, old_data, new_data, changed_at
FROM audit_log
WHERE changed_by = $1
ORDER BY changed_at DESC
LIMIT $2 OFFSET $3;

-- name: GetAuditLogForTable :many
SELECT id, table_name, record_id, action, changed_by, changed_at
FROM audit_log
WHERE table_name = $1
ORDER BY changed_at DESC
LIMIT $2;

-- === STATISTICS & AGGREGATES ===

-- name: GetDashboardStats :one
SELECT
    (SELECT COUNT(*) FROM users) AS total_users,
    (SELECT COUNT(*) FROM posts) AS total_posts,
    (SELECT COUNT(*) FROM comments) AS total_comments,
    (SELECT COUNT(*) FROM posts WHERE status = 'published') AS published_posts,
    (SELECT COUNT(*) FROM posts WHERE created_at >= NOW() - INTERVAL '7 days') AS posts_this_week,
    (SELECT COUNT(*) FROM users WHERE created_at >= NOW() - INTERVAL '7 days') AS signups_this_week,
    (SELECT COUNT(*) FROM comments WHERE created_at >= NOW() - INTERVAL '24 hours') AS comments_last_24h,
    (SELECT COUNT(*) FROM orders WHERE state = 'pending') AS pending_orders;

-- name: GetTopCommenters :many
SELECT u.id, u.name, u.email, COUNT(c.id) AS comment_count,
       RANK() OVER (ORDER BY COUNT(c.id) DESC) AS rank
FROM users u
JOIN comments c ON u.id = c.user_id
GROUP BY u.id, u.name, u.email
ORDER BY comment_count DESC
LIMIT $1;

-- name: GetEngagementTimeSeries :many
SELECT
    DATE_TRUNC('day', created_at) AS day,
    COUNT(*) AS count,
    'post' AS event_type
FROM posts WHERE created_at >= $1
GROUP BY DATE_TRUNC('day', created_at)
UNION ALL
SELECT
    DATE_TRUNC('day', created_at) AS day,
    COUNT(*) AS count,
    'comment' AS event_type
FROM comments WHERE created_at >= $1
GROUP BY DATE_TRUNC('day', created_at)
ORDER BY day DESC;

-- === OLD QUERIES (preserved from original) ===

-- name: CreateCategory :one
INSERT INTO categories (name) VALUES ($1) RETURNING *;

-- name: CreatePost :one
INSERT INTO posts (user_id, category_id, title, content) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: CreateComment :one
INSERT INTO comments (post_id, user_id, content) VALUES ($1, $2, $3) RETURNING *;

-- name: DeleteOldUsers :exec
DELETE FROM users WHERE created_at < $1 AND isadmin = FALSE;

-- name: UpdateUserTimestamp :exec
UPDATE users SET updated_at = $1 WHERE id = $2;
