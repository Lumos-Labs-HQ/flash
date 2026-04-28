package pygen

import (
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

func newGen(provider string, async bool) *Generator {
	return New(&config.Config{
		SchemaDir: "db/schema",
		Queries:   "db/queries/",
		Database:  config.Database{Provider: provider},
		Gen:       config.Gen{Python: config.PythonGen{Enabled: true, Out: "flash_gen", Async: async}},
	})
}

// ── sqlTypeToPython ───────────────────────────────────────────────────────────

func TestSQLTypeToPython(t *testing.T) {
	g := newGen("postgresql", true)
	g.schema = &parser.Schema{}

	cases := []struct {
		sqlType  string
		nullable bool
		want     string
	}{
		{"INTEGER", false, "int"},
		{"BIGINT", false, "int"},
		{"SERIAL", false, "int"},
		{"SMALLINT", false, "int"},
		{"TEXT", false, "str"},
		{"VARCHAR(255)", false, "str"},
		{"UUID", false, "str"},
		{"BOOLEAN", false, "bool"},
		{"FLOAT", false, "float"},
		{"DOUBLE", false, "float"},
		{"REAL", false, "float"},
		{"DECIMAL", false, "Decimal"},
		{"NUMERIC", false, "Decimal"},
		{"TIMESTAMP", false, "datetime"},
		{"DATE", false, "datetime"},
		{"JSON", false, "dict"},
		{"JSONB", false, "dict"},
		// Nullable
		{"INTEGER", true, "Optional[int]"},
		{"TEXT", true, "Optional[str]"},
		{"BOOLEAN", true, "Optional[bool]"},
		// Array
		{"INTEGER[]", false, "List[int]"},
		{"TEXT[]", false, "List[str]"},
	}

	for _, c := range cases {
		got := g.sqlTypeToPython(c.sqlType, c.nullable)
		if got != c.want {
			t.Errorf("sqlTypeToPython(%q, nullable=%v) = %q, want %q", c.sqlType, c.nullable, got, c.want)
		}
	}
}

func TestSQLTypeToPython_EnumType(t *testing.T) {
	g := newGen("postgresql", true)
	g.schema = &parser.Schema{
		Enums: []*parser.Enum{{Name: "status", Values: []string{"active", "inactive"}}},
	}
	got := g.sqlTypeToPython("status", false)
	if !strings.Contains(got, "Literal") {
		t.Errorf("sqlTypeToPython(enum) = %q, want Literal[...]", got)
	}
	if !strings.Contains(got, "'active'") {
		t.Errorf("sqlTypeToPython(enum) = %q, missing 'active'", got)
	}
}

// ── convertSQL ────────────────────────────────────────────────────────────────

func TestConvertSQL_PostgreSQL_NoChange(t *testing.T) {
	g := newGen("postgresql", true)
	sql := "SELECT * FROM users WHERE id = $1"
	if got := g.convertSQL(sql); got != sql {
		t.Errorf("PostgreSQL SQL should not change: %q", got)
	}
}

func TestConvertSQL_MySQL_PercentS(t *testing.T) {
	g := newGen("mysql", true)
	sql := "SELECT * FROM users WHERE id = $1 AND email = $2"
	got := g.convertSQL(sql)
	if strings.Contains(got, "$1") {
		t.Errorf("MySQL SQL should use %%s: %q", got)
	}
	if strings.Count(got, "%s") != 2 {
		t.Errorf("MySQL SQL should have 2 %%s: %q", got)
	}
}

func TestConvertSQL_SQLite_QuestionMark(t *testing.T) {
	g := newGen("sqlite", true)
	sql := "SELECT * FROM users WHERE id = $1 AND name = $2"
	got := g.convertSQL(sql)
	if strings.Contains(got, "$1") || strings.Contains(got, "$2") {
		t.Errorf("SQLite SQL should use ?: %q", got)
	}
	if strings.Count(got, "?") != 2 {
		t.Errorf("SQLite SQL should have 2 ?: %q", got)
	}
}

// ── getReturnType ─────────────────────────────────────────────────────────────

func TestGetReturnType_SingleColumn_One(t *testing.T) {
	g := newGen("postgresql", true)
	g.schema = &parser.Schema{}
	query := &parser.Query{
		Cmd:     ":one",
		Columns: []*parser.QueryColumn{{Name: "id", Type: "INTEGER"}},
	}
	got := g.getReturnType(query)
	if got != "Optional[int]" {
		t.Errorf("getReturnType(:one, single) = %q, want Optional[int]", got)
	}
}

func TestGetReturnType_SingleColumn_Many(t *testing.T) {
	g := newGen("postgresql", true)
	g.schema = &parser.Schema{}
	query := &parser.Query{
		Cmd:     ":many",
		Columns: []*parser.QueryColumn{{Name: "id", Type: "INTEGER"}},
	}
	got := g.getReturnType(query)
	if got != "List[int]" {
		t.Errorf("getReturnType(:many, single) = %q, want List[int]", got)
	}
}

func TestGetReturnType_Exec(t *testing.T) {
	g := newGen("postgresql", true)
	g.schema = &parser.Schema{}
	query := &parser.Query{Cmd: ":exec"}
	got := g.getReturnType(query)
	if got != "int" {
		t.Errorf("getReturnType(:exec) = %q, want int", got)
	}
}

func TestGetReturnType_MultiColumn_One(t *testing.T) {
	g := newGen("postgresql", true)
	g.schema = &parser.Schema{}
	query := &parser.Query{
		Name: "GetUser",
		Cmd:  ":one",
		Columns: []*parser.QueryColumn{
			{Name: "id", Type: "INTEGER"},
			{Name: "email", Type: "TEXT"},
		},
	}
	got := g.getReturnType(query)
	if !strings.Contains(got, "Optional") {
		t.Errorf("getReturnType(multi-col :one) = %q, want Optional[...]", got)
	}
}

// ── needsResultClass ─────────────────────────────────────────────────────────

func TestNeedsResultClass_MultiColumn(t *testing.T) {
	g := newGen("postgresql", true)
	query := &parser.Query{
		Columns: []*parser.QueryColumn{
			{Name: "id"}, {Name: "email"},
		},
	}
	if !g.needsResultClass(query) {
		t.Error("multi-column query should need result class")
	}
}

func TestNeedsResultClass_SingleColumn(t *testing.T) {
	g := newGen("postgresql", true)
	query := &parser.Query{
		Columns: []*parser.QueryColumn{{Name: "id", Type: "INTEGER"}},
	}
	if g.needsResultClass(query) {
		t.Error("single non-wildcard column should not need result class")
	}
}

func TestNeedsResultClass_NoColumns(t *testing.T) {
	g := newGen("postgresql", true)
	if g.needsResultClass(&parser.Query{}) {
		t.Error("no-column query should not need result class")
	}
}

// ── async vs sync method signature ───────────────────────────────────────────

func TestGenerateQueryMethod_AsyncSignature(t *testing.T) {
	g := newGen("postgresql", true)
	g.schema = &parser.Schema{Tables: []*parser.Table{{
		Name:    "users",
		Columns: []*parser.Column{{Name: "id", Type: "INTEGER"}},
	}}}

	var w strings.Builder
	query := &parser.Query{
		Name:    "GetUser",
		SQL:     "SELECT id FROM users WHERE id = $1",
		Cmd:     ":one",
		Params:  []*parser.Param{{Name: "id", Type: "INTEGER"}},
		Columns: []*parser.QueryColumn{{Name: "id", Type: "INTEGER"}},
	}
	g.generateQueryMethod(&w, query)
	if !strings.Contains(w.String(), "async def") {
		t.Errorf("async generator should produce 'async def': %s", w.String())
	}
}

func TestGenerateQueryMethod_SyncSignature(t *testing.T) {
	g := newGen("postgresql", false)
	g.schema = &parser.Schema{Tables: []*parser.Table{{
		Name:    "users",
		Columns: []*parser.Column{{Name: "id", Type: "INTEGER"}},
	}}}

	var w strings.Builder
	query := &parser.Query{
		Name:    "GetUser",
		SQL:     "SELECT id FROM users WHERE id = $1",
		Cmd:     ":one",
		Params:  []*parser.Param{{Name: "id", Type: "INTEGER"}},
		Columns: []*parser.QueryColumn{{Name: "id", Type: "INTEGER"}},
	}
	g.generateQueryMethod(&w, query)
	result := w.String()
	if strings.Contains(result, "async def") {
		t.Errorf("sync generator should not produce 'async def': %s", result)
	}
	if !strings.Contains(result, "def get_user") {
		t.Errorf("sync generator should produce 'def get_user': %s", result)
	}
}
