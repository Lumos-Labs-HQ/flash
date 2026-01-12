# FlashORM Release Notes

## Version 2.2.3 - Latest Release

### ⚡ Performance Improvements

#### Database Adapters
- **87% faster** migration generation (PostgreSQL: 88%, MySQL: 82%, SQLite: 90%)
- PostgreSQL: Split complex 7-way JOIN into 2 simple queries with Go-side merge (70% faster)
- PostgreSQL: Replaced expensive subqueries with LEFT JOIN optimization (50-80% faster)
- SQLite: Parallelized table column fetching with goroutines (10x speedup)
- SQLite: Eliminated N+1 query problem for unique column checks (90-97% faster)
- Pre-compiled regex patterns in schema parsing (5-10ms saved per migration)
- Pre-allocated maps in schema comparisons to reduce GC pressure

#### Code Generators
- Pre-compiled regex patterns in all generators (3-5x faster parsing)
- Go Generator: Slice pre-allocation for `:many` queries (`make([]T, 0, 8)`)
- Python Generator: Statement caching via `self._stmts` dictionary
- Python Generator: Optimized asyncpg row access (direct Record access vs `dict()`)
- JavaScript Generator: Shared utilities, removed redundant regex compilation
- Shared Utilities: `utils.ExtractTableName()` and `utils.IsModifyingQuery()`

### 🔒 Security Fixes
- **CRITICAL**: Fixed SQL injection vulnerability in SQLite PRAGMA queries with table name validation

### � Bug Fixes

#### Database Adapters
- **CRITICAL**: MySQL constraint-backed index filter to prevent migration crashes
- SQLite: Fixed error propagation in `GetAllTablesIndexes`
- MySQL: Fixed enum name collision using `$` separator

#### Code Generators
- Go: Fixed unnecessary imports in generated `models.go` (conditional imports only when needed)
- JavaScript: Removed redundant `.d.ts` files, now only generates `index.d.ts`
- Python: Fixed `generateBatchMethod` to respect async/sync configuration
- Schema Parser: Fixed folder-based parsing to use `schema_dir` config properly

### 🧹 Code Quality Improvements

#### Database Adapters
- Removed **394 lines** of duplicate code (23% reduction)
- Consolidated duplicate `GetTableColumns` and `GetTableIndexes` functions
- Replaced 100+ line `PullCompleteSchema` with 3-line reuse pattern
- Applied DRY principles across all adapters

#### General Refactoring
- Consolidated duplicate `SplitColumns` functions in `utils/sql.go`
- Removed unused regex fields from generator structs
- Fixed empty else blocks in code generation
- Replaced deprecated `strings.Title` with custom `toTitleCase`
- Added proper error handling for `os.Getwd()` calls
- Standardized error messages to "flash" package name
- Interface-based schema validation to reduce reflection usage

### 🌱 Database Seeding (New Feature)

Seed your database with realistic fake data:

```bash
# Seed all tables with default count
flash seed

# Seed specific table with count
flash seed --table users --count 100

# Truncate tables before seeding
flash seed --truncate
```

**Features:**
- Automatic fake data generation based on column types
- Smart relationship handling (foreign keys)
- Support for all data types: strings, numbers, dates, emails, etc.
- Dependency graph for correct insertion order

### 📦 Installation

**NPM (Node.js/TypeScript)**
```bash
npm install -g flashorm
```

**Python**
```bash
pip install flashorm
```

**Go**
```bash
go install github.com/Lumos-Labs-HQ/flash@latest
```

---

For detailed documentation, see:
- [Usage Guide - Go](docs/USAGE_GO.md)
- [Usage Guide - TypeScript](docs/USAGE_TYPESCRIPT.md)
- [Usage Guide - Python](docs/USAGE_PYTHON.md)
- [Contributing](docs/CONTRIBUTING.md)
