---
title: Getting Started
description: Quick start guide for Flash ORM
---

# Getting Started

Welcome to Flash ORM! This guide will help you get up and running in minutes.

## Prerequisites

- **Go 1.24.2+** (for Go projects)
- **Node.js 14+** (for TypeScript/JavaScript projects)
- **Python 3.7+** (for Python projects)
- Database: PostgreSQL, MySQL, SQLite, or MongoDB

## Installation

### Option 1: NPM (Recommended for all platforms)

```bash
npm install -g flashorm
```

### Option 2: Python

```bash
pip install flashorm
```

### Option 3: Go

```bash
go install github.com/Lumos-Labs-HQ/flash@latest
```

### Option 4: Download Binary

Download the latest release from [GitHub Releases](https://github.com/Lumos-Labs-HQ/flash/releases).

## Quick Start

### 1. Initialize Your Project

Choose your database and initialize:

```bash
# For PostgreSQL (default)
flash init --postgresql

# For MySQL
flash init --mysql

# For SQLite
flash init --sqlite
```

### 2. Configure Database Connection

Create a `.env` file in your project root:

```env
# PostgreSQL
DATABASE_URL=postgres://user:password@localhost:5432/mydb

# MySQL
DATABASE_URL=mysql://user:password@localhost:3306/mydb

# SQLite
DATABASE_URL=sqlite://./data.db
```

### 3. Define Your Schema

Edit `db/schema/schema.sql`:

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 4. Create and Apply Migration

```bash
# Create migration
flash migrate "initial schema"

# Apply to database
flash apply
```

### 5. Generate Code

```bash
flash gen
```

## Language-Specific Guides

Choose your preferred language:

- **[Go Guide](/guides/go)** - Type-safe Go code generation
- **[TypeScript Guide](/guides/typescript)** - JavaScript/TypeScript with full type support
- **[Python Guide](/guides/python)** - Async Python with type hints

## Next Steps

- Learn about [Core Concepts](/concepts/schema)
- Explore [Database Support](/databases/postgresql)
- Check out [FlashORM Studio](/concepts/studio) for visual database management
- Read about [Migrations](/concepts/migrations) and [Branching](/concepts/branching)

## Need Help?

- 📖 [Full Documentation](/introduction/what-is-flash)
- 💬 [GitHub Issues](https://github.com/Lumos-Labs-HQ/flash/issues)
- 🐛 [Report a Bug](https://github.com/Lumos-Labs-HQ/flash/issues/new?template=bug_report.md)
- ✨ [Request a Feature](https://github.com/Lumos-Labs-HQ/flash/issues/new?template=feature_request.md)
