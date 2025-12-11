---
title: Technology Stack
description: Technical implementation details and dependencies
---

# Technology Stack & Implementation Details

Technical implementation details, dependencies, and architecture patterns used in FlashORM.

## Table of Contents

- [Core Stack](#core-stack)
- [Database Drivers & Connection Management](#database-drivers--connection-management)
- [CLI & Configuration](#cli--configuration)
- [Code Generation](#code-generation)
- [Studio Technologies](#studio-technologies)
- [Build & Distribution](#build--distribution)
- [Performance & Security](#performance--security)

## Core Stack

### Go 1.24.2

**Key Features Used:**
- `context` - Cancellation and timeouts
- `embed` - Static file embedding (`//go:embed`)
- Interfaces - Database adapter pattern
- Goroutines - Concurrent operations
- Error wrapping - `fmt.Errorf("%w", err)`

### Primary Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `spf13/cobra` | v1.10.1 | CLI framework |
| `spf13/viper` | v1.21.0 | Configuration management |
| `jackc/pgx/v5` | v5.7.6 | PostgreSQL driver |
| `go-sql-driver/mysql` | v1.9.3 | MySQL driver |
| `mattn/go-sqlite3` | v1.14.32 | SQLite driver |
| `Masterminds/squirrel` | v1.5.4 | Query builder |
| `gofiber/fiber/v2` | v2.52.9 | Web framework (Studio) |
| `fatih/color` | v1.18.0 | Terminal colors |
| `joho/godotenv` | v1.5.1 | Environment variables |

## Database Drivers & Connection Management

### PostgreSQL - pgx/v5

**Connection Pool Configuration:**
```go
config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec // Supabase/PgBouncer compatibility
config.MaxConns = 2
config.MinConns = 0
config.MaxConnLifetime = 15 * time.Minute
config.MaxConnIdleTime = 3 * time.Minute
```

**Type Mapping:**
```go
var typeMap = map[string]string{
    "character varying": "VARCHAR",
    "timestamp with time zone": "TIMESTAMP WITH TIME ZONE",
    "jsonb": "JSONB",
    "uuid": "UUID",
    // ... 20+ mappings
}
```

**Dependencies:**
- `jackc/pgpassfile` v1.0.0
- `jackc/pgservicefile` v0.0.0-20240606120523
- `jackc/puddle/v2` v2.2.2
- `lib/pq` v1.10.9 (legacy support)

### MySQL - go-sql-driver/mysql

**Connection Configuration:**
```go
db.SetMaxOpenConns(2)
db.SetMaxIdleConns(0)
db.SetConnMaxLifetime(15 * time.Minute)
db.SetConnMaxIdleTime(3 * time.Minute)
```

**URL Parsing:**
```go
// Converts: mysql://user:pass@host:port/db?ssl-mode=REQUIRED
// To: user:pass@tcp(host:port)/db?tls=skip-verify
```

**Dependencies:**
- `filippo.io/edwards25519` v1.1.0

### SQLite - mattn/go-sqlite3

**Connection Configuration:**
```go
db.SetMaxOpenConns(1)  // Single connection for file-based DB
db.SetMaxIdleConns(1)
db.SetConnMaxLifetime(0)
db.SetConnMaxIdleTime(5 * time.Minute)
```

### Query Builder - Squirrel

**Usage Example:**
```go
// Build complex queries programmatically
query := squirrel.Select("id", "name", "email").
    From("users").
    Where(squirrel.Eq{"is_active": true}).
    OrderBy("created_at DESC").
    Limit(10)

// Generate SQL: SELECT id, name, email FROM users WHERE is_active = ? ORDER BY created_at DESC LIMIT 10
sql, args, err := query.ToSql()
```

## CLI & Configuration

### Cobra CLI Framework

**Command Structure:**
```go
var rootCmd = &cobra.Command{
    Use:   "flash",
    Short: "Lightning-Fast Type-Safe ORM",
    Long:  `A powerful, database-agnostic ORM built in Go`,
}

var migrateCmd = &cobra.Command{
    Use:   "migrate [name]",
    Short: "Create a new migration",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Migration logic
        return nil
    },
}
```

### Viper Configuration

**Configuration Loading:**
```go
viper.SetConfigName("flash")
viper.SetConfigType("json")
viper.AddConfigPath(".")
viper.AddConfigPath("./config")

if err := viper.ReadInConfig(); err != nil {
    // Handle config not found
}
```

**Environment Variable Binding:**
```go
viper.BindEnv("database.url", "DATABASE_URL")
viper.BindEnv("database.provider", "DB_PROVIDER")
```

## Code Generation

### Template Engine

**Custom Template System:**
```go
type Generator struct {
    templates *template.Template
}

func (g *Generator) Generate() error {
    tmpl, err := template.New("queries").Parse(`
type Queries struct {
    db DBTX
}

{{range .Queries}}
func (q *Queries) {{.Name}}(ctx context.Context{{range .Params}}, {{.Name}} {{.Type}}{{end}}) ({{.ReturnType}}, error) {
    {{if .IsMany}}
    rows, err := q.db.QueryContext(ctx, "{{.SQL}}"{{range .Params}}, {{.Name}}{{end}})
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var items []{{.ItemType}}
    for rows.Next() {
        var item {{.ItemType}}
        if err := rows.Scan({{range .Fields}}&item.{{.Name}}, {{end}}); err != nil {
            return nil, err
        }
        items = append(items, item)
    }
    return items, nil
    {{else}}
    row := q.db.QueryRowContext(ctx, "{{.SQL}}"{{range .Params}}, {{.Name}}{{end}})
    var item {{.ItemType}}
    if err := row.Scan({{range .Fields}}&item.{{.Name}}, {{end}}); err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    return &item, nil
    {{end}}
}
{{end}}
`)
    // Execute template
}
```

### Type Inference

**SQL to Language Type Mapping:**
```go
var typeMappings = map[string]map[string]string{
    "go": {
        "serial": "int64",
        "varchar": "string",
        "boolean": "bool",
        "timestamp": "time.Time",
        "jsonb": "[]byte",
    },
    "typescript": {
        "serial": "number",
        "varchar": "string",
        "boolean": "boolean",
        "timestamp": "Date",
        "jsonb": "any",
    },
    "python": {
        "serial": "int",
        "varchar": "str",
        "boolean": "bool",
        "timestamp": "datetime",
        "jsonb": "dict",
    },
}
```

## Studio Technologies

### Web Framework - Fiber

**High-Performance Web Server:**
```go
app := fiber.New(fiber.Config{
    ServerHeader: "FlashORM Studio",
    AppName: "FlashORM Studio v1.0",
    ErrorHandler: func(c *fiber.Ctx, err error) error {
        return c.Status(500).JSON(fiber.Map{
            "error": err.Error(),
        })
    },
})

// API routes
app.Get("/api/tables", getTables)
app.Post("/api/query", executeQuery)

// Static files
app.Static("/", "./web/dist")
```

### Web UI - Vue.js + TypeScript

**Frontend Stack:**
- **Vue 3** - Progressive framework
- **TypeScript** - Type safety
- **Vite** - Fast build tool
- **Tailwind CSS** - Utility-first CSS
- **Monaco Editor** - SQL editor

**Component Structure:**
```
src/
├── components/
│   ├── TableBrowser.vue
│   ├── QueryRunner.vue
│   ├── DataEditor.vue
│   └── SchemaVisualizer.vue
├── views/
│   ├── Dashboard.vue
│   ├── Tables.vue
│   └── Queries.vue
├── composables/
│   ├── useDatabase.ts
│   └── useQueries.ts
└── types/
    └── database.ts
```

## Build & Distribution

### Cross-Platform Builds

**Makefile Build System:**
```makefile
BINARY_NAME=flash
BUILD_DIR=build
LDFLAGS=-s -w -extldflags "-static"

# Development build
dev:
    CGO_ENABLED=0 go build -tags="dev,plugins" -o $(BUILD_DIR)/flash-dev .

# Production build
prod:
    CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/flash .

# Cross-platform builds
build-all: build-linux build-darwin build-windows

build-linux:
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/flash-linux-amd64 .
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/flash-linux-arm64 .

build-darwin:
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/flash-darwin-amd64 .
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/flash-darwin-arm64 .

build-windows:
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/flash-windows-amd64.exe .
```

### Docker Distribution

**Multi-Stage Dockerfile:**
```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o flash .

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/flash .
CMD ["./flash"]
```

### Package Managers

**NPM Package:**
```json
{
  "name": "flashorm",
  "version": "2.1.11",
  "description": "A powerful, database-agnostic ORM",
  "bin": {
    "flash": "./bin/flash.js"
  },
  "scripts": {
    "postinstall": "node scripts/download.js"
  }
}
```

**Python Package:**
```toml
[build-system]
requires = ["setuptools>=45", "wheel"]
build-backend = "setuptools.build_meta"

[project]
name = "flashorm"
version = "2.1.11"
dependencies = []
```

## Performance & Security

### Performance Optimizations

**Connection Pooling:**
```go
// PostgreSQL
config := pgxpool.ParseConfig(dsn)
config.MaxConns = int32(runtime.GOMAXPROCS(0) * 2)
config.MinConns = 2

// Prepared statement caching
type Queries struct {
    db    DBTX
    stmts map[string]*sql.Stmt
}

func (q *Queries) prepareStmt(ctx context.Context, name, query string) (*sql.Stmt, error) {
    if stmt, exists := q.stmts[name]; exists {
        return stmt, nil
    }
    stmt, err := q.db.PrepareContext(ctx, query)
    if err != nil {
        return nil, err
    }
    q.stmts[name] = stmt
    return stmt, nil
}
```

### Security Measures

**SQL Injection Prevention:**
```go
// Parameterized queries only
func (q *Queries) GetUserByID(ctx context.Context, id int64) (User, error) {
    const query = `SELECT id, name, email FROM users WHERE id = $1`
    row := q.db.QueryRowContext(ctx, query, id)
    // No string concatenation or formatting
}
```

**Input Validation:**
```go
func validateMigrationName(name string) error {
    if len(name) == 0 {
        return errors.New("migration name cannot be empty")
    }
    if len(name) > 255 {
        return errors.New("migration name too long")
    }
    // Check for valid characters
    matched, err := regexp.MatchString(`^[a-zA-Z0-9_\-\s]+$`, name)
    if err != nil || !matched {
        return errors.New("migration name contains invalid characters")
    }
    return nil
}
```

### Memory Management

**Streaming Large Result Sets:**
```go
func (e *Exporter) ExportLargeTable(ctx context.Context, w io.Writer) error {
    rows, err := e.db.QueryContext(ctx, "SELECT * FROM large_table")
    if err != nil {
        return err
    }
    defer rows.Close()

    encoder := json.NewEncoder(w)
    for rows.Next() {
        var item LargeStruct
        if err := rows.Scan(&item.Field1, &item.Field2); err != nil {
            return err
        }
        if err := encoder.Encode(item); err != nil {
            return err
        }
    }
    return nil
}
```

### Error Handling

**Structured Error Types:**
```go
type DatabaseError struct {
    Code       string
    Message    string
    Query      string
    Args       []interface{}
    Underlying error
}

func (e *DatabaseError) Error() string {
    return fmt.Sprintf("database error [%s]: %s", e.Code, e.Message)
}

func (e *DatabaseError) Unwrap() error {
    return e.Underlying
}
```

This technology stack provides a robust, performant, and maintainable foundation for FlashORM's database-agnostic ORM capabilities.
