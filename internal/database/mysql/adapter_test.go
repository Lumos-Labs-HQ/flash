package mysql

import (
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func newAdapter() *Adapter {
	return New()
}

// ── MapColumnType ─────────────────────────────────────────────────────────────

func TestMapColumnType(t *testing.T) {
	a := newAdapter()
	cases := []struct{ in, want string }{
		{"varchar", "VARCHAR"},
		{"text", "TEXT"},
		{"longtext", "TEXT"},
		{"int", "INT"},
		{"integer", "INT"},
		{"bigint", "BIGINT"},
		{"boolean", "BOOLEAN"},
		{"datetime", "DATETIME"},
		{"timestamp", "TIMESTAMP"},
		{"decimal", "DECIMAL"},
		{"json", "JSON"},
		{"UNKNOWN", "UNKNOWN"},
	}
	for _, c := range cases {
		got := a.MapColumnType(c.in)
		if got != c.want {
			t.Errorf("MapColumnType(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ── extractEnumValues ─────────────────────────────────────────────────────────

func TestExtractEnumValues_Basic(t *testing.T) {
	vals := extractEnumValues("enum('active','inactive','pending')")
	if len(vals) != 3 {
		t.Fatalf("values = %d, want 3: %v", len(vals), vals)
	}
	if vals[0] != "active" || vals[1] != "inactive" || vals[2] != "pending" {
		t.Errorf("values = %v", vals)
	}
}

func TestExtractEnumValues_NotEnum(t *testing.T) {
	if vals := extractEnumValues("varchar(255)"); vals != nil {
		t.Errorf("non-enum should return nil, got %v", vals)
	}
}

// ── FormatColumnType ──────────────────────────────────────────────────────────

func TestFormatColumnType_PrimaryKey(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "id", Type: "INT", IsPrimary: true}
	got := a.FormatColumnType(col)
	if !strings.Contains(got, "PRIMARY KEY") {
		t.Errorf("FormatColumnType(PK) = %q, missing PRIMARY KEY", got)
	}
}

func TestFormatColumnType_NotNull(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "email", Type: "VARCHAR(255)", Nullable: false}
	got := a.FormatColumnType(col)
	if !strings.Contains(got, "NOT NULL") {
		t.Errorf("FormatColumnType(NOT NULL) = %q, missing NOT NULL", got)
	}
}

func TestFormatColumnType_AutoIncrement(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "id", Type: "INT", IsPrimary: true, IsAutoIncrement: true}
	got := a.FormatColumnType(col)
	if !strings.Contains(got, "AUTO_INCREMENT") {
		t.Errorf("FormatColumnType(AUTO_INCREMENT) = %q, missing AUTO_INCREMENT", got)
	}
}

// ── GenerateCreateTableSQL ────────────────────────────────────────────────────

func TestGenerateCreateTableSQL_Basic(t *testing.T) {
	a := newAdapter()
	table := types.SchemaTable{
		Name: "users",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "INT", IsPrimary: true, IsAutoIncrement: true},
			{Name: "email", Type: "VARCHAR(255)", Nullable: false},
		},
	}
	sql := a.GenerateCreateTableSQL(table)
	for _, want := range []string{"CREATE TABLE", "users", "id", "email"} {
		if !strings.Contains(sql, want) {
			t.Errorf("GenerateCreateTableSQL missing %q:\n%s", want, sql)
		}
	}
}

// ── GenerateAddColumnSQL ──────────────────────────────────────────────────────

func TestGenerateAddColumnSQL(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "phone", Type: "VARCHAR(20)", Nullable: true}
	sql := a.GenerateAddColumnSQL("users", col)
	if !strings.Contains(sql, "ALTER TABLE") || !strings.Contains(sql, "phone") {
		t.Errorf("GenerateAddColumnSQL = %q", sql)
	}
}

// ── GenerateDropColumnSQL ─────────────────────────────────────────────────────

func TestGenerateDropColumnSQL(t *testing.T) {
	a := newAdapter()
	sql := a.GenerateDropColumnSQL("users", "phone")
	if !strings.Contains(sql, "DROP COLUMN") || !strings.Contains(sql, "phone") {
		t.Errorf("GenerateDropColumnSQL = %q", sql)
	}
}

// ── GenerateAddIndexSQL ───────────────────────────────────────────────────────

func TestGenerateAddIndexSQL_Unique(t *testing.T) {
	a := newAdapter()
	idx := types.SchemaIndex{Name: "idx_email", Table: "users", Columns: []string{"email"}, Unique: true}
	sql := a.GenerateAddIndexSQL(idx)
	if !strings.Contains(sql, "UNIQUE") {
		t.Errorf("GenerateAddIndexSQL(unique) = %q, missing UNIQUE", sql)
	}
}

// ── GenerateDropIndexSQL ──────────────────────────────────────────────────────

func TestGenerateDropIndexSQL(t *testing.T) {
	a := newAdapter()
	idx := types.SchemaIndex{Name: "idx_email", Table: "users"}
	sql := a.GenerateDropIndexSQL(idx)
	if !strings.Contains(sql, "DROP INDEX") || !strings.Contains(sql, "idx_email") {
		t.Errorf("GenerateDropIndexSQL = %q", sql)
	}
}
