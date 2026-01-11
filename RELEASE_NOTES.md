# FlashORM Release Notes

## Version 2.2.3 - Latest Release

### ⚡ Performance Improvements

#### Code Generators
- **Pre-compiled Regex Patterns**: All generators now use package-level pre-compiled regex for 3-5x faster parsing
- **Go Generator**: Added slice pre-allocation for `:many` queries (`make([]T, 0, 8)`)
- **Python Generator**: Added statement caching via `self._stmts` dictionary
- **Python Generator**: Optimized asyncpg row access (direct Record access instead of `dict()`)
- **JavaScript Generator**: Uses shared utilities, removed redundant regex compilation
- **Shared Utilities**: Created `utils.ExtractTableName()` and `utils.IsModifyingQuery()` used by all generators

#### Database Common
- Pre-compiled regex patterns in `internal/database/common/utils.go` for SQL parsing

### � Code Quality Improvements

#### Refactoring
- Consolidated duplicate `SplitColumns` functions in `utils/sql.go`
- Removed unused regex fields from generator structs
- Fixed empty else blocks in code generation
- Replaced deprecated `strings.Title` with custom `toTitleCase` function

#### Error Handling
- Added proper error handling for `os.Getwd()` calls in parser
- Standardized error message package names to "flash"

#### Validation
- Implemented interface-based schema validation to reduce reflection usage
- Added type assertion approach with reflection fallback for compatibility

### �🐛 Bug Fixes

#### Go Code Generator
- Fixed unnecessary imports in generated `models.go`
- `database/sql` is now only imported when nullable types are used
- `time` package is now only imported when timestamp/date fields exist

#### JavaScript Code Generator
- Removed redundant `.d.ts` files (`users.d.ts`, `database.d.ts`)
- Now only generates `index.d.ts` for TypeScript type definitions

#### Python Code Generator
- Fixed `generateBatchMethod` to respect async/sync configuration
- Batch methods now correctly generate async code when `async: true`

#### Schema Parser
- Fixed folder-based schema parsing to properly use `schema_dir` config
- Query validator now works correctly with split schema files

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
