-- name: CreateUser :one
INSERT INTO users (name, email)
VALUES ($1, $2)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1;

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
