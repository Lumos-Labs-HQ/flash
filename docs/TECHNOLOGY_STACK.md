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

**Placeholder Formatting:**
```go
// PostgreSQL: $1, $2, $3
qb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

// MySQL/SQLite: ?, ?, ?
qb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)
```

**Usage:**
```go
query := qb.Select("id", "name", "email").
    From("users").
    Where(squirrel.Eq{"id": userID}).
    Limit(1)

sql, args, _ := query.ToSql()
```

**Dependencies:**
- `lann/builder` v0.0.0-20180802200727
- `lann/ps` v0.0.0-20150810152359

## CLI & Configuration

### Cobra CLI Framework

**Command Structure:**
```go
var rootCmd = &cobra.Command{
    Use:   "flash",
    Short: "Type-safe ORM with code generation",
}

func init() {
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
    rootCmd.PersistentFlags().BoolP("force", "f", false, "Skip confirmations")
}
```

**Dependencies:**
- `inconshreveable/mousetrap` v1.1.0 (Windows support)
- `spf13/pflag` v1.0.10 (POSIX flags)

### Viper Configuration

**Config Loading Priority:**
1. Command-line flags (`--config`)
2. Environment variables
3. `./flash.config.json`
4. Default values

**Dependencies:**
- `fsnotify/fsnotify` v1.9.0
- `go-viper/mapstructure/v2` v2.4.0
- `pelletier/go-toml/v2` v2.2.4
- `sagikazarmark/locafero` v0.12.0
- `spf13/afero` v1.15.0
- `spf13/cast` v1.10.0
- `subosito/gotenv` v1.6.0
- `go.yaml.in/yaml/v3` v3.0.4

### Terminal Colors - fatih/color

**Dependencies:**
- `mattn/go-colorable` v0.1.14
- `mattn/go-isatty` v0.0.20

## Code Generation

### Parser System (`internal/parser/`)

**Regex Patterns:**
```go
// Compiled once with sync.Once
createTableRegex = regexp.MustCompile(
    `(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s*\(([\s\S]*?)\);`
)

enumRegex = regexp.MustCompile(
    `(?i)CREATE\s+TYPE\s+(\w+)\s+AS\s+ENUM\s*\(\s*([^)]+)\s*\)`
)

paramRegex = regexp.MustCompile(`\$\d+|\?`)
```

**Type Inference Cache:**
```go
type TypeInferrer struct {
    cache map[string]string // "table:paramIndex:paramName" → "TYPE"
}
```

### Go Generator (`internal/gogen/`)

**Type Mapping:**
```go
var goTypeMap = map[string]string{
    "SERIAL":    "sql.NullInt32",
    "INTEGER":   "sql.NullInt32",
    "BIGINT":    "sql.NullInt64",
    "VARCHAR":   "string",
    "TEXT":      "string",
    "BOOLEAN":   "sql.NullBool",
    "TIMESTAMP": "time.Time",
    "JSONB":     "[]byte",
    "UUID":      "string",
}
```

**String Builder Pre-allocation:**
```go
estimatedSize := len(schema.Tables)*500 + len(queries)*300
sb := strings.Builder{}
sb.Grow(estimatedSize)
```

### JavaScript/TypeScript Generator (`internal/jsgen/`)

**Type Mapping:**
```go
var sqlToTSTypeMap = map[string]string{
    "SERIAL":    "number",
    "INTEGER":   "number",
    "VARCHAR":   "string",
    "TEXT":      "string",
    "BOOLEAN":   "boolean",
    "TIMESTAMP": "Date",
    "JSONB":     "any",
    "UUID":      "string",
}
```

**Database Client Selection:**
```go
switch provider {
case "postgresql":
    return "pg"  // npm: pg
case "mysql":
    return "mysql2"  // npm: mysql2
case "sqlite":
    return "better-sqlite3"  // npm: better-sqlite3
}
```

### Python Generator (`internal/pygen/`)

**Type Mapping:**
```go
var sqlToPyTypeMap = map[string]string{
    "SERIAL":    "int",
    "INTEGER":   "int",
    "VARCHAR":   "str",
    "TEXT":      "str",
    "BOOLEAN":   "bool",
    "TIMESTAMP": "datetime",
    "JSONB":     "dict",
    "UUID":      "str",
}
```

**Database Client Selection:**
```go
switch provider {
case "postgresql":
    return "asyncpg"  // pip: asyncpg
case "mysql":
    return "aiomysql"  // pip: aiomysql
case "sqlite":
    return "aiosqlite"  // pip: aiosqlite
}
```

## Studio Technologies

### Backend - Fiber v2

**Server Setup:**
```go
engine := html.NewFileSystem(http.FS(TemplatesFS), ".html")
app := fiber.New(fiber.Config{
    Views: engine,
})

// Embedded static files
staticFS, _ := fs.Sub(StaticFS, "static")
app.Use("/static", filesystem.New(filesystem.Config{
    Root: http.FS(staticFS),
}))
```

**Dependencies:**
- `valyala/fasthttp` v1.51.0
- `valyala/bytebufferpool` v1.0.0
- `valyala/tcplisten` v1.0.0
- `klauspost/compress` v1.17.9
- `andybalholm/brotli` v1.1.0
- `gofiber/template` v1.8.3
- `gofiber/template/html/v2` v2.1.3
- `gofiber/utils` v1.1.0

### Frontend Stack

**Data Browser (`/`):**
- Vanilla JavaScript
- Fetch API
- Inline cell editing
- Modal system

**SQL Editor (`/sql`):**
- CodeMirror 5.65.2
- Material Darker theme
- SQL syntax highlighting
- Keyboard shortcuts (Ctrl+Enter)

**Schema Visualization (`/schema`):**
- React 18.2.0 (via esm.sh)
- ReactFlow (@xyflow/react) 12.8.4
- Dagre 0.8.5 (auto-layout)
- ES Modules

**CDN Sources:**
```html
<!-- React -->
<script type="importmap">
{
    "imports": {
        "react": "https://esm.sh/react@18.2.0",
        "react-dom/client": "https://esm.sh/react-dom@18.2.0/client",
        "@xyflow/react": "https://esm.sh/@xyflow/react@12.8.4",
        "dagre": "https://esm.sh/dagre@0.8.5"
    }
}
</script>

<!-- CodeMirror -->
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/codemirror/5.65.2/codemirror.min.css">
<script src="https://cdnjs.cloudflare.com/ajax/libs/codemirror/5.65.2/codemirror.min.js"></script>

<!-- Icons -->
<script src="https://code.iconify.design/2/2.2.1/iconify.min.js"></script>

<!-- Fonts -->
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&display=swap" rel="stylesheet">
```

**Performance Optimization:**
```go
// Batch query - 95% fewer DB calls
func (p *PostgresAdapter) GetAllTableRowCounts(ctx context.Context, tables []string) (map[string]int, error) {
    query := `
        SELECT tablename, n_live_tup 
        FROM pg_stat_user_tables 
        WHERE tablename = ANY($1)
    `
    // Single query instead of N queries
}
```

## Build & Distribution

### Makefile Targets

```makefile
build-all:    # Cross-compile for 5 platforms
install:      # Install to GOPATH/bin
clean:        # Remove build artifacts
test:         # Run tests
deps:         # Download dependencies
fmt:          # Format code
lint:         # Lint code
release:      # Create release build
compress:     # UPX compression
```

**Build Configuration:**
```makefile
LDFLAGS=-s -w -extldflags "-static"

# Linux AMD64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -trimpath

# macOS ARM64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -trimpath

# Windows AMD64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -trimpath
```

### NPM Distribution

**Package Structure:**
```
flashorm/
├── package.json      # ~3KB wrapper
├── bin/flash.js      # CLI wrapper
└── scripts/install.js # Binary downloader
```

**Postinstall Script:**
```javascript
const VERSION = '2.0.8';
const REPO = 'Lumos-Labs-HQ/flash';

const platformMap = {
    'darwin': 'darwin',
    'linux': 'linux',
    'win32': 'windows'
};

const archMap = {
    'x64': 'amd64',
    'arm64': 'arm64'
};

const downloadUrl = `https://github.com/${REPO}/releases/download/v${VERSION}/flash-${platform}-${arch}`;
```

### Python Distribution

**Setup Configuration:**
```python
class PostInstallCommand(install):
    def run(self):
        install.run(self)
        install_script = os.path.join(self.install_lib, 'flashorm', 'install.py')
        subprocess.check_call([sys.executable, install_script])

setup(
    name='flashorm',
    version='2.0.8',
    cmdclass={'install': PostInstallCommand},
    entry_points={
        'console_scripts': ['flash=flashorm.cli:main'],
    },
)
```

### GitHub Actions Workflows

**Release Pipeline:**
```yaml
# 1. release.yml - Build binaries on tag push
on:
  push:
    tags: ['v*']

# 2. npmrelease.yml - Publish to NPM after release
on:
  workflow_run:
    workflows: ["Release"]
    types: [completed]

# 3. pypi-release.yml - Publish to PyPI after release
on:
  workflow_run:
    workflows: ["Release"]
    types: [completed]
```

## Performance & Security

### Connection Pooling

**PostgreSQL (Optimized for Supabase/PgBouncer):**
```go
config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
config.MaxConns = 2
config.MinConns = 0
config.MaxConnLifetime = 15 * time.Minute
```

**MySQL/SQLite:**
```go
db.SetMaxOpenConns(2)
db.SetMaxIdleConns(0)
db.SetConnMaxLifetime(15 * time.Minute)
```

### Memory Optimization

**Pre-allocated Slices:**
```go
tables := make([]*Table, 0, 8)
columns := make([]*Column, 0, 16)
queries := make([]*Query, 0, len(files)*4)
```

**String Builder:**
```go
sb := strings.Builder{}
sb.Grow(estimatedSize)
```

**Regex Compilation:**
```go
var (
    createTableRegex *regexp.Regexp
    regexOnce        sync.Once
)

func initRegex() {
    createTableRegex = regexp.MustCompile(...)
}

regexOnce.Do(initRegex)
```

### Transaction Safety

**Migration Execution:**
```go
func (p *PostgresAdapter) ExecuteMigration(ctx context.Context, sql string) error {
    tx, err := p.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx) // Auto-rollback on error
    
    for _, stmt := range statements {
        if _, err := tx.Exec(ctx, stmt); err != nil {
            return err // Rollback triggered
        }
    }
    
    return tx.Commit(ctx)
}
```

### SQL Injection Prevention

**Parameterized Queries Only:**
```go
// ✅ Safe - Parameterized
query := qb.Select("*").From("users").Where(squirrel.Eq{"id": userID})

// ❌ Never used - Dynamic SQL
query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", userID)
```

### Studio Security

**Local Development Tool:**
- Binds to `localhost` by default
- No authentication (local use only)
- Parameterized queries prevent SQL injection
- Transaction rollback on errors
- XSS prevention via proper escaping

**Remote Access (Not Recommended):**
```bash
# Use SSH tunnel if needed
ssh -L 5555:localhost:5555 user@server
flash studio --port 5555
```

## Utility Dependencies

**General:**
- `google/go-cmp` v0.7.0 - Value comparison
- `google/uuid` v1.6.0 - UUID generation
- `mattn/go-runewidth` v0.0.16 - Unicode width
- `rivo/uniseg` v0.2.0 - Unicode segmentation
- `rogpeppe/go-internal` v1.10.0 - Internal utilities

**Crypto & System:**
- `golang.org/x/crypto` v0.43.0
- `golang.org/x/sync` v0.17.0
- `golang.org/x/sys` v0.37.0
- `golang.org/x/text` v0.30.0

## Version Information

**Current Version:** v2.0.8

**Go Version:** 1.24.2

**Supported Platforms:**
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

**Distribution Channels:**
- GitHub Releases (binaries)
- NPM Registry (npm package)
- PyPI (Python package)
- Go Install (`go install github.com/Lumos-Labs-HQ/flash@latest`)
