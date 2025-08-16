# Contributing to Graft

Thank you for your interest in contributing to Graft! This guide will help you get started.

## 🚀 Quick Start

### Prerequisites
- Go 1.24.2+
- Git
- Make (for build automation)

### Setup Development Environment

```bash
# Fork and clone the repository
git clone https://github.com/YOUR_USERNAME/Graft.git
cd Graft

# Set up development environment
make dev-setup

# Build the project
make build-all

# Run tests
make test
```

## 🏗️ Project Structure

```
graft/
├── cmd/                    # CLI commands
├── internal/              # Internal packages
│   ├── config/            # Configuration management
│   ├── database/          # Database adapters
│   ├── migrator/          # Migration logic
│   ├── schema/            # Schema management
│   ├── backup/            # Backup system
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

### 3. Test Your Changes
```bash
# Run all tests
make test

# Format code
make fmt

# Lint code
make lint
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

### ✨ New Features
- New database adapters
- Additional CLI commands
- Enhanced functionality

### 📚 Documentation
- Improve README
- Add examples
- Write tutorials

### 🧪 Testing
- Add unit tests
- Integration tests
- Performance tests

## 🔧 Adding a New Database Adapter

To add support for a new database:

1. **Create adapter file**: `internal/database/newdb.go`
2. **Implement interface**: All `DatabaseAdapter` methods
3. **Add to factory**: Update `NewAdapter()` function
4. **Add template**: Create database-specific templates
5. **Update docs**: Add to README and examples

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
// ... implement other methods
```

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
```

### Testing
```go
func TestFeature(t *testing.T) {
    // Arrange
    setup := createTestSetup()
    
    // Act
    result, err := feature.Execute()
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

## 🚦 Pull Request Guidelines

### Before Submitting
- [ ] Tests pass locally
- [ ] Code is formatted (`make fmt`)
- [ ] Code is linted (`make lint`)
- [ ] Documentation updated
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

## Testing
- [ ] Unit tests added/updated
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Documentation updated
- [ ] No breaking changes
```

## 🐛 Reporting Issues

### Bug Reports
Include:
- Graft version
- Operating system
- Database type and version
- Steps to reproduce
- Expected vs actual behavior

### Feature Requests
Include:
- Clear description of the feature
- Use case and benefits
- Possible implementation approach

## 🏷️ Commit Message Format

Use conventional commits:
```
feat: add MySQL support
fix: resolve migration rollback issue
docs: update installation guide
test: add unit tests for schema parser
refactor: improve error handling
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

Thank you for contributing to Graft! 🚀
