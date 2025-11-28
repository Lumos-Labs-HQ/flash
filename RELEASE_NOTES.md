# FlashORM v2.2.0 Release Notes

## 🎉 What's New

### 🔌 Plugin Architecture

FlashORM now uses a modular plugin system for a lightweight, customizable experience:

- **Base CLI** (~5-10 MB) - Minimal footprint with plugin management
- **Core Plugin** - Complete ORM features (migrations, codegen, export)
- **Studio Plugin** - Visual database editors (SQL, MongoDB, Redis)
- **All Plugin** - Everything bundled together

```bash
# Install base CLI
npm install -g flashorm

# Add plugins as needed
flash add-plug core      # ORM features only
flash add-plug studio    # Visual editors only
flash add-plug all       # Everything

# Manage plugins
flash plugins            # List installed
flash plugins --online   # Check available
flash rm-plug studio     # Remove plugin
```

---

### 🍃 MongoDB Studio

A beautiful, modern interface for MongoDB database management - similar to MongoDB Compass but in your browser!

**Launch MongoDB Studio:**
```bash
flash studio --db "mongodb://localhost:27017/mydb"
# or
flash studio --db "mongodb+srv://user:pass@cluster.mongodb.net/mydb"
```

**Features:**
- 📋 **Collection Browser** - View all collections with document counts
- 📄 **Document Viewer** - Browse documents with syntax-highlighted JSON
- ✏️ **Inline Editing** - Edit documents directly with JSON validation
- ➕ **Create Documents** - Add new documents with JSON editor
- 🗑️ **Delete Documents** - Remove documents with confirmation
- 🔍 **Search & Filter** - Query documents using MongoDB syntax
- 📊 **Database Stats** - View connection info and statistics
- 📋 **Copy as JSON** - One-click copy of any document

---

### 🔴 Redis Studio

A powerful Redis management interface inspired by Upstash Redis Studio with a real CLI terminal!

**Launch Redis Studio:**
```bash
flash studio --redis "redis://localhost:6379"
# or with password
flash studio --redis "redis://:password@localhost:6379"
```

**Features:**

#### 🗂️ Key Browser
- View all keys with type indicators (STRING, LIST, SET, HASH, ZSET)
- Search keys with pattern matching (e.g., `user:*`)
- TTL display and management
- Create new keys of any type
- Delete keys with confirmation

#### 💻 Real CLI Terminal
Full Redis CLI experience with inline input/output:
```
redis> SET mykey "hello"
OK
redis> GET mykey
"hello"
redis> KEYS *
1) "mykey"
redis> MEMORY STATS
peak.allocated: 1048576
total.allocated: 524288
...
redis> HSET user:1 name "John" age 30
(integer) 2
redis> HGETALL user:1
1) "name"
2) "John"
3) "age"
4) "30"
```

- ⬆️⬇️ Command history navigation
- All Redis commands supported
- Color-coded output (OK in green, errors in red)
- `clear` or `Ctrl+L` to clear terminal

#### 📊 Statistics Dashboard
- Memory usage and peak allocation
- Connected clients count
- Total keys across all databases
- Server uptime
- Redis version info

#### 🗄️ Database Selector
- Switch between db0-db15 (Redis's 16 isolated databases)
- Each database is completely isolated

#### 🧹 Purge Database
- One-click purge button to clear all keys in current database
- Confirmation dialog to prevent accidents

---

### 📊 SQL Studio Improvements

The SQL Studio has been enhanced with:
- Better dark theme with improved contrast
- Faster table loading with batch queries
- Enhanced inline editing experience
- Improved schema visualization

---

## 📦 Installation

### NPM (Recommended)
```bash
npm install -g flashorm
flash add-plug all   # Install all features
```

### Python
```bash
pip install flashorm
```

### Go
```bash
go install github.com/Lumos-Labs-HQ/flash@latest
```

### Direct Download
Download from [GitHub Releases](https://github.com/Lumos-Labs-HQ/flash/releases)

---

## 🚀 Quick Start

### SQL Databases (PostgreSQL, MySQL, SQLite)
```bash
flash init --postgresql
flash migrate "create users table"
flash apply
flash studio   # Visual editor
```

### MongoDB
```bash
flash studio --db "mongodb://localhost:27017/mydb"
```

### Redis
```bash
flash studio --redis "redis://localhost:6379"
```

---

## 📚 Documentation

- [Plugin System Guide](docs/PLUGIN_SYSTEM.md)
- [Technology Stack](docs/TECHNOLOGY_STACK.md)
- [Contributing Guide](docs/CONTRIBUTING.md)

---

## 💬 Feedback

- 🐛 [Report bugs](https://github.com/Lumos-Labs-HQ/flash/issues)
- 💡 [Request features](https://github.com/Lumos-Labs-HQ/flash/issues)
- ⭐ [Star the repo](https://github.com/Lumos-Labs-HQ/flash)

---
