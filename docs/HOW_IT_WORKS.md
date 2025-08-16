# How Graft Works

This document explains the internal architecture and workflow of Graft, a database migration CLI tool built in Go.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Core Components](#core-components)
- [Database Adapters](#database-adapters)
- [Migration Workflow](#migration-workflow)
- [Schema Management](#schema-management)
- [Backup System](#backup-system)
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
│    Migrator, Schema Manager, Backup     │
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
- `init`: Project initialization
- `migrate`: Migration file creation
- `apply`: Migration execution
- `status`: Migration status display
- `backup`/`restore`: Backup operations
- `gen`: SQLC code generation
- `reset`: Database reset
- `pull`: Schema extraction
- `raw`: Raw SQL execution

### 2. Configuration System (`internal/config/`)

Uses [Viper](https://github.com/spf13/viper) for configuration management:

```go
type Config struct {
    SchemaPath     string   `json:"schema_path"`
    MigrationsPath string   `json:"migrations_path"`
    SqlcConfigPath string   `json:"sqlc_config_path"`
    BackupPath     string   `json:"backup_path"`
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
    
    // Backup operations
    GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error)
}
```

### 4. Migration System (`internal/migrator/`)

The migrator orchestrates the entire migration process:

**Core Responsibilities:**
- Migration file management
- Conflict detection and resolution
- Schema validation
- Backup coordination
- Transaction management

**Migration File Structure:**
```sql
-- Migration: 20240816055529_create_users_table
-- Created: 2024-08-16T05:55:29Z

-- Up Migration
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Down Migration
DROP TABLE IF EXISTS users CASCADE;
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

### 6. Backup System (`internal/backup/`)

JSON-based backup system with the following features:

**Backup Structure:**
```json
{
  "timestamp": "2024-08-16_05-55-29",
  "version": "5_migrations",
  "comment": "Pre-migration backup",
  "tables": {
    "users": {
      "columns": ["id", "name", "email", "created_at"],
      "data": [
        {"id": 1, "name": "John", "email": "john@example.com"}
      ]
    }
  }
}
```

**Backup Triggers:**
- Before destructive operations
- Manual backup commands
- Schema conflict resolution
- Database reset operations

## Database Adapters

### PostgreSQL Adapter (`postgres.go`)

**Technology Stack:**
- **Driver**: [pgx/v5](https://github.com/jackc/pgx) - High-performance PostgreSQL driver
- **Connection Pool**: pgxpool for connection management
- **Query Builder**: [Squirrel](https://github.com/Masterminds/squirrel) for dynamic SQL

**Key Features:**
- Connection pooling for performance
- Advanced PostgreSQL type support
- JSONB and UUID support
- Full-text search capabilities
- Transaction management

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
    C --> D[Create Migration File]
    D --> E[Parse Current Schema]
    E --> F[Generate Schema Diff]
    F --> G[Create SQL Statements]
    G --> H[Write Migration File]
```

### 2. Migration Application (`graft apply`)

```mermaid
graph TD
    A[User runs 'graft apply'] --> B[Load Configuration]
    B --> C[Connect to Database]
    C --> D[Create Migration Table]
    D --> E[Get Applied Migrations]
    E --> F[Load Pending Migrations]
    F --> G[Detect Conflicts]
    G --> H{Conflicts Found?}
    H -->|Yes| I[Prompt for Backup]
    H -->|No| J[Apply Migrations]
    I --> K[Create Backup]
    K --> J
    J --> L[Update Migration Table]
```

### 3. Conflict Detection

Graft detects various types of conflicts:

**Table Conflicts:**
- Table already exists
- Table being dropped with data
- Column type changes
- Constraint violations

**Resolution Strategies:**
- Automatic backup creation
- User confirmation prompts
- Rollback capabilities
- Manual intervention options

## Schema Management

### SQL Parsing Engine

Graft includes a custom SQL parser that handles:

**Supported SQL Constructs:**
- CREATE TABLE statements
- Column definitions with constraints
- Primary key declarations
- Foreign key relationships
- Index definitions
- Data type specifications

**Parsing Process:**
1. **Tokenization**: Breaks SQL into tokens
2. **Statement Recognition**: Identifies CREATE TABLE statements
3. **Column Extraction**: Parses column definitions
4. **Constraint Analysis**: Identifies constraints and relationships
5. **Validation**: Ensures SQL correctness

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

## Backup System

### Backup Creation Process

1. **Data Extraction**: Queries all tables for data
2. **Metadata Collection**: Gathers table structure information
3. **JSON Serialization**: Converts data to JSON format
4. **File Writing**: Saves backup to timestamped file
5. **Verification**: Validates backup integrity

### Restore Process

1. **Backup Validation**: Verifies backup file integrity
2. **Schema Recreation**: Recreates table structures
3. **Data Insertion**: Restores all table data
4. **Migration Sync**: Updates migration tracking
5. **Verification**: Confirms successful restore

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
  "backup_path": "db/backup",
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
        sql_package: "pgx/v5"
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

All errors are wrapped with contextual information:

```go
return fmt.Errorf("failed to connect to %s database: %w", cfg.Database.Provider, err)
```

## Performance Considerations

### Connection Pooling

- PostgreSQL uses pgxpool for efficient connection management
- MySQL and SQLite use standard database/sql with connection limits
- Configurable pool sizes and timeouts

### Query Optimization

- Prepared statements for repeated queries
- Batch operations for bulk data
- Index-aware query generation
- Transaction batching for migrations

### Memory Management

- Streaming for large table data
- Chunked backup operations
- Efficient JSON marshaling
- Resource cleanup with defer statements

This architecture ensures Graft is scalable, maintainable, and extensible while providing a robust migration experience across multiple database systems.
