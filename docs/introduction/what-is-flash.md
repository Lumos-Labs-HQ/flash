---
title: What is Flash ORM?
description: Overview of Flash ORM features and capabilities
---

# What is Flash ORM?

Flash ORM is a powerful, database-agnostic ORM built in Go that provides Prisma-like functionality with multi-database support and type-safe code generation for Go, JavaScript/TypeScript, and Python.

## 🚀 Key Features

### Multi-Database Support
- **PostgreSQL** - Full support with advanced features
- **MySQL** - Complete compatibility
- **SQLite** - File-based database support
- **MongoDB** - NoSQL document database support

### Lightning Fast Performance
FlashORM significantly outperforms popular ORMs:

| Operation | FlashORM | Drizzle | Prisma |
|-----------|----------|---------|--------|
| Insert 1000 Users | **149ms** | 224ms | 230ms |
| Complex Query x500 | **3156ms** | 12500ms | 56322ms |
| Mixed Workload x1000 | **186ms** | 1174ms | 10863ms |
| **Total Time** | **5980ms** | **17149ms** | **71510ms** |

### Type-Safe Code Generation
Generate type-safe code for:
- **Go** - Idiomatic Go with prepared statements
- **TypeScript/JavaScript** - Full type definitions and async support
- **Python** - Async-first with type hints

### Safe Migration System
- Transaction-based migrations with automatic rollback
- Conflict detection and resolution
- Branch-aware schema management (Git-like branching for databases)

### Visual Database Studio
FlashORM Studio provides a web-based interface for:
- Viewing and editing table data
- Visual schema editor
- SQL query execution
- Relationship visualization
- Migration creation and management

### Schema Introspection
Pull existing database schemas and generate migrations automatically - perfect for legacy projects.

## 🎯 Why Flash ORM?

### Developer Experience
- **Familiar CLI** - Similar to Prisma if you're coming from there
- **Multi-Language Support** - Use the same ORM across different languages
- **Plugin Architecture** - Install only what you need

### Production Ready
- **Safe by Default** - Transaction-based operations with rollback
- **Performance Optimized** - Outperforms competitors significantly
- **Comprehensive Testing** - Extensive test coverage and validation

### Flexible & Powerful
- **Database Agnostic** - Switch databases without rewriting code
- **Advanced Features** - Enums, foreign keys, indexes, constraints
- **Export System** - Multiple formats (JSON, CSV, SQLite)

## 🏗️ Architecture

Flash ORM uses a layered architecture:

```
┌─────────────────────────────────────────┐
│              CLI Layer (cmd/)           │
│         Cobra Commands & Flags          │
├─────────────────────────────────────────┤
│           Business Logic Layer          │
│   Migrator, Schema Manager, Export      │
├─────────────────────────────────────────┤
│          Database Adapter Layer         │
│    PostgreSQL, MySQL, SQLite Adapters   │
├─────────────────────────────────────────┤
│            Database Layer               │
│       Actual Database Connections       │
└─────────────────────────────────────────┘
```

## 🔧 Plugin System

Flash ORM uses a modular plugin architecture:

- **Base CLI** (~5-10 MB) - Core functionality only
- **Core Plugin** (~30 MB) - Full ORM features
- **Studio Plugin** (~29 MB) - Visual database editor
- **All Plugin** (~30 MB) - Complete package

Install only what you need for minimal footprint.

## 🌟 Use Cases

### Backend Development
- **API Development** - Type-safe database operations
- **Microservices** - Consistent ORM across services
- **Legacy Migration** - Schema introspection for existing databases

### Full-Stack Development
- **Web Applications** - Generate code for frontend and backend
- **Mobile Apps** - Type-safe APIs for mobile clients
- **Data Processing** - Efficient bulk operations and exports

### DevOps & Database Management
- **Database Studio** - Visual database management
- **Migration Management** - Safe schema changes
- **Branch-Based Development** - Database branching like Git

## 🚀 Getting Started

Ready to try Flash ORM? [Get started in minutes](/getting-started)!

## 📚 Learn More

- [Core Concepts](/concepts/schema)
- [Language Guides](/guides/go)
- [Database Support](/databases/postgresql)
- [Advanced Features](/advanced/how-it-works)
