---
title: Go Guide
description: Complete guide to using Flash ORM with Go
---

# Flash ORM - Go Usage Guide

A comprehensive guide to using Flash ORM with Go projects, featuring type-safe code generation and high performance.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Schema Definition](#schema-definition)
- [Migrations](#migrations)
- [Code Generation](#code-generation)
- [Working with Generated Code](#working-with-generated-code)
- [Advanced Usage](#advanced-usage)
- [Best Practices](#best-practices)

## Installation

### Install Flash ORM CLI

```bash
go install github.com/Lumos-Labs-HQ/flash@latest
```

Verify installation:
```bash
flash --version
```

## Quick Start

### 1. Create a New Go Project

```bash
mkdir myproject && cd myproject
go mod init myproject
```

### 2. Initialize Flash ORM

```bash
# For PostgreSQL
flash init --postgresql

# For MySQL
flash init --mysql

# For SQLite
flash init --sqlite
```

### 3. Set Database URL

Create a `.env` file:
```env
# PostgreSQL
DATABASE_URL=postgres://user:password@localhost:5432/mydb

# MySQL
DATABASE_URL=user:password@tcp(localhost:3306)/mydb

# SQLite
DATABASE_URL=sqlite://./data.db
```

### 4. Define Your Schema

Edit `db/schema/schema.sql`:
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 5. Create and Apply Migration

```bash
flash migrate "initial schema"
flash apply
```

### 6. Generate Go Code

```bash
flash gen
```

## Configuration

### flash.config.json

```json
{
  "version": "2",
  "schema_dir": "db/schema",
  "queries": "db/queries",
  "migrations_path": "db/migrations",
  "export_path": "db/export",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "go": {
      "enabled": true
    }
  }
}
```

### Environment Variables

```env
DATABASE_URL=postgres://user:password@localhost:5432/mydb
```

## Schema Definition

### Basic Schema

```sql
-- db/schema/users.sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- db/schema/posts.sql
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT FALSE,
    tags TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Advanced Schema Features

```sql
-- Enums
CREATE TYPE user_role AS ENUM ('admin', 'user', 'moderator');

-- Tables with enums
CREATE TABLE user_profiles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    role user_role DEFAULT 'user',
    bio TEXT,
    avatar_url VARCHAR(500)
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE UNIQUE INDEX idx_user_profiles_user_id ON user_profiles(user_id);

-- Constraints
ALTER TABLE users ADD CONSTRAINT chk_name_length CHECK (length(name) >= 2);
ALTER TABLE posts ADD CONSTRAINT chk_title_length CHECK (length(title) >= 1);
```

## Migrations

### Creating Migrations

```bash
# Interactive migration creation
flash migrate

# Named migration
flash migrate "add user profiles table"

# Auto-generate migration from schema changes
flash migrate "update schema" --auto
```

### Migration Files

Migrations are stored in `db/migrations/` with timestamp prefixes:

```
db/migrations/
├── 20240101120000_initial_schema.up.sql
├── 20240101120000_initial_schema.down.sql
├── 20240102130000_add_user_profiles.up.sql
└── 20240102130000_add_user_profiles.down.sql
```

### Applying Migrations

```bash
# Apply all pending migrations
flash apply

# Apply specific number of migrations
flash apply --count 1

# Dry run (show what would be executed)
flash apply --dry-run
```

### Migration Status

```bash
# Check migration status
flash status

# Output:
# Migration Status:
# ✅ 20240101120000 - initial schema
# ✅ 20240102130000 - add user profiles
# ⏳ 20240103140000 - add comments table (pending)
```

## Code Generation

### Generated Files

After running `flash gen`, you'll get:

```
flash_gen/
├── db.go          # Database connection and query interface
├── models.go      # Generated models and types
├── users.go       # User-related queries
├── posts.go       # Post-related queries
└── ...
```

### Generated Code Structure

```go
// flash_gen/db.go
package flash_gen

import (
    "context"
    "database/sql"
)

type DBTX interface {
    ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
    PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
    QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type Queries struct {
    db DBTX
}

func New(db DBTX) *Queries {
    return &Queries{db: db}
}
```

## Working with Generated Code

### Database Connection

```go
// db/database.go
package db

import (
    "database/sql"
    "log"
    "os"

    "yourproject/flash_gen"
    _ "github.com/lib/pq" // PostgreSQL driver
)

var DB *sql.DB
var Queries *flash_gen.Queries

func ConnectDatabase() {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        log.Fatal("DATABASE_URL environment variable is not set")
    }

    var err error
    DB, err = sql.Open("postgres", dbURL)
    if err != nil {
        log.Fatal("Failed to open database:", err)
    }

    if err = DB.Ping(); err != nil {
        log.Fatal("Failed to ping database:", err)
    }

    Queries = flash_gen.New(DB)
    log.Println("Database connected successfully")
}

func CloseDatabase() {
    if DB != nil {
        DB.Close()
        log.Println("Database connection closed")
    }
}
```

### Using Generated Queries

```go
// main.go
package main

import (
    "context"
    "log"
    "time"

    "yourproject/db"
)

func main() {
    db.ConnectDatabase()
    defer db.CloseDatabase()

    ctx := context.Background()

    // Create a user
    userID, err := db.Queries.CreateUser(ctx, flash_gen.CreateUserParams{
        Name:  "John Doe",
        Email: "john@example.com",
    })
    if err != nil {
        log.Fatal("Failed to create user:", err)
    }
    log.Printf("Created user with ID: %d", userID)

    // Get user by ID
    user, err := db.Queries.GetUserByID(ctx, userID)
    if err != nil {
        log.Fatal("Failed to get user:", err)
    }
    log.Printf("User: %+v", user)

    // Create a post
    postID, err := db.Queries.CreatePost(ctx, flash_gen.CreatePostParams{
        UserID:  userID,
        Title:   "My First Post",
        Content: "This is the content of my first post.",
    })
    if err != nil {
        log.Fatal("Failed to create post:", err)
    }

    // Get posts by user
    posts, err := db.Queries.GetPostsByUserID(ctx, userID)
    if err != nil {
        log.Fatal("Failed to get posts:", err)
    }

    for _, post := range posts {
        log.Printf("Post: %s - %s", post.Title, post.Content)
    }
}
```

### Query Files

Create query files in `db/queries/`:

```sql
-- db/queries/users.sql
-- name: GetUserByID :one
SELECT id, name, email, is_active, created_at, updated_at
FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, name, email, is_active, created_at, updated_at
FROM users WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id;

-- name: UpdateUser :exec
UPDATE users SET name = $2, email = $3, updated_at = NOW()
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, name, email, is_active, created_at, updated_at
FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;
```

```sql
-- db/queries/posts.sql
-- name: CreatePost :one
INSERT INTO posts (user_id, title, content) VALUES ($1, $2, $3) RETURNING id;

-- name: GetPostByID :one
SELECT p.id, p.user_id, p.title, p.content, p.published, p.created_at,
       u.name as author_name, u.email as author_email
FROM posts p
JOIN users u ON p.user_id = u.id
WHERE p.id = $1;

-- name: GetPostsByUserID :many
SELECT id, user_id, title, content, published, created_at
FROM posts WHERE user_id = $1 ORDER BY created_at DESC;

-- name: UpdatePost :exec
UPDATE posts SET title = $2, content = $3, published = $4
WHERE id = $1;

-- name: DeletePost :exec
DELETE FROM posts WHERE id = $1;
```

## Advanced Usage

### Transactions

```go
func createUserWithProfile(ctx context.Context, name, email, bio string) error {
    tx, err := db.DB.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    queries := db.Queries.WithTx(tx)

    // Create user
    userID, err := queries.CreateUser(ctx, flash_gen.CreateUserParams{
        Name:  name,
        Email: email,
    })
    if err != nil {
        return err
    }

    // Create profile
    _, err = queries.CreateUserProfile(ctx, flash_gen.CreateUserProfileParams{
        UserID: userID,
        Bio:    bio,
    })
    if err != nil {
        return err
    }

    return tx.Commit()
}
```

### Prepared Statements

Flash ORM automatically caches prepared statements for performance:

```go
// The generated code includes prepared statement caching
type Queries struct {
    db    DBTX
    stmts map[string]*sql.Stmt // Statement cache
}
```

### Context Support

All generated methods accept context for cancellation and timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

user, err := db.Queries.GetUserByID(ctx, userID)
```

## Best Practices

### Project Structure

```
myproject/
├── db/
│   ├── database.go      # Database connection
│   ├── schema/          # Schema files
│   │   ├── users.sql
│   │   └── posts.sql
│   ├── queries/         # Query files
│   │   ├── users.sql
│   │   └── posts.sql
│   └── migrations/      # Migration files
├── flash_gen/           # Generated code (don't edit)
├── handlers/            # HTTP handlers
├── models/              # Business logic models
├── main.go
├── go.mod
└── flash.config.json
```

### Error Handling

```go
func getUser(ctx context.Context, userID int64) (*flash_gen.User, error) {
    user, err := db.Queries.GetUserByID(ctx, userID)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fmt.Errorf("user not found: %w", err)
        }
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return &user, nil
}
```

### Connection Pooling

```go
// db/database.go
func initDB() *sql.DB {
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }

    // Configure connection pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(25)
    db.SetConnMaxLifetime(5 * time.Minute)

    return db
}
```

### Migration Strategy

- Keep migrations small and focused
- Test migrations on staging before production
- Use descriptive migration names
- Never modify existing migrations
- Create new migrations for schema changes

### Performance Tips

- Use prepared statements (automatic in generated code)
- Implement proper indexing
- Use connection pooling
- Batch operations when possible
- Use context for timeouts

## Troubleshooting

### Common Issues

**Migration fails with "table already exists"**
- Check if migration was already applied: `flash status`
- Reset if needed: `flash reset`

**Generated code has compilation errors**
- Ensure schema files are valid SQL
- Check query syntax in `.sql` files
- Regenerate code: `flash gen`

**Database connection fails**
- Verify `DATABASE_URL` environment variable
- Check database server is running
- Ensure user has proper permissions

### Getting Help

- [GitHub Issues](https://github.com/Lumos-Labs-HQ/flash/issues)
- [Documentation](/advanced/how-it-works)
- [Community Discussions](https://github.com/Lumos-Labs-HQ/flash/discussions)
