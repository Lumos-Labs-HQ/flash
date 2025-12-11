---
title: Schema Definition
description: How to define database schemas in Flash ORM
---

# Schema Definition

Flash ORM uses SQL files to define your database schema. This approach gives you full control over your database structure while providing powerful code generation capabilities.

## Table of Contents

- [Basic Schema](#basic-schema)
- [Data Types](#data-types)
- [Constraints](#constraints)
- [Indexes](#indexes)
- [Enums](#enums)
- [Relationships](#relationships)
- [Advanced Features](#advanced-features)
- [Best Practices](#best-practices)

## Basic Schema

### Single Schema File

```sql
-- db/schema/schema.sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Multiple Schema Files

Flash ORM supports splitting your schema across multiple files for better organization:

```
db/schema/
├── users.sql
├── posts.sql
├── comments.sql
└── categories.sql
```

```sql
-- db/schema/users.sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

```sql
-- db/schema/posts.sql
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Data Types

### PostgreSQL Data Types

| SQL Type | Go Type | TypeScript Type | Python Type | Description |
|----------|---------|-----------------|-------------|-------------|
| `SERIAL` | `int64` | `number` | `int` | Auto-incrementing integer |
| `BIGSERIAL` | `int64` | `number` | `int` | Large auto-incrementing integer |
| `INTEGER` | `int32` | `number` | `int` | 32-bit integer |
| `BIGINT` | `int64` | `number` | `int` | 64-bit integer |
| `VARCHAR(n)` | `string` | `string` | `str` | Variable-length string |
| `TEXT` | `string` | `string` | `str` | Unlimited text |
| `BOOLEAN` | `bool` | `boolean` | `bool` | True/false |
| `TIMESTAMP` | `time.Time` | `Date` | `datetime` | Date and time |
| `DATE` | `time.Time` | `Date` | `date` | Date only |
| `TIME` | `time.Time` | `string` | `time` | Time only |
| `JSONB` | `[]byte` | `any` | `dict` | JSON data |
| `UUID` | `string` | `string` | `str` | UUID string |
| `BYTEA` | `[]byte` | `Buffer` | `bytes` | Binary data |

### MySQL Data Types

| SQL Type | Go Type | TypeScript Type | Python Type | Description |
|----------|---------|-----------------|-------------|-------------|
| `AUTO_INCREMENT` | `int64` | `number` | `int` | Auto-incrementing integer |
| `INT` | `int32` | `number` | `int` | 32-bit integer |
| `BIGINT` | `int64` | `number` | `int` | 64-bit integer |
| `VARCHAR(n)` | `string` | `string` | `str` | Variable-length string |
| `TEXT` | `string` | `string` | `str` | Text data |
| `TINYINT(1)` | `bool` | `boolean` | `bool` | Boolean |
| `DATETIME` | `time.Time` | `Date` | `datetime` | Date and time |
| `DATE` | `time.Time` | `Date` | `date` | Date only |
| `JSON` | `[]byte` | `any` | `dict` | JSON data |
| `BLOB` | `[]byte` | `Buffer` | `bytes` | Binary data |

### SQLite Data Types

| SQL Type | Go Type | TypeScript Type | Python Type | Description |
|----------|---------|-----------------|-------------|-------------|
| `INTEGER` | `int64` | `number` | `int` | Integer (auto-incrementing with PRIMARY KEY) |
| `TEXT` | `string` | `string` | `str` | Text data |
| `REAL` | `float64` | `number` | `float` | Floating point |
| `BLOB` | `[]byte` | `Buffer` | `bytes` | Binary data |

## Constraints

### Primary Keys

```sql
-- Single column primary key
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL
);

-- Composite primary key
CREATE TABLE user_permissions (
    user_id INTEGER NOT NULL,
    permission_id INTEGER NOT NULL,
    granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, permission_id)
);
```

### Foreign Keys

```sql
-- Basic foreign key
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    title VARCHAR(255) NOT NULL
);

-- Foreign key with actions
CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    post_id INTEGER REFERENCES posts(id) ON DELETE CASCADE ON UPDATE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    content TEXT NOT NULL
);
```

### Unique Constraints

```sql
-- Single column unique
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL
);

-- Multiple column unique
CREATE TABLE user_follows (
    follower_id INTEGER NOT NULL,
    following_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (follower_id, following_id)
);
```

### Check Constraints

```sql
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) CHECK (price > 0),
    stock_quantity INTEGER CHECK (stock_quantity >= 0),
    discount_percent DECIMAL(5,2) CHECK (discount_percent BETWEEN 0 AND 100)
);
```

### Not Null Constraints

```sql
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    total_amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Indexes

### Basic Indexes

```sql
-- Single column index
CREATE INDEX idx_users_email ON users(email);

-- Multiple column index
CREATE INDEX idx_posts_user_created ON posts(user_id, created_at DESC);

-- Unique index
CREATE UNIQUE INDEX idx_users_username ON users(username);
```

### Partial Indexes

```sql
-- Index only active users
CREATE INDEX idx_active_users ON users(email) WHERE is_active = true;

-- Index recent posts
CREATE INDEX idx_recent_posts ON posts(created_at DESC) WHERE created_at > NOW() - INTERVAL '30 days';
```

### Index Types

```sql
-- B-tree index (default)
CREATE INDEX idx_users_name ON users(name);

-- Hash index (PostgreSQL only)
CREATE INDEX idx_users_email_hash ON users USING HASH (email);

-- GIN index for arrays (PostgreSQL)
CREATE INDEX idx_posts_tags ON posts USING GIN (tags);

-- GIN index for JSONB (PostgreSQL)
CREATE INDEX idx_user_prefs ON user_preferences USING GIN (preferences);
```

## Enums

### PostgreSQL Enums

```sql
-- Create enum type
CREATE TYPE user_role AS ENUM ('admin', 'moderator', 'user');
CREATE TYPE order_status AS ENUM ('pending', 'processing', 'shipped', 'delivered', 'cancelled');

-- Use in tables
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    role user_role DEFAULT 'user'
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    status order_status DEFAULT 'pending',
    total_amount DECIMAL(10,2) NOT NULL
);
```

### MySQL Enums

```sql
-- MySQL enum (stored as string)
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    role ENUM('admin', 'moderator', 'user') DEFAULT 'user'
);

CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT REFERENCES users(id),
    status ENUM('pending', 'processing', 'shipped', 'delivered', 'cancelled') DEFAULT 'pending',
    total_amount DECIMAL(10,2) NOT NULL
);
```

## Relationships

### One-to-Many

```sql
-- One user has many posts
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT
);
```

### Many-to-Many

```sql
-- Users can have many roles, roles can belong to many users
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL
);

CREATE TABLE user_roles (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);
```

### Self-Referencing

```sql
-- Employee hierarchy
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    manager_id INTEGER REFERENCES employees(id),
    department VARCHAR(50)
);
```

## Advanced Features

### Arrays (PostgreSQL)

```sql
CREATE TABLE articles (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    tags TEXT[],  -- Array of strings
    categories INTEGER[],  -- Array of integers
    metadata JSONB
);
```

### JSON/JSONB (PostgreSQL)

```sql
CREATE TABLE user_preferences (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    preferences JSONB,  -- Flexible JSON storage
    settings JSON,      -- Regular JSON
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- GIN index for JSON queries
CREATE INDEX idx_user_prefs ON user_preferences USING GIN (preferences);
```

### Generated Columns (PostgreSQL)

```sql
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    tax_rate DECIMAL(5,4) DEFAULT 0.08,
    -- Generated column
    price_with_tax DECIMAL(10,2) GENERATED ALWAYS AS (price * (1 + tax_rate)) STORED
);
```

### Triggers

```sql
-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### Views

```sql
-- Create a view
CREATE VIEW user_posts AS
SELECT
    u.id as user_id,
    u.name as user_name,
    u.email,
    p.id as post_id,
    p.title,
    p.content,
    p.published,
    p.created_at as post_created_at
FROM users u
LEFT JOIN posts p ON u.id = p.user_id;

-- Use the view in queries
-- name: GetUserWithPosts :many
SELECT * FROM user_posts WHERE user_id = $1;
```

## Best Practices

### Naming Conventions

```sql
-- Tables: lowercase, plural, snake_case
CREATE TABLE user_profiles (
    -- Primary keys: id
    id SERIAL PRIMARY KEY,

    -- Foreign keys: {table}_id
    user_id INTEGER REFERENCES users(id),

    -- Columns: snake_case
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    phone_number VARCHAR(20),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes: idx_{table}_{columns}
CREATE INDEX idx_user_profiles_user_id ON user_profiles(user_id);
CREATE INDEX idx_user_profiles_name ON user_profiles(first_name, last_name);
```

### Schema Organization

```
db/schema/
├── 01_users.sql           # Core entities first
├── 02_authentication.sql  # Authentication related
├── 03_content.sql         # Content management
├── 04_interactions.sql    # User interactions
├── 05_analytics.sql       # Analytics and reporting
└── 06_migrations.sql      # Migration helpers
```

### Performance Considerations

```sql
-- Use appropriate data types
CREATE TABLE logs (
    id BIGSERIAL PRIMARY KEY,
    level VARCHAR(10) NOT NULL,  -- Use constrained VARCHAR instead of TEXT
    message TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add indexes for frequently queried columns
CREATE INDEX idx_logs_level_created ON logs(level, created_at DESC);
CREATE INDEX idx_logs_created_at ON logs(created_at DESC);

-- Use partial indexes for common filters
CREATE INDEX idx_active_users ON users(created_at) WHERE is_active = true;
```

### Data Integrity

```sql
-- Use constraints to enforce business rules
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    total_amount DECIMAL(10,2) NOT NULL CHECK (total_amount > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'shipped', 'delivered', 'cancelled')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Prevent negative inventory
CREATE TABLE inventory (
    product_id INTEGER PRIMARY KEY REFERENCES products(id),
    quantity INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Documentation

```sql
-- Add comments to document your schema
COMMENT ON TABLE users IS 'Registered users of the application';
COMMENT ON COLUMN users.email IS 'Unique email address for authentication';
COMMENT ON COLUMN users.is_active IS 'Soft delete flag - false means deactivated';

-- Document constraints
ALTER TABLE users ADD CONSTRAINT chk_email_format
    CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');
COMMENT ON CONSTRAINT chk_email_format ON users IS 'Ensures email follows valid format';
```

### Migration Safety

```sql
-- Always provide rollback strategies
-- Up migration
ALTER TABLE users ADD COLUMN phone VARCHAR(20);

-- Down migration (in separate file)
ALTER TABLE users DROP COLUMN phone;

-- Use transactions for complex changes
BEGIN;
ALTER TABLE orders ADD COLUMN shipping_address TEXT;
UPDATE orders SET shipping_address = user_addresses.address
FROM user_addresses WHERE orders.user_id = user_addresses.user_id;
COMMIT;
```

Remember: Your schema files are the source of truth for your database structure. Keep them well-organized, properly constrained, and thoroughly tested before applying migrations to production environments.
