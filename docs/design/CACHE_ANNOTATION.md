# `@cache` Annotation — Design Spec

## Overview

The `@cache` annotation generates intelligent caching wrappers for queries with:

- TTL-based expiration
- Tag-based purging
- Dependency-based auto-invalidation (when dependent mutations run, related caches are purged)
- Named cache accessors (`.get()` / `.del()`)

## Annotation Syntax

```sql
-- name: GetUser :one
-- @cache {"ttl": "30s", "name": "GetuserC", "tags": ["users", "profile"], "dep": ["UpdateUserById", "DeleteUser"]}
SELECT id, name, email, avatar_url FROM users WHERE id = $1;
```

### Fields

| Field  | Type     | Required | Description                                                          |
| ------ | -------- | -------- | -------------------------------------------------------------------- |
| `ttl`  | string   | Yes      | Time-to-live: `"30s"`, `"5m"`, `"1h"`, `"24h"`                       |
| `name` | string   | No       | Custom cache accessor name. Default: query name + "Cache"            |
| `tags` | string[] | No       | Tags for bulk purging. `cache.purgeByTag("users")` clears all tagged |
| `dep`  | string[] | No       | Query names that invalidate this cache when executed                 |

## Generated Code (Kotlin Example)

### Input

```sql
-- name: GetUser :one
-- @cache {"ttl": "30s", "name": "UserCache", "tags": ["users"], "dep": ["UpdateUser", "DeleteUser"]}
SELECT id, name, email, avatar_url FROM users WHERE id = $1;

-- name: UpdateUser :exec
UPDATE users SET name = $1, email = $2 WHERE id = $3;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
```

### Generated Output

```kotlin
// ═══ Cache Infrastructure (generated once in FlashCache.kt) ═══

class FlashCache {
    private val store = ConcurrentHashMap<String, CacheEntry<*>>()
    private val tagIndex = ConcurrentHashMap<String, MutableSet<String>>()  // tag → keys
    private val scheduler = ScheduledThreadPoolExecutor(1)

    data class CacheEntry<T>(val value: T, val expiresAt: Long)

    fun <T> get(key: String): T? {
        val entry = store[key] as? CacheEntry<T> ?: return null
        if (System.currentTimeMillis() > entry.expiresAt) {
            store.remove(key)
            return null
        }
        return entry.value
    }

    fun <T> set(key: String, value: T, ttlMs: Long, tags: List<String> = emptyList()) {
        store[key] = CacheEntry(value, System.currentTimeMillis() + ttlMs)
        for (tag in tags) {
            tagIndex.getOrPut(tag) { ConcurrentHashMap.newKeySet() }.add(key)
        }
    }

    fun del(key: String) {
        store.remove(key)
    }

    fun purgeByTag(tag: String) {
        tagIndex[tag]?.forEach { store.remove(it) }
        tagIndex.remove(tag)
    }

    fun purgeByPrefix(prefix: String) {
        store.keys.filter { it.startsWith(prefix) }.forEach { store.remove(it) }
    }

    fun clear() {
        store.clear()
        tagIndex.clear()
    }
}

// ═══ Named Cache Accessor (generated per @cache with name) ═══

object UserCache {
    private val cache = FlashCache.instance  // singleton

    fun get(id: UUID): GetUserRow? {
        return cache.get("UserCache:$id")
    }

    fun del(id: UUID) {
        cache.del("UserCache:$id")
    }

    fun purge() {
        cache.purgeByPrefix("UserCache:")
    }

    internal fun set(id: UUID, value: GetUserRow) {
        cache.set("UserCache:$id", value, ttlMs = 30_000, tags = listOf("users"))
    }
}

// ═══ Query Method (modified to use cache) ═══

fun getUser(id: UUID): GetUserRow? {
    // Check cache first
    UserCache.get(id)?.let { return it }

    // Cache miss — query DB
    val sql = """SELECT id, name, email, avatar_url FROM users WHERE id = ?;"""
    val stmt = stmts.getOrPut("getUser") { conn.prepareStatement(sql) }
    stmt.setObject(1, id)
    val result = stmt.executeQuery().use { rs ->
        if (rs.next()) GetUserRow(...) else null
    }

    // Store in cache
    if (result != null) {
        UserCache.set(id, result)
    }
    return result
}

// ═══ Dependent Mutation (auto-purges cache) ═══

fun updateUser(name: String, email: String, id: UUID) {
    val sql = """UPDATE users SET name = ?, email = ? WHERE id = ?;"""
    val stmt = stmts.getOrPut("updateUser") { conn.prepareStatement(sql) }
    stmt.setString(1, name)
    stmt.setString(2, email)
    stmt.setObject(3, id)
    stmt.executeUpdate()

    // Auto-purge dependent caches (from @cache dep)
    UserCache.del(id)  // purge by the ID param
}

fun deleteUser(id: UUID) {
    val sql = """DELETE FROM users WHERE id = ?;"""
    val stmt = stmts.getOrPut("deleteUser") { conn.prepareStatement(sql) }
    stmt.setObject(1, id)
    stmt.executeUpdate()

    // Auto-purge dependent caches
    UserCache.del(id)
}
```

## Generated Code (Go Example)

```go
// flash_cache.go
type FlashCache struct {
    mu    sync.RWMutex
    store map[string]*cacheEntry
}

type cacheEntry struct {
    value     interface{}
    expiresAt time.Time
    tags      []string
}

var Cache = &FlashCache{store: make(map[string]*cacheEntry)}

func (c *FlashCache) Get(key string) (interface{}, bool) { ... }
func (c *FlashCache) Set(key string, value interface{}, ttl time.Duration, tags ...string) { ... }
func (c *FlashCache) Del(key string) { ... }
func (c *FlashCache) PurgeByTag(tag string) { ... }

// Named accessor
var UserCache = &userCacheAccessor{}

type userCacheAccessor struct{}

func (a *userCacheAccessor) Get(id int64) (*GetUserRow, bool) {
    key := fmt.Sprintf("UserCache:%d", id)
    v, ok := Cache.Get(key)
    if !ok { return nil, false }
    return v.(*GetUserRow), true
}

func (a *userCacheAccessor) Del(id int64) {
    Cache.Del(fmt.Sprintf("UserCache:%d", id))
}

// Query method with cache
func (q *Queries) GetUser(id int64) (GetUserRow, error) {
    if cached, ok := UserCache.Get(id); ok {
        return *cached, nil
    }
    // ... DB query ...
    UserCache.Set(id, &result)
    return result, nil
}

// Mutation with auto-purge
func (q *Queries) UpdateUser(name string, email string, id int64) error {
    // ... DB update ...
    UserCache.Del(id)  // auto-purge
    return nil
}
```

## Generated Code (TypeScript Example)

```typescript
// flash_cache.ts
class FlashCache {
    private store = new Map<string, { value: any; expiresAt: number; tags: string[] }>();

    get<T>(key: string): T | null { ... }
    set<T>(key: string, value: T, ttlMs: number, tags?: string[]) { ... }
    del(key: string) { ... }
    purgeByTag(tag: string) { ... }
}

export const cache = new FlashCache();

// Named accessor
export const UserCache = {
    get: (id: string) => cache.get<GetUserResult>(`UserCache:${id}`),
    del: (id: string) => cache.del(`UserCache:${id}`),
    purge: () => cache.purgeByPrefix("UserCache:"),
};

// Query with cache
async getUserById(id: string): Promise<GetUserResult | null> {
    const cached = UserCache.get(id);
    if (cached) return cached;

    const result = await this.pool.query(sql, [id]);
    if (result.rows[0]) UserCache.set(id, result.rows[0]);
    return result.rows[0] || null;
}

// Mutation with auto-purge
async updateUser(name: string, email: string, id: string): Promise<void> {
    await this.pool.query(sql, [name, email, id]);
    UserCache.del(id);  // auto-purge
}
```

## Dependency Resolution

### How `dep` works

When `@cache` has `"dep": ["UpdateUser", "DeleteUser"]`:

1. Parser stores the dependency list on the cached query
2. Code generator, when generating `UpdateUser` and `DeleteUser`:
   - Finds all `@cache` queries that list them as dependencies
   - Determines the cache key param (matches param names between cached query and mutation)
   - Injects `CacheName.del(matchingParam)` after the mutation executes

### Key Matching Algorithm

```
GetUser($1=id) → cached with key "UserCache:$id"
UpdateUser($1=name, $2=email, $3=id) → dep of GetUser

Match: GetUser's key param is "id" (first param)
       UpdateUser has "id" at position $3
       → Generate: UserCache.del(id)  // uses the id param from UpdateUser
```

### Multiple Dependencies

```sql
-- name: GetUserPosts :many
-- @cache {"ttl": "1m", "dep": ["CreatePost", "UpdatePost", "DeletePost"]}
SELECT * FROM posts WHERE user_id = $1;

-- name: CreatePost :one
INSERT INTO posts (title, user_id) VALUES ($1, $2) RETURNING *;
```

Match: GetUserPosts key = "user_id", CreatePost has "user_id" at $2
→ Generate: `GetUserPostsCache.del(userId)` in CreatePost

## Tag-Based Purging

```sql
-- @cache {"ttl": "5m", "tags": ["users", "admin-panel"]}
```

User can manually purge:

```kotlin
FlashCache.instance.purgeByTag("users")      // purges all user-related caches
FlashCache.instance.purgeByTag("admin-panel") // purges all admin caches
```

## Implementation Plan

### Phase 1: Parser

- [ ] Parse `@cache` JSON annotation in `parseQueryFile`
- [ ] Add `CacheDef` struct to `Query` type
- [ ] Validate dep references (warn if dep query doesn't exist)

### Phase 2: Cache Infrastructure

- [ ] Generate `FlashCache.kt` / `flash_cache.go` / `flash_cache.ts` / `flash_cache.py`
- [ ] Thread-safe in-memory implementation
- [ ] TTL expiration (lazy on read + optional background cleanup)

### Phase 3: Named Accessors

- [ ] Generate `object TypeNameCache` with `.get()` / `.del()` / `.purge()`
- [ ] Key format: `"{name}:{param1}:{param2}"`

### Phase 4: Query Integration

- [ ] Wrap cached queries: check cache → miss → query DB → store
- [ ] Return type unchanged (transparent to caller)

### Phase 5: Dependency Injection

- [ ] For each mutation query, find all @cache queries that dep on it
- [ ] Match key params by name
- [ ] Inject `.del(matchedParam)` after mutation executes

### Phase 6: Tag Support

- [ ] Track tags in cache store
- [ ] Generate `purgeByTag()` method
- [ ] Expose on FlashCache singleton
