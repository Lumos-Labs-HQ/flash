# How Graft Works

This document explains the internal architecture and workflow of Graft, a database migration CLI tool built in Go.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Core Components](#core-components)
- [Database Adapters](#database-adapters)
- [Migration Workflow](#migration-workflow)
- [Safe Migration System](#safe-migration-system)
- [Schema Management](#schema-management)
- [Export System](#export-system)
- [Configuration System](#configuration-system)
- [Template System](#template-system)

## Architecture Overview

Graft follows a layered architecture with clear separation of concerns:

```
┌─────────────────────────────────────────┐
│              CLI Layer (cmd/)           │
│         Cobra Commands & Flags          │
├─────────────────────────────────────────┤
│           Business Logic Layer          │
│   Migrator, Schema Manager, Export      │
├─────────────────────────────────────────┤
│          Database Adapter Layer         │
│    PostgreSQL, MySQL, SQLite Adapters   │
├─────────────────────────────────────────┤
│            Database Layer               │
│       Actual Database Connections       │
└─────────────────────────────────────────┘
```

## Core Components

### 1. CLI Layer (`cmd/`)

Built using the [Cobra](https://github.com/spf13/cobra) framework, the CLI layer handles:

- **Command Parsing**: Processes user commands and arguments
- **Flag Management**: Handles global and command-specific flags
- **Configuration Loading**: Initializes Viper configuration
- **Error Handling**: Provides user-friendly error messages

**Key Commands:**
- `init`: Project initialization with database-specific templates
- `migrate`: Migration file creation with schema diff
- `apply`: Safe migration execution with transaction rollback
- `status`: Migration status display with detailed information
- `export`: Database export operations (JSON, CSV, SQLite)
- `gen`: SQLC code generation
- `reset`: Database reset with export option
- `pull`: Schema extraction from existing database
- `raw`: Raw SQL execution

### 2. Configuration System (`internal/config/`)

Uses [Viper](https://github.com/spf13/viper) for configuration management:

```go
type Config struct {
    SchemaPath     string   `json:"schema_path"`
    MigrationsPath string   `json:"migrations_path"`
    SqlcConfigPath string   `json:"sqlc_config_path"`
    ExportPath     string   `json:"export_path"`
    Database       Database `json:"database"`
}
```

**Features:**
- JSON-based configuration files
- Environment variable support
- Default value fallbacks
- Configuration validation
- Directory auto-creation

### 3. Database Adapter Pattern (`internal/database/`)

The adapter pattern enables multi-database support through a common interface:

```go
type DatabaseAdapter interface {
    Connect(ctx context.Context, url string) error
    Close() error
    Ping(ctx context.Context) error
    
    // Migration operations
    CreateMigrationsTable(ctx context.Context) error
    GetAppliedMigrations(ctx context.Context) (map[string]*time.Time, error)
    RecordMigration(ctx context.Context, migrationID, name, checksum string) error
    ExecuteMigration(ctx context.Context, migrationSQL string) error
    
    // Schema operations
    GetCurrentSchema(ctx context.Context) ([]types.SchemaTable, error)
    GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error)
    
    // Conflict detection
    CheckTableExists(ctx context.Context, tableName string) (bool, error)
    CheckColumnExists(ctx context.Context, tableName, columnName string) (bool, error)
    
    // Export operations
    GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error)
    GetAllTableNames(ctx context.Context) ([]string, error)
}
```

### 4. Migration System (`internal/migrator/`)

The migrator orchestrates the entire migration process with safe execution:

**Core Responsibilities:**
- Migration file management
- Conflict detection and resolution
- Schema validation
- Export coordination
- Transaction management with rollback

**Migration File Structure:**
```sql
-- Migration: create_users_table
-- Created: 2025-10-21T13:29:02Z

CREATE TABLE IF NOT EXISTS "users" (
  "id" SERIAL PRIMARY KEY,
  "name" VARCHAR(255) NOT NULL,
  "email" VARCHAR(255) UNIQUE NOT NULL,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

### 5. Schema Management (`internal/schema/`)

Handles schema parsing, validation, and diff generation:

**Schema Parsing Process:**
1. **File Reading**: Reads SQL schema files
2. **SQL Cleaning**: Removes comments and normalizes whitespace
3. **Statement Splitting**: Separates individual SQL statements
4. **Table Extraction**: Parses CREATE TABLE statements
5. **Relationship Mapping**: Identifies foreign key relationships

**Schema Diff Generation:**
```go
type SchemaDiff struct {
    NewTables      []SchemaTable
    DroppedTables  []string
    ModifiedTables []TableDiff
    NewIndexes     []SchemaIndex
    DroppedIndexes []string
}
```

### 6. Export System (`internal/export/`)

Multi-format export system with the following features:

**Export Formats:**
- **JSON**: Structured data with metadata
- **CSV**: Individual files per table
- **SQLite**: Portable database file

**Export Structure (JSON):**
```json
{
  "timestamp": "2025-10-21 14:00:07",
  "version": "1.0",
  "comment": "Database export",
  "tables": {
    "users": [
      {"id": 1, "name": "John", "email": "john@example.com"}
    ]
  }
}
```

**Export Triggers:**
- Manual export commands (`graft export`)
- Before destructive operations
- Schema conflict resolution
- Database reset operations

## Database Adapters

### PostgreSQL Adapter (`postgres.go`)

**Technology Stack:**
- **Driver**: [pgx/v5](https://github.com/jackc/pgx) - High-performance PostgreSQL driver
- **Connection Pool**: pgxpool for connection management
- **Query Builder**: [Squirrel](https://github.com/Masterminds/squirrel) for dynamic SQL

**Key Features:**
- Connection pooling optimized for Supabase/PgBouncer
- Advanced PostgreSQL type support
- JSONB and UUID support
- Transaction management with rollback
- Exec mode for pooler compatibility

**Type Mapping:**
```go
var pgTypeMap = map[string]string{
    "character varying": "VARCHAR",
    "timestamp with time zone": "TIMESTAMP WITH TIME ZONE",
    "jsonb": "JSONB",
    "uuid": "UUID",
    // ... more mappings
}
```

### MySQL Adapter (`mysql.go`)

**Technology Stack:**
- **Driver**: [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)
- **Query Builder**: Squirrel with question mark placeholders
- **Connection**: Standard database/sql interface

**Key Features:**
- MySQL-specific SQL generation
- AUTO_INCREMENT handling
- MySQL type system support
- InnoDB engine optimization

### SQLite Adapter (`sqlite.go`)

**Technology Stack:**
- **Driver**: [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)
- **File-based**: Single file database
- **Lightweight**: Minimal resource usage

**Key Features:**
- File-based database management
- SQLite-specific constraints
- Embedded database support
- Cross-platform compatibility

## Migration Workflow

### 1. Migration Creation (`graft migrate`)

```mermaid
graph TD
    A[User runs 'graft migrate'] --> B[Load Configuration]
    B --> C[Generate Migration ID]
    C --> D[Parse Current Schema]
    D --> E[Generate Schema Diff]
    E --> F[Create SQL Statements]
    F --> G[Write Migration File]
```

### 2. Safe Migration Application (`graft apply`)

```mermaid
graph TD
    A[User runs 'graft apply'] --> B[Load Configuration]
    B --> C[Connect to Database]
    C --> D[Create Migration Table]
    D --> E[Get Applied Migrations]
    E --> F[Load Pending Migrations]
    F --> G[Detect Conflicts]
    G --> H{Conflicts Found?}
    H -->|Yes| I[Prompt for Export]
    H -->|No| J[Apply Migrations Safely]
    I --> K[Create Export]
    K --> L[Reset & Apply All]
    J --> M[Transaction per Migration]
    L --> M
    M --> N[Update Migration Table]
```

## Safe Migration System

### Transaction-Based Execution

Each migration runs in its own transaction with automatic rollback on failure:

```go
func (p *PostgresAdapter) ExecuteMigration(ctx context.Context, migrationSQL string) error {
    tx, err := p.pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback(ctx) // Auto-rollback on error

    statements := p.parseSQLStatements(migrationSQL)
    
    for _, stmt := range statements {
        if _, err := tx.Exec(ctx, stmt); err != nil {
            return fmt.Errorf("failed to execute statement: %w", err)
        }
    }
    
    return tx.Commit(ctx)
}
```

### Migration Safety Features

**Corruption Prevention:**
- Each migration in separate transaction
- Automatic rollback on any failure
- Migration state tracking
- Broken migration cleanup

**Error Recovery:**
```go
func (m *Migrator) applySingleMigrationSafely(ctx context.Context, migration types.Migration) error {
    if err := m.adapter.ExecuteMigration(ctx, content); err != nil {
        fmt.Printf("❌ Failed at migration: %s\n", migration.ID)
        fmt.Printf("   Error: %v\n", err)
        fmt.Println("   Transaction rolled back. Fix the error and run 'graft apply' again.")
        return err
    }
    
    return m.adapter.RecordMigration(ctx, migration.ID, migration.Name, checksum)
}
```

### Conflict Detection and Resolution

**Conflict Types:**
- Table already exists
- Column conflicts
- Constraint violations
- Data type mismatches

**Resolution Strategies:**
- Automatic export creation
- Interactive conflict resolution
- Database reset with full migration replay
- Manual intervention options

## Schema Management

### SQL Parsing Engine

Graft includes a custom SQL parser that handles:

**Supported SQL Constructs:**
- CREATE TABLE statements with all constraints
- Column definitions with data types
- Primary key and foreign key relationships
- Index definitions
- Complex constraint declarations

**Parsing Process:**
1. **Tokenization**: Breaks SQL into meaningful tokens
2. **Statement Recognition**: Identifies CREATE TABLE statements
3. **Column Extraction**: Parses column definitions and constraints
4. **Relationship Analysis**: Maps foreign key relationships
5. **Validation**: Ensures SQL syntax correctness

### Schema Diff Algorithm

```go
func (sm *SchemaManager) GenerateSchemaDiff(ctx context.Context, targetSchemaPath string) (*types.SchemaDiff, error) {
    // 1. Parse target schema from file
    targetTables, err := sm.ParseSchemaFile(targetSchemaPath)
    
    // 2. Get current database schema
    currentTables, err := sm.adapter.GetCurrentSchema(ctx)
    
    // 3. Compare schemas and generate diff
    return sm.compareSchemas(currentTables, targetTables), nil
}
```

## Export System

### Export Creation Process

1. **Table Discovery**: Queries database for all tables
2. **Data Extraction**: Retrieves data from each table (excluding `_graft_migrations`)
3. **Format Conversion**: Converts to requested format (JSON/CSV/SQLite)
4. **File Writing**: Saves export to timestamped file
5. **Verification**: Validates export integrity

### Export Formats

**JSON Export:**
```json
{
  "timestamp": "2025-10-21 14:00:07",
  "version": "1.0",
  "comment": "Database export",
  "tables": {
    "users": [
      {"id": 1, "name": "Alice", "email": "alice@example.com"}
    ],
    "posts": [
      {"id": 1, "user_id": 1, "title": "Hello World", "content": "..."}
    ]
  }
}
```

**CSV Export:**
- Creates directory with timestamp
- Individual CSV file per table
- Headers with column names
- Proper CSV escaping

**SQLite Export:**
- Creates portable .db file
- Preserves table structure
- Maintains data relationships
- Cross-platform compatible

### Export Commands

```bash
# Export as JSON (default)
graft export

# Export as CSV
graft export --csv

# Export as SQLite
graft export --sqlite
```

## Configuration System

### Configuration Loading Priority

1. Command-line flags (`--config`)
2. Environment variables
3. Local config file (`./graft.config.json`)
4. Default values

### Environment Variable Support

```bash
# Database connection
export DATABASE_URL="postgres://user:pass@localhost:5432/db"

# Override config paths
export GRAFT_MIGRATIONS_PATH="custom/migrations"
export GRAFT_SCHEMA_PATH="custom/schema.sql"
```

### Configuration Structure

```json
{
  "schema_path": "db/schema/schema.sql",
  "migrations_path": "db/migrations",
  "sqlc_config_path": "sqlc.yml",
  "export_path": "db/export",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  }
}
```

## Template System

### Project Initialization Templates

The template system generates database-specific configurations:

```go
type ProjectTemplate struct {
    DatabaseType DatabaseType
}

func (pt *ProjectTemplate) GetGraftConfig() string {
    return fmt.Sprintf(`{
  "schema_path": "db/schema/schema.sql",
  "migrations_path": "db/migrations",
  "sqlc_config_path": "sqlc.yml",
  "export_path": "db/export",
  "database": {
    "provider": "%s",
    "url_env": "DATABASE_URL"
  }
}`, pt.DatabaseType)
}
```

### SQLC Integration Templates

Generates SQLC configuration for each database type:

**PostgreSQL SQLC Config:**
```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries/"
    schema: "db/schema/"
    gen:
      go:
        package: "graft"
        out: "graft_gen/"
```

## Error Handling and Logging

### Error Propagation

Graft uses Go's standard error handling with context:

```go
func (m *Migrator) Apply(ctx context.Context, migrationName, schemaPath string) error {
    if err := m.validateMigrations(ctx); err != nil {
        return fmt.Errorf("migration validation failed: %w", err)
    }
    
    if err := m.applyMigrations(ctx); err != nil {
        return fmt.Errorf("failed to apply migrations: %w", err)
    }
    
    return nil
}
```

### User-Friendly Messages

All errors include contextual information and recovery suggestions:

```go
fmt.Printf("❌ Failed at migration: %s\n", migration.ID)
fmt.Printf("   Error: %v\n", err)
fmt.Println("   Transaction rolled back. Fix the error and run 'graft apply' again.")
```

## Performance Considerations

### Connection Pooling

- PostgreSQL uses pgxpool with optimized settings for poolers
- MySQL and SQLite use standard database/sql with connection limits
- Configurable pool sizes and timeouts
- Supabase/PgBouncer compatibility

### Query Optimization

- Prepared statements for repeated queries
- Batch operations for bulk data
- Index-aware query generation
- Transaction batching for migrations
- Streaming for large exports

### Memory Management

- Streaming for large table data
- Chunked export operations
- Efficient JSON marshaling
- Resource cleanup with defer statements

This architecture ensures Graft is scalable, maintainable, and extensible while providing a robust and safe migration experience across multiple database systems.
