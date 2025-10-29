-- name: IsadminUser :one
SELECT isadmin FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserEmail :one
-- Get a single user's email address
SELECT email FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserName :one
-- Get a single user's name
SELECT name FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserCreatedAt :one
-- Get when a user was created
SELECT created_at FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserIdByEmail :one
-- Get user ID by email
SELECT id FROM users WHERE email = $1 LIMIT 1;

-- name: GetAllUserEmails :many
-- Get all user email addresses
SELECT email FROM users;

-- name: GetAllUserIds :many
-- Get all user IDs
SELECT id FROM users;

-- name: GetAdminUserEmails :many
-- Get emails of all admin users
SELECT email FROM users WHERE isadmin = true;

-- name: GetUserFullInfo :one
-- Get complete user information (returns full Users object)
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: GetAllUsers :many
-- Get all users with complete information
SELECT * FROM users;

-- name: GetUserEmailAndName :one
-- Get user's email and name (returns object with two fields)
SELECT email, name FROM users WHERE id = $1 LIMIT 1;

-- name: CountUsers :one
-- Count total users
SELECT COUNT(*) as count FROM users;

-- name: GetRecentUsers :many
-- Get recently created users
SELECT id, name, email, created_at FROM users ORDER BY created_at DESC LIMIT 10;

-- COMPLEX QUERIES - Production Level

-- name: GetUserWithPostCount :one
-- Get user with their total post count (JOIN + aggregation)
SELECT 
    u.id, 
    u.name, 
    u.email, 
    u.isadmin,
    COUNT(p.id) as post_count
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
WHERE u.id = $1
GROUP BY u.id, u.name, u.email, u.isadmin
LIMIT 1;

-- name: GetUsersWithPostCounts :many
-- Get all users with their post counts
SELECT 
    u.id, 
    u.name, 
    u.email,
    COUNT(p.id) as post_count
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
GROUP BY u.id, u.name, u.email
ORDER BY post_count DESC;

-- name: GetUserPostsWithCategory :many
-- Get user's posts with category information (multiple JOINs)
SELECT 
    p.id as post_id,
    p.title,
    p.content,
    p.created_at,
    c.name as category_name,
    c.id as category_id
FROM posts p
INNER JOIN users u ON p.user_id = u.id
INNER JOIN categories c ON p.category_id = c.id
WHERE u.id = $1
ORDER BY p.created_at DESC;

-- name: GetPostWithCommentCount :one
-- Get post details with comment count
SELECT 
    p.id,
    p.title,
    p.content,
    u.name as author_name,
    c.name as category_name,
    COUNT(cm.id) as comment_count
FROM posts p
INNER JOIN users u ON p.user_id = u.id
INNER JOIN categories c ON p.category_id = c.id
LEFT JOIN comments cm ON p.id = cm.post_id
WHERE p.id = $1
GROUP BY p.id, p.title, p.content, u.name, c.name
LIMIT 1;

-- name: GetActiveUsersWithStats :many
-- Get users who have posted, with their statistics (complex aggregation)
SELECT 
    u.id,
    u.name,
    u.email,
    u.isadmin,
    COUNT(DISTINCT p.id) as total_posts,
    COUNT(DISTINCT c.id) as total_comments,
    MAX(p.created_at) as last_post_date
FROM users u
INNER JOIN posts p ON u.id = p.user_id
LEFT JOIN comments c ON u.id = c.user_id
GROUP BY u.id, u.name, u.email, u.isadmin
HAVING COUNT(DISTINCT p.id) > 0
ORDER BY total_posts DESC;

-- name: GetUserEngagementScore :one
-- Calculate user engagement score (complex calculation)
SELECT 
    u.id,
    u.name,
    (COUNT(DISTINCT p.id) * 10 + COUNT(DISTINCT c.id) * 5) as engagement_score
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
LEFT JOIN comments c ON u.id = c.user_id
WHERE u.id = $1
GROUP BY u.id, u.name
LIMIT 1;

-- name: SearchUsersByName :many
-- Search users by name pattern (LIKE query)
SELECT id, name, email, created_at
FROM users
WHERE name ILIKE $1
ORDER BY name
LIMIT 50;

-- name: GetTopContributors :many
-- Get top contributors (users with most posts and comments)
SELECT 
    u.id,
    u.name,
    u.email,
    COUNT(DISTINCT p.id) as posts,
    COUNT(DISTINCT c.id) as comments,
    (COUNT(DISTINCT p.id) + COUNT(DISTINCT c.id)) as total_contributions
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
LEFT JOIN comments c ON u.id = c.user_id
GROUP BY u.id, u.name, u.email
HAVING (COUNT(DISTINCT p.id) + COUNT(DISTINCT c.id)) > 0
ORDER BY total_contributions DESC
LIMIT $1;

-- name: GetCategoryWithPostStats :many
-- Get categories with post and comment statistics
SELECT 
    c.id,
    c.name,
    COUNT(DISTINCT p.id) as post_count,
    COUNT(DISTINCT cm.id) as comment_count,
    COUNT(DISTINCT p.user_id) as unique_authors
FROM categories c
LEFT JOIN posts p ON c.id = p.category_id
LEFT JOIN comments cm ON p.id = cm.post_id
GROUP BY c.id, c.name
ORDER BY post_count DESC;

-- name: GetUserRecentActivity :many
-- Get user's recent posts and comments combined
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
-- Calculate average posts per user (aggregate function)
SELECT 
    AVG(post_count) as avg_posts
FROM (
    SELECT COUNT(p.id) as post_count
    FROM users u
    LEFT JOIN posts p ON u.id = p.user_id
    GROUP BY u.id
) as user_posts;

-- name: GetMostCommentedPosts :many
-- Get posts with most comments (subquery)
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
-- Check if user exists by email (boolean result)
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1) as exists;

-- name: GetUsersCreatedBetween :many
-- Get users created within a date range
SELECT id, name, email, created_at
FROM users
WHERE created_at BETWEEN $1 AND $2
ORDER BY created_at DESC;

-- name: UpdateUserAdminStatus :exec
-- Update user admin status
UPDATE users 
SET isadmin = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeleteInactiveUsers :exec
-- Delete users with no posts or comments
DELETE FROM users 
WHERE id NOT IN (
    SELECT DISTINCT user_id FROM posts
    UNION
    SELECT DISTINCT user_id FROM comments
)
AND created_at < $1;

-- name: CreateUser :exec
-- Insert a new user
INSERT INTO users (name, email, address, isadmin)
VALUES ($1, $2, $3, $4);

-- name: CreatePost :exec
-- Insert a new post
INSERT INTO posts (user_id, category_id, title, content)
VALUES ($1, $2, $3, $4);

-- name: CreateComment :exec
-- Insert a new comment
INSERT INTO comments (post_id, user_id, content)
VALUES ($1, $2, $3);

-- name: CreateCategory :exec
-- Insert a new category
INSERT INTO categories (name)
VALUES ($1);

-- name: GetPostsByCategory :many
-- Get all posts for a specific category
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
