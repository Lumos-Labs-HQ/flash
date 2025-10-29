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
