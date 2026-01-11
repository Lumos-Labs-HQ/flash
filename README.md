<h1 align="center">⚡ Flash ORM</h1>

<p align="center">
  <a href="https://go.dev/doc/go1.23">
    <img src="https://img.shields.io/badge/Go-1.23%2B-blue.svg" alt="Go Version">
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/License-MIT-green.svg" alt="License: MIT">
  </a>
  <a href="https://github.com/Lumos-Labs-HQ/flash/releases">
    <img src="https://img.shields.io/github/v/release/Lumos-Labs-HQ/flash?label=Release" alt="Release">
  </a>
  <a href="https://www.npmjs.com/package/flashorm">
    <img src="https://img.shields.io/npm/v/flashorm?color=blue&label=npm" alt="npm version">
  </a>
  <a href="https://pypi.org/project/flashorm/">
    <img src="https://img.shields.io/pypi/v/flashorm?color=green&label=python" alt="PyPI version">
  </a>
</p>

<p align="center">
  <a href="docs/USAGE_GO.md">📗 Go Guide</a> •
  <a href="docs/USAGE_TYPESCRIPT.md">📘 TypeScript Guide</a> •
  <a href="docs/USAGE_PYTHON.md">📙 Python Guide</a> •
  <a href="RELEASE_NOTES.md">📋 Release Notes</a>
</p>

![image](.github/flash-orm.png)

---

A powerful, database-agnostic ORM built in Go that provides Prisma-like functionality with multi-database support and type-safe code generation for Go, JavaScript, and TypeScript.

## ✨ Features

- 🗃️ **Multi-Database Support**: PostgreSQL, MySQL, SQLite
- 🔄 **Migration Management**: Create, apply, and track migrations
- 🔒 **Safe Migration System**: Transaction-based execution with automatic rollback
- 📤 **Smart Export System**: Multiple formats (JSON, CSV, SQLite) for data portability
- 🔧 **Code Generation**: Generate type-safe code for Go, JavaScript/TypeScript, and Python
- 🟢 **Node.js Support**: First-class JavaScript/TypeScript support with type definitions
- 🐍 **Python Support**: Full Python with async support
- 🎨 **Enum Support**: PostgreSQL ENUM types with full migration support
- ⚡ **Blazing Fast**: Outperforms Drizzle and Prisma in benchmarks
- 🎯 **Prisma-like Commands**: Familiar CLI interface
- 🔍 **Schema Introspection**: Pull schema from existing databases
- 📊 **FlashORM Studio**: Visual database editor with table management, data editing, and auto-migration creation
- 🛡️ **Conflict Detection**: Automatic detection and resolution of migration conflicts

## 📊 Performance Benchmarks

FlashORM significantly outperforms popular ORMs in real-world scenarios:

| Operation                                  | FlashORM   | Drizzle     | Prisma      |
| ------------------------------------------ | ---------- | ----------- | ----------- |
| Insert 1000 Users                          | **149ms**  | 224ms       | 230ms       |
| Insert 10 Cat + 5K Posts + 15K Comments    | **2410ms** | 3028ms      | 3977ms      |
| Complex Query x500                         | **3156ms** | 12500ms     | 56322ms     |
| Mixed Workload x1000 (75% read, 25% write) | **186ms**  | 1174ms      | 10863ms     |
| Stress Test Simple Query x2000             | **79ms**   | 160ms       | 118ms       |
| **TOTAL**                                  | **5980ms** | **17149ms** | **71510ms** |



## 🚀 Installation

### NPM (Node.js/TypeScript Projects)

```bash
npm install -g flashorm
```

### Python Install

```bash
pip install flashorm
```

### Go Install

```bash
go install github.com/Lumos-Labs-HQ/flash@latest
```

### From Source

```bash
git clone https://github.com/Lumos-Labs-HQ/flash.git
cd flash
make build-all
```

### Download Binary

Download the latest binary from [Releases](https://github.com/Lumos-Labs-HQ/flash/releases).

# FlashORM - Database ORM

## 🏁 Quick Start

### 1. Initialize Your Project

```bash
cd your-project
flash init --postgresql  # or --mysql, --sqlite
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
flash migrate "create users table"
```

### 4. Apply Migrations Safely

```bash
flash apply
```

### 5. Check Status

```bash
flash status
```

## 📋 Commands

| Command                 | Description                                         |
| ----------------------- | --------------------------------------------------- |
| `flash init`            | Initialize project with database-specific templates |
| `flash migrate <name>`  | Create a new migration file                         |
| `flash apply`           | Apply pending migrations with transaction safety    |
| `flash down`            | Rollback the last applied migration                 |
| `flash status`          | Show migration status                               |
| `flash pull`            | Extract schema from existing database               |
| `flash studio`          | Launch visual database editor                       |
| `flash export [format]` | Export database (JSON, CSV, SQLite)                 |
| `flash seed`            | Seed database with realistic fake data              |
| `flash reset`           | Reset database (⚠️ destructive)                     |
| `flash gen`             | Generate type-safe code                             |
| `flash raw <sql>`       | Execute raw SQL                                     |

### Global Flags

- `--force` - Skip confirmation prompts
- `--help` - Show help

## 🗄️ Database Support

### PostgreSQL

```bash
flash init --postgresql
export DATABASE_URL="postgres://user:pass@localhost:5432/db"
```

### MySQL

```bash
flash init --mysql
export DATABASE_URL="user:pass@tcp(localhost:3306)/db"
```

### SQLite

```bash
flash init --sqlite
export DATABASE_URL="sqlite://./database.db"
```

## 🔧 Configuration

FlashORM uses `flash.config.json` for configuration:

```json
{
  "version": "2",
  "schema_dir": "db/schema",
  "queries": "db/queries/",
  "migrations_path": "db/migrations",
  "export_path": "db/export",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "go": { "enabled": true },
    "js": { "enabled": true },
    "python": { "enabled": true }
  }
}
```

### Schema Folder Support

Organize schemas in separate files for better maintainability:

```
db/schema/
├── users.sql
├── posts.sql
└── comments.sql
```

Set `schema_dir` in config to enable this feature.

## 📁 Project Structure

After running `flash init`:

```
your-project/
├── flash.config.json      # FlashORM configuration
├── .env                  # Environment variables
└── db/
    ├── schema/
    │   └── schema.sql    # Database schema
    ├── queries/
    │   └── users.sql     # SQL queries for SQLC
    ├── migrations/       # Migration files (auto-created)
    └── export/          # Export files (auto-created)
```

## 🔒 Safe Migration System

### Transaction-Based Execution

Each migration runs in its own transaction with automatic rollback on failure:

```bash
flash apply
```

Output:

```
📦 Applying 2 migration(s)...
  [1/2] 20251021132902_init
      ✅ Applied
  [2/2] 20251021140530_add_users_index
      ✅ Applied
✅ All migrations applied successfully
```

### Error Handling

If a migration fails, the transaction is automatically rolled back:

```
📦 Applying 2 migration(s)...
  [1/2] 20251021132902_init
      ✅ Applied
  [2/2] 20251021140530_bad_migration
❌ Failed at migration: 20251021140530_bad_migration
   Error: syntax error at or near "INVALID"
   Transaction rolled back. Fix the error and run 'flash apply' again.
```

## 🔄 Migration Workflow

### 1. Create Migration

```bash
flash migrate "add user roles"
```

Creates a timestamped SQL file with Up and Down sections:

```sql
-- Migration: add_user_roles
-- Created: 2025-10-21T13:29:02Z

-- +migrate Up
ALTER TABLE users ADD COLUMN role VARCHAR(50) DEFAULT 'user';
CREATE INDEX idx_users_role ON users(role);

-- +migrate Down
DROP INDEX idx_users_role;
ALTER TABLE users DROP COLUMN role;
```

### 2. Apply Migrations

```bash
flash apply
```

### 3. Check Status

```bash
flash status
```

Output:

```
Database: Connected ✅
Migrations: 3 total, 2 applied, 1 pending

┌─────────────────────────────────┬─────────┬─────────────────────┐
│ Migration                       │ Status  │ Applied At          │
├─────────────────────────────────┼─────────┼─────────────────────┤
│ 20251021_create_users_table     │ Applied │ 2025-10-21 13:29:02 │
│ 20251021_add_user_email_index   │ Applied │ 2025-10-21 13:30:15 │
│ 20251021_add_user_roles         │ Pending │ -                   │
└─────────────────────────────────┴─────────┴─────────────────────┘
```

### 4. Rollback Migration

```bash
flash down
```

Rolls back the last applied migration using the `-- +migrate Down` section.

## 📊 Studio (Visual Database Editor)

FlashORM Studio provides powerful visual interfaces for database management:

### SQL Studio (PostgreSQL, MySQL, SQLite)

**Features:**
- 📋 **Visual Schema Designer**: View and edit database schema with interactive diagrams
- ➕ **Add Tables**: Create new tables with columns, constraints, and relationships
- ✏️ **Edit Tables**: Modify existing table structures, add/remove columns
- 🔗 **Relationship Management**: Visualize and manage foreign key relationships
- 📝 **Data Browser**: View, edit, insert, and delete records with a spreadsheet-like interface
- 🔄 **Auto-Migration Creation**: Automatically generates migration files from schema changes
- 🎨 **Enhanced UI**: Improved visibility with better contrast and larger interactive elements

**Usage:**

```bash
# Start Studio with config file (auto-detects database)
flash studio

# Start Studio with direct database connection
flash studio --db "postgresql://user:pass@localhost:5432/mydb"

# Custom port
flash studio --port 3000
```

### 🍃 MongoDB Studio

A beautiful, modern interface for MongoDB - similar to MongoDB Compass!

**Usage:**
```bash
flash studio --db "mongodb://localhost:27017/mydb"
# or with MongoDB Atlas
flash studio --db "mongodb+srv://user:pass@cluster.mongodb.net/mydb"
```

**Features:**
- 📋 **Collection Browser** - View all collections with document counts
- 📄 **Document Viewer** - Browse documents with syntax-highlighted JSON
- ✏️ **Inline Editing** - Edit documents directly with JSON validation
- ➕ **Create Documents** - Add new documents with JSON editor
- 🗑️ **Delete Documents** - Remove documents with confirmation
- 🔍 **Search & Filter** - Query documents using MongoDB filter syntax
- 📊 **Database Stats** - View connection info and statistics
- 📋 **Copy as JSON** - One-click copy of any document

### 🔴 Redis Studio

A powerful Redis management interface with a real CLI terminal - inspired by Upstash!

**Usage:**
```bash
flash studio --redis "redis://localhost:6379"
# or with password
flash studio --redis "redis://:password@localhost:6379"
```

**Features:**
- 🗂️ **Key Browser** - View keys with type indicators (STRING, LIST, SET, HASH, ZSET)
- 🔍 **Pattern Search** - Search keys with wildcards (e.g., `user:*`)
- ➕ **Create Keys** - Add new keys of any Redis type
- ⏰ **TTL Management** - View and set key expiration
- 💻 **Real CLI Terminal** - Full Redis CLI with command history (↑↓ arrows)
- 📊 **Statistics** - Memory usage, connected clients, server info
- 🗄️ **Database Selector** - Switch between db0-db15
- 🧹 **Purge Database** - Clear all keys with one click

**CLI Examples:**
```
redis> SET mykey "hello"
OK
redis> GET mykey
"hello"
redis> HSET user:1 name "John" age 30
(integer) 2
redis> MEMORY STATS
peak.allocated: 1048576
...
```

### Studio Workflow

1. **View Schema**: Interactive diagram shows all tables and relationships
2. **Edit Schema**: Click tables to modify structure, add columns, or change types
3. **Manage Data**: Browse and edit data directly in the Studio interface
4. **Generate Migrations**: Changes automatically create migration files for version control

### Troubleshooting

- Database connection errors: verify `DATABASE_URL` and network access.
- Migration failures: inspect the migration SQL file, fix and re-run `flash apply`.

## 📤 Export System

Export your database to multiple formats for portability and analysis:

### JSON Export (Default)

```bash
flash export
# or
flash export --json
```

Creates structured JSON with metadata:

```json
{
  "timestamp": "2025-10-21 14:00:07",
  "version": "1.0",
  "comment": "Database export",
  "tables": {
    "users": [{ "id": 1, "name": "Alice", "email": "alice@example.com" }],
    "posts": [{ "id": 1, "user_id": 1, "title": "Hello World" }]
  }
}
```

### CSV Export

```bash
flash export --csv
```

Creates directory with individual CSV files per table:

```
db/export/export_2025-10-21_14-00-07_csv/
├── users.csv
├── posts.csv
└── comments.csv
```

### SQLite Export

```bash
flash export --sqlite
```

Creates portable SQLite database file:

```
db/export/export_2025-10-21_14-00-07.db
```

## 🔗 SQLC Integration

Generate type-safe Go code from SQL:

```bash
# Generate types after migrations
flash gen

# Apply migrations and generate types
flash apply && flash gen
```

Example generated code:

```go
type User struct {
    ID        int32     `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

func (q *Queries) GetUser(ctx context.Context, id int32) (User, error) {
    // Generated implementation
}
```

## 🌱 Database Seeding

Populate your database with realistic fake data for development and testing:

### Basic Usage

```bash
# Seed all tables with default count (10 rows each)
flash seed

# Seed with custom count
flash seed --count 100

# Seed specific table
flash seed --table users --count 50

# Truncate tables before seeding
flash seed --truncate
```

### Features

- **Smart Data Generation**: Automatically generates realistic data based on column names and types
  - `email` columns → realistic emails
  - `name` columns → human names
  - `phone` columns → phone numbers
  - `created_at` columns → timestamps
  - And many more patterns
- **Foreign Key Support**: Respects relationships, seeds parent tables first
- **Dependency Graph**: Automatically determines correct insertion order
- **All Databases**: Works with PostgreSQL, MySQL, and SQLite

### Example Output

```
🌱 Seeding database...
  ✅ users: 50 records
  ✅ posts: 100 records
  ✅ comments: 200 records
✨ Seeding complete!
```

## 🛠️ Advanced Usage

### Production Deployment

```bash
# Deploy without interactive prompts
flash apply --force

# Create export before deployment
flash export --json
flash apply --force
```

### Development Workflow

```bash
# Reset database during development
flash reset --force

# Extract schema from existing database
flash pull
```

### Documentation
```bash
cd docs
npm install
npm run docs:dev
```

### Smart Pull Feature

When you run `flash pull`, FlashORM intelligently manages your schema files:
- **New tables**: Creates new `.sql` files in your schema directory
- **Modified tables**: Updates existing schema files
- **Dropped tables**: Comments out the schema file (preserves history)

This ensures your schema files always reflect the actual database state while maintaining version control history.

### Raw SQL Execution

```bash
# Execute raw SQL
flash raw "SELECT COUNT(*) FROM users;"

# Execute SQL file
flash raw scripts/cleanup.sql
```

<!-- ## 🚀 Roadmap & Future Features -->

<!-- ### Coming Soon
- 🐍 **Python Support**: Use FlashORM with Python projects -->

## 🐛 Troubleshooting

### Common Issues

**Database Connection Failed**

```bash
Error: failed to connect to database
```

- Check your `DATABASE_URL` environment variable
- Verify database is running and accessible
- Check firewall and network settings

**Migration Failed with Rollback**

```bash
❌ Failed at migration: 20251021140530_bad_migration
   Transaction rolled back. Fix the error and run 'flash apply' again.
```

- Check the migration SQL syntax
- Verify table/column names exist
- Fix the migration file and run `flash apply` again

## 🤝 Contributing

We welcome contributions! Here's how to get started:

```bash
git clone https://github.com/Lumos-Labs-HQ/flash.git
cd flash

make dev-setup

make build-all
```

### Development Guidelines

- Follow Go conventions and best practices
- Add tests for new features
- Update documentation
- Use conventional commit messages
- Test migration safety features

See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for detailed guidelines.

## 📚 Documentation

- [Go Usage Guide](docs/USAGE_GO.md) - Complete guide for Go developers
- [TypeScript Usage Guide](docs/USAGE_TYPESCRIPT.md) - Complete guide for JS/TS developers
- [Python Usage Guide](docs/USAGE_PYTHON.md) - Complete guide for Python developers
- [How It Works](docs/HOW_IT_WORKS.md) - Technical deep dive
- [Technology Stack](docs/TECHNOLOGY_STACK.md) - Libraries and tools used
- [Contributing](docs/CONTRIBUTING.md) - How to contribute

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [Prisma](https://www.prisma.io/) migration system
- Built with [Cobra](https://github.com/spf13/cobra) CLI framework
- Database drivers: [pgx](https://github.com/jackc/pgx), [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql), [go-sqlite3](https://github.com/mattn/go-sqlite3)

---
