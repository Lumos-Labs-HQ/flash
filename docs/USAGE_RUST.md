# 🦀 Rust Usage Guide

FlashORM generates type-safe, async Rust code using **sqlx** from your SQL query files. All generated code is fully async, uses `#[derive(FromRow)]` for automatic row mapping, and provides proper `Option<T>` handling for nullable columns.

## Quick Start

### 1. Initialize

```bash
# In a Rust project (Cargo.toml detected automatically)
flash init --postgresql
```

### 2. Add Dependencies

Add these to your `Cargo.toml`:

```toml
[dependencies]
sqlx = { version = "0.9", features = ["runtime-tokio", "postgres", "chrono", "uuid", "rust_decimal", "macros"] }
tokio = { version = "1", features = ["full"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
chrono = { version = "0.4", features = ["serde"] }
uuid = { version = "1", features = ["serde"] }
rust_decimal = { version = "1", features = ["serde-with-str"] }
dotenvy = "0.15"
```

**Feature flags by database:**

| Database | sqlx feature |
|----------|-------------|
| PostgreSQL | `postgres` |
| MySQL | `mysql` |
| SQLite | `sqlite` |

### 3. Configure

```toml
# flash.toml
version = "2"
schema_dir = "db/schema"
queries = "db/queries/"
migrations_path = "db/migrations"

[database]
provider = "postgresql"
url_env = "DATABASE_URL"

[gen.rust]
enabled = true
out = "src/flash_gen"
driver = "sqlx"
```

### 4. Write Schema & Queries

**`db/schema/schema.sql`:**
```sql
CREATE TABLE users (
    id    SERIAL PRIMARY KEY,
    name  VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    age   INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**`db/queries/users.sql`:**
```sql
-- name: GetUser :one
SELECT id, name, email, age, created_at
FROM users WHERE id = $1 LIMIT 1;

-- name: ListUsers :many
SELECT id, name, email, age, created_at
FROM users ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (name, email, age)
VALUES ($1, $2, $3)
RETURNING id, name, email, age, created_at;

-- name: UpdateUserName :one
UPDATE users SET name = $1 WHERE id = $2
RETURNING id, name, email, age, created_at;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: CountUsers :one
SELECT COUNT(*) AS count FROM users;
```

### 5. Generate

```bash
flash gen
```

### 6. Use in Code

```rust
mod flash_gen;

use flash_gen::db::Queries;
use sqlx::PgPool;

#[tokio::main]
async fn main() -> Result<(), sqlx::Error> {
    dotenvy::dotenv().ok();
    let url = std::env::var("DATABASE_URL").expect("DATABASE_URL not set");
    let pool = PgPool::connect(&url).await?;

    let queries = Queries::new(pool);

    // Create a user
    let user = queries.create_user("Alice", "alice@example.com", 30).await?;
    println!("Created: {} (id={})", user.name, user.id);

    // Get by ID
    let found = queries.get_user(user.id).await?;
    println!("Found: {:?}", found);

    // List all
    let all = queries.list_users().await?;
    println!("Total users: {}", all.len());

    // Count
    let count = queries.count_users().await?;
    println!("Count: {}", count);

    // Delete
    queries.delete_user(user.id).await?;
    println!("Deleted user {}", user.id);

    Ok(())
}
```

---

## Generated Code Structure

```
src/flash_gen/
├── mod.rs          # Module declarations (re-exports)
├── models.rs       # Table structs with #[derive(FromRow)]
├── db.rs           # Queries struct (connection wrapper)
└── users.rs        # Per-query-file: impl Queries { async fn ... }
```

### `models.rs`

```rust
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Users {
    pub id: i32,
    pub name: String,
    pub email: String,
    pub age: Option<i32>,
    pub created_at: DateTime<Utc>,
}
```

### `db.rs`

```rust
pub struct Queries {
    pub(crate) pool: sqlx::PgPool,
}

impl Queries {
    pub fn new(pool: sqlx::PgPool) -> Self {
        Self { pool }
    }
}
```

### `users.rs` (per-query-file)

```rust
use super::models::*;
use super::db::Queries;

impl Queries {
    /// GetUser
    pub async fn get_user(&self, id: i32) -> Result<Users, sqlx::Error> {
        sqlx::query_as::<_, Users>(
            "SELECT id, name, email, age, created_at FROM users WHERE id = $1 LIMIT 1;"
        )
        .bind(id)
        .fetch_one(&self.pool)
        .await
    }

    /// ListUsers
    pub async fn list_users(&self) -> Result<Vec<Users>, sqlx::Error> {
        sqlx::query_as::<_, Users>(
            "SELECT id, name, email, age, created_at FROM users ORDER BY created_at DESC;"
        )
        .fetch_all(&self.pool)
        .await
    }

    /// CountUsers
    pub async fn count_users(&self) -> Result<i64, sqlx::Error> {
        sqlx::query_scalar::<sqlx::Postgres, i64>(
            "SELECT COUNT(*) AS count FROM users;"
        )
        .fetch_one(&self.pool)
        .await
    }

    /// DeleteUser
    pub async fn delete_user(&self, id: i32) -> Result<(), sqlx::Error> {
        sqlx::query("DELETE FROM users WHERE id = $1;")
            .bind(id)
            .execute(&self.pool)
            .await?;
        Ok(())
    }
}
```

---

## Query Commands

| Annotation | Rust Return Type | sqlx Method |
|------------|-----------------|-------------|
| `:one` | `Result<T, sqlx::Error>` | `query_as` / `query_scalar` |
| `:many` | `Result<Vec<T>, sqlx::Error>` | `query_as` / `query_scalar` + `fetch_all` |
| `:exec` | `Result<(), sqlx::Error>` | `query` + `execute` |
| `:execresult` | `Result<PgQueryResult, sqlx::Error>` | `query` + `execute` |

---

## Type Mapping

### PostgreSQL

| SQL Type | Rust Type |
|----------|-----------|
| `SERIAL`, `INTEGER`, `INT4` | `i32` |
| `BIGSERIAL`, `BIGINT`, `INT8` | `i64` |
| `SMALLINT`, `INT2` | `i16` |
| `REAL`, `FLOAT4` | `f32` |
| `DOUBLE PRECISION`, `FLOAT8` | `f64` |
| `NUMERIC`, `DECIMAL` | `Decimal` |
| `BOOLEAN` | `bool` |
| `TEXT`, `VARCHAR(N)` | `String` |
| `UUID` | `Uuid` |
| `TIMESTAMPTZ` | `DateTime<Utc>` |
| `TIMESTAMP` | `NaiveDateTime` |
| `DATE` | `chrono::NaiveDate` |
| `TIME` | `chrono::NaiveTime` |
| `BYTEA` | `Vec<u8>` |
| `JSON`, `JSONB` | `serde_json::Value` |
| `TEXT[]` | `Vec<String>` |
| `INTEGER[]` | `Vec<i32>` |
| Nullable column | `Option<T>` |

### MySQL

| SQL Type | Rust Type |
|----------|-----------|
| `INT` | `i32` |
| `BIGINT` | `i64` |
| `FLOAT` | `f32` |
| `DOUBLE` | `f64` |
| `BOOLEAN`, `TINYINT(1)` | `bool` |
| `VARCHAR(N)`, `TEXT` | `String` |
| `DATETIME` | `NaiveDateTime` |
| `BLOB` | `Vec<u8>` |
| `JSON` | `serde_json::Value` |

---

## Parameter Handling

### Simple Params (≤2)

Passed as individual function arguments:

```rust
pub async fn get_user(&self, id: i32) -> Result<Users, sqlx::Error>
pub async fn update_name(&self, name: &str, id: i32) -> Result<Users, sqlx::Error>
```

String types use `&str` references, byte arrays use `&[u8]`.

### Struct Params (>2)

When a query has more than 2 parameters, a params struct is generated:

```rust
pub struct CreateUserParams {
    pub name: String,
    pub email: String,
    pub age: i32,
}

pub async fn create_user(&self, params: &CreateUserParams) -> Result<Users, sqlx::Error>
```

---

## Row Structs

When a query returns columns that don't match any table exactly, a dedicated row struct is generated:

```sql
-- name: GetUserWithPostCount :one
SELECT u.id, u.name, COUNT(p.id) AS post_count
FROM users u LEFT JOIN posts p ON p.user_id = u.id
WHERE u.id = $1 GROUP BY u.id, u.name;
```

Generates:

```rust
#[derive(Debug, Clone, sqlx::FromRow, serde::Serialize, serde::Deserialize)]
pub struct GetUserWithPostCountRow {
    pub id: i32,
    pub name: String,
    pub post_count: i64,
}
```

When the query columns exactly match a table (same names, same order), the model struct is reused directly.

---

## Enum Support

PostgreSQL enums are mapped to Rust enums with `sqlx::Type`:

```sql
CREATE TYPE user_role AS ENUM ('admin', 'user', 'moderator');
```

Generates:

```rust
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "user_role", rename_all = "snake_case")]
pub enum UserRole {
    #[sqlx(rename = "admin")]
    Admin,
    #[sqlx(rename = "user")]
    User,
    #[sqlx(rename = "moderator")]
    Moderator,
}
```

---

## Multi-Database Config

```toml
[[databases]]
name = "main"
provider = "postgresql"
url_env = "DATABASE_URL"
schema_dir = "db/main/schema"
queries = "db/main/queries/"

[databases.gen.rust]
enabled = true
out = "src/main_gen"

[[databases]]
name = "analytics"
provider = "postgresql"
url_env = "ANALYTICS_URL"
schema_dir = "db/analytics/schema"
queries = "db/analytics/queries/"

[databases.gen.rust]
enabled = true
out = "src/analytics_gen"
```

```bash
flash gen --db main
flash gen --db analytics
flash gen          # generates for all databases
```

---

## Incremental Generation

FlashORM tracks checksums of your schema and query files. On subsequent `flash gen` runs:

- Only modified query files are regenerated
- Schema changes trigger full regeneration of models
- Use `flash gen -f` to force full regeneration

---

## Tips

1. **Add `mod flash_gen;`** to your `src/main.rs` or `src/lib.rs`
2. **Don't edit generated files** — they're overwritten on `flash gen`
3. **Use `.await?`** — all generated methods return `Result<T, sqlx::Error>`
4. **Connection pooling** — pass `sqlx::PgPool` (not a single connection) to `Queries::new()`
5. **Feature flags** — enable `uuid`, `chrono`, `rust_decimal` in your sqlx dependency for full type support
