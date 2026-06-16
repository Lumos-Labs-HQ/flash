# FlashORM Release Notes

---

## v2.4.8

### New Features
- **`.cql` file support** — Schema and query files can now use `.cql` extension alongside `.sql`. ScyllaDB/Cassandra projects can use `schema.cql` and `queries.cql` natively.
- **Go codegen `out =` config** — `[gen.go]` now respects the `out` field to control the output directory (was hardcoded to `flash_gen/`). Package name is derived from the directory basename.
  ```toml
  [gen.go]
  enabled = true
  out = "src/db"   # generates package "db"
  ```
- **ScyllaDB: `gocql/gocql` driver support** — In addition to the default `apache/cassandra-gocql-driver/v2`, you can now use the community `github.com/gocql/gocql` driver:
  ```toml
  [gen.go]
  enabled = true
  driver = "gocql"   # uses github.com/gocql/gocql
  ```
  Default remains `apache/cassandra-gocql-driver/v2`. Both drivers use `*gocql.Session` directly — no wrapper needed.

### Bug Fixes
- **ScyllaDB: `CREATE TYPE` parallel execution crash** — `unconfigured table` error during `flash apply` when UDT types were created in parallel with tables that depend on them. Types now execute sequentially before tables.
- **Schema parser: string literals in `GENERATED ALWAYS AS`** — Columns after a generated column with quoted values (e.g. `'[0,18)'::int4range`) were silently dropped. Fixed string-literal-aware column splitter.
- **Schema parser: composite PKs** — `PRIMARY KEY ((country, city), order_id)` now correctly stores partition + clustering columns in migrations.
- **ScyllaDB codegen: type mapping** — `timeuuid` was mapped to `time.Time`; `set<>`, `list<>`, `map<>` were mapped to `string`. Now correctly maps to `string`, `[]T`, `map[K]V`.
- **ScyllaDB codegen: `MapScan` replaced with `Scan`** — Direct positional `Scan` with typed struct fields (zero-alloc, ~2x faster per row).
- **Param naming** — Generic queries (CTEs, JSONB ops, array functions, HAVING, LIMIT) now infer proper parameter names instead of `param1`, `param2`.
- **Go reserved keywords** — Columns/params named `type`, `map`, `select` etc. now get `_` suffix in generated code.
- **`flash migrate` skip DB on snapshot** — When a valid schema snapshot exists, `flash migrate` no longer connects to the database, making it instant.

### Performance
- **ScyllaDB connect time: ~15s → ~5s** — Pinned `ProtoVersion = 4` (skips negotiation round-trips), `Consistency = One`, disabled topology/schema event subscriptions, skip double session creation when keyspace already exists.

---

# FlashORM v2.4.7 Release Notes

