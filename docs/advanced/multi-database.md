# Multi-Database Configuration

FlashORM supports multiple databases in a single project. Each database gets its own schema, queries, migrations, and code generation config.

## Configuration

```toml
version = "2"

[[databases]]
name = "main"
provider = "postgresql"
url_env = "DATABASE_URL"
schema_dir = "db/main/schema"
queries = "db/main/queries/"
migrations_path = "db/main/migrations"
default = true

[databases.gen.kotlin]
enabled = true
package = "com.example.main.flashgen"

[[databases]]
name = "analytics"
provider = "clickhouse"
url_env = "ANALYTICS_URL"
schema_dir = "db/analytics/schema"
queries = "db/analytics/queries/"
migrations_path = "db/analytics/migrations"

[databases.gen.go]
enabled = true
out = "gen/analytics"
```

## Commands

All commands support `--db <name>`:

```bash
flash gen --db main           # Generate code for "main" only
flash gen --db analytics      # Generate code for "analytics" only
flash gen                     # Generate default DB (or all if no default)
flash gen -f                  # Force regenerate, skip cache

flash apply --db main         # Apply migrations to "main"
flash migrate "add_col" --db analytics
flash studio --db main        # Open studio for "main"
flash dblist                  # List all configured databases
```

## Default Database

Add `default = true` to one database entry:

- `flash gen` without `--db` → generates only the default
- No default set → generates ALL databases
- `--db <name>` always targets specific database

## Backward Compatibility

The single-database format still works unchanged:

```toml
version = "2"
schema_dir = "db/schema"
queries = "db/queries/"

[database]
provider = "postgresql"
url_env = "DATABASE_URL"

[gen.kotlin]
enabled = true
package = "com.example.flashgen"
```
