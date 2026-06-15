---
title: ScyllaDB & Cassandra Guide
description: Using Flash ORM with ScyllaDB and Apache Cassandra databases
---

# ScyllaDB & Cassandra Guide

Complete guide to using Flash ORM with ScyllaDB and Apache Cassandra, including CQL schema design, UDTs, materialized views, and multi-keyspace management.

::: warning Beta Status
ScyllaDB / Cassandra support is currently in **beta**. Most ORM features (migrations, code generation, seeding) work, but some edge cases may require manual tuning. See [release notes](/notes/RELEASE_NOTES) for known limitations.
:::

## Table of Contents

- [Installation & Setup](#installation--setup)
- [Data Types](#data-types)
- [Schema Design](#schema-design)
- [Keyspaces](#keyspaces)
- [User-Defined Types (UDTs)](#user-defined-types-udts)
- [Materialized Views](#materialized-views)
- [Migrations](#migrations)
- [Code Generation](#code-generation)
- [Studio](#studio)

## Installation & Setup

### ScyllaDB Installation

```bash
# Docker (ScyllaDB)
docker run --name scylla -p 9042:9042 -d scylladb/scylla:latest

# Docker (Apache Cassandra)
docker run --name cassandra -p 9042:9042 -d cassandra:latest
```

### Connection Configuration

```bash
# ScyllaDB connection URL
export DATABASE_URL="scylla://localhost:9042/keyspace_name"

# Cassandra connection URL (same format)
export DATABASE_URL="scylla://localhost:9042/keyspace_name"

# With credentials
export DATABASE_URL="scylla://user:password@host1:9042,host2:9042/mykeyspace"

# With query params
export DATABASE_URL="scylla://localhost:9042/mykeyspace?consistency=quorum&timeout=10s&page_size=5000"
```

### Connection Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `consistency` | string | `local_quorum` | Consistency level (one, quorum, local_quorum, all) |
| `dc` | string | — | Preferred data center |
| `token_aware` | bool | `true` | Token-aware routing |
| `num_conns` | int | `2` | Connections per host |
| `page_size` | int | `5000` | Default query page size |
| `timeout` | duration | `11s` | Connection timeout |
| `connect_timeout` | duration | `5s` | Initial connect timeout |
| `keepalive` | duration | `30s` | TCP keepalive interval |
| `ssl` | bool | `false` | Enable TLS |

### Flash ORM Setup

```bash
# Initialize with ScyllaDB
flash init --scylla

# Verify connection
flash status
```

### Auto-Create Default Keyspace

When `DATABASE_URL` has no keyspace path (e.g., `scylla://localhost:9042`), a `flash` keyspace with `SimpleStrategy` / `replication_factor=1` is auto-created on first connection.

## Data Types

### CQL to Go Type Mapping

| CQL Type | Go Type | Notes |
|----------|---------|-------|
| `ascii` | `string` | ASCII text |
| `text` | `string` | UTF-8 text |
| `varchar` | `string` | Variable-length string |
| `int` | `int32` | 32-bit signed int |
| `bigint` | `int64` | 64-bit signed int |
| `smallint` | `int16` | 16-bit signed int |
| `tinyint` | `int8` | 8-bit signed int |
| `varint` | `string` | Arbitrary precision integer |
| `float` | `float32` | 32-bit float |
| `double` | `float64` | 64-bit float |
| `boolean` | `bool` | True/false |
| `decimal` | `string` | Arbitrary precision decimal |
| `counter` | `int64` | Distributed counter |
| `timestamp` | `time.Time` | Date and time |
| `uuid` | `string` | Type 4 UUID |
| `timeuuid` | `string` | Type 1 UUID |
| `inet` | `string` | IP address |
| `date` | `time.Time` | Date only |
| `time` | `time.Time` | Time only |
| `duration` | `string` | nanosecond-precision duration |
| `blob` | `[]byte` | Binary data |
| `map<K,V>` | `map[K]V` | Typed map |
| `list<T>` | `[]T` | Typed list |
| `set<T>` | `[]T` | Typed set (Go: `[]T`) |
| `tuple<T1,T2,...>` | Custom struct | Typed tuple |
| `frozen<T>` | Custom struct | Frozen (immutable) type |

### Collection and UDT Column Types

```sql
-- Collections
CREATE TABLE users (
    id uuid PRIMARY KEY,
    name text,
    emails set<text>,
    tags list<text>,
    metadata map<text, text>,
    addresses list<frozen<address>>  -- frozen UDT in list
);

CREATE TYPE address (
    street text,
    city text,
    zip text
);
```

Generated Go types:

```go
type User struct {
    ID        string             `json:"id"`
    Name      string             `json:"name"`
    Emails    []string           `json:"emails"`
    Tags      []string           `json:"tags"`
    Metadata  map[string]string  `json:"metadata"`
    Addresses []*Address         `json:"addresses"`
}

type Address struct {
    Street string `json:"street"`
    City   string `json:"city"`
    Zip    string `json:"zip"`
}
```

## Schema Design

### Table Structure

```sql
-- db/schema/schema.sql

-- Keyspace-qualified tables work
CREATE TABLE ap.users (
    id uuid PRIMARY KEY,
    username text,
    email text,
    created_at timestamp
);

CREATE TABLE ap.posts (
    id uuid PRIMARY KEY,
    user_id uuid,
    title text,
    content text,
    tags set<text>,
    created_at timestamp
);

-- Clustering columns
CREATE TABLE ap.events (
    user_id uuid,
    event_time timestamp,
    event_type text,
    metadata map<text, text>,
    PRIMARY KEY (user_id, event_time)
) WITH CLUSTERING ORDER BY (event_time DESC);
```

### Clustering Keys

```sql
CREATE TABLE sensor_data (
    sensor_id uuid,
    recorded_at timestamp,
    temperature double,
    humidity double,
    PRIMARY KEY (sensor_id, recorded_at)
) WITH CLUSTERING ORDER BY (recorded_at DESC);
```

### WITH Clauses

Full CQL `WITH` clauses are supported in schema parsing:

```sql
CREATE TABLE metrics (
    id uuid PRIMARY KEY,
    value double
) WITH
    compaction = {'class': 'SizeTieredCompactionStrategy'},
    gc_grace_seconds = 86400,
    default_time_to_live = 604800;
```

## Keyspaces

### Multi-Keyspace Operations

FlashORM's ScyllaDB adapter iterates all user keyspaces (excluding system keyspaces). Tables are returned with keyspace-grouped display names:

```bash
# List all tables across keyspaces
flash status

# Shows:
# ap.users (5 columns, 100 rows)
# ap.posts (6 columns, 500 rows)
# archive.snapshots (4 columns, 50 rows)
```

### Keyspace Resolution

When running operations on qualified table names like `ap.users`, FlashORM extracts the keyspace from the name and connects to the correct keyspace regardless of which keyspace was last active.

```bash
# Pull schema from specific keyspace
flash pull  # pulls all user keyspaces

# Apply migrations to correct keyspace
flash apply  # resolves keyspace from migration metadata
```

## User-Defined Types (UDTs)

### Creating UDTs

```sql
-- db/schema/schema.sql
CREATE TYPE social_link (
    platform text,
    url text
);

CREATE TYPE user_profile (
    display_name text,
    bio text,
    links list<frozen<social_link>>
);

CREATE TABLE users (
    id uuid PRIMARY KEY,
    profile frozen<user_profile>
);
```

### UDT Code Generation

Go codegen produces dedicated struct types for UDTs:

```go
type SocialLink struct {
    Platform string `json:"platform"`
    URL      string `json:"url"`
}

type UserProfile struct {
    DisplayName string        `json:"display_name"`
    Bio         string        `json:"bio"`
    Links       []*SocialLink `json:"links"`
}
```

## Materialized Views

### Creating Materialized Views

```sql
CREATE MATERIALIZED VIEW posts_by_user AS
SELECT user_id, id, title, created_at
FROM ap.posts
WHERE user_id IS NOT NULL AND id IS NOT NULL
PRIMARY KEY (user_id, created_at)
WITH CLUSTERING ORDER BY (created_at DESC);
```

Materialized views are parsed, ordered after base tables in migrations, and generated with `IF NOT EXISTS` for idempotent `flash apply`.

## Migrations

### Generating Migrations

```bash
# Create migration (compares schema files to current snapshot)
flash migrate "add users table"

# Apply all pending migrations
flash apply

# Rollback last migration
flash down

# Check migration status
flash status
```

### Migration SQL

```sql
-- UP migration (auto-generated)
CREATE TABLE IF NOT EXISTS ap.users (
    id uuid PRIMARY KEY,
    name text,
    email text
);

CREATE TYPE IF NOT EXISTS ap.social_link (
    platform text,
    url text
);

CREATE MATERIALIZED VIEW IF NOT EXISTS ap.posts_by_user AS
SELECT user_id, id, title FROM ap.posts
WHERE user_id IS NOT NULL AND id IS NOT NULL
PRIMARY KEY (user_id, id);

-- DOWN migration
DROP MATERIALIZED VIEW IF EXISTS ap.posts_by_user;
DROP TYPE IF EXISTS ap.social_link;
DROP TABLE IF EXISTS ap.users;
```

### System Table Filtering

Reset command excludes ScyllaDB system tables (`system_*`, `system_schema.*`, `system_auth.*`). Schema diff filters out internal tables to prevent accidental DROP statements.

## Code Generation

### Query Files

```sql
-- db/queries/users.sql

-- name: GetUserByID :one
SELECT id, username, email, created_at
FROM ap.users WHERE id = ?;

-- name: CreateUser :exec
INSERT INTO ap.users (id, username, email)
VALUES (uuid(), ?, ?);

-- name: GetUsersByTag :many
SELECT id, username, tags
FROM ap.users WHERE tags CONTAINS ?;
```

### CQL Query Annotations

| Annotation | Return Type | gocql Equivalent |
|------------|-------------|------------------|
| `:one` | Single row | `session.Query(...).Scan(...)` |
| `:many` | Multiple rows | `session.Query(...).Iter().MapScan(...)` |
| `:exec` | No return | `session.Query(...).Exec()` |

### Generated Go Code

```go
// flash_gen/users.go
package flash_gen

import (
    "context"
    "fmt"
    "github.com/apache/cassandra-gocql-driver/v2"  // use gocql import alias
)

type Queries struct {
    session *gocql.Session
}

func New(session *gocql.Session) *Queries {
    return &Queries{session: session}
}

func (q *Queries) GetUserByID(ctx context.Context, id string) (User, error) {
    query := q.session.Query(
        `SELECT id, username, email, created_at FROM ap.users WHERE id = ?`,
        id,
    )
    var row User
    if err := query.Scan(&row.ID, &row.Username, &row.Email, &row.CreatedAt); err != nil {
        return row, fmt.Errorf("GetUserByID: %w", err)
    }
    return row, nil
}
```

### Important Codegen Notes

- **Row struct fields are value types** (`string`, not `*string`) — gocql's MapScan outputs values via `fmt.Sprint`.
- **`database/sql` is NOT imported** for ScyllaDB-generated code — only `context`, `fmt`, and `gocql`.
- **`uuid`/`timeuuid` maps to `string`** in Go structs (not `time.Time`).
- **Collection params** map correctly: `set<text>` → `[]string`, `list<int>` → `[]int32`.
- **`RETURNING` clauses are stripped** — CQL doesn't support them. Queries with `RETURNING` downgrade from `:one` to `:exec`.

### Configuration

```toml
[gen.go]
enabled = true
# gocql driver is automatically used when provider is scylla/cassandra

[gen.go.drivers.gocql]
# No additional config needed for gocql
```

## Studio

### Launching ScyllaDB Studio

```bash
# Connect to ScyllaDB
flash studio "scylla://localhost:9042/mykeyspace"

# With credentials
flash studio "scylla://user:pass@host1:9042,host2:9042/mykeyspace?consistency=quorum"
```

### Keyspace Grouping

The database view shows collapsible keyspace sections with table counts:

```
📁 ap (3 tables)
  ├── users
  ├── posts  
  └── events
📁 archive (1 table)
  └── snapshots
```

### Editor Autocomplete

CQL keyword suggestions are provided in the Studio SQL editor:

**Keywords:** `KEYSPACE`, `MATERIALIZED VIEW`, `CLUSTERING ORDER BY`, `ALLOW FILTERING`, `CONTAINS`, `USING TTL`

**Types:** `uuid`, `timeuuid`, `frozen`, `counter`, `ascii`, `inet`, `varint`, `duration`

### Metrics

The metrics page shows table sizes and row counts per keyspace for ScyllaDB clusters.

## Known Limitations

- **RETURNING not supported** — Move RETURNING columns to a separate SELECT query.
- **No auto-increment IDs** — Use `uuid()` or application-generated UUIDs.
- **Eventually consistent** — Respect CQL consistency levels for your use case.
- **No JOINs** — CQL doesn't support SQL-style JOINs; denormalize or query separately.

## Additional Resources

- [ScyllaDB Documentation](https://opensource.docs.scylladb.com/stable/)
- [CQL Reference](https://cassandra.apache.org/doc/latest/cassandra/cql/)
- [Go CQL Driver (gocql)](https://github.com/apache/cassandra-gocql-driver)
