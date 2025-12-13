---
title: Why Flash ORM?
description: Why choose Flash ORM over other ORMs
---

# Why Flash ORM?

Flash ORM stands out from other ORMs through its unique combination of performance, safety, and developer experience. Here's why developers choose Flash ORM.

## ⚡ Performance That Matters

### Real-World Benchmarks

Flash ORM doesn't just claim to be fast - it proves it:

| Operation | FlashORM | Drizzle | Prisma |
|-----------|----------|---------|--------|
| Insert 1000 Users | **149ms** | 224ms | 230ms |
| Complex Query x500 | **3156ms** | 12500ms | 56322ms |
| Mixed Workload x1000 | **186ms** | 1174ms | 10863ms |
| **Total Time** | **5980ms** | **17149ms** | **71510ms** |

**Flash ORM is 2.8x faster than Drizzle and 11.9x faster than Prisma.**

### Why So Fast?

1. **Go Performance** - Built in Go for maximum performance
2. **Prepared Statements** - Automatic statement caching and reuse
3. **Connection Pooling** - Optimized database connection management
4. **Minimal Overhead** - Direct SQL execution without unnecessary abstractions
5. **Parallel Processing** - Concurrent operations where possible

## 🛡️ Safety First

### Transaction-Based Migrations
Every migration runs in a transaction with automatic rollback on failure:

```sql
-- Migration runs in transaction
BEGIN;
-- Your schema changes here
ALTER TABLE users ADD COLUMN email VARCHAR(255);
-- If anything fails, automatic rollback
COMMIT;
```

### Conflict Detection
Flash ORM automatically detects and prevents:
- Column conflicts
- Foreign key violations
- Constraint conflicts
- Data integrity issues

### Branch-Aware Development
Manage database schema changes like code:
```bash
# Create feature branch
flash branch create feature/user-profiles

# Make schema changes
flash migrate "add user profiles"

# Switch branches
flash checkout main

# Merge when ready
flash branch merge feature/user-profiles
```

## 🎯 Developer Experience

### Familiar CLI
If you've used Prisma, you'll feel right at home:

```bash
# Initialize project
flash init --postgresql

# Create migration
flash migrate "add users table"

# Apply changes
flash apply

# Generate code
flash gen

# Open visual editor
flash studio
```

### Multi-Language Support
One ORM for your entire stack:

```typescript
// TypeScript
const user = await db.getUserById(1);
```

```python
# Python
user = await db.get_user_by_id(1)
```

```go
// Go
user, err := db.GetUserById(ctx, 1)
```

### Type Safety Everywhere
Full type safety across all supported languages with IDE autocomplete.

## 🔄 Database Agnostic

### Switch Databases Effortlessly
Change databases without rewriting code:

```json
// flash.config.json
{
  "database": {
    "provider": "postgresql"
    // Change to "mysql" or "sqlite" anytime
  }
}
```

### Consistent API
Same code works across PostgreSQL, MySQL, SQLite, and MongoDB.

## 🏗️ Production Ready

### Comprehensive Testing
- Extensive unit and integration tests
- Real database testing with Docker
- Performance regression testing
- Cross-platform compatibility

### Enterprise Features
- **Audit Logging** - Track all database changes
- **Backup & Export** - Multiple formats (JSON, CSV, SQLite)
- **Schema Introspection** - Pull from existing databases
- **Visual Studio** - Web-based database management

### Security
- **SQL Injection Prevention** - Parameterized queries only
- **Connection Security** - SSL/TLS support
- **Access Control** - Configurable permissions

## 🛠️ Plugin Architecture

### Minimal Footprint
Install only what you need:

```bash
# Core functionality only (~30MB)
flash add-plug core

# Visual studio only (~29MB)
flash add-plug studio

# Everything (~30MB)
flash add-plug all
```

### Extensible
Build custom plugins for specialized functionality.

## 🌟 Unique Features

### Visual Database Studio
Web-based interface for database management:
- View/edit data visually
- Create migrations graphically
- Execute SQL queries
- Visualize relationships
- Manage branches

### Schema Introspection
Pull schemas from existing databases:

```bash
# Pull current schema
flash pull

# Generate migration for differences
flash migrate "sync with production"
```

### Advanced Export
Export data in multiple formats:

```bash
# Export to JSON
flash export --format json

# Export to CSV
flash export --format csv

# Export to SQLite
flash export --format sqlite
```

## 🚀 Future-Proof

### Active Development
- Regular releases with new features
- Community-driven development
- Open source with commercial support available

### Roadmap Highlights
- **Redis Support** - Key-value store integration
- **GraphQL Integration** - Auto-generated GraphQL APIs
- **Advanced Analytics** - Query performance insights
- **Cloud Integration** - Managed database support

## 💡 When to Choose Flash ORM

### Choose Flash ORM if you need:

- **Maximum Performance** - Speed-critical applications
- **Type Safety** - Full-stack type safety
- **Multi-Language Support** - Polyglot development teams
- **Database Flexibility** - Switch databases easily
- **Visual Management** - Database Studio for teams
- **Production Safety** - Enterprise-grade reliability

### Consider alternatives if you need:

- **Single Language** - Language-specific ORMs might be simpler
- **NoSQL Focus** - MongoDB-only solutions
- **Complex Relationships** - Graph databases
- **Real-time Features** - Specialized real-time databases

## 🏁 Get Started

Ready to experience the difference? [Get started now](/getting-started)!
