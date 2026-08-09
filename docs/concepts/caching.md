# Query Caching

FlashORM provides built-in Redis-backed query caching with automatic invalidation. Add a `@cache` annotation to any query and FlashORM generates cache-first wrappers, named accessors, and auto-purge mutations.

## Configuration

Add a `[cache]` section to your `flash.toml`:

```toml
[cache]
enabled = true
redis_url_env = "REDIS_URL"     # env var for Redis connection
default_ttl = "5m"              # default TTL when @cache omits ttl
prefix = "flash"                # key prefix: "flash:UserCache:42"
```

Set your Redis URL in `.env`:

```bash
REDIS_URL=redis://localhost:6379
# or with auth:
REDIS_URL=redis://:password@redis-host:6379/0
```

## Annotation Syntax

```sql
-- name: GetUser :one
-- @cache {"ttl": "30s", "name": "UserCache", "tags": ["users"], "dep": ["UpdateUser", "DeleteUser"]}
SELECT id, name, email FROM users WHERE id = $1;
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ttl` | string | No | Time-to-live: `"30s"`, `"5m"`, `"1h"`. Uses `default_ttl` if omitted |
| `name` | string | No | Custom cache name. Default: `QueryName + "Cache"` |
| `tags` | string[] | No | Tags for bulk purging |
| `dep` | string[] | No | Query names that auto-invalidate this cache |

### TTL Formats

- `"30s"` — 30 seconds
- `"5m"` — 5 minutes
- `"1h"` — 1 hour
- `"24h"` — 24 hours
- `"7d"` — 7 days

## Generated Code

When `[cache] enabled = true`, FlashORM generates three additional files:

| File | Purpose |
|------|---------|
| `cache.{ext}` | Redis client wrapper (`FlashCache`) |
| `cache_accessors.{ext}` | Named accessors + typed tag constants |
| `cached_queries.{ext}` | Cache-first wrappers + auto-purge mutations |

## Usage

### Cache-First Queries

For every `@cache` annotated query, a `*Cached` wrapper is generated:

::: code-group

```go [Go]
// Direct (always hits database)
user, err := queries.Getuser(42)

// Cached (checks Redis first, falls back to DB, stores result)
user, err := queries.GetuserCached(42)
```

```typescript [TypeScript]
// Direct
const user = await queries.getUser(42);

// Cached
const user = await getUserCached(queries, 42);
```

```python [Python]
# Direct
user = await queries.get_user(42)

# Cached
user = await get_user_cached(queries, 42)
```

```rust [Rust]
// Direct
let user = queries.get_user(42).await?;

// Cached
let user = queries.get_user_cached(42).await?;
```

```kotlin [Kotlin]
// Direct
val user = queries.getUser(42)

// Cached
val user = cachedQueries.getUserCached(42, User::class.java)
```

```java [Java]
// Direct
Users user = queries.getUser(42);

// Cached
Users user = cachedQueries.getUserCached(42, Users.class);
```

:::

### Auto-Purge Mutations

Mutations listed in `dep` get an `*AndPurge` wrapper that automatically invalidates dependent caches:

::: code-group

```go [Go]
// Direct (does NOT purge cache)
err := queries.Updateuser(arg)

// Auto-purge (updates DB, then purges UserCache, UserProfileCache, etc.)
err := queries.UpdateuserAndPurge(arg)
```

```typescript [TypeScript]
await updateUserAndPurge(queries, name, email, bio, id);
```

```python [Python]
await update_user_and_purge(queries, name, email, bio, id)
```

```rust [Rust]
queries.update_user_and_purge(params).await?;
```

:::

### Named Accessors (Manual Control)

Each `@cache` generates a named accessor for manual cache operations:

::: code-group

```go [Go]
// Get from cache
result, hit := flash_gen.UserCache.Get(42)

// Set manually
flash_gen.UserCache.Set(42, userData)

// Delete specific key
flash_gen.UserCache.Del(42)

// Purge all UserCache entries
flash_gen.UserCache.Purge()
```

```rust [Rust]
UserCache::get::<User>(42);
UserCache::set(42, &user);
UserCache::del(42);
UserCache::purge();
```

```typescript [TypeScript]
await UserCache.get(42);
await UserCache.set(42, user);
await UserCache.del(42);
await UserCache.purge();
```

:::

### Tag-Based Purging (Type-Safe)

Tags allow bulk invalidation across multiple caches. FlashORM generates typed constants so you can't typo a tag name:

::: code-group

```go [Go]
// Type-safe — compiler catches typos
flash_gen.Cache.PurgeTag(flash_gen.TagUsers)
flash_gen.Cache.PurgeTag(flash_gen.TagPosts)

// Available constants (auto-generated from your @cache tags):
// TagUsers, TagPosts, TagComments, TagFollowers, TagNotifications, TagTags
```

```rust [Rust]
// Enum-based — compiler enforced
CacheTag::Users.purge();
CacheTag::Posts.purge();
```

```typescript [TypeScript]
// Enum-based
await purgeTag(CacheTag.Users);
await purgeTag(CacheTag.Posts);
```

```python [Python]
# Enum-based
purge_tag(CacheTag.USERS)
purge_tag(CacheTag.POSTS)
```

```kotlin [Kotlin]
CacheTag.USERS.purge()
CacheTag.POSTS.purge()
```

```java [Java]
CacheTag.USERS.purge();
CacheTag.POSTS.purge();
```

:::

## Dependency Resolution

The `dep` field controls automatic cache invalidation:

```sql
-- name: GetUser :one
-- @cache {"ttl": "60s", "name": "UserCache", "dep": ["UpdateUser", "DeleteUser"]}
SELECT * FROM users WHERE id = $1;

-- name: GetUserProfile :one
-- @cache {"ttl": "30s", "name": "UserProfileCache", "dep": ["UpdateUser", "CreatePost"]}
SELECT u.*, COUNT(p.id) AS posts FROM users u LEFT JOIN posts p ...

-- name: UpdateUser :exec
UPDATE users SET name = $2, email = $3 WHERE id = $1;
```

When `UpdateUserAndPurge(id, ...)` runs:
1. Executes the UPDATE
2. Purges `UserCache:{id}` (param `id` matches)
3. Purges `UserProfileCache:{id}` (param `id` matches)

### Partial Key Matching

If the mutation doesn't have all the cache key params, it purges by prefix:

```sql
-- @cache on GetUserPosts(user_id, status) → key = "Cache:{user_id}:{status}"
--   dep: ["CreatePost"]

-- CreatePost has user_id but NOT status
-- → purges "Cache:{user_id}:*" (all statuses for that user)
```

## Redis Libraries

| Language | Library |
|----------|---------|
| Go | `github.com/redis/go-redis/v9` |
| TypeScript | `ioredis` |
| Python | `redis-py` |
| Kotlin | `io.lettuce:lettuce-core` |
| Java | `io.lettuce:lettuce-core` |
| Rust | `redis` crate |

## Key Format

```
{prefix}:{cacheName}:{param1}:{param2}
```

Examples:
- `flash:UserCache:42`
- `flash:PostSlugCache:hello-world`
- `flash:FeedCache:20:0`

## Best Practices

1. **Short TTLs for frequently changing data** — `"5s"` to `"30s"` for feeds/notifications
2. **Longer TTLs for stable data** — `"5m"` to `"1h"` for user profiles, tag lists
3. **Always list mutations in `dep`** — ensures cache consistency
4. **Use tags for related caches** — `"tags": ["users"]` lets you purge all user-related caches at once
5. **Use `*AndPurge` wrappers** — don't call raw mutations if you want auto-invalidation
6. **Graceful degradation** — if Redis is down, cached queries fall back to database automatically
