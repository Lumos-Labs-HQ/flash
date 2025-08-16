# Graft - Database Migration CLI Tool

A powerful, database-agnostic migration CLI tool built in Go that provides Prisma-like functionality with multi-database support and SQLC integration.

## ✨ Features

- 🗃️ **Multi-Database Support**: PostgreSQL, MySQL, SQLite
- 🔄 **Migration Management**: Create, apply, and track migrations
- 💾 **Smart Backup System**: Automatic backups before destructive operations
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

### 4. Apply Migrations
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
| `graft apply` | Apply pending migrations |
| `graft status` | Show migration status |
| `graft backup [comment]` | Create database backup |
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
  "backup_path": "db/backup",
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
    └── backup/          # Backup files (auto-created)
```

## 🔄 Migration Workflow

### 1. Create Migration
```bash
graft migrate "add user roles"
```

Creates a timestamped SQL file:
```sql
-- Migration: 20240816060520_add_user_roles
-- Created: 2024-08-16T06:05:20Z

-- Up Migration
ALTER TABLE users ADD COLUMN role VARCHAR(50) DEFAULT 'user';
CREATE INDEX idx_users_role ON users(role);

-- Down Migration
DROP INDEX IF EXISTS idx_users_role;
ALTER TABLE users DROP COLUMN role;
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
│ 20240816_create_users_table     │ Applied │ 2024-08-16 06:00:00 │
│ 20240816_add_user_email_index   │ Applied │ 2024-08-16 06:01:00 │
│ 20240816_add_user_roles         │ Pending │ -                   │
└─────────────────────────────────┴─────────┴─────────────────────┘
```

## 💾 Backup System

Automatic backups before destructive operations:

```bash
# Manual backup
graft backup "before major update"

# Restore from backup
graft restore db/backup/backup_2024-08-16_06-05-20.json
```

Backup format (JSON):
```json
{
  "timestamp": "2024-08-16_06-05-20",
  "comment": "before major update",
  "tables": {
    "users": {
      "columns": ["id", "name", "email"],
      "data": [
        {"id": 1, "name": "John", "email": "john@example.com"}
      ]
    }
  }
}
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

# Create backup before deployment
graft backup "pre-deployment $(date)"
graft apply --force
```

### Development Workflow
```bash
# Reset database during development
graft reset --force

# Create backup before major changes
graft backup "before refactoring"
```

### Raw SQL Execution
```bash
# Execute raw SQL
graft raw "SELECT COUNT(*) FROM users;"

# Execute SQL file
graft raw --file scripts/cleanup.sql
```

## 🚀 Roadmap & Future Features

### Coming Soon
- 🔍 **Schema Introspection (`graft pull`)**: Extract schema from existing databases
- 📊 **Migration Rollback**: Rollback applied migrations
- 🔄 **Migration Squashing**: Combine multiple migrations
- 📈 **Performance Monitoring**: Track migration performance
- 🌐 **Remote Schema Sync**: Sync with remote databases

## 🐛 Troubleshooting

### Common Issues

**Database Connection Failed**
```bash
Error: failed to connect to database
```
- Check your `DATABASE_URL` environment variable
- Verify database is running and accessible
- Check firewall and network settings

**Migration Already Applied**
```bash
Error: migration already applied
```
- Use `graft status` to check migration state
- Use `graft reset` to start fresh (⚠️ destructive)

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

See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for detailed guidelines.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [Prisma](https://www.prisma.io/) migration system
- Built with [Cobra](https://github.com/spf13/cobra) CLI framework
- Database drivers: [pgx](https://github.com/jackc/pgx), [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql), [go-sqlite3](https://github.com/mattn/go-sqlite3)

---
