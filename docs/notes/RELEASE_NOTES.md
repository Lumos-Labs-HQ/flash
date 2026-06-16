# FlashORM v2.4.5 Release Notes

## Version 2.4.5 — ScyllaDB Support & Multi-Provider Hardening

**Released:** June 2025

---

### ✨ New Features

- **ScyllaDB & Apache Cassandra Support (Beta)** — Full database adapter with multi-cluster connections, keyspace-aware operations, and migration pipelines. Currently in beta — not all features are supported yet. See [ScyllaDB Guide](/databases/scylladb) for details.

- **CQL UDT Support** — `CREATE TYPE` statements for user-defined types are now parsed, included in schema snapshots, and generated in migrations. Go codegen produces proper struct types for UDTs.

- **CQL Materialized View Support** — `CREATE MATERIALIZED VIEW` statements are parsed, ordered after tables in migrations, and generated with `IF NOT EXISTS` for idempotent `flash apply`.

- **CQL Collection & UDT Column Types** — `set<text>`, `list<text>`, `map<text,text>`, `frozen<address>`, `list<frozen<social_link>>` are correctly parsed, diffed, and generated in Go code as `[]string`, `[]*Address`, etc.

- **Multi-Keyspace Support** — ScyllaDB adapter iterates all user keyspaces (excluding system keyspaces). GetTables returns keyspace-grouped results with display name stripping (`users` not `ap.users`).

- **ClickHouse (Beta)** — Adapter updated for compatibility. Under active development.

- **Auto-Create Default Keyspace** — When `DATABASE_URL=scylla://host:port` has no keyspace path, a `flash` keyspace with SimpleStrategy/replication_factor=1 is auto-created.

- **Editor Autocomplete (CQL)** — SQL editor in Studio now provides full CQL keyword suggestions: `KEYSPACE`, `MATERIALIZED VIEW`, `CLUSTERING ORDER BY`, CQL types (`uuid`, `timeuuid`, `frozen`, `counter`), and CQL clauses (`ALLOW FILTERING`, `CONTAINS`, `USING TTL`).

- **Keyspace Grouping in Studio** — Database view in Studio shows collapsible keyspace sections with table counts under each.

---

### 🐛 Bug Fixes

- **CQL Category: `RETURNING` Clause** — `INSERT` statements with `RETURNING` (PostgreSQL syntax) are stripped from CQL query strings since CQL doesn't support it. Queries downgrade from `:one` to `:exec`.

- **CQL Category: `WITH CLUSTERING ORDER BY` Parsing** — Fixed duplicate column generation caused by the WITH clause bleeding into column extraction via `utils.SplitColumns`.

- **CQL Category: Angle Bracket Types in Columns** — `SplitColumns` now tracks `<`/`>` depth. Types like `map<text,text>` and `frozen<tuple<text,int>>` are not broken at inner commas.

- **CQL Category: `INTO ap.users` INSERT Regex** — Fixed param name inference for keyspace-qualified table names. `INSERT INTO ap.users (...)` now correctly infers `Id`, `Username`, etc. instead of `Param1`, `Param2`.

- **CQL Category: Multi-INSERT Params** — Multi-statement INSERTs like `INSERT INTO A ...; INSERT INTO B ...` now correctly collect column names from all INSERT clauses for param naming.

- **CQL Category: SET Clause Params** — UPDATE statements with `?`-style params in SET clauses (`SET full_name = ?, bio = ?`) now correctly infer param names.

- **CQL Category: BETWEEN/>= Params** — WHERE clauses with `age >= ? AND age <= ?` now infer `age`/`age2` instead of `param1`/`param2`.

- **CQL Category: CONTAINS Params** — WHERE clauses with `skills CONTAINS ?` now infer the column name.

- **CQL Category: Collection Params** — `set<text>`, `list<text>` params now map to `[]string` in Go instead of `string`, matching gocql's marshaling requirements.

- **CQL Category: System Table Filtering** — Reset command no longer generates DROP statements for system tables. Schema diff filters ScyllaDB internal tables.

- **CQL Category: MapScan for Row Types** — Generated gocql query methods now use `MapScan` instead of `Scan`, avoiding unmarshal errors for CQL types.

- **CQL Category: Go Row Structs** — Gocql Row structs use `string` value types (not `*string` pointers) since MapScan produces value types via `fmt.Sprint`.

- **CQL Category: Go Models** — `uuid`/`timeuuid` now maps to `string` in Go structs instead of `time.Time`. Collection types (`set<text>` → `[]string`). UDTs produce dedicated struct types.

- **CQL Category: gocql Import** — Per-query files correctly import `context`, `fmt`, and `gocql`. `database/sql` is not imported for ScyllaDB-generated code.

- **CQL Category: Keyspace Resolution** — `GetTableColumns`, `GetTableData`, etc. correctly extract keyspace from qualified names like `ap.users`, resolving to the right keyspace regardless of which keyspace was last active.

- **PostgreSQL Category: `(NOW())` False Table Match** — Fixed `ExtractTableName` matching `FROM` inside `EXTRACT(EPOCH FROM (NOW() - p.created_at))` as a table name. Parenthesized content is stripped before regex matching.

- **PostgreSQL Category: CTE Recognition** — WITH CTEs like `user_post_stats AS (...)` are now recognized as query-local names and skipped during schema table validation.

- **General: Studio Schema Visualizer** — Now iterates all user keyspaces for ScyllaDB and labels nodes with keyspace info.

- **General: Studio Export** — ENUM type export (PostgreSQL-specific `pg_type` query) is gated behind provider check; won't fail on ScyllaDB.

- **General: Studio Metrics** — ScyllaDB metrics page shows table sizes and row counts per keyspace.

---

### ⚡ Performance Improvements

- **Schema Parser** — `SplitColumns` angle bracket tracking avoids false column splits for CQL types, reducing parse errors and silent table drops.

- **Code Generation** — Results column types in gocql use `string` (value type), reducing pointer allocation overhead in generated code.

---

### 🗑️ Breaking Changes

- **ScyllaDB `RETURNING`** — Queries with `RETURNING` in `.sql` files will be silently downgraded from `:one`/`:many` to `:exec` for ScyllaDB. Move RETURNING columns to a separate SELECT query for CQL.

- **Go Codegen — gocql Row Types** — Row struct fields are now `string` (value type) instead of `*string` (nullable). Update your code accordingly.

---

### 📦 Installation

```bash
# Install or upgrade
go install github.com/Lumos-Labs-HQ/flash-orm

# Verify
flash --version  # 2.4.5
```

---

### 🩹 Post-Release Patches

#### 🐛 Bug Fixes

- **CQL Category: `COALESCE` Handling** — `COALESCE` in SELECT columns, SET clauses, and WHERE filters is now correctly parsed and passed through to generated CQL queries without breaking parameter inference or table-name extraction.

- **CQL Category: Migration Generation** — Schema diffs for CQL UDTs, materialized views, and collection types now produce correct UP/DOWN SQL. ScyllaDB migrations no longer include SQL syntax not supported by CQL.

- **CQL Category: Query Codegen Fixes** — Fixed parameter inference edge cases for multi-statement INSERTs, BETWEEN/>= patterns, CONTAINS clauses, and keyspace-qualified table names (`INSERT INTO ap.users (...)`).

- **Go Codegen: ScyllaDB gocql** — Row struct fields now use `string` value types (not `*string` pointers). Import `database/sql` is excluded for ScyllaDB-generated code. Per-query files correctly import `context`, `fmt`, and `gocql`.

- **Studio Fixes** — Fixed table-not-showing issues in SQL Studio. Table data retrieval improved for ScyllaDB/ClickHouse adapters. Schema validation no longer fails on CQL-specific column types.

- **Schema Parser** — Fixed `ExtractTableName` matching `FROM` inside `EXTRACT(EPOCH FROM ...)` as a table name. CTE recognition now skips WITH-clause query-local names during schema validation.

- **Export System** — ENUM-type export (PostgreSQL-specific) is now gated behind a provider check, preventing failures on ScyllaDB and ClickHouse.

---

### 🔗 Related Documentation

- [ScyllaDB Setup Guide](/databases/scylladb)
- [ClickHouse Setup Guide](/databases/clickhouse)
- [Migrations](/concepts/migrations)
- [Code Generation](/concepts/code-generation)
- [Studio](/concepts/studio)
