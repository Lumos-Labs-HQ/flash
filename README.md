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
| `flash status`          | Show migration status                               |
| `flash pull`            | Extract schema from existing database               |
| `flash studio`          | flash studio                                        |
| `flash export [format]` | Export database (JSON, CSV, SQLite)                 |
| `flash reset`           | Reset database (⚠️ destructive)                     |
| `flash gen`             | Generate SQLC types                                 |
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
  "schema_path": "db/schema/schema.sql",
  "queries": "db/queries/",
  "migrations_path": "db/migrations",
  "export_path": "db/export",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "js": {
      "enabled": true
    }
  }
}
```

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

Creates a timestamped SQL file:

```sql
-- Migration: add_user_roles
-- Created: 2025-10-21T13:29:02Z

ALTER TABLE users ADD COLUMN role VARCHAR(50) DEFAULT 'user';
CREATE INDEX idx_users_role ON users(role);
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

## 📊 Studio (Visual Database Editor)

FlashORM Studio provides a powerful visual interface for database management:

### Features

- 📋 **Visual Schema Designer**: View and edit database schema with interactive diagrams
- ➕ **Add Tables**: Create new tables with columns, constraints, and relationships
- ✏️ **Edit Tables**: Modify existing table structures, add/remove columns
- 🔗 **Relationship Management**: Visualize and manage foreign key relationships
- 📝 **Data Browser**: View, edit, insert, and delete records with a spreadsheet-like interface
- 🔄 **Auto-Migration Creation**: Automatically generates migration files from schema changes
- 🎨 **Enhanced UI**: Improved visibility with better contrast and larger interactive elements

### Usage

**Start Studio with config file** (auto-detects database):

```bash
flash studio
```

**Start Studio with direct database connection**:

```bash
flash studio --db "postgresql://jack:secret123@localhost:5432/mydb"
```

**Custom port**:

```bash
flash studio --port 3000
```

Open http://localhost:5555 by default (or your specified port).

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

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [Prisma](https://www.prisma.io/) migration system
- Built with [Cobra](https://github.com/spf13/cobra) CLI framework
- Database drivers: [pgx](https://github.com/jackc/pgx), [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql), [go-sqlite3](https://github.com/mattn/go-sqlite3)

---
