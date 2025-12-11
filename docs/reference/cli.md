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
flash studio [flags]
```

**Flags:**
- `--port, -p`: Port to run studio on (default: 5555)
- `--browser, -b`: Open browser automatically (default: true)
- `--db`: Database URL (overrides config)
- `--redis`: Redis URL for Redis Studio

**Examples:**
```bash
flash studio
flash studio --port 3000
flash studio --db "postgres://user:pass@localhost:5432/mydb"
flash studio --redis "redis://localhost:6379"
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
