---
title: Schema Reference
description: Complete reference for FlashORM schema definition syntax
---

# Schema Reference

This page provides a complete reference for defining database schemas in FlashORM.

## Overview

FlashORM uses SQL DDL (Data Definition Language) to define your database schema. Schema files are stored in the `db/schema/` directory with `.sql` extension.

## Supported Databases

FlashORM supports schema definition for:
- PostgreSQL
- MySQL
- SQLite
- MongoDB (limited schema support)

## Basic Table Creation

### PostgreSQL/MySQL/SQLite

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### SQLite Specific

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Data Types

### PostgreSQL

| SQL Type | Go Type | Description |
|----------|---------|-------------|
| `SERIAL` | `int32` | Auto-incrementing integer |
| `BIGSERIAL` | `int64` | Auto-incrementing big integer |
| `INTEGER` | `int32` | Integer |
| `BIGINT` | `int64` | Big integer |
| `VARCHAR(n)` | `string` | Variable length string |
| `TEXT` | `string` | Unlimited text |
| `BOOLEAN` | `bool` | True/false |
| `TIMESTAMP` | `time.Time` | Date and time |
| `TIMESTAMPTZ` | `time.Time` | Timestamp with timezone |
| `DATE` | `time.Time` | Date only |
| `UUID` | `string` | UUID string |
| `JSON` | `interface{}` | JSON data |
| `JSONB` | `interface{}` | Binary JSON |
| `BYTEA` | `[]byte` | Binary data |

### MySQL

| SQL Type | Go Type | Description |
|----------|---------|-------------|
| `INT AUTO_INCREMENT` | `int32` | Auto-incrementing integer |
| `BIGINT AUTO_INCREMENT` | `int64` | Auto-incrementing big integer |
| `INT` | `int32` | Integer |
| `BIGINT` | `int64` | Big integer |
| `VARCHAR(n)` | `string` | Variable length string |
| `TEXT` | `string` | Unlimited text |
| `TINYINT(1)` | `bool` | Boolean |
| `DATETIME` | `time.Time` | Date and time |
| `TIMESTAMP` | `time.Time` | Timestamp |
| `DATE` | `time.Time` | Date only |
| `JSON` | `interface{}` | JSON data |
| `BLOB` | `[]byte` | Binary data |

### SQLite

| SQL Type | Go Type | Description |
|----------|---------|-------------|
| `INTEGER PRIMARY KEY` | `int64` | Auto-incrementing integer |
| `INTEGER` | `int64` | Integer |
| `TEXT` | `string` | String/text |
| `REAL` | `float64` | Floating point |
| `BLOB` | `[]byte` | Binary data |

## Constraints

### Primary Keys

```sql
-- Single column
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255)
);

-- Composite primary key
CREATE TABLE user_permissions (
    user_id INTEGER,
    permission_id INTEGER,
    PRIMARY KEY (user_id, permission_id)
);
```

### Foreign Keys

```sql
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT
);
```

### Unique Constraints

```sql
-- Single column
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE
);

-- Multiple columns
CREATE TABLE user_follows (
    follower_id INTEGER,
    following_id INTEGER,
    UNIQUE(follower_id, following_id)
);
```

### Check Constraints

```sql
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) CHECK (price > 0),
    quantity INTEGER CHECK (quantity >= 0)
);
```

## Indexes

```sql
-- Single column index
CREATE INDEX idx_users_email ON users(email);

-- Composite index
CREATE INDEX idx_posts_user_created ON posts(user_id, created_at);

-- Unique index
CREATE UNIQUE INDEX idx_users_username ON users(username);

-- Partial index
CREATE INDEX idx_active_users ON users(created_at) WHERE active = true;
```

## Enums

### PostgreSQL

```sql
CREATE TYPE user_role AS ENUM ('admin', 'moderator', 'user');
CREATE TYPE post_status AS ENUM ('draft', 'published', 'archived');

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    role user_role NOT NULL DEFAULT 'user'
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    status post_status NOT NULL DEFAULT 'draft'
);
```

### MySQL

```sql
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    role ENUM('admin', 'moderator', 'user') NOT NULL DEFAULT 'user'
);
```

## Views

```sql
CREATE VIEW active_users AS
SELECT id, name, email
FROM users
WHERE active = true;

CREATE VIEW post_summary AS
SELECT
    p.id,
    p.title,
    u.name as author,
    COUNT(c.id) as comment_count
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN comments c ON p.id = c.post_id
GROUP BY p.id, p.title, u.name;
```

## Triggers

```sql
-- PostgreSQL
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
```

## Schema Organization

### Multiple Files

You can split your schema across multiple files:

```
db/schema/
├── 001_users.sql
├── 002_posts.sql
├── 003_comments.sql
└── 004_indexes.sql
```

### Naming Convention

- Use descriptive names for tables and columns
- Use snake_case for SQL identifiers
- Prefix related tables (e.g., `user_posts`, `user_profiles`)
- Use consistent naming patterns

## Advanced Features

### Partitioning (PostgreSQL)

```sql
CREATE TABLE logs (
    id SERIAL,
    level VARCHAR(10),
    message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) PARTITION BY RANGE (created_at);

CREATE TABLE logs_2024_01 PARTITION OF logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

### Inheritance (PostgreSQL)

```sql
CREATE TABLE person (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE TABLE employee (
    salary DECIMAL(10,2),
    department VARCHAR(100)
) INHERITS (person);
```

### Generated Columns (PostgreSQL/MySQL)

```sql
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    price DECIMAL(10,2),
    tax_rate DECIMAL(3,2) DEFAULT 0.08,
    total_price DECIMAL(10,2) GENERATED ALWAYS AS (price * (1 + tax_rate)) STORED
);
```

## Migration Considerations

When writing schemas that will be migrated:

1. **Add defaults** for new columns
2. **Make columns nullable** initially if needed
3. **Use safe operations** that can be rolled back
4. **Test migrations** on copies of production data

## Examples

### Blog Schema

```sql
-- Users
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Categories
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    description TEXT
);

-- Posts
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    content TEXT NOT NULL,
    excerpt TEXT,
    author_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
    published BOOLEAN DEFAULT FALSE,
    published_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Comments
CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    post_id INTEGER REFERENCES posts(id) ON DELETE CASCADE,
    author_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_posts_author ON posts(author_id);
CREATE INDEX idx_posts_category ON posts(category_id);
CREATE INDEX idx_posts_published ON posts(published, published_at);
CREATE INDEX idx_comments_post ON comments(post_id);
CREATE INDEX idx_comments_author ON comments(author_id);
```

### E-commerce Schema

```sql
-- Products
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL CHECK (price > 0),
    sku VARCHAR(100) UNIQUE,
    stock_quantity INTEGER DEFAULT 0 CHECK (stock_quantity >= 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Customers
CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Orders
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER REFERENCES customers(id),
    total_amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Order Items
CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(10,2) NOT NULL,
    total_price DECIMAL(10,2) NOT NULL
);
```
