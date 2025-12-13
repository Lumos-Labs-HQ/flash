---
title: Query API Reference
description: Complete reference for FlashORM query definition syntax
---

# Query API Reference

This page provides a complete reference for defining SQL queries in FlashORM.

## Overview

FlashORM uses named SQL queries stored in `.sql` files in the `db/queries/` directory. These queries are parsed and used to generate type-safe code.

## Query Syntax

### Named Queries

Queries are defined with a special comment syntax:

```sql
-- name: GetUser :one
SELECT * FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (name, email)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateUser :exec
UPDATE users
SET name = $2, email = $3
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
```

### Query Types

- `:one` - Returns a single row
- `:many` - Returns multiple rows
- `:exec` - Executes without returning data

## Parameters

Parameters are referenced by position using `$1`, `$2`, etc.:

```sql
-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: GetPostsByUser :many
SELECT * FROM posts
WHERE user_id = $1 AND published = $2
ORDER BY created_at DESC;
```

## Generated Code

### Go

For a query like:

```sql
-- name: GetUser :one
SELECT id, name, email, created_at FROM users
WHERE id = $1;
```

Generated Go code:

```go
type User struct {
    ID        int32     `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

func (q *Queries) GetUser(ctx context.Context, id int32) (User, error) {
    // Implementation with prepared statement
}
```

### TypeScript

```typescript
export interface User {
  id: number;
  name: string;
  email: string;
  createdAt: Date;
}

export function getUser(id: number): Promise<User>;
```

### Python

```python
@dataclass
class User:
    id: int
    name: str
    email: str
    created_at: datetime

async def get_user(id: int) -> User:
    # Implementation
```

## Advanced Queries

### Joins

```sql
-- name: GetPostWithAuthor :one
SELECT
    p.id,
    p.title,
    p.content,
    p.created_at as post_created_at,
    u.id as author_id,
    u.name as author_name,
    u.email as author_email
FROM posts p
JOIN users u ON p.user_id = u.id
WHERE p.id = $1;
```

### Aggregations

```sql
-- name: GetUserStats :one
SELECT
    u.id,
    u.name,
    COUNT(p.id) as post_count,
    COUNT(c.id) as comment_count,
    MAX(p.created_at) as last_post_date
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
LEFT JOIN comments c ON u.id = c.user_id
WHERE u.id = $1
GROUP BY u.id, u.name;
```

### Complex Queries

```sql
-- name: GetPostsWithMetadata :many
WITH post_stats AS (
    SELECT
        p.id,
        COUNT(c.id) as comment_count,
        AVG(c.rating) as avg_rating
    FROM posts p
    LEFT JOIN comments c ON p.id = c.post_id
    GROUP BY p.id
)
SELECT
    p.id,
    p.title,
    p.content,
    p.created_at,
    u.name as author_name,
    ps.comment_count,
    ps.avg_rating
FROM posts p
JOIN users u ON p.user_id = u.id
JOIN post_stats ps ON p.id = ps.id
WHERE p.published = true
ORDER BY p.created_at DESC;
```

## Special Types

### Arrays (PostgreSQL)

```sql
-- name: GetUsersByIds :many
SELECT * FROM users
WHERE id = ANY($1::int[]);
```

Generated as `[]int32` in Go.

### JSON

```sql
-- name: GetUserWithMetadata :one
SELECT
    id,
    name,
    metadata::json as metadata
FROM users
WHERE id = $1;
```

Generated as `interface{}` in Go.

### Enums

```sql
-- name: GetPostsByStatus :many
SELECT * FROM posts
WHERE status = $1;
```

Where status is an enum type.

## Batch Operations

### Multiple Inserts

```sql
-- name: CreateUsers :copyfrom
INSERT INTO users (name, email) VALUES ($1, $2);
```

Generates batch insert methods.

### Bulk Updates

```sql
-- name: UpdateUserEmails :exec
UPDATE users
SET email = data.email
FROM (VALUES
    (1, 'new1@example.com'),
    (2, 'new2@example.com')
) AS data(id, email)
WHERE users.id = data.id;
```

## Error Handling

Queries that might fail should be handled appropriately:

```sql
-- name: GetUserOrNull :one
SELECT * FROM users
WHERE id = $1;
```

In Go, this returns `(User, error)` where error is `sql.ErrNoRows` if not found.

## Performance Considerations

### Prepared Statements

FlashORM automatically uses prepared statements for all queries, providing:
- SQL injection protection
- Query plan caching
- 2-5x performance improvement for repeated queries

### Indexing

Ensure your queries are supported by appropriate indexes:

```sql
-- Good: Uses index on email
CREATE INDEX idx_users_email ON users(email);

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;
```

### Query Optimization

- Use `EXPLAIN ANALYZE` to check query performance
- Avoid `SELECT *` in production for large tables
- Use appropriate `LIMIT` clauses
- Consider pagination for large result sets

## Pagination

### Offset-based

```sql
-- name: GetPostsPaginated :many
SELECT * FROM posts
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
```

### Cursor-based

```sql
-- name: GetPostsAfterCursor :many
SELECT * FROM posts
WHERE created_at > $1
ORDER BY created_at ASC
LIMIT $2;
```

## Transactions

Queries can be executed within transactions:

```go
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.Rollback()

// Use tx instead of db
user, err := queries.WithTx(tx).CreateUser(ctx, name, email)
if err != nil {
    return err
}

return tx.Commit()
```

## File Organization

### By Feature

```
db/queries/
├── users.sql
├── posts.sql
├── comments.sql
└── admin.sql
```

### By Type

```
db/queries/
├── queries.sql      # All :one and :many queries
├── mutations.sql    # All :exec queries
└── views.sql        # Complex read queries
```

## Naming Conventions

- Use PascalCase for query names: `GetUser`, `CreatePost`, `UpdateUser`
- Use descriptive names that indicate the operation and entity
- Group related queries in the same file
- Use consistent parameter ordering

## Examples

### User Management

```sql
-- name: GetUser :one
SELECT id, name, email, created_at, updated_at FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, name, email, created_at, updated_at FROM users
WHERE email = $1;

-- name: ListUsers :many
SELECT id, name, email, created_at FROM users
ORDER BY created_at DESC
LIMIT $1;

-- name: CreateUser :one
INSERT INTO users (name, email)
VALUES ($1, $2)
RETURNING id, name, email, created_at, updated_at;

-- name: UpdateUser :exec
UPDATE users
SET name = $2, email = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
```

### Blog Posts

```sql
-- name: GetPost :one
SELECT
    p.id,
    p.title,
    p.content,
    p.created_at,
    p.updated_at,
    u.name as author_name,
    c.name as category_name
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN categories c ON p.category_id = c.id
WHERE p.id = $1;

-- name: ListPosts :many
SELECT
    p.id,
    p.title,
    p.created_at,
    u.name as author_name
FROM posts p
JOIN users u ON p.user_id = u.id
WHERE p.published = true
ORDER BY p.created_at DESC
LIMIT $1;

-- name: CreatePost :one
INSERT INTO posts (title, content, user_id, category_id, published)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPostsByAuthor :many
SELECT * FROM posts
WHERE user_id = $1
ORDER BY created_at DESC;
```

### Analytics

```sql
-- name: GetUserActivity :one
SELECT
    u.id,
    u.name,
    COUNT(DISTINCT p.id) as total_posts,
    COUNT(DISTINCT c.id) as total_comments,
    MAX(p.created_at) as last_post_date,
    MAX(c.created_at) as last_comment_date
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
LEFT JOIN comments c ON u.id = c.user_id
WHERE u.id = $1
GROUP BY u.id, u.name;

-- name: GetPopularPosts :many
SELECT
    p.id,
    p.title,
    COUNT(c.id) as comment_count,
    p.created_at
FROM posts p
LEFT JOIN comments c ON p.id = c.post_id
WHERE p.published = true
GROUP BY p.id, p.title, p.created_at
HAVING COUNT(c.id) > 0
ORDER BY comment_count DESC
LIMIT 10;
```
