---
title: Architecture
description: Complete end-to-end architecture of Flash ORM
---

# Flash ORM Architecture

This document explains the complete internal architecture of Flash ORM — how schema definition, migrations, code generation, seeding, and the studio work together from configuration to execution.

## Table of Contents

- [High-Level Overview](#high-level-overview)
- [Configuration Layer](#configuration-layer)
- [Schema & Parsing Pipeline](#schema--parsing-pipeline)
- [Migration Engine](#migration-engine)
- [Code Generation Pipeline](#code-generation-pipeline)
- [Database Seeding](#database-seeding)
- [Studio Architecture](#studio-architecture)
- [Database Adapters](#database-adapters)
- [Plugin System](#plugin-system)
- [End-to-End Flow](#end-to-end-flow)
- [Key Design Decisions](#key-design-decisions)

---

## High-Level Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Flash ORM Architecture                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐             │
│   │  Schema Files │    │  Query Files │    │ flash.config │             │
│   │  db/schema/   │    │  db/queries/ │    │    .toml     │             │
│   └──────┬───────┘    └──────┬───────┘    └──────┬───────┘             │
│          │                   │                   │                      │
│          └───────────────────┼───────────────────┘                      │
│                              ▼                                          │
│   ┌──────────────────────────────────────────────────────┐             │
│   │                   Schema Parser                       │             │
│   │  - Parses CREATE TABLE, CREATE TYPE, CREATE INDEX    │             │
│   │  - Extracts columns, constraints, enums, FKs         │             │
│   │  - Validates references & detects circular deps      │             │
│   └────────────────────────┬─────────────────────────────┘             │
│                            │                                            │
│              ┌─────────────┼─────────────┐                             │
│              ▼             ▼             ▼                             │
│   ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                  │
│   │   Migration  │ │   Code Gen   │ │    Seeder    │                  │
│   │    Engine    │ │   Pipeline   │ │              │                  │
│   └──────┬───────┘ └──────┬───────┘ └──────┬───────┘                  │
│          │                │                │                           │
│   ┌──────┴────────────────┴────────────────┴───────┐                   │
│   │              Database Adapter Interface         │                   │
│   │    ┌─────────┐ ┌─────────┐ ┌─────────┐        │                   │
│   │    │ Postgres│ │  MySQL  │ │  SQLite │        │                   │
│   │    └─────────┘ └─────────┘ └─────────┘        │                   │
│   └────────────────────────────────────────────────┘                   │
│                                                                         │
│   ┌──────────────────────────────────────────────────────┐             │
│   │              Studio Servers (HTTP)                    │             │
│   │    ┌──────────┐ ┌──────────┐ ┌──────────┐           │             │
│   │    │  SQL     │ │ MongoDB  │ │  Redis   │           │             │
│   │    │  Studio  │ │  Studio  │ │  Studio  │           │             │
│   │    └──────────┘ └──────────┘ └──────────┘           │             │
│   └──────────────────────────────────────────────────────┘             │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Configuration Layer

Flash ORM is driven by a single configuration file: `flash.toml`.

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

[gen.js]
enabled = true
out = "flash_gen"

[gen.python]
enabled = true
out = "flash_gen"
async = true
```

### Configuration Resolution

1. **Load**: `config.Load()` reads `flash.toml` (with `sync.Once` caching)
2. **Defaults**: Missing fields are populated with sensible defaults
3. **Validation**: `config.Validate()` ensures provider is supported and paths are valid
4. **Environment**: `config.GetDatabaseURL()` reads the actual DB URL from the specified env var

### Dual-Mode Support

Flash supports two workflows:

| Mode | Description | Entry Point |
|------|-------------|-------------|
| **Config-driven** | Uses `flash.toml` for all operations | `flash studio`, `flash apply`, `flash gen` |
| **URL-driven** | Pass connection string directly | `flash studio "postgres://..."` |

---

## Schema & Parsing Pipeline

### Input Sources

```
db/schema/
├── users.sql          # CREATE TABLE users (...)
├── posts.sql          # CREATE TABLE posts (...)
└── enums.sql          # CREATE TYPE status AS ENUM (...)

db/queries/
├── users.sql          # -- name: GetUser :one
└── posts.sql          # -- name: ListPosts :many
```

### Schema Parsing Flow

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Raw SQL Files  │────▶│  SchemaParser    │────▶│  parser.Schema  │
│  db/schema/*.sql│     │                  │     │                 │
└─────────────────┘     │  1. Split stmts  │     │  - Tables[]     │
                        │  2. Parse tables │     │  - Enums[]      │
                        │  3. Parse enums  │     │  - Indexes[]    │
                        │  4. Parse indexes│     └─────────────────┘
                        └──────────────────┘
```

**Key parsing steps:**

1. **Statement Splitting**: SQL is split on `;` while respecting string literals, comments, and PostgreSQL dollar-quoted blocks
2. **Table Parsing**: `CREATE TABLE` statements are parsed into `SchemaTable` structs with columns, indexes, and constraints
3. **Enum Parsing**: `CREATE TYPE ... AS ENUM` extracts values into `SchemaEnum`. MySQL `ENUM(...)` inline enums are also supported
4. **Index Parsing**: Standalone `CREATE INDEX` statements are captured separately
5. **Validation**: Foreign key references are checked for existence; circular dependencies are detected

### Query Parsing Flow

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Query SQL Files│────▶│  QueryParser     │────▶│  []parser.Query │
│  db/queries/*.sql│     │                  │     │                 │
└─────────────────┘     │  1. Parse name   │     │  - Name         │
                        │  2. Parse cmd    │     │  - SQL          │
                        │  3. Infer params │     │  - Params[]     │
                        │  4. Infer return │     │  - ReturnType   │
                        └──────────────────┘     └─────────────────┘
```

**Query annotations drive code generation:**

| Annotation | Behavior | Return Type |
|------------|----------|-------------|
| `:one` | Returns exactly one row | Single struct or error |
| `:many` | Returns multiple rows | Slice of structs |
| `:exec` | Execute without returning | `error` only |
| `:execresult` | Execute returning result | `sql.Result, error` |
| `:execrows` | Execute returning affected rows | `int64, error` |

---

## Migration Engine

The migration system uses a **Drizzle-style snapshot approach** for diffing. Instead of simulating pending migrations against the live database, Flash diffs your schema files against a locally-stored JSON snapshot.

### Migration Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                      Migration Engine                           │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌──────────────┐         ┌──────────────────────────────┐    │
│   │  db/schema/  │         │  .flash/schema_snapshot.json │    │
│   │  *.sql files │         │  (last known schema state)   │    │
│   └──────┬───────┘         └──────────────┬───────────────┘    │
│          │                                 │                    │
│          └──────────────┬──────────────────┘                    │
│                         ▼                                       │
│              ┌────────────────────┐                             │
│              │   SchemaManager    │                             │
│              │   GenerateSchemaDiff│                            │
│              └────────┬───────────┘                             │
│                       │                                         │
│                       ▼                                         │
│              ┌────────────────────┐                             │
│              │    SchemaDiff      │                             │
│              │  - NewTables[]     │                             │
│              │  - DroppedTables[] │                             │
│              │  - ModifiedTables[]│                             │
│              │  - NewIndexes[]    │                             │
│              │  - DroppedIndexes[]│                             │
│              └────────┬───────────┘                             │
│                       │                                         │
│                       ▼                                         │
│              ┌────────────────────┐                             │
│              │  generateSQLFromDiff│                            │
│              │                     │                             │
│              │  PostgreSQL → ALTER │                             │
│              │  MySQL → MODIFY     │                             │
│              │  SQLite → RECREATE  │                             │
│              └────────┬───────────┘                             │
│                       │                                         │
│                       ▼                                         │
│              ┌────────────────────┐                             │
│              │  Migration File    │                             │
│              │  YYYYMMDDHHMMSS_   │                             │
│              │     name.sql       │                             │
│              └────────────────────┘                             │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

### Snapshot Diffing (Source of Truth)

```go
// 1. Load local snapshot
snapshot, _ := schema.LoadSchemaSnapshot(snapshotPath)
currentTables, currentEnums := snapshot.Tables, snapshot.Enums

// 2. Parse target schema from files
targetTables, targetEnums, targetIndexes, _ := sm.ParseSchemaPath(schemaPath)

// 3. Diff snapshot vs target
diff := sm.compareSchemas(current, target, currentEnums, targetEnums, targetIndexes)
```

**Why snapshot diffing?**

- **Offline generation**: You can generate migrations without a running database
- **No phantom diffs**: Unapplied migrations don't cause false positives
- **Fast**: No database round-trips needed for diffing

### Provider-Specific Migration SQL

| Operation | PostgreSQL | MySQL | SQLite |
|-----------|-----------|-------|--------|
| Add column | `ALTER TABLE ... ADD COLUMN` | `ALTER TABLE ... ADD COLUMN` | `ALTER TABLE ... ADD COLUMN` |
| Drop column | `ALTER TABLE ... DROP COLUMN` | `ALTER TABLE ... DROP COLUMN` | `ALTER TABLE ... DROP COLUMN` (3.35+) |
| Modify column | `ALTER TABLE ... ALTER COLUMN TYPE` | `ALTER TABLE ... MODIFY COLUMN` | **Full table recreation** |
| Add index | `CREATE INDEX` | `CREATE INDEX` | `CREATE INDEX` |
| Drop index | `DROP INDEX` | `DROP INDEX` | `DROP INDEX` |

**SQLite Column Modification**: SQLite does not support `ALTER COLUMN`. Flash generates a full table recreation:

```sql
PRAGMA foreign_keys=OFF;
CREATE TABLE "users_new" (...new schema...);
INSERT INTO "users_new" (...) SELECT ... FROM "users";
DROP TABLE "users";
ALTER TABLE "users_new" RENAME TO "users";
PRAGMA foreign_keys=ON;
```

### Apply Flow

```
flash apply
    │
    ▼
┌────────────────────┐
│ loadMigrationsFromDir│  ← Reads db/migrations/*.sql
└────────┬───────────┘
         │
         ▼
┌────────────────────┐
│ getAppliedMigrations│  ← SELECT FROM _flash_migrations
│                     │  ← Gracefully handles missing table
└────────┬───────────┘
         │
         ▼
┌────────────────────┐
│ FilterPending      │  ← Compares file list vs applied set
└────────┬───────────┘
         │
         ▼
┌────────────────────┐
│ DetectConflicts    │  ← Checks for NOT NULL without DEFAULT
└────────┬───────────┘
         │
         ▼
┌────────────────────┐
│ applyMigrations    │  ← Each in its own transaction
│                     │  ← SHA256 checksum verification
└────────────────────┘
```

### Checksum Verification

Each migration is checksummed with SHA256:

```go
checksum := utils.ComputeChecksum(content) // SHA256 of file content
// Stored in _flash_migrations table
```

This detects tampering — if someone edits an already-applied migration, Flash warns you.

---

## Code Generation Pipeline

Flash generates type-safe database code for **Go**, **TypeScript/JavaScript**, and **Python**.

### Generation Flow

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  parser.Schema  │────▶│   gencommon.     │────▶│  Per-language   │
│  parser.Query[] │     │   GenerationCache│     │  Generators     │
└─────────────────┘     └──────────────────┘     │  ┌───────────┐  │
                                                 │  │    Go     │  │
                                                 │  │  gogen    │  │
                                                 │  ├───────────┤  │
                                                 │  │    JS     │  │
                                                 │  │  jsgen    │  │
                                                 │  ├───────────┤  │
                                                 │  │  Python   │  │
                                                 │  │  pygen    │  │
                                                 │  └───────────┘  │
                                                 └─────────────────┘
```

### Incremental Generation

Flash uses a caching system to avoid regenerating unchanged files:

```go
type GenerationCache struct {
    SchemaChecksum      string            // Hash of all schema files
    ConfigChecksum      string            // Hash of generation config
    QueryFileChecksums  map[string]string // Per-query file SHA256
    GeneratedFileChecksums map[string]string // Detects manual edits
}
```

**Full regeneration** triggers when:
- Schema files change
- Config changes (provider, output dir, async flag)

**Incremental regeneration** skips files whose query SQL hasn't changed.

### Go Generation (`gogen`)

```
flash_gen/
├── db.go          # DBTX interface + Queries struct
├── models.go      # Structs for all tables + enum constants
└── *.go           # One file per query file
```

**Key features:**
- **Prepared statement caching** for hot queries (`len(params) <= 3 && !UNION`)
- **Raw string literals** for SQL with backtick-safe concatenation
- **MySQL inline ENUM support** — `enum('active','inactive')` generates typed constants
- **Zero-value generation** for `:one` queries that return no rows
- **Multi-driver support** — `pgx` and `database/sql` for PostgreSQL

### TypeScript/JavaScript Generation (`jsgen`)

```
flash_gen/
├── index.js       # Runtime query implementations
├── index.d.ts     # TypeScript declarations
└── database.js    # Connection wrapper
```

**Key features:**
- Named prepared statements for PostgreSQL
- Provider-specific execution blocks (pg, mysql2, sqlite3)
- `.d.ts` files for full IDE autocomplete

### Python Generation (`pygen`)

```
flash_gen/
├── __init__.py    # Package exports
├── database.py    # Queries class
└── *.pyi          # Type stubs for IDE support
```

**Key features:**
- Sync or async generation (`asyncpg`, `aiomysql`, `aiosqlite`, `psycopg3`, etc.)
- Batch insert generation for `:exec` INSERT queries (`executemany`)
- `.pyi` stub files for IDE autocomplete

### Type Mapping

| Database Type | Go | TypeScript | Python |
|---------------|-----|------------|--------|
| `SERIAL`, `BIGINT` | `int64` | `number` | `int` |
| `INTEGER` | `int32` | `number` | `int` |
| `SMALLINT`, `TINYINT` | `int16`/`int8` | `number` | `int` |
| `VARCHAR`, `TEXT` | `string` | `string` | `str` |
| `CHAR` | `string` | `string` | `str` |
| `BOOLEAN` | `bool` | `boolean` | `bool` |
| `TIMESTAMP`, `TIMESTAMPTZ`, `DATETIME` | `time.Time` | `Date` | `datetime` |
| `DATE` | `time.Time` | `Date` | `date` |
| `TIME` | `string` | `string` | `time` |
| `JSON`, `JSONB` | `[]byte` | `any` | `dict` |
| `UUID` | `string` | `string` | `str` |
| `BYTEA`, `BLOB` | `[]byte` | `Buffer` | `bytes` |
| `ENUM('a','b')` | `Status` (const) | `'a' \| 'b'` | `Literal['a','b']` |
| `ARRAY` | `[]string` | `string[]` | `List[str]` |
| Nullable | `sql.NullString` | `string \| null` | `Optional[str]` |

### Supported Drivers

Flash generates code for multiple database drivers per language:

**Go**
| Driver | Package | Description |
|--------|---------|-------------|
| `database/sql` | Standard library | Generic SQL interface (default) |
| `pgx` | `github.com/jackc/pgx/v5` | Native PostgreSQL driver |

**JavaScript / TypeScript**
| Driver | Package | Database |
|--------|---------|----------|
| `pg` | `pg` | PostgreSQL (default) |
| `postgres` | `postgres` | PostgreSQL |
| `mysql2` | `mysql2` | MySQL |
| `better-sqlite3` | `better-sqlite3` | SQLite |
| `bun:sqlite` | Built-in | SQLite |

**Python**
| Driver | Package | Database | Mode |
|--------|---------|----------|------|
| `asyncpg` | `asyncpg` | PostgreSQL | Async (default) |
| `psycopg3` | `psycopg` | PostgreSQL | Sync / Async |
| `aiomysql` | `aiomysql` | MySQL | Async (default) |
| `pymysql` | `PyMySQL` | MySQL | Sync |
| `aiosqlite` | `aiosqlite` | SQLite | Async (default) |
| `sqlite3` | Standard library | SQLite | Sync |

---

## Database Seeding

Flash ORM includes an automatic seeding system that generates realistic fake data based on column names and SQL types.

### Seeding Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Seeding Pipeline                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   ┌──────────────┐                                          │
│   │  Schema Files │         ┌──────────────────────┐       │
│   │  db/schema/*.sql│──────▶│   Schema Parser      │       │
│   └──────────────┘         │   - Tables           │       │
│                            │   - Columns          │       │
│                            │   - FK constraints   │       │
│                            └──────────┬───────────┘       │
│                                       │                    │
│                            ┌──────────▼───────────┐       │
│                            │  Dependency Graph    │       │
│                            │  Topological sort    │       │
│                            │  (handles self-FKs)  │       │
│                            └──────────┬───────────┘       │
│                                       │                    │
│         ┌─────────────────────────────┼─────────────────┐ │
│         ▼                             ▼                 ▼ │
│   ┌────────────┐              ┌────────────┐    ┌────────┐│
│   │DataGenerator│             │insertBatch │    │insert  ││
│   │- Patterns  │             │- Multi-row │    │Record  ││
│   │- Type list │             │- RETURNING │    │- Last  ││
│   │- UUID v4   │             │  (Postgres)│    │  ID    ││
│   │- Coordinated│            │- Single-row│    │        ││
│   │  timestamps│             │  (MySQL)   │    │        ││
│   └────────────┘             └────────────┘    └────────┘│
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Seeding Flow

1. **Parse Schema**: Extract tables, columns, types, and foreign keys
2. **Build Dependency Graph**: Topologically sort tables by FK dependencies
3. **Generate Data**: For each column, pick a generator based on name pattern or SQL type
4. **Insert Batches**: Smart batching — multi-row for speed, single-row when IDs are needed
5. **Track IDs**: Store inserted IDs so subsequent tables can reference them
6. **Self-References**: Accumulate IDs within the current table for self-referencing FKs

### Smart Batching Strategy

| Database | Table Referenced by FKs | Table NOT Referenced |
|----------|------------------------|---------------------|
| PostgreSQL | Multi-row `INSERT ... RETURNING` | Multi-row `INSERT ... RETURNING` |
| MySQL | Single-row + `LAST_INSERT_ID()` | Multi-row insert (fast) |
| SQLite | Single-row + `last_insert_rowid()` | Multi-row insert (fast) |

---

## Studio Architecture

Flash provides three studio interfaces:

| Studio | Protocol | Data Operations |
|--------|----------|-----------------|
| **SQL Studio** | PostgreSQL, MySQL, SQLite | Full CRUD, SQL editor, schema viz, import/export |
| **MongoDB Studio** | MongoDB | Collections, documents, aggregation pipeline, indexes |
| **Redis Studio** | Redis | Keys, CLI, pub/sub, slow log, Lua scripts, memory stats |

### Launch Architecture

```bash
# Auto-detect from URL protocol
flash studio "postgres://localhost:5432/db"     → SQL Studio
flash studio "mongodb://localhost:27017/db"     → MongoDB Studio
flash studio "redis://localhost:6379"           → Redis Studio
flash studio "sqlite:///path/to/db.sqlite"      → SQL Studio

# Load from config (no argument)
flash studio                                    → Reads flash.toml
```

### SQL Studio Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      SQL Studio Server                       │
│                    (Go net/http + embed)                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   Frontend (Vanilla JS)          Backend (Go)              │
│   ┌─────────────────┐            ┌─────────────────┐       │
│   │  sql.js         │◄──────────►│  server.go      │       │
│   │  - CodeMirror   │   HTTP     │  - REST handlers│       │
│   │  - Query exec   │            │  - Static files │       │
│   │  - Results      │            └────────┬────────┘       │
│   └─────────────────┘                     │                │
│                                            │                │
│   ┌─────────────────┐            ┌────────▼────────┐       │
│   │  studio.js      │◄──────────►│  service.go     │       │
│   │  - Data grid    │            │  - Business logic│      │
│   │  - Pagination   │            │  - Import/export │      │
│   │  - Cell editing │            │  - Schema ops    │      │
│   └─────────────────┘            └────────┬────────┘       │
│                                            │                │
│                              ┌─────────────▼─────────────┐ │
│                              │    Database Adapter       │ │
│                              │  ┌─────┐┌─────┐┌────────┐ │ │
│                              │  │Pg   ││MySQL││ SQLite │ │ │
│                              │  └─────┘└─────┘└────────┘ │ │
│                              └───────────────────────────┘ │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Key features:**
- **No query modification**: User's exact SQL is sent and executed (no comment stripping)
- **Error classification**: Syntax errors return HTTP 400; server errors return 500
- **Batch export**: Fetches data in 1000-row chunks to avoid memory issues
- **Topological import**: Sorts tables by foreign key dependencies before importing

### MongoDB Studio Architecture

```
┌────────────────────────────────────────────────────────────┐
│                    MongoDB Studio Server                    │
├────────────────────────────────────────────────────────────┤
│                                                             │
│   Frontend (Vanilla JS SPA)      Backend (Go)              │
│   ┌─────────────────┐            ┌─────────────────┐       │
│   │  mongodb.js     │◄──────────►│  server.go      │       │
│   │  - JSON tree    │   HTTP     │  - REST API     │       │
│   │  - Table view   │            │  - BSON convert │       │
│   │  - Aggregation  │            └────────┬────────┘       │
│   └─────────────────┘                     │                │
│                              ┌────────────▼────────────┐  │
│                              │  go.mongodb.org/mongo   │  │
│                              │  - Connection pooling   │  │
│                              │  - Thread-safe adapter  │  │
│                              └─────────────────────────┘  │
│                                                             │
└────────────────────────────────────────────────────────────┘
```

**Critical design:** The MongoDB adapter uses `sync.RWMutex` to protect the shared database reference. All methods use `currentDB()` (RLock) to prevent data races during concurrent requests.

### Redis Studio Architecture

```
┌────────────────────────────────────────────────────────────┐
│                    Redis Studio Server                      │
├────────────────────────────────────────────────────────────┤
│                                                             │
│   Frontend (Vanilla JS SPA)      Backend (Go)              │
│   ┌─────────────────┐            ┌─────────────────┐       │
│   │  redis.js       │◄──────────►│  server.go      │       │
│   │  - Key browser  │   HTTP     │  - REST API     │       │
│   │  - Terminal     │            │  - go-redis/v9  │       │
│   │  - Pub/Sub      │            │  - Pipelines    │       │
│   └─────────────────┘            └─────────────────┘       │
│                                                             │
└────────────────────────────────────────────────────────────┘
```

**Performance optimizations:**
- **Pipeline batching**: `GetKeys` batches all `TYPE` + `TTL` commands into 2 Redis round-trips instead of N+1
- **Cursor pagination**: Uses Redis `SCAN` with cursor tracking for key browsing
- **Debounced search**: 300ms debounce on search input to avoid excessive scans

---

## Database Adapters

All databases implement a common `DatabaseAdapter` interface:

```go
type DatabaseAdapter interface {
    Connect(ctx context.Context, url string) error
    Close() error
    Ping(ctx context.Context) error

    // Schema introspection
    GetCurrentSchema(ctx context.Context) ([]SchemaTable, error)
    GetCurrentEnums(ctx context.Context) ([]SchemaEnum, error)
    GetAllTableNames(ctx context.Context) ([]string, error)
    GetTableColumns(ctx context.Context, table string) ([]SchemaColumn, error)

    // Data operations
    GetTableData(ctx context.Context, table string) ([]map[string]any, error)
    ExecuteQuery(ctx context.Context, query string) (*QueryResult, error)
    ExecuteMigration(ctx context.Context, sql string) error

    // DDL generation
    GenerateCreateTableSQL(table SchemaTable) string
    GenerateAddColumnSQL(table string, col SchemaColumn) string
    GenerateDropColumnSQL(table, column string) string
    GenerateAlterColumnSQL(table string, col SchemaColumn, oldType string) string
    GenerateAddIndexSQL(index SchemaIndex) string
    GenerateDropIndexSQL(index SchemaIndex) string

    // Migration tracking
    CreateMigrationsTable(ctx context.Context) error
    GetAppliedMigrations(ctx context.Context) (map[string]*time.Time, error)
    ExecuteAndRecordMigration(ctx context.Context, id, name, checksum, sql string) error
    RemoveMigrationRecord(ctx context.Context, id string) error
}
```

### Adapter Implementations

| Adapter | Driver | Key Characteristics |
|---------|--------|---------------------|
| **PostgreSQL** | `pgx/v5` | Connection pool, enum types, `RETURNING` support |
| **MySQL** | `go-sql-driver/mysql` | `FOREIGN_KEY_CHECKS`, `MODIFY COLUMN` |
| **SQLite** | `mattn/go-sqlite3` | Table recreation for column changes, `PRAGMA` handling |
| **MongoDB** | `mongo-driver` | BSON conversion, `sync.RWMutex` for thread safety |

---

## Plugin System

Flash ORM uses a plugin architecture to keep the core binary small while supporting optional features.

### Plugin Types

| Plugin | Purpose | Auto-install |
|--------|---------|-------------|
| **core** | ORM, migrations, code generation, seeding | Yes (on first ORM command) |
| **studio** | Web-based visual database editor | No (manual install) |

### Plugin Lifecycle

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   flash     │────▶│  ~/.flash/  │────▶│  Plugin     │
│   binary    │     │  plugins/   │     │  binary     │
└─────────────┘     └─────────────┘     └─────────────┘
      │                                    │
      │         ┌──────────────┐          │
      └────────►│ Plugin host  │◄─────────┘
                │ (RPC/gRPC)   │
                └──────────────┘
```

1. The main `flash` binary acts as a CLI host
2. Commands are dispatched to plugin binaries in `~/.flash/plugins/`
3. The `core` plugin handles all ORM operations
4. The `studio` plugin handles web UI operations

### Plugin Commands

```bash
# Install a plugin
flash add-plug studio

# Remove a plugin
flash rm-plug studio

# List installed plugins
flash plugins list

# Update all plugins
flash update

# Update plugins + flash binary
flash update --self
```

---

## End-to-End Flow

### Complete Development Workflow

```
Step 1: Initialize Project
───────────────────────────
$ flash init --postgresql

Creates:
├── flash.toml
├── db/schema/
├── db/queries/
├── db/migrations/
└── .env (with DATABASE_URL)


Step 2: Define Schema
─────────────────────
# db/schema/users.sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


Step 3: Generate Migration
──────────────────────────
$ flash migrate "create users table"

1. Parses db/schema/*.sql
2. Loads .flash/schema_snapshot.json (or queries DB)
3. Diff → detects new table "users"
4. Generates UP/DOWN SQL
5. Writes: db/migrations/20240101120000_create_users_table.sql
6. Updates snapshot


Step 4: Apply Migration
───────────────────────
$ flash apply

1. Loads all migration files
2. Checks _flash_migrations table
3. Finds pending migrations
4. Runs each in a transaction
5. Records checksum + timestamp


Step 5: Write Queries
─────────────────────
# db/queries/users.sql
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (email, status) VALUES ($1, $2) RETURNING id;


Step 6: Generate Code
─────────────────────
$ flash gen

1. Parses schema + queries
2. Generates Go/TS/Python code
3. Writes to flash_gen/


Step 7: Use in Application
──────────────────────────
# Go example
queries := flash_gen.New(db)
user, err := queries.GetUserByEmail(ctx, "john@example.com")


Step 8: Seed Test Data
──────────────────────
$ flash seed --count 50

Generates realistic fake data for all tables with FK relationships handled automatically.


Step 9: Launch Studio (Optional)
─────────────────────────────────
$ flash studio

Opens web UI for:
- Browsing data
- Running SQL queries
- Managing migrations
- Visual schema editing
```

### Data Flow Diagram

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Developer  │────▶│  Schema SQL │────▶│   Parser    │
│             │     │  db/schema/ │     │             │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                              ┌────────────────┼────────────────┐
                              ▼                ▼                ▼
                        ┌─────────┐     ┌──────────┐     ┌──────────┐
                        │Migration│     │ Code Gen │     │  Seeder  │
                        │ Engine  │     │ Pipeline │     │          │
                        └────┬────┘     └────┬─────┘     └────┬─────┘
                             │               │                │
                             ▼               ▼                ▼
                        ┌─────────┐     ┌──────────┐     ┌──────────┐
                        │   DB    │     │ flash_gen│     │ Test Data│
│  flash studio ────────►│  Browser │
└─────────────┘     └─────────────┘     └──────────┘
```

---

## Key Design Decisions

### 1. Snapshot-Based Diffing

**Problem**: Traditional migration tools diff against the live database. If previous migrations haven't been applied, the diff is wrong.

**Solution**: Flash stores a JSON snapshot of the schema state after each migration generation. The next diff compares schema files against this snapshot, not the live DB.

### 2. Adapter Pattern

**Problem**: Supporting 4+ databases with different SQL dialects.

**Solution**: Common `DatabaseAdapter` interface with provider-specific implementations. Each adapter handles DDL generation, type mapping, and connection pooling.

### 3. Incremental Code Generation

**Problem**: Regenerating all code on every change is slow.

**Solution**: SHA256 checksums per query file. Only changed files trigger regeneration. Schema changes trigger full regen.

### 4. Transaction-Per-Migration

**Problem**: Multi-migration failures leave the database in an inconsistent state.

**Solution**: Each migration runs in its own transaction. If it fails, only that migration rolls back. The migration record is inserted in the same transaction as the DDL.

### 5. Raw String Literal Safety

**Problem**: Go generated code uses backtick raw strings for SQL. MySQL backtick identifiers break compilation.

**Solution**: Detect backticks and split the raw string into concatenated segments: `` ` + "`" + ` ``.

### 6. Automatic Seeding

**Problem**: Manually creating test data is tedious and error-prone.

**Solution**: Pattern-based data generation that reads column names and types to produce realistic data. Foreign key relationships are resolved automatically via dependency graph + ID tracking.

### 7. Plugin Architecture

**Problem**: CLI binaries grow large as features are added.

**Solution**: Core binary stays small. Heavy features (studio web UI) are distributed as optional plugins. Auto-install on first use for core functionality.

---

## File Structure Reference

```
flash/
├── cmd/                          # CLI commands
│   ├── studio.go                 # Studio launcher with auto-detection
│   ├── apply.go                  # Migration apply
│   ├── migrate.go                # Migration generation
│   ├── gen.go                    # Code generation
│   ├── seed.go                   # Database seeding
│   └── ...
├── internal/
│   ├── config/                   # Config parsing & validation
│   │   └── config.go
│   ├── schema/                   # Schema parsing & diffing
│   │   ├── schema.go             # Main parser
│   │   ├── snapshot.go           # JSON snapshot I/O
│   │   └── compare.go            # Diff engine
│   ├── migrator/                 # Migration engine
│   │   ├── migrator.go           # Generation logic
│   │   └── operations.go         # Apply/rollback
│   ├── database/                 # Database adapters
│   │   ├── adapter.go            # Common interface
│   │   ├── sqlite/
│   │   ├── postgres/
│   │   ├── mysql/
│   │   └── mongodb/
│   ├── seeder/                   # Database seeding
│   │   ├── seeder.go             # Main seeding logic
│   │   ├── faker.go              # Data generation
│   │   ├── graph.go              # Dependency graph
│   │   └── data.json             # Fake data corpus
│   ├── gogen/                    # Go code generator
│   ├── jsgen/                    # JS/TS code generator
│   ├── pygen/                    # Python code generator
│   ├── gencommon/                # Shared generation logic
│   │   └── cache.go              # Incremental cache
│   ├── parser/                   # SQL parser
│   │   ├── schema.go             # Schema parsing
│   │   └── query.go              # Query parsing
│   ├── studio/                   # Studio implementations
│   │   ├── sql/                  # SQL Studio
│   │   ├── mongodb/              # MongoDB Studio
│   │   └── redis/                # Redis Studio
│   └── utils/                    # Utilities
│       └── utils.go              # Checksum, file utils
└── docs/                         # Documentation
```
