---
title: ClickHouse Guide
description: Using Flash ORM with ClickHouse databases
---

# ClickHouse Guide

Complete guide to using Flash ORM with ClickHouse, including columnar schema design, engine selection, and analytics-optimized migrations.

::: warning Beta Status
ClickHouse support is currently in **beta**. Core ORM features (migrations, code generation) work, but some edge cases may require manual tuning.
:::

## Table of Contents

- [Installation & Setup](#installation--setup)
- [Data Types](#data-types)
- [Table Engines](#table-engines)
- [Schema Design](#schema-design)
- [Migrations](#migrations)
- [Code Generation](#code-generation)
- [Studio](#studio)

## Installation & Setup

### ClickHouse Installation

```bash
# Docker
docker run --name clickhouse -p 8123:8123 -p 9000:9000 \\
  -e CLICKHOUSE_DB=default \\
  -d clickhouse/clickhouse-server:latest

# Linux
sudo apt install clickhouse-server clickhouse-client
sudo systemctl start clickhouse-server
```

### Connection Configuration

```bash
# Native protocol (port 9000)
export DATABASE_URL="clickhouse://localhost:9000/default"

# With credentials
export DATABASE_URL="clickhouse://user:password@localhost:9000/mydb"

# With query params
export DATABASE_URL="clickhouse://localhost:9000/mydb?dial_timeout=10s&max_open_conns=5"

# HTTP protocol (port 8123)
export DATABASE_URL="clickhouse://localhost:8123/default?protocol=http"
```

### Flash ORM Setup

```bash
# Initialize with ClickHouse
flash init --clickhouse

# Verify connection
flash status
```

## Data Types

### ClickHouse to Go Type Mapping

| ClickHouse Type | Go Type | Notes |
|-----------------|---------|-------|
| `Int8` | `int8` | 8-bit signed |
| `Int16` | `int16` | 16-bit signed |
| `Int32` | `int32` | 32-bit signed |
| `Int64` | `int64` | 64-bit signed |
| `UInt8` | `uint8` | 8-bit unsigned |
| `UInt16` | `uint16` | 16-bit unsigned |
| `UInt32` | `uint32` | 32-bit unsigned |
| `UInt64` | `uint64` | 64-bit unsigned |
| `Float32` | `float32` | 32-bit float |
| `Float64` | `float64` | 64-bit float |
| `String` | `string` | Variable-length string |
| `FixedString(N)` | `string` | Fixed-length string |
| `Date` | `time.Time` | Date only |
| `Date32` | `time.Time` | Wide-range date |
| `DateTime` | `time.Time` | Date and time |
| `DateTime64` | `time.Time` | Sub-second datetime |
| `UUID` | `string` | UUID |
| `Bool` | `bool` | Boolean |
| `Decimal(P,S)` | `string` | Fixed-point decimal |
| `JSON` | `string` or `[]byte` | JSON object type |

## Table Engines

### Recommended Engines

```sql
-- MergeTree family (default, analytics-optimized)
CREATE TABLE events (
    event_date Date,
    event_type String,
    user_id UInt64,
    properties String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(event_date)
ORDER BY (event_type, event_date);

-- ReplacingMergeTree (deduplication)
CREATE TABLE users (
    id UInt64,
    name String,
    email String,
    updated_at DateTime64
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY id;

-- SummingMergeTree (auto-aggregation)
CREATE TABLE page_views (
    page String,
    date Date,
    views UInt64
) ENGINE = SummingMergeTree()
ORDER BY (page, date);

-- AggregatingMergeTree (materialized aggregations)
CREATE TABLE hourly_metrics (
    metric_name String,
    hour DateTime,
    count SimpleAggregateFunction(sum, UInt64)
) ENGINE = AggregatingMergeTree()
ORDER BY (metric_name, hour);
```

### Migration Table Engine

FlashORM uses `ReplacingMergeTree()` for the `_flash_migrations` tracking table to support the append-only nature of ClickHouse:

```sql
CREATE TABLE IF NOT EXISTS _flash_migrations (
    id              String,
    migration_name  String,
    checksum        String,
    started_at      DateTime DEFAULT now(),
    finished_at     Nullable(DateTime),
    applied_steps_count UInt32 DEFAULT 0
) ENGINE = ReplacingMergeTree()
ORDER BY id
```

## Schema Design

### Table Structure

```sql
-- db/schema/schema.sql

CREATE TABLE analytics.events (
    event_date Date DEFAULT today(),
    event_type String,
    user_id UInt64,
    platform String,
    properties String,
    created_at DateTime64 DEFAULT now64()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(event_date)
ORDER BY (event_type, event_date);

CREATE TABLE analytics.user_profiles (
    id UInt64,
    name String,
    email String,
    signup_date Date,
    updated_at DateTime64 DEFAULT now64()
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY id;
```

### Partitioning

```sql
-- Monthly partitions
CREATE TABLE orders (
    order_id UInt64,
    customer_id UInt64,
    order_date Date,
    amount Decimal(10,2)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(order_date)
ORDER BY (customer_id, order_date);

-- Yearly partitions
PARTITION BY toYear(order_date)

-- Daily partitions
PARTITION BY toYYYYMMDD(order_date)
```

### ORDER BY Design

```sql
-- Optimize for event type queries
ORDER BY (event_type, event_date)

-- Optimize for user-centric queries
ORDER BY (user_id, event_date)

-- Multi-column sparse index
ORDER BY (platform, event_type, event_date)
```

## Migrations

### Generating Migrations

```bash
flash migrate "create events table"
flash apply
flash status
```

### ClickHouse Migration Notes

- **No traditional transactions** — ClickHouse doesn't support ROLLBACK/COMMIT in the SQL sense. Each migration statement is atomic at the partition level.
- **No DROP COLUMN** (older versions) — Some ClickHouse versions don't support `ALTER TABLE ... DROP COLUMN`. Use software-level migration strategies.
- **ALTER TABLE is fast** — Adding columns, changing defaults is nearly instantaneous.

### Migration File Example

```sql
-- UP: 20240615120000_create_events.up.sql
CREATE TABLE IF NOT EXISTS analytics.events (
    event_date Date DEFAULT today(),
    event_type String,
    user_id UInt64,
    properties String,
    created_at DateTime64 DEFAULT now64()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(event_date)
ORDER BY (event_type, event_date);

-- DOWN: 20240615120000_create_events.down.sql
DROP TABLE IF EXISTS analytics.events;
```

## Code Generation

### Query Files

```sql
-- db/queries/events.sql

-- name: GetEventsByType :many
SELECT event_type, user_id, properties, event_date
FROM analytics.events
WHERE event_type = $1
ORDER BY event_date DESC
LIMIT 100;

-- name: InsertEvent :exec
INSERT INTO analytics.events (event_type, user_id, platform, properties)
VALUES ($1, $2, $3, $4);

-- name: GetDailyMetrics :many
SELECT
    event_type,
    event_date,
    count() as event_count
FROM analytics.events
WHERE event_date >= $1
GROUP BY event_type, event_date
ORDER BY event_date DESC;
```

### Generated Go Code

```go
// flash_gen/events.go
package flash_gen

import (
    "context"
    "database/sql"
    "time"
)

type Event struct {
    EventType  string    `json:"event_type"`
    UserID     uint64    `json:"user_id"`
    Properties string    `json:"properties"`
    EventDate  time.Time `json:"event_date"`
}

func (q *Queries) GetEventsByType(ctx context.Context, eventType string) ([]Event, error) {
    rows, err := q.db.QueryContext(ctx,
        `SELECT event_type, user_id, properties, event_date
         FROM analytics.events WHERE event_type = $1
         ORDER BY event_date DESC LIMIT 100`,
        eventType,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var items []Event
    for rows.Next() {
        var item Event
        if err := rows.Scan(&item.EventType, &item.UserID, &item.Properties, &item.EventDate); err != nil {
            return nil, err
        }
        items = append(items, item)
    }
    return items, rows.Err()
}
```

### Configuration

```toml
[gen.go]
enabled = true
# Standard database/sql driver works for ClickHouse
```

## Studio

### Launching ClickHouse Studio

```bash
flash studio "clickhouse://localhost:9000/default"
flash studio "clickhouse://user:password@localhost:9000/mydb"
```

Studio provides table browsing, query execution, and schema viewing for ClickHouse databases, including engine type and partition key display.

### Studio Features

- Table browser with engine/partition info
- Query editor with ClickHouse SQL syntax support
- Schema viewer showing ORDER BY and PARTITION BY clauses
- Data preview with pagination

## Known Limitations

- **No traditional transactions** — ClickHouse is append-optimized. Migration rollbacks may not fully revert data.
- **Upsert behavior** — Use ReplacingMergeTree with FINAL for deduplication.
- **Point queries slower** — ClickHouse is designed for analytical (OLAP) workloads, not OLTP point queries.
- **No FOREIGN KEYs** — ClickHouse doesn't support referential constraints.

## Additional Resources

- [ClickHouse Documentation](https://clickhouse.com/docs)
- [ClickHouse Go Driver](https://github.com/ClickHouse/clickhouse-go)
- [ClickHouse SQL Reference](https://clickhouse.com/docs/en/sql-reference)
