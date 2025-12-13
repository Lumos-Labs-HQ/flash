---
title: Configuration Reference
description: Complete reference for FlashORM configuration options
---

# Configuration Reference

This page provides a complete reference for FlashORM configuration options in `flash.config.json`.

## File Structure

FlashORM uses a JSON configuration file named `flash.config.json` in your project root.

## Configuration Schema

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

## Configuration Options

### `version` (string)

Configuration format version. Currently `"2"`.

### `schema_dir` (string)

Directory containing SQL schema files. Default: `"db/schema"`

::: tip
This replaces the deprecated `schema_path` option.
:::

### `queries` (string)

Directory containing SQL query files. Default: `"db/queries/"`

### `migrations_path` (string)

Directory for migration files. Default: `"db/migrations"`

### `export_path` (string)

Directory for exported data files. Default: `"db/export"`

### `database` (object)

Database configuration.

#### `database.provider` (string)

Database provider. Options:
- `"postgresql"` (default)
- `"mysql"`
- `"sqlite"`
- `"mongodb"`

#### `database.url_env` (string)

Environment variable name for database URL. Default: `"DATABASE_URL"`

### `gen` (object)

Code generation configuration.

#### `gen.go` (object)

Go code generation settings.

##### `gen.go.enabled` (boolean)

Enable Go code generation. Default: `true` when no other generators are enabled.

#### `gen.js` (object)

JavaScript/TypeScript code generation settings.

##### `gen.js.enabled` (boolean)

Enable JavaScript/TypeScript code generation. Default: `false`

##### `gen.js.out` (string)

Output directory for generated JS/TS code. Default: `"flash_gen"`

#### `gen.python` (object)

Python code generation settings.

##### `gen.python.enabled` (boolean)

Enable Python code generation. Default: `false`

##### `gen.python.out` (string)

Output directory for generated Python code. Default: `"flash_gen"`

##### `gen.python.async` (boolean)

Generate async Python code. Default: `true`

## Database URLs

### PostgreSQL

```bash
export DATABASE_URL="postgres://user:password@localhost:5432/database"
# or
export DATABASE_URL="postgresql://user:password@localhost:5432/database"
```

### MySQL

```bash
export DATABASE_URL="user:password@tcp(localhost:3306)/database"
```

### SQLite

```bash
export DATABASE_URL="sqlite://./data.db"
# or for in-memory
export DATABASE_URL="sqlite://:memory:"
```

### MongoDB

```bash
export DATABASE_URL="mongodb://localhost:27017/database"
```

## Environment Variables

You can override configuration using environment variables:

- `FLASH_SCHEMA_DIR`: Override `schema_dir`
- `FLASH_QUERIES_DIR`: Override `queries`
- `FLASH_MIGRATIONS_DIR`: Override `migrations_path`
- `FLASH_EXPORT_DIR`: Override `export_path`
- `FLASH_DATABASE_PROVIDER`: Override `database.provider`

## Project Structure

FlashORM expects the following directory structure:

```
project/
├── flash.config.json
├── db/
│   ├── schema/
│   │   └── *.sql          # Schema files
│   ├── queries/
│   │   └── *.sql          # Query files
│   ├── migrations/        # Generated migrations
│   └── export/            # Exported data
├── flash_gen/             # Generated code
└── .env                   # Environment variables
```

## Examples

### Go Project with PostgreSQL

```json
{
  "version": "2",
  "schema_dir": "db/schema",
  "queries": "db/queries/",
  "migrations_path": "db/migrations",
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

### Node.js Project with TypeScript

```json
{
  "version": "2",
  "schema_dir": "db/schema",
  "queries": "db/queries/",
  "migrations_path": "db/migrations",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "js": {
      "enabled": true,
      "out": "src/generated"
    }
  }
}
```

### Python Project

```json
{
  "version": "2",
  "schema_dir": "db/schema",
  "queries": "db/queries/",
  "migrations_path": "db/migrations",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "python": {
      "enabled": true,
      "out": "flashorm_gen",
      "async": true
    }
  }
}
```

### Multi-Language Project

```json
{
  "version": "2",
  "schema_dir": "db/schema",
  "queries": "db/queries/",
  "migrations_path": "db/migrations",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "go": {
      "enabled": true
    },
    "js": {
      "enabled": true,
      "out": "frontend/src/generated"
    },
    "python": {
      "enabled": true,
      "out": "backend/flashorm_gen",
      "async": true
    }
  }
}
```
