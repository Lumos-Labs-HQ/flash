package template

import (
	"fmt"
	"strings"
)

type DatabaseType string

const (
	SQLite     DatabaseType = "sqlite"
	PostgreSQL DatabaseType = "postgresql"
	MySQL      DatabaseType = "mysql"
	ClickHouse DatabaseType = "clickhouse"
	ScyllaDB   DatabaseType = "scylla"
)

type ProjectTemplate struct {
	DatabaseType    DatabaseType
	IsNodeProject   bool
	IsPythonProject bool
	IsKotlinProject bool
	IsJavaProject   bool
	IsRustProject   bool
	JavaPackage     string // auto-detected from pom.xml / build.gradle
	KotlinPackage   string // auto-detected from build.gradle.kts / pom.xml
}

type dbConfig struct {
	provider         string
	engine           string
	primaryKey       string
	autoIncrement    string
	textType         string
	timestampType    string
	timestampDefault string
	queryParam       string
	returnType       string
	envExample       string
}

var dbConfigs = map[DatabaseType]dbConfig{
	SQLite: {
		provider:         "sqlite",
		engine:           "sqlite",
		primaryKey:       "INTEGER PRIMARY KEY AUTOINCREMENT",
		autoIncrement:    "AUTOINCREMENT",
		textType:         "TEXT",
		timestampType:    "DATETIME",
		timestampDefault: "CURRENT_TIMESTAMP",
		queryParam:       "?",
		returnType:       ":one",
		envExample:       "sqlite://./data.sqlite",
	},
	MySQL: {
		provider:         "mysql",
		engine:           "mysql",
		primaryKey:       "INT AUTO_INCREMENT PRIMARY KEY",
		autoIncrement:    "AUTO_INCREMENT",
		textType:         "VARCHAR(255)",
		timestampType:    "TIMESTAMP",
		timestampDefault: "CURRENT_TIMESTAMP",
		queryParam:       "?",
		returnType:       ":execresult",
		envExample:       "mysql://username:password@localhost:3306/database_name",
	},
	PostgreSQL: {
		provider:         "postgresql",
		engine:           "postgresql",
		primaryKey:       "SERIAL PRIMARY KEY",
		autoIncrement:    "SERIAL",
		textType:         "VARCHAR(255)",
		timestampType:    "TIMESTAMP WITH TIME ZONE",
		timestampDefault: "NOW()",
		queryParam:       "$1",
		returnType:       ":one",
		envExample:       "postgres://username:password@localhost:5432/database_name",
	},
	ClickHouse: {
		provider:         "clickhouse",
		engine:           "clickhouse",
		primaryKey:       "UInt64",
		autoIncrement:    "",
		textType:         "String",
		timestampType:    "DateTime",
		timestampDefault: "now()",
		queryParam:       "?",
		returnType:       ":exec",
		envExample:       "clickhouse://username:password@localhost:9000/database_name",
	},
	ScyllaDB: {
		provider:         "scylla",
		engine:           "scylla",
		primaryKey:       "uuid PRIMARY KEY",
		autoIncrement:    "",
		textType:         "text",
		timestampType:    "timestamp",
		timestampDefault: "toTimestamp(now())",
		queryParam:       "?",
		returnType:       ":one",
		envExample:       "scylla://host:9042/keyspace_name",
	},
}

func NewProjectTemplate(dbType DatabaseType, isNodeProject bool, isPythonProject bool) *ProjectTemplate {
	return &ProjectTemplate{
		DatabaseType:    dbType,
		IsNodeProject:   isNodeProject,
		IsPythonProject: isPythonProject,
	}
}

func NewProjectTemplateExt(dbType DatabaseType, isNode, isPython, isKotlin, isJava, isRust bool) *ProjectTemplate {
	return &ProjectTemplate{
		DatabaseType:    dbType,
		IsNodeProject:   isNode,
		IsPythonProject: isPython,
		IsKotlinProject: isKotlin,
		IsJavaProject:   isJava,
		IsRustProject:   isRust,
	}
}

func (pt *ProjectTemplate) GetFlashORMConfig() string {
	cfg := dbConfigs[pt.DatabaseType]

	config := pt.getDriverHeaderComment() + "\n"
	config += "version = \"2\"\n"
	config += "schema_dir = \"db/schema\"\n"
	config += "queries = \"db/queries/\"\n"
	config += "migrations_path = \"db/migrations\"\n"
	config += "export_path = \"db/export\"\n\n"

	config += "[database]\n"
	config += fmt.Sprintf("provider = \"%s\"\n", cfg.provider)
	config += "url_env = \"DATABASE_URL\"\n"

	genSection := pt.getGenSection()
	if genSection != "" {
		config += "\n" + genSection
	}

	return config
}

func (pt *ProjectTemplate) GetSchema() string {
	switch pt.DatabaseType {
	case ScyllaDB:
		return scyllaSchema
	default:
		return pt.getRelationalSchema()
	}
}

const scyllaSchema = `-- === KEYSPACE ===
CREATE KEYSPACE myapp
WITH replication = {
    'class': 'SimpleStrategy',
    'replication_factor': 1
};

-- === TABLES ===

CREATE TABLE myapp.users (
    id          uuid PRIMARY KEY,
    username    text,
    email       text,
    full_name   text,
    is_active   boolean,
    tags        set<text>,
    metadata    map<text, text>,
    created_at  timestamp,
    updated_at  timestamp
);

CREATE INDEX myapp.users_email_idx ON myapp.users (email);
`

func (pt *ProjectTemplate) getRelationalSchema() string {
	cfg := dbConfigs[pt.DatabaseType]
	updateClause := ""
	if pt.DatabaseType == MySQL {
		updateClause = " ON UPDATE CURRENT_TIMESTAMP"
	}

	return fmt.Sprintf(`CREATE TABLE users (
    id %s,
    name %s NOT NULL,
    email %s UNIQUE NOT NULL,
    created_at %s NOT NULL DEFAULT %s,
    updated_at %s NOT NULL DEFAULT %s%s
);
`, cfg.primaryKey, cfg.textType, cfg.textType, cfg.timestampType,
		cfg.timestampDefault, cfg.timestampType, cfg.timestampDefault, updateClause)
}

func (pt *ProjectTemplate) GetQueries() string {
	if pt.DatabaseType == ScyllaDB {
		return scyllaQueries
	}
	cfg := dbConfigs[pt.DatabaseType]
	param2 := cfg.queryParam
	if pt.DatabaseType == PostgreSQL {
		param2 = "$2"
	}

	return fmt.Sprintf(`-- name: GetUser :one
SELECT id, name, email, created_at, updated_at FROM users
WHERE id = %s LIMIT 1;

-- name: CreateUser %s
INSERT INTO users (name, email)
VALUES (%s, %s)%s;
`, cfg.queryParam, cfg.returnType, cfg.queryParam, param2, pt.getReturningClause())
}

const scyllaQueries = `-- name: CreateUser :exec
INSERT INTO myapp.users (id, username, email, full_name, is_active, tags, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetUserByID :one
SELECT * FROM myapp.users WHERE id = ?;

-- name: GetUserByEmail :many
SELECT id, username, email, created_at FROM myapp.users WHERE email = ? ALLOW FILTERING;

-- name: UpdateUserProfile :exec
UPDATE myapp.users SET full_name = ?, updated_at = ? WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM myapp.users WHERE id = ?;
`

func (pt *ProjectTemplate) getReturningClause() string {
	if pt.DatabaseType == MySQL {
		return ""
	}
	return "\nRETURNING id, name, email, created_at, updated_at"
}

func (pt *ProjectTemplate) GetEnvTemplate() string {
	cfg := dbConfigs[pt.DatabaseType]
	return fmt.Sprintf("DATABASE_URL=%s\n", cfg.envExample)
}

func (pt *ProjectTemplate) GetDirectoryStructure() []string {
	return []string{"db/schema", "db/queries"}
}

// GetAgentGuide returns FLASH.md — a complete, self-contained reference an AI
// agent can read to understand and drive FlashORM in this project. The content
// is tailored to the database chosen at init time (parameter style, connection
// string, return-type convention) and embeds the exact schema/queries that were
// scaffolded, so the guide always matches the code on disk.
//
// The guide is authored as a raw string literal, which cannot contain backticks;
// the sentinel § is used wherever a backtick is needed and replaced at the end.
// DB-specific values are injected via @@TOKEN@@ placeholders.
func (pt *ProjectTemplate) GetAgentGuide() string {
	cfg := dbConfigs[pt.DatabaseType]

	ext := ".sql"
	schemaDir := "db/schema"
	if pt.DatabaseType == ScyllaDB {
		ext = ".cql"
	}

	param := cfg.queryParam

	guide := `# FLASH.md — FlashORM Agent Guide

> Auto-generated by §flash init§. This file is the single source of truth for
> using FlashORM in this project. It is written for AI coding agents: read it
> top to bottom and you can drive Flash correctly without guessing.
>
> **This project's database:** §@@PROVIDER@@§ · **query parameter style:** §@@PARAM@@§ · **schema files:** §@@EXT@@§

---

## 1. What FlashORM is

FlashORM is a **SQL-first, type-safe ORM and code generator**. You do not write
an ORM DSL — you write plain SQL, and Flash generates type-safe client code from
it for **Go, TypeScript, JavaScript, Python, Kotlin, Java, and Rust**.

Two inputs drive everything:

- §@@SCHEMADIR@@/§ — your table definitions (DDL, §@@EXT@@§ files).
- §db/queries/§ — your named queries, each tagged with a §-- name:§ comment.

§flash gen§ reads both and emits type-safe functions/structs. The §.sql§ files
are the source of truth; generated code is disposable and should never be edited
by hand.

---

## 2. Project layout

§§§
flash.toml            # project config: provider, codegen targets, cache
@@SCHEMADIR@@/            # schema DDL — CREATE TABLE ... (edit these)
  schema@@EXT@@
db/queries/           # named SQL queries — one -- name: per query (edit these)
  users@@EXT@@
db/migrations/        # generated migration files (created by 'flash migrate'/'apply')
.env                  # DATABASE_URL and other secrets (never commit)
flash_gen/            # generated type-safe code (created by 'flash gen'; do not edit)
§§§

---

## 3. Everyday workflow

§§§
1. flash init --@@PROVIDER@@        # scaffold (already done — this file proves it)
2. edit @@SCHEMADIR@@/schema@@EXT@@   # define/adjust tables
3. flash migrate "add users"      # create a migration from schema changes
4. flash apply                    # apply pending migrations to the database
5. edit db/queries/*@@EXT@@         # write named queries
6. flash gen                      # generate type-safe code into flash_gen/
7. flash studio                   # (optional) open the visual DB editor
§§§

Fast iteration without migrations: §flash apply§ creates tables directly from the
schema for a fresh dev DB; use §flash migrate§ + §flash apply§ for tracked changes.
Re-run §flash gen§ after every change to a §@@EXT@@§ file.

---

## 4. CLI command reference

| Command | What it does | Key flags |
|---|---|---|
| §flash init [name]§ | Scaffold a new project | §--@@PROVIDER@@§, §--sqlite§, §--mysql§, §--clickhouse§, §--scylla§ |
| §flash migrate <name>§ | Create a migration from schema changes | |
| §flash apply§ | Apply pending migrations | §--force§ (prod), §--db <name>§ |
| §flash down [id]§ | Roll back migration(s) | |
| §flash status§ | Show applied/pending migrations | §--db <name>§ |
| §flash pull§ | Reverse-engineer schema from an existing DB | |
| §flash reset§ | Drop & recreate the database (dev) | §--force§ |
| §flash raw <sql-or-file>§ | Execute raw SQL or a §.sql§ file | |
| §flash gen§ | Generate type-safe code from SQL | §-f/--force§, §--db <name>§ |
| §flash export§ | Export tables (JSON/CSV/SQLite) | §--csv§, §--sqlite§ |
| §flash seed [tables...]§ | Insert fake test data | §--count N§, §--truncate§ |
| §flash branch [name]§ | List/create/delete schema branches | |
| §flash checkout <branch>§ | Switch schema branches | |
| §flash studio [URL]§ | Visual editor (SQL, MongoDB, Redis) | §--db <name>§ |
| §flash dblist§ | List configured databases | |
| §flash issues§ | File a bug/feature report to the Flash repo | see §12§ |
| §flash update§ | Update plugins and the flash binary | |

Every data command accepts §--db <name>§ for multi-database configs (see §5§).

---

## 5. flash.toml reference

Every field Flash reads (defaults shown in parentheses):

§§§toml
version = "2"                    # config schema version
schema_dir = "@@SCHEMADIR@@"        # where DDL lives ("db/schema")
queries = "db/queries/"          # where named queries live ("db/queries/")
migrations_path = "db/migrations"
export_path = "db/export"
env_path = ""                    # extra .env file to load, if any
json_path = ""                   # dir with .json type defs for @json columns

[database]
provider = "@@PROVIDER@@"           # @@PROVIDERLIST@@
url_env  = "DATABASE_URL"        # env var that holds the connection string

# Code generation — enable one or more targets. 'out' defaults to "flash_gen".
[gen.go]
enabled = true
# driver = "pgx"                 # or "database/sql" (see §10§ for options)
# out = "flash_gen"

# [gen.js]     enabled=true  out="flash_gen"  driver="pg"
# [gen.python] enabled=true  out="flash_gen"  async=true  driver="asyncpg"
# [gen.kotlin] enabled=true  out="..."  package="com.example.db"  driver="jdbc"
# [gen.java]   enabled=true  out="..."  package="com.example.db"  driver="jdbc"
# [gen.rust]   enabled=true  out="src/flash_gen"  driver="sqlx"  macros=false

# Query caching via @cache annotations (see §9§). Off unless enabled.
[cache]
enabled = false
redis_url_env = "REDIS_URL"      # env var with the Redis URL
default_ttl   = "5m"             # applied when @cache omits ttl
prefix        = "flash"          # cache-key prefix
§§§

**Multi-database:** instead of a single §[database]§, declare several with
§[[databases]]§ blocks (each with its own §name§, §provider§, §url_env§, and
§[databases.gen.*]§). Then target one with §--db <name>§, or mark one
§default = true§. §flash dblist§ lists them.

§§§toml
[[databases]]
name = "main"
provider = "@@PROVIDER@@"
url_env = "MAIN_DATABASE_URL"
default = true

[[databases]]
name = "analytics"
provider = "clickhouse"
url_env = "ANALYTICS_URL"
§§§

---

## 6. Database configuration & environment

The connection string is **never** written in §flash.toml§. Flash reads the env
var named by §url_env§ (default §DATABASE_URL§). §.env§ is auto-loaded (also
§.env.local§, and §.env.<name>§ via §--env <name>§).

This project's §.env§:

§§§
DATABASE_URL=@@ENV@@
§§§

Connection string formats by provider:

| Provider | §url_env§ value example |
|---|---|
| postgresql | §postgres://user:pass@localhost:5432/dbname§ |
| mysql | §mysql://user:pass@localhost:3306/dbname§ |
| sqlite | §sqlite://./data.sqlite§ |
| clickhouse | §clickhouse://user:pass@localhost:9000/dbname§ |
| scylla / cassandra | §scylla://host:9042/keyspace§ |

To point Flash at a different database, change the env var — not the config.

---

## 7. Writing schema (§@@SCHEMADIR@@/schema@@EXT@@§)

Plain DDL. This is what was scaffolded for §@@PROVIDER@@§:

§§§sql
@@SCHEMA@@
§§§

Add tables/columns here, then §flash migrate "<msg>"§ + §flash apply§. Flash
auto-detects column/table renames and preserves data where it can.

---

## 8. Writing queries — the §-- name:§ grammar

Each query is introduced by a comment of the exact form:

§§§
-- name: <QueryName> <:cmd>
<SQL>;
§§§

- §<QueryName>§ becomes the generated function name (PascalCase recommended).
- §<:cmd>§ is the **result mode** and MUST be one of:

| §:cmd§ | Returns | Use for |
|---|---|---|
| §:one§ | exactly one row (or not-found) | single-row SELECT / INSERT ... RETURNING |
| §:many§ | a slice/list of rows | multi-row SELECT |
| §:exec§ | nothing (just runs) | INSERT/UPDATE/DELETE without a return |
| §:execresult§ | rows-affected / last-insert-id | mutations needing a result (MySQL inserts) |

**Parameters use §@@PARAM@@§-style placeholders for @@PROVIDER@@.** Postgres uses
§$1, $2, ...§; MySQL/SQLite/ClickHouse/Scylla use §?§.

This project's scaffolded queries:

§§§sql
@@QUERIES@@
§§§

Notes:
- Prefer §RETURNING§ (Postgres/SQLite) so an §INSERT§ can be §:one§.
- MySQL has no §RETURNING§: inserts are §:execresult§ (last-insert-id).
- Put each query in any §@@EXT@@§ file under §db/queries/§; filenames are free.

---

## 9. Annotations

Annotations are §--§ comment lines placed **directly above** (or right after) a
§-- name:§ line, before the SQL body.

### §-- @required: col1, col2§  (or §*§)
Marks result columns as non-nullable in generated types even when the schema
allows NULL (primarily for CQL/ScyllaDB, where non-PK columns are nullable by
default). §-- @required: *§ marks all.

### §-- @json as <name>§
Declares a typed structure for a JSON/JSONB column so the generator emits a
typed class/struct instead of a raw string. Type defs can also live in §.json§
files pointed to by §json_path§.

### §-- @cache { ... }§  (see §10§)
Caches a read query's result in Redis.

---

## 10. Query caching (Redis)

Turn it on in §flash.toml§:

§§§toml
[cache]
enabled = true
redis_url_env = "REDIS_URL"
default_ttl = "5m"
prefix = "flash"
§§§

Annotate a read query. **Bare** §-- @cache§ uses all defaults; the JSON form
overrides individual fields:

§§§sql
-- name: GetUserByEmail :one
-- @cache {"ttl": "30s", "name": "UserByEmail", "tags": ["users"], "dep": ["UpdateUser", "DeleteUser"]}
SELECT id, name, email FROM users WHERE email = @@PARAM@@ LIMIT 1;
§§§

@cache fields:

| Field | Meaning | Default |
|---|---|---|
| §ttl§ | time-to-live | §default_ttl§ from config |
| §name§ | cache accessor name | §<QueryName>Cache§ |
| §tags§ | labels for bulk purge | none |
| §dep§ | query names that invalidate this cache when they run | none |

**TTL syntax** (parsed by Flash): a bare number is seconds (§90§), or a unit
string — §30s§, §5m§, §1h§, §2d§ — including decimals (§1.5h§) and compound
durations (§1h30m§). Unknown units or negatives are rejected.

**Cache key format:** §{prefix}:{name}:{param1}:{param2}...§ — the key is built
from the query's parameters.

**Auto-invalidation:** listing a mutation in §dep§ makes it purge this cache when
it runs. If the cache key is a single parameter that the mutation also takes,
Flash deletes that exact entry; otherwise it purges the whole set by prefix.

The generator produces cache accessors (get / delete / purge) alongside the
query function. The cache is read-through: a miss runs the SQL and populates
Redis; §dep§ mutations keep it consistent.

---

## 11. Code generation & drivers

§flash gen§ auto-detects the project (§package.json§ → JS, §pyproject.toml§ →
Python, §Cargo.toml§ → Rust, Gradle/Maven → Kotlin/Java, else Go) and writes to
the enabled §[gen.*]§ targets. Force a clean rebuild with §flash gen -f§.

Driver options for **@@PROVIDER@@**:

§§§
@@DRIVERS@@
§§§

Set the driver with §driver = "<name>"§ inside the relevant §[gen.*]§ block.

---

## 12. FlashORM Studio

§flash studio§ launches a local visual editor. It supports SQL databases, and
also MongoDB and Redis via a connection URL:

§§§
flash studio                                  # uses DATABASE_URL
flash studio "postgres://user:pass@host:5432/db"
flash studio "mongodb://localhost:27017/mydb"
flash studio "redis://localhost:6379"
§§§

---

## 13. Reporting bugs — §flash issues§

If you (an agent) hit a bug, a broken generation, or unexpected behavior while
using Flash, file a structured report so it can be fixed:

§§§
flash issues \
  -k bug \
  -t "gen: <short, specific summary>" \
  -b "<what you were doing and what FlashORM did>" \
  --repro "1. ...  2. ...  3. ..." \
  --expected "<what should happen>" \
  --actual "<what happened, incl. exact error text>"
§§§

- §-k§ kind: §bug§ (default), §feature§, §question§, §docs§.
- §--print§ composes the report and prints it **without submitting** — use this
  first to review it.
- §-y§ submits non-interactively (skips the confirm prompt).
- Version, OS, Go version, and DB provider are attached automatically.

The command **validates** the report (a real title, a substantive body, and — for
bugs — reproduction steps or expected/actual) and refuses to file low-effort
issues, so only actionable reports reach the repo.

---

## 14. Golden rules for an agent

1. **Edit §@@EXT@@§ files, never §flash_gen/§.** Generated code is overwritten.
2. **Re-run §flash gen§** after any schema or query change.
3. **Use §@@PARAM@@§ placeholders** for this project's provider (@@PROVIDER@@).
4. **Pick the right §:cmd§** — §:one§/§:many§ for reads, §:exec§/§:execresult§ for writes.
5. **Secrets go in §.env§** via §url_env§; never hard-code a connection string.
6. **Migrations:** §flash migrate "<msg>"§ then §flash apply§. Don't hand-edit applied migrations.
7. **Caching is opt-in:** enable §[cache]§, then annotate reads with §-- @cache§ and wire §dep§ mutations.
8. **Something broke? Run §flash issues§** with real reproduction steps.
`

	replacements := []struct{ from, to string }{
		{"@@PROVIDER@@", cfg.provider},
		{"@@PARAM@@", param},
		{"@@ENV@@", cfg.envExample},
		{"@@EXT@@", ext},
		{"@@SCHEMADIR@@", schemaDir},
		{"@@PROVIDERLIST@@", "postgresql | mysql | sqlite | clickhouse | scylla"},
		{"@@SCHEMA@@", strings.TrimRight(pt.GetSchema(), "\n")},
		{"@@QUERIES@@", strings.TrimRight(pt.GetQueries(), "\n")},
		{"@@DRIVERS@@", pt.getDriverHeaderComment()},
	}
	for _, r := range replacements {
		guide = strings.ReplaceAll(guide, r.from, r.to)
	}
	guide = strings.ReplaceAll(guide, "§", "`")
	return guide
}

func ValidateDatabaseType(dbType string) DatabaseType {
	types := map[string]DatabaseType{
		"sqlite":     SQLite,
		"mysql":      MySQL,
		"postgresql": PostgreSQL,
		"postgres":   PostgreSQL,
		"clickhouse": ClickHouse,
		"scylla":     ScyllaDB,
		"scylladb":   ScyllaDB,
		"cassandra":  ScyllaDB,
	}

	if dt, exists := types[dbType]; exists {
		return dt
	}
	return PostgreSQL
}

func (pt *ProjectTemplate) getGenSection() string {
	if pt.IsNodeProject {
		return `[gen.js]
enabled = true
out = "flash_gen"`
	}
	if pt.IsPythonProject {
		return `[gen.python]
enabled = true
out = "flash_gen"
async = true`
	}
	if pt.IsRustProject {
		return `[gen.rust]
enabled = true
out = "src/flash_gen"
driver = "sqlx"`
	}
	if pt.IsKotlinProject {
		pkg := pt.KotlinPackage
		if pkg == "" {
			pkg = projectDirName(".")
		}
		outPath := "src/main/kotlin/" + strings.ReplaceAll(pkg, ".", "/") + "/flashgen"
		return fmt.Sprintf(`[gen.kotlin]
enabled = true
out = "%s"
package = "%s.flashgen"`, outPath, pkg)
	}
	if pt.IsJavaProject {
		pkg := pt.JavaPackage
		if pkg == "" {
			pkg = projectDirName(".")
		}
		outPath := "src/main/java/" + strings.ReplaceAll(pkg, ".", "/") + "/flashgen"
		return fmt.Sprintf(`[gen.java]
enabled = true
out = "%s"
package = "%s.flashgen"`, outPath, pkg)
	}
	return `[gen.go]
enabled = true`
}

func (pt *ProjectTemplate) getDriverHeaderComment() string {
	switch pt.DatabaseType {
	case PostgreSQL:
		return `# FlashORM — PostgreSQL Drivers
#   Go:     "pgx" (native) | "database/sql" (lib/pq)
#   JS:     "pg" (node-postgres) | "postgres" (porsager/postgres)
#   Python: "psycopg3" | "asyncpg"
#   Kotlin: "jdbc" (default) | "exposed" | "r2dbc"
#   Java:   "jdbc" (default) | "jooq" | "hibernate"
#   Rust:   "sqlx" (default)
# Add driver = "<name>" inside the [gen.*] block below.`
	case MySQL:
		return `# FlashORM — MySQL Drivers
#   Go:     "database/sql" (go-sql-driver/mysql)
#   JS:     "mysql2" | "serverless-mysql"
#   Python: "pymysql" (sync) | "asyncmy" (async)
#   Kotlin: "jdbc" (default) | "exposed" | "r2dbc"
#   Java:   "jdbc" (default) | "jooq" | "hibernate"
#   Rust:   "sqlx" (default)
# Add driver = "<name>" inside the [gen.*] block below.`
	case SQLite:
		return `# FlashORM — SQLite Drivers
#   Go:     "database/sql" (mattn/go-sqlite3, modernc.org/sqlite)
#   JS:     "better-sqlite3" | "bun:sqlite"
#   Python: "sqlite3" (sync) | "aiosqlite" (async)
#   Kotlin: "jdbc" (default) | "exposed"
#   Java:   "jdbc" (default)
#   Rust:   "sqlx" (default)
# Add driver = "<name>" inside the [gen.*] block below.`
	case ClickHouse:
		return `# FlashORM — ClickHouse Drivers
#   Go:     "clickhouse-go/v2"
#   JS:     "@clickhouse/client"
#   Python: "clickhouse-driver" (sync) | "asynch" (async)
#   Kotlin: "jdbc" (default)
#   Java:   "jdbc" (default) | "jooq"
#   Rust:   "sqlx" (default)
# Add driver = "<name>" inside the [gen.*] block below.`
	case ScyllaDB:
		return `# FlashORM — ScyllaDB/Cassandra Drivers
#   Go:     "apache/cassandra-gocql-driver/v2" (default) | "gocql"
#   JS:     "cassandra-driver"
#   Python: "scylla-driver" (sync) | "cassandra-driver" (async)
#   Kotlin: uses DataStax Java Driver (CqlSession) — no driver key needed
#   Java:   uses DataStax Java Driver (CqlSession) — no driver key needed
#   Rust:   "sqlx" (default) — ScyllaDB not yet supported for Rust
# Add driver = "gocql" inside the [gen.go] block to use gocql/gocql instead.`
	default:
		return `# FlashORM — See docs for available drivers per database.`
	}
}
