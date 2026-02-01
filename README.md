<h1 align="center">⚡ Flash ORM</h1>

<p align="center">
  <a href="https://go.dev/doc/go1.23">
    <img src="https://img.shields.io/badge/Go-1.23%2B-blue.svg" alt="Go Version">
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/License-MIT-green.svg" alt="License: MIT">
  </a>
  <a href="https://github.com/Lumos-Labs-HQ/flash/releases">
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
  <a href="RELEASE_NOTES.md">📋 Release Notes</a>
</p>

![image](.github/flash-orm.png)

---

A powerful, database-agnostic ORM built in Go with multi-database support and type-safe code generation for Go, JavaScript/TypeScript, and Python.

## ✨ Features

- 🗃️ **Multi-Database Support**: PostgreSQL, MySQL, SQLite (full ORM)
- 🔄 **Migration Management**: Create, apply, and track migrations with transaction safety
- 📤 **Smart Export System**: JSON, CSV, SQLite formats
- 🔧 **Code Generation**: Type-safe code for Go, JavaScript/TypeScript, and Python
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
# 1. Initialize project
flash init --postgresql  # or --mysql, --sqlite

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

## 🗄️ Database Support

| Database | ORM Support | Studio |
|----------|-------------|--------|
| PostgreSQL | ✅ Full | ✅ SQL Studio |
| MySQL | ✅ Full | ✅ SQL Studio |
| SQLite | ✅ Full | ✅ SQL Studio |
| MongoDB | ❌ | ✅ Visual Management |
| Redis | ❌ | ✅ Visual Management |

## 📊 FlashORM Studio

### SQL Studio (PostgreSQL, MySQL, SQLite)

```bash
flash studio
flash studio --db "postgres://user:pass@localhost:5432/mydb"
```

Features: Schema designer, data browser, relationship visualization, auto-migration creation

### MongoDB Studio

Visual management interface for MongoDB (no ORM features).

```bash
flash studio --db "mongodb://localhost:27017/mydb"
flash studio --db "mongodb+srv://user:pass@cluster.mongodb.net/mydb"
```

Features: Collection browser, document editor, query interface, bulk operations

### Redis Studio

Visual management interface for Redis (no ORM features).

```bash
flash studio --redis "redis://localhost:6379"
flash studio --redis "redis://:password@localhost:6379"
```

Features: Key browser, CLI terminal, memory analysis, Lua scripting, Pub/Sub, ACL management

## 🌱 Database Seeding

```bash
# Seed all tables
flash seed

# Custom count
flash seed --count 100

# Multiple tables with different counts
flash seed users:100 posts:500 comments:1000

# Truncate and reseed
flash seed --truncate --force
```

Smart data generation based on column names (email, name, phone, etc.)

## 🔧 Configuration

```json
{
  "version": "2",
  "schema_dir": "db/schema",
  "queries": "db/queries/",
  "migrations_path": "db/migrations",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "go": { "enabled": true },
    "js": { "enabled": true },
    "python": { "enabled": true }
  }
}
```

## 📤 Export System

```bash
flash export           # JSON (default)
flash export --csv     # CSV files per table
flash export --sqlite  # Portable SQLite file
```

## 🛠️ Advanced Usage

```bash
# Production deployment
flash apply --force

# Reset database (development)
flash reset --force

# Extract schema from existing database
flash pull

# Execute raw SQL
flash raw "SELECT COUNT(*) FROM users;"
```

## 📚 Documentation

- [Go Usage Guide](docs/USAGE_GO.md)
- [TypeScript Usage Guide](docs/USAGE_TYPESCRIPT.md)
- [Python Usage Guide](docs/USAGE_PYTHON.md)
- [How It Works](docs/HOW_IT_WORKS.md)
- [Contributing](docs/CONTRIBUTING.md)

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
