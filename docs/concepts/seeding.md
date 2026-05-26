---
title: Database Seeding
description: Populate your database with realistic test data automatically
---

# Database Seeding

Flash ORM includes a powerful seeding system that automatically generates realistic test data for your database tables. Foreign key relationships are handled automatically — no configuration needed.

## Quick Start

```bash
# Seed all tables with default count (10 rows each)
flash seed

# Seed with custom row count
flash seed --count 100

# Seed specific tables
flash seed users posts

# Seed multiple tables with different counts
flash seed users:100 posts:500 comments:1000

# Truncate before seeding (fresh start)
flash seed --truncate

# Skip confirmation prompts
flash seed --truncate --force

# Preview without inserting
flash seed --dry-run

# Exclude specific tables
flash seed --exclude logs,sessions
```

## How It Works

### Automatic Foreign Key Handling

FlashORM reads your schema, builds a dependency graph, and seeds tables in the correct order:

```sql
-- Given this schema:
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100)
);

CREATE TABLE posts (
  id SERIAL PRIMARY KEY,
  user_id INTEGER REFERENCES users(id),
  title VARCHAR(200)
);

CREATE TABLE comments (
  id SERIAL PRIMARY KEY,
  post_id INTEGER REFERENCES posts(id),
  content TEXT
);
```

```bash
flash seed users:50 posts:200 comments:1000
```

FlashORM automatically:
1. Seeds `users` first (no dependencies)
2. Seeds `posts` with valid `user_id` values from seeded users
3. Seeds `comments` with valid `post_id` values from seeded posts
4. Handles self-referencing FKs (e.g., `parent_id` in comment threads)

### Smart Data Generation

FlashORM generates appropriate data based on column names and types:

| Column Pattern | Generated Data |
|---------------|----------------|
| `email` | realistic emails (`john.doe1@gmail.com`) |
| `username`, `login` | unique usernames |
| `password`, `pwd` | secure random strings |
| `token`, `api_key` | hex tokens |
| `name`, `first_name`, `last_name` | human names |
| `phone`, `tel` | phone numbers |
| `url`, `website` | URLs |
| `slug`, `permalink` | URL-friendly slugs |
| `address`, `city`, `country` | location data |
| `ip_address`, `ip` | IPv4 addresses |
| `color`, `hex` | `#RRGGBB` colors |
| `created_at` / `updated_at` | coordinated timestamps |
| `price`, `amount` | currency values |
| `is_active`, `is_verified` | booleans |
| `has_permission`, `can_edit` | booleans |
| `description`, `bio`, `content` | lorem text |
| `status`, `role`, `category` | predefined values |
| `uuid`, `id` | UUIDs or auto-increment |
| `metadata`, `meta` | JSON objects |
| `hash`, `checksum` | hex digests |

### Supported SQL Types

All common SQL types are handled automatically:

- **Integers**: `INT`, `BIGINT`, `SMALLINT`, `TINYINT`, `SERIAL`
- **Strings**: `VARCHAR`, `TEXT`, `CHAR`
- **Booleans**: `BOOLEAN`, `BOOL`
- **Temporal**: `TIMESTAMP`, `TIMESTAMPTZ`, `DATETIME`, `DATE`, `TIME`, `YEAR`
- **Numeric**: `DECIMAL`, `NUMERIC`, `FLOAT`, `REAL`, `DOUBLE`
- **Special**: `UUID`, `JSON`, `JSONB`, `BYTEA`, `BLOB`, `ARRAY`, `ENUM`

---

## Command Reference

```bash
flash seed [tables...] [flags]
```

### Positional Arguments

Specify tables with optional counts using `table:count` syntax:

```bash
# Seed users with 100 rows, posts with 500
flash seed users:100 posts:500

# Schema-qualified names work too
flash seed public.users:50
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--count` | `-c` | Rows per table | 10 |
| `--truncate` | `-t` | Truncate tables before seeding | false |
| `--force` | `-f` | Skip confirmations | false |
| `--dry-run` | `-d` | Preview data without inserting | false |
| `--exclude` | `-x` | Comma-separated tables to skip | — |

---

## Examples

### Development Setup

```bash
# Fresh database with test data
flash reset --force
flash apply
flash seed --count 50
```

### Seed Specific Tables

```bash
# Seed only users table
flash seed users --count 100

# Seed with fresh data
flash seed posts --truncate --count 200
```

### Multiple Tables with Different Counts

```bash
# Realistic data distribution
flash seed users:50 posts:200 comments:1000

# E-commerce example
flash seed categories:10 products:100 orders:500 order_items:2000

# Social media example
flash seed users:100 posts:500 likes:5000 follows:2000
```

### Preview Before Inserting

```bash
# See sample data without modifying the database
flash seed --dry-run
flash seed users:5 posts:10 --dry-run
```

### Exclude Tables

```bash
# Skip audit/sensitive tables
flash seed --exclude audit_logs,sessions,password_resets
```

### Large Dataset

```bash
# Generate large dataset for performance testing
flash seed --count 1000 --force

# Stress test with specific distributions
flash seed users:10000 posts:50000 comments:200000 --force
```

### CI/CD Pipeline

```bash
# Automated test setup
flash reset --force && flash apply && flash seed --count 25 --force
```

---

## Output

```
🌱 Starting database seeding...
📊 Found 5 tables
📋 Insertion order: users → categories → posts → comments → likes

🔒 Transaction started
  📝 Seeding users (50 records)...
  ✅ users seeded successfully (50 records)
  📝 Seeding categories (10 records)...
  ✅ categories seeded successfully (10 records)
  📝 Seeding posts (200 records)...
  ✅ posts seeded successfully (200 records)
  📝 Seeding comments (1000 records)...
  ✅ comments seeded successfully (1000 records)
  📝 Seeding likes (5000 records)...
  ✅ likes seeded successfully (5000 records)
🔓 Transaction committed

✅ Database seeding completed successfully!
```

---

## Supported Databases

- ✅ PostgreSQL — Multi-row `RETURNING` for fast batch inserts
- ✅ MySQL — Single-row inserts with `LAST_INSERT_ID()` for ID tracking
- ✅ SQLite — Multi-row inserts when possible, `last_insert_rowid()` for IDs

---

## Tips

### For Testing

```bash
# Reset and seed before each test run
flash reset --force && flash apply && flash seed
```

### For Demo Data

```bash
# Create realistic demo environment
flash seed --count 25  # Just enough to look populated
```

### For Load Testing

```bash
# Large dataset
flash seed --count 10000 --force
```

### Custom Seed Data

For custom seed data beyond auto-generation, create SQL seed files:

```sql
-- db/seeds/admin_users.sql
INSERT INTO users (name, email, role) VALUES
  ('Admin', 'admin@example.com', 'admin'),
  ('Support', 'support@example.com', 'support');
```

Then run:
```bash
flash raw --file db/seeds/admin_users.sql
```
