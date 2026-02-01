---
title: Performance
description: Performance benchmarks and optimization techniques
---

# Performance

Flash ORM delivers exceptional performance through optimized code generation, efficient database connections, and smart caching strategies. Here's a comprehensive look at performance characteristics and optimization techniques.

## Table of Contents

- [Benchmark Results](#benchmark-results)
- [Performance Factors](#performance-factors)
- [Optimization Techniques](#optimization-techniques)
- [Memory Management](#memory-management)
- [Connection Pooling](#connection-pooling)
- [Query Optimization](#query-optimization)
- [Caching Strategies](#caching-strategies)
- [Profiling & Monitoring](#profiling--monitoring)

## Benchmark Results

### Comprehensive Benchmarks

Flash ORM significantly outperforms popular ORMs in real-world scenarios:

| Operation | FlashORM | Drizzle | Prisma |
|-----------|----------|---------|--------|
| Insert 1000 Users | **149ms** | 224ms | 230ms |
| Insert 10 Cat + 5K Posts + 15K Comments | **2410ms** | 3028ms | 3977ms |
| Complex Query x500 | **3156ms** | 12500ms | 56322ms |
| Mixed Workload x1000 (75% read, 25% write) | **186ms** | 1174ms | 10863ms |
| Stress Test Simple Query x2000 | **79ms** | 160ms | 118ms |
| **TOTAL** | **5980ms** | **17149ms** | **71510ms** |

**Performance Gains:**
- **2.8x faster** than Drizzle
- **11.9x faster** than Prisma
- **Up to 20x faster** on complex queries

### Benchmark Methodology

**Test Environment:**
- **Database:** PostgreSQL 15 on SSD storage
- **Hardware:** 8-core CPU, 16GB RAM
- **Network:** Local connection (no network latency)
- **Data:** Realistic dataset with proper indexing

**Test Scenarios:**
1. **Bulk Inserts:** Mass data insertion with constraints
2. **Complex Queries:** Multi-table joins with aggregations
3. **Mixed Workloads:** Combination of reads and writes
4. **Concurrent Operations:** Parallel query execution

## v2.3.0 Performance Improvements

Version 2.3.0 introduces significant performance optimizations across all components:

### Database Adapters (87% Faster)

Migration generation is now **87% faster** on average:

| Database | Improvement | Key Optimizations |
|----------|-------------|-------------------|
| PostgreSQL | 88% faster | Split complex 7-way JOIN into 2 simple queries with Go-side merge |
| MySQL | 82% faster | Constraint-backed index optimization |
| SQLite | 90% faster | Parallelized table column fetching with goroutines |

**PostgreSQL Optimizations:**
- Split complex 7-way JOIN into 2 simple queries (70% faster)
- Replaced expensive subqueries with LEFT JOIN optimization (50-80% faster)
- Pre-allocated maps to reduce GC pressure

**SQLite Optimizations:**
- Parallelized table column fetching with goroutines (10x speedup)
- Eliminated N+1 query problem for unique column checks (90-97% faster)
- Pre-compiled regex patterns in schema parsing

### Code Generators (3-5x Faster Parsing)

All code generators now use pre-compiled regex patterns:

```go
// Before: Compiled on every call
re := regexp.MustCompile(`pattern`)

// After: Pre-compiled and reused
var rePattern = regexp.MustCompile(`pattern`)
```

**Language-Specific Improvements:**

| Language | Improvement | Details |
|----------|-------------|---------|
| Go | Faster `:many` queries | Slice pre-allocation: `make([]T, 0, 8)` |
| Python | Statement caching | `self._stmts` dictionary for prepared statements |
| Python | Optimized row access | Direct Record access vs `dict()` conversion |
| JavaScript | Shared utilities | Removed redundant regex compilation |

### Shared Utilities

New shared utility functions reduce code duplication and improve performance:

```go
// utils/sql.go
utils.ExtractTableName(query)  // Fast table name extraction
utils.IsModifyingQuery(query)  // Detect INSERT/UPDATE/DELETE
utils.SplitColumns(columns)    // Optimized column parsing
```

### Schema Operations

Pre-allocated maps and slices reduce garbage collection pressure:

```go
// Before
result := make(map[string]*Table)

// After: Pre-allocated with estimated capacity
result := make(map[string]*Table, len(tables))
```

## Performance Factors

### Code Generation Efficiency

**Prepared Statements:**
```go
// Generated code uses prepared statements automatically
func (q *Queries) GetUserByID(ctx context.Context, id int64) (User, error) {
    stmt, err := q.prepareStmt(ctx, "GetUserByID", "SELECT id, name, email FROM users WHERE id = $1")
    if err != nil {
        return User{}, err
    }
    row := stmt.QueryRowContext(ctx, id)
    // Direct execution, no query parsing overhead
}
```

**Type Safety Without Runtime Cost:**
- Compile-time type checking
- Zero runtime reflection
- Direct database driver calls

### Database-Specific Optimizations

**PostgreSQL:**
- Native `pgx/v5` driver (fastest Go PostgreSQL driver)
- Binary protocol for better performance
- Prepared statement caching
- Connection pooling with `pgxpool`

**MySQL:**
- Optimized connection configuration
- Query result streaming
- Minimal memory allocation

**SQLite:**
- WAL mode for concurrent reads
- Memory-mapped I/O
- Single-writer, multiple-reader design

## Optimization Techniques

### Connection Pooling

**PostgreSQL Connection Pool:**
```go
config, _ := pgxpool.ParseConfig(dsn)
config.MaxConns = int32(runtime.GOMAXPROCS(0) * 2)  // 2x CPU cores
config.MinConns = 4
config.MaxConnLifetime = 15 * time.Minute
config.MaxConnIdleTime = 3 * time.Minute
config.HealthCheckPeriod = 1 * time.Minute

pool, err := pgxpool.NewWithConfig(context.Background(), config)
```

**Benefits:**
- Connection reuse reduces overhead
- Automatic health checking
- Configurable limits prevent resource exhaustion

### Query Optimization

**Prepared Statement Caching:**
```go
type Queries struct {
    db    DBTX
    stmts map[string]*sql.Stmt  // Statement cache
}

func (q *Queries) prepareStmt(ctx context.Context, name, query string) (*sql.Stmt, error) {
    if stmt, exists := q.stmts[name]; exists {
        return stmt, nil
    }
    stmt, err := q.db.PrepareContext(ctx, query)
    if err != nil {
        return nil, err
    }
    q.stmts[name] = stmt
    return stmt, nil
}
```

**Batch Operations:**
```go
func (q *Queries) CreateUsers(ctx context.Context, users []User) error {
    tx, err := q.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.PrepareContext(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)")
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, user := range users {
        _, err = stmt.ExecContext(ctx, user.Name, user.Email)
        if err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

### Memory Management

**Streaming Large Result Sets:**
```go
func (e *Exporter) ExportLargeTable(ctx context.Context, w io.Writer) error {
    rows, err := e.db.QueryContext(ctx, "SELECT * FROM large_table")
    if err != nil {
        return err
    }
    defer rows.Close()

    encoder := json.NewEncoder(w)
    for rows.Next() {
        var item LargeStruct
        if err := rows.Scan(&item.Field1, &item.Field2); err != nil {
            return err
        }
        // Stream each item immediately
        if err := encoder.Encode(item); err != nil {
            return err
        }
    }
    return nil
}
```

**Memory Pool for Frequent Allocations:**
```go
var userPool = sync.Pool{
    New: func() interface{} {
        return &User{}
    },
}

func getUser() *User {
    return userPool.Get().(*User)
}

func putUser(u *User) {
    // Reset fields
    u.ID = 0
    u.Name = ""
    u.Email = ""
    userPool.Put(u)
}
```

## Connection Pooling

### Advanced Pool Configuration

**Dynamic Pool Sizing:**
```go
func optimizePoolSize() int {
    cpuCores := runtime.GOMAXPROCS(0)
    // Rule of thumb: 2-4 connections per CPU core
    // Adjust based on workload characteristics
    if isWriteHeavy {
        return cpuCores * 2
    }
    return cpuCores * 4
}
```

**Pool Monitoring:**
```go
type PoolStats struct {
    TotalConnections int
    IdleConnections  int
    ActiveConnections int
    WaitCount        int64
    WaitDuration     time.Duration
}

func getPoolStats(pool *pgxpool.Pool) PoolStats {
    stats := pool.Stat()
    return PoolStats{
        TotalConnections: int(stats.TotalConns()),
        IdleConnections:  int(stats.IdleConns()),
        ActiveConnections: int(stats.AcquiredConns()),
        WaitCount:        stats.AcquireCount(),
        WaitDuration:     stats.AcquireDuration(),
    }
}
```

### Connection Health Checks

**Automatic Health Monitoring:**
```go
func monitorPoolHealth(ctx context.Context, pool *pgxpool.Pool) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := pool.Ping(ctx); err != nil {
                log.Printf("Pool health check failed: %v", err)
                // Implement reconnection logic
            }
        }
    }
}
```

## Query Optimization

### Index Utilization

**Generated Code with Index Hints:**
```sql
-- Generated queries automatically work with indexes
SELECT id, name, email FROM users WHERE email = $1
-- Uses index on email column

SELECT * FROM posts WHERE user_id = $1 AND created_at > $2 ORDER BY created_at DESC
-- Uses composite index on (user_id, created_at)
```

**Index Effectiveness Monitoring:**
```sql
-- PostgreSQL: Check index usage
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- MySQL: Index usage
SHOW INDEX FROM users;
ANALYZE TABLE users;
```

### Query Plan Analysis

**Automatic EXPLAIN Integration:**
```go
func (q *Queries) ExplainQuery(ctx context.Context, query string, args ...interface{}) (string, error) {
    explainQuery := fmt.Sprintf("EXPLAIN (FORMAT JSON) %s", query)
    row := q.db.QueryRowContext(ctx, explainQuery, args...)

    var plan string
    if err := row.Scan(&plan); err != nil {
        return "", err
    }

    return plan, nil
}
```

### N+1 Query Prevention

**Eager Loading in Generated Code:**
```sql
// Generated query with joins to prevent N+1
-- name: GetPostsWithAuthors :many
SELECT
    p.id, p.title, p.content, p.created_at,
    u.id as author_id, u.name as author_name, u.email as author_email
FROM posts p
JOIN users u ON p.user_id = u.id
WHERE p.published = true
ORDER BY p.created_at DESC;
```

## Caching Strategies

### Prepared Statement Caching

**Global Statement Cache:**
```go
type StatementCache struct {
    mu    sync.RWMutex
    stmts map[string]*sql.Stmt
}

func (c *StatementCache) Get(ctx context.Context, db DBTX, name, query string) (*sql.Stmt, error) {
    c.mu.RLock()
    if stmt, exists := c.stmts[name]; exists {
        c.mu.RUnlock()
        return stmt, nil
    }
    c.mu.RUnlock()

    c.mu.Lock()
    defer c.mu.Unlock()

    // Double-check after acquiring write lock
    if stmt, exists := c.stmts[name]; exists {
        return stmt, nil
    }

    stmt, err := db.PrepareContext(ctx, query)
    if err != nil {
        return nil, err
    }

    c.stmts[name] = stmt
    return stmt, nil
}
```

### Result Caching

**Application-Level Caching:**
```go
type Cache interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{}, ttl time.Duration)
    Delete(key string)
}

func (q *Queries) GetUserByIDCached(ctx context.Context, id int64) (*User, error) {
    cacheKey := fmt.Sprintf("user:%d", id)

    if cached, found := q.cache.Get(cacheKey); found {
        return cached.(*User), nil
    }

    user, err := q.GetUserByID(ctx, id)
    if err != nil {
        return nil, err
    }

    if user != nil {
        q.cache.Set(cacheKey, user, 5*time.Minute)
    }

    return user, nil
}
```

### Schema Caching

**Parsed Schema Cache:**
```go
type SchemaCache struct {
    mu     sync.RWMutex
    schemas map[string]*Schema
    ttl    time.Duration
}

func (c *SchemaCache) Get(dbURL string) (*Schema, error) {
    c.mu.RLock()
    if schema, exists := c.schemas[dbURL]; exists {
        c.mu.RUnlock()
        return schema, nil
    }
    c.mu.RUnlock()

    // Parse schema from database
    schema, err := parseSchemaFromDB(dbURL)
    if err != nil {
        return nil, err
    }

    c.mu.Lock()
    c.schemas[dbURL] = schema
    c.mu.Unlock()

    // Schedule cleanup
    time.AfterFunc(c.ttl, func() {
        c.mu.Lock()
        delete(c.schemas, dbURL)
        c.mu.Unlock()
    })

    return schema, nil
}
```

## Profiling & Monitoring

### Performance Profiling

**Go Profiling Integration:**
```go
import _ "net/http/pprof"

func startProfiling() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}

// Access profiling data:
// http://localhost:6060/debug/pprof/
// go tool pprof http://localhost:6060/debug/pprof/profile
```

### Query Performance Monitoring

**Query Timing:**
```go
func (q *Queries) withTiming(ctx context.Context, name string, fn func() error) error {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        if duration > 100*time.Millisecond {
            log.Printf("Slow query %s: %v", name, duration)
        }
    }()
    return fn()
}
```

### Database Performance Metrics

**Connection Pool Metrics:**
```go
type Metrics struct {
    QueryCount        int64
    QueryDuration     time.Duration
    ConnectionCount   int64
    ErrorCount        int64
    CacheHitRate      float64
}

func (m *Metrics) RecordQuery(duration time.Duration, err error) {
    atomic.AddInt64(&m.QueryCount, 1)
    atomic.AddInt64((*int64)(&m.QueryDuration), int64(duration))

    if err != nil {
        atomic.AddInt64(&m.ErrorCount, 1)
    }
}
```

### Benchmarking Tools

**Built-in Benchmarking:**
```bash
# Run performance benchmarks
flash benchmark --database postgres --duration 30s --concurrency 10

# Output:
# Benchmark Results:
# Total Queries: 45,231
# Average Latency: 2.1ms
# P95 Latency: 8.5ms
# QPS: 1,507
# Error Rate: 0.01%
```

### Memory Profiling

**Memory Usage Analysis:**
```go
func printMemoryStats() {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)

    fmt.Printf("Memory Stats:\n")
    fmt.Printf("  Alloc: %d KB\n", m.Alloc/1024)
    fmt.Printf("  TotalAlloc: %d KB\n", m.TotalAlloc/1024)
    fmt.Printf("  Sys: %d KB\n", m.Sys/1024)
    fmt.Printf("  NumGC: %d\n", m.NumGC)
}
```

### Database-Specific Profiling

**PostgreSQL:**
```sql
-- Enable query logging
SET log_statement = 'all';
SET log_duration = on;
SET log_min_duration_statement = 100;  -- Log queries > 100ms

-- View active queries
SELECT * FROM pg_stat_activity;

-- Query performance analysis
SELECT
    query,
    calls,
    total_time,
    mean_time,
    rows
FROM pg_stat_statements
ORDER BY mean_time DESC;
```

**MySQL:**
```sql
-- Enable slow query log
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 1;  -- Log queries > 1 second

-- Show process list
SHOW PROCESSLIST;

-- Query analysis
SHOW ENGINE INNODB STATUS;
```

Flash ORM's performance optimizations ensure it delivers exceptional speed while maintaining type safety and developer productivity. The combination of efficient code generation, optimized database connections, and smart caching strategies results in performance that rivals hand-written SQL code.
