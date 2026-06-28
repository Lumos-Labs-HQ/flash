# `-- @required` Annotation (CQL)

ScyllaDB/Cassandra tables have no `NOT NULL` constraints. By default, all non-PK columns generate nullable params. The `-- @required` annotation lets you declare which params must be non-null in the generated code.

## Usage

### All params required

```sql
-- @required: *
-- name: CreateUser :exec
INSERT INTO myapp.users (id, username, email, bio, tags, created_at)
VALUES (?, ?, ?, ?, ?, ?);
```

**Generated:**
```kotlin
data class CreateUserParams(
    val id: UUID,
    val username: String,
    val email: String,
    val bio: String,
    val tags: Set<String>,
    val createdAt: Instant
)
```

### Specific params required

```sql
-- @required: id, username, email
-- name: CreateUser :exec
INSERT INTO myapp.users (id, username, email, bio, tags, created_at)
VALUES (?, ?, ?, ?, ?, ?);
```

**Generated:**
```kotlin
data class CreateUserParams(
    val id: UUID,
    val username: String,
    val email: String,
    val bio: String?,        // nullable (not in @required)
    val tags: Set<String>?,  // nullable
    val createdAt: Instant?  // nullable
)
```

## Placement

The annotation can appear before OR after the `-- name:` line:

```sql
-- @required: id, username
-- name: CreateUser :exec
INSERT INTO ...

-- OR --

-- name: CreateUser :exec
-- @required: id, username
INSERT INTO ...
```

## Validation

Invalid column names produce a build error:

```sql
-- @required: nonexistent_col
-- name: CreateUser :exec
INSERT INTO myapp.users (id, username) VALUES (?, ?);
```

```
Error: @required param "nonexistent_col" not found in query "CreateUser" params.
       Available: [id, username]
```

## SQL Databases

For PostgreSQL/MySQL/SQLite, params automatically use `NOT NULL` from the schema. `@required` is not needed but works if specified.

## All Language Output

| Language | Non-null | Nullable |
|----------|----------|----------|
| Kotlin | `String` | `String?` |
| TypeScript | `field: string` | `field?: string \| null` |
| Python | `str` | `Optional[str]` |
| Go | `string` | `*string` |
| Java | `String` (always boxed) | `String` (always boxed) |
