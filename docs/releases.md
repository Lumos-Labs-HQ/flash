---
title: Release Notes
description: Flash ORM release notes and changelog
---

# FlashORM Release Notes

## Version 2.3.0 - Latest Release

### 🔴 Redis Studio (New Feature)

A comprehensive web-based Redis management interface with advanced features:

**Key Management**
- Export/Import keys to/from JSON format
- Bulk TTL update - set expiration for multiple keys by pattern
- Database purge with confirmation

**Monitoring & Analysis**
- **Memory Analysis** - Per-key memory usage, type statistics, memory overview
- **Slow Log Viewer** - View slow queries with duration, command, and client info
- **Cluster/Replication Info** - View cluster state, nodes, and replication status

**Advanced Features**
- **Lua Script Editor** - Write, execute, and load Lua scripts with KEYS/ARGV support
- **Pub/Sub Management** - Publish messages and view active channels with subscriber counts
- **Config Viewer/Editor** - View, modify, and rewrite Redis configuration
- **ACL Management** - View users and ACL security log (Redis 6.0+)

**UI Improvements**
- Fixed visual gaps in data tables with sticky headers
- State persistence across browser sessions
- Responsive design with dark theme

```bash
# Launch Redis Studio
flash studio redis --url "redis://localhost:6379"
```

### 🍃 MongoDB Studio Improvements

- **Bulk Delete Documents** - Delete multiple documents at once using `$in` operator
- **Delete Database** - Drop entire databases with confirmation
- **Collection Context Menu** - Right-click options for collection management
- **Improved Collection Selection** - Fixed active state highlighting

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

### 🐛 Bug Fixes

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

# Seed multiple tables with different counts
flash seed users:100 posts:500 comments:1000

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

## Previous Releases

### Version 2.2.21

#### 🐛 Bug Fixes
- Go Code Generator: Fixed unnecessary imports in generated `models.go`
- JavaScript Code Generator: Removed redundant `.d.ts` files
- Schema Parser: Fixed folder-based schema parsing

### Version 2.1.11

#### ✨ New Features

- **MongoDB Support**: Full MongoDB integration with document modeling
- **Branch-Aware Migrations**: Git-like branching for database schemas
- **Enhanced Export System**: Improved data export with compression
- **Plugin Architecture**: Modular plugin system for reduced footprint

#### 🐛 Bug Fixes

- Fixed connection pooling issues in high-concurrency scenarios
- Improved error handling for malformed SQL queries
- Resolved memory leaks in long-running processes

#### 📊 Performance Improvements

- 15% faster query execution through optimized prepared statements
- Reduced memory usage by 20% in code generation
- Improved startup time for CLI commands

### Version 2.0.8

#### ✨ Major Features

- **Multi-Language Code Generation**: Support for Go, TypeScript, and Python
- **Visual Studio Interface**: Web-based database management UI
- **Advanced Migration System**: Safe migrations with automatic rollback
- **Schema Introspection**: Pull schemas from existing databases

#### 🔄 Breaking Changes

- Configuration file format updated to v2
- CLI command structure reorganized
- Plugin system introduced (base CLI is now minimal)

### Version 1.5.0

#### ✨ Features

- **PostgreSQL Full Support**: Complete PostgreSQL feature set
- **MySQL Integration**: MySQL database support
- **SQLite Support**: File-based database operations
- **Basic Code Generation**: Initial Go code generation

#### 🐛 Bug Fixes

- Fixed migration ordering issues
- Improved error messages for common mistakes
- Resolved connection timeout problems

### Version 1.0.0

#### 🎉 Initial Release

- **Core ORM Functionality**: Basic CRUD operations
- **Migration System**: Simple migration management
- **CLI Interface**: Command-line database operations
- **Go Support**: Initial Go language support

---

## Beta Releases

### Version 2.2.0-beta1

#### ✨ Experimental Features

- **Redis Integration**: Key-value store support (experimental)
- **GraphQL API Generation**: Auto-generated GraphQL schemas
- **Advanced Analytics**: Query performance insights
- **Cloud Database Support**: AWS RDS, Google Cloud SQL integration

#### ⚠️ Known Issues

- Redis integration may have connection stability issues
- GraphQL generation is in early stages
- Cloud integrations require additional configuration

---

## Installation Instructions

### NPM Installation
```bash
npm install -g flashorm
```

### Python Installation
```bash
pip install flashorm
```

### Go Installation
```bash
go install github.com/Lumos-Labs-HQ/flash@latest
```

### Binary Downloads
Download from [GitHub Releases](https://github.com/Lumos-Labs-HQ/flash/releases)

---

## Migration Guide

### From v1.x to v2.x

1. **Update Configuration**: Convert `flash.config.json` to v2 format
2. **Reinstall CLI**: Use new plugin system
3. **Regenerate Code**: Run `flash gen` to update generated files
4. **Test Migrations**: Verify migration compatibility

### Breaking Changes in v2.0

- Configuration file requires version field
- Plugin system requires separate installation
- Some CLI commands have been reorganized
- Generated code structure has changed

---

## Future Roadmap

### Planned Features

- **Enhanced Plugin Ecosystem**: Community plugin marketplace
- **Advanced Query Builder**: Visual query construction
- **Real-time Collaboration**: Multi-user studio sessions
- **Kubernetes Integration**: Cloud-native database operations
- **Machine Learning Integration**: AI-powered query optimization

### Long-term Vision

- **Universal Database API**: Single API for all database types
- **Auto-scaling**: Automatic performance optimization
- **Multi-cloud Support**: Seamless cloud database management
- **Advanced Analytics**: Built-in business intelligence features

---

For detailed documentation, see:
- [Usage Guide - Go](guides/go)
- [Usage Guide - TypeScript](guides/typescript)
- [Usage Guide - Python](guides/python)
- [Contributing Guide](contributing)
- [API Reference](reference/cli)
