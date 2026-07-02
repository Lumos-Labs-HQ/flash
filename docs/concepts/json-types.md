# Typed JSON Columns

FlashORM generates typed data classes for JSONB/JSON columns using `@json` annotations. This eliminates manual JSON parsing and gives you compile-time type safety for your JSON data.

## Overview

Instead of working with raw JSON strings:
```kotlin
val raw: String? = row.settings  // {"theme":"dark","font_size":14}
// Manual parsing needed every time...
```

FlashORM generates typed classes:
```kotlin
val settings: Settings? = row.settings
val theme = settings?.theme         // String?
val fontSize = settings?.fontSize   // Int?
```

## Quick Start

### 1. Define the JSON structure

**Inline (in query file):**
```sql
-- name: GetUser :one
-- @json settings {"theme": "string", "language": "string", "font_size": "int"}
SELECT id, name, settings FROM users WHERE id = $1;
```

**Import from file:**
```sql
-- name: GetUser :one
-- @json import user_settings.json as settings
SELECT id, name, settings FROM users WHERE id = $1;
```

### 2. Create JSON type file (for imports)

`db/json/user_settings.json`:
```json
{
    "theme": "string",
    "language": "string",
    "font_size": "int",
    "notifications": "boolean"
}
```

### 3. Configure `json_path` in flash.toml

```toml
json_path = "db/json"
```

### 4. Run code generation

```bash
flash gen
```

## Annotation Syntax

### Inline Definition

```sql
-- @json <column_name> {"field": "type", "field2": "type", ...}
```

The column name must match a column in the SELECT list.

### Import from File

```sql
-- @json import <filename>.json as <column_name>
```

The file is resolved relative to `json_path` in your flash.toml config.

### Multiple Annotations

You can annotate multiple JSON columns per query:

```sql
-- name: GetUserFull :one
-- @json settings {"theme": "string", "font_size": "int"}
-- @json metadata {"level": "int", "badges": "string[]"}
-- @json preferences {"notifications": "boolean", "email_digest": "boolean"}
SELECT id, name, settings, metadata, preferences FROM users WHERE id = $1;
```

## Supported Types

| Type | Description | Kotlin | Go | TypeScript | Python |
|------|-------------|--------|-----|-----------|--------|
| `string` | Text value | `String?` | `*string` | `string \| null` | `Optional[str]` |
| `int` | Integer | `Int?` | `*int64` | `number \| null` | `Optional[int]` |
| `float` | Decimal | `Double?` | `*float64` | `number \| null` | `Optional[float]` |
| `boolean` | True/false | `Boolean?` | `*bool` | `boolean \| null` | `Optional[bool]` |
| `string[]` | String array | `List<String>?` | `[]string` | `string[] \| null` | `Optional[List[str]]` |
| `int[]` | Integer array | `List<Int>?` | `[]int64` | `number[] \| null` | `Optional[List[int]]` |
| `float[]` | Decimal array | `List<Double>?` | `[]float64` | `number[] \| null` | `Optional[List[float]]` |
| `boolean[]` | Boolean array | `List<Boolean>?` | `[]bool` | `boolean[] \| null` | `Optional[List[bool]]` |
| `any` | Any value | `Any?` | `interface{}` | `any` | `Any` |

## Accessing Unmentioned Fields

All generated JSON types include a `raw` field that preserves the full JSON object. Fields you didn't define in `@json` are still accessible:

```kotlin
// @json settings {"theme": "string", "font_size": "int"}
// But the actual JSON also has "sidebar_width", "color_scheme", etc.

val settings = row.settings
val theme = settings?.theme              // typed: String?
val fontSize = settings?.fontSize        // typed: Int?

// Access fields NOT in the @json definition:
val sidebar = settings?.get("sidebar_width")     // String? via helper
val raw = settings?.raw?.get("color_scheme")     // direct JsonObject access
```

Nothing is lost — `@json` provides typed access to known fields while preserving access to everything else.

## Works With All Query Types

### SELECT (return columns)

```sql
-- name: GetUser :one
-- @json settings {"theme": "string"}
SELECT id, name, settings FROM users WHERE id = $1;
```

Generated return type uses `Settings?` instead of `String?`.

### INSERT with RETURNING

```sql
-- name: CreateUser :one
-- @json settings {"theme": "string"}
INSERT INTO users (name, settings) VALUES ($1, $2)
RETURNING id, name, settings;
```

Both the param `settings` and the return column are typed as `Settings`.

### UPDATE

```sql
-- name: UpdateSettings :one
-- @json settings {"theme": "string", "font_size": "int"}
UPDATE users SET settings = $1, updated_at = NOW() WHERE id = $2
RETURNING id, settings;
```

The param accepts a `Settings` object, serialized to JSON automatically.

### Multiple queries, same type

If multiple queries use `-- @json import settings.json as settings`, the type is generated **once** in a shared file — no duplicates.

## Generated Code by Language

### Kotlin

```kotlin
// JsonTypes.kt (shared file)
data class Settings(
    val theme: String?,
    val fontSize: Int?,
    val raw: JsonObject? = null
) {
    fun toJson(): String
    fun get(key: String): String?
    companion object {
        fun fromJson(json: String?): Settings?
    }
}

// Usage:
val user = queries.getUser(id)
val theme = user?.settings?.theme
```

### Go

```go
// json_typed.go
type Settings struct {
    Theme    *string `json:"theme"`
    FontSize *int64  `json:"font_size"`
}

// Usage:
user, _ := q.GetUser(ctx, id)
var settings Settings
json.Unmarshal([]byte(*user.Settings), &settings)
```

### TypeScript

```typescript
// index.d.ts
interface Settings {
    theme: string | null;
    font_size: number | null;
}

// Usage:
const user = await queries.getUser(id)
const settings = JSON.parse(user.settings) as Settings
```

### Python

```python
# json_typed.py
@dataclass
class Settings:
    """JSON type for column 'settings'"""
    theme: Optional[str]
    font_size: Optional[int]

# Usage:
user = await queries.get_user(id)
settings = Settings(**json.loads(user.settings))
```

### Java

```java
// Settings.java
public record Settings(
    String theme,
    Integer fontSize
) {}

// Usage:
var user = queries.getUser(id);
var settings = new Gson().fromJson(user.settings(), Settings.class);
```

## JSON File Format

JSON type definition files use a simple key-value format:

```json
{
    "field_name": "type",
    "nested_field": "type"
}
```

**Example — `db/json/notification_data.json`:**
```json
{
    "post_id": "string",
    "comment_id": "string",
    "action": "string",
    "actor_name": "string",
    "preview": "string",
    "read_at": "string",
    "extra_ids": "string[]"
}
```

Field names use `snake_case` in JSON but are converted to `camelCase` in Kotlin/Java generated code.

## Configuration

```toml
# flash.toml
json_path = "db/json"   # directory for @json import files (relative to project root)
```

If `json_path` is not set, imports are resolved relative to the working directory.

## Best Practices

1. **Use imports for shared types** — If multiple queries use the same JSON structure, define it in a `.json` file and import it. This keeps the type in one place.

2. **Define only the fields you use** — You don't need to list every possible field. The `raw` accessor handles the rest.

3. **Use `any` sparingly** — It provides no type safety. Prefer specific types where possible.

4. **Keep JSON files simple** — One flat object per file. Nested objects aren't supported yet.

5. **Field names match JSON keys** — Use the exact key names from your JSON data (snake_case). The generated code converts them to the language's convention automatically.
