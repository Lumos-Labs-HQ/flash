<h1 align="center">⚡ Flash ORM</h1>

<p align="center">
  <a href="https://go.dev/doc/go1.23">
    <img src="https://img.shields.io/badge/Go-1.23%2B-blue.svg" alt="Go Version">
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/License-MIT-green.svg" alt="License: MIT">
  </a>
  <a href="https://github.com/Lumos-Labs-HQ/flash">
    <img src="https://img.shields.io/github/v/release/Lumos-Labs-HQ/flash?label=Release" alt="Release">
  </a>
  <a href="https://www.npmjs.com/package/flashorm">
    <img src="https://img.shields.io/npm/v/flashorm?color=blue&label=npm" alt="npm version">
  </a>
  <a href="https://pypi.org/project/flashorm/">
    <img src="https://img.shields.io/pypi/v/flashorm?color=green&label=python" alt="PyPI version">
  </a>
</p>

<p align="center">
  <a href="docs/USAGE_GO.md">📗 Go Guide</a> •
  <a href="docs/USAGE_TYPESCRIPT.md">📘 TypeScript Guide</a> •
  <a href="docs/USAGE_PYTHON.md">📙 Python Guide</a> •
  <a href="docs/notes/RELEASE_NOTES.md">📋 Release Notes</a>
</p>

![image](/img/flash-orm.png)

---

A powerful, database-agnostic ORM built in Go with multi-database support and type-safe code generation for **Go, JavaScript/TypeScript, Python, Kotlin, and Java**.

## ✨ Features

- 🗃️ **Multi-Database Support**: PostgreSQL, MySQL, SQLite, ScyllaDB/Cassandra, ClickHouse (full ORM)
- 🔄 **Migration Management**: Create, apply, and track migrations with transaction safety
- 📤 **Smart Export System**: JSON, CSV, SQLite formats
- 🔧 **Code Generation**: Type-safe code for Go, TypeScript/JavaScript, Python, **Kotlin**, and **Java**
- 🌱 **Database Seeding**: Generate realistic fake data for development
- ⚡ **Blazing Fast**: Outperforms Drizzle and Prisma in benchmarks
- 📊 **FlashORM Studio**: Visual database management for SQL, MongoDB, and Redis

## 📊 Performance

| Operation | FlashORM | Drizzle | Prisma |
|-----------|----------|---------|--------|
| Insert 1000 Users | **149ms** | 224ms | 230ms |
| Complex Query x500 | **3156ms** | 12500ms | 56322ms |
| Mixed Workload x1000 | **186ms** | 1174ms | 10863ms |
| **TOTAL** | **5980ms** | **17149ms** | **71510ms** |

**2.8x faster** than Drizzle, **11.9x faster** than Prisma

## 🚀 Installation

### Direct installer by platform

#### Linux

```bash
curl -fsSL https://lumos-labs-hq.github.io/flash/install.sh | bash
```

#### macOS

```bash
curl -fsSL https://lumos-labs-hq.github.io/flash/install.sh | bash
```

#### Windows PowerShell

```powershell
irm https://lumos-labs-hq.github.io/flash/install.ps1 | iex
```

### Package manager alternatives

```bash
# NPM (Node.js/TypeScript)
npm install -g flashorm

# Python
pip install flashorm

# Go
go install github.com/Lumos-Labs-HQ/flash@latest
```

## 🏁 Quick Start

```bash
# 1. Initialize project (auto-detects Go/Node/Python/Kotlin/Java)
flash init --postgresql  # or --mysql, --sqlite, --scylla, --clickhouse

# 2. Set database URL
export DATABASE_URL="postgres://user:pass@localhost:5432/mydb"

# 3. Create and apply migrations
flash migrate "create users table"
flash apply

# 4. Generate type-safe code
flash gen
```

## 📋 Commands

| Command | Description |
|---------|-------------|
| `flash init` | Initialize project |
| `flash migrate <name>` | Create migration |
| `flash apply` | Apply migrations |
| `flash down` | Rollback migration |
| `flash status` | Show status |
| `flash pull` | Extract schema from database |
| `flash studio` | Launch visual editor |
| `flash export` | Export database |
| `flash seed` | Seed with fake data |
| `flash gen` | Generate type-safe code |
| `flash gen -f` | Force regenerate (skip cache) |
| `flash gen --db <name>` | Generate for specific database |
| `flash dblist` | List all configured databases |
| `flash update` | Update plugins and flash binary |
| `flash uninstall` | Remove flash and ~/.flash |

## 🗄️ Multi-Database Support

Configure multiple databases in one project:

```toml
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

[databases.gen.go]
enabled = true
out = "gen/analytics"
```

All commands support `--db <name>`: `flash gen --db main`, `flash apply --db analytics`, `flash studio --db main`

## 🗄️ Database Support

| Database | ORM Support | Studio |
|----------|-------------|--------|
| PostgreSQL | ✅ Full | ✅ SQL Studio |
| MySQL | ✅ Full | ✅ SQL Studio |
| SQLite | ✅ Full | ✅ SQL Studio |
| ScyllaDB / Cassandra | ✅ Beta | ✅ SQL Studio |
| ClickHouse | ✅ Beta | ✅ SQL Studio |
| MongoDB | ❌ | ✅ Visual Management |
| Redis | ❌ | ✅ Visual Management |

## 🔧 Code Generation

FlashORM generates type-safe query code from your SQL files. Auto-detects project type on `flash init`.

### Go
```toml
[gen.go]
enabled = true
out = "flash_gen"
driver = "pgx"  # or "database/sql" (default)
```

### TypeScript / JavaScript
```toml
[gen.js]
enabled = true
out = "flash_gen"
driver = "pg"  # pg | postgres | mysql2 | better-sqlite3 | bun:sqlite
```

### Python
```toml
[gen.python]
enabled = true
out = "flash_gen"
async = true
driver = "asyncpg"  # asyncpg | psycopg3 | pymysql | aiosqlite | sqlite3
```

### Kotlin *(new in 2.6.0)*
```toml
[gen.kotlin]
enabled = true
out = "src/main/kotlin/com/example/db/flashgen"
package = "com.example.db.flashgen"
driver = "jdbc"  # jdbc (default) | exposed | r2dbc
```

Generates: `Models.kt`, `UsersQueries.kt`, `Queries.kt` (unified entry point)

```kotlin
val q = Queries.newq(conn)
val user = q.getUser(42)           // Users?
val posts = q.getPostsByUser(42)   // List<GetPostsByUserRow>
```

### Java *(new in 2.6.0)*
```toml
[gen.java]
enabled = true
out = "src/main/java/com/example/db/flashgen"
package = "com.example.db.flashgen"
driver = "jdbc"  # jdbc (default) | jooq | hibernate
```

Generates: `Users.java`, `UsersQueries.java`, `Queries.java` (unified entry point)

```java
var q = Queries.newq(conn);
Users user = q.getUser(42);                          // Users
List<GetPostsByUserRow> posts = q.getPostsByUser(42); // List<GetPostsByUserRow>
```

## 🔧 Configuration

```toml
version = "2"
schema_dir = "db/schema"
queries = "db/queries/"
migrations_path = "db/migrations"
env_path = "config/.env"   # optional: custom .env file path

[database]
provider = "postgresql"
url_env = "DATABASE_URL"

[gen.go]
enabled = true
```

## 📊 FlashORM Studio

### SQL Studio (PostgreSQL, MySQL, SQLite, ScyllaDB, ClickHouse)

```bash
flash studio
flash studio "postgres://user:pass@localhost:5432/mydb"
flash studio "scylla://localhost:9042/keyspace"
flash studio "clickhouse://localhost:9000"
```

Features: Schema designer, data browser, relationship visualization, auto-migration creation

### MongoDB Studio

```bash
flash studio "mongodb://localhost:27017/mydb"
flash studio "mongodb+srv://user:pass@cluster.mongodb.net/mydb"
```

### Redis Studio

```bash
flash studio "redis://localhost:6379"
flash studio "redis://:password@localhost:6379"
```

## 🌱 Database Seeding

```bash
flash seed                                    # seed all tables
flash seed --count 100                        # custom count
flash seed users:100 posts:500 comments:1000  # per-table counts
flash seed --truncate --force                 # truncate and reseed
```

## 📤 Export System

```bash
flash export           # JSON (default)
flash export --csv     # CSV files per table
flash export --sqlite  # Portable SQLite file
```

## 🛠️ Advanced Usage

```bash
flash apply --force        # Production deployment
flash reset --force        # Reset database (development)
flash pull                 # Extract schema from existing database
flash raw "SELECT COUNT(*) FROM users;"
flash raw db/seed.sql      # Execute SQL file (supports comment blocks)
```

## 📚 Documentation

- [Go Usage Guide](docs/USAGE_GO.md)
- [TypeScript Usage Guide](docs/USAGE_TYPESCRIPT.md)
- [Python Usage Guide](docs/USAGE_PYTHON.md)
- [How It Works](docs/HOW_IT_WORKS.md)
- [Architecture & Internals](ARCHITECTURE.md)
- [Release Notes](docs/notes/RELEASE_NOTES.md)
- [Contributing](docs/contributing.md)

## 🤝 Contributing

```bash
git clone https://github.com/Lumos-Labs-HQ/flash.git
cd flash
make dev-setup
make build-all
```

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

<p align="center">
  Built with ❤️ by <a href="https://github.com/Lumos-Labs-HQ">Lumos Labs</a>
</p>
