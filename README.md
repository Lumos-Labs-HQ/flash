# FlashORM - Database ORM

A powerful, database-agnostic ORM built in Go that provides Prisma-like functionality with multi-database support and type-safe code generation for Go, JavaScript, and TypeScript.

## ✨ Features

- 🗃️ **Multi-Database Support**: PostgreSQL, MySQL, SQLite
- 🔄 **Migration Management**: Create, apply, and track migrations
- 🔒 **Safe Migration System**: Transaction-based execution with automatic rollback
- 📤 **Smart Export System**: Multiple formats (JSON, CSV, SQLite) for data portability
- 🔧 **Code Generation**: Generate type-safe code for Go and JavaScript/TypeScript
- 🟢 **Node.js Support**: First-class JavaScript/TypeScript support with type definitions
- 🎨 **Enum Support**: PostgreSQL ENUM types with full migration support
- ⚡ **Blazing Fast**: Outperforms Drizzle and Prisma in benchmarks
- 🎯 **Prisma-like Commands**: Familiar CLI interface
- 🔍 **Schema Introspection**: Pull schema from existing databases
- 📊 **FlashORM Studio**: similar to Prisma Studio, where users can view and edit data visually
- 🛡️ **Conflict Detection**: Automatic detection and resolution of migration conflicts

## 📊 Performance Benchmarks

FlashORM significantly outperforms popular ORMs in real-world scenarios:

| Operation | FlashORM | Drizzle | Prisma |
|-----------|-------|---------|--------|
| Insert 1000 Users | **158ms** | 224ms | 230ms |
| Insert 10 Cat + 5K Posts + 15K Comments | **2410ms** | 3028ms | 3977ms |
| Complex Query x500 | **4071ms** | 12500ms | 56322ms |
| Mixed Workload x1000 (75% read, 25% write) | **186ms** | 1174ms | 10863ms |
| Stress Test Simple Query x2000 | **122ms** | 160ms | 223ms |
| **TOTAL** | **6947ms** | **17149ms** | **71551ms** |

*Benchmarks run on PostgreSQL with identical schemas and queries. FlashORM is **2.5x faster** than Drizzle and **10x faster** than Prisma.*

## 🚀 Installation

### NPM (Node.js/TypeScript Projects)
```bash
npm install -g FlashORM-orm
```

### Go Install
```bash
go install github.com/Lumos-Labs-HQ/FlashORM@latest
```

### From Source
```bash
git clone https://github.com/Lumos-Labs-HQ/FlashORM.git
cd FlashORM
make build-all
```

### Download Binary
Download the latest binary from [Releases](https://github.com/Lumos-Labs-HQ/FlashORM/releases).
# FlashORM - Database ORM

## 🏁 Quick Start

### 1. Initialize Your Project
```bash
cd your-project
FlashORM init --postgresql  # or --mysql, --sqlite
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
FlashORM migrate "create users table"
```

### 4. Apply Migrations Safely
```bash
FlashORM apply
```

### 5. Check Status
```bash
FlashORM status
```

## 📋 Commands

| Command | Description |
|---------|-------------|
| `FlashORM init` | Initialize project with database-specific templates |
| `FlashORM migrate <name>` | Create a new migration file |
| `FlashORM apply` | Apply pending migrations with transaction safety |
| `FlashORM status` | Show migration status |
| `FlashORM pull` | Extract schema from existing database |
| `FlashORM studio` | 
| `FlashORM export [format]` | Export database (JSON, CSV, SQLite) |
| `FlashORM reset` | Reset database (⚠️ destructive) |
| `FlashORM gen` | Generate SQLC types |
| `FlashORM raw <sql>` | Execute raw SQL |

### Global Flags
- `--force` - Skip confirmation prompts
- `--help` - Show help

## 🗄️ Database Support

### PostgreSQL
```bash
FlashORM init --postgresql
export DATABASE_URL="postgres://user:pass@localhost:5432/db"
```

### MySQL
```bash
FlashORM init --mysql
export DATABASE_URL="user:pass@tcp(localhost:3306)/db"
```

### SQLite
```bash
FlashORM init --sqlite
export DATABASE_URL="sqlite://./database.db"
```

## 🔧 Configuration

FlashORM uses `FlashORM.config.json` for configuration:

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

After running `FlashORM init`:

```
your-project/
├── FlashORM.config.json      # FlashORM configuration
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
FlashORM apply
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
   Transaction rolled back. Fix the error and run 'FlashORM apply' again.
```

## 🔄 Migration Workflow

### 1. Create Migration
```bash
FlashORM migrate "add user roles"
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
FlashORM apply
```

### 3. Check Status
```bash
FlashORM status
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

## Studio (visual editor)

Start the optional Studio UI:

```bash
FlashORM studio
```

For open FlashORM studio without projct init

```bash
FlashORM studio --db "postgresql://jack:secret123@localhost:5432/mydb"
```

Open http://localhost:5555 by default (or the port you pass with `--port`).

### Troubleshooting

- Database connection errors: verify `DATABASE_URL` and network access.
- Migration failures: inspect the migration SQL file, fix and re-run `FlashORM apply`.


## 📤 Export System

Export your database to multiple formats for portability and analysis:

### JSON Export (Default)
```bash
FlashORM export
# or
FlashORM export --json
```

Creates structured JSON with metadata:
```json
{
  "timestamp": "2025-10-21 14:00:07",
  "version": "1.0",
  "comment": "Database export",
  "tables": {
    "users": [
      {"id": 1, "name": "Alice", "email": "alice@example.com"}
    ],
    "posts": [
      {"id": 1, "user_id": 1, "title": "Hello World"}
    ]
  }
}
```

### CSV Export
```bash
FlashORM export --csv
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
FlashORM export --sqlite
```

Creates portable SQLite database file:
```
db/export/export_2025-10-21_14-00-07.db
```

## 🔗 SQLC Integration

Generate type-safe Go code from SQL:

```bash
# Generate types after migrations
FlashORM gen

# Apply migrations and generate types
FlashORM apply && FlashORM gen
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
FlashORM apply --force

# Create export before deployment
FlashORM export --json
FlashORM apply --force
```

### Development Workflow
```bash
# Reset database during development
FlashORM reset --force

# Extract schema from existing database
FlashORM pull
```

### Raw SQL Execution
```bash
# Execute raw SQL
FlashORM raw "SELECT COUNT(*) FROM users;"

# Execute SQL file
FlashORM raw scripts/cleanup.sql
```

## 🚀 Roadmap & Future Features

### Coming Soon
- 🐍 **Python Support**: Use FlashORM with Python projects

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
   Transaction rolled back. Fix the error and run 'FlashORM apply' again.
```
- Check the migration SQL syntax
- Verify table/column names exist
- Fix the migration file and run `FlashORM apply` again

## 🤝 Contributing

We welcome contributions! Here's how to get started:

```bash
git clone https://github.com/Lumos-Labs-HQ/FlashORM.git
cd FlashORM

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