# FlashORM - Go Usage Guide

A comprehensive guide to using FlashORM with Go projects.

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

---

## Installation

### Install FlashORM CLI

```bash
go install github.com/Lumos-Labs-HQ/flash@latest
```

Verify installation:
```bash
flash --version
```

---

## Quick Start

### 1. Create a New Go Project

```bash
mkdir myproject && cd myproject
go mod init myproject
```

### 2. Initialize FlashORM

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

---

## Configuration

### flash.config.json

```json
{
  "version": "2",
  "schema_dir": "db/schema",
  "queries": "db/queries/",
  "migrations_path": "db/migrations",
  "export_path": "db/export",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "go": {
      "enabled": true,
      "package": "models",
      "output": "flash_gen"
    }
  }
}
```

### Using Schema Directory

For larger projects, organize schemas in separate files:

```
db/schema/
├── users.sql
├── posts.sql
├── comments.sql
└── tags.sql
```

Configure in `flash.config.json`:
```json
{
  "schema_dir": "db/schema"
}
```

---

## Schema Definition

### Table Definition

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Indexes

```sql
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role_active ON users(role, is_active);
CREATE UNIQUE INDEX idx_users_email_unique ON users(email);
```

### Foreign Keys

```sql
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT
);
```

### PostgreSQL Enums

```sql
CREATE TYPE user_role AS ENUM ('admin', 'moderator', 'user', 'guest');

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    role user_role DEFAULT 'user'
);
```

---

## Migrations

### Create Migration

```bash
flash migrate "add phone to users"
```

This creates a file like `db/migrations/20251204123456_add_phone_to_users.sql`:

```sql
-- Migration: add_phone_to_users
-- Created: 2025-12-04T12:34:56Z

-- +migrate Up
ALTER TABLE users ADD COLUMN phone VARCHAR(20);

-- +migrate Down
ALTER TABLE users DROP COLUMN phone;
```

### Apply Migrations

```bash
# Apply all pending migrations
flash apply

# Apply with force (skip confirmation)
flash apply --force
```

### Check Status

```bash
flash status
```

Output:
```
Database: Connected ✅
Migrations: 3 total, 2 applied, 1 pending

ID                  NAME                  STATUS    APPLIED AT
──────────────────  ────────────────────  ────────  ─────────────────────
20251204120000      initial_schema        Applied   2025-12-04 12:00:00
20251204123456      add_phone_to_users    Applied   2025-12-04 12:34:56
20251204140000      add_posts_table       Pending   -
```

### Rollback Migration

```bash
flash down
```

---

## Code Generation

### Generate Models

```bash
flash gen
```

This generates code in `flash_gen/`:

```
flash_gen/
├── models.go      # Struct definitions
├── database.go    # Database connection
└── users.go       # User queries
```

### Generated Structs

```go
// flash_gen/models.go
package flash_gen

import "time"

type User struct {
    ID        int32     `json:"id" db:"id"`
    Name      string    `json:"name" db:"name"`
    Email     string    `json:"email" db:"email"`
    Role      string    `json:"role" db:"role"`
    IsActive  bool      `json:"is_active" db:"is_active"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Post struct {
    ID        int32     `json:"id" db:"id"`
    UserID    int32     `json:"user_id" db:"user_id"`
    Title     string    `json:"title" db:"title"`
    Content   *string   `json:"content" db:"content"` // Nullable
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

---

## Working with Generated Code

### Database Connection

```go
package main

import (
    "database/sql"
    "log"
    "os"

    "myproject/flash_gen"
    _ "github.com/lib/pq" // PostgreSQL driver
)

func main() {
    // Connect to database
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Use generated code
    queries := flash_gen.New(db)
    
    // ... your application logic
}
```

### CRUD Operations

Write your queries in `db/queries/users.sql`:

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (name, email, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateUser :one
UPDATE users 
SET name = $2, email = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
```

Use in your code:

```go
package main

import (
    "context"
    "log"
    
    "myproject/flash_gen"
)

func main() {
    ctx := context.Background()
    
    // Create user
    user, err := queries.CreateUser(ctx, flash_gen.CreateUserParams{
        Name:  "John Doe",
        Email: "john@example.com",
        Role:  "user",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Get user
    user, err = queries.GetUser(ctx, user.ID)
    
    // List users
    users, err := queries.ListUsers(ctx)
    
    // Update user
    user, err = queries.UpdateUser(ctx, flash_gen.UpdateUserParams{
        ID:    user.ID,
        Name:  "Jane Doe",
        Email: "jane@example.com",
    })
    
    // Delete user
    err = queries.DeleteUser(ctx, user.ID)
}
```

---

## Advanced Usage

### Pull Schema from Existing Database

```bash
# Extract schema from live database
flash pull
```

This updates your schema files to match the database.

### Export Database

```bash
# Export to JSON
flash export --json

# Export to CSV
flash export --csv

# Export to SQLite file
flash export --sqlite
```

### Raw SQL Execution

```bash
# Execute SQL command
flash raw "SELECT COUNT(*) FROM users;"

# Execute SQL file
flash raw ./scripts/seed.sql
```

### Reset Database

```bash
# Warning: Drops all tables!
flash reset --force
```

### Studio (Visual Editor)

```bash
flash studio
```

Opens a web interface for:
- Visual schema editing
- Data browsing and editing
- Auto-migration creation

---

## Best Practices

### 1. Organize Schemas by Domain

```
db/schema/
├── auth/
│   ├── users.sql
│   └── sessions.sql
├── content/
│   ├── posts.sql
│   └── comments.sql
└── core/
    └── settings.sql
```

### 2. Use Meaningful Migration Names

```bash
# Good
flash migrate "add email verification to users"
flash migrate "create posts comments relationship"

# Avoid
flash migrate "update"
flash migrate "fix"
```

### 3. Always Include Down Migrations

```sql
-- +migrate Up
ALTER TABLE users ADD COLUMN phone VARCHAR(20);

-- +migrate Down
ALTER TABLE users DROP COLUMN phone;
```

### 4. Test Migrations Locally First

```bash
# Create test database
flash reset --force
flash apply
flash status
```

### 5. Use Transactions

FlashORM automatically wraps migrations in transactions. For complex queries, use transactions in your code:

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

qtx := queries.WithTx(tx)

// Perform operations...

return tx.Commit()
```

---

## Troubleshooting

### Connection Issues

```bash
# Check database URL
echo $DATABASE_URL

# Test connection
flash status
```

### Migration Conflicts

```bash
# Check current state
flash status

# Reset if needed (development only)
flash reset --force
flash apply
```

### Type Mapping Issues

| PostgreSQL | MySQL | SQLite | Go Type |
|------------|-------|--------|---------|
| SERIAL | INT AUTO_INCREMENT | INTEGER | int32 |
| BIGSERIAL | BIGINT AUTO_INCREMENT | INTEGER | int64 |
| VARCHAR | VARCHAR | TEXT | string |
| TEXT | TEXT | TEXT | string |
| BOOLEAN | TINYINT(1) | INTEGER | bool |
| TIMESTAMP | DATETIME | TEXT | time.Time |
| JSONB | JSON | TEXT | json.RawMessage |

---

## Resources

- [FlashORM GitHub](https://github.com/Lumos-Labs-HQ/flash)
- [Release Notes](/releases.md)
- [Contributing Guide](/contributing)
- [Plugin System](PLUGIN_SYSTEM.md)
