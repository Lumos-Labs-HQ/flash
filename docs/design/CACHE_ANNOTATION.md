# `@cache` Annotation — Final Architecture

## Overview

The `@cache` annotation generates intelligent caching wrappers for queries using **Redis** with:

- **Redis-backed** caching using official Redis libraries per language
- **TTL-based expiration** handled natively by Redis
- **Tag-based purging** for bulk invalidation
- **Dependency-based auto-invalidation** — mutations automatically purge related caches
- **Named cache accessors** with `.get()` / `.del()` / `.purge()`
- **All 6 languages supported**: Go, TypeScript, Python, Kotlin, Java, Rust

## Configuration

```toml
# flash.toml

[cache]
enabled = true                    # false (default) — no cache code generated
redis_url_env = "REDIS_URL"       # env var name for Redis connection
default_ttl = "5m"                # global default TTL if @cache omits ttl
prefix = "flash"                  # key prefix for namespacing: "flash:UserCache:123"
```

### Configuration Rules

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `false` | When false, `@cache` annotations are ignored — no cache code generated |
| `redis_url_env` | `"REDIS_URL"` | Environment variable holding Redis URL |
| `default_ttl` | `"5m"` | Default TTL when `@cache` annotation omits `ttl` field |
| `prefix` | `"flash"` | Key prefix: `{prefix}:{cacheName}:{params}` |

### Environment Variable

```bash
# .env
REDIS_URL=redis://localhost:6379
# or with auth:
REDIS_URL=redis://:password@redis-host:6379/0
# or cluster:
REDIS_URL=redis://node1:6379,node2:6379,node3:6379
```

## Annotation Syntax

```sql
-- name: GetUser :one
-- @cache {"ttl": "30s", "name": "UserCache", "tags": ["users", "profile"], "dep": ["UpdateUser", "DeleteUser"]}
SELECT id, name, email, avatar_url FROM users WHERE id = $1;
```

### Fields

| Field  | Type     | Required | Description |
|--------|----------|----------|-------------|
| `ttl`  | string   | No       | Time-to-live: `"30s"`, `"5m"`, `"1h"`, `"24h"`. Uses `default_ttl` if omitted |
| `name` | string   | No       | Custom cache name. Default: `QueryName + "Cache"` |
| `tags` | string[] | No       | Tags for bulk purging |
| `dep`  | string[] | No       | Query names that auto-invalidate this cache when executed |

## Architecture

```
┌─────────────────────────────────────────────┐
│            Generated Code                    │
├─────────────────────────────────────────────┤
│                                             │
│  ┌─────────────────────┐                   │
│  │   FlashCache        │ ← Redis client     │
│  │   .get(key)         │                   │
│  │   .set(key,val,ttl) │                   │
│  │   .del(key)         │                   │
│  │   .purge_tag(tag)   │                   │
│  └──────────┬──────────┘                   │
│             │                               │
│        ┌────▼────┐                          │
│        │  Redis   │                          │
│        │  Server  │                          │
│        └──────────┘                          │
│                                             │
│  ┌─────────────────────┐                   │
│  │  Named Accessors    │                   │
│  │  UserCache.get(id)  │                   │
│  │  UserCache.del(id)  │                   │
│  └─────────────────────┘                   │
│                                             │
│  ┌─────────────────────┐                   │
│  │  Query Methods      │                   │
│  │  getUser() → cache  │                   │
│  └─────────────────────┘                   │
│                                             │
│  ┌─────────────────────┐                   │
│  │  Mutations          │                   │
│  │  updateUser() →     │                   │
│  │    auto-purge deps  │                   │
│  └─────────────────────┘                   │
└─────────────────────────────────────────────┘
```

## Redis Libraries Per Language

| Language | Library | Crate/Package |
|----------|---------|---------------|
| **Go** | `github.com/redis/go-redis/v9` | Official Redis Go client |
| **TypeScript** | `ioredis` | Most popular, cluster support |
| **Python** | `redis-py` (`redis`) | Official Redis Python client |
| **Kotlin** | `io.lettuce:lettuce-core` | Reactive Redis for JVM |
| **Java** | `io.lettuce:lettuce-core` | Same as Kotlin |
| **Rust** | `redis` crate | Official Redis Rust client |

## Generated File Structure

```
flash_gen/
├── cache.{ext}            # FlashCache — Redis client wrapper
├── cache_accessors.{ext}  # Named accessors (UserCache, PostCache, etc.)
└── ... (query files with cache integration)
```

## Generated Code — All Languages

### Key Format

```
{prefix}:{cacheName}:{param1}:{param2}
```

Example: `flash:UserCache:42` or `flash:UserPostsCache:user_123`

### Serialization

JSON serialize/deserialize all cached values (`serde_json` / `JSON.stringify` / `json.dumps` / `Gson` / `encoding/json`)

---

## Go — Generated Code

### `cache.go`

```go
package flash_gen

import (
    "context"
    "encoding/json"
    "os"
    "time"

    "github.com/redis/go-redis/v9"
)

type FlashCache struct {
    client *redis.Client
    prefix string
    ctx    context.Context
}

func NewFlashCache(prefix string) *FlashCache {
    url := os.Getenv("REDIS_URL")
    opts, _ := redis.ParseURL(url)
    return &FlashCache{
        client: redis.NewClient(opts),
        prefix: prefix,
        ctx:    context.Background(),
    }
}

func (c *FlashCache) Get(key string, dest interface{}) bool {
    val, err := c.client.Get(c.ctx, c.prefix+":"+key).Bytes()
    if err != nil { return false }
    json.Unmarshal(val, dest)
    return true
}

func (c *FlashCache) Set(key string, value interface{}, ttl time.Duration, tags ...string) {
    data, _ := json.Marshal(value)
    c.client.Set(c.ctx, c.prefix+":"+key, data, ttl)
    for _, tag := range tags {
        c.client.SAdd(c.ctx, c.prefix+":tag:"+tag, key)
    }
}

func (c *FlashCache) Del(key string) {
    c.client.Del(c.ctx, c.prefix+":"+key)
}

func (c *FlashCache) PurgeByTag(tag string) {
    tagKey := c.prefix + ":tag:" + tag
    keys, _ := c.client.SMembers(c.ctx, tagKey).Result()
    for _, k := range keys {
        c.client.Del(c.ctx, c.prefix+":"+k)
    }
    c.client.Del(c.ctx, tagKey)
}

func (c *FlashCache) Close() {
    c.client.Close()
}
```

---

## Rust — Generated Code

### `cache.rs`

```rust
use redis::{Client, Commands};
use serde::{Serialize, de::DeserializeOwned};
use std::time::Duration;

pub struct FlashCache {
    client: Client,
    prefix: String,
}

impl FlashCache {
    pub fn new(prefix: &str) -> Self {
        let url = std::env::var("REDIS_URL").expect("REDIS_URL not set");
        let client = Client::open(url).expect("Failed to connect to Redis");
        Self { client, prefix: prefix.to_string() }
    }

    pub fn get<T: DeserializeOwned>(&self, key: &str) -> Option<T> {
        let mut conn = self.client.get_connection().ok()?;
        let full_key = format!("{}:{}", self.prefix, key);
        let data: Option<String> = conn.get(&full_key).ok()?;
        data.and_then(|d| serde_json::from_str(&d).ok())
    }

    pub fn set<T: Serialize>(&self, key: &str, value: &T, ttl: Duration, tags: &[&str]) {
        if let Ok(mut conn) = self.client.get_connection() {
            let full_key = format!("{}:{}", self.prefix, key);
            let data = serde_json::to_string(value).unwrap();
            let _: () = conn.set_ex(&full_key, &data, ttl.as_secs() as u64).unwrap_or(());
            for tag in tags {
                let tag_key = format!("{}:tag:{}", self.prefix, tag);
                let _: () = conn.sadd(&tag_key, key).unwrap_or(());
            }
        }
    }

    pub fn del(&self, key: &str) {
        if let Ok(mut conn) = self.client.get_connection() {
            let full_key = format!("{}:{}", self.prefix, key);
            let _: () = conn.del(&full_key).unwrap_or(());
        }
    }

    pub fn purge_by_tag(&self, tag: &str) {
        if let Ok(mut conn) = self.client.get_connection() {
            let tag_key = format!("{}:tag:{}", self.prefix, tag);
            let keys: Vec<String> = conn.smembers(&tag_key).unwrap_or_default();
            for k in &keys {
                let full_key = format!("{}:{}", self.prefix, k);
                let _: () = conn.del(&full_key).unwrap_or(());
            }
            let _: () = conn.del(&tag_key).unwrap_or(());
        }
    }
}
```

---

## TypeScript — Generated Code

### `cache.ts`

```typescript
import Redis from "ioredis";

export class FlashCache {
    private redis: Redis;
    private prefix: string;

    constructor(prefix: string = "flash") {
        this.redis = new Redis(process.env.REDIS_URL!);
        this.prefix = prefix;
    }

    async get<T>(key: string): Promise<T | null> {
        const data = await this.redis.get(`${this.prefix}:${key}`);
        return data ? JSON.parse(data) : null;
    }

    async set<T>(key: string, value: T, ttlMs: number, tags: string[] = []) {
        const fullKey = `${this.prefix}:${key}`;
        await this.redis.set(fullKey, JSON.stringify(value), "PX", ttlMs);
        for (const tag of tags) {
            await this.redis.sadd(`${this.prefix}:tag:${tag}`, key);
        }
    }

    async del(key: string) {
        await this.redis.del(`${this.prefix}:${key}`);
    }

    async purgeByTag(tag: string) {
        const tagKey = `${this.prefix}:tag:${tag}`;
        const keys = await this.redis.smembers(tagKey);
        for (const k of keys) {
            await this.redis.del(`${this.prefix}:${k}`);
        }
        await this.redis.del(tagKey);
    }

    async close() {
        await this.redis.quit();
    }
}
```

---

## Python — Generated Code

### `cache.py` (Redis backend)

```python
import os
import json
from typing import TypeVar, Optional, List
from dataclasses import asdict
import redis

T = TypeVar("T")

class FlashCache:
    def __init__(self, prefix: str = "flash"):
        url = os.environ.get("REDIS_URL", "redis://localhost:6379")
        self.client = redis.from_url(url)
        self.prefix = prefix

    def get(self, key: str, cls: type[T]) -> Optional[T]:
        data = self.client.get(f"{self.prefix}:{key}")
        if data is None:
            return None
        return cls(**json.loads(data))

    def set(self, key: str, value, ttl_seconds: int, tags: List[str] = None):
        full_key = f"{self.prefix}:{key}"
        data = json.dumps(asdict(value) if hasattr(value, "__dataclass_fields__") else value)
        self.client.setex(full_key, ttl_seconds, data)
        for tag in (tags or []):
            self.client.sadd(f"{self.prefix}:tag:{tag}", key)

    def delete(self, key: str):
        self.client.delete(f"{self.prefix}:{key}")

    def purge_by_tag(self, tag: str):
        tag_key = f"{self.prefix}:tag:{tag}"
        keys = self.client.smembers(tag_key)
        for k in keys:
            self.client.delete(f"{self.prefix}:{k.decode()}")
        self.client.delete(tag_key)
```

---

## Kotlin/Java — Generated Code

### `FlashCache.kt` (Redis backend)

```kotlin
import io.lettuce.core.RedisClient
import io.lettuce.core.api.sync.RedisCommands
import com.google.gson.Gson

class FlashCache(private val prefix: String = "flash") {
    private val client = RedisClient.create(System.getenv("REDIS_URL") ?: "redis://localhost:6379")
    private val connection = client.connect()
    private val commands: RedisCommands<String, String> = connection.sync()
    private val gson = Gson()

    fun <T> get(key: String, clazz: Class<T>): T? {
        val data = commands.get("$prefix:$key") ?: return null
        return gson.fromJson(data, clazz)
    }

    fun <T> set(key: String, value: T, ttlSeconds: Long, tags: List<String> = emptyList()) {
        val fullKey = "$prefix:$key"
        commands.setex(fullKey, ttlSeconds, gson.toJson(value))
        for (tag in tags) {
            commands.sadd("$prefix:tag:$tag", key)
        }
    }

    fun del(key: String) {
        commands.del("$prefix:$key")
    }

    fun purgeByTag(tag: String) {
        val tagKey = "$prefix:tag:$tag"
        val keys = commands.smembers(tagKey)
        for (k in keys) {
            commands.del("$prefix:$k")
        }
        commands.del(tagKey)
    }

    fun close() {
        connection.close()
        client.shutdown()
    }
}
```

---

## Query Integration Pattern

### Cached Query (all languages follow this pattern):

```
fn get_user(id) → Result<User>:
    1. Build key: "{prefix}:UserCache:{id}"
    2. Try cache.get(key)
    3. If hit → return cached value
    4. If miss → execute SQL query
    5. Store result: cache.set(key, result, ttl, tags)
    6. Return result
```

### Mutation with Auto-Purge:

```
fn update_user(name, email, id):
    1. Execute SQL mutation
    2. For each @cache that lists this in dep:
       - Match param names (find "id" param)
       - Call cache.del("{prefix}:UserCache:{id}")
```

---

## Dependency Resolution Algorithm

```
@cache on GetUser(id) → key = "UserCache:{id}"
  dep: ["UpdateUser", "DeleteUser"]

UpdateUser(name, email, id) → matches "id" param
  → inject: cache.del("UserCache:{id}") after mutation

DeleteUser(id) → matches "id" param  
  → inject: cache.del("UserCache:{id}") after mutation
```

### Multi-param keys:

```
@cache on GetUserPosts(user_id, status) → key = "UserPostsCache:{user_id}:{status}"
  dep: ["CreatePost"]

CreatePost(title, body, user_id) → matches "user_id" but NOT "status"
  → inject: cache.purge_prefix("UserPostsCache:{user_id}:") // wildcard purge
```

---

## Config Struct (Go)

```go
type CacheConfig struct {
    Enabled     bool   `toml:"enabled"`       // false by default
    RedisURLEnv string `toml:"redis_url_env"` // env var name, default "REDIS_URL"
    DefaultTTL  string `toml:"default_ttl"`   // "5m" default
    Prefix      string `toml:"prefix"`        // "flash" default
}
```

---

## Implementation Plan

### Phase 1: Config & Parser
- [ ] Add `[cache]` section to Config struct
- [ ] Parse `-- @cache {...}` annotations in query parser
- [ ] Add `CacheDef` to `Query` struct
- [ ] Validate `dep` references exist

### Phase 2: Redis Cache Generation
- [ ] Generate `cache.{ext}` with Redis client wrapper
- [ ] Use official Redis libraries per language
- [ ] JSON serialization for all cached values
- [ ] Tag tracking via Redis SETs
- [ ] TTL via Redis SETEX/PX

### Phase 3: Named Accessors
- [ ] Generate typed accessor per `@cache` annotation
- [ ] Key building from params
- [ ] `.get()` / `.del()` / `.purge()` methods

### Phase 4: Query Wrapping
- [ ] Inject cache-check-first logic into cached queries
- [ ] Transparent to caller (same return type)
- [ ] Handle Redis errors gracefully (fallback to DB on connection failure)

### Phase 5: Auto-Invalidation
- [ ] Resolve `dep` → find dependent mutations
- [ ] Match params by name between cached query and mutation
- [ ] Inject purge calls after mutation execution
- [ ] Handle partial key matches with prefix purge

### Phase 6: All Languages
- [ ] Go (github.com/redis/go-redis/v9)
- [ ] TypeScript (ioredis)
- [ ] Python (redis-py)
- [ ] Kotlin (io.lettuce:lettuce-core)
- [ ] Java (io.lettuce:lettuce-core)
- [ ] Rust (redis crate)
