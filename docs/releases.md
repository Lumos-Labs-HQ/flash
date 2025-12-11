---
title: Release Notes
description: Flash ORM release notes and changelog
---

# FlashORM Release Notes

## Version 2.2.21 - Latest Release

### 🐛 Bug Fixes

#### Go Code Generator
- Fixed unnecessary imports in generated `models.go`
- `database/sql` is now only imported when nullable types are used
- `time` package is now only imported when timestamp/date fields exist

#### JavaScript Code Generator
- Removed redundant `.d.ts` files (`users.d.ts`, `database.d.ts`)
- Now only generates `index.d.ts` for TypeScript type definitions

#### Schema Parser
- Fixed folder-based schema parsing to properly use `schema_dir` config
- Query validator now works correctly with split schema files

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
