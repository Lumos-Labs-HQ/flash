-- name: IsadminUser :one
SELECT isadmin FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserEmail :one
SELECT email FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserName :one
SELECT name FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;


-- name: GetActiveUsersWithStats :many
WITH post_stats AS (
    SELECT 
        user_id, 
        COUNT(*) as total_posts, 
        MAX(created_at) as last_post_date
    FROM posts
    GROUP BY user_id
),
comment_stats AS (
    SELECT 
        user_id, 
        COUNT(*) as total_comments
    FROM comments
    GROUP BY user_id
)
SELECT 
    u.id,
    u.name,
    u.email,
    u.isadmin,
    ps.total_posts,
    COALESCE(cs.total_comments, 0) as total_comments,
    ps.last_post_date
FROM users u
INNER JOIN post_stats ps ON u.id = ps.user_id
LEFT JOIN comment_stats cs ON u.id = cs.user_id
ORDER BY ps.total_posts DESC;

-- name: GetTopActiveUsers :many
WITH ranked_users AS (
    SELECT 
        user_id,
        COUNT(*) as total_posts
    FROM posts
    GROUP BY user_id
    ORDER BY total_posts DESC
    LIMIT 10
)
SELECT 
    u.id,
    u.name,
    u.email,
    ru.total_posts,
    COALESCE(cs.total_comments, 0) as total_comments
FROM ranked_users ru
INNER JOIN users u ON u.id = ru.user_id
LEFT JOIN (
    SELECT user_id, COUNT(*) as total_comments
    FROM comments
    WHERE user_id IN (SELECT user_id FROM ranked_users)
    GROUP BY user_id
) cs ON u.id = cs.user_id
ORDER BY ru.total_posts DESC;


-- name: CreateUser :exec
INSERT INTO users (name, email, address, isadmin)
VALUES ($1, $2, $3, $4);

-- name: CreatePost :exec
INSERT INTO posts (user_id, category_id, title, content)
VALUES ($1, $2, $3, $4);

-- name: CreateComment :exec
INSERT INTO comments (post_id, user_id, content)
VALUES ($1, $2, $3);

-- name: CreateCategory :exec
INSERT INTO categories (name)
VALUES ($1);

SELECT 
    p.id,
    p.title,
    p.content,
    p.created_at,
    'post' as activity_type
FROM posts p
WHERE p.user_id = $1
UNION ALL
SELECT 
    c.id,
    'Comment' as title,
    c.content,
    c.created_at,
    'comment' as activity_type
FROM comments c
WHERE c.user_id = $1
ORDER BY created_at DESC
LIMIT 20;

-- name: GetAveragePostsPerUser :one
SELECT 
    AVG(post_count) as avg_posts
FROM (
    SELECT COUNT(p.id) as post_count
    FROM users u
    LEFT JOIN posts p ON u.id = p.user_id
    GROUP BY u.id
) as user_posts;

-- name: GetMostCommentedPosts :many
SELECT 
    p.id,
    p.title,
    u.name as author,
    COUNT(c.id) as comment_count
FROM posts p
INNER JOIN users u ON p.user_id = u.id
LEFT JOIN comments c ON p.id = c.post_id
GROUP BY p.id, p.title, u.name
ORDER BY comment_count DESC
LIMIT $1;

-- name: CheckUserExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1) as exists;

-- name: GetUsersCreatedBetween :many
SELECT id, name, email, created_at
FROM users
WHERE created_at BETWEEN $1 AND $2
ORDER BY created_at DESC;

-- name: UpdateUserAdminStatus :exec
UPDATE users 
SET isadmin = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeleteInactiveUsers :exec
DELETE FROM users 
WHERE id NOT IN (
    SELECT DISTINCT user_id FROM posts
    UNION
    SELECT DISTINCT user_id FROM comments
)
AND created_at < $1;

-- name: CreateUser :exec
INSERT INTO users (name, email, address, isadmin)
VALUES ($1, $2, $3, $4);

-- name: CreatePost :exec
INSERT INTO posts (user_id, category_id, title, content)
VALUES ($1, $2, $3, $4);

-- name: CreateComment :exec
INSERT INTO comments (post_id, user_id, content)
VALUES ($1, $2, $3);

-- name: CreateCategory :exec
INSERT INTO categories (name)
VALUES ($1);

-- name: GetPostsByCategory :many
SELECT 
    p.id,
    p.title,
    p.content,
    u.name as author_name,
    p.created_at
FROM posts p
INNER JOIN users u ON p.user_id = u.id
WHERE p.category_id = $1
ORDER BY p.created_at DESC;
