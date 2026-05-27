---
title: Examples
description: Complete examples for every Flash ORM feature
---

# Flash ORM Examples

This section contains comprehensive, copy-paste-ready examples for every Flash ORM feature. Each example is self-contained and covers real-world use cases.

## Quick Navigation

| Example | What You'll Learn |
|---------|-----------------|
| [Complete Workflows](/examples/complete-workflow) | End-to-end project setup in Go, TypeScript, and Python |
| [CLI Commands](/examples/cli-commands) | Every CLI command with all flags and options |
| [Schema Patterns](/examples/schema-patterns) | Common database schema designs |
| [Query Patterns](/examples/query-patterns) | SQL query patterns for code generation |
| [Seeding Patterns](/examples/seeding-patterns) | Database seeding for realistic test data |

## Example by Task

### Starting a New Project

```bash
# 1. Initialize
flash init --postgresql

# 2. Define schema
# Edit db/schema/schema.sql

# 3. Create and apply migration
flash migrate "initial schema"
flash apply

# 4. Seed with test data
flash seed --count 50

# 5. Generate code
flash gen

# 6. Launch studio
flash studio
```

### Adding a New Feature

```bash
# 1. Edit schema files
# Add new tables/columns to db/schema/

# 2. Generate migration
flash migrate "add comments table"

# 3. Apply migration
flash apply

# 4. Add queries
# Edit db/queries/comments.sql

# 5. Regenerate code
flash gen

# 6. Seed new tables
flash seed comments:100
```

### Resetting Development Database

```bash
# Full reset with fresh data
flash reset --force
flash apply
flash seed --count 25
```

### Exporting Data

```bash
# Export entire database to JSON
flash export --format json --output backup.json

# Export specific table to CSV
flash export --table users --format csv --output users.csv

# Export to SQLite for local analysis
flash export --format sqlite --output local_copy.db
```

## Language-Specific Examples

### Go

```go
// Connect and query
queries := flash_gen.New(db)
user, err := queries.GetUserByID(ctx, 1)
```

### TypeScript

```typescript
// Connect and query
const db = New(pool);
const user = await db.getUserById(1);
```

### Python

```python
# Connect and query
db = new(pool)
user = await db.get_user_by_id(1)
```

Browse the pages above for complete, runnable examples.
