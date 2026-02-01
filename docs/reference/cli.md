---
title: CLI Reference
description: Complete reference for FlashORM CLI commands
---

# CLI Reference

This page provides a complete reference for all FlashORM CLI commands.

## Global Options

- `--config, -c`: Specify config file path (default: `./flash.config.json`)
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

### `flash gen`

Generate type-safe code from SQL queries.

```bash
flash gen
```

Generates code based on your `flash.config.json` configuration for Go, TypeScript/JavaScript, and Python.

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
- `--db`: Database URL (overrides config)
- `--url`: Connection URL (for redis/mongodb subcommands)

**Examples:**
```bash
# SQL Studio (PostgreSQL, MySQL, SQLite)
flash studio
flash studio --port 3000
flash studio --db "postgres://user:pass@localhost:5432/mydb"
flash studio sql --db "mysql://user:pass@localhost:3306/mydb"

# MongoDB Studio
flash studio mongodb --url "mongodb://localhost:27017/mydb"
flash studio mongodb --url "mongodb+srv://user:pass@cluster.mongodb.net/mydb"

# Redis Studio
flash studio redis --url "redis://localhost:6379"
flash studio redis --url "redis://:password@localhost:6379" --port 3000
```

### `flash pull`

Pull schema from existing database.

```bash
flash pull [flags]
```

**Flags:**
- `--db`: Database URL to pull from
- `--output, -o`: Output directory for schema files

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
```

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

### `flash status`

Show current migration and branch status.

```bash
flash status
```

### `flash seed`

Seed database with realistic fake data for development and testing.

```bash
flash seed [tables...] [flags]
```

**Arguments:**
- `tables...`: Optional list of tables with counts in format `table:count` (e.g., `users:100 posts:500`)

**Flags:**
- `--count, -c`: Number of rows to generate per table (default: 10)
- `--table, -t`: Specific table to seed (alternative to positional args)
- `--truncate`: Truncate tables before seeding
- `--force, -f`: Skip confirmation
- `--relations`: Include foreign key relationships

**Examples:**
```bash
# Seed all tables with default count (10)
flash seed

# Seed all tables with 100 rows each
flash seed --count 100

# Seed specific table
flash seed --table users --count 50

# Seed multiple tables with different counts
flash seed users:100 posts:500 comments:1000

# Truncate and reseed
flash seed --truncate --force

# Seed with relationship handling
flash seed --relations --count 50
```

**Smart Data Generation:**
FlashORM automatically generates appropriate data based on column names:
- `email` → realistic emails
- `name`, `first_name`, `last_name` → human names
- `phone` → phone numbers
- `url`, `website` → URLs
- `address`, `city`, `country` → location data
- `created_at`, `updated_at` → timestamps
- `password` → hashed passwords
- `price`, `amount` → currency values

### `flash reset`

Reset database to clean state (drops all tables).

```bash
flash reset [flags]
```

**Flags:**
- `--force, -f`: Skip confirmation

### `flash down`

Rollback migrations.

```bash
flash down [count] [flags]
```

**Parameters:**
- `count`: Number of migrations to rollback (default: 1)

**Flags:**
- `--force, -f`: Skip confirmation

### `flash raw`

Execute raw SQL commands.

```bash
flash raw [flags]
```

**Flags:**
- `--file, -f`: SQL file to execute
- `--query, -q`: Inline SQL query

### `flash plugins`

Manage FlashORM plugins.

```bash
flash plugins [command]
```

**Subcommands:**
- `list`: List installed plugins
- `add <name>`: Install plugin
- `remove <name>`: Remove plugin

### `flash add-plug`

Install a plugin.

```bash
flash add-plug <plugin-name>
```

**Available plugins:**
- `core`: Core functionality (init, migrate, gen, apply)
- `studio`: Web studio interface
- `all`: All plugins

### `flash rm-plug`

Remove a plugin.

```bash
flash rm-plug <plugin-name>
```

## Configuration

FlashORM uses `flash.config.json` for configuration:

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
      "enabled": true
    },
    "js": {
      "enabled": false,
      "out": "flash_gen"
    },
    "python": {
      "enabled": false,
      "out": "flash_gen",
      "async": true
    }
  }
}
```

## Environment Variables

- `DATABASE_URL`: Database connection string
- `FLASH_CONFIG`: Path to config file (alternative to --config)

## Exit Codes

- `0`: Success
- `1`: Error
- `2`: Plugin not found
- `3`: Migration conflict
