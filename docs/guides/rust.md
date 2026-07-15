---
title: Rust Guide
---

# Rust Guide

FlashORM generates type-safe, async Rust code using [sqlx](https://github.com/launchbadge/sqlx). Generated code is fully async, uses `#[derive(FromRow)]` for automatic row mapping, and handles nullable columns with `Option<T>`.

## Setup

### Prerequisites

- Rust 1.75+ (2024 edition recommended)
- A running PostgreSQL, MySQL, or SQLite database
- FlashORM CLI installed

### Initialize Project

```bash
# FlashORM auto-detects Cargo.toml
flash init --postgresql
```

### Dependencies

Add to your `Cargo.toml`:

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

::: tip Feature Flags
Match the sqlx database feature to your provider:
- PostgreSQL: `"postgres"`
- MySQL: `"mysql"`  
- SQLite: `"sqlite"`

Enable `"uuid"`, `"chrono"`, `"rust_decimal"` for full type support.
:::

## Configuration

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

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `false` | Enable Rust generation |
| `out` | `src/flash_gen` | Output directory |
| `driver` | `sqlx` | Driver (only sqlx supported) |

## Writing Queries

Use standard SQL with FlashORM annotations:

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

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: CountUsers :one
SELECT COUNT(*) AS count FROM users;
```

### Query Commands

| Command | Description | Return Type |
|---------|-------------|-------------|
| `:one` | Returns single row | `Result<T, sqlx::Error>` |
| `:many` | Returns multiple rows | `Result<Vec<T>, sqlx::Error>` |
| `:exec` | Executes, no return | `Result<(), sqlx::Error>` |
| `:execresult` | Returns affected rows | `Result<PgQueryResult, sqlx::Error>` |

## Generated Code

### Directory Structure

```
src/flash_gen/
├── mod.rs          # pub mod declarations
├── models.rs       # Table structs
├── db.rs           # Queries connection wrapper
└── users.rs        # Query methods (one file per .sql file)
```

### Models

Each table becomes a struct with `#[derive(FromRow)]`:

```rust
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Users {
    pub id: i32,
    pub name: String,
    pub email: String,
    pub age: Option<i32>,
    pub created_at: DateTime<Utc>,
}
```

### Connection Wrapper

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

### Query Methods

Generated as `impl Queries` blocks:

```rust
impl Queries {
    pub async fn get_user(&self, id: i32) -> Result<Users, sqlx::Error> {
        sqlx::query_as::<_, Users>(
            "SELECT id, name, email, age, created_at FROM users WHERE id = $1 LIMIT 1;"
        )
        .bind(id)
        .fetch_one(&self.pool)
        .await
    }

    pub async fn count_users(&self) -> Result<i64, sqlx::Error> {
        sqlx::query_scalar::<sqlx::Postgres, i64>(
            "SELECT COUNT(*) AS count FROM users;"
        )
        .fetch_one(&self.pool)
        .await
    }
}
```

## Type Mapping

### PostgreSQL → Rust

| SQL Type | Rust Type | Notes |
|----------|-----------|-------|
| `SERIAL`, `INTEGER` | `i32` | |
| `BIGSERIAL`, `BIGINT` | `i64` | Also `COUNT(*)` |
| `SMALLINT` | `i16` | |
| `REAL`, `FLOAT4` | `f32` | |
| `DOUBLE PRECISION` | `f64` | |
| `NUMERIC`, `DECIMAL` | `Decimal` | Requires `rust_decimal` feature |
| `BOOLEAN` | `bool` | |
| `TEXT`, `VARCHAR` | `String` | |
| `UUID` | `Uuid` | Requires `uuid` feature |
| `TIMESTAMPTZ` | `DateTime<Utc>` | Requires `chrono` feature |
| `TIMESTAMP` | `NaiveDateTime` | |
| `DATE` | `chrono::NaiveDate` | |
| `BYTEA` | `Vec<u8>` | |
| `JSON`, `JSONB` | `serde_json::Value` | |
| `TEXT[]` | `Vec<String>` | |
| Any nullable column | `Option<T>` | |

### Parameter Types

Function parameters use references for heap types:

| SQL Type | Parameter Type |
|----------|---------------|
| `TEXT`, `VARCHAR` | `&str` |
| `BYTEA` | `&[u8]` |
| `INTEGER` | `i32` |
| `UUID` | `Uuid` |
| All others | Value type |

## Advanced Features

### Params Struct

Queries with more than 2 parameters get a generated params struct:

```sql
-- name: CreateUser :one
INSERT INTO users (name, email, age)
VALUES ($1, $2, $3)
RETURNING *;
```

```rust
pub struct CreateUserParams {
    pub name: String,
    pub email: String,
    pub age: i32,
}

impl Queries {
    pub async fn create_user(&self, params: &CreateUserParams) -> Result<Users, sqlx::Error> {
        sqlx::query_as::<_, Users>(...)
            .bind(&params.name)
            .bind(&params.email)
            .bind(&params.age)
            .fetch_one(&self.pool)
            .await
    }
}
```

### Row Structs

When query columns don't match any table model:

```sql
-- name: GetUserWithPostCount :one
SELECT u.id, u.name, COUNT(p.id) AS post_count
FROM users u LEFT JOIN posts p ON p.user_id = u.id
WHERE u.id = $1 GROUP BY u.id, u.name;
```

Generates a dedicated struct:

```rust
#[derive(Debug, Clone, sqlx::FromRow, serde::Serialize, serde::Deserialize)]
pub struct GetUserWithPostCountRow {
    pub id: i32,
    pub name: String,
    pub post_count: i64,
}
```

### PostgreSQL Enums

```sql
CREATE TYPE user_role AS ENUM ('admin', 'user', 'moderator');
```

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

### Single-Column Queries

Queries returning a single column use `query_scalar` (no `FromRow` needed):

```sql
-- name: CountUsers :one
SELECT COUNT(*) AS count FROM users;

-- name: UserExists :one
SELECT EXISTS (SELECT 1 FROM users WHERE id = $1) AS exists;
```

```rust
pub async fn count_users(&self) -> Result<i64, sqlx::Error> { ... }
pub async fn user_exists(&self, id: i32) -> Result<String, sqlx::Error> { ... }
```

## Usage Example

```rust
mod flash_gen;

use flash_gen::db::Queries;
use flash_gen::users::CreateUserParams;
use sqlx::PgPool;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    dotenvy::dotenv().ok();
    let url = std::env::var("DATABASE_URL")?;
    let pool = PgPool::connect(&url).await?;
    let q = Queries::new(pool);

    // Create
    let params = CreateUserParams {
        name: "Alice".into(),
        email: "alice@example.com".into(),
        age: 30,
    };
    let user = q.create_user(&params).await?;
    println!("Created: {} (id={})", user.name, user.id);

    // Read
    let found = q.get_user(user.id).await?;
    println!("Found: {:?}", found);

    // List
    let all = q.list_users().await?;
    println!("{} users total", all.len());

    // Delete
    q.delete_user(user.id).await?;

    Ok(())
}
```

## Incremental Generation

FlashORM caches checksums and only regenerates changed files:

```bash
flash gen       # Only regenerates modified query files
flash gen -f    # Force full regeneration
```

## Error Handling

All generated methods return `Result<T, sqlx::Error>`. Use standard Rust error handling:

```rust
match q.get_user(999).await {
    Ok(user) => println!("Found: {}", user.name),
    Err(sqlx::Error::RowNotFound) => println!("User not found"),
    Err(e) => eprintln!("Database error: {e}"),
}
```
