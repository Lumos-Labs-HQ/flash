# Graft ORM

A powerful, database-agnostic migration CLI tool built in Go with multi-database support and type-safe code generation.

## Features

- 🗃️ **Multi-Database Support**: PostgreSQL, MySQL, SQLite
- 🔄 **Migration Management**: Create, apply, and track migrations
- 🔒 **Safe Migration System**: Transaction-based execution with automatic rollback
- 📤 **Smart Export System**: Multiple formats (JSON, CSV, SQLite)
- 🔧 **Code Generation**: Generate type-safe Go and JavaScript code
- ⚡ **Fast & Reliable**: Built in Go for performance
- 🎯 **Prisma-like Commands**: Familiar CLI interface

## Installation

```bash
npm install -g graft-orm
```

## Quick Start

### 1. Initialize Your Project

```bash
cd your-project
graft init --postgresql  # or --mysql, --sqlite
```

### 2. Configure Database

```bash
# Set your database URL
export DATABASE_URL="postgres://user:password@localhost:5432/mydb"

# Or create .env file
echo "DATABASE_URL=postgres://user:password@localhost:5432/mydb" > .env
```

### 3. Create Your First Migration

```bash
graft migrate "create users table"
```

### 4. Apply Migrations

```bash
graft apply
```

### 5. Generate Code

```bash
# Generate Go code
graft gen

# Generate JavaScript code
graft gen --js
```

## Commands

| Command | Description |
|---------|-------------|
| `graft init` | Initialize project with database-specific templates |
| `graft migrate <name>` | Create a new migration file |
| `graft apply` | Apply pending migrations |
| `graft status` | Show migration status |
| `graft pull` | Extract schema from existing database |
| `graft export [format]` | Export database (JSON, CSV, SQLite) |
| `graft reset` | Reset database |
| `graft gen` | Generate code |
| `graft raw <sql>` | Execute raw SQL |

## Configuration

Graft uses `graft.config.json`:

```json
{
  "version": "2",
  "schema_path": "db/schema/schema.sql",
  "queries": "db/queries/",
  "migrations_path": "db/migrations",
  "export_path": "db/export",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "go": {
      "sql_package": "pgx/v5"
    }
  }
}
```

## JavaScript Code Generation

Generate type-safe JavaScript database clients:

```bash
graft gen --js
```

This creates:
- `types.js` - JSDoc type definitions
- `queries.js` - Query methods
- `database.js` - Database client
- `package.json` - Dependencies

### Usage Example

```javascript
const { New } = require('./graft_gen/database');
const { Pool } = require('pg');

const pool = new Pool({
  connectionString: process.env.DATABASE_URL
});

const db = New(pool);

// Use generated queries
const user = await db.getUser(1);
const users = await db.listUsers();
```

## Programmatic API

```javascript
const graft = require('graft-orm');

// Execute commands
graft.exec('status');
graft.exec('migrate "add users"');

// Get binary path
const binaryPath = graft.getBinaryPath();
```

## Documentation

Full documentation: https://github.com/Rana718/Graft

## License

MIT License - see [LICENSE](https://github.com/Rana718/Graft/blob/main/LICENSE)
