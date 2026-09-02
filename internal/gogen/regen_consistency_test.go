package gogen

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
)

// assertParses verifies generated source is syntactically valid Go.
func assertParses(t *testing.T, name, src string) {
	t.Helper()
	if _, err := parser.ParseFile(token.NewFileSet(), name, src, parser.AllErrors); err != nil {
		t.Errorf("generated file %s does not parse: %v", name, err)
	}
}

// newGenFixture creates a full generation fixture: schema dir, queries dir,
// out dir, and a configured Generator.
func newGenFixture(t *testing.T) (*Generator, string, string, string) {
	t.Helper()
	root := t.TempDir()
	schemaDir := filepath.Join(root, "schema")
	queriesDir := filepath.Join(root, "queries")
	outDir := filepath.Join(root, "flash_gen")
	for _, d := range []string{schemaDir, queriesDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(schemaDir, "schema.sql"), []byte(`
CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	email VARCHAR(255) NOT NULL,
	name TEXT,
	age INTEGER
);
CREATE TABLE posts (
	id SERIAL PRIMARY KEY,
	user_id INTEGER NOT NULL,
	title TEXT NOT NULL,
	view_count INTEGER DEFAULT 0
);
`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		SchemaDir:  schemaDir,
		SchemaPath: filepath.Join(schemaDir, "schema.sql"),
		Queries:    queriesDir,
		Gen:        config.Gen{Go: config.GoGen{Enabled: true, Driver: "database/sql", Out: outDir}},
		Database:   config.Database{Provider: "postgresql"},
	}
	return New(cfg), schemaDir, queriesDir, outDir
}

func writeQueries(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// generatedGoParses checks every generated .go file parses as valid Go.
func generatedGoParses(t *testing.T, outDir string) {
	t.Helper()
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read out dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(outDir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		assertParses(t, e.Name(), string(src))
	}
}

func TestGenerate_TwoRunsStableOutput(t *testing.T) {
	g, _, queriesDir, outDir := newGenFixture(t)
	writeQueries(t, queriesDir, "users.sql", `
-- name: GetUser :one
SELECT id, email, name FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, email FROM users ORDER BY id;

-- name: CreateUser :exec
INSERT INTO users (email, name) VALUES ($1, $2);

-- name: UpdateViewCount :exec
UPDATE posts SET view_count = $1 WHERE id = $2;
`)
	if err := g.Generate(); err != nil {
		t.Fatalf("first Generate: %v", err)
	}
	generatedGoParses(t, outDir)

	first := snapshotDir(t, outDir)

	// Second run with NO changes: byte-identical output (cache says skip).
	if err := g.Generate(); err != nil {
		t.Fatalf("second Generate: %v", err)
	}
	second := snapshotDir(t, outDir)
	for name, content := range first {
		if second[name] != content {
			t.Errorf("file %s changed between identical runs", name)
		}
	}
	generatedGoParses(t, outDir)
}

func TestGenerate_DeletedQueryFilePurgesOrphan(t *testing.T) {
	g, _, queriesDir, outDir := newGenFixture(t)
	writeQueries(t, queriesDir, "users.sql", "-- name: GetUser :one\nSELECT id FROM users WHERE id = $1;\n")
	writeQueries(t, queriesDir, "posts.sql", "-- name: GetPost :one\nSELECT id FROM posts WHERE id = $1;\n")
	if err := g.Generate(); err != nil {
		t.Fatalf("first Generate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "posts.go")); err != nil {
		t.Fatalf("posts.go should exist after first run: %v", err)
	}

	// Delete the query source file.
	if err := os.Remove(filepath.Join(queriesDir, "posts.sql")); err != nil {
		t.Fatal(err)
	}
	if err := g.Generate(); err != nil {
		t.Fatalf("second Generate after delete: %v", err)
	}

	// The orphaned output must be gone.
	if _, err := os.Stat(filepath.Join(outDir, "posts.go")); !os.IsNotExist(err) {
		t.Errorf("orphaned posts.go still exists after query file deletion (err=%v)", err)
	}
	// The surviving file must be intact.
	if _, err := os.Stat(filepath.Join(outDir, "users.go")); err != nil {
		t.Errorf("users.go missing after delete run: %v", err)
	}

	// The cache file must no longer reference the deleted source.
	cacheBytes, err := os.ReadFile(filepath.Join(outDir, ".flash_cache.json"))
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	if strings.Contains(string(cacheBytes), "posts.sql") {
		t.Error("cache still references deleted posts.sql after purge")
	}
	generatedGoParses(t, outDir)
}

func TestGenerate_RenamedQueryFilePurgesOldOutput(t *testing.T) {
	g, _, queriesDir, outDir := newGenFixture(t)
	writeQueries(t, queriesDir, "users.sql", "-- name: GetUser :one\nSELECT id FROM users WHERE id = $1;\n")
	if err := g.Generate(); err != nil {
		t.Fatalf("first Generate: %v", err)
	}

	// Rename users.sql → members.sql
	if err := os.Rename(filepath.Join(queriesDir, "users.sql"), filepath.Join(queriesDir, "members.sql")); err != nil {
		t.Fatal(err)
	}
	if err := g.Generate(); err != nil {
		t.Fatalf("second Generate after rename: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "users.go")); !os.IsNotExist(err) {
		t.Errorf("old users.go still exists after rename (err=%v)", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "members.go")); err != nil {
		t.Errorf("members.go missing after rename: %v", err)
	}
	generatedGoParses(t, outDir)
}

func TestGenerate_SchemaChangeFullRegen(t *testing.T) {
	g, schemaDir, queriesDir, outDir := newGenFixture(t)
	writeQueries(t, queriesDir, "users.sql", "-- name: GetUser :one\nSELECT id, email FROM users WHERE id = $1;\n")
	if err := g.Generate(); err != nil {
		t.Fatalf("first Generate: %v", err)
	}

	// Add a column to the schema — models.go must be regenerated with it.
	if err := os.WriteFile(filepath.Join(schemaDir, "schema.sql"), []byte(`
CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	email VARCHAR(255) NOT NULL,
	reputation INTEGER DEFAULT 0
);
CREATE TABLE posts (
	id SERIAL PRIMARY KEY,
	user_id INTEGER NOT NULL,
	title TEXT NOT NULL,
	view_count INTEGER DEFAULT 0
);
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := g.Generate(); err != nil {
		t.Fatalf("second Generate after schema change: %v", err)
	}
	models, err := os.ReadFile(filepath.Join(outDir, "models.go"))
	if err != nil {
		t.Fatalf("read models.go: %v", err)
	}
	if !strings.Contains(string(models), "Reputation") {
		t.Error("models.go missing new Reputation field after schema change")
	}
	// email must still be present (full regen, not append)
	if !strings.Contains(string(models), "Email") {
		t.Error("models.go lost Email field")
	}
	generatedGoParses(t, outDir)
}

func TestGenerate_DroppedTableRemovesStaleTypes(t *testing.T) {
	g, schemaDir, queriesDir, outDir := newGenFixture(t)
	writeQueries(t, queriesDir, "users.sql", "-- name: GetUser :one\nSELECT id FROM users WHERE id = $1;\n")
	writeQueries(t, queriesDir, "posts.sql", "-- name: GetPost :one\nSELECT id FROM posts WHERE id = $1;\n")
	if err := g.Generate(); err != nil {
		t.Fatalf("first Generate: %v", err)
	}
	models, _ := os.ReadFile(filepath.Join(outDir, "models.go"))
	if !strings.Contains(string(models), "Posts") {
		t.Fatal("control: Posts model should exist in first run")
	}

	// Drop the posts table AND its queries.
	if err := os.WriteFile(filepath.Join(schemaDir, "schema.sql"), []byte(`
CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	email VARCHAR(255) NOT NULL,
	name TEXT,
	age INTEGER
);
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(queriesDir, "posts.sql")); err != nil {
		t.Fatal(err)
	}
	if err := g.Generate(); err != nil {
		t.Fatalf("second Generate after table drop: %v", err)
	}
	models, _ = os.ReadFile(filepath.Join(outDir, "models.go"))
	if strings.Contains(string(models), "Posts") {
		t.Error("models.go still contains dropped Posts model (stale type)")
	}
	generatedGoParses(t, outDir)
}

func TestGenerate_ParamsStructWhenMoreThanTwo(t *testing.T) {
	g, _, queriesDir, outDir := newGenFixture(t)
	writeQueries(t, queriesDir, "users.sql", `
-- name: SearchUsers :many
SELECT id FROM users WHERE email = $1 AND name = $2 AND age = $3;

-- name: GetByEmail :one
SELECT id FROM users WHERE email = $1;
`)
	if err := g.Generate(); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	src, err := os.ReadFile(filepath.Join(outDir, "users.go"))
	if err != nil {
		t.Fatal(err)
	}
	// The struct name casing follows QueryPascal's byte-based shortcut
	// ("Searchusers" here); match case-insensitively.
	if !strings.Contains(strings.ToLower(string(src)), "type searchusersparams struct") {
		t.Error("3-param query should generate a SearchUsersParams struct")
	}
	if strings.Contains(strings.ToLower(string(src)), "type getbyemailparams struct") {
		t.Error("1-param query should NOT generate a Params struct")
	}
	generatedGoParses(t, outDir)
}

func snapshotDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || e.Name() == ".flash_cache.json" {
			continue // cache file legitimately changes (LastGeneration timestamp)
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = string(data)
	}
	return out
}
