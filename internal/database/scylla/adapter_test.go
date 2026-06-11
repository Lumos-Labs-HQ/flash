package scylla

import (
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func TestMapColumnType_Basic(t *testing.T) {
	a := New()
	cases := map[string]string{
		"text":          "TEXT",
		"varchar":       "VARCHAR",
		"int":           "INT",
		"bigint":        "BIGINT",
		"ascii":         "ASCII",
		"boolean":       "BOOLEAN",
		"uuid":          "UUID",
		"list<text>":    "LIST<text>",
		"map<text,int>": "MAP<text,int>",
	}
	for in, want := range cases {
		if got := a.MapColumnType(in); got != want {
			t.Errorf("MapColumnType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMapColumnType_Unknown(t *testing.T) {
	a := New()
	if got := a.MapColumnType("set<int>"); got != "SET<int>" {
		t.Errorf("MapColumnType(set<int>) = %q, want SET<int>", got)
	}
	if got := a.MapColumnType("custom"); got != "CUSTOM" {
		t.Errorf("MapColumnType(custom) = %q, want CUSTOM", got)
	}
}

func TestProviderName(t *testing.T) {
	if got := New().ProviderName(); got != "scylla" {
		t.Errorf("got %q, want scylla", got)
	}
}

func TestQuoteIdentifier(t *testing.T) {
	a := New()
	if got := a.QuoteIdentifier("users"); got != `"users"` {
		t.Errorf("got %q", got)
	}
	if got := a.QuoteIdentifier(`table"name`); got != `"table""name"` {
		t.Errorf("got %q", got)
	}
}

func TestGenerateCreateTableSQL(t *testing.T) {
	a := New()
	a.keyspace = "test_ks"
	table := types.SchemaTable{
		Name: "users",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "uuid", IsPrimary: true},
			{Name: "email", Type: "text"},
			{Name: "age", Type: "int"},
		},
	}
	sql := a.GenerateCreateTableSQL(table)
	if !strings.Contains(sql, `"test_ks"."users"`) {
		t.Errorf("missing keyspace.table: %s", sql)
	}
	if !strings.Contains(sql, `PRIMARY KEY ("id")`) {
		t.Errorf("missing PRIMARY KEY: %s", sql)
	}
	if !strings.Contains(sql, `"id" uuid`) {
		t.Errorf("missing id column: %s", sql)
	}
}

func TestGenerateCreateTableSQL_CompositePK(t *testing.T) {
	a := New()
	a.keyspace = "test_ks"
	table := types.SchemaTable{
		Name: "events",
		Columns: []types.SchemaColumn{
			{Name: "user_id", Type: "uuid", IsPrimary: true},
			{Name: "ts", Type: "timestamp", IsPrimary: true},
			{Name: "data", Type: "text"},
		},
	}
	sql := a.GenerateCreateTableSQL(table)
	if !strings.Contains(sql, `PRIMARY KEY`) {
		t.Errorf("missing PRIMARY KEY: %s", sql)
	}
	if !strings.Contains(sql, `"user_id"`) && strings.Contains(sql, `"ts"`) {
		t.Errorf("missing composite PK columns: %s", sql)
	}
}

func TestGenerateCreateTableSQL_NoColumns(t *testing.T) {
	a := New()
	a.keyspace = "test_ks"
	table := types.SchemaTable{Name: "empty", Columns: nil}
	sql := a.GenerateCreateTableSQL(table)
	// CQL requires at least one column with PRIMARY KEY
	if !strings.Contains(sql, `CREATE TABLE`) {
		t.Errorf("missing CREATE TABLE: %s", sql)
	}
}

func TestGenerateAddColumnSQL(t *testing.T) {
	a := New()
	a.keyspace = "test_ks"
	col := types.SchemaColumn{Name: "phone", Type: "text"}
	sql := a.GenerateAddColumnSQL("users", col)
	if !strings.Contains(sql, `"test_ks"."users"`) {
		t.Errorf("missing keyspace qualifier: %s", sql)
	}
	if !strings.Contains(sql, `ADD "phone"`) {
		t.Errorf("missing ADD column: %s", sql)
	}
}

func TestGenerateDropColumnSQL(t *testing.T) {
	a := New()
	a.keyspace = "test_ks"
	sql := a.GenerateDropColumnSQL("users", "phone")
	if !strings.Contains(sql, `"test_ks"."users"`) {
		t.Errorf("missing keyspace qualifier: %s", sql)
	}
	if !strings.Contains(sql, `DROP "phone"`) {
		t.Errorf("missing DROP column: %s", sql)
	}
}

func TestGenerateAlterColumnSQL(t *testing.T) {
	a := New()
	a.keyspace = "test_ks"
	col := types.SchemaColumn{Name: "email", Type: "text"}
	// Same type - no SQL
	if sql := a.GenerateAlterColumnSQL("users", col, "text"); sql != "" {
		t.Errorf("expected empty for same type, got %q", sql)
	}
	// Different type
	sql := a.GenerateAlterColumnSQL("users", col, "varchar")
	if !strings.Contains(sql, `ALTER`) && !strings.Contains(sql, `TYPE text`) {
		t.Errorf("unexpected: %s", sql)
	}
}

func TestGenerateAddIndexSQL(t *testing.T) {
	a := New()
	a.keyspace = "test_ks"
	idx := types.SchemaIndex{Name: "idx_email", Table: "users", Columns: []string{"email"}}
	sql := a.GenerateAddIndexSQL(idx)
	if !strings.Contains(sql, `CREATE INDEX`) {
		t.Errorf("missing CREATE INDEX: %s", sql)
	}
	if !strings.Contains(sql, `"idx_email"`) {
		t.Errorf("missing index name: %s", sql)
	}
}

func TestGenerateDropIndexSQL(t *testing.T) {
	a := New()
	a.keyspace = "test_ks"
	idx := types.SchemaIndex{Name: "idx_email", Table: "users", Columns: []string{"email"}}
	sql := a.GenerateDropIndexSQL(idx)
	if !strings.Contains(sql, `DROP INDEX`) {
		t.Errorf("missing DROP INDEX: %s", sql)
	}
}

func TestSanitizeKeyspace(t *testing.T) {
	if got := sanitizeKeyspace("my-branch"); got != "my_branch" {
		t.Errorf("got %q, want my_branch", got)
	}
	if got := sanitizeKeyspace("main"); got != "main" {
		t.Errorf("got %q, want main", got)
	}
}

func TestFormatColumnType(t *testing.T) {
	a := New()
	col := types.SchemaColumn{Name: "age", Type: "int"}
	if got := a.FormatColumnType(col); got != "int" {
		t.Errorf("got %q", got)
	}
}
