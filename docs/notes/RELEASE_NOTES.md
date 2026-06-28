# flash 2.7.3

**Release Date:** 2026-06-28

## 🚀 New Features

### `-- @required` Annotation for CQL Params

Mark CQL INSERT/UPDATE params as non-nullable in generated code:

```sql
-- @required: *
-- name: CreateUser :exec
INSERT INTO myapp.users (id, username, email, bio) VALUES (?, ?, ?, ?);

-- @required: id, username, email
-- name: CreateUserPartial :exec
INSERT INTO myapp.users (id, username, email, bio) VALUES (?, ?, ?, ?);
```

**Generated Kotlin:**
```kotlin
// @required: * → all non-null
data class CreateUserParams(val id: UUID, val username: String, val email: String, val bio: String)

// @required: id, username, email → only specified non-null
data class CreateUserPartialParams(val id: UUID, val username: String, val email: String, val bio: String?)
```

- Works with `-- @required: *` (all params non-null) or specific column names
- Annotation can appear before OR after `-- name:` line
- Validates param names — error if invalid name listed
- Only affects input params, not output columns
- All generators respect it: Go, TypeScript, Python, Kotlin, Java

### Nullable Params from Schema

Params now inherit nullability from their corresponding schema column:

- **CQL**: All non-PK columns are nullable → params are `Type?` / `Optional[Type]` / `Type | null`
- **PostgreSQL/MySQL/SQLite**: Columns with `NOT NULL` → params are non-null; without → nullable
- `-- @required` overrides schema defaults for CQL

### `.cql` Extension for ScyllaDB/Cassandra

`flash init --scylla` now creates `schema.cql` and `users.cql` instead of `.sql`. The parser accepts both `.sql` and `.cql` files.

---

## 🔧 Bug Fixes

### ScyllaDB/Cassandra

- **`timestamp` → `Instant`** (not `LocalDateTime`) — uses `java.time.Instant` for CQL timestamp type
- **`import java.time.Instant`** in Models.kt, Users.kt, Queries.kt
- **Keyspace prefix stripped** from model names (`myapp.users` → `Users`)
- **`!!` on non-nullable getters** — PK fields use `row.getUuid("id")!!`, nullable fields have no assertion
- **CQL getter nullable** — `cqlKtGetter` takes `nullable` param, adds `!!` only for non-null columns

### Parser

- **`$N || col` concat** → infers `col_prefix: TEXT` (dollar-style params)
- **ILIKE `'%' || $N || '%'`** → infers column name from ILIKE context
- **`OFFSET $N` / `OFFSET ?`** → correctly infers `offset: INTEGER`
- **LIMIT/OFFSET priority** — checked before ILIKE to prevent false matches
- **SET counter `col = col + $N`** → infers `col_delta` with correct type
- **SET COALESCE `col = COALESCE($N, col)`** → infers column name and type
- **Multi-assignment SET** → `..., col = $N` matches anywhere in SET clause
- **Invalid regex syntax** — removed Go-unsupported `(?!` lookahead

### Code Generation

- **`@Suppress("DuplicatedCode")`** on generated Kotlin query classes
- **`UUID::class.java`** instead of `java.util.UUID::class.java` (no redundant qualifier)
- **`-f` flag in production** — added persistent flags to plugin executor
- **`--db` flag in production** — works in both dev and plugin mode

---

## 📦 Version

`2.7.3` (prod) · `2.7.3-beta-dev` (dev)
