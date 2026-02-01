---
title: Database Seeding
description: Populate your database with realistic test data
---

# Database Seeding

Flash ORM includes a powerful seeding system that automatically generates realistic test data for your database tables.

## Quick Start

```bash
# Seed all tables with default count (10 rows each)
flash seed

# Seed with custom row count
flash seed --count 100

# Seed specific table
flash seed --table users --count 50

# Seed multiple tables with different counts
flash seed users:100 posts:500 comments:1000

# Truncate before seeding (fresh start)
flash seed --truncate

# Skip confirmation prompts
flash seed --truncate --force
```

## Features

### Smart Data Generation

FlashORM automatically generates appropriate data based on column names and types:

| Column Pattern | Generated Data |
|---------------|----------------|
| `email` | realistic emails (john.doe@example.com) |
| `name`, `first_name`, `last_name` | human names |
| `phone` | phone numbers |
| `url`, `website` | URLs |
| `address`, `city`, `country` | location data |
| `created_at`, `updated_at` | timestamps |
| `password` | hashed passwords |
| `uuid`, `id` | UUIDs or auto-increment |
| `price`, `amount` | currency values |
| `description`, `bio` | lorem ipsum text |

### Foreign Key Support

FlashORM automatically handles relationships:

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
```

When you run `flash seed`:
1. Seeds `users` table first
2. Uses existing user IDs when seeding `posts`
3. Maintains referential integrity

### Dependency Graph

FlashORM builds a dependency graph to determine the correct seeding order:

```
seeding order:
  1. users (no dependencies)
  2. categories (no dependencies)
  3. posts (depends on users)
  4. comments (depends on users, posts)
```

## Command Options

```bash
flash seed [tables...] [flags]
```

### Positional Arguments

You can specify tables with custom counts using the `table:count` syntax:

```bash
# Seed users with 100 rows, posts with 500, comments with 1000
flash seed users:100 posts:500 comments:1000
```

This is useful when you need different amounts of data for different tables.

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--count` | `-c` | Rows to generate per table | 10 |
| `--table` | `-t` | Seed specific table only | all |
| `--truncate` | | Truncate tables before seeding | false |
| `--force` | `-f` | Skip confirmation prompts | false |
| `--relations` | | Include foreign key relationships | true |

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
flash seed --table users --count 100

# Seed with fresh data
flash seed --table posts --truncate --count 200
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

## Output

```
🌱 Seeding database...
  Analyzing schema dependencies...
  Seeding order: users → posts → comments

  ✅ users: 50 records inserted
  ✅ posts: 100 records inserted  
  ✅ comments: 200 records inserted

✨ Seeding complete! 350 total records.
```

## Supported Databases

- ✅ PostgreSQL
- ✅ MySQL
- ✅ SQLite

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
flash raw db/seeds/admin_users.sql
```
