---
title: Configuration Reference
description: Complete reference for FlashORM configuration options
---

# Configuration Reference

This page provides a complete reference for FlashORM configuration options in `flash.toml`.

## File Structure

FlashORM uses a TOML configuration file named `flash.toml` in your project root.

## Configuration Schema

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
driver = "database/sql"

[gen.js]
enabled = false
out = "flash_gen"
driver = "pg"

[gen.python]
enabled = false
out = "flash_gen"
async = true
driver = "asyncpg"
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

### `database` (table)

Database configuration.

#### `database.provider` (string)

Database provider. Options:
- `"postgresql"` (default)
- `"mysql"`
- `"sqlite"`
- `"mongodb"`

#### `database.url_env` (string)

Environment variable name for database URL. Default: `"DATABASE_URL"`

### `gen` (tables)

Code generation configuration. Each language has its own table: `[gen.go]`, `[gen.js]`, `[gen.python]`.

#### `[gen.go]`

Go code generation settings.

##### `gen.go.enabled` (boolean)

Enable Go code generation. Default: `true` when no other generators are enabled.

##### `gen.go.driver` (string)

Go database driver. Default: `"database/sql"`

| Driver | Description | Best For |
|--------|-------------|----------|
| `database/sql` | Standard library SQL interface | Portability, simplicity |
| `pgx` | jackc/pgx/v5 native driver | PostgreSQL performance, native types |

```toml
[gen.go]
enabled = true
driver = "pgx"
```

#### `[gen.js]`

JavaScript/TypeScript code generation settings.

##### `gen.js.enabled` (boolean)

Enable JavaScript/TypeScript code generation. Default: `false`

##### `gen.js.out` (string)

Output directory for generated JS/TS code. Default: `"flash_gen"`

##### `gen.js.driver` (string)

JavaScript database driver. Default: `"pg"`

| Driver | Description | Database |
|--------|-------------|----------|
| `pg` | node-postgres (default) | PostgreSQL |
| `postgres` | porsager/postgres | PostgreSQL |
| `mysql2` | mysql2 package | MySQL |
| `better-sqlite3` | Synchronous SQLite | SQLite |
| `bun:sqlite` | Bun native SQLite | SQLite |

```toml
[gen.js]
enabled = true
out = "flash_gen"
driver = "mysql2"
```

#### `[gen.python]`

Python code generation settings.

##### `gen.python.enabled` (boolean)

Enable Python code generation. Default: `false`

##### `gen.python.out` (string)

Output directory for generated Python code. Default: `"flash_gen"`

##### `gen.python.async` (boolean)

Generate async Python code. Default: `true`

##### `gen.python.driver` (string)

Python database driver. Defaults depend on `database.provider`:

**PostgreSQL:**
| Driver | Description | Mode |
|--------|-------------|------|
| `asyncpg` | asyncpg native (default async) | Async |
| `psycopg3` | psycopg 3.x | Sync or Async |

**MySQL:**
| Driver | Description | Mode |
|--------|-------------|------|
| `aiomysql` | aiomysql (default async) | Async |
| `pymysql` | PyMySQL | Sync |

**SQLite:**
| Driver | Description | Mode |
|--------|-------------|------|
| `aiosqlite` | aiosqlite (default async) | Async |
| `sqlite3` | Standard library sqlite3 | Sync |

```toml
[gen.python]
enabled = true
out = "flash_gen"
async = true
driver = "psycopg3"
```

## Driver Selection by Database

### PostgreSQL

| Language | Default Driver | Alternative Drivers |
|----------|---------------|---------------------|
| Go | `database/sql` | `pgx` |
| JavaScript | `pg` | `postgres` |
| Python | `asyncpg` (async) | `psycopg3` |

### MySQL

| Language | Default Driver | Alternative Drivers |
|----------|---------------|---------------------|
| Go | `database/sql` | — |
| JavaScript | `mysql2` | — |
| Python | `aiomysql` (async) | `pymysql` |

### SQLite

| Language | Default Driver | Alternative Drivers |
|----------|---------------|---------------------|
| Go | `database/sql` | — |
| JavaScript | `better-sqlite3` | `bun:sqlite` |
| Python | `aiosqlite` (async) | `sqlite3` |

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
├── flash.toml
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

### Go Project with PostgreSQL (pgx)

```toml
version = "2"
schema_dir = "db/schema"
queries = "db/queries/"
migrations_path = "db/migrations"

[database]
provider = "postgresql"
url_env = "DATABASE_URL"

[gen.go]
enabled = true
driver = "pgx"
```

### Node.js Project with TypeScript (postgres driver)

```toml
version = "2"
schema_dir = "db/schema"
queries = "db/queries/"
migrations_path = "db/migrations"

[database]
provider = "postgresql"
url_env = "DATABASE_URL"

[gen.js]
enabled = true
out = "src/generated"
driver = "postgres"
```

### Python Project (psycopg3, sync)

```toml
version = "2"
schema_dir = "db/schema"
queries = "db/queries/"
migrations_path = "db/migrations"

[database]
provider = "postgresql"
url_env = "DATABASE_URL"

[gen.python]
enabled = true
out = "flashorm_gen"
async = false
driver = "psycopg3"
```

### Python Project (MySQL with PyMySQL)

```toml
version = "2"
schema_dir = "db/schema"
queries = "db/queries/"
migrations_path = "db/migrations"

[database]
provider = "mysql"
url_env = "DATABASE_URL"

[gen.python]
enabled = true
out = "flashorm_gen"
async = false
driver = "pymysql"
```

### Multi-Language Project

```toml
version = "2"
schema_dir = "db/schema"
queries = "db/queries/"
migrations_path = "db/migrations"

[database]
provider = "postgresql"
url_env = "DATABASE_URL"

[gen.go]
enabled = true
driver = "database/sql"

[gen.js]
enabled = true
out = "frontend/src/generated"
driver = "pg"

[gen.python]
enabled = true
out = "backend/flashorm_gen"
async = true
driver = "asyncpg"
```
