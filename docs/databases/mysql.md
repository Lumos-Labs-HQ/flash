---
title: MySQL Guide
description: Using Flash ORM with MySQL databases
---

# MySQL Guide

Complete guide to using Flash ORM with MySQL databases, including MySQL-specific features, optimizations, and best practices.

## Table of Contents

- [Installation & Setup](#installation--setup)
- [MySQL-Specific Features](#mysql-specific-features)
- [Data Types](#data-types)
- [Storage Engines](#storage-engines)
- [Indexing Strategies](#indexing-strategies)
- [Query Optimization](#query-optimization)
- [Replication Support](#replication-support)
- [Partitioning](#partitioning)
- [Full-Text Search](#full-text-search)
- [JSON Support](#json-support)

## Installation & Setup

### MySQL Installation

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install mysql-server

# macOS with Homebrew
brew install mysql
brew services start mysql

# Docker
docker run --name mysql -e MYSQL_ROOT_PASSWORD=mypassword -p 3306:3306 -d mysql:8.0
```

### Connection Configuration

```bash
# Create database
mysql -u root -p -e "CREATE DATABASE myapp;"

# Set environment variable
export DATABASE_URL="mysql://user:password@localhost:3306/myapp?parseTime=true"
```

### Flash ORM Setup

```bash
# Initialize with MySQL
flash init --mysql

# Verify connection
flash status
```

## MySQL-Specific Features

### Connection Parameters

```env
# Basic connection
DATABASE_URL=mysql://user:pass@localhost:3306/dbname

# With SSL
DATABASE_URL=mysql://user:pass@localhost:3306/dbname?tls=skip-verify

# With timeout settings
DATABASE_URL=mysql://user:pass@localhost:3306/dbname?timeout=30s&readTimeout=30s&writeTimeout=30s

# With charset
DATABASE_URL=mysql://user:pass@localhost:3306/dbname?charset=utf8mb4&parseTime=true
```

### MySQL-Specific Config

```json
// flash.config.json
{
  "database": {
    "provider": "mysql",
    "url_env": "DATABASE_URL",
    "mysql": {
      "charset": "utf8mb4",
      "parse_time": true,
      "loc": "Local",
      "timeout": "30s",
      "read_timeout": "30s",
      "write_timeout": "30s"
    }
  }
}
```

## Data Types

### Numeric Types

```sql
CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,        -- Exact decimal
    discount FLOAT DEFAULT 0.0,          -- Single precision float
    weight DOUBLE,                       -- Double precision float
    quantity INT NOT NULL,               -- 32-bit integer
    stock BIGINT DEFAULT 0,              -- 64-bit integer
    rating TINYINT CHECK (rating >= 1 AND rating <= 5),  -- 8-bit integer
    is_available BOOLEAN DEFAULT TRUE    -- TINYINT(1) alias
);
```

### Text Types

```sql
CREATE TABLE content (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,         -- Up to 255 characters
    description TEXT,                    -- Up to 65,535 characters
    content MEDIUMTEXT,                  -- Up to 16,777,215 characters
    notes LONGTEXT,                      -- Up to 4,294,967,295 characters
    short_code CHAR(10)                  -- Fixed 10 characters
);
```

### Date/Time Types

```sql
CREATE TABLE events (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    event_date DATE NOT NULL,
    event_time TIME,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    event_datetime DATETIME
);
```

### Binary Types

```sql
CREATE TABLE files (
    id INT AUTO_INCREMENT PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    content BLOB,                        -- Up to 65,535 bytes
    large_content MEDIUMBLOB,            -- Up to 16,777,215 bytes
    huge_content LONGBLOB,               -- Up to 4,294,967,295 bytes
    hash VARCHAR(64) UNIQUE,             -- File hash
    size BIGINT                          -- File size in bytes
);
```

## Storage Engines

### InnoDB (Default)

```sql
-- Explicitly specify InnoDB
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE = InnoDB;

-- Foreign keys (InnoDB only)
CREATE TABLE posts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE = InnoDB;
```

### MyISAM

```sql
-- Full-text search capable
CREATE TABLE articles (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    FULLTEXT (title, content)
) ENGINE = MyISAM;
```

### Memory

```sql
-- In-memory tables for temporary data
CREATE TABLE cache (
    cache_key VARCHAR(255) PRIMARY KEY,
    cache_value TEXT,
    expires_at TIMESTAMP
) ENGINE = MEMORY;
```

## Indexing Strategies

### Index Types

```sql
-- B-tree indexes (default)
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_posts_user_created ON posts(user_id, created_at DESC);

-- Unique indexes
CREATE UNIQUE INDEX idx_users_username ON users(username);

-- Full-text indexes (MyISAM)
CREATE FULLTEXT INDEX idx_articles_content ON articles(title, content);

-- Spatial indexes (with spatial data types)
CREATE SPATIAL INDEX idx_locations_coords ON locations(coordinates);
```

### Composite Indexes

```sql
-- Index for common query patterns
CREATE INDEX idx_orders_status_date ON orders(status, created_at DESC);

-- Covering indexes
CREATE INDEX idx_users_active_covering ON users(is_active, created_at)
INCLUDE (id, name, email);

-- Index for range queries
CREATE INDEX idx_products_price_category ON products(price, category_id);
```

### Index Maintenance

```sql
-- Analyze index usage
SHOW INDEX FROM users;

-- Check index cardinality
SELECT
    TABLE_NAME,
    INDEX_NAME,
    CARDINALITY,
    PAGES,
    FILTER_CONDITION
FROM INFORMATION_SCHEMA.STATISTICS
WHERE TABLE_SCHEMA = DATABASE();

-- Rebuild indexes
ALTER TABLE users DROP INDEX idx_users_email;
ALTER TABLE users ADD INDEX idx_users_email (email);
```

## Query Optimization

### EXPLAIN Analysis

```sql
-- Analyze query execution
EXPLAIN SELECT * FROM posts
WHERE user_id = 123 AND published = true
ORDER BY created_at DESC LIMIT 10;

-- Format: TRADITIONAL, JSON, TREE (MySQL 8.0.16+)
EXPLAIN FORMAT=JSON SELECT * FROM posts
WHERE user_id = 123 AND published = true;
```

### Query Optimization Techniques

```sql
-- Use STRAIGHT_JOIN for specific join order
SELECT STRAIGHT_JOIN u.name, p.title
FROM users u
JOIN posts p ON u.id = p.user_id
WHERE u.is_active = true;

-- Force index usage
SELECT * FROM posts FORCE INDEX(idx_posts_user_created)
WHERE user_id = 123 AND created_at > '2024-01-01';

-- Query hints
SELECT /*+ MAX_EXECUTION_TIME(1000) */ * FROM large_table;
```

### Connection Optimization

```go
// MySQL-specific connection configuration
db, err := sql.Open("mysql", dsn)
if err != nil {
    log.Fatal(err)
}

// Set connection pool settings
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(25)
db.SetConnMaxLifetime(5 * time.Minute)

// MySQL-specific pragmas
_, err = db.Exec("SET sql_mode = 'STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE,ERROR_FOR_DIVISION_BY_ZERO'")
```

## Replication Support

### Read/Write Splitting

```go
// Configure read/write databases
type DBManager struct {
    writeDB *sql.DB  // Master
    readDBs []*sql.DB // Slaves
}

func (m *DBManager) GetReadDB() *sql.DB {
    // Round-robin or random selection
    return m.readDBs[rand.Intn(len(m.readDBs))]
}

func (m *DBManager) GetWriteDB() *sql.DB {
    return m.writeDB
}
```

### Replication-Aware Queries

```sql
-- Force read from master (for fresh data)
SELECT /*+ MAX_EXECUTION_TIME(100) */ balance
FROM accounts WHERE user_id = ?;

-- Allow reading from slaves (for older data)
SELECT created_at, title FROM posts
WHERE user_id = ? ORDER BY created_at DESC LIMIT 10;
```

## Partitioning

### Range Partitioning

```sql
-- Partition by date range
CREATE TABLE orders (
    id INT AUTO_INCREMENT,
    user_id INT NOT NULL,
    total DECIMAL(10,2) NOT NULL,
    status ENUM('pending', 'processing', 'shipped', 'delivered') DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (YEAR(created_at)) (
    PARTITION p2023 VALUES LESS THAN (2024),
    PARTITION p2024 VALUES LESS THAN (2025),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);
```

### Hash Partitioning

```sql
-- Partition by user_id for even distribution
CREATE TABLE user_sessions (
    id INT AUTO_INCREMENT,
    user_id INT NOT NULL,
    session_data JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    PRIMARY KEY (id, user_id)
) PARTITION BY HASH(user_id) PARTITIONS 4;
```

### List Partitioning

```sql
-- Partition by region
CREATE TABLE regional_sales (
    id INT AUTO_INCREMENT,
    region ENUM('north', 'south', 'east', 'west') NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    sale_date DATE NOT NULL,
    PRIMARY KEY (id, region)
) PARTITION BY LIST(region) (
    PARTITION p_north VALUES IN ('north'),
    PARTITION p_south VALUES IN ('south'),
    PARTITION p_east VALUES IN ('east'),
    PARTITION p_west VALUES IN ('west')
);
```

### Partitioning Queries

```sql
-- Query specific partitions
SELECT * FROM orders PARTITION (p2024)
WHERE created_at >= '2024-01-01';

-- Partition maintenance
ALTER TABLE orders ADD PARTITION (
    PARTITION p2025 VALUES LESS THAN (2026)
);

-- Remove old partitions
ALTER TABLE orders DROP PARTITION p2023;
```

## Full-Text Search

### Full-Text Indexes

```sql
-- Create table with full-text index
CREATE TABLE articles (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    tags VARCHAR(500),
    FULLTEXT (title, content, tags)
) ENGINE = InnoDB;

-- Or add index separately
CREATE FULLTEXT INDEX idx_articles_fulltext
ON articles(title, content, tags);
```

### Full-Text Search Queries

```sql
-- Basic full-text search
SELECT * FROM articles
WHERE MATCH(title, content, tags) AGAINST('database optimization' IN NATURAL LANGUAGE MODE);

-- Boolean mode search
SELECT * FROM articles
WHERE MATCH(title, content) AGAINST('+database -mysql' IN BOOLEAN MODE);

-- With relevance scoring
SELECT
    id, title,
    MATCH(title, content, tags) AGAINST('database' IN NATURAL LANGUAGE MODE) as relevance
FROM articles
WHERE MATCH(title, content, tags) AGAINST('database' IN NATURAL LANGUAGE MODE)
ORDER BY relevance DESC;
```

### Full-Text Search in Queries

```sql
-- db/queries/articles.sql

-- name: SearchArticles :many
SELECT
    id, title, content,
    MATCH(title, content, tags) AGAINST($1 IN NATURAL LANGUAGE MODE) as relevance
FROM articles
WHERE MATCH(title, content, tags) AGAINST($1 IN NATURAL LANGUAGE MODE)
ORDER BY relevance DESC
LIMIT $2 OFFSET $3;

-- name: SearchArticlesAdvanced :many
SELECT * FROM articles
WHERE MATCH(title, content) AGAINST($1 IN BOOLEAN MODE);

-- name: GetArticleSuggestions :many
SELECT title FROM articles
WHERE MATCH(title) AGAINST($1 IN NATURAL LANGUAGE MODE)
ORDER BY MATCH(title) AGAINST($1 IN NATURAL LANGUAGE MODE) DESC
LIMIT 5;
```

## JSON Support

### JSON Columns

```sql
-- MySQL 5.7.8+ supports JSON
CREATE TABLE user_preferences (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    preferences JSON,
    settings JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### JSON Functions

```sql
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

-- Query JSON data
SELECT
    user_id,
    JSON_EXTRACT(preferences, '$.theme') as theme,
    JSON_EXTRACT(preferences, '$.notifications.email') as email_notifications
FROM user_preferences;

-- Update JSON data
UPDATE user_preferences
SET preferences = JSON_SET(preferences, '$.theme', 'light')
WHERE user_id = 1;

-- Search in JSON arrays
SELECT * FROM user_preferences
WHERE JSON_CONTAINS(preferences, '"urgent"', '$.tags');
```

### JSON Queries

```sql
-- db/queries/user_preferences.sql

-- name: GetUserPreferences :one
SELECT * FROM user_preferences WHERE user_id = $1;

-- name: UpdateUserTheme :exec
UPDATE user_preferences
SET preferences = JSON_SET(preferences, '$.theme', $2)
WHERE user_id = $1;

-- name: GetUsersWithDarkTheme :many
SELECT u.* FROM users u
JOIN user_preferences up ON u.id = up.user_id
WHERE JSON_EXTRACT(up.preferences, '$.theme') = 'dark';

-- name: AddUserPreference :exec
UPDATE user_preferences
SET preferences = JSON_MERGE_PATCH(preferences, $2)
WHERE user_id = $1;
```

### JSON Indexes

```sql
-- Generated column for indexing
ALTER TABLE user_preferences
ADD COLUMN theme VARCHAR(50) GENERATED ALWAYS AS (JSON_EXTRACT(preferences, '$.theme')) STORED;

CREATE INDEX idx_user_prefs_theme ON user_preferences(theme);

-- Functional index (MySQL 8.0.13+)
CREATE INDEX idx_user_prefs_theme_func ON user_preferences(
    (JSON_EXTRACT(preferences, '$.theme'))
);
```

## MySQL-Specific Optimizations

### Query Cache (MySQL 5.7 and earlier)

```sql
-- Enable query cache
SET GLOBAL query_cache_size = 268435456;  -- 256MB
SET GLOBAL query_cache_type = ON;

-- Query cache hints
SELECT SQL_CACHE * FROM users WHERE id = 1;
SELECT SQL_NO_CACHE COUNT(*) FROM users;
```

### InnoDB Optimizations

```sql
-- InnoDB buffer pool size (80% of available RAM)
SET GLOBAL innodb_buffer_pool_size = 1073741824;  -- 1GB

-- InnoDB log file size
SET GLOBAL innodb_log_file_size = 268435456;  -- 256MB

-- Disable autocommit for better performance
SET autocommit = 0;
START TRANSACTION;
-- Your queries here
COMMIT;
SET autocommit = 1;
```

### Connection Optimizations

```sql
-- Connection-specific settings
SET SESSION sql_mode = 'STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE';
SET SESSION group_concat_max_len = 4096;
SET SESSION max_execution_time = 30000;  -- 30 seconds
```

## Best Practices

### Schema Design

```sql
-- Use appropriate character sets
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    name VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Use ENUM for constrained values
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    status ENUM('pending', 'processing', 'shipped', 'delivered', 'cancelled') DEFAULT 'pending',
    priority ENUM('low', 'normal', 'high', 'urgent') DEFAULT 'normal'
);
```

### Performance Monitoring

```sql
-- Show current connections
SHOW PROCESSLIST;

-- Show engine status
SHOW ENGINE INNODB STATUS;

-- Show table status
SHOW TABLE STATUS LIKE 'users';

-- Query performance
SHOW PROFILES;
SET profiling = 1;
-- Run your query
SHOW PROFILES;
```

### Backup and Recovery

```bash
# Logical backup
mysqldump -u root -p myapp > backup.sql

# Physical backup (InnoDB)
mysqlbackup --backup-dir=/backup/dir --user=root --password backup

# Point-in-time recovery
mysqlbinlog --start-datetime="2024-01-01 00:00:00" binlog.000001 | mysql -u root -p
```

MySQL's widespread adoption and rich feature set make it an excellent choice for many applications. Flash ORM provides comprehensive support for MySQL's unique features while maintaining the same clean, type-safe API across all supported languages.
