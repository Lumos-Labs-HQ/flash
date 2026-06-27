# flash 2.7.0

**Release Date:** 2026-06-27

## 🚀 New Features

### Multi-Database Support

FlashORM now supports multiple databases in a single project. Each database gets its own schema, queries, migrations, and code generation config.

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

**Commands:**
- `flash dblist` — shows all configured databases with provider and paths
- `flash gen --db main` — generate code for a specific database
- `flash migrate "name" --db analytics` — create migration for specific database
- `flash apply --db main` — apply migrations to specific database
- `flash studio --db analytics` — open studio for specific database
- All commands accept `--db <name>` flag

**Default database:**
- `default = true` on a database entry — `flash gen` without `--db` only generates the default
- No default set — `flash gen` generates ALL databases
- Single-db config (old format) works unchanged

### `flash gen -f` — Force Regeneration

Skip incremental cache checking and regenerate all files:
```bash
flash gen -f          # regenerate all
flash gen -f --db main  # force regen specific db
```

### Auto-Derived `out` Path for Java/Kotlin

When `out` is not set, it's computed from `package`:
```toml
[gen.kotlin]
enabled = true
package = "com.example.myapp.flashgen"
# out auto-computed: src/main/kotlin/com/example/myapp/flashgen
```

### `flash uninstall`

Removes the flash binary and `~/.flash` directory:
```bash
flash uninstall       # interactive confirmation
flash uninstall -f    # skip confirmation
```

---

## 🔧 Bug Fixes & Improvements

### Parser

- **`IN ($1,$2,$3)` → `= ANY($1)` rewrite** — collapses multi-param IN lists into a single typed array parameter
- **COALESCE `SET col = COALESCE($N, col)`** — correctly infers param name and type from the target column
- **Range params** — `col >= $1 AND col <= $2` now infers `col_start`/`col_end` instead of `col`/`col2`
- **CTE param inference** — `InferParamTypeByName` fallback for params not in the primary table (`*_count` → INTEGER, `limit`/`offset` → INTEGER)
- **Cross-table type lookup** — params referencing columns in JOINed/subquery tables resolve correctly
- **Broader WHERE matching** — `WHERE (col = $N ...)` inside parentheses now matches correctly
- **JSONB `->> $N`** — infers param name as `key`
- **Concurrent map fix** — `sync.RWMutex` on TypeInferrer cache prevents crash with multiple query files
- **POM parent skip** — `<parent><groupId>` no longer pollutes detected package name
- **Lowercase packages** — all auto-detected Java/Kotlin package names are lowercased

### Wildcard Expansion

- **Single qualified wildcard** (`SELECT u.* FROM users u`) — resolves alias, expands to all table columns
- **`SELECT DISTINCT u.*`** — strips DISTINCT before resolving alias
- **Multi-wildcard** (`SELECT f.*, u.*`) — resolves each alias independently, expands both tables, prefixes duplicates with alias (`f_id`, `u_id`)
- **Mixed wildcard + explicit** (`SELECT p.*, COUNT(*) AS likes`) — expands `p.*` and keeps explicit columns

### Code Generation (all languages)

- **Params objects** — queries with >2 params get a typed Params struct/record/TypedDict/data class
- **Renamed `Args` → `Params`** — `CreateUserParams`, `UpdatePostParams`, etc.
- **Top-level declarations** — Kotlin data classes and Java records in their own files (no inner class conflicts)
- **Enum getters** — `EnumType.valueOf(rs.getString("col"))` for Kotlin/Java
- **Array getters** — `.map { it.toString() }` (Kotlin), proper cast (Java)
- **Array setters** — `stmt.setArray(N, conn.createArrayOf("type", arr))` for both
- **Redundant qualifier removed** — `UUID::class.java` instead of `java.util.UUID::class.java`
- **`@Suppress("DuplicatedCode")`** — silences IntelliJ warning on generated Kotlin

### Kotlin Specific

- **camelCase** for all field/param names — `userId`, `createdAt`, `isActive`
- **Enum columns** — `UserRole.valueOf(rs.getString("role"))` with nullable support

### Java Specific

- **PreparedStatement try-with-resources** — no more cached stmts map, no IDE warnings
- **Duplicate proxy dedup** in `Queries.java`
- **No `.toString()`** on String params

---

## 📦 Version

`2.7.0` (prod) · `2.7.0-beta-dev` (dev)
