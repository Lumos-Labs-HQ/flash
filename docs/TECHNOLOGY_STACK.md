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


## Database Drivers

### PostgreSQL - pgx/v5

**Repository**: https://github.com/jackc/pgx  
**Version**: v5.4.3

**Features:**
- High-performance PostgreSQL driver
- Connection pooling with pgxpool
- Native PostgreSQL protocol implementation
- Support for PostgreSQL-specific types (JSONB, UUID, arrays)
- Prepared statement caching
- Copy protocol support
- Context-aware operations

**Usage in Graft:**
```go
type PostgresAdapter struct {
    pool *pgxpool.Pool
    qb   squirrel.StatementBuilderType
}

func (p *PostgresAdapter) Connect(ctx context.Context, url string) error {
    pool, err := pgxpool.New(ctx, url)
    if err != nil {
        return fmt.Errorf("failed to create connection pool: %w", err)
    }
    p.pool = pool
    return nil
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
- Backup and restore APIs

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
        // Migration logic
    },
}
```

**Commands Implemented:**
- `graft init` - Project initialization
- `graft migrate` - Create migrations
- `graft apply` - Apply migrations
- `graft status` - Show migration status
- `graft backup` - Create backups
- `graft restore` - Restore from backup
- `graft reset` - Reset database
- `graft gen` - Generate SQLC code
- `graft pull` - Extract schema
- `graft raw` - Execute raw SQL

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

### File System Operations

**afero** (via Viper dependency):
- File system abstraction
- Memory-based file systems for testing
- OS file system operations
- Path utilities

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

## Architecture Patterns

### Adapter Pattern

Used for database abstraction:
```go
type DatabaseAdapter interface {
    Connect(ctx context.Context, url string) error
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

Used for project initialization:
```go
type ProjectTemplate struct {
    DatabaseType DatabaseType
}

func (pt *ProjectTemplate) GetGraftConfig() string {
    // Database-specific configuration generation
}
```

### Strategy Pattern

Used for conflict resolution:
```go
type ConflictResolver interface {
    Resolve(conflict types.MigrationConflict) error
}
```

## Performance Optimizations

### Connection Pooling

- **PostgreSQL**: pgxpool with configurable pool size
- **MySQL/SQLite**: database/sql with connection limits
- Connection reuse and lifecycle management

### Query Optimization

- Prepared statement caching
- Batch operations for bulk data
- Index-aware query generation
- Transaction batching

### Memory Management

- Streaming for large datasets
- Chunked operations
- Resource cleanup with defer
- Efficient JSON marshaling

## Security Considerations

### Database Connections

- TLS/SSL support for all databases
- Connection string validation
- Environment variable isolation
- No hardcoded credentials

### File Operations

- Path validation and sanitization
- Permission checking
- Atomic file operations
- Backup integrity verification

### SQL Injection Prevention

- Parameterized queries only
- Input validation
- Query builder usage
- No dynamic SQL construction

## Testing Strategy

### Unit Testing

- Database adapter testing with mocks
- Configuration validation testing
- Schema parsing testing
- Migration logic testing

### Integration Testing

- Real database connections
- End-to-end workflow testing
- Cross-platform testing
- Performance benchmarking

### Test Dependencies

```go
// Test-specific dependencies would include:
// - testify for assertions
// - dockertest for database containers
// - gomock for mocking
```

This comprehensive technology stack ensures Graft is robust, performant, and maintainable while supporting multiple database systems and providing an excellent developer experience.
