---
title: SQLite Guide
description: Using Flash ORM with SQLite databases
---

# SQLite Guide

Complete guide to using Flash ORM with SQLite databases, including SQLite-specific features, optimizations, and embedded database best practices.

## Table of Contents

- [Installation & Setup](#installation--setup)
- [SQLite-Specific Features](#sqlite-specific-features)
- [Data Types](#data-types)
- [SQLite Pragmas](#sqlite-pragmas)
- [Performance Optimization](#performance-optimization)
- [Concurrent Access](#concurrent-access)
- [Backup & Restore](#backup--restore)
- [Full-Text Search](#full-text-search)
- [JSON Support](#json-support)
- [Virtual Tables](#virtual-tables)
- [Embedded Applications](#embedded-applications)

## Installation & Setup

### SQLite Installation

SQLite comes pre-installed on most systems:

```bash
# Check if SQLite is installed
sqlite3 --version

# Ubuntu/Debian (if not installed)
sudo apt install sqlite3

# macOS (usually pre-installed)
# Windows - download from sqlite.org
```

### Connection Configuration

```bash
# File-based database
export DATABASE_URL="sqlite://./data.db"

# In-memory database (temporary)
export DATABASE_URL="sqlite://:memory:"

# With query parameters
export DATABASE_URL="sqlite://./data.db?_journal_mode=WAL&_synchronous=NORMAL"
```

### Flash ORM Setup

```bash
# Initialize with SQLite
flash init --sqlite

# Verify connection
flash status
```

## SQLite-Specific Features

### Connection Parameters

```env
# File path
DATABASE_URL=sqlite://./data.db

# Relative path
DATABASE_URL=sqlite://../data/app.db

# Absolute path
DATABASE_URL=sqlite:///home/user/app/data.db

# In-memory (ephemeral)
DATABASE_URL=sqlite://:memory:

# With pragmas
DATABASE_URL=sqlite://./data.db?_journal_mode=WAL&_foreign_keys=on&_synchronous=NORMAL
```

### SQLite-Specific Config

```json
// flash.config.json
{
  "database": {
    "provider": "sqlite",
    "url_env": "DATABASE_URL",
    "sqlite": {
      "journal_mode": "WAL",
      "synchronous": "NORMAL",
      "foreign_keys": true,
      "cache_size": -64000,
      "temp_store": "memory"
    }
  }
}
```

## Data Types

### SQLite Type Affinity

SQLite uses dynamic typing with type affinity:

```sql
CREATE TABLE flexible_data (
    id INTEGER PRIMARY KEY,
    name TEXT,                    -- String data
    count INTEGER DEFAULT 0,      -- Integer data
    price REAL,                   -- Floating point
    data BLOB,                    -- Binary data
    metadata TEXT,                -- JSON as text
    created_at TEXT DEFAULT CURRENT_TIMESTAMP  -- ISO 8601 timestamp
);
```

### Type Mapping

| SQLite Type | Go Type | TypeScript Type | Python Type | Description |
|-------------|---------|-----------------|-------------|-------------|
| `INTEGER` | `int64` | `number` | `int` | Signed 64-bit integer |
| `REAL` | `float64` | `number` | `float` | 64-bit floating point |
| `TEXT` | `string` | `string` | `str` | UTF-8 encoded text |
| `BLOB` | `[]byte` | `Buffer` | `bytes` | Binary data |

### Date/Time Handling

```sql
CREATE TABLE events (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    event_date TEXT,              -- ISO 8601: '2024-01-15'
    event_time TEXT,              -- ISO 8601: '14:30:00'
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);

-- Insert with datetime functions
INSERT INTO events (title, event_date, event_time)
VALUES ('Meeting', date('now'), time('now'));
```

## SQLite Pragmas

### Performance Pragmas

```sql
-- Journal mode for better concurrency
PRAGMA journal_mode = WAL;

-- Synchronous mode (balance between safety and speed)
PRAGMA synchronous = NORMAL;

-- Cache size (negative = KB, positive = pages)
PRAGMA cache_size = -64000;  -- 64MB

-- Temporary storage
PRAGMA temp_store = memory;

-- Memory-mapped I/O
PRAGMA mmap_size = 268435456;  -- 256MB
```

### Schema Pragmas

```sql
-- Foreign key enforcement
PRAGMA foreign_keys = ON;

-- Case-sensitive LIKE
PRAGMA case_sensitive_like = ON;

-- Recursive triggers
PRAGMA recursive_triggers = ON;
```

### Connection Setup

```go
// SQLite-specific connection setup
db, err := sql.Open("sqlite3", "./data.db")
if err != nil {
    log.Fatal(err)
}

// Set pragmas
pragmas := []string{
    "PRAGMA journal_mode = WAL;",
    "PRAGMA synchronous = NORMAL;",
    "PRAGMA foreign_keys = ON;",
    "PRAGMA cache_size = -64000;",
}

for _, pragma := range pragmas {
    if _, err := db.Exec(pragma); err != nil {
        log.Printf("Failed to set pragma: %v", err)
    }
}
```

## Performance Optimization

### WAL Mode

```sql
-- Enable Write-Ahead Logging for better concurrency
PRAGMA journal_mode = WAL;

-- Check WAL mode
PRAGMA journal_mode;

-- WAL file management
PRAGMA wal_checkpoint(TRUNCATE);
```

### Indexing Strategies

```sql
-- Single column indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);

-- Composite indexes
CREATE INDEX idx_posts_user_created ON posts(user_id, created_at DESC);

-- Partial indexes
CREATE INDEX idx_active_users ON users(created_at) WHERE is_active = 1;

-- Expression indexes
CREATE INDEX idx_users_name_lower ON users(lower(name));
```

### Query Optimization

```sql
-- Analyze query performance
EXPLAIN QUERY PLAN
SELECT * FROM posts
WHERE user_id = ? AND published = 1
ORDER BY created_at DESC LIMIT 10;

-- Use appropriate indexes
CREATE INDEX idx_posts_user_published_created
ON posts(user_id, published, created_at DESC);
```

### Connection Pooling

SQLite has limitations with concurrent writes:

```go
// Single connection for SQLite (no pooling needed)
db, err := sql.Open("sqlite3", "./data.db")
if err != nil {
    log.Fatal(err)
}

// Set connection limits for SQLite
db.SetMaxOpenConns(1)  // SQLite doesn't support concurrent writes
db.SetMaxIdleConns(1)
db.SetConnMaxLifetime(0)
```

## Concurrent Access

### SQLite Concurrency Model

```sql
-- Check if database is locked
PRAGMA lock_status;

-- Busy timeout (milliseconds)
PRAGMA busy_timeout = 5000;

-- Handle busy errors in application code
for retries := 0; retries < 3; retries++ {
    err := db.Exec(query, args...)
    if err == nil {
        break
    }
    if strings.Contains(err.Error(), "database is locked") {
        time.Sleep(time.Duration(retries+1) * 100 * time.Millisecond)
        continue
    }
    return err
}
```

### Read-Only Connections

```go
// Open read-only connection
db, err := sql.Open("sqlite3", "./data.db?mode=ro")

// Or use immutable mode for CDNs
db, err := sql.Open("sqlite3", "./data.db?immutable=1")
```

## Backup & Restore

### Online Backup

```sql
-- Backup to another database
ATTACH DATABASE 'backup.db' AS backup;
INSERT INTO backup.users SELECT * FROM main.users;
INSERT INTO backup.posts SELECT * FROM main.posts;
DETACH DATABASE backup;
```

### Hot Backup with WAL

```sql
-- Checkpoint WAL before backup
PRAGMA wal_checkpoint(TRUNCATE);

-- Copy files
-- data.db
-- data.db-wal (if exists)
-- data.db-shm (if exists)
```

### Backup API

```go
// Backup database
func backupDatabase(src, dst string) error {
    srcDB, err := sql.Open("sqlite3", src)
    if err != nil {
        return err
    }
    defer srcDB.Close()

    dstDB, err := sql.Open("sqlite3", dst)
    if err != nil {
        return err
    }
    defer dstDB.Close()

    // Attach destination
    _, err = srcDB.Exec(fmt.Sprintf("ATTACH DATABASE '%s' AS backup", dst))
    if err != nil {
        return err
    }

    // Copy all tables
    rows, err := srcDB.Query("SELECT name FROM sqlite_master WHERE type='table'")
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var tableName string
        rows.Scan(&tableName)
        _, err = srcDB.Exec(fmt.Sprintf("INSERT INTO backup.%s SELECT * FROM main.%s", tableName, tableName))
        if err != nil {
            return err
        }
    }

    return srcDB.Exec("DETACH DATABASE backup")
}
```

## Full-Text Search

### FTS5 Virtual Tables

```sql
-- Create FTS5 virtual table
CREATE VIRTUAL TABLE articles_fts USING fts5(
    title, content, tags,
    content=articles,
    content_rowid=id
);

-- Populate FTS table
INSERT INTO articles_fts(rowid, title, content, tags)
SELECT id, title, content, tags FROM articles;

-- Search
SELECT rowid, * FROM articles_fts
WHERE articles_fts MATCH 'database OR sqlite'
ORDER BY rank;

-- With snippet highlighting
SELECT
    rowid,
    snippet(articles_fts, 1, '<mark>', '</mark>', '...', 50) as highlighted_title,
    snippet(articles_fts, 2, '<mark>', '</mark>', '...', 100) as highlighted_content
FROM articles_fts
WHERE articles_fts MATCH 'database';
```

### FTS Queries

```sql
-- db/queries/articles.sql

-- name: SearchArticles :many
SELECT a.*, fts.rank
FROM articles a
JOIN articles_fts fts ON a.id = fts.rowid
WHERE articles_fts MATCH $1
ORDER BY fts.rank DESC
LIMIT $2 OFFSET $3;

-- name: SearchArticlesWithHighlights :many
SELECT
    a.id, a.title,
    snippet(articles_fts, 1, '<b>', '</b>', '...', 50) as highlighted_title,
    snippet(articles_fts, 2, '<b>', '</b>', '...', 100) as highlighted_content,
    fts.rank
FROM articles a
JOIN articles_fts fts ON a.id = fts.rowid
WHERE articles_fts MATCH $1
ORDER BY fts.rank DESC;

-- name: UpdateFTS :exec
INSERT INTO articles_fts(rowid, title, content, tags)
VALUES ($1, $2, $3, $4)
ON CONFLICT(rowid) DO UPDATE SET
    title = excluded.title,
    content = excluded.content,
    tags = excluded.tags;
```

### FTS Maintenance

```sql
-- Optimize FTS index
INSERT INTO articles_fts(articles_fts) VALUES('optimize');

-- Rebuild FTS index
INSERT INTO articles_fts(articles_fts) VALUES('rebuild');

-- Check FTS status
SELECT * FROM articles_fts('integrity-check');
```

## JSON Support

### JSON Functions

SQLite has built-in JSON support (3.38.0+):

```sql
-- JSON columns
CREATE TABLE user_preferences (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    preferences TEXT,  -- JSON as text
    settings TEXT,     -- JSON as text
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

-- Insert JSON
INSERT INTO user_preferences (user_id, preferences)
VALUES (1, '{
    "theme": "dark",
    "notifications": {"email": true, "push": false},
    "language": "en"
}');

-- Query JSON
SELECT
    user_id,
    json_extract(preferences, '$.theme') as theme,
    json_extract(preferences, '$.notifications.email') as email_notifs
FROM user_preferences;

-- Update JSON
UPDATE user_preferences
SET preferences = json_set(preferences, '$.theme', 'light')
WHERE user_id = 1;
```

### JSON Queries

```sql
-- db/queries/user_preferences.sql

-- name: GetUserPreferences :one
SELECT * FROM user_preferences WHERE user_id = $1;

-- name: UpdateUserTheme :exec
UPDATE user_preferences
SET preferences = json_set(preferences, '$.theme', $2)
WHERE user_id = $1;

-- name: GetUsersWithDarkTheme :many
SELECT u.* FROM users u
JOIN user_preferences up ON u.id = up.user_id
WHERE json_extract(up.preferences, '$.theme') = 'dark';

-- name: AddUserPreference :exec
UPDATE user_preferences
SET preferences = json_patch(preferences, $2)
WHERE user_id = $1;
```

## Virtual Tables

### Custom Virtual Tables

```sql
-- Create a simple virtual table module
-- (Requires custom SQLite extension)

-- Example: CSV virtual table
CREATE VIRTUAL TABLE temp.csv_data USING csv(
    filename='data.csv',
    header=yes
);

SELECT * FROM csv_data WHERE column1 = 'value';
```

### R-Tree for Spatial Data

```sql
-- R-Tree for spatial indexing
CREATE VIRTUAL TABLE locations USING rtree(
    id,              -- Integer primary key
    minX, maxX,      -- X coordinate bounds
    minY, maxY       -- Y coordinate bounds
);

-- Insert spatial data
INSERT INTO locations VALUES (1, -122.0, -121.0, 37.0, 38.0);

-- Spatial queries
SELECT id FROM locations
WHERE minX >= -123.0 AND maxX <= -121.0
  AND minY >= 36.0 AND maxY <= 39.0;
```

## Embedded Applications

### Desktop Applications

```go
// SQLite for desktop app data
func initDatabase() (*sql.DB, error) {
    // Create data directory
    dataDir := filepath.Join(os.Getenv("APPDATA"), "MyApp")
    os.MkdirAll(dataDir, 0755)

    dbPath := filepath.Join(dataDir, "app.db")
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }

    // Optimize for single-user desktop app
    pragmas := []string{
        "PRAGMA journal_mode = WAL;",
        "PRAGMA synchronous = NORMAL;",
        "PRAGMA cache_size = -32000;",  // 32MB
        "PRAGMA foreign_keys = ON;",
        "PRAGMA busy_timeout = 30000;", // 30 seconds
    }

    for _, pragma := range pragmas {
        if _, err := db.Exec(pragma); err != nil {
            return nil, err
        }
    }

    return db, nil
}
```

### Mobile Applications

```python
# SQLite for mobile app
import sqlite3
import os

def init_mobile_db():
    # iOS/Android data directory
    if platform.system() == 'iOS':
        db_path = os.path.join(os.path.expanduser('~'), 'Documents', 'app.db')
    else:  # Android
        db_path = os.path.join(os.path.expanduser('~'), 'app.db')

    conn = sqlite3.connect(db_path)

    # Mobile optimizations
    conn.execute("PRAGMA journal_mode = WAL")
    conn.execute("PRAGMA synchronous = NORMAL")
    conn.execute("PRAGMA cache_size = -2000")  # 2MB for mobile
    conn.execute("PRAGMA temp_store = memory")
    conn.execute("PRAGMA mmap_size = 268435456")  # 256MB

    return conn
```

### IoT Devices

```sql
-- SQLite for IoT sensor data
CREATE TABLE sensor_readings (
    id INTEGER PRIMARY KEY,
    sensor_id TEXT NOT NULL,
    reading_type TEXT NOT NULL,
    value REAL NOT NULL,
    unit TEXT,
    timestamp TEXT DEFAULT CURRENT_TIMESTAMP,
    quality INTEGER CHECK (quality BETWEEN 0 AND 100)
);

-- Efficient storage for time-series data
CREATE INDEX idx_sensor_time ON sensor_readings(sensor_id, timestamp DESC);
CREATE INDEX idx_reading_type_time ON sensor_readings(reading_type, timestamp DESC);
```

## Best Practices

### Schema Design

```sql
-- Use appropriate types
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    is_active INTEGER DEFAULT 1,  -- Use INTEGER for booleans
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);

-- Foreign keys
CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    content TEXT,
    published INTEGER DEFAULT 0,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
);
```

### Performance Tips

```sql
-- Analyze database
ANALYZE;

-- Vacuum database regularly
VACUUM;

-- Reindex if needed
REINDEX;

-- Monitor database size
PRAGMA page_count;
PRAGMA page_size;
```

### Backup Strategy

```bash
# Simple file copy backup
cp data.db data.db.backup

# Or use SQLite backup API
sqlite3 data.db ".backup backup.db"

# Automated backup script
#!/bin/bash
BACKUP_DIR="./backups"
DATE=$(date +%Y%m%d_%H%M%S)
sqlite3 data.db ".backup $BACKUP_DIR/backup_$DATE.db"
find $BACKUP_DIR -name "backup_*.db" -mtime +30 -delete
```

### Error Handling

```go
// Handle SQLite-specific errors
func handleSQLiteError(err error) {
    if err == nil {
        return
    }

    if strings.Contains(err.Error(), "database is locked") {
        log.Println("Database is busy, retrying...")
        time.Sleep(time.Second)
        // Retry logic
    } else if strings.Contains(err.Error(), "no such table") {
        log.Println("Table doesn't exist, running migrations...")
        // Run migrations
    } else {
        log.Printf("SQLite error: %v", err)
    }
}
```

SQLite's simplicity, reliability, and zero-configuration nature make it perfect for embedded applications, desktop software, and development environments. Flash ORM provides full support for SQLite's unique features while maintaining the same clean API across all supported languages.
