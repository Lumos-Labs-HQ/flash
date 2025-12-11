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

- **Go 1.24.2+**
- **Git**
- **Make** (for build automation)
- **Docker** (for testing with databases)

### Setup Development Environment

```bash
# Fork and clone the repository
git clone https://github.com/YOUR_USERNAME/flash.git
cd flash

# Set up development environment
make dev-setup

# Build the project
make build-all

# Run tests
make test
```

## Development Setup

### Environment Setup

```bash
# Install Go dependencies
go mod download

# Set up pre-commit hooks
make setup-hooks

# Create development databases (requires Docker)
make db-setup

# Verify setup
make verify
```

### Development Build

```bash
# Build development version with all features
make dev

# Install locally
make install-dev

# Test the installation
flash --version
```

### IDE Setup

**VS Code:**
- Install Go extension
- Configure settings for Go development
- Set up debugging configuration

**GoLand/IntelliJ:**
- Import project as Go module
- Configure Go SDK 1.24.2+
- Set up run configurations

## Project Structure

```
FlashORM/
├── cmd/                    # CLI commands
│   ├── root.go            # Root command
│   ├── init.go            # Init command
│   ├── migrate.go         # Migration commands
│   ├── gen.go             # Code generation
│   └── studio.go          # Studio command
├── internal/              # Internal packages
│   ├── config/            # Configuration management
│   ├── database/          # Database adapters
│   │   ├── adapter.go     # Common interface
│   │   ├── postgres/      # PostgreSQL adapter
│   │   ├── mysql/         # MySQL adapter
│   │   └── sqlite/        # SQLite adapter
│   ├── migrator/          # Migration logic
│   ├── schema/            # Schema management
│   ├── parser/            # SQL parser
│   ├── gogen/             # Go code generator
│   ├── jsgen/             # JavaScript/TypeScript generator
│   ├── pygen/             # Python generator
│   ├── export/            # Data export system
│   ├── studio/            # Web studio
│   └── utils/             # Utilities
├── template/              # Project templates
├── docs/                  # Documentation
├── example/               # Example projects
│   ├── go/                # Go example
│   ├── python/            # Python example
│   └── ts/                # TypeScript example
├── test/                  # Test utilities
├── scripts/               # Build and release scripts
├── Makefile              # Build automation
└── go.mod                # Go module
```

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
# Run all tests
make test

# Run specific test
go test ./internal/migrator -v

# Test with different databases
make test-postgres
make test-mysql
make test-sqlite

# Run integration tests
make test-integration

# Test CLI commands
make test-cli
```

### 5. Format and Lint

```bash
# Format code
make fmt

# Lint code
make lint

# Check for security issues
make security

# Run all quality checks
make quality
```

### 6. Update Documentation

```bash
# Build documentation
make docs

# Serve documentation locally
make docs-serve

# Check for broken links
make docs-check
```

### 7. Commit and Push

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

### 8. Create Pull Request

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
    // Setup
    cfg := &config.Config{
        // test configuration
    }

    migrator, err := NewMigrator(cfg)
    require.NoError(t, err)

    // Test
    err = migrator.ApplyMigration(context.Background(), migration)

    // Assert
    assert.NoError(t, err)
    // Additional assertions...
}
```

### Integration Tests

```go
// Example integration test
func TestMigrationWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup test database
    db := setupTestDatabase(t)
    defer db.Close()

    // Run migration workflow
    cfg := createTestConfig(db)
    migrator, err := NewMigrator(cfg)
    require.NoError(t, err)

    // Create and apply migration
    migration := createTestMigration()
    err = migrator.ApplyMigration(context.Background(), migration)
    assert.NoError(t, err)

    // Verify migration was applied
    applied, err := migrator.GetAppliedMigrations(context.Background())
    assert.NoError(t, err)
    assert.Contains(t, applied, migration.ID)
}
```

### Database-Specific Tests

```bash
# Test with PostgreSQL
make test-postgres

# Test with MySQL
make test-mysql

# Test with SQLite
make test-sqlite

# Test with MongoDB
make test-mongodb
```

### CLI Testing

```bash
# Test CLI commands
go run main.go --help
go run main.go version

# Test with example project
cd example/go
go run ../../main.go status
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

// Struct comments
type Migrator struct {
    // Field comments
    adapter database.DatabaseAdapter
    cfg     *config.Config
}

// Function comments
// ApplyMigration applies a single migration to the database
func (m *Migrator) ApplyMigration(ctx context.Context, migration *Migration) error {
    // Implementation
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
// Good error handling
func (m *Migrator) ApplyMigration(ctx context.Context, migration *Migration) error {
    if migration == nil {
        return fmt.Errorf("migration cannot be nil")
    }

    if err := m.validateMigration(migration); err != nil {
        return fmt.Errorf("invalid migration: %w", err)
    }

    // Implementation
    return nil
}

// Bad error handling
func (m *Migrator) ApplyMigration(ctx context.Context, migration *Migration) error {
    // Don't ignore errors
    m.validateMigration(migration) // Wrong!

    // Don't return generic errors
    return errors.New("something went wrong") // Wrong!
}
```

### Documentation

```go
// Package-level documentation
// Package migrator handles database migrations for FlashORM.
// It provides functionality to create, apply, and rollback migrations
// across different database systems.
package migrator

// Function documentation
// CreateMigration creates a new migration file with the given name.
// The migration will include both up and down SQL scripts.
// Returns the path to the created migration file or an error.
func CreateMigration(name string) (string, error) {
    // Implementation
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

   ### Breaking Changes
   - Remove deprecated API endpoints
   ```

4. **Run Release Tests**
   ```bash
   make test-release
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
- **Discord**: Real-time help

## Additional Resources

### Learning Resources

- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Testing](https://golang.org/pkg/testing/)
- [Database Internals](https://www.databass.dev/)

### Development Tools

- **golangci-lint**: Linting and code quality
- **goimports**: Import management
- **delve**: Go debugger
- **benchstat**: Benchmark analysis

### Project Scripts

```bash
# View all available make targets
make help

# Clean build artifacts
make clean

# Update dependencies
make deps

# Generate mocks for testing
make mocks

# Run security checks
make security
```

Thank you for contributing to FlashORM! Your contributions help make the project better for everyone. Whether it's fixing bugs, adding features, improving documentation, or helping other contributors, every contribution is valuable.
