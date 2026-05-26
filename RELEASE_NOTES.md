# FlashORM Release Notes

## Version 2.4.0 — Latest Release

### Seed Command Rewrite

The seed command has been completely rewritten for production use:

- **Automatic FK handling** — Foreign key relationships are now handled automatically. The `--relations` flag has been removed; relations are always resolved via dependency graph + ID tracking.
- **Simplified flags** — Removed `--relations`, `--batch`, and `--no-transaction`. Added `--dry-run` (preview without inserting) and `--exclude` (skip tables).
- **MySQL ID extraction fixed** — MySQL now correctly extracts inserted IDs via `SELECT LAST_INSERT_ID()`, making FK relationships work on MySQL.
- **Self-referencing FKs fixed** — Tables with `parent_id` referencing themselves (e.g., comment threads, category trees) now work correctly.
- **SQLite multi-row inserts** — SQLite now uses fast multi-row inserts when ID tracking isn't needed.
- **Smart batching** — Tables referenced by FKs use single-row inserts (to capture IDs). Unreferenced tables use multi-row for speed.
- **Schema-qualified names** — `public.users:100` parsing fixed with `strings.LastIndex`.
- **All SQL types supported** — BIGINT, SMALLINT, TINYINT, NUMERIC, REAL, DOUBLE, JSONB, ARRAY, ENUM, BYTEA, BLOB, TIME, TIMESTAMPTZ, CHAR, MEDIUMINT, YEAR all generate correct values.
- **RFC 4122 UUID v4** — UUID generator is now standards-compliant.
- **Coordinated timestamps** — When a table has both `created_at` and `updated_at`, `updated_at` is always >= `created_at`.
- **20+ new column patterns** — username, password, token, slug, bio, metadata, lat/lng, ip, color, gender, role, locale, currency, country, dob, age, percent, is_/has_/can_ booleans, sort_order, version, priority, progress, hash, ref_code.
- **Word-boundary pattern matching** — Prevents false positives (e.g., `message` no longer matches `age`). Longest-keyword wins for specificity.
- **ENUM parsing** — Extracts values from `ENUM('a','b')` type definitions and picks randomly.
- **Column def parsing fix** — `DECIMAL(10, 2) NOT NULL` no longer loses the `NOT NULL` constraint due to comma splitting.

### Multi-Driver Matrix

Code generation now supports multiple database drivers per language:

**Go**
- `database/sql` (default) — Standard library, works with all SQL databases
- `pgx` — jackc/pgx/v5 for PostgreSQL (connection pool, native types, better performance)

**JavaScript / TypeScript**
- `pg` (default) — node-postgres for PostgreSQL
- `postgres` — porsager/postgres (tagged template literals, lightweight)
- `mysql2` — MySQL driver
- `better-sqlite3` — Synchronous SQLite driver
- `bun:sqlite` — Bun's native SQLite driver

**Python**
- PostgreSQL: `asyncpg` (default async) / `psycopg3` (sync or async)
- MySQL: `aiomysql` (default async) / `pymysql` (sync)
- SQLite: `aiosqlite` (default async) / `sqlite3` (sync)

Configure via `flash.toml`:

```toml
[gen.go]
enabled = true
driver = "pgx"

[gen.js]
enabled = true
driver = "postgres"

[gen.python]
enabled = true
driver = "psycopg3"
async = true
```

### Documentation Overhaul

- **New Examples section** — Complete examples for every feature:
  - End-to-end workflows for Go, TypeScript, and Python
  - Every CLI command with all flags and real-world patterns
  - Common schema patterns (blog, e-commerce, social, SaaS, audit logging)
  - SQL query patterns (CRUD, relationships, aggregation, search, JSON, window functions)
  - Seeding patterns with generated data reference tables
- **Updated CLI Reference** — Corrected seed command flags, added all examples.
- **Updated Seeding docs** — Removed outdated flags, added dry-run and exclude examples.
- **Enhanced Architecture docs** — Added seeding pipeline, plugin system, complete data flow diagrams, and multi-driver matrix.
- **Updated Configuration Reference** — Documented all supported drivers per language.
- **Updated Language Guides** — Added driver selection sections for Go, TypeScript, and Python.
- **Better VitePress navigation** — Added Examples to top nav and sidebar.


### Plugin System

The plugin architecture has been redesigned:

- **core** — ORM, migrations, and seeding. Installs automatically the first time any ORM command is run; no manual setup required.
- **studio** — Visual database editor. Optional, installed manually when needed.

The `all` plugin has been removed. A new `flash update` command updates all installed plugins. Use `--self` to also update the flash binary, or `--self-only` to update only the binary.

### SQL Studio — Export & Import

Export and import are now available directly from the SQL Studio interface. Three export modes are supported: Schema Only, Data Only, and Complete (schema + data).

**Performance**
- The full database schema is now fetched in a single query at the start of export and reused throughout, eliminating one query per table for schema introspection.
- Data export no longer issues a row count query before fetching — rows are paged directly until exhausted, removing an extra round-trip per table.

**User Experience**
- A full-screen progress overlay appears during export and import with live status messages at each stage.
- A progress bar transitions from an animated state while the server is working to a percentage fill as each phase completes.
- An accurate summary is shown on import completion — tables created, rows inserted, and any errors.

### Dependency Reduction

- Fiber framework removed — replaced with stdlib `net/http`
- Viper removed — replaced with stdlib `encoding/json`
- lib/pq removed — pgx/v5 is now the sole PostgreSQL driver
- mapstructure removed — plain `json` struct tags used throughout
- Approximately 8 fewer transitive dependencies; smaller binary size

### Bug Fixes

- Fixed `down` command missing from plugin registry
- Fixed CSS duplication across studio static assets

---

For detailed documentation, see:
- [Usage Guide — Go](docs/USAGE_GO.md)
- [Usage Guide — TypeScript](docs/USAGE_TYPESCRIPT.md)
- [Usage Guide — Python](docs/USAGE_PYTHON.md)
- [Contributing](docs/contributing.md)
