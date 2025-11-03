# Graft - Database Migration CLI Tool

A powerful, database-agnostic migration CLI tool built in Go that provides Prisma-like functionality with multi-database support and SQLC integration.

## ✨ Features

- 🗃️ **Multi-Database Support**: PostgreSQL, MySQL, SQLite
- 🔄 **Migration Management**: Create, apply, and track migrations
- 🔒 **Safe Migration System**: Transaction-based execution with automatic rollback
- 📤 **Smart Export System**: Multiple formats (JSON, CSV, SQLite) for data portability
- 🔧 **SQLC Integration**: Generate Go types from SQL schemas
- ⚡ **Fast & Reliable**: Built in Go for performance and reliability
- 🎯 **Prisma-like Commands**: Familiar CLI interface

## 🚀 Installation

### Using Go Install (Recommended)
```bash
go install github.com/Rana718/Graft@latest
```

### From Source
```bash
git clone https://github.com/Rana718/Graft.git
cd Graft
make build-all
# Binary will be in build/ directory
```

### Download Binary
Download the latest binary from [Releases](https://github.com/Rana718/Graft/releases) for your platform.

## 🏁 Quick Start

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

### 4. Apply Migrations Safely
```bash
graft apply
```

### 5. Check Status
```bash
graft status
```

## 📋 Commands

| Command | Description |
|---------|-------------|
| `graft init` | Initialize project with database-specific templates |
| `graft migrate <name>` | Create a new migration file |
| `graft apply` | Apply pending migrations with transaction safety |
| `graft status` | Show migration status |
| `graft pull` | Extract schema from existing database |
| `graft export [format]` | Export database (JSON, CSV, SQLite) |
| `graft reset` | Reset database (⚠️ destructive) |
| `graft gen` | Generate SQLC types |
| `graft raw <sql>` | Execute raw SQL |

### Global Flags
- `--config` - Specify config file path
- `--force` - Skip confirmation prompts
- `--help` - Show help

## 🗄️ Database Support

### PostgreSQL
```bash
graft init --postgresql
export DATABASE_URL="postgres://user:pass@localhost:5432/db"
```

### MySQL
```bash
graft init --mysql
export DATABASE_URL="user:pass@tcp(localhost:3306)/db"
```

### SQLite
```bash
graft init --sqlite
export DATABASE_URL="sqlite://./database.db"
```

## 🔧 Configuration

Graft uses `graft.config.json` for configuration:

```json
{
  "schema_path": "db/schema/schema.sql",
  "migrations_path": "db/migrations",
  "sqlc_config_path": "sqlc.yml",
  "export_path": "db/export",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  }
}
```

## 📁 Project Structure

After running `graft init`:

```
your-project/
├── graft.config.json      # Graft configuration
├── sqlc.yml              # SQLC configuration
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
graft apply
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
   Transaction rolled back. Fix the error and run 'graft apply' again.
```

## 🔄 Migration Workflow

### 1. Create Migration
```bash
graft migrate "add user roles"
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
graft apply
```

### 3. Check Status
```bash
graft status
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

## 📤 Export System

Export your database to multiple formats for portability and analysis:

### JSON Export (Default)
```bash
graft export
# or
graft export --json
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
graft export --csv
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
graft export --sqlite
```

Creates portable SQLite database file:
```
db/export/export_2025-10-21_14-00-07.db
```

## 🔗 SQLC Integration

Generate type-safe Go code from SQL:

```bash
# Generate types after migrations
graft gen

# Apply migrations and generate types
graft apply && graft gen
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
graft apply --force

# Create export before deployment
graft export --json
graft apply --force
```

### Development Workflow
```bash
# Reset database during development
graft reset --force

# Extract schema from existing database
graft pull
```

### Raw SQL Execution
```bash
# Execute raw SQL
graft raw "SELECT COUNT(*) FROM users;"

# Execute SQL file
graft raw scripts/cleanup.sql
```

## 🚀 Roadmap & Future Features

### Coming Soon
- 🟨 **JavaScript/TypeScript Support**: Use Graft with Node.js projects
- 🐍 **Python Support**: Use Graft with Python projects

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
   Transaction rolled back. Fix the error and run 'graft apply' again.
```
- Check the migration SQL syntax
- Verify table/column names exist
- Fix the migration file and run `graft apply` again

**SQLC Not Found**
```bash
Error: sqlc not found in PATH
```
- Install SQLC: https://docs.sqlc.dev/en/latest/overview/install.html
- Or remove `sqlc_config_path` from config

## 🤝 Contributing

We welcome contributions! Here's how to get started:

```bash
git clone https://github.com/Rana718/Graft.git
cd Graft

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

## 🎨 Graft Studio

Visual database editor similar to Prisma Studio. View, edit, and manage your database through an intuitive web interface.

```bash
# Start Graft Studio
graft studio

# Custom port
graft studio --port 3000
```

**Features:**
- 📊 Browse all tables and data
- ✏️ Inline editing (double-click cells)
- 💾 Batch save changes
- ➕ Add new records
- 🗑️ Delete records
- 🔍 Search tables
- 📄 Pagination for large datasets

See [web/studio/README.md](web/studio/README.md) for more details.

