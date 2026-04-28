package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
)

func newSchemaParser(t *testing.T, schemaDir string) *SchemaParser {
	t.Helper()
	cfg := &config.Config{
		SchemaDir:  schemaDir,
		SchemaPath: filepath.Join(schemaDir, "schema.sql"),
	}
	return NewSchemaParser(cfg)
}

// ── parseCreateTables ────────────────────────────────────────────────────────

func TestParseCreateTables_Basic(t *testing.T) {
	p := newSchemaParser(t, t.TempDir())
	sql := `CREATE TABLE users (
		id SERIAL PRIMARY KEY,
		email VARCHAR(255) NOT NULL,
		name TEXT
	);`
	tables := p.parseCreateTables(sql)
	if len(tables) != 1 {
		t.Fatalf("tables = %d, want 1", len(tables))
	}
	if tables[0].Name != "users" {
		t.Errorf("name = %q, want users", tables[0].Name)
	}
	if len(tables[0].Columns) != 3 {
		t.Errorf("columns = %d, want 3", len(tables[0].Columns))
	}
}

func TestParseCreateTables_Multiple(t *testing.T) {
	p := newSchemaParser(t, t.TempDir())
	sql := `
	CREATE TABLE users (id SERIAL PRIMARY KEY, email TEXT NOT NULL);
	CREATE TABLE posts (id SERIAL PRIMARY KEY, title TEXT NOT NULL);
	`
	tables := p.parseCreateTables(sql)
	if len(tables) != 2 {
		t.Errorf("tables = %d, want 2", len(tables))
	}
}

func TestParseCreateTables_SkipsConstraintLines(t *testing.T) {
	p := newSchemaParser(t, t.TempDir())
	sql := `CREATE TABLE posts (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`
	tables := p.parseCreateTables(sql)
	if len(tables) != 1 {
		t.Fatalf("tables = %d, want 1", len(tables))
	}
	// FOREIGN KEY line must not become a column
	for _, col := range tables[0].Columns {
		if col.Name == "FOREIGN" {
			t.Error("FOREIGN KEY constraint was parsed as a column")
		}
	}
}

func TestParseCreateTables_NullableDetection(t *testing.T) {
	p := newSchemaParser(t, t.TempDir())
	sql := `CREATE TABLE t (
		id SERIAL PRIMARY KEY,
		required TEXT NOT NULL,
		optional TEXT
	);`
	tables := p.parseCreateTables(sql)
	if len(tables) != 1 {
		t.Fatalf("tables = %d, want 1", len(tables))
	}
	cols := map[string]*Column{}
	for _, c := range tables[0].Columns {
		cols[c.Name] = c
	}
	if cols["required"].Nullable {
		t.Error("required should not be nullable")
	}
	if !cols["optional"].Nullable {
		t.Error("optional should be nullable")
	}
}

// ── parseCreateEnums ─────────────────────────────────────────────────────────

func TestParseCreateEnums_Basic(t *testing.T) {
	p := newSchemaParser(t, t.TempDir())
	sql := `CREATE TYPE status AS ENUM ('active', 'inactive', 'pending');`
	enums := p.parseCreateEnums(sql)
	if len(enums) != 1 {
		t.Fatalf("enums = %d, want 1", len(enums))
	}
	if enums[0].Name != "status" {
		t.Errorf("name = %q, want status", enums[0].Name)
	}
	if len(enums[0].Values) != 3 {
		t.Errorf("values = %d, want 3", len(enums[0].Values))
	}
}

func TestParseCreateEnums_Multiple(t *testing.T) {
	p := newSchemaParser(t, t.TempDir())
	sql := `
	CREATE TYPE role AS ENUM ('admin', 'user');
	CREATE TYPE status AS ENUM ('active', 'inactive');
	`
	enums := p.parseCreateEnums(sql)
	if len(enums) != 2 {
		t.Errorf("enums = %d, want 2", len(enums))
	}
}

// ── SchemaParser.Parse ────────────────────────────────────────────────────────

func TestSchemaParser_Parse_Dir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "users.sql"), []byte(`
		CREATE TABLE users (id SERIAL PRIMARY KEY, email TEXT NOT NULL);
	`), 0644)
	os.WriteFile(filepath.Join(dir, "posts.sql"), []byte(`
		CREATE TABLE posts (id SERIAL PRIMARY KEY, title TEXT NOT NULL);
	`), 0644)

	p := newSchemaParser(t, dir)
	schema, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(schema.Tables) != 2 {
		t.Errorf("tables = %d, want 2", len(schema.Tables))
	}
}

func TestSchemaParser_Parse_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	p := newSchemaParser(t, dir)
	schema, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(schema.Tables) != 0 {
		t.Errorf("tables = %d, want 0", len(schema.Tables))
	}
}

// ── TypeInferrer ─────────────────────────────────────────────────────────────

func TestTypeInferrer_InferParamName_Insert(t *testing.T) {
	ti := NewTypeInferrer()
	sql := `INSERT INTO users (email, name) VALUES ($1, $2)`
	if got := ti.InferParamName(sql, 1); got != "email" {
		t.Errorf("param 1 = %q, want email", got)
	}
	if got := ti.InferParamName(sql, 2); got != "name" {
		t.Errorf("param 2 = %q, want name", got)
	}
}

func TestTypeInferrer_InferParamName_Where(t *testing.T) {
	ti := NewTypeInferrer()
	sql := `SELECT * FROM users WHERE id = $1`
	if got := ti.InferParamName(sql, 1); got != "id" {
		t.Errorf("param 1 = %q, want id", got)
	}
}

func TestTypeInferrer_InferParamName_Limit(t *testing.T) {
	ti := NewTypeInferrer()
	sql := `SELECT * FROM users LIMIT $1`
	if got := ti.InferParamName(sql, 1); got != "limit" {
		t.Errorf("param 1 = %q, want limit", got)
	}
}

func TestTypeInferrer_InferParamName_Fallback(t *testing.T) {
	ti := NewTypeInferrer()
	sql := `SELECT 1`
	got := ti.InferParamName(sql, 1)
	if got != "param1" {
		t.Errorf("fallback = %q, want param1", got)
	}
}

func TestTypeInferrer_InferParamType_WhereColumn(t *testing.T) {
	ti := NewTypeInferrer()
	table := &Table{
		Name: "users",
		Columns: []*Column{
			{Name: "id", Type: "SERIAL"},
			{Name: "email", Type: "TEXT"},
		},
	}
	sql := `SELECT * FROM users WHERE id = $1`
	got := ti.InferParamType(sql, 1, table, "id")
	if got != "SERIAL" {
		t.Errorf("type = %q, want SERIAL", got)
	}
}

func TestTypeInferrer_InferParamType_Limit(t *testing.T) {
	ti := NewTypeInferrer()
	table := &Table{Name: "users", Columns: []*Column{}}
	got := ti.InferParamType(`SELECT * FROM users LIMIT $1`, 1, table, "limit")
	if got != "INTEGER" {
		t.Errorf("type = %q, want INTEGER", got)
	}
}

func TestTypeInferrer_Cache(t *testing.T) {
	ti := NewTypeInferrer()
	table := &Table{
		Name:    "users",
		Columns: []*Column{{Name: "id", Type: "SERIAL"}},
	}
	sql := `SELECT * FROM users WHERE id = $1`
	// Call twice — second should hit cache
	first := ti.InferParamType(sql, 1, table, "id")
	second := ti.InferParamType(sql, 1, table, "id")
	if first != second {
		t.Errorf("cache inconsistency: %q != %q", first, second)
	}
}
