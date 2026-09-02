package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
)

// Shared fixtures for the pipeline-level smoke suite.

// smokeSchema is a realistic multi-table schema exercising the common
// column types, nullability, and enum/JSONB features.
const smokeSchema = `
CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	email VARCHAR(255) NOT NULL,
	name TEXT,
	age INTEGER,
	is_active BOOLEAN NOT NULL DEFAULT true,
	rating DOUBLE PRECISION,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
	updated_at TIMESTAMP,
	tags TEXT[],
	preferences JSONB,
	settings JSONB
);

CREATE TABLE posts (
	id SERIAL PRIMARY KEY,
	user_id INTEGER NOT NULL REFERENCES users(id),
	title TEXT NOT NULL,
	body TEXT,
	status VARCHAR(20) DEFAULT 'draft',
	view_count INTEGER DEFAULT 0,
	published_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE comments (
	id SERIAL PRIMARY KEY,
	post_id INTEGER NOT NULL REFERENCES posts(id),
	user_id INTEGER NOT NULL REFERENCES users(id),
	body TEXT NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE TYPE post_status AS ENUM ('draft', 'published', 'archived');
`

func writeSmokeSchema(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "schema.sql"), []byte(smokeSchema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
}

// newSmokeParser returns a QueryParser with the standard smoke schema loaded,
// bound to a temp queries dir.
func newSmokeParser(t *testing.T) (*QueryParser, *Schema, string) {
	t.Helper()
	root := t.TempDir()
	schemaDir := filepath.Join(root, "schema")
	queriesDir := filepath.Join(root, "queries")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatalf("mkdir schema: %v", err)
	}
	if err := os.MkdirAll(queriesDir, 0755); err != nil {
		t.Fatalf("mkdir queries: %v", err)
	}
	writeSmokeSchema(t, schemaDir)

	cfg := &config.Config{
		SchemaDir:  schemaDir,
		SchemaPath: filepath.Join(schemaDir, "schema.sql"),
		Queries:    queriesDir,
	}
	sp := NewSchemaParser(cfg)
	schema, err := sp.Parse()
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	return NewQueryParser(cfg), schema, queriesDir
}

func writeQueryFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write query file %s: %v", name, err)
	}
}

// paramSummary renders a query's params as "name type" pairs for easy assertion.
func paramSummary(q *Query) string {
	parts := make([]string, 0, len(q.Params))
	for _, p := range q.Params {
		parts = append(parts, p.Name+" "+p.Type)
	}
	return strings.Join(parts, ", ")
}

func findQuery(t *testing.T, queries []*Query, name string) *Query {
	t.Helper()
	for _, q := range queries {
		if q.Name == name {
			return q
		}
	}
	t.Fatalf("query %q not found among %d parsed queries", name, len(queries))
	return nil
}

// ── QueryParser.Parse end-to-end (previously zero coverage) ─────────────────

func TestParsePipeline_BasicSelect(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", `
-- name: GetUser :one
-- Fetch a single user by id
SELECT id, email, name FROM users WHERE id = $1;
`)
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	q := findQuery(t, queries, "GetUser")
	if q.Cmd != ":one" {
		t.Errorf("Cmd = %q, want :one", q.Cmd)
	}
	if q.SourceFile != "users" {
		t.Errorf("SourceFile = %q, want users", q.SourceFile)
	}
	// Param type is the schema-declared column type verbatim (SERIAL);
	// language generators map it to int64 downstream.
	if len(q.Params) != 1 || q.Params[0].Name != "id" || q.Params[0].Type != "SERIAL" {
		t.Errorf("params = [%s], want [id SERIAL]", paramSummary(q))
	}
	if len(q.Columns) != 3 {
		t.Fatalf("columns = %d, want 3: %+v", len(q.Columns), q.Columns)
	}
	wantTypes := map[string]string{"id": "SERIAL", "email": "VARCHAR(255)", "name": "TEXT"}
	for _, col := range q.Columns {
		if want, ok := wantTypes[col.Name]; ok {
			if col.Type != want {
				t.Errorf("column %s type = %q, want %q", col.Name, col.Type, want)
			}
		}
	}
	// name is nullable, others are not
	for _, col := range q.Columns {
		if col.Name == "name" && !col.Nullable {
			t.Error("name column should be nullable")
		}
		if col.Name == "id" && col.Nullable {
			t.Error("id column should not be nullable")
		}
	}
}

func TestParsePipeline_MultipleQueriesPerFile(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", `
-- name: GetUser :one
SELECT id, name FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, name FROM users ORDER BY id;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: CreateUser :exec
INSERT INTO users (email, name) VALUES ($1, $2);
`)
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(queries) != 4 {
		t.Fatalf("queries = %d, want 4", len(queries))
	}
	for _, name := range []string{"GetUser", "ListUsers", "DeleteUser", "CreateUser"} {
		findQuery(t, queries, name)
	}
}

func TestParsePipeline_CQLFilesGlobbed(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.cql", "-- name: GetUser :one\nSELECT id FROM users WHERE id = $1;\n")
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(queries) != 1 || queries[0].Name != "GetUser" {
		t.Fatalf("queries = %+v, want one GetUser", queries)
	}
}

func TestParsePipeline_EmptyDir(t *testing.T) {
	p, schema, _ := newSmokeParser(t)
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(queries) != 0 {
		t.Errorf("queries = %d, want 0", len(queries))
	}
}

func TestParsePipeline_DirWithOnlyNonSQLFiles(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "notes.txt", "not sql at all")
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(queries) != 0 {
		t.Errorf("queries = %d, want 0 (non-.sql/.cql files ignored)", len(queries))
	}
}

func TestParsePipeline_RelativeQueriesPath(t *testing.T) {
	// Queries path is resolved against cwd when relative — verify a relative
	// path pointing into a temp dir works from the current package dir.
	root := t.TempDir()
	schemaDir := filepath.Join(root, "schema")
	queriesDir := filepath.Join(root, "queries")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(queriesDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeSmokeSchema(t, schemaDir)
	writeQueryFile(t, queriesDir, "users.sql", "-- name: GetUser :one\nSELECT id FROM users WHERE id = $1;\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	cfg := &config.Config{
		SchemaDir:  schemaDir,
		SchemaPath: filepath.Join(schemaDir, "schema.sql"),
		Queries:    "queries",
	}
	schema, err := NewSchemaParser(cfg).Parse()
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	queries, err := NewQueryParser(cfg).Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(queries) != 1 || queries[0].Name != "GetUser" {
		t.Fatalf("queries = %+v", queries)
	}
}

func TestParsePipeline_UnknownTableErrors(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "bad.sql", "-- name: Bad :one\nSELECT id FROM nonexistent_table WHERE id = $1;\n")
	_, err := p.Parse(schema)
	if err == nil {
		t.Fatal("expected error for unknown table")
	}
	if !strings.Contains(err.Error(), "nonexistent_table") {
		t.Errorf("error should mention the table name, got: %v", err)
	}
}

func TestParsePipeline_ErrorAbortsAllFiles(t *testing.T) {
	// One bad file must fail the whole Parse, not silently return partial results.
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "good.sql", "-- name: Good :one\nSELECT id FROM users WHERE id = $1;\n")
	writeQueryFile(t, dir, "bad.sql", "-- name: Bad :one\nSELECT id FROM nope WHERE id = $1;\n")
	_, err := p.Parse(schema)
	if err == nil {
		t.Fatal("expected error when one file references an unknown table")
	}
}

func TestParsePipeline_NameWithoutCmdSilentlyDropped(t *testing.T) {
	// Documents current behavior: "-- name: X" with no cmd token drops the
	// query AND its SQL lines (parseQueryFile keeps currentQuery == nil).
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", `
-- name: NoCmd
SELECT id FROM users WHERE id = $1;

-- name: Good :one
SELECT id FROM users WHERE id = $2;
`)
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("queries = %d, want 1 (NoCmd silently dropped)", len(queries))
	}
	findQuery(t, queries, "Good")
}

func TestParsePipeline_NameAnnotationSpaceVariant(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", "-- name : GetUser :one\nSELECT id FROM users WHERE id = $1;\n")
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	q := findQuery(t, queries, "GetUser")
	if q.Cmd != ":one" {
		t.Errorf("Cmd = %q, want :one", q.Cmd)
	}
}

func TestParsePipeline_NoSpaceAfterDashesDropped(t *testing.T) {
	// "--name: X :one" lacks the required prefix "-- name:" — treated as a
	// plain comment, query silently dropped (documents current behavior).
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", "--name: X :one\nSELECT id FROM users WHERE id = $1;\n")
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(queries) != 0 {
		t.Errorf("queries = %d, want 0 (--name: without space is a comment)", len(queries))
	}
}

func TestParsePipeline_ExtraTokensAfterCmdIgnored(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", "-- name: GetUser :one extra tokens here\nSELECT id FROM users WHERE id = $1;\n")
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	q := findQuery(t, queries, "GetUser")
	if q.Cmd != ":one" {
		t.Errorf("Cmd = %q, want :one", q.Cmd)
	}
}

func TestParsePipeline_SQLBeforeAnnotationDropped(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", `
SELECT id FROM users WHERE id = $1;

-- name: GetUser :one
SELECT id FROM users WHERE id = $1;
`)
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("queries = %d, want 1 (leading SQL dropped)", len(queries))
	}
}

func TestParsePipeline_LastCommentWins(t *testing.T) {
	// Only the last comment line before a query survives (each `--` line
	// overwrites the previous). Locks in current behavior.
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", `
-- name: GetUser :one
-- first comment
-- second comment
SELECT id FROM users WHERE id = $1;
`)
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	q := findQuery(t, queries, "GetUser")
	if q.Comment != "second comment" {
		t.Errorf("Comment = %q, want %q (last comment wins)", q.Comment, "second comment")
	}
}

func TestParsePipeline_MultiLineSQLJoinedWithSpaces(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", `
-- name: GetUser :one
SELECT id, name
FROM users
WHERE id = $1
  AND email = $2;
`)
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	q := findQuery(t, queries, "GetUser")
	if strings.Contains(q.SQL, "\n") {
		t.Errorf("SQL should be single-line, got %q", q.SQL)
	}
	if len(q.Params) != 2 {
		t.Fatalf("params = [%s], want 2", paramSummary(q))
	}
	if q.Params[0].Name != "id" || q.Params[1].Name != "email" {
		t.Errorf("param names = %s, want id, email", paramSummary(q))
	}
}

func TestParsePipeline_UnknownCmdAccepted(t *testing.T) {
	// The parser does not validate Cmd — any token is stored verbatim.
	// Generators fall back to exec-style handling. Locks in current behavior.
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", "-- name: Weird :bogus\nSELECT id FROM users WHERE id = $1;\n")
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	q := findQuery(t, queries, "Weird")
	if q.Cmd != ":bogus" {
		t.Errorf("Cmd = %q, want :bogus (stored verbatim)", q.Cmd)
	}
}

func TestParsePipeline_DuplicateQueryNamesAcrossFiles(t *testing.T) {
	// Documents current behavior: duplicate names are silently allowed.
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "a.sql", "-- name: GetUser :one\nSELECT id FROM users WHERE id = $1;\n")
	writeQueryFile(t, dir, "b.sql", "-- name: GetUser :one\nSELECT email FROM users WHERE id = $1;\n")
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(queries) != 2 {
		t.Fatalf("queries = %d, want 2 (duplicates allowed)", len(queries))
	}
}

func TestParsePipeline_AnnotationBeforeNameLine(t *testing.T) {
	// @required/@json/@cache before "-- name:" must attach to the NEXT query
	// via the pending* variables.
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", `
-- @required: id, email
-- name: GetUser :one
SELECT id, email FROM users WHERE id = $1 AND email = $2;
`)
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	q := findQuery(t, queries, "GetUser")
	if len(q.RequiredCols) != 2 || q.RequiredCols[0] != "id" || q.RequiredCols[1] != "email" {
		t.Errorf("RequiredCols = %v, want [id email]", q.RequiredCols)
	}
}

func TestParsePipeline_RequiredStar(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", `
-- name: GetUser :one
-- @required: *
SELECT id FROM users WHERE id = $1;
`)
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	q := findQuery(t, queries, "GetUser")
	if len(q.RequiredCols) != 1 || q.RequiredCols[0] != "*" {
		t.Errorf("RequiredCols = %v, want [*]", q.RequiredCols)
	}
}

func TestParsePipeline_CacheAnnotationDefaults(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", `
-- @cache
-- name: GetUser :one
SELECT id FROM users WHERE id = $1;
`)
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	q := findQuery(t, queries, "GetUser")
	if q.CacheDef == nil {
		t.Fatal("CacheDef should be set by bare -- @cache")
	}
}

func TestParsePipeline_CacheAnnotationFull(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", `
-- @cache {"ttl": "30s", "name": "UserCache", "tags": ["users"], "dep": ["UpdateUser"]}
-- name: GetUser :one
SELECT id FROM users WHERE id = $1;
`)
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	q := findQuery(t, queries, "GetUser")
	if q.CacheDef == nil {
		t.Fatal("CacheDef missing")
	}
	if q.CacheDef.TTL != "30s" || q.CacheDef.Name != "UserCache" {
		t.Errorf("CacheDef = %+v, want ttl=30s name=UserCache", q.CacheDef)
	}
	if len(q.CacheDef.Tags) != 1 || q.CacheDef.Tags[0] != "users" {
		t.Errorf("Tags = %v, want [users]", q.CacheDef.Tags)
	}
	if len(q.CacheDef.Dep) != 1 || q.CacheDef.Dep[0] != "UpdateUser" {
		t.Errorf("Dep = %v, want [UpdateUser]", q.CacheDef.Dep)
	}
}

func TestParsePipeline_CacheAnnotationInvalidJSON(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", `
-- @cache {not json
-- name: GetUser :one
SELECT id FROM users WHERE id = $1;
`)
	_, err := p.Parse(schema)
	if err == nil {
		t.Fatal("expected error for invalid @cache JSON")
	}
}

func TestParsePipeline_RequiredUnknownParam(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", `
-- @required: nonexistent
-- name: GetUser :one
SELECT id FROM users WHERE id = $1;
`)
	_, err := p.Parse(schema)
	if err == nil {
		t.Fatal("expected error when @required names a param that does not exist")
	}
}

func TestParsePipeline_UnknownInsertColumn(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", "-- name: Bad :exec\nINSERT INTO users (nope) VALUES ($1);\n")
	_, err := p.Parse(schema)
	if err == nil {
		t.Fatal("expected error for unknown INSERT column")
	}
}

func TestParsePipeline_UnknownUpdateColumn(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", "-- name: Bad :exec\nUPDATE users SET nope = $1 WHERE id = $2;\n")
	_, err := p.Parse(schema)
	if err == nil {
		t.Fatal("expected error for unknown UPDATE SET column")
	}
}

func TestParsePipeline_InsertColumnValueCountMismatch(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", "-- name: Bad :exec\nINSERT INTO users (email, name) VALUES ($1);\n")
	_, err := p.Parse(schema)
	if err == nil {
		t.Fatal("expected error for INSERT column/value count mismatch")
	}
}

func TestParsePipeline_ReturningColumnsExtracted(t *testing.T) {
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", "-- name: CreateUser :one\nINSERT INTO users (email, name) VALUES ($1, $2) RETURNING id, created_at;\n")
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	q := findQuery(t, queries, "CreateUser")
	names := make([]string, 0, len(q.Columns))
	for _, col := range q.Columns {
		names = append(names, col.Name)
	}
	if len(q.Columns) != 2 || names[0] != "id" || names[1] != "created_at" {
		t.Errorf("RETURNING columns = %v, want [id created_at]", names)
	}
}

func TestParsePipeline_RelativeQueriesPathAbsolute(t *testing.T) {
	// Same as RelativeQueriesPath but with absolute path (control case).
	p, schema, dir := newSmokeParser(t)
	writeQueryFile(t, dir, "users.sql", "-- name: GetUser :one\nSELECT id FROM users WHERE id = $1;\n")
	queries, err := p.Parse(schema)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("queries = %d, want 1", len(queries))
	}
}
