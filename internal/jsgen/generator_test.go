package jsgen

import (
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

func newGen(provider string) *Generator {
	return New(&config.Config{
		SchemaDir: "db/schema",
		Queries:   "db/queries/",
		Database:  config.Database{Provider: provider},
		Gen:       config.Gen{JS: config.JSGen{Enabled: true, Out: "flash_gen"}},
	})
}

// ── mapSQLTypeToJS ────────────────────────────────────────────────────────────

func TestMapSQLTypeToJS(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}

	cases := []struct{ sqlType, want string }{
		{"INTEGER", "number"},
		{"BIGINT", "number"},
		{"SERIAL", "number"},
		{"FLOAT", "number"},
		{"DECIMAL", "number"},
		{"NUMERIC", "number"},
		{"TEXT", "string"},
		{"VARCHAR(255)", "string"},
		{"UUID", "string"},
		{"BOOLEAN", "boolean"},
		{"JSONB", "Object"},
		{"JSON", "Object"},
		{"TIMESTAMP", "Date"},
		{"DATE", "Date"},
		{"BYTEA", "Uint8Array"},
	}
	for _, c := range cases {
		got := g.mapSQLTypeToJS(c.sqlType)
		if got != c.want {
			t.Errorf("mapSQLTypeToJS(%q) = %q, want %q", c.sqlType, got, c.want)
		}
	}
}

func TestMapSQLTypeToJS_ArrayType(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}
	got := g.mapSQLTypeToJS("TEXT[]")
	if got != "string[]" {
		t.Errorf("mapSQLTypeToJS(TEXT[]) = %q, want string[]", got)
	}
}

func TestMapSQLTypeToJS_EnumType(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{
		Enums: []*parser.Enum{{Name: "status", Values: []string{"active", "inactive"}}},
	}
	got := g.mapSQLTypeToJS("status")
	if !strings.Contains(got, "'active'") || !strings.Contains(got, "'inactive'") {
		t.Errorf("mapSQLTypeToJS(enum) = %q, want union type", got)
	}
}

func TestMapSQLTypeToJS_InlineEnum(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}
	got := g.mapSQLTypeToJS("enum('a','b')")
	if !strings.Contains(got, "'a'") || !strings.Contains(got, "'b'") {
		t.Errorf("mapSQLTypeToJS(inline enum) = %q, want union", got)
	}
}

// ── convertSQL ────────────────────────────────────────────────────────────────

func TestConvertSQL_PostgreSQL_NoChange(t *testing.T) {
	g := newGen("postgresql")
	sql := "SELECT * FROM users WHERE id = $1 AND email = $2"
	got := g.convertSQL(sql)
	if got != sql {
		t.Errorf("PostgreSQL SQL should not be converted: %q", got)
	}
}

func TestConvertSQL_MySQL_QuestionMark(t *testing.T) {
	g := newGen("mysql")
	sql := "SELECT * FROM users WHERE id = $1 AND email = $2"
	got := g.convertSQL(sql)
	if strings.Contains(got, "$1") || strings.Contains(got, "$2") {
		t.Errorf("MySQL SQL should use ?: %q", got)
	}
	if strings.Count(got, "?") != 2 {
		t.Errorf("MySQL SQL should have 2 ?: %q", got)
	}
}

func TestConvertSQL_SQLite_QuestionMark(t *testing.T) {
	g := newGen("sqlite")
	sql := "SELECT * FROM users WHERE id = $1"
	got := g.convertSQL(sql)
	if strings.Contains(got, "$1") {
		t.Errorf("SQLite SQL should use ?: %q", got)
	}
}

// ── extractEnumValuesFromType ─────────────────────────────────────────────────

func TestExtractEnumValuesFromType(t *testing.T) {
	vals := extractEnumValuesFromType("enum('active','inactive')")
	if len(vals) != 2 || vals[0] != "active" || vals[1] != "inactive" {
		t.Errorf("values = %v", vals)
	}
}

func TestExtractEnumValuesFromType_NotEnum(t *testing.T) {
	if vals := extractEnumValuesFromType("varchar(255)"); vals != nil {
		t.Errorf("non-enum should return nil, got %v", vals)
	}
}

// ── getReturnType ─────────────────────────────────────────────────────────────

func TestGetReturnType_SingleCountColumn(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}
	query := &parser.Query{
		SQL:     "SELECT COUNT(*) FROM users",
		Cmd:     ":one",
		Columns: []*parser.QueryColumn{{Name: "count", Type: "INTEGER"}},
	}
	got := g.getReturnType(query)
	if got != "number" {
		t.Errorf("getReturnType(count) = %q, want number", got)
	}
}

func TestGetReturnType_MultipleColumns(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}
	query := &parser.Query{
		SQL: "SELECT id, email FROM users",
		Cmd: ":many",
		Columns: []*parser.QueryColumn{
			{Name: "id", Type: "INTEGER"},
			{Name: "email", Type: "TEXT"},
		},
	}
	got := g.getReturnType(query)
	if got != "Object" {
		t.Errorf("getReturnType(multi-col) = %q, want Object", got)
	}
}
