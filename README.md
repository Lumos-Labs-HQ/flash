# Graft - Database Migration CLI Tool

Graft is a Go-based CLI tool that provides database migration capabilities similar to Prisma, with support for schema comparison, backup management, and optional SQLC integration.

## Features

🔧 **Core Capabilities:**
- Project-aware configuration management
- Automatic project root detection
- Support for JSON configuration files
- Database-agnostic design (currently supports PostgreSQL)

🗃️ **Migration Management:**
- Track migrations in local files and database table
- Compare and validate schema changes
- Automatic backup prompts for destructive operations
- Checksum validation for migration integrity

💬 **Prisma-like Commands:**
- `graft init` - Initialize project
- `graft migrate` - Create new migrations
- `graft apply` - Apply pending migrations
- `graft deploy` - Deploy migrations (production)
- `graft reset` - Drop all data and re-apply migrations
- `graft status` - Show migration status
- `graft backup` - Create database backup
- `graft restore` - Restore from backup
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

### Manual Installation

1. Download the binary from releases
2. Place it in your PATH
3. Make it executable: `chmod +x graft`

## Quick Start

### 1. Initialize Graft in Your Project

```bash
cd your-project
graft init
```

This creates:
- `graft.config.json` - Configuration file
- `db/schema/` - Schema directory with example schema
- `db/queries/` - SQL queries directory for SQLC
- `sqlc.yml` - SQLC configuration file
- `.env.example` - Environment variables template

Note: Migration and backup directories (`db/migrations/`, `db/backup/`) are created automatically when needed. SQLC generated files go to `graft_gen/` (created by SQLC).

### 2. Configure Database Connection

Set your database URL as an environment variable:

```bash
export DATABASE_URL="postgres://user:password@localhost:5432/mydb"
```

Or create a `.env` file:
```bash
cp .env.example .env
# Edit .env with your database credentials
```

### 3. Create Your First Migration

```bash
graft migrate "create users table"
```

This generates a migration file in `migrations/` directory with a template:

```json
{
  "migration": {
    "id": "20240131103000_create_users_table",
    "name": "create users table",
    "up": "-- CreateTable: create users table\n-- Add your SQL commands here\n\n-- Example:\n-- CREATE TABLE users (\n--     id SERIAL PRIMARY KEY,\n--     name VARCHAR(255) NOT NULL,\n--     created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()\n-- );",
    "down": "-- Drop or alter statements to reverse the migration\n-- Add your reverse SQL commands here\n\n-- Example:\n-- DROP TABLE IF EXISTS users CASCADE;",
    "checksum": "...",
    "created_at": "2024-01-31T10:30:00Z"
  }
}
```

Edit the migration file to add your SQL:

```json
{
  "migration": {
    "id": "20240131103000_create_users_table",
    "name": "create users table",
    "up": "CREATE TABLE users (\n    id SERIAL PRIMARY KEY,\n    name VARCHAR(255) NOT NULL,\n    email VARCHAR(255) UNIQUE NOT NULL,\n    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()\n);",
    "down": "DROP TABLE IF EXISTS users CASCADE;",
    "checksum": "...",
    "created_at": "2024-01-31T10:30:00Z"
  }
}
```

### 4. Apply Migrations

```bash
graft apply
```

### 5. Check Status

```bash
graft status
```

## CLI Commands

### Global Flags

- `--config string` - Config file path (default: `./graft.config.json`)
- `--force, -f` - Skip confirmation prompts
- `--help, -h` - Show help

### `graft init`
Initialize graft in the current project.

```bash
graft init
```

**What it does:**
- Creates `graft.config.json` with default settings
- Creates required directories (`migrations/`, `db_backup/`, `db/`)
- Creates example schema file (`db/schema.sql`)
- Creates `.env.example` template

### `graft migrate [name]`
Create a new migration file.

```bash
graft migrate "add user roles"
graft migrate  # Interactive mode - prompts for name
```

**Options:**
- Provide migration name as argument or enter interactively
- Generates timestamped migration file in JSON format
- Creates template with `up` and `down` SQL sections

### `graft apply`
Apply all pending migrations with conflict detection.

```bash
graft apply
graft apply --force  # Skip confirmations
```

**What it does:**
- Checks for migration conflicts
- Prompts for backup if conflicts detected
- Applies all pending migrations in order
- Updates migration tracking table

### `graft deploy`
Deploy all pending migrations (production-ready).

```bash
graft deploy
graft deploy --force  # Skip confirmations
```

**What it does:**
- Applies all pending migrations without conflict detection
- Suitable for production environments
- Assumes migrations are pre-tested

### `graft status`
Show current migration status.

```bash
graft status
```

**Output includes:**
- Database connection status
- Total, applied, and pending migration counts
- Detailed list of each migration with status and timestamps

### `graft reset`
Drop all data and optionally remove migration files.

```bash
graft reset
graft reset --force  # Skip confirmations and backup
```

**⚠️ WARNING:** This is destructive and will delete all data!

**What it does:**
- Prompts for confirmation
- Offers to create backup before reset
- Drops all database tables
- Optionally removes migration files

### `graft backup [comment]`
Create a manual database backup.

```bash
graft backup "before major update"
graft backup "pre-production backup"
graft backup  # Uses default comment
```

**What it does:**
- Creates JSON backup of all table data
- Includes migration history
- Saves with timestamp-based filename
- Stores in configured backup directory

### `graft restore <backup-file>`
Restore database from backup file.

```bash
graft restore db_backup/backup_2024-01-31_10-30-00.json
graft restore --force backup.json  # Skip confirmations
```

**⚠️ WARNING:** This will overwrite all existing data!

**What it does:**
- Prompts for confirmation
- Restores all table data from backup
- Restores migration history
- Uses database transactions for consistency

### `graft sqlc-migrate`
Apply migrations and run SQLC generate.

```bash
graft sqlc-migrate
```

**Requirements:**
- SQLC must be installed and in PATH
- `sqlc_config_path` must be set in config
- Valid `sqlc.yaml` configuration file

**What it does:**
- Applies all pending migrations
- Runs `sqlc generate` to update Go types
- Reports any errors from either step

## Configuration

### `graft.config.json`

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

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `schema_path` | Path to schema file | `db/schema/schema.sql` |
| `migrations_path` | Directory for migration files | `db/migrations` |
| `sqlc_config_path` | Path to SQLC config | `sqlc.yml` |
| `backup_path` | Directory for backups | `db/backup` |
| `database.provider` | Database provider | `postgresql` |
| `database.url_env` | Environment variable for DB URL | `DATABASE_URL` |

## Migration Files

Migration files are stored as JSON in the `db/migrations/` directory:

```
db/migrations/
├── 20240131103000_create_users_table.json
├── 20240131104500_add_user_roles.json
└── 20240131110000_create_posts_table.json
```

### Migration File Format

```json
{
  "migration": {
    "id": "20240131103000_create_users_table",
    "name": "create users table",
    "up": "CREATE TABLE users (\n    id SERIAL PRIMARY KEY,\n    name VARCHAR(255) NOT NULL,\n    email VARCHAR(255) UNIQUE NOT NULL,\n    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()\n);",
    "down": "DROP TABLE IF EXISTS users CASCADE;",
    "checksum": "a1b2c3d4e5f6...",
    "created_at": "2024-01-31T10:30:00Z"
  }
}
```

## Database Support

Currently supported databases:
- ✅ PostgreSQL

Planned support:
- 🔄 MySQL
- 🔄 SQLite
- 🔄 SQL Server

## SQLC Integration

Graft comes with built-in SQLC integration for seamless Go code generation:

1. **Automatic Setup**: `graft init` creates a complete SQLC configuration:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries/"
    schema: "db/schema/"
    gen:
      go:
        package: "graft"
        out: "graft_gen/"
        sql_package: "pgx/v5"
```

2. **Automatic Generation**: 
   - `graft migrate` automatically runs SQLC generate after creating migrations
   - `graft sqlc-migrate` applies migrations and generates types

3. **Directory Structure**:
```
db/
├── schema/
│   └── schema.sql      # Database schema
├── queries/
│   └── users.sql       # SQL queries for SQLC
└── migrations/         # Migration files
graft_gen/              # Generated Go types
```

## Backup System

Graft automatically prompts for backups before destructive operations:

- **Backup Location**: `db/backup/backup_YYYY-MM-DD_HHMMSS.json`
- **Format**: JSON with complete table data
- **Automatic Prompts**: Before `reset` and schema conflicts
- **Smart Backup**: Only creates backups when database contains data

### Backup Structure

```json
{
  "timestamp": "2024-01-31_10-30-00",
  "version": "5_migrations",
  "comment": "Pre-reset backup",
  "tables": {
    "users": {
      "columns": ["id", "name", "email", "created_at"],
      "data": [
        {"id": 1, "name": "John", "email": "john@example.com", "created_at": "2024-01-31T10:00:00Z"}
      ]
    }
  }
}
```

## Best Practices

1. **Always Review Migrations**: Check generated migration files before applying
2. **Use Descriptive Names**: Make migration names clear and descriptive
3. **Test Migrations**: Test migrations on development databases first
4. **Backup Production**: Always backup production databases before migrations
5. **Version Control**: Commit migration files to version control
6. **Sequential Migrations**: Apply migrations in order, don't skip
7. **Use --force Carefully**: Only use `--force` flag when you're certain

## Troubleshooting

### Common Issues

**Database Connection Failed**
```bash
Error: database URL not found in environment variable DATABASE_URL
```
**Solution:** Set the `DATABASE_URL` environment variable or create a `.env` file.

**Migration Already Exists**
```bash
Error: migration validation failed: migration already exists
```
**Solution:** Check for duplicate migration names or use `graft status` to see applied migrations.

**SQLC Generate Failed**
```bash
Error: sqlc not found in PATH
```
**Solution:** Install SQLC or remove `sqlc_config_path` from config.

**Config File Not Found**
```bash
Error: failed to load config
```
**Solution:** Run `graft init` to create config file or specify path with `--config`.

### Debug Mode

Use the `--force` flag to skip confirmations during development:

```bash
graft apply --force
graft reset --force
```

## Examples

### Basic Workflow

```bash
# Initialize project
graft init

# Set database URL
export DATABASE_URL="postgres://user:pass@localhost:5432/mydb"

# Create first migration
graft migrate "initial schema"

# Edit the migration file, then apply
graft apply

# Check status
graft status

# Create another migration
graft migrate "add indexes"

# Apply new migration
graft apply
```

### Production Deployment

```bash
# Deploy migrations in production
graft deploy

# Check status
graft status

# Create backup before major changes
graft backup "before v2.0 deployment"
```

### Development with SQLC

```bash
# Set SQLC config in graft.config.json
# "sqlc_config_path": "sqlc.yaml"

# Apply migrations and generate types
graft sqlc-migrate
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
- PostgreSQL support via [pgx](https://github.com/jackc/pgx)
