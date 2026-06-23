# flash 2.6.1

**Release Date:** 2026-06-23

## 🔧 Bug Fixes & Improvements

### Args Object Threshold Changed: 3 → 2

Queries with **more than 2 parameters** now receive a typed args object instead of flat positional arguments. Previously the threshold was 3, so 3-param queries still used flat args. This was inconsistent — it is now consistently applied to any query with more than 2 params.

**Before (3+ params, flat):**

```go
func (q *Queries) CreateUserFull(name string, email string, age int32) (Users, error)
```

**After (>2 params, args object):**

```go
type CreateUserFullParams struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int32  `json:"age"`
}
func (q *Queries) CreateUserFull(arg CreateUserFullParams) (Users, error)
```

This applies consistently across all code generation targets:

| Language   | Args type                                   |
| ---------- | ------------------------------------------- |
| Go         | `type XxxParams struct { ... }`             |
| TypeScript | `export interface XxxArgs { ... }`          |
| Python     | `class XxxArgs(TypedDict): ...`             |
| Kotlin     | `data class XxxArgs(val ..., ...)`          |
| Java       | `public record XxxArgs(Type field, ...) {}` |

---

### `IN ($1, $2, $3)` Rewritten to `= ANY($1)`

Queries using `col IN ($1, $2, $3)` were parsed as multiple separate parameters, producing broken generated code like `name1 string, name2 string, name3 string`. This is now fixed at the parser level.

**Before:**

```sql
-- name: GetUsersByNames :many
SELECT id, name, email FROM users WHERE name IN ($1, $2, $3);
```

Generated 3 params: `name1 TEXT`, `name2 TEXT`, `name3 TEXT` → became `name1, name2, name3` args.

**After:**
The parser rewrites the SQL to `WHERE name = ANY($1)` before code generation, collapsing the list into a single array parameter and renumbering subsequent params.

```go
// Go
func (q *Queries) GetUsersByNames(name []string) ([]GetUsersByNamesRow, error)
```

```ts
// TypeScript
getUsers(name: string[]): Promise<GetUsersByNamesRow[]>
```

```python
# Python
async def get_users_by_names(self, name: List[str]) -> List[GetUsersByNamesRow]: ...
```

Multiple `IN` lists in the same query are each collapsed independently with correct renumbering:

```sql
-- Before: WHERE name IN ($1, $2) AND email IN ($3, $4, $5) AND status = $6
-- After:  WHERE name = ANY($1) AND email = ANY($2) AND status = $3
```

A single-value `IN ($1)` is left unchanged (no rewrite needed).

---

### Args Class Naming Fixed (Python / all languages)

Query names like `CreateUserFull` were being mangled by `ToPascalCase` to `Createuserfull`, producing `class CreateuserfullArgs` instead of `class CreateUserFullArgs`. Fixed by using `QueryPascal` which preserves already-PascalCase names.

This affected row class names, args class names, and return type names across all generators.

---

### Auto-Detected `package` for Kotlin & Java

The `package` field in `[gen.kotlin]` and `[gen.java]` is now **automatically detected** from your build file on `flash init` — no need to write it manually.

| Build file         | Detection method                           |
| ------------------ | ------------------------------------------ |
| `build.gradle.kts` | Reads `group = "..."` or `group("...")`    |
| `build.gradle`     | Reads `group '...'` or `group = '...'`     |
| `pom.xml`          | Reads `<groupId>...</groupId>` (`:` → `.`) |

**Before (manual):**

```toml
[gen.kotlin]
enabled = true
out = "src/main/kotlin/com/example/db/flashgen"
package = "com.example.db.flashgen"   # had to write this yourself
```

**After (auto-detected):**

```toml
[gen.kotlin]
enabled = true
out = "src/main/kotlin/com/example/db/flashgen"
package = "com.example.flashgen"      # inferred from build.gradle.kts: group = "com.example"
```

If detection fails (no build file or no group), the field is omitted and can be set manually.

---
