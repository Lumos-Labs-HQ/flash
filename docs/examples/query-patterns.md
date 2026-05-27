---
title: Query Patterns
description: SQL query patterns for Flash ORM code generation
---

# Query Patterns

Production-ready SQL query patterns organized by use case.

---

## CRUD Operations

### Create

```sql
-- Basic insert returning ID
-- name: CreateUser :one
INSERT INTO users (username, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id, created_at;

-- Insert with all fields
-- name: CreatePost :one
INSERT INTO posts (author_id, slug, title, content, excerpt, status, published_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, created_at;

-- Bulk insert (batch)
-- name: CreateTags :exec
INSERT INTO tags (name, slug, color)
SELECT * FROM UNNEST($1::text[], $2::text[], $3::text[]);
```

### Read

```sql
-- Get single record by ID
-- name: GetUserByID :one
SELECT id, username, email, bio, avatar_url, is_active, role, created_at
FROM users WHERE id = $1;

-- Get single record with related data
-- name: GetPostWithAuthor :one
SELECT p.*, u.username as author_name, u.avatar_url as author_avatar
FROM posts p
JOIN users u ON p.author_id = u.id
WHERE p.id = $1;

-- List with pagination
-- name: ListUsers :many
SELECT id, username, email, is_active, role, created_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- List with filtering
-- name: SearchPosts :many
SELECT id, slug, title, excerpt, status, published_at, view_count
FROM posts
WHERE status = $1
  AND ($2::text IS NULL OR title ILIKE '%' || $2 || '%')
  AND ($3::timestamp IS NULL OR published_at >= $3)
ORDER BY published_at DESC
LIMIT $4 OFFSET $5;
```

### Update

```sql
-- Partial update (only provided fields)
-- name: UpdateUser :exec
UPDATE users SET
    username = COALESCE($2, username),
    email = COALESCE($3, email),
    bio = COALESCE($4, bio),
    avatar_url = COALESCE($5, avatar_url),
    role = COALESCE($6, role),
    updated_at = NOW()
WHERE id = $1;

-- Update counter
-- name: IncrementPostViews :exec
UPDATE posts SET view_count = view_count + 1 WHERE id = $1;

-- Update status
-- name: PublishPost :exec
UPDATE posts SET status = 'published', published_at = NOW() WHERE id = $1;
```

### Delete

```sql
-- Hard delete
-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- Soft delete
-- name: SoftDeletePost :exec
UPDATE posts SET deleted_at = NOW() WHERE id = $1;
```

---

## Relationships

### One-to-Many

```sql
-- Get posts by author
-- name: GetPostsByAuthor :many
SELECT id, slug, title, status, view_count, created_at
FROM posts WHERE author_id = $1 ORDER BY created_at DESC;

-- Count posts by author
-- name: CountPostsByAuthor :one
SELECT COUNT(*) FROM posts WHERE author_id = $1;
```

### Many-to-Many

```sql
-- Get tags for a post
-- name: GetPostTags :many
SELECT t.id, t.name, t.slug, t.color
FROM tags t
JOIN post_tags pt ON t.id = pt.tag_id
WHERE pt.post_id = $1;

-- Get posts by tag
-- name: GetPostsByTag :many
SELECT p.id, p.slug, p.title, p.excerpt, p.published_at
FROM posts p
JOIN post_tags pt ON p.id = pt.post_id
WHERE pt.tag_id = $1
ORDER BY p.published_at DESC;
```

### Self-Referencing (Tree)

```sql
-- Get top-level comments
-- name: GetPostComments :many
SELECT c.*, u.username as author_name
FROM comments c
LEFT JOIN users u ON c.author_id = u.id
WHERE c.post_id = $1 AND c.parent_id IS NULL
ORDER BY c.created_at DESC;

-- Get replies to a comment
-- name: GetCommentReplies :many
SELECT c.*, u.username as author_name
FROM comments c
LEFT JOIN users u ON c.author_id = u.id
WHERE c.parent_id = $1
ORDER BY c.created_at;

-- Get full comment tree (PostgreSQL recursive CTE)
-- name: GetCommentTree :many
WITH RECURSIVE tree AS (
    SELECT c.*, 0 as depth, ARRAY[c.id] as path
    FROM comments c WHERE c.id = $1
    UNION ALL
    SELECT c.*, t.depth + 1, t.path || c.id
    FROM comments c
    JOIN tree t ON c.parent_id = t.id
)
SELECT * FROM tree ORDER BY path;
```

---

## Aggregation

### Basic Aggregates

```sql
-- Dashboard stats
-- name: GetDashboardStats :one
SELECT
    (SELECT COUNT(*) FROM users) as total_users,
    (SELECT COUNT(*) FROM posts WHERE status = 'published') as total_posts,
    (SELECT COUNT(*) FROM comments) as total_comments,
    (SELECT COUNT(*) FROM users WHERE created_at >= NOW() - INTERVAL '24 hours') as new_users_today;

-- Daily post counts
-- name: GetDailyPostCounts :many
SELECT DATE(created_at) as date, COUNT(*) as count
FROM posts
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY DATE(created_at)
ORDER BY date;
```

### Group By

```sql
-- Posts by status
-- name: GetPostStatusCounts :many
SELECT status, COUNT(*) as count
FROM posts
GROUP BY status;

-- Top authors by post count
-- name: GetTopAuthors :many
SELECT u.id, u.username, COUNT(p.id) as post_count
FROM users u
LEFT JOIN posts p ON u.id = p.author_id
GROUP BY u.id, u.username
ORDER BY post_count DESC
LIMIT $1;
```

---

## Search and Filter

### Full-Text Search

```sql
-- Search posts by title/content
-- name: SearchPosts :many
SELECT id, slug, title, excerpt, published_at
FROM posts
WHERE status = 'published'
  AND (
      title ILIKE '%' || $1 || '%'
      OR content ILIKE '%' || $1 || '%'
      OR excerpt ILIKE '%' || $1 || '%'
  )
ORDER BY published_at DESC
LIMIT $2 OFFSET $3;
```

### Date Range Filter

```sql
-- Get posts from date range
-- name: GetPostsByDateRange :many
SELECT id, slug, title, published_at, view_count
FROM posts
WHERE status = 'published'
  AND published_at BETWEEN $1 AND $2
ORDER BY published_at DESC;
```

### Multi-Filter

```sql
-- Flexible search with multiple optional filters
-- name: ListProducts :many
SELECT id, sku, name, slug, price, status, quantity
FROM products
WHERE status = COALESCE($1, status)
  AND ($2::decimal IS NULL OR price >= $2)
  AND ($3::decimal IS NULL OR price <= $3)
  AND ($4::text IS NULL OR name ILIKE '%' || $4 || '%')
  AND ($5::boolean IS NULL OR featured = $5)
ORDER BY
    CASE WHEN $6 = 'price_asc' THEN price END ASC,
    CASE WHEN $6 = 'price_desc' THEN price END DESC,
    CASE WHEN $6 = 'name' THEN name END ASC,
    created_at DESC
LIMIT $7 OFFSET $8;
```

---

## Transactions

### Atomic Operations

```sql
-- Transfer ownership
-- name: TransferPostOwnership :exec
UPDATE posts SET author_id = $2, updated_at = NOW() WHERE id = $1;

-- Update with related records
-- name: MergeUsers :exec
UPDATE posts SET author_id = $2 WHERE author_id = $1;
UPDATE comments SET author_id = $2 WHERE author_id = $1;
UPDATE likes SET user_id = $2 WHERE user_id = $1;
DELETE FROM users WHERE id = $1;
```

---

## JSON Operations (PostgreSQL)

```sql
-- Get user with settings
-- name: GetUserSettings :one
SELECT id, username, settings
FROM users WHERE id = $1;

-- Update nested JSON field
-- name: UpdateUserSetting :exec
UPDATE users
SET settings = jsonb_set(
    COALESCE(settings, '{}'),
    ARRAY[$2],
    to_jsonb($3)
)
WHERE id = $1;
```

---

## Window Functions

```sql
-- Rank posts by views
-- name: GetTopPostsWithRank :many
SELECT
    id, title, view_count,
    RANK() OVER (ORDER BY view_count DESC) as rank
FROM posts
WHERE status = 'published'
LIMIT $1;

-- Running total
-- name: GetDailyCumulativeUsers :many
SELECT
    DATE(created_at) as date,
    COUNT(*) as new_users,
    SUM(COUNT(*)) OVER (ORDER BY DATE(created_at)) as cumulative
FROM users
GROUP BY DATE(created_at)
ORDER BY date;
```
