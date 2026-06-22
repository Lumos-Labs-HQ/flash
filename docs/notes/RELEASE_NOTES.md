# flash 2.6.0

**Release Date:** 2026-06-22

## 🚀 New Features

### Kotlin & Java Code Generation

FlashORM now generates fully type-safe **Kotlin** and **Java** code from your SQL schemas and queries — joining Go, TypeScript/JavaScript, and Python.

#### Kotlin (`[gen.kotlin]`)
- Generates `Models.kt` — `data class` per table, `enum class` per enum
- Generates `{QueryFile}.kt` — `class UsersQueries` with typed methods per query
- Generates `Queries.kt` — unified aggregator: `Queries.newq(conn)` as single entry point
- Row data classes (`GetUserWithPostCountRow`) emitted top-level for JOIN/aggregate results
- Drivers: **JDBC** (default), **Exposed**, **R2DBC**, **DataStax CqlSession** (ScyllaDB)
- Typed JDBC getters by column name (`rs.getInt("id")`, `rs.getString("email")`, etc.)
- `ResultSet.use {}` for automatic resource cleanup
- Nullable-safe: LEFT JOIN / computed columns use `String?`, `LocalDateTime?` etc.

#### Java (`[gen.java]`)
- Generates one `public record` per table and per enum — each in its own `.java` file
- Generates `{QueryFile}Queries.java` — typed methods per query
- Generates `Queries.java` — unified aggregator with `Queries.newq(conn)`
- Row types (`GetUserWithPostCountRow.java`) as separate top-level `public record` files
- Drivers: **JDBC** (default), **jOOQ**, **Hibernate**, **DataStax CqlSession** (ScyllaDB)
- Java text block `"""` with correct newline
- Typed JDBC getters (`rs.getObject("id", UUID.class)`, `rs.getTimestamp("col").toLocalDateTime()`, etc.)

#### Config (`flash.toml`)
```toml
[gen.kotlin]
enabled = true
out = "src/main/kotlin/com/example/db/flashgen"
package = "com.example.db.flashgen"   # sets package declaration in all .kt files
driver = "jdbc"                        # jdbc (default) | exposed | r2dbc

[gen.java]
enabled = true
out = "src/main/java/com/example/db/flashgen"
package = "com.example.db.flashgen"   # sets package declaration in all .java files
driver = "jdbc"                        # jdbc (default) | jooq | hibernate
```

#### CQL typed getters (ScyllaDB / Cassandra)
Both Kotlin and Java use proper DataStax typed API:
- `row.getUuid("id")`, `row.getString("name")`, `row.getLong("count")`
- `row.getSet("tags", String::class.java)` / `row.getSet("tags", String.class)`
- `row.getList("items", UUID::class.java)`, `row.getMap("meta", String.class, Object.class)`
- `SimpleStatement.newInstance(sql, args...)` for parameterized execution

### Project Auto-Detection on `flash init`

`flash init` now auto-detects your project type and generates the correct `[gen.*]` section:

| Detected by | Project type | Generated section |
|-------------|-------------|-------------------|
| `build.gradle.kts`, `settings.gradle.kts` | Kotlin | `[gen.kotlin]` |
| `build.gradle` with `kotlin` keyword | Kotlin | `[gen.kotlin]` |
| `pom.xml` with `kotlin` keyword | Kotlin | `[gen.kotlin]` |
| `pom.xml` (no kotlin) | Java | `[gen.java]` |
| `build.gradle` (no kotlin) | Java | `[gen.java]` |
| `package.json` | Node.js | `[gen.js]` |
| `requirements.txt`, `pyproject.toml`, `setup.py` | Python | `[gen.python]` |

Kotlin always takes priority over Java when both indicators are present.

### `env_path` in `flash.toml`

Specify a custom `.env` file path. Loaded after the default `.env` / `.env.local` with override semantics:

```toml
env_path = "config/.env.production"
```

---

## 🔧 Bug Fixes

### Migration Generator
- **Duplicate `FOREIGN KEY` (PostgreSQL)** — PostgreSQL's `FormatColumnType` already emits inline `REFERENCES "table"("col") ON DELETE ...` per column. The generator was also appending a separate `FOREIGN KEY (...) REFERENCES ...` block, producing duplicate constraints. Fixed: the separate block is removed for PostgreSQL.
- **Composite `PRIMARY KEY` split into multiple columns** — Tables with `PRIMARY KEY (user_id, post_id)` were generating `user_id ... PRIMARY KEY` and `post_id ... PRIMARY KEY` separately — invalid SQL. Fixed for PostgreSQL, MySQL, and SQLite: detects multiple `IsPrimary` columns and emits a single `PRIMARY KEY (col1, col2)` table constraint.

### `flash raw` — Comment-prefixed SQL files
Files where INSERT/UPDATE statements are preceded by comment blocks (`-- Seed: users`) were silently dropped — the entire chunk after splitting on `;` started with `--`. Fixed: comment lines are stripped per-chunk before checking for real SQL content.

### Code Generation — Row type naming
`utils.ToPascalCase("GetUserWithPostCount")` mangled already-PascalCase query names to `"Getuserwithpostcount"`. Added `queryPascal()` which preserves existing PascalCase names, so Row types are now correctly named `GetUserWithPostCountRow` instead of `GetuserwithpostcountRow`.

### Kotlin — Row data classes top-level
Row data classes were being emitted inside the `class UsersQueries { }` body — Kotlin forbids nested `data class`. Fixed: Row types are now emitted before the class declaration.

### Java — `public record` in one file per type
Multiple `public record` / `public enum` types in a single `Models.java` caused `Class X should be declared in a file named X.java` errors. Fixed: each type gets its own `.java` file.

### Kotlin — NULL safety for JOIN results
`getString(...)` on LEFT JOIN columns (e.g. `category_name` from `LEFT JOIN categories`) returned `null` at runtime, crashing with `NullPointerException: getString(...) must not be null`. Fixed: Row data class fields for non-primitive types use nullable Kotlin types (`String?`, `LocalDateTime?`), and `ktTypedGetter` uses nullable-safe getters for those columns.

### Kotlin — Default driver
The empty driver `""` was incorrectly defaulting to Exposed (`org.jetbrains.exposed.sql.Database`), causing `Unresolved reference 'Database'` for projects without the Exposed dependency. Fixed: `""` now defaults to JDBC (`java.sql.Connection`).

### Kotlin — Redundant `.toString()` on String params
`stmt.setString(N, name.toString())` was generated for `String` parameters — redundant and flagged by the IDE. Fixed: `stmt.setString(N, name)` for text types, `stmt.setObject(N, value)` for enums/arrays.

---

## 📦 Version

`2.6.0` (prod) · `2.6.0-beta-dev` (dev)
