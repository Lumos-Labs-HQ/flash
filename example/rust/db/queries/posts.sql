-- ============================================================
-- Posts — basic CRUD
-- ============================================================

-- name: GetPost :one
SELECT id, user_id, title, body, views, published, created_at, updated_at
FROM posts WHERE id = $1;

-- name: ListPosts :many
SELECT id, user_id, title, body, views, published, created_at, updated_at
FROM posts
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListPublishedPosts :many
SELECT id, user_id, title, body, views, published, created_at
FROM posts
WHERE published = TRUE
ORDER BY created_at DESC;

-- name: ListPostsByUser :many
SELECT id, title, body, views, published, created_at
FROM posts
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: SearchPostsByTitle :many
SELECT id, title, views, published, created_at
FROM posts
WHERE title ILIKE '%' || $1 || '%'
ORDER BY views DESC;

-- name: CreatePost :one
INSERT INTO posts (user_id, title, body)
VALUES ($1, $2, $3)
RETURNING id, user_id, title, body, views, published, created_at, updated_at;

-- name: UpdatePost :exec
UPDATE posts
SET title = $2, body = $3, updated_at = NOW()
WHERE id = $1;

-- name: PublishPost :exec
UPDATE posts SET published = TRUE, updated_at = NOW() WHERE id = $1;

-- name: DeletePost :exec
DELETE FROM posts WHERE id = $1;

-- name: IncrementPostViews :execresult
UPDATE posts SET views = views + 1 WHERE id = $1;

-- name: DeletePostsByUser :execresult
DELETE FROM posts WHERE user_id = $1;

-- name: CountPosts :one
SELECT COUNT(*) FROM posts;

-- name: CountPublishedPosts :one
SELECT COUNT(*) FROM posts WHERE published = TRUE;

-- name: CountPostsByUser :one
SELECT COUNT(*) FROM posts WHERE user_id = $1;

-- name: GetTopPostsByViews :many
SELECT id, title, views, user_id, created_at
FROM posts
WHERE published = TRUE
ORDER BY views DESC
LIMIT $1;

-- ============================================================
-- Comments — basic operations
-- ============================================================

-- name: GetComment :one
SELECT id, post_id, user_id, parent_id, body, created_at
FROM comments WHERE id = $1;

-- name: ListCommentsByPost :many
SELECT id, user_id, parent_id, body, created_at
FROM comments
WHERE post_id = $1
ORDER BY created_at ASC;

-- name: CreateComment :one
INSERT INTO comments (post_id, user_id, parent_id, body)
VALUES ($1, $2, $3, $4)
RETURNING id, post_id, user_id, parent_id, body, created_at;

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = $1;

-- name: CountCommentsByPost :one
SELECT COUNT(*) FROM comments WHERE post_id = $1;

-- ============================================================
-- Tags — basic operations
-- ============================================================

-- name: ListTags :many
SELECT id, name FROM tags ORDER BY name;

-- name: CreateTag :one
INSERT INTO tags (name) VALUES ($1)
RETURNING id, name;

-- name: AddTagToPost :exec
INSERT INTO post_tags (post_id, tag_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveTagFromPost :exec
DELETE FROM post_tags WHERE post_id = $1 AND tag_id = $2;

-- name: ListTagsForPost :many
SELECT t.id, t.name
FROM tags t
JOIN post_tags pt ON pt.tag_id = t.id
WHERE pt.post_id = $1
ORDER BY t.name;

-- ============================================================
-- Complex: Multi-table INNER JOINs
-- ============================================================

-- name: GetPostWithAuthor :many
SELECT p.id, p.title, p.views, p.created_at, u.name AS author_name, u.email AS author_email
FROM posts p
JOIN users u ON u.id = p.user_id
WHERE p.published = TRUE
ORDER BY p.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetPostWithTags :many
SELECT p.id, p.title, t.name AS tag_name
FROM posts p
JOIN post_tags pt ON pt.post_id = p.id
JOIN tags t ON t.id = pt.tag_id
WHERE p.id = $1
ORDER BY t.name;

-- name: GetUserPostsWithTags :many
SELECT p.id AS post_id, p.title, p.views, p.published, t.name AS tag_name
FROM posts p
JOIN post_tags pt ON pt.post_id = p.id
JOIN tags t ON t.id = pt.tag_id
WHERE p.user_id = $1
ORDER BY p.created_at DESC;

-- ============================================================
-- Complex: GROUP BY with HAVING
-- ============================================================

-- name: GetActivePosters :many
SELECT u.id, u.name, u.email, COUNT(p.id) AS post_count
FROM users u
JOIN posts p ON p.user_id = u.id
WHERE p.published = TRUE
GROUP BY u.id, u.name, u.email
HAVING COUNT(p.id) >= $1
ORDER BY post_count DESC;

-- name: GetPopularTags :many
SELECT t.id, t.name, COUNT(pt.post_id) AS usage_count
FROM tags t
JOIN post_tags pt ON pt.tag_id = t.id
GROUP BY t.id, t.name
HAVING COUNT(pt.post_id) >= $1
ORDER BY usage_count DESC;

-- ============================================================
-- Complex: Subqueries in WHERE
-- ============================================================

-- name: GetUsersWithPublishedPosts :many
SELECT id, name, email, created_at
FROM users
WHERE id IN (SELECT DISTINCT user_id FROM posts WHERE published = TRUE)
ORDER BY name;

-- name: GetPostsAboveAvgViews :many
SELECT id, title, views, created_at
FROM posts
WHERE published = TRUE AND views > (SELECT AVG(views) FROM posts WHERE published = TRUE)
ORDER BY views DESC;

-- name: GetUsersWithRecentActivity :many
SELECT id, name, email, created_at
FROM users
WHERE id IN (
    SELECT DISTINCT user_id FROM posts WHERE created_at > NOW() - INTERVAL '30 days'
    UNION
    SELECT DISTINCT user_id FROM comments WHERE created_at > NOW() - INTERVAL '30 days'
)
ORDER BY name;

-- ============================================================
-- Complex: UPDATE/DELETE with subquery conditions
-- ============================================================

-- name: DeactivateUsersWithNoPosts :execresult
UPDATE users SET is_active = FALSE, updated_at = NOW()
WHERE id NOT IN (SELECT DISTINCT user_id FROM posts);

-- name: DeleteUnpublishedPostsOlderThan :execresult
DELETE FROM posts
WHERE published = FALSE AND created_at < NOW() - INTERVAL '90 days';

-- name: BoostPopularPosts :execresult
UPDATE posts SET views = views + 100
WHERE id IN (
    SELECT id FROM posts WHERE published = TRUE ORDER BY views DESC LIMIT $1
);

-- ============================================================
-- Complex: CTE (simple, single-table aggregation)
-- ============================================================

-- name: GetUserStats :many
WITH user_post_counts AS (
    SELECT user_id, COUNT(*) AS post_count, SUM(views) AS total_views
    FROM posts
    WHERE published = TRUE
    GROUP BY user_id
)
SELECT u.id, u.name, u.email, upc.post_count, upc.total_views
FROM users u
JOIN user_post_counts upc ON upc.user_id = u.id
ORDER BY upc.total_views DESC
LIMIT $1;

-- ============================================================
-- Complex: Window functions (single-table)
-- ============================================================

-- name: GetPostRanksByViews :many
SELECT id, title, views, RANK() OVER (ORDER BY views DESC) AS view_rank
FROM posts
WHERE published = TRUE
ORDER BY view_rank
LIMIT $1;

-- name: GetUserPostTimeline :many
SELECT id, title, views, created_at, ROW_NUMBER() OVER (ORDER BY created_at) AS post_number
FROM posts
WHERE user_id = $1 AND published = TRUE
ORDER BY created_at;

-- ============================================================
-- Complex: INSERT with complex expressions
-- ============================================================

-- name: CreatePostWithCategory :one
INSERT INTO posts (user_id, title, body, published)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, title, body, views, published, created_at, updated_at;

-- name: BulkPublishUserPosts :execresult
UPDATE posts SET published = TRUE, updated_at = NOW()
WHERE user_id = $1 AND published = FALSE;

-- ============================================================
-- Categories
-- ============================================================

-- name: ListCategories :many
SELECT id, name, description, parent_id, sort_order, created_at
FROM categories
ORDER BY sort_order, name;

-- name: CreateCategory :one
INSERT INTO categories (name, description, parent_id, sort_order)
VALUES ($1, $2, $3, $4)
RETURNING id, name, description, parent_id, sort_order, created_at;

-- name: AddPostToCategory :exec
INSERT INTO post_categories (post_id, category_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemovePostFromCategory :exec
DELETE FROM post_categories WHERE post_id = $1 AND category_id = $2;

-- name: GetPostsByCategory :many
SELECT p.id, p.title, p.views, p.created_at
FROM posts p
JOIN post_categories pc ON pc.post_id = p.id
WHERE pc.category_id = $1 AND p.published = TRUE
ORDER BY p.created_at DESC;

-- name: GetCategoriesForPost :many
SELECT c.id, c.name, c.description
FROM categories c
JOIN post_categories pc ON pc.category_id = c.id
WHERE pc.post_id = $1
ORDER BY c.name;
