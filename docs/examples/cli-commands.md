---
title: CLI Commands Examples
description: Every Flash ORM CLI command with all examples
---

# CLI Commands Examples

Complete examples for every Flash ORM CLI command. Each section shows all flags and real-world usage patterns.

---

## `flash init` — Initialize Project

Create a new Flash ORM project with the correct structure.

```bash
# Initialize for PostgreSQL (default)
flash init --postgresql

# Initialize for MySQL
flash init --mysql

# Initialize for SQLite
flash init --sqlite

# All init commands create:
# ├── flash.toml
# ├── db/schema/
# ├── db/queries/
# ├── db/migrations/
# └── .env
```

---

## `flash migrate` — Create Migrations

Generate migration files from schema changes.

```bash
# Interactive migration (prompts for name)
flash migrate

# Named migration
flash migrate "add users table"

# Auto-generate SQL from schema changes (recommended)
flash migrate "add posts and comments" --auto

# Create empty migration for custom SQL
flash migrate "custom data fix" --empty

# Migration files are created in db/migrations/:
# 20240101120000_add_users_table.up.sql
# 20240101120000_add_users_table.down.sql
```

---

## `flash apply` — Apply Migrations

Run pending migrations against the database.

```bash
# Apply all pending migrations
flash apply

# Apply with force (skip confirmations)
flash apply --force

# Dry run - show what would be executed without running
flash apply --dry-run

# Apply specific number of migrations
flash apply --count 1
```

---

## `flash down` — Rollback Migrations

Rollback previously applied migrations.

```bash
# Rollback last migration
flash down

# Rollback last 3 migrations
flash down 3

# Force rollback without confirmation
flash down 1 --force
```

---

## `flash gen` — Generate Code

Generate type-safe code from schema and queries.

```bash
# Generate code for all enabled languages
flash gen

# The generated code goes to flash_gen/ by default
# Go:     flash_gen/db.go, flash_gen/models.go, flash_gen/*.go
# JS/TS:  flash_gen/index.js, flash_gen/index.d.ts
# Python: flash_gen/__init__.py, flash_gen/database.py
```

---

## `flash seed` — Seed Database

Generate and insert realistic fake data. Foreign key relationships are handled automatically.

```bash
# Seed all tables with default count (10 records each)
flash seed

# Seed all tables with custom count
flash seed --count 100

# Seed specific tables
flash seed users posts

# Seed with custom counts per table
flash seed users:100 posts:500 comments:2000

# Schema-qualified table names work too
flash seed public.users:50

# Truncate tables before seeding (fresh start)
flash seed --truncate

# Skip confirmation prompts
flash seed --truncate --force

# Preview data without inserting (dry run)
flash seed --dry-run

# Exclude specific tables
flash seed --exclude logs,sessions

# Combine options
flash seed users:50 posts:200 --truncate --force

# E-commerce example
flash seed categories:10 products:100 orders:500 order_items:2000

# Social media example
flash seed users:100 posts:500 likes:5000 follows:2000

# Large dataset for performance testing
flash seed --count 10000 --force
```

### Seeding with Relationships

Given this schema:

```sql
CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(100));
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    title VARCHAR(200)
);
CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    post_id INTEGER REFERENCES posts(id),
    content TEXT
);
```

```bash
# Flash automatically seeds in dependency order:
# 1. users (no dependencies)
# 2. posts (depends on users)
# 3. comments (depends on posts)
flash seed users:50 posts:200 comments:1000
```

---

## `flash studio` — Launch Studio

Launch the visual database management interface.

```bash
# SQL Studio - auto-detects from config
flash studio

# SQL Studio with custom port
flash studio --port 8080

# SQL Studio without opening browser
flash studio --no-browser

# Direct URL (auto-detects studio type)
flash studio "postgres://user:pass@localhost:5432/mydb"
flash studio "mysql://user:pass@localhost:3306/mydb"
flash studio "sqlite:///path/to/db.sqlite"

# MongoDB Studio
flash studio "mongodb://localhost:27017/mydb"
flash studio "mongodb+srv://user:pass@cluster.mongodb.net/mydb"

# Redis Studio
flash studio "redis://localhost:6379"
flash studio "redis://:password@localhost:6379"

# MongoDB Studio explicit
flash studio mongodb

# Redis Studio explicit
flash studio redis
```

---

## `flash pull` — Pull Schema

Introspect an existing database and generate schema files.

```bash
# Pull from database URL
flash pull --db "postgres://user:pass@localhost:5432/mydb"

# Pull with custom output directory
flash pull --db "postgres://..." --output db/schema

# Pull into current project
flash pull
```

---

## `flash export` — Export Data

Export database data to various formats.

```bash
# Export entire database to JSON
flash export --format json --output backup.json

# Export to CSV
flash export --format csv --output data.csv

# Export to SQLite file
flash export --format sqlite --output dump.db

# Export specific table
flash export --table users --format json --output users.json

# Export with custom query
flash export --query "SELECT * FROM users WHERE active = true" --format csv
```

---

## `flash branch` — Branch Management

Git-like branching for database schemas.

```bash
# Create new branch
flash branch create feature-auth

# Switch to branch
flash branch switch feature-auth

# List all branches
flash branch list

# Merge branch into current
flash branch merge feature-auth

# Delete branch
flash branch delete feature-auth
```

---

## `flash status` — Show Status

Display current migration and branch status.

```bash
flash status

# Example output:
# Database: postgresql
# Branch: main
# Migrations:
#   ✅ 20240101120000_initial_schema
#   ✅ 20240102130000_add_users_table
#   ⏳ 20240103140000_add_posts_table (pending)
```

---

## `flash reset` — Reset Database

Drop all tables and reset to clean state.

```bash
# Reset with confirmation prompt
flash reset

# Force reset without confirmation
flash reset --force

# Common pattern: full reset + re-apply + seed
flash reset --force && flash apply && flash seed --count 50
```

---

## `flash raw` — Execute Raw SQL

Run raw SQL commands directly.

```bash
# Execute SQL file
flash raw --file db/seeds/admin_users.sql

# Execute inline query
flash raw --query "SELECT * FROM users LIMIT 5"

# Execute multiple statements from file
flash raw --file db/fixes/cleanup.sql
```

---

## `flash plugins` — Plugin Management

Manage Flash ORM plugins.

```bash
# List installed plugins
flash plugins list

# Add studio plugin
flash add-plug studio

# Remove plugin
flash rm-plug studio

# Update all plugins
flash update

# Update plugins and binary
flash update --self

# Update only binary
flash update --self-only
```

---

## Combined Workflow Examples

### Development Setup

```bash
# Fresh start for development
flash reset --force
flash apply
flash seed --count 50
flash studio
```

### CI/CD Pipeline

```bash
# Automated test database setup
flash reset --force
flash apply --force
flash seed --count 25 --force
npm test
```

### Schema Change Workflow

```bash
# 1. Edit schema files
vim db/schema/posts.sql

# 2. Generate migration
flash migrate "add post metadata"

# 3. Review migration
# vim db/migrations/20240101120000_add_post_metadata.up.sql

# 4. Apply
flash apply

# 5. Regenerate code
flash gen

# 6. Test
flash seed posts:10
```

### Production Deployment

```bash
# 1. Backup before migration
flash export --format sqlite --output pre-migration-backup.db

# 2. Apply migrations
flash apply

# 3. Verify
flash status

# 4. Regenerate if needed
flash gen
```
