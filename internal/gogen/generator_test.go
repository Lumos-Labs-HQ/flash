package gogen

import (
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

func newGen() *Generator {
	return New(&config.Config{
		SchemaDir: "db/schema",
		Queries:   "db/queries/",
		Database:  config.Database{Provider: "postgresql"},
	})
}

// ── mapSQLTypeToGo ────────────────────────────────────────────────────────────

func TestMapSQLTypeToGo(t *testing.T) {
	g := newGen()
	g.schema = &parser.Schema{}

	cases := []struct {
		sqlType  string
		nullable bool
		want     string
	}{
		{"INTEGER", false, "int64"},
		{"BIGINT", false, "int64"},
		{"SERIAL", false, "int64"},
		{"TEXT", false, "string"},
		{"VARCHAR(255)", false, "string"},
		{"BOOLEAN", false, "bool"},
		{"TIMESTAMP", false, "time.Time"},
		{"DATE", false, "time.Time"},
		{"FLOAT", false, "float64"},
		{"DECIMAL", false, "float64"},
		{"NUMERIC", false, "float64"},
		{"UUID", false, "string"},
		{"JSONB", false, "[]byte"},
		{"JSON", false, "[]byte"},
		// Nullable variants
		{"INTEGER", true, "sql.NullInt64"},
		{"TEXT", true, "sql.NullString"},
		{"BOOLEAN", true, "sql.NullBool"},
		{"FLOAT", true, "sql.NullFloat64"},
		{"TIMESTAMP", true, "sql.NullTime"},
	}

	for _, c := range cases {
		got := g.mapSQLTypeToGo(c.sqlType, c.nullable)
		if got != c.want {
			t.Errorf("mapSQLTypeToGo(%q, nullable=%v) = %q, want %q", c.sqlType, c.nullable, got, c.want)
		}
	}
}

func TestMapSQLTypeToGo_EnumType(t *testing.T) {
	g := newGen()
	g.schema = &parser.Schema{
		Enums: []*parser.Enum{{Name: "status", Values: []string{"active", "inactive"}}},
	}
	got := g.mapSQLTypeToGo("status", false)
	if got != "Status" {
		t.Errorf("mapSQLTypeToGo(enum) = %q, want Status", got)
	}
}

func TestMapSQLTypeToGo_InlineEnum(t *testing.T) {
	g := newGen()
	g.schema = &parser.Schema{}
	got := g.mapSQLTypeToGo("enum('a','b','c')", false)
	if got != "string" {
		t.Errorf("mapSQLTypeToGo(inline enum) = %q, want string", got)
	}
}

// ── getZeroValue ──────────────────────────────────────────────────────────────

func TestGetZeroValue(t *testing.T) {
	g := newGen()
	cases := []struct{ goType, want string }{
		{"string", `""`},
		{"int64", "0"},
		{"float64", "0"},
		{"bool", "false"},
		{"time.Time", "time.Time{}"},
		{"sql.NullString", "sql.NullString{}"},
		{"[]byte", "nil"},
	}
	for _, c := range cases {
		got := g.getZeroValue(c.goType)
		if got != c.want {
			t.Errorf("getZeroValue(%q) = %q, want %q", c.goType, got, c.want)
		}
	}
}

// ── extractEnumValues ─────────────────────────────────────────────────────────

func TestExtractEnumValues(t *testing.T) {
	vals := extractEnumValues("enum('active','inactive','pending')")
	if len(vals) != 3 {
		t.Fatalf("values = %d, want 3: %v", len(vals), vals)
	}
	if vals[0] != "active" {
		t.Errorf("vals[0] = %q, want active", vals[0])
	}
}

func TestExtractEnumValues_NotEnum(t *testing.T) {
	if vals := extractEnumValues("varchar(255)"); vals != nil {
		t.Errorf("non-enum should return nil, got %v", vals)
	}
}

// ── expandWildcardColumns ─────────────────────────────────────────────────────

func TestExpandWildcardColumns_Wildcard(t *testing.T) {
	g := newGen()
	g.schema = &parser.Schema{
		Tables: []*parser.Table{{
			Name: "users",
			Columns: []*parser.Column{
				{Name: "id", Type: "INTEGER"},
				{Name: "email", Type: "TEXT"},
			},
		}},
	}
	query := &parser.Query{
		SQL:     "SELECT * FROM users",
		Columns: []*parser.QueryColumn{{Name: "*", Table: "users"}},
	}
	expanded := g.expandWildcardColumns(query)
	if len(expanded) != 2 {
		t.Errorf("expanded = %d, want 2", len(expanded))
	}
}

func TestExpandWildcardColumns_NoWildcard(t *testing.T) {
	g := newGen()
	g.schema = &parser.Schema{}
	query := &parser.Query{
		SQL:     "SELECT id FROM users",
		Columns: []*parser.QueryColumn{{Name: "id", Type: "INTEGER"}},
	}
	expanded := g.expandWildcardColumns(query)
	if len(expanded) != 1 || expanded[0].Name != "id" {
		t.Errorf("expanded = %v, want [{id INTEGER}]", expanded)
	}
}

// ── computeConfigChecksum ─────────────────────────────────────────────────────

func TestComputeConfigChecksum_Deterministic(t *testing.T) {
	g := newGen()
	h1 := g.computeConfigChecksum()
	h2 := g.computeConfigChecksum()
	if h1 != h2 {
		t.Errorf("config checksum not deterministic: %q != %q", h1, h2)
	}
	if h1 == "" {
		t.Error("config checksum should not be empty")
	}
}

func TestComputeConfigChecksum_DifferentConfig(t *testing.T) {
	g1 := New(&config.Config{SchemaDir: "a", Database: config.Database{Provider: "postgresql"}})
	g2 := New(&config.Config{SchemaDir: "b", Database: config.Database{Provider: "postgresql"}})
	if g1.computeConfigChecksum() == g2.computeConfigChecksum() {
		t.Error("different configs should produce different checksums")
	}
}

// ── mapParamTypeToGo ──────────────────────────────────────────────────────────

func TestMapParamTypeToGo_PipeType(t *testing.T) {
	g := newGen()
	g.schema = &parser.Schema{}
	// "TEXT | null" style — should use first part
	got := g.mapParamTypeToGo("TEXT | null")
	if !strings.Contains(got, "string") {
		t.Errorf("mapParamTypeToGo(TEXT | null) = %q, want string", got)
	}
}
