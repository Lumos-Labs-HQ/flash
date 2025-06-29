# Graft - Database Migration CLI Tool

Graft is a Go-based CLI tool that provides database migration capabilities similar to Prisma, with support for schema comparison, backup management, and optional SQLC integration.

## Features

🔧 **Core Capabilities:**
- Project-aware configuration management
- Automatic project root detection
- Support for JSON and YAML configuration files
- Database-agnostic design (currently supports PostgreSQL)

🗃️ **Migration Management:**
- Track migrations in local files and database table
- Compare and validate schema changes
- Automatic backup prompts for destructive operations
- Checksum validation for migration integrity

💬 **Prisma-like Commands:**
- `graft migrate` - Create new migrations
- `graft apply` - Apply pending migrations
- `graft deploy` - Record migrations without execution
- `graft reset` - Drop all data and re-apply migrations
- `graft status` - Show migration status
- `graft sqlc-migrate` - Apply migrations + run SQLC

🔁 **SQLC Integration:**
- Automatic `sqlc generate` after migrations
- Configurable SQLC config path
- Seamless Go type generation

🔒 **Backup System:**
- Automatic backup prompts for destructive operations
- JSON-based backup format
- Timestamped backup directories
- Complete table data preservation

## Installation

### From Source

```bash
git clone <repository-url>
cd graft
go build -o graft .
```

### Using Go Install

```bash
go install github.com/your-username/graft@latest
```

## Quick Start

### 1. Initialize Graft in Your Project

```bash
cd your-project
graft init
```

This creates:
- `graft.config.json` - Configuration file
- `migrations/` - Migration files directory
- `db_backup/` - Backup directory
- `db/` - Schema directory

### 2. Configure Database Connection

Set your database URL as an environment variable:

```bash
export DATABASE_URL="postgres://user:password@localhost:5432/mydb"
```

### 3. Create Your First Migration

```bash
graft migrate "create users table"
```

Edit the generated migration file:

```sql
-- Migration: create users table
-- Created at: 2024-01-15 10:30:00

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 4. Apply Migrations

```bash
graft apply
```

### 5. Check Status

```bash
graft status
```

## Configuration

### Example `graft.config.json`

```json
{
  "schema_path": "db/schema.sql",
  "migrations_path": "migrations",
  "sqlc_config_path": "sqlc.yaml",
  "backup_path": "db_backup",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  }
}
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `schema_path` | Path to schema file | `db/schema.sql` |
| `migrations_path` | Directory for migration files | `migrations` |
| `sqlc_config_path` | Path to SQLC config (optional) | `""` |
| `backup_path` | Directory for backups | `db_backup` |
| `database.provider` | Database provider | `postgresql` |
| `database.url_env` | Environment variable for DB URL | `DATABASE_URL` |

## Commands

### `graft init`
Initialize graft in the current project.

```bash
graft init
```

### `graft migrate [name]`
Create a new migration file.

```bash
graft migrate "add user roles"
graft migrate  # Interactive mode
```

### `graft apply`
Apply all pending migrations.

```bash
graft apply
graft apply --force  # Skip confirmations
```

### `graft deploy`
Record migrations as applied without executing them.

```bash
graft deploy
```

### `graft reset`
Drop all data and re-apply all migrations.

```bash
graft reset
graft reset --force  # Skip confirmations and backup
```

### `graft status`
Show current migration status.

```bash
graft status
```

### `graft sqlc-migrate`
Apply migrations and run SQLC generate.

```bash
graft sqlc-migrate
```

## SQLC Integration

If you're using SQLC for Go code generation, graft can automatically run `sqlc generate` after applying migrations.

1. Set `sqlc_config_path` in your config:

```json
{
  "sqlc_config_path": "sqlc.yaml"
}
```

2. Use `graft sqlc-migrate` or `graft apply` - both will run SQLC automatically.

## Backup System

Graft automatically prompts for backups before destructive operations:

- **Backup Location**: `db_backup/YYYY-MM-DD_HHMMSS/backup.json`
- **Format**: JSON with complete table data
- **Automatic Prompts**: Before `reset` and schema conflicts

### Backup Structure

```json
{
  "timestamp": "2024-01-15 10:30:00",
  "tables": [
    {
      "table_name": "users",
      "columns": ["id", "name", "email", "created_at"],
      "rows": [
        {"id": 1, "name": "John", "email": "john@example.com", "created_at": "2024-01-15T10:00:00Z"}
      ]
    }
  ]
}
```

## Migration Files

Migration files are stored in the `migrations/` directory with timestamp-based naming:

```
migrations/
├── 20240115103000_create_users_table.sql
├── 20240115104500_add_user_roles.sql
└── 20240115110000_create_posts_table.sql
```

### Migration File Format

```sql
-- Migration: create users table
-- Created at: 2024-01-15 10:30:00

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Database Support

Currently supported databases:
- ✅ PostgreSQL

Planned support:
- 🔄 MySQL
- 🔄 SQLite
- 🔄 SQL Server

## Best Practices

1. **Always Review Migrations**: Check generated migration files before applying
2. **Use Descriptive Names**: Make migration names clear and descriptive
3. **Test Migrations**: Test migrations on development databases first
4. **Backup Production**: Always backup production databases before migrations
5. **Version Control**: Commit migration files to version control
6. **Sequential Migrations**: Apply migrations in order, don't skip

## Troubleshooting

### Common Issues

**Database Connection Failed**
```bash
Error: failed to connect to database: database URL not found in environment variable DATABASE_URL
```
Solution: Set the `DATABASE_URL` environment variable.

**Migration Already Exists**
```bash
Error: migration validation failed: migration already exists
```
Solution: Check for duplicate migration names or use `graft status` to see applied migrations.

**SQLC Generate Failed**
```bash
⚠️ SQLC generate failed: exec: "sqlc": executable file not found in $PATH
```
Solution: Install SQLC or remove `sqlc_config_path` from config.

### Debug Mode

Use the `--force` flag to skip confirmations during development:

```bash
graft apply --force
graft reset --force
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Acknowledgments

- Inspired by [Prisma](https://www.prisma.io/) migration system
- Built with [Cobra](https://github.com/spf13/cobra) CLI framework
- Uses [Viper](https://github.com/spf13/viper) for configuration management
