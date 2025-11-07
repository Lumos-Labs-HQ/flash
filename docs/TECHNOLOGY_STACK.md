# Technology Stack & Dependencies

This document outlines all the technologies, libraries, and tools used in the FlashORM project.

## Table of Contents

- [Core Technologies](#core-technologies)
- [Database Drivers](#database-drivers)
- [CLI Framework](#cli-framework)
- [Configuration Management](#configuration-management)
- [Build Tools](#build-tools)
- [Code Generation System](#code-generation-system)
- [Export System](#export-system)
- [FlashORM Studio Technologies](#FlashORM-studio-technologies)
  - [Backend - Go Fiber](#backend---go-fiber)
  - [Frontend Technologies](#frontend-technologies)
  - [Studio Architecture](#studio-architecture)
  - [Studio Pages](#studio-pages)
- [Architecture Patterns](#architecture-patterns)
- [Safe Migration System](#safe-migration-system)
- [Performance Optimizations](#performance-optimizations)
- [Security Considerations](#security-considerations)
- [Testing Strategy](#testing-strategy)
- [NPM Distribution System](#npm-distribution-system)
- [Version Information](#version-information)

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
- Embedded file system (go:embed) for static assets

## Database Drivers

### PostgreSQL - pgx/v5

**Repository**: https://github.com/jackc/pgx  
**Version**: v5.7.6

**Features:**
- High-performance PostgreSQL driver
- Connection pooling with pgxpool optimized for Supabase/PgBouncer
- Native PostgreSQL protocol implementation
- Support for PostgreSQL-specific types (JSONB, UUID, arrays)
- Prepared statement caching
- Copy protocol support
- Context-aware operations
- Transaction safety with automatic rollback

**Usage in FlashORM:**
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
```

**Additional Dependencies:**
- `github.com/jackc/pgpassfile v1.0.0` - Password file parsing
- `github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761` - Service file support
- `github.com/jackc/puddle/v2 v2.2.2` - Connection pooling
- `github.com/lib/pq v1.10.9` - PostgreSQL driver (legacy support)

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

**Additional Dependency:**
- `filippo.io/edwards25519 v1.1.0` - Cryptographic operations

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

## CLI Framework

### Cobra

**Repository**: https://github.com/spf13/cobra  
**Version**: v1.10.1

**Features:**
- Powerful CLI framework
- Subcommand support
- Flag parsing and validation
- Auto-generated help
- Shell completion
- Man page generation
- POSIX-compliant flags

**Commands Implemented:**

| Command | Description | Flags |
|---------|-------------|-------|
| `FlashORM init` | Initialize a new FlashORM project | `--postgresql`, `--mysql`, `--sqlite` |
| `FlashORM migrate [name]` | Create a new migration | `--empty`, `-e` |
| `FlashORM apply` | Apply pending migrations | `--force`, `-f` |
| `FlashORM status` | Show migration status | None |
| `FlashORM export` | Export database | `--json`, `-j`, `--csv`, `-c`, `--sqlite`, `-s` |
| `FlashORM reset` | Reset database | `--force`, `-f` |
| `FlashORM gen` | Generate type-safe code | None |
| `FlashORM pull` | Extract schema from database | `--backup`, `-b`, `--output`, `-o` |
| `FlashORM raw <sql>` | Execute raw SQL | `--query`, `-q`, `--file` |
| `FlashORM studio` | Launch visual database editor | `--port`, `--db` |

**Additional Dependencies:**
- `github.com/inconshreveable/mousetrap v1.1.0` - Windows command-line support
- `github.com/spf13/pflag v1.0.10` - POSIX flag parsing

### Color Output - fatih/color

**Repository**: https://github.com/fatih/color  
**Version**: v1.18.0

**Features:**
- Cross-platform colored terminal output
- Multiple color and style options
- Windows support
- Performance optimized

**Additional Dependencies:**
- `github.com/mattn/go-colorable v0.1.14` - Colorable writer
- `github.com/mattn/go-isatty v0.0.20` - TTY detection

## Configuration Management

### Viper

**Repository**: https://github.com/spf13/viper  
**Version**: v1.21.0

**Features:**
- Configuration management
- Multiple format support (JSON, YAML, TOML, HCL)
- Environment variable binding
- Remote configuration support
- Configuration watching
- Default value handling

**Configuration Structure:**
```go
type Config struct {
    SchemaPath     string    `json:"schema_path"`
    MigrationsPath string    `json:"migrations_path"`
    ExportPath     string    `json:"export_path"`
    Database       Database  `json:"database"`
    Gen            GenConfig `json:"gen"`
}

type GenConfig struct {
    Go GenLanguageConfig `json:"go"`
    JS GenLanguageConfig `json:"js"`
}
```

**Additional Dependencies:**
- `github.com/fsnotify/fsnotify v1.9.0` - File system notifications
- `github.com/go-viper/mapstructure/v2 v2.4.0` - Struct mapping
- `github.com/pelletier/go-toml/v2 v2.2.4` - TOML support
- `github.com/sagikazarmark/locafero v0.12.0` - Virtual file system
- `github.com/spf13/afero v1.15.0` - File system abstraction
- `github.com/spf13/cast v1.10.0` - Type casting
- `github.com/subosito/gotenv v1.6.0` - Environment loading
- `go.yaml.in/yaml/v3 v3.0.4` - YAML support

### godotenv

**Repository**: https://github.com/joho/godotenv  
**Version**: v1.5.1

**Features:**
- Load environment variables from .env files
- Multiple .env file support
- Variable expansion
- Override protection

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
compress:     # Compress binaries with UPX
```

### Go Modules

**go.mod Features:**
- Dependency versioning
- Semantic versioning support
- Module proxy support
- Vendor directory support
- Replace directives for local development

### Additional Build Dependencies

- `github.com/google/go-cmp v0.7.0` - Value comparison
- `github.com/mattn/go-runewidth v0.0.16` - Unicode width calculation
- `github.com/rogpeppe/go-internal v1.10.0` - Internal utilities
- `github.com/rivo/uniseg v0.2.0` - Unicode segmentation
- `github.com/google/uuid v1.6.0` - UUID generation
- `golang.org/x/crypto v0.43.0` - Cryptographic operations
- `golang.org/x/sync v0.17.0` - Concurrency primitives
- `golang.org/x/sys v0.37.0` - System calls
- `golang.org/x/text v0.30.0` - Text processing

## Code Generation System

### Custom Go Generator (`internal/gogen/`)

**Features:**
- Type-safe Go code generation from SQL
- Automatic struct generation from tables
- Query method generation with context support
- PostgreSQL ENUM to Go const types
- Null-safe type handling with sql.Null* types
- Zero runtime dependencies

**Generated Output:**
```go
// FlashORM_gen/models.go
type Users struct {
    ID        sql.NullInt32  `json:"id" db:"id"`
    Name      string         `json:"name" db:"name"`
    Email     string         `json:"email" db:"email"`
    CreatedAt time.Time      `json:"created_at" db:"created_at"`
}

// FlashORM_gen/db.go
type DBTX interface {
    Exec(query string, args ...interface{}) (sql.Result, error)
    Query(query string, args ...interface{}) (*sql.Rows, error)
    QueryRow(query string, args ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
    return &Queries{db: db}
}
```

**Query Annotations:**
- `:one` - Returns single row or null
- `:many` - Returns array of rows
- `:exec` - Returns affected row count

### Custom JavaScript/TypeScript Generator (`internal/jsgen/`)

**Features:**
- Type-safe JavaScript/TypeScript code generation
- Automatic TypeScript definition generation
- Query parsing with special annotations
- PostgreSQL ENUM to TypeScript union types
- Zero runtime dependencies
- Full IntelliSense support

**Type Mapping:**
```go
var sqlToTSTypeMap = map[string]string{
    "SERIAL":                      "number",
    "INTEGER":                     "number",
    "BIGINT":                      "number",
    "VARCHAR":                     "string",
    "TEXT":                        "string",
    "BOOLEAN":                     "boolean",
    "TIMESTAMP WITH TIME ZONE":    "Date",
    "JSONB":                       "any",
    "UUID":                        "string",
}
```

**Generated Output:**
```typescript
// index.d.ts
export interface Users {
  id: number | null;
  name: string;
  email: string;
  created_at: Date;
}

export class Queries {
  getUser(id: number): Promise<Users | null>;
  createUser(name: string, email: string): Promise<Users | null>;
  listUsers(): Promise<Users[]>;
}

export function New(db: any): Queries;
```

### Query Builder - Squirrel

**Repository**: https://github.com/Masterminds/squirrel  
**Version**: v1.5.4

**Features:**
- Fluent SQL query builder
- Multiple database support
- Placeholder formatting
- Query caching
- Type-safe query construction

**Additional Dependencies:**
- `github.com/lann/builder v0.0.0-20180802200727-47ae307949d0` - Builder pattern
- `github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0` - Persistent data structures

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

## FlashORM Studio Technologies

### Backend - Go Fiber

#### Web Framework - Fiber v2

**Repository**: https://github.com/gofiber/fiber  
**Version**: v2.52.9

**Features:**
- Express-inspired web framework for Go
- Fast HTTP engine built on fasthttp
- Zero memory allocation router
- Template engine support
- Static file serving
- Middleware support
- RESTful API support

**Usage in FlashORM:**
```go
app := fiber.New(fiber.Config{
    Views: engine,
})

// UI Routes
app.Get("/", s.handleIndex)
app.Get("/sql", s.handleSQL)
app.Get("/schema", s.handleSchema)

// API Routes
api := app.Group("/api")
api.Get("/tables", s.handleGetTables)
api.Post("/tables/:name/save", s.handleSaveChanges)
api.Post("/sql", s.handleExecuteSQL)
```

**Additional Dependencies:**
- `github.com/valyala/fasthttp v1.51.0` - Fast HTTP implementation
- `github.com/valyala/bytebufferpool v1.0.0` - Byte buffer pooling
- `github.com/valyala/tcplisten v1.0.0` - TCP listener
- `github.com/klauspost/compress v1.17.9` - Compression algorithms
- `github.com/andybalholm/brotli v1.1.0` - Brotli compression

#### Template Engine - Gofiber HTML

**Repository**: https://github.com/gofiber/template  
**Version**: v2.1.3

**Features:**
- HTML template rendering
- Go template syntax
- Fast compilation
- Embedded template support

**Additional Dependencies:**
- `github.com/gofiber/template v1.8.3` - Base template package
- `github.com/gofiber/utils v1.1.0` - Utility functions

**Implementation:**
```go
//go:embed static/*
var StaticFS embed.FS

//go:embed templates/*
var TemplatesFS embed.FS

engine := html.NewFileSystem(http.FS(TemplatesFS), ".html")
app := fiber.New(fiber.Config{
    Views: engine,
})
```

### Frontend Technologies

#### React Ecosystem (Schema Visualization)

**React**: v18.2.0  
**React DOM**: v18.2.0  
**Source**: https://esm.sh (ES Module CDN)

**Features:**
- Modern React with Hooks
- Client-side rendering
- Component-based architecture
- ES Modules via importmap

**Usage:**
```html
<script type="importmap">
{
    "imports": {
        "react": "https://esm.sh/react@18.2.0",
        "react-dom/client": "https://esm.sh/react-dom@18.2.0/client"
    }
}
</script>
```

#### ReactFlow (Schema Diagram)

**Package**: @xyflow/react  
**Version**: v12.8.4  
**Repository**: https://github.com/xyflow/xyflow

**Features:**
- Interactive node-based diagrams
- Automatic graph layout with dagre
- Drag and drop nodes
- Zoom and pan controls
- Custom node rendering
- Edge relationships

**Additional Library:**
- **Dagre**: v0.8.5 - Graph layout algorithm for automatic node positioning

**Usage:**
```jsx
import { ReactFlow, Background, Controls, MiniMap } from '@xyflow/react';
import dagre from 'dagre';

// Automatic layout
const dagreGraph = new dagre.graphlib.Graph();
dagreGraph.setGraph({ rankdir: 'TB' });
```

#### CodeMirror 5 (SQL Editor)

**Version**: 5.65.2  
**Source**: https://cdnjs.cloudflare.com

**Features:**
- Syntax highlighting for SQL
- Code completion
- Line numbers
- Bracket matching
- Material Darker theme
- Keyboard shortcuts (Ctrl+Enter to execute)
- SQL mode support

**Files Loaded:**
- `codemirror.min.css` - Base styles
- `theme/material-darker.min.css` - Dark theme
- `codemirror.min.js` - Core editor
- `mode/sql/sql.min.js` - SQL syntax highlighting

**Usage:**
```javascript
const editor = CodeMirror.fromTextArea(document.getElementById('sql-editor'), {
    mode: 'text/x-sql',
    theme: 'material-darker',
    lineNumbers: true,
    autofocus: true,
    extraKeys: {
        'Ctrl-Enter': runQuery
    }
});
```

#### Iconify

**Version**: 2.2.1 (index), 3.1.0 (sql/schema)  
**Source**: https://code.iconify.design

**Features:**
- Icon framework with 100,000+ icons
- Material Design Icons (MDI)
- Zero dependencies
- On-demand loading
- SVG-based icons

**Icons Used:**
- `mdi:table` - Data browser
- `mdi:code-braces` - SQL editor
- `mdi:file-tree` - Schema visualization
- `mdi:refresh` - Refresh button
- Various action icons

#### Google Fonts

**Fonts Used:**
- **Inter** (wght@400;500;600) - Main UI font (index, sql pages)
- **JetBrains Mono** (wght@400;500;600) - Code font (schema page)

**Source**: https://fonts.googleapis.com

#### Vanilla JavaScript

**Custom Scripts:**
- `static/js/studio.js` - Main table browser logic
- `static/js/modal.js` - Modal component system
- `static/js/index.js` - Index page controller
- `static/js/sql.js` - SQL editor logic
- `static/js/schema.js` - React schema visualization

**Features:**
- Fetch API for AJAX requests
- Real-time table editing
- Inline cell editing with double-click
- Pagination and search
- CSV export from SQL results
- Resizable split panes

### Studio Architecture

#### Embedded File System

**Go embed Package**: Standard library

**Features:**
- Embed static files in binary
- No external file dependencies
- Single binary distribution
- Hot reload in development

**Implementation:**
```go
//go:embed static/*
var StaticFS embed.FS

//go:embed templates/*
var TemplatesFS embed.FS

// Serve static files
staticFS, _ := fs.Sub(StaticFS, "static")
app.Use("/static", filesystem.New(filesystem.Config{
    Root: http.FS(staticFS),
}))
```

#### API Design

**RESTful Endpoints:**

```go
// Table Operations
GET    /api/tables              // List all tables
GET    /api/tables/:name        // Get table data with pagination
POST   /api/tables/:name/save   // Save batch changes
POST   /api/tables/:name/add    // Add new row
POST   /api/tables/:name/delete // Delete multiple rows
DELETE /api/tables/:name/rows/:id // Delete single row

// SQL Operations
POST   /api/sql                 // Execute SQL query

// Schema Operations
GET    /api/schema              // Get database schema
```

**Response Format:**
```json
{
    "columns": [
        {
            "name": "id",
            "type": "INTEGER",
            "nullable": false,
            "primaryKey": true
        }
    ],
    "rows": [
        {"id": 1, "name": "Alice"}
    ],
    "total": 100,
    "page": 1,
    "limit": 50
}
```

### Studio Pages

#### 1. Data Browser (`/`)

**Technologies:**
- Vanilla JavaScript
- Fetch API for AJAX
- Modal system for forms
- Real-time table editing
- Pagination and search

**Features:**
```javascript
// Inline editing
cell.addEventListener('dblclick', () => {
    cell.contentEditable = true;
    cell.focus();
});

// Batch operations
const changes = [];
changes.push({ id, column, value });
await saveChanges(tableName, changes);
```

**Files:**
- Template: `templates/index.html`
- Scripts: `static/js/studio.js`, `static/js/modal.js`, `static/js/index.js`
- Styles: `static/css/index.css`

#### 2. SQL Editor (`/sql`)

**Technologies:**
- CodeMirror 5 for SQL editing
- Split-pane resizable interface
- CSV export functionality
- Vanilla JavaScript

**Features:**
```javascript
// Execute query with Ctrl+Enter
async function runQuery() {
    const sql = editor.getValue();
    const response = await fetch('/api/sql', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: sql })
    });
    displayResults(await response.json());
}

// Export results to CSV
function exportToCSV() {
    const csv = generateCSV(results);
    downloadFile(csv, 'export.csv');
}
```

**Files:**
- Template: `templates/sql.html`
- Script: `static/js/sql.js`
- Styles: `static/css/sql.css`

#### 3. Schema Visualization (`/schema`)

**Technologies:**
- React 18.2.0
- ReactFlow 12.8.4
- Dagre 0.8.5 for auto-layout
- ES Modules

**Features:**
```javascript
// Auto-layout with dagre
function layoutNodes(nodes, edges) {
    const dagreGraph = new dagre.graphlib.Graph();
    dagreGraph.setGraph({ rankdir: 'TB' });
    
    nodes.forEach(node => {
        dagreGraph.setNode(node.id, { width: 250, height: 100 });
    });
    
    edges.forEach(edge => {
        dagreGraph.setEdge(edge.source, edge.target);
    });
    
    dagre.layout(dagreGraph);
    
    return nodes.map(node => ({
        ...node,
        position: dagreGraph.node(node.id)
    }));
}

// Custom table nodes with columns
function TableNode({ data }) {
    return (
        <div className="table-node">
            <div className="table-name">{data.name}</div>
            <div className="columns">
                {data.columns.map(col => (
                    <div key={col.name} className="column">
                        {col.name}: {col.type}
                    </div>
                ))}
            </div>
        </div>
    );
}
```

**Files:**
- Template: `templates/schema.html`
- Script: `static/js/schema.js`
- Styles: `static/css/schema.css`

## Architecture Patterns

### Adapter Pattern

Used for database abstraction:
```go
type DatabaseAdapter interface {
    Connect(ctx context.Context, url string) error
    ExecuteMigration(ctx context.Context, migrationSQL string) error
    RecordMigration(ctx context.Context, migrationID, name, checksum string) error
    GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error)
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

func (pt *ProjectTemplate) GetFlashORMConfig() string {
    return fmt.Sprintf(`{
  "schema_path": "db/schema/schema.sql",
  "migrations_path": "db/migrations",
  "export_path": "db/export",
  "database": {
    "provider": "%s",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "go": {
      "enabled": true,
      "output_path": "FlashORM_gen"
    },
    "js": {
      "enabled": false,
      "output_path": "FlashORM_gen"
    }
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
        fmt.Println("   Transaction rolled back. Fix the error and run 'FlashORM apply' again.")
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

**PostgreSQL (pgxpool):**
```go
config.MaxConns = 10
config.MinConns = 2
config.MaxConnLifetime = time.Hour
config.MaxConnIdleTime = 30 * time.Minute
config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec // Pooler compatibility
```

**MySQL/SQLite (database/sql):**
- Connection reuse and lifecycle management
- Configurable pool sizes and timeouts

### Query Optimization

- Prepared statement caching
- Batch operations for bulk data
- Index-aware query generation
- Transaction batching for migrations
- Streaming for large exports

### Studio Performance

**Batch Query Optimization:**
```go
// Single query for all table row counts
func (p *PostgresAdapter) GetAllTableRowCounts(ctx context.Context, tables []string) (map[string]int, error) {
    query := `
        SELECT schemaname || '.' || tablename as table_name, n_live_tup as row_count
        FROM pg_stat_user_tables
        WHERE tablename = ANY($1)
    `
    // 95% fewer database queries
}
```

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
- Query builder usage (Squirrel)
- No dynamic SQL construction
- Transaction isolation

### Studio Security

**Development Tool - Local Use Only:**
- No authentication (local development tool)
- Binds to localhost by default
- CORS disabled for localhost
- SQL injection prevention via parameterized queries
- XSS prevention via proper escaping
- Transaction rollback on errors

**Production Usage (Not Recommended):**
```bash
# If you must use remotely, use SSH tunnel
ssh -L 5555:localhost:5555 user@server
FlashORM studio --port 5555
```

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

### Test Helpers

**Files:**
- `test/test-helper.go` - Common test utilities
- `test/db-helper.sh` - Database setup scripts
- `test/test-studio.sh` - Studio testing
- `test/github-workflow-test.sh` - CI/CD tests

## NPM Distribution System

### Package: FlashORM-orm

**Registry**: https://www.npmjs.com/package/FlashORM-orm  
**Repository**: https://github.com/Lumos-Labs-HQ/FlashORM

**Installation:**
```bash
npm install -g FlashORM-orm
```

**Features:**
- Automatic binary download from GitHub releases
- Cross-platform (Linux, macOS, Windows)
- Multi-architecture (x64, ARM64, ARM)
- Small package (~3KB)
- Postinstall script for binary setup
- Programmatic API

**Binary Download System:**
```javascript
const VERSION = '2.0.0';
const REPO = 'Lumos-Labs-HQ/FlashORM';
const downloadUrl = `https://github.com/${REPO}/releases/download/v${VERSION}/FlashORM-${platform}-${arch}`;
```

**Files:**
- `npm/package.json` - NPM package configuration
- `npm/index.js` - Programmatic API
- `npm/bin/FlashORM.js` - CLI wrapper
- `npm/scripts/install.js` - Post-install binary downloader

**GitHub Actions Automation:**
- Triggers after successful GitHub release
- Auto-updates version from git tag
- Publishes to NPM registry
- Verifies installation

## Version Information

**Current Version**: v2.0.0

**Version History:**
- v2.0.0: Full rewrite with Studio, React-based schema visualization, improved CLI
- v1.7.0: Added JavaScript/TypeScript code generation
- v1.6.0: Added export system and safe migration features
- v1.5.0: Enhanced schema management and conflict detection
- Previous versions: Core migration functionality

This comprehensive technology stack ensures FlashORM is robust, performant, and maintainable while supporting multiple database systems, providing safe migration execution, offering flexible export capabilities, and delivering first-class support for both Go and Node.js ecosystems with a powerful visual database management studio.
