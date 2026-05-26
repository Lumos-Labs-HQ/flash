---
title: Complete Workflows
description: End-to-end project workflows for Go, TypeScript, and Python
---

# Complete Workflows

These are full end-to-end workflows from project initialization to running queries. Each workflow is completely self-contained.

---

## Go Workflow

### 1. Project Setup

```bash
mkdir blog-app && cd blog-app
go mod init github.com/yourname/blog-app
go install github.com/Lumos-Labs-HQ/flash@latest
```

### 2. Initialize Flash ORM

```bash
flash init --postgresql
```

### 3. Configure Environment

```bash
cat > .env << 'EOF'
DATABASE_URL=postgres://postgres:postgres@localhost:5432/blog?sslmode=disable
EOF
```

### 4. Define Schema

```sql
-- db/schema/users.sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    bio TEXT,
    avatar_url VARCHAR(500),
    is_active BOOLEAN DEFAULT TRUE,
    role VARCHAR(20) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- db/schema/posts.sql
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    slug VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    excerpt TEXT,
    status VARCHAR(20) DEFAULT 'draft',
    cover_image VARCHAR(500),
    published_at TIMESTAMP,
    view_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- db/schema/comments.sql
CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    parent_id INTEGER REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_approved BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- db/schema/tags.sql
CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    color VARCHAR(7) DEFAULT '#000000'
);

CREATE TABLE post_tags (
    post_id INTEGER REFERENCES posts(id) ON DELETE CASCADE,
    tag_id INTEGER REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);

-- db/schema/likes.sql
CREATE TABLE likes (
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    post_id INTEGER REFERENCES posts(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, post_id)
);

-- Indexes
CREATE INDEX idx_posts_author ON posts(author_id);
CREATE INDEX idx_posts_status ON posts(status);
CREATE INDEX idx_posts_published ON posts(published_at);
CREATE INDEX idx_comments_post ON comments(post_id);
CREATE INDEX idx_comments_parent ON comments(parent_id);
CREATE INDEX idx_post_tags_tag ON post_tags(tag_id);
```

### 5. Create and Apply Migration

```bash
flash migrate "blog schema"
flash apply
```

### 6. Seed Test Data

```bash
flash seed users:20 posts:100 comments:500 tags:30 --truncate
```

### 7. Write Queries

```sql
-- db/queries/users.sql
-- name: GetUserByID :one
SELECT id, username, email, bio, avatar_url, is_active, role, created_at
FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT id, username, email, bio, avatar_url, is_active, role, created_at
FROM users WHERE username = $1;

-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, bio, avatar_url, role)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at;

-- name: UpdateUser :exec
UPDATE users SET
    username = COALESCE($2, username),
    email = COALESCE($3, email),
    bio = COALESCE($4, bio),
    avatar_url = COALESCE($5, avatar_url),
    role = COALESCE($6, role),
    updated_at = NOW()
WHERE id = $1;

-- name: ListUsers :many
SELECT id, username, email, is_active, role, created_at
FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- db/queries/posts.sql
-- name: CreatePost :one
INSERT INTO posts (author_id, slug, title, content, excerpt, status, cover_image, published_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, created_at;

-- name: GetPostBySlug :one
SELECT p.*, u.username as author_name, u.avatar_url as author_avatar
FROM posts p JOIN users u ON p.author_id = u.id
WHERE p.slug = $1;

-- name: ListPublishedPosts :many
SELECT p.id, p.slug, p.title, p.excerpt, p.cover_image, p.published_at, p.view_count,
       u.username as author_name, u.avatar_url as author_avatar
FROM posts p JOIN users u ON p.author_id = u.id
WHERE p.status = 'published' AND p.published_at <= NOW()
ORDER BY p.published_at DESC LIMIT $1 OFFSET $2;

-- name: GetPostsByAuthor :many
SELECT id, slug, title, status, published_at, view_count
FROM posts WHERE author_id = $1 ORDER BY created_at DESC;

-- name: IncrementViewCount :exec
UPDATE posts SET view_count = view_count + 1 WHERE id = $1;

-- name: UpdatePost :exec
UPDATE posts SET
    title = COALESCE($2, title),
    content = COALESCE($3, content),
    excerpt = COALESCE($4, excerpt),
    status = COALESCE($5, status),
    cover_image = COALESCE($6, cover_image),
    published_at = COALESCE($7, published_at),
    updated_at = NOW()
WHERE id = $1;

-- db/queries/comments.sql
-- name: CreateComment :one
INSERT INTO comments (post_id, author_id, parent_id, content)
VALUES ($1, $2, $3, $4) RETURNING id, created_at;

-- name: GetCommentsByPost :many
SELECT c.*, u.username as author_name, u.avatar_url as author_avatar
FROM comments c LEFT JOIN users u ON c.author_id = u.id
WHERE c.post_id = $1 AND c.parent_id IS NULL AND c.is_approved = TRUE
ORDER BY c.created_at DESC;

-- name: GetReplies :many
SELECT c.*, u.username as author_name
FROM comments c LEFT JOIN users u ON c.author_id = u.id
WHERE c.parent_id = $1 AND c.is_approved = TRUE
ORDER BY c.created_at;

-- name: ApproveComment :exec
UPDATE comments SET is_approved = TRUE WHERE id = $1;

-- db/queries/tags.sql
-- name: CreateTag :one
INSERT INTO tags (name, slug, description, color)
VALUES ($1, $2, $3, $4) RETURNING id;

-- name: GetTagBySlug :one
SELECT * FROM tags WHERE slug = $1;

-- name: ListTags :many
SELECT t.*, COUNT(pt.post_id) as post_count
FROM tags t LEFT JOIN post_tags pt ON t.id = pt.tag_id
GROUP BY t.id ORDER BY post_count DESC;

-- name: GetPostTags :many
SELECT t.* FROM tags t
JOIN post_tags pt ON t.id = pt.tag_id
WHERE pt.post_id = $1;

-- name: AddTagToPost :exec
INSERT INTO post_tags (post_id, tag_id) VALUES ($1, $2);

-- db/queries/likes.sql
-- name: ToggleLike :execrows
WITH deleted AS (
    DELETE FROM likes WHERE user_id = $1 AND post_id = $2 RETURNING *
)
INSERT INTO likes (user_id, post_id)
SELECT $1, $2 WHERE NOT EXISTS (SELECT 1 FROM deleted);

-- name: GetLikeCount :one
SELECT COUNT(*) FROM likes WHERE post_id = $1;

-- name: HasLiked :one
SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = $1 AND post_id = $2);
```

### 8. Generate Code

```bash
flash gen
```

### 9. Use in Application

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "os"

    "github.com/yourname/blog-app/flash_gen"
    _ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
    db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    if err != nil { log.Fatal(err) }
    defer db.Close()

    queries := flash_gen.New(db)
    ctx := context.Background()

    // Create a user
    userID, err := queries.CreateUser(ctx, flash_gen.CreateUserParams{
        Username:     "johndoe",
        Email:        "john@example.com",
        PasswordHash: "hashed_password",
        Bio:          sql.NullString{String: "Go developer", Valid: true},
        Role:         sql.NullString{String: "user", Valid: true},
    })
    if err != nil { log.Fatal(err) }
    log.Println("Created user:", userID)

    // Create a post
    postID, err := queries.CreatePost(ctx, flash_gen.CreatePostParams{
        AuthorID:  int32(userID),
        Slug:      "hello-world",
        Title:     "Hello World",
        Content:   "My first post!",
        Excerpt:   sql.NullString{String: "First post", Valid: true},
        Status:    sql.NullString{String: "published", Valid: true},
        PublishedAt: sql.NullTime{Time: time.Now(), Valid: true},
    })
    if err != nil { log.Fatal(err) }
    log.Println("Created post:", postID)

    // List posts
    posts, err := queries.ListPublishedPosts(ctx, flash_gen.ListPublishedPostsParams{
        Limit: 10, Offset: 0,
    })
    if err != nil { log.Fatal(err) }
    for _, p := range posts {
        log.Printf("Post: %s by %s", p.Title, p.AuthorName)
    }
}
```

---

## TypeScript Workflow

### 1. Project Setup

```bash
mkdir blog-api && cd blog-api
npm init -y
npm install typescript @types/node ts-node --save-dev
npm install -g flashorm
```

### 2. Initialize Flash ORM

```bash
flash init --postgresql
```

### 3. Configure

```toml
# flash.toml
version = "2"
schema_dir = "db/schema"
queries = "db/queries"

[database]
provider = "postgresql"
url_env = "DATABASE_URL"

[gen.js]
enabled = true
out = "flash_gen"
```

### 4. Use Same Schema and Queries as Go example above

### 5. Generate Code

```bash
flash gen
```

### 6. Use in Application

```typescript
import { Pool } from 'pg';
import { New } from './flash_gen/database';

const pool = new Pool({
    connectionString: process.env.DATABASE_URL,
});

const db = New(pool);

async function main() {
    // Create user
    const userId = await db.createUser({
        username: 'johndoe',
        email: 'john@example.com',
        password_hash: 'hashed',
        bio: 'TS developer',
        role: 'user',
    });
    console.log('Created user:', userId);

    // Create post
    const postId = await db.createPost({
        author_id: userId,
        slug: 'hello-world',
        title: 'Hello World',
        content: 'My first post!',
        status: 'published',
        published_at: new Date(),
    });
    console.log('Created post:', postId);

    // List published posts
    const posts = await db.listPublishedPosts({ limit: 10, offset: 0 });
    for (const p of posts) {
        console.log(`${p.title} by ${p.author_name}`);
    }

    await pool.end();
}

main().catch(console.error);
```

---

## Python Workflow

### 1. Project Setup

```bash
mkdir blog-api && cd blog-api
python -m venv venv
source venv/bin/activate
pip install flashorm asyncpg
```

### 2. Initialize Flash ORM

```bash
flash init --postgresql
```

### 3. Configure

```toml
# flash.toml
version = "2"
schema_dir = "db/schema"
queries = "db/queries"

[database]
provider = "postgresql"
url_env = "DATABASE_URL"

[gen.python]
enabled = true
out = "flash_gen"
async = true
```

### 4. Use Same Schema and Queries as Go example above

### 5. Generate Code

```bash
flash gen
```

### 6. Use in Application

```python
import asyncio
import asyncpg
from flash_gen import new

async def main():
    pool = await asyncpg.create_pool('postgresql://...')
    db = new(pool)

    # Create user
    user_id = await db.create_user(
        username='johndoe',
        email='john@example.com',
        password_hash='hashed',
        bio='Python developer',
        role='user',
    )
    print(f'Created user: {user_id}')

    # Create post
    post_id = await db.create_post(
        author_id=user_id,
        slug='hello-world',
        title='Hello World',
        content='My first post!',
        status='published',
        published_at=datetime.now(),
    )
    print(f'Created post: {post_id}')

    # List posts
    posts = await db.list_published_posts(limit=10, offset=0)
    for p in posts:
        print(f"{p['title']} by {p['author_name']}")

    await pool.close()

asyncio.run(main())
```
