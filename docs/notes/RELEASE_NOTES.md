# flash 2.7.6

**Release Date:** 2026-07-02

## 🚀 New Features

### `-- @json` Annotation — Typed JSON Columns

Define typed data classes for JSONB/JSON columns directly in your query files. Generated code includes full serialization/deserialization — no manual parsing needed.

**Inline definition:**
```sql
-- name: GetUser :one
-- @json settings {"theme": "string", "language": "string", "font_size": "int"}
-- @json metadata {"level": "int", "badges": "string[]", "xp_points": "int"}
SELECT id, name, settings, metadata FROM users WHERE id = $1;
```

**Import from file:**
```sql
-- name: GetUser :one
-- @json import user_settings.json as settings
-- @json import user_metadata.json as metadata
SELECT id, name, settings, metadata FROM users WHERE id = $1;
```

**Config — set JSON files directory:**
```toml
json_path = "db/json"
```

**Supported types:** `string`, `int`, `float`, `boolean`, `string[]`, `int[]`, `float[]`, `boolean[]`, `any`

**Generated output (Kotlin example):**
```kotlin
data class Settings(
    val theme: String?,
    val language: String?,
    val fontSize: Int?,
    val raw: JsonObject? = null  // access unmentioned fields
) {
    fun toJson(): String
    fun get(key: String): String?  // get any field not in the type definition
    companion object {
        fun fromJson(json: String?): Settings?
    }
}

// Row class uses typed fields directly:
data class GetUserRow(
    val id: UUID,
    val name: String,
    val settings: Settings?,   // auto-parsed from JSON
    val metadata: Metadata?    // auto-parsed from JSON
)

// Params for INSERT/UPDATE are typed too:
fun updateSettings(settings: Settings, id: UUID)
// Generated: stmt.setString(1, settings.toJson())
```

**Key behaviors:**
- All JSON fields are nullable by default
- Multiple `-- @json` annotations per query supported
- Works with SELECT (return types), INSERT, and UPDATE (param types)
- Unmentioned fields accessible via `.raw` or `.get("key")` — nothing is lost
- JSON types generated in shared `JsonTypes.kt` file — no duplicates
- Columns without `@json` remain as raw `String`/`bytes` — no breakage

**All 5 languages supported:**

| Language | Type | Serialization |
|----------|------|--------------|
| Kotlin | `data class` + Gson | `.fromJson()` / `.toJson()` |
| Java | `record` + Gson | `Gson.fromJson()` / `Gson.toJson()` |
| Go | `struct` with `json` tags | `json.Unmarshal()` / `json.Marshal()` |
| TypeScript | `interface` | `JSON.parse() as Type` |
| Python | `@dataclass` | `TypeName(**json.loads())` |

### Smart Migration: Column Rename Detection

FlashORM now detects column renames instead of treating them as drop + add (which loses data).

```sql
-- Before: username VARCHAR(100)
-- After:  display_name VARCHAR(100)

-- Generated migration:
ALTER TABLE "users" RENAME COLUMN "username" TO "display_name";

-- Down migration:
ALTER TABLE "users" RENAME COLUMN "display_name" TO "username";
```

- Detects renames when a column disappears and a new one with the **same type** appears
- Only triggers for unambiguous matches (exactly one candidate)
- Works for PostgreSQL, MySQL, SQLite, ClickHouse

### Smart Migration: Table Rename Detection

Same pattern for entire tables — preserves all data.

```sql
-- Generated:
ALTER TABLE "posts" RENAME TO "articles";
-- Down:
ALTER TABLE "articles" RENAME TO "posts";
```

### Improved Down Migrations

- Dropped tables: down migration now generates full `CREATE TABLE` from schema snapshot
- Dropped enums: down migration generates `CREATE TYPE ... AS ENUM(...)` from snapshot
- Reversible column/table renames in both directions

### SQLite Trigger Support in Migrations

`BEGIN...END` blocks in SQLite triggers are now parsed correctly. Previously, semicolons inside trigger bodies would break the migration.

```sql
CREATE TRIGGER trg_likes_insert
AFTER INSERT ON likes
BEGIN
    UPDATE posts SET likes_count = likes_count + 1 WHERE id = NEW.post_id;
END;
```

---

## 🔧 Bug Fixes

### LATERAL Subquery Params

Fixed parameter name/type inference for queries with `$N = ANY(col)` inside LATERAL subqueries:

```sql
-- Before: params named param1, param2, param3 with wrong types
-- After:  users: UUID, channelId: UUID, id: UUID (correct!)
SELECT m.*, COALESCE(r.reactions, '[]'::jsonb) AS reactions
FROM messages m
LEFT JOIN LATERAL (
    SELECT jsonb_agg(jsonb_build_object('me', $1 = ANY(grouped.users))) AS reactions
    FROM (...) grouped
) r ON TRUE
WHERE m.channel_id = $2 AND m.id = $3;
```

- `$N = ANY(qualified.col)` — handles dotted column names like `grouped.users`
- `? = ANY(...)` — new pattern for question-mark style params
- WHERE clause `?` param mismatch with subqueries — fixed position counting
- Plural-to-singular ID fallback: `users` → tries `user_id` for type inference

### COALESCE Type Inference

`COALESCE(a.attachments, '[]'::jsonb)` now correctly resolves to `JSONB` (was `TEXT`).

- Detects `::type` casts on fallback arguments
- Detects `jsonb_agg()`, `json_agg()`, `jsonb_build_object()`, `to_jsonb()` as JSONB
- LATERAL alias columns resolved via schema lookup

### Array Type Mapping (Kotlin/Java)

`UUID[]` now correctly maps to `List<UUID>` (was `UUID` — missing the array wrapper).

- Fixed switch case ordering: `[]` suffix check before `contains("uuid")` 
- Affects all array types: `INT[]` → `List<Int>`, `TEXT[]` → `List<String>`, etc.

### `jsonb_set()` Param Names

```sql
-- Before: param1, param2, id
-- After:  path, value, id
UPDATE users SET metadata = jsonb_set(COALESCE(metadata, '{}'), $1, $2) WHERE id = $3;
```

### JSONB Aggregate Functions

Added type detection for `JSONB_AGG()`, `JSON_AGG()`, `JSONB_BUILD_OBJECT()`, `JSONB_BUILD_ARRAY()`, `TO_JSONB()` — all resolve to `JSONB` type.

---

## 📦 Version

`2.7.6` (prod) · `2.7.6-beta-dev` (dev)
