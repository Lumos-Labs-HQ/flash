---
title: CLI Reference
description: Complete reference for FlashORM CLI commands
---

# CLI Reference

This page provides a complete reference for all FlashORM CLI commands.

## Global Options

- `--config, -c`: Specify config file path (default: `./flash.toml`)
- `--force, -f`: Skip confirmations
- `--version, -v`: Show CLI version
- `--help, -h`: Show help

## Commands

### `flash init`

Initialize a new FlashORM project.

```bash
flash init [flags]
```

**Flags:**
- `--sqlite`: Initialize for SQLite
- `--postgresql`: Initialize for PostgreSQL
- `--mysql`: Initialize for MySQL

**Examples:**
```bash
flash init --postgresql
flash init --sqlite
flash init --mysql
```

---

### `flash migrate`

Create a new migration.

```bash
flash migrate [name] [flags]
```

**Flags:**
- `--empty, -e`: Create empty migration (no auto-generated SQL)
- `--auto, -a`: Auto-generate SQL from schema changes

**Examples:**
```bash
flash migrate "add user table"
flash migrate "update schema" --auto
flash migrate --empty "custom migration"
```

---

### `flash apply`

Apply pending migrations to the database.

```bash
flash apply [flags]
```

**Flags:**
- `--force, -f`: Skip confirmations

**Examples:**
```bash
flash apply
flash apply --force
```

---

### `flash down`

Rollback migrations.

```bash
flash down [count] [flags]
```

**Parameters:**
- `count`: Number of migrations to rollback (default: 1)

**Flags:**
- `--force, -f`: Skip confirmations

**Examples:**
```bash
flash down
flash down 3
flash down 1 --force
```

---

### `flash gen`

Generate type-safe code from SQL queries.

```bash
flash gen
```

Generates code based on your `flash.toml` configuration for Go, TypeScript/JavaScript, and Python.

---

### `flash studio`

Launch FlashORM Studio web interface.

```bash
flash studio [subcommand] [flags]
```

**Subcommands:**
- `sql` (default): Launch SQL Studio for PostgreSQL, MySQL, or SQLite
- `mongodb`: Launch MongoDB Studio
- `redis`: Launch Redis Studio

**Flags:**
- `--port, -p`: Port to run studio on (default: 5555)
- `--browser, -b`: Open browser automatically (default: true)
- `--no-browser`: Disable automatic browser opening

**Examples:**
```bash
# SQL Studio (PostgreSQL, MySQL, SQLite) — loads from config
flash studio

# Auto-detect studio type from URL protocol
flash studio "postgres://user:pass@localhost:5432/mydb"
flash studio "mysql://user:pass@localhost:3306/mydb"
flash studio "sqlite:///path/to/db.sqlite"

# MongoDB Studio — auto-detected from mongodb:// URL
flash studio "mongodb://localhost:27017/mydb"
flash studio "mongodb+srv://user:pass@cluster.mongodb.net/mydb"

# Redis Studio — auto-detected from redis:// URL
flash studio "redis://localhost:6379"
flash studio "redis://:password@localhost:6379" --port 3000
```

---

### `flash pull`

Pull schema from existing database.

```bash
flash pull [flags]
```

**Flags:**
- `--db`: Database URL to pull from
- `--output, -o`: Output directory for schema files

**Examples:**
```bash
flash pull --db "postgres://user:pass@localhost:5432/mydb"
flash pull --db "postgres://..." --output db/schema
flash pull
```

---

### `flash export`

Export data from database.

```bash
flash export [flags]
```

**Flags:**
- `--format, -f`: Export format (json, csv, sqlite)
- `--output, -o`: Output file path
- `--table, -t`: Specific table to export
- `--query, -q`: Custom SQL query for export

**Examples:**
```bash
flash export --format json --output data.json
flash export --table users --format csv
flash export --format sqlite --output dump.db
```

---

### `flash branch`

Manage database schema branches.

```bash
flash branch [command]
```

**Subcommands:**
- `create <name>`: Create new branch
- `switch <name>`: Switch to branch
- `merge <source>`: Merge branch
- `list`: List all branches
- `delete <name>`: Delete branch

**Examples:**
```bash
flash branch create feature-auth
flash branch switch feature-auth
flash branch list
flash branch merge feature-auth
flash branch delete feature-auth
```

---

### `flash status`

Show current migration and branch status.

```bash
flash status
```

---

### `flash seed`

Seed database with realistic fake data. Foreign key relationships are handled automatically.

```bash
flash seed [tables...] [flags]
```

**Arguments:**
- `tables...`: Optional list of tables with counts in format `table:count` (e.g., `users:100 posts:500`)

**Flags:**
- `--count, -c`: Number of rows to generate per table (default: 10)
- `--truncate, -t`: Truncate tables before seeding
- `--force, -f`: Skip confirmation
- `--dry-run, -d`: Preview data without inserting
- `--exclude, -x`: Comma-separated tables to skip

**Examples:**
```bash
# Seed all tables with default count (10)
flash seed

# Seed all tables with 100 rows each
flash seed --count 100

# Seed specific tables
flash seed users posts

# Seed multiple tables with different counts
flash seed users:100 posts:500 comments:1000

# Truncate and reseed
flash seed --truncate --force

# Preview without inserting
flash seed --dry-run

# Exclude tables
flash seed --exclude logs,sessions

# E-commerce seeding
flash seed categories:10 products:100 orders:500

# Social media seeding
flash seed users:100 posts:500 likes:5000
```

**Smart Data Generation:**
FlashORM automatically generates appropriate data based on column names:
- `email` → realistic emails
- `username` → unique usernames
- `password` → secure random strings
- `token`, `api_key` → hex tokens
- `name`, `first_name`, `last_name` → human names
- `phone` → phone numbers
- `url`, `website` → URLs
- `address`, `city`, `country` → location data
- `ip_address` → IPv4 addresses
- `color` → `#RRGGBB` hex colors
- `created_at` / `updated_at` → coordinated timestamps
- `is_active`, `has_permission`, `can_edit` → booleans
- `price`, `amount` → currency values
- `metadata` → JSON objects

---

### `flash reset`

Reset database to clean state (drops all tables).

```bash
flash reset [flags]
```

**Flags:**
- `--force, -f`: Skip confirmation

**Examples:**
```bash
flash reset
flash reset --force
flash reset --force && flash apply && flash seed --count 50
```

---

### `flash raw`

Execute raw SQL commands.

```bash
flash raw [flags]
```

**Flags:**
- `--file, -f`: SQL file to execute
- `--query, -q`: Inline SQL query

**Examples:**
```bash
flash raw --file db/seeds/admin_users.sql
flash raw --query "SELECT * FROM users LIMIT 5"
```

---

### `flash plugins`

Manage FlashORM plugins.

```bash
flash plugins [command]
```

**Subcommands:**
- `list`: List installed plugins
- `add <name>`: Install plugin
- `remove <name>`: Remove plugin

---

### `flash add-plug`

Install a plugin.

```bash
flash add-plug <plugin-name>
```

**Available plugins:**
- `core`: ORM, migrations, code generation, seeding. Auto-installs on first use.
- `studio`: Web-based visual database editor. Optional, install when needed.

**Example:**
```bash
flash add-plug studio
```

---

### `flash rm-plug`

Remove a plugin.

```bash
flash rm-plug <plugin-name>
```

**Example:**
```bash
flash rm-plug studio
```

---

### `flash update`

Update installed plugins and optionally the flash binary itself.

```bash
flash update [flags]
```

**Flags:**
- `--self`: Update all plugins and the flash binary
- `--self-only`: Update only the flash binary, leave plugins untouched

**Examples:**
```bash
# Update all installed plugins
flash update

# Update plugins and the flash binary
flash update --self

# Update only the flash binary
flash update --self-only
```

---

## Configuration

FlashORM uses `flash.toml` for configuration:

```toml
version = "2"
schema_dir = "db/schema"
queries = "db/queries/"
migrations_path = "db/migrations"
export_path = "db/export"

[database]
provider = "postgresql"
url_env = "DATABASE_URL"

[gen.go]
enabled = true

[gen.js]
enabled = false
out = "flash_gen"

[gen.python]
enabled = false
out = "flash_gen"
async = true
```

## Environment Variables

- `DATABASE_URL`: Database connection string
- `FLASH_CONFIG`: Path to config file (alternative to --config)

## Exit Codes

- `0`: Success
- `1`: Error
- `2`: Plugin not found
- `3`: Migration conflict
