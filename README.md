# 🪴 Flash ORM

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-blue.svg)](https://go.dev/doc/go1.23)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/Lumos-Labs-HQ/flash?label=Release)](https://github.com/Lumos-Labs-HQ/flash/releases>)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen)](#)

![image](.github/flash-orm.png)
---

A powerful, database-agnostic ORM built in Go that provides Prisma-like functionality with multi-database support and type-safe code generation for Go, JavaScript, and TypeScript.

## ✨ Features

- 🗃️ **Multi-Database Support**: PostgreSQL, MySQL, SQLite  
- 🔄 **Migration Management**: Create, apply, and track migrations  
- 🔒 **Safe Migration System**: Transaction-based execution with automatic rollback  
- 📤 **Smart Export System**: Multiple formats (JSON, CSV, SQLite) for data portability  
- 🔧 **Code Generation**: Generate type-safe code for Go and JavaScript/TypeScript  
- 🟢 **Node.js Support**: First-class JavaScript/TypeScript support with type definitions  
- 🎨 **Enum Support**: PostgreSQL ENUM types with full migration support  
- ⚡ **Blazing Fast**: Outperforms Drizzle and Prisma in benchmarks  
- 🎯 **Prisma-like Commands**: Familiar CLI interface  
- 🔍 **Schema Introspection**: Pull schema from existing databases  
- 📊 **Flash ORM Studio**: Visual data browser and editor for database inspection  
- 🛡️ **Conflict Detection**: Automatic detection and resolution of migration conflicts  

---

## 📊 Performance Benchmarks

Flash ORM significantly outperforms popular ORMs in real-world scenarios.

### 🔹 Summary Chart

| ORM | Relative Performance | Efficiency Ratio |
|------|----------------------|------------------|
| **Flash ORM** | 🟢 100% (Baseline) | **1.0x** |
| **Drizzle** | 🟡 ~42% slower | 2.5x less efficient |
| **Prisma** | 🔴 ~90% slower | 10x less efficient |

### 📈 Detailed Metrics

| Operation | Flash ORM | Drizzle | Prisma |
|-----------|-------|---------|--------|
| Insert 1000 Users | **158ms** | 224ms | 230ms |
| Insert 10 Cat + 5K Posts + 15K Comments | **2410ms** | 3028ms | 3977ms |
| Complex Query ×500 | **4071ms** | 12500ms | 56322ms |
| Mixed Workload ×1000 (75% read, 25% write) | **186ms** | 1174ms | 10863ms |
| Stress Test Simple Query ×2000 | **122ms** | 160ms | 223ms |
| **TOTAL** | **6947ms** | **17149ms** | **71551ms** |

---

## 🚀 Installation

### NPM (Node.js / TypeScript Projects)

```bash
npm install -g Flash ORM-orm
````

### Go Install

```bash
go install github.com/Lumos-Labs-HQ/Flash ORM@latest
```

### From Source

```bash
git clone https://github.com/Lumos-Labs-HQ/Flash ORM.git
cd Flash ORM
make build-all
```

### Download Binary

Download the latest binary from [Releases](<https://github.com/Lumos-Labs-HQ/Flash> ORM/releases).

---

## 🏁 Quick Start

### 1. Initialize Your Project

```bash
cd your-project
Flash ORM init --postgresql  # or --mysql, --sqlite
```

### 2. Configure Database

```bash
export DATABASE_URL="postgres://user:password@localhost:5432/mydb"
echo "DATABASE_URL=postgres://user:password@localhost:5432/mydb" > .env
```

### 3. Create Your First Migration

```bash
Flash ORM migrate "create users table"
```

### 4. Apply Migrations Safely

```bash
Flash ORM apply
```

### 5. Check Status

```bash
Flash ORM status
```

---

## 📋 Commands

| Command                 | Description                                         |
| ----------------------- | --------------------------------------------------- |
| `flash init`            | Initialize project with database-specific templates |
| `Flash migrate <name>`  | Create a new migration file                         |
| `Flash apply`           | Apply pending migrations with transaction safety    |
| `Flash status`          | Show migration status                               |
| `Flash pull`            | Extract schema from existing database               |
| `Flash studio`          | Launch Studio visual data browser                   |
| `Flash export [format]` | Export database (JSON, CSV, SQLite)                 |
| `Flash reset`           | Reset database (⚠️ destructive)                     |
| `Flash gen`             | Generate SQLC types                                 |
| `Flash raw <sql>`       | Execute raw SQL                                     |

### Global Flags

- `--force` - Skip confirmation prompts
- `--help` - Show help

---

## 🗄️ Database Support

### PostgreSQL

```bash
flash init --postgresql
export DATABASE_URL="postgres://user:pass@localhost:5432/db"
```

### MySQL

```bash
flash init --mysql
export DATABASE_URL="user:pass@tcp(localhost:3306)/db"
```

### SQLite

```bash
flash init --sqlite
export DATABASE_URL="sqlite://./database.db"
```

---

## 🔧 Configuration

Flash ORM uses `flash.config.json` for configuration.

```json
{
  "version": "2",
  "schema_path": "db/schema/schema.sql",
  "queries": "db/queries/",
  "migrations_path": "db/migrations",
  "export_path": "db/export",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "js": { "enabled": true }
  }
}
```

---

## 📁 Project Structure

After running `flash init`:

```bash
your-project/
├── flash.config.json      # Flash ORM configuration
├── .env                   # Environment variables
└── db/
    ├── schema/            # Database schema
    ├── queries/           # SQL queries for SQLC
    ├── migrations/        # Migration files (auto-created)
    └── export/            # Export files (auto-created)
```

### Directory Notes

- **`db/schema/`** — Contains your declarative database schema.
- **`db/migrations/`** — Auto-generated, timestamped SQL migration files.
- **`db/export/`** — Stores exported data files (JSON, CSV, SQLite).

---

## 🔒 Safe Migration System

Each migration runs in a [transaction](https://www.postgresql.org/docs/current/sql-transaction.html) and automatically rolls back on failure.

### Example Output

```bash
📦 Applying 2 migration(s)...
  [1/2] 20251021132902_init ✅
  [2/2] 20251021140530_add_users_index ✅

✅ All migrations applied successfully
```

On error:

```
❌ Failed at migration: 20251021140530_bad_migration
Transaction rolled back. Fix and re-run 'Flash ORM apply'.
```

---

## 🔄 Migration Workflow

1. **Create Migration**

   ```bash
   Flash ORM migrate "add user roles"
   ```

2. **Apply Migrations**

   ```bash
   Flash ORM apply
   ```

3. **Check Status**

   ```bash
   Flash ORM status
   ```

---

## 🧭 Studio (Visual Editor)

Run the optional visual data editor:

```bash
flash studio
```

Or open it directly with a connection:

```bash
flash studio --db "postgresql://jack:secret123@localhost:5432/mydb"
```

Default interface: [http://localhost:5555](http://localhost:5555)

### Built With

- [React](https://react.dev/)
- [Vite](https://vitejs.dev/)
- [Tailwind CSS](https://tailwindcss.com/)
- [Inter Font](https://rsms.me/inter/)

---

## 📤 Export System

Export databases in various formats:

### JSON Export

```bash
flash export --json
```

### CSV Export

```bash
flash export --csv
```

### SQLite Export

```bash
flash export --sqlite
```

---

## 🔗 SQLC Integration

Generate type-safe Go code from SQL:

```bash
flash apply && flash gen
```

Uses [SQLC](https://docs.sqlc.dev/en/latest/index.html) for type-safe Go query generation.

---

## 🧠 Advanced Usage

### Production Deployment

```bash
flash apply --force
```

### Development Workflow

```bash
flash reset --force
flash pull
```

### Raw SQL Execution

```bash
flash raw "SELECT COUNT(*) FROM users;"
```

---

## 🚀 Roadmap

- 🐍 Python Support
- 🌐 WebAssembly bindings
- 🧩 Schema visualizer

---

## 🐛 Troubleshooting

**Database Connection Failed**

```bash
Error: failed to connect to database
```

- Verify `DATABASE_URL`
- Ensure DB service is running

**Migration Failed**

```bash
❌ Transaction rolled back
```

- Fix the SQL syntax and re-run

---

## 🤝 Contributing

```bash
git clone https://github.com/Lumos-Labs-HQ/Flash ORM.git
cd flash
make dev-setup
make build-all
```

---

## 📄 License

MIT License — see [LICENSE](LICENSE).

---

## 🙏 Acknowledgments

- Inspired by [Prisma](https://www.prisma.io/)
- Built with [Cobra CLI](https://github.com/spf13/cobra)
- Database drivers:

  - [pgx](https://github.com/jackc/pgx)
  - [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)
  - [go-sqlite3](https://github.com/mattn/go-sqlite3)
