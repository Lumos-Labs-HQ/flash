-- name: CreateUser :exec
INSERT INTO users (name, email, address, isadmin)
VALUES ($1, $2, $3, $4);

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: CreateCategory :one
INSERT INTO categories (name)
VALUES ($1)
RETURNING *;

-- name: CreatePost :one
INSERT INTO posts (user_id, category_id, title, content)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateComment :one
INSERT INTO comments (post_id, user_id, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetPostWithComments :many
SELECT p.id AS post_id, p.title, p.content, u.name AS author, c.content AS comment_text, cu.name AS commenter
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN comments c ON p.id = c.post_id
LEFT JOIN users cu ON c.user_id = cu.id
WHERE p.id = $1;

-- name: GetPostDetailsWithAllRelations :one
SELECT 
    p.id,
    p.title,
    p.content,
    p.status,
    p.created_at,
    p.updated_at,
    u.id as author_id,
    u.name as author_name,
    u.email as author_email,
    u.role as author_role,
    u.isadmin as author_is_admin,
    cat.id as category_id,
    cat.name as category_name,
    COUNT(DISTINCT c.id) as comment_count,
    COUNT(DISTINCT c.user_id) as unique_commenters,
    STRING_AGG(DISTINCT c.content, ' | ' ORDER BY c.content) as all_comments,
    ARRAY_AGG(DISTINCT cu.name ORDER BY cu.name) as commenter_names,
    MAX(c.created_at) as last_comment_date,
    LENGTH(p.content) as content_length,
    EXTRACT(EPOCH FROM (NOW() - p.created_at)) / 3600 as hours_since_created
FROM posts p
INNER JOIN users u ON p.user_id = u.id
INNER JOIN categories cat ON p.category_id = cat.id
LEFT JOIN comments c ON p.id = c.post_id
LEFT JOIN users cu ON c.user_id = cu.id
WHERE p.id = $1
GROUP BY 
    p.id, p.title, p.content, p.status, p.created_at, p.updated_at,
    u.id, u.name, u.email, u.role, u.isadmin,
    cat.id, cat.name;

-- name: GetComplexUserAnalytics :many
WITH user_post_stats AS (
    SELECT 
        u.id as user_id,
        u.name,
        u.email,
        u.role,
        u.isadmin,
        u.created_at as user_created_at,
        COUNT(DISTINCT p.id) as total_posts,
        COUNT(DISTINCT CASE WHEN p.status = 'published' THEN p.id END) as published_posts,
        COUNT(DISTINCT CASE WHEN p.status = 'draft' THEN p.id END) as draft_posts,
        MAX(p.created_at) as last_post_date,
        AVG(LENGTH(p.content)) as avg_post_length
    FROM users u
    LEFT JOIN posts p ON u.id = p.user_id
    GROUP BY u.id, u.name, u.email, u.role, u.isadmin, u.created_at
),
user_comment_stats AS (
    SELECT 
        u.id as user_id,
        COUNT(c.id) as total_comments,
        COUNT(DISTINCT c.post_id) as posts_commented_on,
        MAX(c.created_at) as last_comment_date
    FROM users u
    LEFT JOIN comments c ON u.id = c.user_id
    GROUP BY u.id
),
category_engagement AS (
    SELECT 
        p.user_id,
        COUNT(DISTINCT p.category_id) as categories_used,
        STRING_AGG(DISTINCT cat.name, ', ' ORDER BY cat.name) as category_names
    FROM posts p
    INNER JOIN categories cat ON p.category_id = cat.id
    GROUP BY p.user_id
)
SELECT 
    ups.user_id as id,
    ups.name,
    ups.email,
    ups.role,
    ups.isadmin,
    ups.user_created_at,
    COALESCE(ups.total_posts, 0) as total_posts,
    COALESCE(ups.published_posts, 0) as published_posts,
    COALESCE(ups.draft_posts, 0) as draft_posts,
    COALESCE(ucs.total_comments, 0) as total_comments,
    COALESCE(ucs.posts_commented_on, 0) as posts_commented_on,
    COALESCE(ce.categories_used, 0) as categories_used,
    COALESCE(ce.category_names, '') as category_names,
    ups.last_post_date,
    ucs.last_comment_date,
    COALESCE(ups.avg_post_length, 0)::NUMERIC(10,2) as avg_post_length,
    CASE 
        WHEN ups.total_posts > 10 AND ucs.total_comments > 20 THEN 'highly_active'
        WHEN ups.total_posts > 5 OR ucs.total_comments > 10 THEN 'active'
        WHEN ups.total_posts > 0 OR ucs.total_comments > 0 THEN 'casual'
        ELSE 'inactive'
    END as activity_level,
    (COALESCE(ups.total_posts, 0) + COALESCE(ucs.total_comments, 0)) as engagement_score
FROM user_post_stats ups
LEFT JOIN user_comment_stats ucs ON ups.user_id = ucs.user_id
LEFT JOIN category_engagement ce ON ups.user_id = ce.user_id
WHERE ups.total_posts > $1 OR ucs.total_comments > $2
ORDER BY engagement_score DESC, ups.last_post_date DESC NULLS LAST
LIMIT $3;
