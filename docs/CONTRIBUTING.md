# Contributing to FlashORM

Thank you for your interest in contributing to FlashORM! This guide will help you get started.

## 🚀 Quick Start

### Prerequisites
- Go 1.24.2+
- Git
- Make (for build automation)

### Setup Development Environment

```bash
# Fork and clone the repository
git clone https://github.com/YOUR_USERNAME/FlashORM.git
cd FlashORM

# Set up development environment
make dev-setup

# Build the project
make build-all

# Run tests
make test
```

## 🏗️ Project Structure

```
FlashORM/
├── cmd/                    # CLI commands
├── internal/              # Internal packages
│   ├── config/            # Configuration management
│   ├── database/          # Database adapters
│   ├── migrator/          # Migration logic
│   ├── schema/            # Schema management
│   ├── export/            # Export system
│   ├── pull/              # Schema introspection
│   └── utils/             # Utility functions
├── template/              # Project templates
├── docs/                  # Documentation
└── example/               # Example project
```

## 🛠️ Development Workflow

### 1. Create a Branch
```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/bug-description
```

### 2. Make Changes
- Follow Go conventions
- Add tests for new features
- Update documentation if needed
- Test safe migration features

### 3. Test Your Changes
```bash
# Run all tests
make test

# Format code
make fmt

# Lint code
make lint

# Test with example project
cd example && FlashORM apply
```

### 4. Commit and Push
```bash
git add .
git commit -m "feat: add new feature"
git push origin feature/your-feature-name
```

### 5. Create Pull Request
- Use the PR template
- Describe your changes clearly
- Link any related issues

## 🎯 What Can You Contribute?

### 🐛 Bug Fixes
- Fix existing issues
- Improve error handling
- Performance optimizations
- Safe migration improvements

### ✨ New Features
- New database adapters
- Additional CLI commands
- Enhanced export formats
- Migration safety features

### 📚 Documentation
- Improve README
- Add examples
- Write tutorials
- Update API docs

### 🧪 Testing
- Add unit tests
- Integration tests
- Performance tests
- Migration safety tests

## 🔧 Adding a New Database Adapter

To add support for a new database:

1. **Create adapter file**: `internal/database/newdb.go`
2. **Implement interface**: All `DatabaseAdapter` methods
3. **Add transaction safety**: Implement safe migration execution
4. **Add to factory**: Update `NewAdapter()` function
5. **Add template**: Create database-specific templates
6. **Update docs**: Add to README and examples

Example structure:
```go
type NewDBAdapter struct {
    db *sql.DB
}

func NewNewDBAdapter() *NewDBAdapter {
    return &NewDBAdapter{}
}

func (n *NewDBAdapter) Connect(ctx context.Context, url string) error {
    // Implementation
}

func (n *NewDBAdapter) ExecuteMigration(ctx context.Context, migrationSQL string) error {
    // Must implement transaction safety
    tx, err := n.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback() // Auto-rollback on error
    
    // Execute statements
    // ...
    
    return tx.Commit()
}
// ... implement other methods
```

## 🔒 Safe Migration Development

When working on migration features:

### Transaction Safety
- Each migration must run in its own transaction
- Automatic rollback on any failure
- Proper error handling and reporting

### Export Integration
- Ensure export works before destructive operations
- Support all export formats (JSON, CSV, SQLite)
- Test export/import roundtrip

### Conflict Detection
- Implement proper conflict detection
- Provide clear resolution options
- Test with various conflict scenarios

## 📝 Code Style

### Go Conventions
- Use `gofmt` for formatting
- Follow effective Go guidelines
- Use meaningful variable names
- Add comments for public functions

### Error Handling
```go
// Good: Wrap errors with context
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Migration-specific error handling
if err := m.applySingleMigrationSafely(ctx, migration); err != nil {
    fmt.Printf("❌ Failed at migration: %s\n", migration.ID)
    fmt.Printf("   Error: %v\n", err)
    fmt.Println("   Transaction rolled back. Fix the error and run 'FlashORM apply' again.")
    return err
}
```

### Testing
```go
func TestSafeMigration(t *testing.T) {
    // Arrange
    setup := createTestSetup()
    
    // Act
    err := migrator.Apply(ctx)
    
    // Assert
    assert.NoError(t, err)
    // Verify migration was recorded
    // Verify database state is correct
}
```

## 🚦 Pull Request Guidelines

### Before Submitting
- [ ] Tests pass locally
- [ ] Code is formatted (`make fmt`)
- [ ] Code is linted (`make lint`)
- [ ] Documentation updated
- [ ] Migration safety tested
- [ ] Export functionality tested
- [ ] Commit messages are clear

### PR Template
```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Performance improvement
- [ ] Migration safety improvement

## Testing
- [ ] Unit tests added/updated
- [ ] Manual testing completed
- [ ] Migration safety verified
- [ ] Export functionality tested

## Checklist
- [ ] Code follows style guidelines
- [ ] Documentation updated
- [ ] No breaking changes
- [ ] Transaction safety maintained
```

## 🐛 Reporting Issues

### Bug Reports
Include:
- FlashORM version (`FlashORM --version`)
- Operating system
- Database type and version
- Steps to reproduce
- Expected vs actual behavior
- Migration files (if applicable)
- Export/import logs

### Feature Requests
Include:
- Clear description of the feature
- Use case and benefits
- Possible implementation approach
- Impact on migration safety

## 🏷️ Commit Message Format

Use conventional commits:
```
feat: add MySQL support with transaction safety
fix: resolve migration rollback issue in PostgreSQL adapter
docs: update export system documentation
test: add unit tests for safe migration execution
refactor: improve error handling in export system
```

## 🎉 Recognition

Contributors are recognized in:
- README contributors section
- Release notes
- GitHub insights

## 📞 Getting Help

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and ideas
- **Pull Request Comments**: For code-specific discussions

## 📜 License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to FlashORM! 🚀
