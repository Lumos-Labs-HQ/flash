# Flash ORM Release Notes

---

# FlashORM 2.8.2

**Release Date:** 2026-08-25

## New Generated FLASH.md Guide

`flash init` now generates a more accurate, self-contained `FLASH.md` guide
for developers and AI coding agents. The guide documents the actual project
configuration instead of relying on generic assumptions.

The generated guide now includes:

- The database provider and correct SQL parameter style for the project.
- The configured schema, query, migration, export, and generated-code paths.
- The complete schema and starter queries created by `flash init`.
- Migration, generation, Studio, export, seeding, branching, and raw SQL
  workflows.
- Query commands including `:one`, `:many`, `:exec`, and `:execresult`.
- Typed JSON with `-- @json`, required fields with `-- @required`, and Redis
  query caching with `-- @cache`.
- Multi-database configuration and `--db <name>` usage.
- Supported databases, drivers, connection strings, and generated-code rules.
- A clear warning not to edit generated files directly.

## Language-Specific Examples

The generated `FLASH.md` now detects which code generator was selected by
`flash init` and adds an example for that language.

Supported generated targets:

| Target | Generated guide coverage |
| --- | --- |
| Go | `Newq`, generated query methods, `database/sql`, and pgx guidance |
| JavaScript/TypeScript | `Newq`, database-client setup, JavaScript output, and TypeScript declarations |
| Python | `newq`, async and sync modes, and driver connection handling |
| Kotlin | Generated package path, `Queries.newq`, JDBC, Exposed, and R2DBC guidance |
| Java | Generated package path, `Queries.newq`, JDBC, jOOQ, and Hibernate guidance |
| Rust | Cargo dependencies, `sqlx` pools, async query methods, nullable values, and validation modes |

The enabled `[gen.*]` block in `flash.toml` remains the source of truth. The
guide tells agents to read that block before using or modifying generated code.

## Rust SQLx Validation Modes

The Rust section now explains both supported SQLx generation modes.

### Runtime-Checked Queries

Runtime checking is the default:

```toml
[gen.rust]
enabled = true
out = "src/flash_gen"
driver = "sqlx"
macros = false
```

Flash generates `sqlx::query`, `sqlx::query_as`, and `sqlx::query_scalar`
calls with `.bind(...)` parameters. A live database is not required while
running `flash gen`, `cargo check`, or `cargo build`.

SQL syntax, parameter compatibility, and row decoding are checked when the
generated async query method executes. Runtime failures are returned as
`Result<T, sqlx::Error>`.

```bash
flash gen
cargo build
DATABASE_URL="postgres://user:pass@localhost/database" cargo run
```

### Compile-Time Checked Macros

Compile-time validation is enabled explicitly:

```toml
[gen.rust]
enabled = true
out = "src/flash_gen"
driver = "sqlx"
macros = true
```

Flash then generates `sqlx::query!`, `sqlx::query_as!`, and
`sqlx::query_scalar!`. `DATABASE_URL` must point to a running database whose
schema matches the Flash schema whenever Cargo compiles the generated code.

```bash
flash migrate "sync schema"
flash apply
flash gen -f
DATABASE_URL="postgres://user:pass@localhost/database" cargo check
```

Changing `macros` invalidates the generation cache, so Flash regenerates the
Rust query files using the selected validation mode.

## Compatibility

This release does not require existing projects to use compile-time macros.
`macros = false` remains the default, and existing generated targets continue
to work without enabling the new guide behavior manually. Run `flash init` in
a new project to generate `FLASH.md`; existing projects can regenerate or copy
the guide when adopting the updated workflow.

## Tests

Template tests now verify that `FLASH.md`:

- Identifies every supported generated language correctly.
- Includes a language-specific usage example.
- Documents both Rust runtime and compile-time validation modes.
- Replaces all database and language template markers before writing the file.
