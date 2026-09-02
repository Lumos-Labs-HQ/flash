package langs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
)

// This suite runs the full Generate() pipeline of every non-Go language
// generator against a shared schema+queries fixture and asserts the
// language-specific output files exist and contain the expected symbols.
// It is intentionally light on exact-output matching (unit tests in each
// generator package pin details) — here we verify end-to-end plumbing:
// files written, query methods emitted, params typed, cache layer wired.

const langsSchema = `
CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	email VARCHAR(255) NOT NULL,
	name TEXT,
	age INTEGER,
	is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE posts (
	id SERIAL PRIMARY KEY,
	user_id INTEGER NOT NULL REFERENCES users(id),
	title TEXT NOT NULL,
	view_count INTEGER DEFAULT 0
);
`

const langsQueries = `
-- name: GetUser :one
SELECT id, email, name FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, email FROM users ORDER BY id;

-- name: CreateUser :exec
INSERT INTO users (email, name, age) VALUES ($1, $2, $3);

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: GetPost :one
SELECT p.id, p.title, u.email AS author_email
FROM posts p JOIN users u ON u.id = p.user_id
WHERE p.id = $1;
`

// langFixture prepares schema + queries dirs under a temp root and returns
// (root, schemaDir, queriesDir).
func langFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	schemaDir := filepath.Join(root, "db", "schema")
	queriesDir := filepath.Join(root, "db", "queries")
	for _, d := range []string{schemaDir, queriesDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(schemaDir, "schema.sql"), []byte(langsSchema), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(queriesDir, "queries.sql"), []byte(langsQueries), 0644); err != nil {
		t.Fatal(err)
	}
	return root, schemaDir, queriesDir
}

func readOut(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func mustContainAll(t *testing.T, name, src string, frags ...string) {
	t.Helper()
	for _, f := range frags {
		if !strings.Contains(src, f) {
			t.Errorf("%s missing %q", name, f)
		}
	}
}

func mustNotContain(t *testing.T, name, src string, frags ...string) {
	t.Helper()
	for _, f := range frags {
		if strings.Contains(src, f) {
			t.Errorf("%s must not contain %q", name, f)
		}
	}
}

func baseConfig(root, schemaDir, queriesDir string) *config.Config {
	return &config.Config{
		SchemaDir:      schemaDir,
		SchemaPath:     filepath.Join(schemaDir, "schema.sql"),
		Queries:        queriesDir,
		MigrationsPath: filepath.Join(root, "db", "migrations"),
		Database:       config.Database{Provider: "postgresql"},
	}
}

// ── JavaScript / TypeScript ───────────────────────────────────────────────────

func TestJSGen_FullPipeline(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	out := filepath.Join(root, "flash_gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Gen.JS = config.JSGen{Enabled: true, Out: out, Driver: "pg"}

	if err := jsgenNew(cfg).Generate(); err != nil {
		t.Fatalf("js Generate: %v", err)
	}

	index := readOut(t, out, "queries.js")
	// Query methods use camelCase in a per-source class
	mustContainAll(t, "queries.js", index,
		"class Queries", "getUser", "listUsers", "createUser", "countUsers", "getPost",
	)
	// postgres driver: $1 params preserved
	if !strings.Contains(index, "$1") {
		t.Error("queries.js: pg driver should keep $1 placeholders")
	}
	// statement caching infra
	mustContainAll(t, "queries.js", index, "_stmts")

	dts := readOut(t, out, "index.d.ts")
	mustContainAll(t, "index.d.ts", dts, "interface Users", "getPost", "GetPostResult", "CreateUserParams")

	// migrations shim exists
	if _, err := os.Stat(filepath.Join(out, "migrations.js")); err != nil {
		t.Errorf("migrations.js missing: %v", err)
	}

	// No cache files without cache enabled
	mustNotExist(t, out, "cache.js")
	mustNotExist(t, out, "cache_accessors.js")
}

func TestJSGen_CacheLayer(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	if err := os.WriteFile(filepath.Join(queriesDir, "cached.sql"), []byte(`
-- @cache {"ttl": "30s", "name": "UserCache", "tags": ["users"], "dep": ["CreateUser"]}
-- name: GetUserCached :one
SELECT id, email FROM users WHERE id = $1;
`), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "flash_gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Gen.JS = config.JSGen{Enabled: true, Out: out}
	cfg.Cache = config.CacheConfig{Enabled: true, DefaultTTL: "1m", Prefix: "flash"}

	if err := jsgenNew(cfg).Generate(); err != nil {
		t.Fatalf("js Generate: %v", err)
	}
	accessors := readOut(t, out, "cache_accessors.js")
	mustContainAll(t, "cache_accessors.js", accessors, "UserCache")
	cached := readOut(t, out, "cached_queries.js")
	mustContainAll(t, "cached_queries.js", cached, "GetUserCachedCached", "UserCache:")
}

func TestJSGen_MySQLDriverUsesQuestionMarks(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	out := filepath.Join(root, "flash_gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Database.Provider = "mysql"
	cfg.Gen.JS = config.JSGen{Enabled: true, Out: out, Driver: "mysql2"}

	if err := jsgenNew(cfg).Generate(); err != nil {
		t.Fatalf("js Generate: %v", err)
	}
	index := readOut(t, out, "queries.js")
	if strings.Contains(index, "$1") {
		t.Error("mysql2 driver must not keep $1 placeholders")
	}
	if !strings.Contains(index, "?") {
		t.Error("mysql2 driver should use ? placeholders")
	}
}

// ── Python ────────────────────────────────────────────────────────────────────

func TestPyGen_FullPipeline_Async(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	out := filepath.Join(root, "flash_gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Gen.Python = config.PythonGen{Enabled: true, Out: out, Async: true, Driver: "asyncpg"}

	if err := pygenNew(cfg).Generate(); err != nil {
		t.Fatalf("python Generate: %v", err)
	}

	models := readOut(t, out, "models.py")
	mustContainAll(t, "models.py", models, "class Users", "class Posts")

	// Per-source file carries the Queries class with async snake_case methods.
	qsrc := readOut(t, out, "queries.py")
	mustContainAll(t, "queries.py", qsrc, "class Queries", "async def get_user", "async def list_users", "GetPostRow", "CreateUserParams")

	db := readOut(t, out, "database.py")
	mustContainAll(t, "database.py", db, "def newq")

	init := readOut(t, out, "__init__.py")
	mustContainAll(t, "__init__.py", init, "from .database import newq", "from .models import")

	stub := readOut(t, out, "database.pyi")
	mustContainAll(t, "database.pyi", stub, "get_user", "Users")

	mustNotExist(t, out, "cache.py")
}

func TestPyGen_SyncMode(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	out := filepath.Join(root, "flash_gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Gen.Python = config.PythonGen{Enabled: true, Out: out, Async: false, Driver: "psycopg3"}

	if err := pygenNew(cfg).Generate(); err != nil {
		t.Fatalf("python Generate: %v", err)
	}
	qsrc := readOut(t, out, "queries.py")
	if strings.Contains(qsrc, "async def") {
		t.Error("sync mode must not emit async def")
	}
	mustContainAll(t, "queries.py", qsrc, "def get_user")
}

func TestPyGen_CacheLayer(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	if err := os.WriteFile(filepath.Join(queriesDir, "cached.sql"), []byte(`
-- @cache {"ttl": "30s", "name": "UserCache"}
-- name: GetUserCached :one
SELECT id, email FROM users WHERE id = $1;
`), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "flash_gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Gen.Python = config.PythonGen{Enabled: true, Out: out, Async: true}
	cfg.Cache = config.CacheConfig{Enabled: true, DefaultTTL: "1m"}

	if err := pygenNew(cfg).Generate(); err != nil {
		t.Fatalf("python Generate: %v", err)
	}
	mustContainAll(t, "cache_accessors.py", readOut(t, out, "cache_accessors.py"), "class UserCache")
	mustContainAll(t, "cached_queries.py", readOut(t, out, "cached_queries.py"), "getusercached_cached")
}

// ── Kotlin ────────────────────────────────────────────────────────────────────

func TestKotlinGen_FullPipeline(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	out := filepath.Join(root, "src", "gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Gen.Kotlin = config.KotlinGen{Enabled: true, Out: out, Package: "com.example.flashgen"}

	if err := kotlingenNew(cfg).Generate(); err != nil {
		t.Fatalf("kotlin Generate: %v", err)
	}

	models := readOut(t, out, "Models.kt")
	mustContainAll(t, "Models.kt", models, "package com.example.flashgen", "data class Users", "data class Posts")

	// Per-source file (PascalCase of source stem) carries the query class.
	queries := readOut(t, out, "Queries.kt")
	mustContainAll(t, "Queries.kt", queries, "class QueriesQueries", "fun getUser", "fun listUsers", "fun createUser")

	// FlashMigrations.kt shim
	if _, err := os.Stat(filepath.Join(out, "FlashMigrations.kt")); err != nil {
		t.Errorf("FlashMigrations.kt missing: %v", err)
	}
}

func TestKotlinGen_JoinRowAndParams(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	out := filepath.Join(root, "src", "gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Gen.Kotlin = config.KotlinGen{Enabled: true, Out: out, Package: "com.example.flashgen"}

	if err := kotlingenNew(cfg).Generate(); err != nil {
		t.Fatalf("kotlin Generate: %v", err)
	}
	src := readOut(t, out, "Queries.kt")
	mustContainAll(t, "Queries.kt", src, "GetPostRow", "CreateUserParams")
	// JDBC uses ? placeholders even for postgres provider
	if strings.Contains(src, "$1") {
		t.Error("Kotlin JDBC path must use ? placeholders")
	}
}

func TestKotlinGen_CacheLayer(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	if err := os.WriteFile(filepath.Join(queriesDir, "cached.sql"), []byte(`
-- @cache {"ttl": "30s", "name": "UserCache"}
-- name: GetUserCached :one
SELECT id, email FROM users WHERE id = $1;
`), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "src", "gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Gen.Kotlin = config.KotlinGen{Enabled: true, Out: out, Package: "com.example.flashgen"}
	cfg.Cache = config.CacheConfig{Enabled: true, DefaultTTL: "1m"}

	if err := kotlingenNew(cfg).Generate(); err != nil {
		t.Fatalf("kotlin Generate: %v", err)
	}
	mustContainAll(t, "CacheAccessors.kt", readOut(t, out, "CacheAccessors.kt"), "object UserCache")
	mustContainAll(t, "CachedQueries.kt", readOut(t, out, "CachedQueries.kt"), "GetUserCachedCached")
}

// ── Java ──────────────────────────────────────────────────────────────────────

func TestJavaGen_FullPipeline(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	out := filepath.Join(root, "src", "gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Gen.Java = config.JavaGen{Enabled: true, Out: out, Package: "com.example.flashgen"}

	if err := javagenNew(cfg).Generate(); err != nil {
		t.Fatalf("java Generate: %v", err)
	}

	models := readOut(t, out, "Users.java")
	mustContainAll(t, "Users.java", models, "package com.example.flashgen", "public record Users")

	queriesProxy := readOut(t, out, "Queries.java")
	mustContainAll(t, "Queries.java", queriesProxy, "public class Queries")

	// Per-query-file class file (one public class per file)
	perFile := readOut(t, out, "QueriesQueries.java")
	mustContainAll(t, "QueriesQueries.java", perFile, "getUser", "createUser")

	if _, err := os.Stat(filepath.Join(out, "FlashMigrations.java")); err != nil {
		t.Errorf("FlashMigrations.java missing: %v", err)
	}
}

func TestJavaGen_RowAndParamsClasses(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	out := filepath.Join(root, "src", "gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Gen.Java = config.JavaGen{Enabled: true, Out: out, Package: "com.example.flashgen"}

	if err := javagenNew(cfg).Generate(); err != nil {
		t.Fatalf("java Generate: %v", err)
	}
	src := readOut(t, out, "QueriesQueries.java")
	mustContainAll(t, "QueriesQueries.java", src, "GetPostRow", "CreateUserParams")
}

func TestJavaGen_CacheLayer(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	if err := os.WriteFile(filepath.Join(queriesDir, "cached.sql"), []byte(`
-- @cache {"ttl": "30s", "name": "UserCache"}
-- name: GetUserCached :one
SELECT id, email FROM users WHERE id = $1;
`), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "src", "gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Gen.Java = config.JavaGen{Enabled: true, Out: out, Package: "com.example.flashgen"}
	cfg.Cache = config.CacheConfig{Enabled: true, DefaultTTL: "1m"}

	if err := javagenNew(cfg).Generate(); err != nil {
		t.Fatalf("java Generate: %v", err)
	}
	mustContainAll(t, "CacheAccessors.java", readOut(t, out, "CacheAccessors.java"), "class UserCache")
	mustContainAll(t, "CachedQueries.java", readOut(t, out, "CachedQueries.java"), "GetUserCachedCached")
}

// ── Rust ──────────────────────────────────────────────────────────────────────

func TestRustGen_FullPipeline(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	out := filepath.Join(root, "src", "flash_gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Gen.Rust = config.RustGen{Enabled: true, Out: out, Driver: "sqlx"}

	if err := rustgenNew(cfg).Generate(); err != nil {
		t.Fatalf("rust Generate: %v", err)
	}

	models := readOut(t, out, "models.rs")
	mustContainAll(t, "models.rs", models, "pub struct Users", "pub struct Posts", "FromRow")

	db := readOut(t, out, "db.rs")
	mustContainAll(t, "db.rs", db, "pub struct Queries")

	modrs := readOut(t, out, "mod.rs")
	mustContainAll(t, "mod.rs", modrs, "pub mod models", "pub mod db")

	perFile := readOut(t, out, "queries.rs")
	mustContainAll(t, "queries.rs", perFile, "get_user", "create_user", "get_post")

	mustNotExist(t, out, "cache.rs")
}

func TestRustGen_CacheLayer(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	if err := os.WriteFile(filepath.Join(queriesDir, "cached.sql"), []byte(`
-- @cache {"ttl": "30s", "name": "UserCache"}
-- name: GetUserCached :one
SELECT id, email FROM users WHERE id = $1;
`), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "src", "flash_gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Gen.Rust = config.RustGen{Enabled: true, Out: out, Driver: "sqlx"}
	cfg.Cache = config.CacheConfig{Enabled: true, DefaultTTL: "1m"}

	if err := rustgenNew(cfg).Generate(); err != nil {
		t.Fatalf("rust Generate: %v", err)
	}
	mustContainAll(t, "cache_accessors.rs", readOut(t, out, "cache_accessors.rs"), "UserCache")
	mustContainAll(t, "cached_queries.rs", readOut(t, out, "cached_queries.rs"), "get_user_cached")
}

func TestRustGen_SQLiteQuestionMarks(t *testing.T) {
	root, schemaDir, queriesDir := langFixture(t)
	out := filepath.Join(root, "src", "flash_gen")
	cfg := baseConfig(root, schemaDir, queriesDir)
	cfg.Database.Provider = "sqlite"
	cfg.Gen.Rust = config.RustGen{Enabled: true, Out: out, Driver: "sqlx"}

	if err := rustgenNew(cfg).Generate(); err != nil {
		t.Fatalf("rust Generate: %v", err)
	}
	src := readOut(t, out, "queries.rs")
	if strings.Contains(src, "$1") {
		t.Error("sqlite provider must not keep $1 placeholders in Rust output")
	}
}

// ── Incremental generation across ALL languages ──────────────────────────────

// Every language generator must delete the per-source output file when the
// query source file is deleted (the purge behavior gogen already pins).
func TestAllLangs_DeletedQueryFilePurgesOutput(t *testing.T) {
	type langSetup struct {
		name      string
		generate  func(cfg *config.Config) error
		outOf     func(cfg *config.Config) string
		perSource string // output file produced for queries.sql
	}

	langs := []langSetup{
		{
			name:     "js",
			generate: func(cfg *config.Config) error { return jsgenNew(cfg).Generate() },
			outOf:    func(cfg *config.Config) string { return cfg.Gen.JS.Out },
		},
		{
			name:     "python",
			generate: func(cfg *config.Config) error { return pygenNew(cfg).Generate() },
			outOf:    func(cfg *config.Config) string { return cfg.Gen.Python.Out },
		},
		{
			name:     "kotlin",
			generate: func(cfg *config.Config) error { return kotlingenNew(cfg).Generate() },
			outOf:    func(cfg *config.Config) string { return cfg.Gen.Kotlin.Out },
		},
		{
			name:     "java",
			generate: func(cfg *config.Config) error { return javagenNew(cfg).Generate() },
			outOf:    func(cfg *config.Config) string { return cfg.Gen.Java.Out },
		},
		{
			name:     "rust",
			generate: func(cfg *config.Config) error { return rustgenNew(cfg).Generate() },
			outOf:    func(cfg *config.Config) string { return cfg.Gen.Rust.Out },
		},
	}

	for _, lang := range langs {
		t.Run(lang.name, func(t *testing.T) {
			root, schemaDir, queriesDir := langFixture(t)
			// Use a dedicated source file so we can delete only it.
			if err := os.WriteFile(filepath.Join(queriesDir, "extra.sql"), []byte("-- name: Extra :one\nSELECT id FROM users WHERE id = $1;\n"), 0644); err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(root, "gen_"+lang.name)
			cfg := baseConfig(root, schemaDir, queriesDir)
			switch lang.name {
			case "js":
				cfg.Gen.JS = config.JSGen{Enabled: true, Out: out}
			case "python":
				cfg.Gen.Python = config.PythonGen{Enabled: true, Out: out, Async: true}
			case "kotlin":
				cfg.Gen.Kotlin = config.KotlinGen{Enabled: true, Out: out, Package: "com.example"}
			case "java":
				cfg.Gen.Java = config.JavaGen{Enabled: true, Out: out, Package: "com.example"}
			case "rust":
				cfg.Gen.Rust = config.RustGen{Enabled: true, Out: out}
			}

			if err := lang.generate(cfg); err != nil {
				t.Fatalf("first Generate: %v", err)
			}
			// Discover which output file(s) track the extra source. Output
			// naming differs per language: extra.js / extra.py / Extra.kt /
			// ExtraQueries.java / extra.rs.
			var before []string
			entries, _ := os.ReadDir(out)
			for _, e := range entries {
				base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
				stem := strings.ToLower(base)
				// Kotlin appends "Queries"; Java appends "Queries" after PascalCase.
				stem = strings.TrimSuffix(stem, "queries")
				if stem == "extra" {
					before = append(before, e.Name())
				}
			}
			if len(before) == 0 {
				t.Fatalf("no output file for extra.sql after first run (out=%s)", out)
			}

			// Delete the source and regenerate.
			if err := os.Remove(filepath.Join(queriesDir, "extra.sql")); err != nil {
				t.Fatal(err)
			}
			if err := lang.generate(cfg); err != nil {
				t.Fatalf("second Generate: %v", err)
			}

			entries, _ = os.ReadDir(out)
			perSourceStem := func(name string) string {
				base := strings.TrimSuffix(name, filepath.Ext(name))
				stem := strings.ToLower(base)
				// Kotlin file: Queries.kt (class QueriesQueries); Java file:
				// QueriesQueries.java. Strip the trailing "Queries" only when
				// the base is longer than the suffix.
				if len(base) > len("Queries") && strings.HasSuffix(base, "Queries") {
					stem = strings.ToLower(strings.TrimSuffix(base, "Queries"))
				}
				return stem
			}
			for _, e := range entries {
				if perSourceStem(e.Name()) == "extra" {
					t.Errorf("orphaned output %s survived source deletion", e.Name())
				}
			}
			// The main source's output must survive.
			foundMain := false
			for _, e := range entries {
				if perSourceStem(e.Name()) == "queries" {
					foundMain = true
				}
			}
			if !foundMain {
				t.Errorf("main queries output missing after purge run (entries: %v)", entryNames(entries))
			}
		})
	}
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// ── Cache invalidation on schema change across all languages ─────────────────

func TestAllLangs_SchemaChangeTriggersFullRegen(t *testing.T) {
	langs := []struct {
		name     string
		generate func(cfg *config.Config) error
	}{
		{"js", func(cfg *config.Config) error { return jsgenNew(cfg).Generate() }},
		{"python", func(cfg *config.Config) error { return pygenNew(cfg).Generate() }},
		{"kotlin", func(cfg *config.Config) error { return kotlingenNew(cfg).Generate() }},
		{"java", func(cfg *config.Config) error { return javagenNew(cfg).Generate() }},
		{"rust", func(cfg *config.Config) error { return rustgenNew(cfg).Generate() }},
	}

	for _, lang := range langs {
		t.Run(lang.name, func(t *testing.T) {
			root, schemaDir, queriesDir := langFixture(t)
			out := filepath.Join(root, "gen_"+lang.name)
			cfg := baseConfig(root, schemaDir, queriesDir)
			switch lang.name {
			case "js":
				cfg.Gen.JS = config.JSGen{Enabled: true, Out: out}
			case "python":
				cfg.Gen.Python = config.PythonGen{Enabled: true, Out: out, Async: true}
			case "kotlin":
				cfg.Gen.Kotlin = config.KotlinGen{Enabled: true, Out: out, Package: "com.example"}
			case "java":
				cfg.Gen.Java = config.JavaGen{Enabled: true, Out: out, Package: "com.example"}
			case "rust":
				cfg.Gen.Rust = config.RustGen{Enabled: true, Out: out}
			}

			if err := lang.generate(cfg); err != nil {
				t.Fatalf("first Generate: %v", err)
			}

			// Add a column — models must be regenerated with it.
			if err := os.WriteFile(filepath.Join(schemaDir, "schema.sql"), []byte(`
CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	email VARCHAR(255) NOT NULL,
	name TEXT,
	age INTEGER,
	is_active BOOLEAN NOT NULL DEFAULT true,
	reputation INTEGER DEFAULT 0
);

CREATE TABLE posts (
	id SERIAL PRIMARY KEY,
	user_id INTEGER NOT NULL REFERENCES users(id),
	title TEXT NOT NULL,
	view_count INTEGER DEFAULT 0
);
`), 0644); err != nil {
				t.Fatal(err)
			}
			if err := lang.generate(cfg); err != nil {
				t.Fatalf("second Generate: %v", err)
			}

			// Find the models file for each language and verify the new field.
			var modelsName, marker string
			switch lang.name {
			case "js":
				modelsName, marker = "index.d.ts", "reputation" // JS models live in the d.ts
			case "python":
				modelsName, marker = "models.py", "reputation"
			case "kotlin":
				modelsName, marker = "Models.kt", "reputation"
			case "java":
				modelsName, marker = "Users.java", "reputation"
			case "rust":
				modelsName, marker = "models.rs", "reputation"
			}
			src := readOut(t, out, modelsName)
			if !strings.Contains(strings.ToLower(src), marker) {
				t.Errorf("%s not regenerated after schema change (%s)", modelsName, marker)
			}
		})
	}
}

// ── helpers binding the concrete generators (avoids import cycles) ───────────

func mustNotExist(t *testing.T, dir, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
		t.Errorf("%s must not exist (err=%v)", name, err)
	}
}
