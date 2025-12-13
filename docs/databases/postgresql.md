---
title: PostgreSQL Guide
description: Using Flash ORM with PostgreSQL databases
---

# PostgreSQL Guide

Complete guide to using Flash ORM with PostgreSQL, including advanced features like JSONB, arrays, enums, and PostgreSQL-specific optimizations.

## Table of Contents

- [Installation & Setup](#installation--setup)
- [PostgreSQL-Specific Features](#postgresql-specific-features)
- [Data Types](#data-types)
- [Advanced Schema Features](#advanced-schema-features)
- [Performance Optimization](#performance-optimization)
- [JSONB Support](#jsonb-support)
- [Array Support](#array-support)
- [Enum Support](#enum-support)
- [Full-Text Search](#full-text-search)
- [Partitioning](#partitioning)
- [Extensions](#extensions)

## Installation & Setup

### PostgreSQL Installation

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install postgresql postgresql-contrib

# macOS with Homebrew
brew install postgresql
brew services start postgresql

# Docker
docker run --name postgres -e POSTGRES_PASSWORD=mypassword -p 5432:5432 -d postgres:13
```

### Connection Configuration

```bash
# Create database
createdb myapp

# Set environment variable
export DATABASE_URL="postgres://username:password@localhost:5432/myapp?sslmode=disable"
```

### Flash ORM Setup

```bash
# Initialize with PostgreSQL
flash init --postgresql

# Verify connection
flash status
```

## PostgreSQL-Specific Features

### Connection Parameters

```env
# Basic connection
DATABASE_URL=postgres://user:pass@localhost:5432/dbname

# With SSL
DATABASE_URL=postgres://user:pass@localhost:5432/dbname?sslmode=require

# With connection pool settings
DATABASE_URL=postgres://user:pass@localhost:5432/dbname?pool_max_conns=10&pool_min_conns=2

# Unix socket
DATABASE_URL=postgres://user:pass@/dbname?host=/tmp
```

### PostgreSQL-Specific Config

```json
// flash.config.json
{
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL",
    "pg": {
      "ssl_mode": "require",
      "search_path": "public,extensions",
      "timezone": "UTC",
      "application_name": "flash-orm"
    }
  }
}
```

## Data Types

### Numeric Types

```sql
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,        -- Exact decimal
    discount REAL DEFAULT 0.0,           -- Single precision float
    weight DOUBLE PRECISION,             -- Double precision float
    quantity INTEGER NOT NULL,           -- 32-bit integer
    stock BIGINT DEFAULT 0,              -- 64-bit integer
    rating SMALLINT CHECK (rating >= 1 AND rating <= 5)  -- 16-bit integer
);
```

### Text Types

```sql
CREATE TABLE content (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,         -- Limited text
    description TEXT,                    -- Unlimited text
    short_code CHAR(10),                 -- Fixed length
    tags VARCHAR(100)[]                  -- Array of strings
);
```

### Date/Time Types

```sql
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    start_date DATE NOT NULL,
    start_time TIME,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,  -- With timezone
    event_duration INTERVAL
);
```

### Binary Types

```sql
CREATE TABLE files (
    id SERIAL PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    content BYTEA,                       -- Binary data
    hash VARCHAR(64) UNIQUE,             -- File hash
    size BIGINT                          -- File size in bytes
);
```

## Advanced Schema Features

### Generated Columns

```sql
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    subtotal DECIMAL(10,2) NOT NULL,
    tax_rate DECIMAL(5,4) DEFAULT 0.08,
    tax_amount DECIMAL(10,2) GENERATED ALWAYS AS (subtotal * tax_rate) STORED,
    total DECIMAL(10,2) GENERATED ALWAYS AS (subtotal + tax_amount) STORED
);
```

### Check Constraints

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    age INTEGER CHECK (age >= 13 AND age <= 120),
    status VARCHAR(20) DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'suspended')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_email CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);
```

### Partial Indexes

```sql
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT FALSE,
    published_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index only published posts
CREATE INDEX idx_published_posts ON posts(published_at) WHERE published = true;

-- Index recent posts
CREATE INDEX idx_recent_posts ON posts(created_at DESC)
WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '30 days';
```

### Functional Indexes

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    email VARCHAR(255) UNIQUE
);

-- Index on lowercase email for case-insensitive search
CREATE INDEX idx_users_email_lower ON users(LOWER(email));

-- Index on full name
CREATE INDEX idx_users_full_name ON users((first_name || ' ' || last_name));
```

## Performance Optimization

### Indexing Strategies

```sql
-- Composite indexes for common query patterns
CREATE INDEX idx_posts_user_published ON posts(user_id, published, created_at DESC);

-- Covering indexes
CREATE INDEX idx_users_active_covering ON users(is_active, created_at)
INCLUDE (id, name, email);

-- Partial indexes for filtered queries
CREATE INDEX idx_active_users ON users(created_at) WHERE is_active = true;

-- Expression indexes
CREATE INDEX idx_posts_title_search ON posts USING GIN (to_tsvector('english', title));
```

### Query Optimization

```sql
-- Use EXPLAIN to analyze queries
EXPLAIN ANALYZE
SELECT * FROM posts
WHERE user_id = 123 AND published = true
ORDER BY created_at DESC LIMIT 10;

-- Use appropriate join types
EXPLAIN ANALYZE
SELECT u.name, COUNT(p.id) as post_count
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
WHERE u.is_active = true
GROUP BY u.id, u.name;
```

### Connection Pooling

```go
// Optimized connection configuration
config := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
config.MaxConns = 10
config.MinConns = 2
config.MaxConnLifetime = 15 * time.Minute
config.MaxConnIdleTime = 3 * time.Minute
config.HealthCheckPeriod = 1 * time.Minute

pool, err := pgxpool.NewWithConfig(context.Background(), config)
```

## JSONB Support

### JSONB Columns

```sql
CREATE TABLE user_preferences (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    preferences JSONB DEFAULT '{}',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert JSON data
INSERT INTO user_preferences (user_id, preferences)
VALUES (1, '{
    "theme": "dark",
    "notifications": {
        "email": true,
        "push": false
    },
    "language": "en"
}');
```

### JSONB Queries

```sql
-- Query files for JSONB queries
-- db/queries/user_preferences.sql

-- name: GetUserPreferences :one
SELECT * FROM user_preferences WHERE user_id = $1;

-- name: UpdateUserTheme :exec
UPDATE user_preferences
SET preferences = preferences || '{"theme": $2}'::jsonb
WHERE user_id = $1;

-- name: GetUsersWithDarkTheme :many
SELECT u.* FROM users u
JOIN user_preferences up ON u.id = up.user_id
WHERE up.preferences->>'theme' = 'dark';

-- name: GetUsersWithEmailNotifications :many
SELECT u.* FROM users u
JOIN user_preferences up ON u.id = up.user_id
WHERE up.preferences->'notifications'->>'email' = 'true';

-- name: AddUserPreference :exec
UPDATE user_preferences
SET preferences = preferences || $2::jsonb
WHERE user_id = $1;

-- name: RemoveUserPreference :exec
UPDATE user_preferences
SET preferences = preferences - $2
WHERE user_id = $1;
```

### JSONB Indexes

```sql
-- GIN index for JSONB queries
CREATE INDEX idx_user_prefs_gin ON user_preferences USING GIN (preferences);

-- Specific path index
CREATE INDEX idx_user_prefs_theme ON user_preferences ((preferences->>'theme'));

-- Composite JSONB index
CREATE INDEX idx_user_prefs_theme_lang ON user_preferences (
    (preferences->>'theme'),
    (preferences->>'language')
);
```

### Advanced JSONB Operations

```sql
-- JSONB containment
SELECT * FROM user_preferences
WHERE preferences @> '{"theme": "dark"}';

-- JSONB existence
SELECT * FROM user_preferences
WHERE preferences ? 'notifications';

-- JSONB array operations
UPDATE user_preferences
SET preferences = preferences || '{"tags": ["urgent", "important"]}'::jsonb
WHERE user_id = 1;

-- Query array elements
SELECT * FROM user_preferences
WHERE preferences->'tags' ? 'urgent';
```

## Array Support

### Array Columns

```sql
CREATE TABLE articles (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    tags TEXT[],                      -- Array of strings
    categories INTEGER[],             -- Array of integers
    keywords TEXT[],                  -- Search keywords
    published BOOLEAN DEFAULT FALSE
);

-- Insert with arrays
INSERT INTO articles (title, tags, categories, keywords)
VALUES (
    'PostgreSQL Arrays Guide',
    ARRAY['postgresql', 'arrays', 'guide'],
    ARRAY[1, 3, 5],
    ARRAY['database', 'sql', 'tutorial']
);
```

### Array Queries

```sql
-- db/queries/articles.sql

-- name: GetArticlesByTag :many
SELECT * FROM articles
WHERE $1 = ANY(tags);

-- name: GetArticlesByTags :many
SELECT * FROM articles
WHERE tags && $1::text[];  -- Overlap operator

-- name: AddArticleTag :exec
UPDATE articles
SET tags = array_append(tags, $2)
WHERE id = $1;

-- name: RemoveArticleTag :exec
UPDATE articles
SET tags = array_remove(tags, $2)
WHERE id = $1;

-- name: GetArticlesWithTagCount :many
SELECT
    id, title, tags,
    array_length(tags, 1) as tag_count
FROM articles
WHERE published = true;

-- name: SearchArticlesByKeywords :many
SELECT * FROM articles
WHERE keywords @> $1::text[];  -- Contains operator
```

### Array Indexes

```sql
-- GIN index for array operations
CREATE INDEX idx_articles_tags_gin ON articles USING GIN (tags);
CREATE INDEX idx_articles_keywords_gin ON articles USING GIN (keywords);

-- B-tree index for exact matches
CREATE INDEX idx_articles_categories ON articles USING GIN (categories);
```

### Array Functions

```sql
-- Array operations in queries
SELECT
    id,
    title,
    array_length(tags, 1) as tag_count,
    tags[1:3] as first_three_tags,  -- Array slice
    'database' = ANY(tags) as has_database_tag
FROM articles;

-- Unnest arrays for aggregation
SELECT
    unnest(tags) as tag,
    COUNT(*) as count
FROM articles
GROUP BY tag
ORDER BY count DESC;
```

## Enum Support

### Creating Enums

```sql
-- Create enum types
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended', 'banned');
CREATE TYPE order_status AS ENUM ('pending', 'processing', 'shipped', 'delivered', 'cancelled');
CREATE TYPE priority AS ENUM ('low', 'medium', 'high', 'urgent');

-- Use in tables
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    status user_status DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    status order_status DEFAULT 'pending',
    priority priority DEFAULT 'medium',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Enum Queries

```sql
-- db/queries/users.sql

-- name: GetUsersByStatus :many
SELECT * FROM users WHERE status = $1::user_status;

-- name: UpdateUserStatus :exec
UPDATE users SET status = $2::user_status WHERE id = $1;

-- name: GetUserStatusStats :one
SELECT
    COUNT(*) FILTER (WHERE status = 'active') as active_count,
    COUNT(*) FILTER (WHERE status = 'inactive') as inactive_count,
    COUNT(*) FILTER (WHERE status = 'suspended') as suspended_count
FROM users;

-- db/queries/orders.sql

-- name: GetOrdersByStatus :many
SELECT * FROM orders WHERE status = $1::order_status;

-- name: GetOrdersByPriority :many
SELECT * FROM orders WHERE priority = $1::priority;

-- name: UpdateOrderStatus :exec
UPDATE orders SET status = $2::order_status WHERE id = $1;
```

### Enum Indexes

```sql
-- Index enum columns
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_priority ON orders(priority);

-- Composite indexes with enums
CREATE INDEX idx_orders_status_priority ON orders(status, priority, created_at DESC);
```

### Enum Operations

```sql
-- Enum ordering (uses creation order)
SELECT * FROM users
ORDER BY status;  -- active, inactive, suspended, banned

-- Enum comparison
SELECT * FROM orders
WHERE status > 'pending'::order_status;  -- processing, shipped, delivered, cancelled

-- Enum in array
SELECT * FROM orders
WHERE status = ANY($1::order_status[]);
```

## Full-Text Search

### TSVECTOR and TSQUERY

```sql
-- Add full-text search columns
CREATE TABLE articles (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    search_vector TSVECTOR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create search vector trigger
CREATE OR REPLACE FUNCTION articles_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.content, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER articles_search_vector_trigger
    BEFORE INSERT OR UPDATE ON articles
    FOR EACH ROW EXECUTE FUNCTION articles_search_vector_update();
```

### Full-Text Search Queries

```sql
-- db/queries/articles.sql

-- name: SearchArticles :many
SELECT
    id, title, content, created_at,
    ts_rank(search_vector, query) as rank
FROM articles, to_tsquery('english', $1) query
WHERE search_vector @@ query
ORDER BY rank DESC
LIMIT $2 OFFSET $3;

-- name: SearchArticlesWithHighlights :many
SELECT
    id, title,
    ts_headline('english', content, query, 'StartSel=<mark>, StopSel=</mark>') as highlighted_content,
    ts_rank(search_vector, query) as rank
FROM articles, to_tsquery('english', $1) query
WHERE search_vector @@ query
ORDER BY rank DESC;

-- name: GetArticleSearchSuggestions :many
SELECT word FROM ts_stat($$
    SELECT search_vector FROM articles
$$) ORDER BY nentry DESC, ndoc DESC LIMIT 10;
```

### Full-Text Indexes

```sql
-- GIN index for full-text search
CREATE INDEX idx_articles_search_gin ON articles USING GIN (search_vector);

-- Partial index for published articles only
CREATE INDEX idx_published_articles_search ON articles USING GIN (search_vector)
WHERE published = true;
```

## Partitioning

### Range Partitioning

```sql
-- Create partitioned table
CREATE TABLE orders (
    id SERIAL,
    user_id INTEGER NOT NULL,
    total DECIMAL(10,2) NOT NULL,
    status order_status DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create partitions
CREATE TABLE orders_2024_01 PARTITION OF orders
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE orders_2024_02 PARTITION OF orders
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Default partition for older data
CREATE TABLE orders_default PARTITION OF orders DEFAULT;
```

### Hash Partitioning

```sql
-- Hash partitioning for even distribution
CREATE TABLE user_sessions (
    id SERIAL,
    user_id INTEGER NOT NULL,
    session_data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    PRIMARY KEY (id, user_id)
) PARTITION BY HASH (user_id);

-- Create 4 partitions
CREATE TABLE user_sessions_0 PARTITION OF user_sessions
    FOR VALUES WITH (MODULUS 4, REMAINDER 0);

CREATE TABLE user_sessions_1 PARTITION OF user_sessions
    FOR VALUES WITH (MODULUS 4, REMAINDER 1);

CREATE TABLE user_sessions_2 PARTITION OF user_sessions
    FOR VALUES WITH (MODULUS 4, REMAINDER 2);

CREATE TABLE user_sessions_3 PARTITION OF user_sessions
    FOR VALUES WITH (MODULUS 4, REMAINDER 3);
```

### Partitioning Queries

```sql
-- db/queries/orders.sql

-- name: GetOrdersByDateRange :many
SELECT * FROM orders
WHERE created_at >= $1 AND created_at < $2
ORDER BY created_at DESC;

-- name: GetMonthlyOrderStats :many
SELECT
    DATE_TRUNC('month', created_at) as month,
    COUNT(*) as order_count,
    SUM(total) as total_amount
FROM orders
WHERE created_at >= $1 AND created_at < $2
GROUP BY month
ORDER BY month;

-- Partition maintenance
-- name: CreateNextMonthPartition :exec
-- This would be a raw query for partition management
```

## Extensions

### Useful PostgreSQL Extensions

```sql
-- Install extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "postgis";

-- UUID generation
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Cryptographic functions
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    salt VARCHAR(32) DEFAULT encode(gen_random_bytes(16), 'hex')
);

-- PostGIS for geospatial data
CREATE TABLE locations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    coordinates GEOGRAPHY(POINT, 4326),
    address TEXT
);
```

### Extension Queries

```sql
-- db/queries/documents.sql

-- name: CreateDocument :one
INSERT INTO documents (title, content) VALUES ($1, $2) RETURNING *;

-- name: GetDocumentsNearby :many
SELECT id, name, address,
       ST_Distance(coordinates, ST_MakePoint($1, $2)::geography) as distance
FROM locations
WHERE ST_DWithin(coordinates, ST_MakePoint($1, $2)::geography, $3)
ORDER BY distance;
```

PostgreSQL's advanced features make it an excellent choice for complex applications. Flash ORM fully supports these features, allowing you to leverage PostgreSQL's full power while maintaining clean, type-safe code generation across all your supported languages.
