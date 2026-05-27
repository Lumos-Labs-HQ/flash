---
title: Seeding Patterns
description: Database seeding patterns and examples for realistic test data
---

# Seeding Patterns

Flash ORM's seeding system automatically generates realistic fake data. This page shows patterns for common scenarios.

---

## Basic Seeding

### Seed All Tables

```bash
# Default: 10 records per table
flash seed

# Custom count for all tables
flash seed --count 100

# Truncate and reseed
flash seed --truncate --count 50
```

### Seed Specific Tables

```bash
# Seed only users and posts
flash seed users posts

# Seed with different counts
flash seed users:50 posts:200 comments:1000

# Exclude tables you don't want seeded
flash seed --exclude logs,audit_logs,sessions
```

### Dry Run (Preview)

```bash
# See what would be generated without inserting
flash seed --dry-run

# Preview specific tables
flash seed users:5 posts:10 --dry-run
```

---

## Realistic Data Distribution

### Blog Platform

```bash
# Seed users first (authors and readers)
flash seed users:20

# Then seed content
flash seed categories:5 tags:30 posts:100 comments:500

# Or all at once (Flash handles order automatically)
flash seed users:20 categories:5 tags:30 posts:100 comments:500
```

### E-Commerce

```bash
# Product catalog
flash seed categories:10 products:500

# With customer activity
flash seed users:1000 products:500 orders:5000 order_items:15000 reviews:2000
```

### Social Media

```bash
# Network effect data
flash seed users:100 posts:500 comments:2000 likes:10000 follows:2000
```

### SaaS/Multi-Tenant

```bash
# Tenant structure
flash seed tenants:5 users:50 projects:100 tasks:500
```

---

## Automated Column Generation

Flash automatically generates data based on column names and types:

### Identity Columns

| Column Name | Generated Value |
|-------------|----------------|
| `email`, `user_email` | `john.doe1@gmail.com` |
| `username`, `login` | `john_doe_1` |
| `first_name`, `fname` | `John` |
| `last_name`, `lname` | `Doe` |
| `password`, `pwd` | `aB3$kL9mPqR2` |
| `name` | `John Doe` |

### Content Columns

| Column Name | Generated Value |
|-------------|----------------|
| `title`, `headline` | `Getting Started with Go Programming` |
| `content`, `body` | Lorem ipsum paragraph |
| `bio`, `about` | Short biography paragraph |
| `description`, `summary` | Sample description paragraph |
| `slug`, `permalink` | `getting-started-42` |
| `phone`, `tel` | `+1-234-567-8901` |

### Location Columns

| Column Name | Generated Value |
|-------------|----------------|
| `address`, `addr` | `123 Main Street, New York, NY 12345` |
| `city` | `New York` |
| `state`, `province` | `California` |
| `zip`, `postal` | `12345` |
| `country` | `US` |
| `latitude`, `lat` | `40.7128` |
| `longitude`, `lng` | `-74.0060` |

### Technical Columns

| Column Name | Generated Value |
|-------------|----------------|
| `ip_address`, `ip` | `192.168.1.1` |
| `color`, `hex` | `#3f7a9c` |
| `token`, `api_key` | `a1b2c3d4e5f6...` (hex) |
| `hash`, `checksum` | `a1b2c3d4...` (hex) |
| `metadata`, `meta` | `{"generated": true}` |
| `locale`, `lang` | `en` |
| `currency` | `USD` |

### Business Columns

| Column Name | Generated Value |
|-------------|----------------|
| `company`, `organization` | `Tech Solutions Inc` |
| `product` | `Premium Subscription` |
| `status` | `active` |
| `category` | `Technology` |
| `priority` | `medium` |
| `role` | `user` |
| `gender` | `male` |
| `tag`, `label` | `technology` |

### Numeric Columns

| Column Name | Generated Value |
|-------------|----------------|
| `price`, `amount` | `$1,234.56` |
| `quantity`, `qty` | `1-1000` |
| `rating`, `score` | `1-5` |
| `percent`, `percentage` | `0-100%` |
| `progress` | `0-100` |
| `sort_order` | `0-999` |
| `age` | `18-97` |
| `version` | `2.14.3` |

### Boolean Columns

Boolean prefixes are automatically detected:

| Column Name | Generated Value |
|-------------|----------------|
| `is_active` | `true`/`false` |
| `is_verified` | `true`/`false` |
| `has_permission` | `true`/`false` |
| `can_edit` | `true`/`false` |
| `is_published`, `published` | `true`/`false` |
| `is_featured`, `featured` | `true`/`false` |
| `deleted`, `archived`, `locked` | `true`/`false` |

### Temporal Columns

Coordinated timestamps: when both `created_at` and `updated_at` exist, `updated_at` is always >= `created_at`.

| Column Name | Generated Value |
|-------------|----------------|
| `created_at` | Past date (up to 1 year ago) |
| `updated_at` | Between `created_at` and now |
| `published_at` | Past date |
| `dob`, `birth_date` | `18-78` years ago |
| `duration` | `1-3600` seconds |

---

## SQL Type Handling

Flash handles all common SQL types:

| SQL Type | Generated Value |
|----------|----------------|
| `INTEGER`, `INT` | `1-1,000,000` |
| `BIGINT` | `int64` value |
| `SMALLINT` | `1-32,767` |
| `TINYINT` | `1-127` |
| `SERIAL` | Auto-increment (skipped) |
| `VARCHAR`, `TEXT` | Random sentence |
| `CHAR` | `6-char` code |
| `BOOLEAN`, `BOOL` | `true`/`false` |
| `TIMESTAMP`, `DATETIME` | Past date |
| `TIMESTAMPTZ` | Past date with timezone |
| `DATE` | Past date (`time.Time`) |
| `TIME` | `HH:MM:SS` |
| `DECIMAL`, `NUMERIC` | `$0-$10,000` |
| `FLOAT`, `REAL`, `DOUBLE` | Float value |
| `UUID` | RFC 4122 v4 UUID |
| `JSON`, `JSONB` | `{"generated": true}` |
| `BYTEA`, `BLOB` | Binary data |
| `ARRAY` | `{item1,item2,item3}` |
| `ENUM('a','b')` | Random enum value |

---

## CI/CD Integration

### Test Database Setup

```bash
#!/bin/bash
# scripts/setup-test-db.sh

set -e

export DATABASE_URL="postgres://test:test@localhost:5432/test_db"

# Reset and setup
flash reset --force
flash apply --force
flash seed --count 25 --force

echo "Test database ready!"
```

### GitHub Actions

```yaml
# .github/workflows/test.yml
name: Tests

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4
      - name: Setup Flash ORM
        run: |
          go install github.com/Lumos-Labs-HQ/flash@latest
          flash init --postgresql
      - name: Setup Database
        run: |
          flash apply --force
          flash seed --count 10 --force
      - name: Run Tests
        run: go test ./...
```

---

## Common Workflows

### Fresh Development Environment

```bash
# Complete reset and seed
flash reset --force
flash apply
flash seed --count 50
flash studio
```

### Adding Test Data After Schema Change

```bash
# Migrate, then seed new tables
flash migrate "add reviews table"
flash apply
flash seed reviews:100
```

### Performance Testing

```bash
# Large dataset for load testing
flash seed --count 10000 --force

# Specific distribution
flash seed users:10000 posts:50000 comments:200000 --force
```

### Demo Environment

```bash
# Just enough data to look realistic
flash seed --truncate --count 25
```
