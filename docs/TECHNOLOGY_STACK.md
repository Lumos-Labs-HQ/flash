# Technology Stack & Dependencies

This document outlines all the technologies, libraries, and tools used in the Graft project.

## Table of Contents

- [Core Technologies](#core-technologies)
- [Go Dependencies](#go-dependencies)
- [Database Drivers](#database-drivers)
- [CLI Framework](#cli-framework)
- [Configuration Management](#configuration-management)
- [Build Tools](#build-tools)
- [Development Tools](#development-tools)
- [External Integrations](#external-integrations)

## Core Technologies

### Go Programming Language

**Version**: Go 1.24.2+

**Why Go?**
- **Performance**: Compiled language with excellent performance characteristics
- **Concurrency**: Built-in goroutines and channels for concurrent operations
- **Cross-platform**: Single binary deployment across Linux, Windows, and macOS
- **Standard Library**: Rich standard library with excellent database support
- **Static Typing**: Type safety and compile-time error detection
- **Memory Management**: Garbage collection with low latency
- **Ecosystem**: Mature ecosystem with excellent database and CLI libraries

**Go Features Used:**
- Context package for cancellation and timeouts
- Interfaces for database adapter pattern
- Goroutines for concurrent operations
- Channels for communication
- Error handling with wrapped errors
- Reflection for configuration unmarshaling
- Transaction management for safe migrations

## Database Drivers

### PostgreSQL - pgx/v5

**Repository**: https://github.com/jackc/pgx  
**Version**: v5.4.3

**Features:**
- High-performance PostgreSQL driver
- Connection pooling with pgxpool optimized for Supabase/PgBouncer
- Native PostgreSQL protocol implementation
- Support for PostgreSQL-specific types (JSONB, UUID, arrays)
- Prepared statement caching
- Copy protocol support
- Context-aware operations
- Transaction safety with automatic rollback

**Usage in Graft:**
```go
type PostgresAdapter struct {
    pool *pgxpool.Pool
    qb   squirrel.StatementBuilderType
}

func (p *PostgresAdapter) Connect(ctx context.Context, url string) error {
    config, err := pgxpool.ParseConfig(url)
    if err != nil {
        return fmt.Errorf("failed to parse connection URL: %w", err)
    }
    
    // Use exec mode for pooler compatibility (Supabase, PgBouncer)
    config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
    
    pool, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        return fmt.Errorf("failed to create connection pool: %w", err)
    }
    p.pool = pool
    return nil
}

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

### MySQL - go-sql-driver/mysql

**Repository**: https://github.com/go-sql-driver/mysql  
**Version**: v1.9.3

**Features:**
- Pure Go MySQL driver
- Full MySQL protocol support
- TLS/SSL support
- Connection pooling
- Prepared statements
- Multiple result sets
- Custom data types
- Transaction safety

**Usage in Graft:**
```go
type MySQLAdapter struct {
    db *sql.DB
    qb squirrel.StatementBuilderType
}

func (m *MySQLAdapter) Connect(ctx context.Context, url string) error {
    db, err := sql.Open("mysql", url)
    if err != nil {
        return fmt.Errorf("failed to open MySQL connection: %w", err)
    }
    m.db = db
    return nil
}

func (m *MySQLAdapter) ExecuteMigration(ctx context.Context, migrationSQL string) error {
    tx, err := m.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback() // Auto-rollback on error
    
    // Execute statements safely
    // ...
    
    return tx.Commit()
}
```

### SQLite - mattn/go-sqlite3

**Repository**: https://github.com/mattn/go-sqlite3  
**Version**: v1.14.32

**Features:**
- CGO-based SQLite3 binding
- Full SQLite feature support
- File-based database
- In-memory database support
- Custom functions and aggregates
- Export/import APIs
- Transaction safety

**Usage in Graft:**
```go
type SQLiteAdapter struct {
    db *sql.DB
    qb squirrel.StatementBuilderType
}

func (s *SQLiteAdapter) Connect(ctx context.Context, url string) error {
    db, err := sql.Open("sqlite3", url)
    if err != nil {
        return fmt.Errorf("failed to open SQLite connection: %w", err)
    }
    s.db = db
    return nil
}
```

## CLI Framework

### Cobra

**Repository**: https://github.com/spf13/cobra  
**Version**: v1.8.0

**Features:**
- Powerful CLI framework
- Subcommand support
- Flag parsing and validation
- Auto-generated help
- Shell completion
- Man page generation
- POSIX-compliant flags

**Usage in Graft:**
```go
var rootCmd = &cobra.Command{
    Use:   "graft",
    Short: "A database migration CLI tool",
    Long:  `Graft is a Go-based CLI tool...`,
}

var migrateCmd = &cobra.Command{
    Use:   "migrate [name]",
    Short: "Create a new migration",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Migration logic with safe execution
    },
}
```

**Commands Implemented:**
- `graft init` - Project initialization with database templates
- `graft migrate` - Create migrations with schema diff
- `graft apply` - Apply migrations with transaction safety
- `graft status` - Show migration status with detailed info
- `graft export` - Export database (JSON, CSV, SQLite)
- `graft reset` - Reset database with export option
- `graft gen` - Generate SQLC code
- `graft pull` - Extract schema from database
- `graft raw` - Execute raw SQL files

### Color Output - fatih/color

**Repository**: https://github.com/fatih/color  
**Version**: v1.18.0

**Features:**
- Cross-platform colored terminal output
- Multiple color and style options
- Windows support
- Performance optimized

**Usage in Graft:**
```go
func showBanner() {
    greenColor := color.New(color.FgGreen, color.Bold)
    greenColor.Println("✅ Migration applied successfully")
    
    color.New(color.FgRed, color.Bold).Printf("❌ Failed at migration: %s\n", migration.ID)
}
```

## Configuration Management

### Viper

**Repository**: https://github.com/spf13/viper  
**Version**: v1.18.2

**Features:**
- Configuration management
- Multiple format support (JSON, YAML, TOML, HCL)
- Environment variable binding
- Remote configuration support
- Configuration watching
- Default value handling

**Usage in Graft:**
```go
func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        viper.AddConfigPath(".")
        viper.SetConfigType("json")
        viper.SetConfigName("graft.config")
    }
    
    viper.AutomaticEnv()
    viper.ReadInConfig()
}

type Config struct {
    SchemaPath     string   `json:"schema_path"`
    MigrationsPath string   `json:"migrations_path"`
    SqlcConfigPath string   `json:"sqlc_config_path"`
    ExportPath     string   `json:"export_path"`
    Database       Database `json:"database"`
}
```

### godotenv

**Repository**: https://github.com/joho/godotenv  
**Version**: v1.5.1

**Features:**
- Load environment variables from .env files
- Multiple .env file support
- Variable expansion
- Override protection

**Usage in Graft:**
```go
func initConfig() {
    if err := godotenv.Load(); err != nil {
        godotenv.Load(".env")
        godotenv.Load(".env.local")
    }
}
```

## Build Tools

### Makefile

**Features:**
- Cross-platform compilation
- Automated builds for Linux, Windows, macOS
- Development workflow automation
- Dependency management
- Testing and linting integration

**Key Targets:**
```makefile
build-all:    # Build for all platforms
install:      # Install to GOPATH/bin
clean:        # Clean build artifacts
test:         # Run tests
deps:         # Download dependencies
fmt:          # Format code
lint:         # Lint code
release:      # Create release build
```

### Go Modules

**go.mod Features:**
- Dependency versioning
- Semantic versioning support
- Module proxy support
- Vendor directory support
- Replace directives for local development

## Development Tools

### Query Builder - Squirrel

**Repository**: https://github.com/Masterminds/squirrel  
**Version**: v1.5.4

**Features:**
- Fluent SQL query builder
- Multiple database support
- Placeholder formatting
- Query caching
- Type-safe query construction

**Usage in Graft:**
```go
// PostgreSQL with dollar placeholders
p.qb = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

// MySQL with question mark placeholders
m.qb = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)

// Build queries
query := p.qb.Select("*").From("users").Where(squirrel.Eq{"id": userID})
```

## External Integrations

### SQLC Integration

**SQLC**: https://sqlc.dev/

**Features:**
- Generate Go code from SQL
- Type-safe database queries
- Multiple database support
- Query validation
- Performance optimization

**Integration in Graft:**
```go
func runSQLCGenerate(configPath string) error {
    if _, err := exec.LookPath("sqlc"); err != nil {
        return fmt.Errorf("sqlc not found in PATH. Please install SQLC: https://docs.sqlc.dev/en/latest/overview/install.html")
    }

    cmd := exec.Command("sqlc", "generate", "-f", configPath)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

**Generated SQLC Config:**
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

## Export System

### JSON Export

**Features:**
- Structured data export with metadata
- Timestamp and version tracking
- Table-wise data organization
- Metadata preservation

**Export Structure:**
```json
{
  "timestamp": "2025-10-21 14:00:07",
  "version": "1.0",
  "comment": "Database export",
  "tables": {
    "users": [
      {"id": 1, "name": "Alice", "email": "alice@example.com"}
    ]
  }
}
```

### CSV Export

**Features:**
- Individual CSV files per table
- Proper CSV escaping and formatting
- Header row with column names
- Directory-based organization

### SQLite Export

**Features:**
- Portable database file creation
- Schema and data preservation
- Cross-platform compatibility
- Relationship maintenance

**Implementation:**
```go
func exportToSQLite(ctx context.Context, adapter database.DatabaseAdapter, data types.BackupData, exportPath string) (string, error) {
    timestamp := time.Now().Format("2006-01-02_15-04-05")
    filePath := filepath.Join(exportPath, fmt.Sprintf("export_%s.db", timestamp))
    
    sqliteDB, err := sql.Open("sqlite3", filePath)
    if err != nil {
        return "", err
    }
    defer sqliteDB.Close()
    
    // Create tables and insert data
    // ...
    
    return filePath, nil
}
```

## Architecture Patterns

### Adapter Pattern

Used for database abstraction with safe migration support:
```go
type DatabaseAdapter interface {
    Connect(ctx context.Context, url string) error
    ExecuteMigration(ctx context.Context, migrationSQL string) error // Transaction-safe
    RecordMigration(ctx context.Context, migrationID, name, checksum string) error
    GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error)
    // ... other methods
}

func NewAdapter(provider string) DatabaseAdapter {
    switch provider {
    case "postgresql", "postgres":
        return NewPostgresAdapter()
    case "mysql":
        return NewMySQLAdapter()
    case "sqlite", "sqlite3":
        return NewSQLiteAdapter()
    }
}
```

### Template Method Pattern

Used for project initialization with database-specific templates:
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

### Strategy Pattern

Used for export format selection:
```go
func PerformExport(ctx context.Context, adapter database.DatabaseAdapter, exportPath, format string) (string, error) {
    switch format {
    case "csv":
        return exportToCSV(exportData, exportPath)
    case "sqlite":
        return exportToSQLite(ctx, adapter, exportData, exportPath)
    default:
        return exportToJSON(exportData, exportPath)
    }
}
```

## Safe Migration System

### Transaction Management

**Features:**
- Each migration runs in its own transaction
- Automatic rollback on any failure
- Migration state tracking
- Broken migration cleanup

**Implementation:**
```go
func (m *Migrator) applySingleMigrationSafely(ctx context.Context, migration types.Migration) error {
    content, err := os.ReadFile(migration.FilePath)
    if err != nil {
        return fmt.Errorf("failed to read migration file: %w", err)
    }
    
    if err := m.adapter.ExecuteMigration(ctx, string(content)); err != nil {
        fmt.Printf("❌ Failed at migration: %s\n", migration.ID)
        fmt.Printf("   Error: %v\n", err)
        fmt.Println("   Transaction rolled back. Fix the error and run 'graft apply' again.")
        return err
    }

    checksum := fmt.Sprintf("%x", len(content))
    return m.adapter.RecordMigration(ctx, migration.ID, migration.Name, checksum)
}
```

### Conflict Resolution

**Features:**
- Automatic conflict detection
- Interactive resolution prompts
- Export creation before destructive operations
- Database reset with full migration replay

## Performance Optimizations

### Connection Pooling

- **PostgreSQL**: pgxpool with Supabase/PgBouncer optimization
- **MySQL/SQLite**: database/sql with connection limits
- Connection reuse and lifecycle management
- Configurable pool sizes and timeouts

### Query Optimization

- Prepared statement caching
- Batch operations for bulk data
- Index-aware query generation
- Transaction batching for migrations
- Streaming for large exports

### Memory Management

- Streaming for large datasets
- Chunked export operations
- Resource cleanup with defer
- Efficient JSON marshaling
- Connection pool optimization

## Security Considerations

### Database Connections

- TLS/SSL support for all databases
- Connection string validation
- Environment variable isolation
- No hardcoded credentials
- Secure connection pooling

### File Operations

- Path validation and sanitization
- Permission checking
- Atomic file operations
- Export integrity verification
- Secure temporary file handling

### SQL Injection Prevention

- Parameterized queries only
- Input validation and sanitization
- Query builder usage
- No dynamic SQL construction
- Transaction isolation

## Testing Strategy

### Unit Testing

- Database adapter testing with mocks
- Configuration validation testing
- Schema parsing testing
- Migration logic testing
- Export functionality testing
- Safe migration testing

### Integration Testing

- Real database connections
- End-to-end workflow testing
- Cross-platform testing
- Performance benchmarking
- Export/import roundtrip testing

### Test Dependencies

```go
// Test-specific dependencies include:
// - testify for assertions
// - dockertest for database containers
// - gomock for mocking
// - Custom test helpers for database setup
```

## Version Information

**Current Version**: v1.6.0

**Version History:**
- v1.6.0: Added export system and safe migration features
- v1.5.0: Enhanced schema management and conflict detection
- Previous versions: Core migration functionality

This comprehensive technology stack ensures Graft is robust, performant, and maintainable while supporting multiple database systems, providing safe migration execution, and offering flexible export capabilities for an excellent developer experience.
