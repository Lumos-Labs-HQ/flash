# flash 2.7.7

**Release Date:** 2026-07-16

## 🚀 New Features

### 🦀 Rust Code Generation (sqlx)

FlashORM now generates type-safe, async Rust code using **sqlx**. Full support for PostgreSQL, MySQL, and SQLite.

**Configuration:**
```toml
[gen.rust]
enabled = true
out = "src/flash_gen"
driver = "sqlx"
```

**Generated output:**
```
src/flash_gen/
├── mod.rs          # Module re-exports
├── models.rs       # Table structs with #[derive(FromRow)]
├── db.rs           # Queries struct (pool wrapper)
└── users.rs        # Per-query-file async methods
```

**Usage:**
```rust
let queries = Queries::new(pool);

// :one → fetch_one
let user = queries.get_user(1).await?;

// :many → fetch_all
let users = queries.list_users().await?;

// :exec → execute
queries.delete_user(1).await?;

// Single-column queries use query_scalar
let count = queries.count_users().await?;  // i64
```

**Key features:**
- Async by default — all methods are `pub async fn`
- `#[derive(FromRow, Serialize, Deserialize)]` on all structs
- `sqlx::query_scalar` for single-column results (COUNT, EXISTS, etc.)
- `sqlx::query_as` for multi-column struct results
- `Option<T>` for nullable columns
- PostgreSQL enums → `#[derive(sqlx::Type)]` enums
- Params struct auto-generated for queries with >2 parameters
- `&str` and `&[u8]` references for string/binary params (zero-copy)
- Proper type mapping: UUID, DateTime, Decimal, serde_json::Value
- Incremental generation — only changed files are regenerated

**Supported databases:**

| Database | Pool Type | Param Style |
|----------|-----------|-------------|
| PostgreSQL | `sqlx::PgPool` | `$1, $2, ...` |
| MySQL | `sqlx::MySqlPool` | `?` |
| SQLite | `sqlx::SqlitePool` | `?` |

**Auto-detection:** Projects with `Cargo.toml` are automatically detected as Rust on `flash init`.

### Improved SQL Keyword Recognition

Added `CURRENT_TIMESTAMP`, `CURRENT_DATE`, `CURRENT_TIME`, `COALESCE`, `NULLIF`, `NOW`, window functions (`ROW_NUMBER`, `RANK`, `DENSE_RANK`, `LAG`, `LEAD`), and common SQL functions to the keyword list. This fixes false "column does not exist" errors when using these in SET/WHERE clauses.

### COUNT() Returns BIGINT

`COUNT(*)` expressions are now correctly typed as `BIGINT` (i64/Long) instead of `INTEGER`. This matches the actual return type of COUNT in PostgreSQL, MySQL, and SQLite, preventing runtime type mismatch errors.

---

## 🔧 Improvements

- Rust project detection via `Cargo.toml` or `.rs` source files in `flash init`
- Driver header comments updated with Rust driver info for all database types
- `flash gen` command description updated to include Rust

---

