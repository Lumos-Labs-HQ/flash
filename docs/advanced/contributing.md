---
title: Contributing
description: How to contribute to Flash ORM
---

# Contributing to FlashORM

Thank you for your interest in contributing to FlashORM! This guide will help you get started with development, testing, and contributing to the project.

## Table of Contents

- [Quick Start](#quick-start)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Code Style](#code-style)
- [Submitting Changes](#submitting-changes)
- [Release Process](#release-process)
- [Community](#community)

## Quick Start

### Prerequisites

- **Go 1.26+**
- **Git**
- **Task** (`go install github.com/go-task/task/v3/cmd/task@latest`)
- **Docker** (for testing with databases)

### Setup Development Environment

```bash
# Fork and clone the repository
git clone https://github.com/Lumos-Labs-HQ/flash.git
cd flash

# Build the project
task build

# Run tests
task test
```

## Development Setup

### Environment Setup

```bash
# Install Go dependencies
go mod tidy

# Verify setup — print current version
task version
```

### Development Build

```bash
# Build development version (dev + plugins tags)
task build

# Install locally to $GOPATH/bin
task install

# Test the installation
flash --version
```

## Project Structure

```
FlashORM/
├── cmd/                        # CLI commands (cobra)
│   ├── root.go                # Root command (production)
│   ├── root_dev.go            # Root command (dev: all-in-one binary)
│   ├── base.go                # Command registration (prod)
│   ├── base_dev.go            # Command registration (dev)
│   ├── helpers.go             # loadConfigForDB() — multi-db config resolver
│   ├── init.go                # flash init
│   ├── migrate.go             # flash migrate
│   ├── apply.go               # flash apply
│   ├── down.go                # flash down
│   ├── status.go              # flash status
│   ├── gen.go                 # flash gen + runGenForConfig()
│   ├── seed.go                # flash seed
│   ├── export.go              # flash export
│   ├── pull.go                # flash pull
│   ├── raw.go                 # flash raw
│   ├── reset.go               # flash reset
│   ├── studio.go              # flash studio
│   ├── branch.go              # flash branch
│   ├── dblist.go              # flash dblist (multi-db listing)
│   ├── uninstall.go           # flash uninstall
│   ├── plugins.go             # flash plugins
│   ├── add_plugin.go          # flash add-plugin
│   ├── remove_plugin.go       # flash remove-plugin
│   └── update.go              # flash update
├── internal/
│   ├── config/                # Config loading, validation, multi-db resolution
│   │   └── config.go          # Load(), ResolveForDB(), GetDefaultDB(), Validate()
│   ├── schema/                # SQL DDL parser & diff engine
│   │   ├── parser.go          # parseCreateTableStatement(), splitColumnDefinitions()
│   │   ├── schema.go          # ParseSchemaDir(), GenerateSchemaDiff()
│   │   ├── compare.go         # compareSchemas() → SchemaDiff
│   │   ├── snapshot.go        # SaveSchemaSnapshot(), LoadSchemaSnapshot()
│   │   └── sqlcompare.go      # CompareWithDatabase() — live DB vs file diff
│   ├── parser/                # Query parsing & type inference
│   │   ├── query.go           # QueryParser.Parse(), analyzeQuery(), rewriteINListToANY()
│   │   ├── inferrer.go        # TypeInferrer: InferParamName(), InferParamType()
│   │   ├── schema.go          # SchemaParser.Parse() (schema file reading)
│   │   ├── indexed_schema.go  # Fast column lookup by table+name
│   │   └── regex_cache.go     # GetCachedPattern() — compiled regex pool
│   ├── migrator/              # Migration lifecycle
│   │   ├── migrator.go        # NewMigrator(), GenerateMigration()
│   │   ├── operations.go      # Apply(), Down(), Status(), Reset()
│   │   └── branch_aware.go    # Branch-aware migration support
│   ├── gencommon/             # Shared codegen utilities
│   │   ├── cache.go           # GenerationCache — checksum-based incremental gen
│   │   ├── schema.go          # SchemaExpander: ExpandWildcardColumns(), extractTableAliases()
│   │   ├── naming.go          # QueryPascal(), ToCamelCase()
│   │   ├── pipeline.go        # GroupQueries(), Run() — parallel file generation
│   │   ├── enum.go            # ExtractEnumValues()
│   │   └── collections.go     # CQL collection type parsing
│   ├── gogen/                 # Go code generator
│   │   ├── generator.go       # Generate(), generateSQLQueryMethod(), mapSQLTypeToGo()
│   │   └── incremental.go     # Per-file incremental generation
│   ├── jsgen/                 # TypeScript/JavaScript generator
│   │   ├── generator.go       # Generate(), generateOptimizedQueryMethod(), mapSQLTypeToJS()
│   │   └── incremental.go
│   ├── pygen/                 # Python generator
│   │   ├── generator.go       # Generate(), generateQueryMethod(), sqlTypeToPython()
│   │   └── incremental.go
│   ├── kotlingen/             # Kotlin generator
│   │   ├── generator.go       # Generate(), generateDB(), ktTypedGetter(), ktTypedSetter()
│   │   └── incremental.go     # generateSingleKtFile()
│   ├── javagen/               # Java generator
│   │   ├── generator.go       # Generate(), generateDB(), javaTypedGetter(), javaTypedSetter()
│   │   └── incremental.go     # generateSingleJavaFile()
│   ├── database/              # Database adapters
│   │   ├── adapter.go         # Adapter interface definition
│   │   ├── factory.go         # NewAdapter(provider) → Adapter
│   │   ├── postgres/          # PostgreSQL: schema, operations, branch
│   │   ├── mysql/             # MySQL: schema, operations, branch
│   │   ├── sqlite/            # SQLite: schema, operations, branch
│   │   ├── scylla/            # ScyllaDB/Cassandra
│   │   ├── clickhouse/        # ClickHouse
│   │   └── mongodb/           # MongoDB (studio-only, no ORM)
│   ├── studio/                # Visual database management
│   │   ├── sql/               # SQL Studio (Postgres/MySQL/SQLite/Scylla/ClickHouse)
│   │   ├── mongodb/           # MongoDB Studio
│   │   └── redis/             # Redis Studio
│   ├── seeder/                # Smart data seeding
│   │   ├── seeder.go          # Seed() — dependency-ordered insertion
│   │   ├── faker.go           # DataGenerator — realistic fake data
│   │   └── graph.go           # DependencyGraph — topological sort for FK order
│   ├── pull/                  # Reverse-engineer DB → schema files
│   ├── export/                # Export DB to JSON/CSV/SQLite
│   ├── backup/                # Table-level backup before destructive ops
│   ├── branch/                # Schema branching (DB-level isolation)
│   ├── plugin/                # Plugin binary management
│   ├── utils/                 # SQL utilities, naming, validation
│   └── types/                 # Shared type definitions
├── template/                  # flash init templates & project detection
│   ├── init.go                # GetFlashORMConfig(), getGenSection()
│   ├── detect_test.go         # Project type detection tests
│   └── package.go             # DetectJavaPackage(), DetectKotlinPackage()
├── plugins/
│   ├── core/                  # Core plugin binary (prod only)
│   └── studio/                # Studio plugin binary (prod only)
├── example/
│   ├── go/                    # Go example project
│   ├── ts/                    # TypeScript example
│   ├── python/                # Python example
│   ├── java/                  # Java + Kotlin example
│   └── multidb/               # Multi-database example
├── docs/                      # Documentation (VitePress)
│   ├── ARCHITECTURE.md        # Internal architecture & algorithms
│   ├── contributing.md        # This file
│   └── notes/RELEASE_NOTES.md # Changelog
├── test/integration/          # Integration tests (require Docker/DB)
├── Taskfile.yml               # Build tasks
└── main.go                    # Entry point
```

### Key Algorithms & Data Structures

| Component | Algorithm | Complexity | Location |
|-----------|-----------|------------|----------|
| Schema dependency sort | Topological sort (Kahn's) | O(V+E) | `schema/schema.go:sortTablesByDependencies` |
| Schema diff | Map-based set diff | O(n) | `schema/compare.go:compareSchemas` |
| Migration conflict detection | Checksum comparison | O(n) | `migrator/operations.go:hasConflicts` |
| Query param deduplication | Ordered unique set | O(n) | `parser/query.go:extractOrderedParamNums` |
| Wildcard expansion | Alias map + set intersection | O(tables × cols) | `gencommon/schema.go:ExpandWildcardColumns` |
| IN-list rewrite | Regex match + renumber | O(params) | `parser/query.go:rewriteINListToANY` |
| Incremental generation | SHA256 file checksums | O(files) | `gencommon/cache.go` |
| Seeder dependency order | Topological sort | O(V+E) | `seeder/graph.go:BuildInsertionOrder` |
| Type inference | Priority-ordered regex chain | O(patterns) | `parser/inferrer.go:InferParamType` |
| Concurrent query parsing | Worker pool + WaitGroup | O(files/workers) | `parser/query.go:parseFilesConcurrently` |

For detailed algorithm descriptions, see [ARCHITECTURE.md](https://github.com/Lumos-Labs-HQ/flash/blob/main/ARCHITECTURE.md).

## Development Workflow

### 1. Choose an Issue

- Check [GitHub Issues](https://github.com/Lumos-Labs-HQ/flash/issues) for open issues
- Look for issues labeled `good first issue` or `help wanted`
- Comment on the issue to indicate you're working on it

### 2. Create a Branch

```bash
# Create feature branch
git checkout -b feature/your-feature-name

# Or create bug fix branch
git checkout -b fix/issue-number-description

# Or create documentation branch
git checkout -b docs/improve-contributing-guide
```

### 3. Make Changes

Follow the development guidelines:

- **Write tests first** (TDD approach)
- **Keep commits small and focused**
- **Follow Go conventions**
- **Update documentation** when needed
- **Test with all supported databases**

### 4. Test Your Changes

```bash
# Run all unit tests
task test

# Run integration tests
task test-integration

# Run a specific package test manually
go test ./internal/migrator -v
```

### 5. Format and Lint

```bash
# Format code
task fmt

# Lint code
task lint
```

### 6. Commit and Push

```bash
# Add changes
git add .

# Commit with descriptive message
git commit -m "feat: add new feature description

- What was changed
- Why it was changed
- How it was implemented"

# Push to your fork
git push origin feature/your-feature-name
```

### 7. Create Pull Request

- Go to GitHub and create a PR
- Fill out the PR template
- Link to the issue you're solving
- Request review from maintainers

## Testing

### Unit Tests

```go
// Example test file: internal/migrator/migrator_test.go
package migrator

import (
    "context"
    "testing"

    "github.com/Lumos-Labs-HQ/flash/internal/config"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMigrator_ApplyMigration(t *testing.T) {
    cfg := &config.Config{}

    migrator, err := NewMigrator(cfg)
    require.NoError(t, err)

    err = migrator.ApplyMigration(context.Background(), migration)

    assert.NoError(t, err)
}
```

### Integration Tests

```bash
# Build and run integration tests
task test-integration
```

### CLI Testing

```bash
# Build and test CLI commands manually
task build
./build/flash --help
./build/flash version
```

## Code Style

### Go Code Style

Follow standard Go conventions:

```go
// Package comment
package migrator

import (
    "context"
    "fmt"

    "github.com/Lumos-Labs-HQ/flash/internal/config"
)

// Migrator handles database migrations
type Migrator struct {
    adapter database.DatabaseAdapter
    cfg     *config.Config
}

// ApplyMigration applies a single migration to the database
func (m *Migrator) ApplyMigration(ctx context.Context, migration *Migration) error {
    return nil
}
```

### Naming Conventions

- **Packages:** lowercase, single word (e.g., `migrator`, `config`)
- **Types:** PascalCase (e.g., `DatabaseAdapter`, `Migration`)
- **Functions:** PascalCase (e.g., `ApplyMigration`, `GetUserByID`)
- **Variables:** camelCase (e.g., `userID`, `migrationName`)
- **Constants:** PascalCase (e.g., `MaxRetries`, `DefaultTimeout`)

### Error Handling

```go
func (m *Migrator) ApplyMigration(ctx context.Context, migration *Migration) error {
    if migration == nil {
        return fmt.Errorf("migration cannot be nil")
    }

    if err := m.validateMigration(migration); err != nil {
        return fmt.Errorf("invalid migration: %w", err)
    }

    return nil
}
```

## Submitting Changes

### Pull Request Guidelines

**PR Title Format:**
```
type(scope): description

Examples:
feat(migrator): add support for MongoDB migrations
fix(config): resolve environment variable parsing issue
docs(contributing): improve testing guidelines
refactor(parser): simplify SQL parsing logic
```

**PR Description Template:**
```markdown
## Description
Brief description of the changes

## Type of Change
- [ ] Bug fix (non-breaking change)
- [ ] New feature (non-breaking change)
- [ ] Breaking change
- [ ] Documentation update
- [ ] Refactoring

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed
- [ ] All tests pass

## Checklist
- [ ] Code follows Go conventions
- [ ] Documentation updated
- [ ] Tests added for new functionality
- [ ] No breaking changes
- [ ] Ready for review

## Related Issues
Closes #123
```

### Code Review Process

1. **Automated Checks**: CI runs tests, linting, and security checks
2. **Peer Review**: At least one maintainer reviews the code
3. **Testing**: Reviewer may request additional tests
4. **Approval**: PR is approved and merged
5. **Release**: Changes are included in the next release

## Release Process

### Version Numbering

Follows [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Workflow

1. **Create Release Branch**
   ```bash
   git checkout -b release/v1.2.0
   ```

2. **Update Version**
   ```go
   // cmd/root.go
   Version = "1.2.0"
   ```

3. **Update Changelog**
   ```markdown
   ## v1.2.0 - 2024-01-15

   ### Features
   - Add MongoDB support

   ### Bug Fixes
   - Fix migration rollback issue
   ```

4. **Run Release Tests**
   ```bash
   task test
   task test-integration
   ```

5. **Create Git Tag**
   ```bash
   git tag v1.2.0
   git push origin v1.2.0
   ```

6. **GitHub Actions**
   - Builds binaries for all platforms
   - Creates GitHub release
   - Publishes to npm and PyPI
   - Updates documentation

## Community

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and community discussion
- **Discord**: Real-time chat (if available)
- **Twitter**: Announcements and updates

### Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help newcomers learn
- Follow the [Contributor Covenant](https://www.contributor-covenant.org/)

### Recognition

Contributors are recognized in:
- **GitHub Contributors list**
- **CHANGELOG.md** for significant contributions
- **Release notes** for major features

### Getting Help

- **Documentation**: Check docs first
- **Issues**: Search existing issues
- **Discussions**: Ask the community

## Additional Resources

### Learning Resources

- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Testing](https://golang.org/pkg/testing/)

### Development Tools

- **golangci-lint**: Linting and code quality
- **goimports**: Import management
- **delve**: Go debugger

### All Task Commands

```bash
# Show all available tasks
task --list

task build            # Build dev binary (dev + plugins tags)
task install          # Install dev binary to $GOPATH/bin
task test             # Run all unit tests
task test-integration # Run integration tests
task fmt              # Format code
task lint             # Lint code
task version          # Print current version
```

Thank you for contributing to FlashORM! Every contribution — bug fixes, features, documentation, or helping other contributors — is valuable.
