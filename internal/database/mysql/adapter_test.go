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

func TestFormatColumnType_Unique(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "email", Type: "VARCHAR(255)", IsUnique: true, Nullable: false}
	got := a.FormatColumnType(col)
	if !strings.Contains(got, "UNIQUE") {
		t.Errorf("FormatColumnType(UNIQUE) = %q, missing UNIQUE", got)
	}
	// UNIQUE column should NOT also have PRIMARY KEY
	if strings.Contains(got, "PRIMARY KEY") {
		t.Errorf("FormatColumnType(UNIQUE) = %q, should not contain PRIMARY KEY", got)
	}
}

func TestFormatColumnType_Default(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "status", Type: "VARCHAR(20)", Nullable: false, Default: "'active'"}
	got := a.FormatColumnType(col)
	if !strings.Contains(got, "DEFAULT 'active'") {
		t.Errorf("FormatColumnType(DEFAULT) = %q, missing DEFAULT clause", got)
	}
}

func TestFormatColumnType_Check(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "age", Type: "INT", Nullable: false, Check: "age >= 0"}
	got := a.FormatColumnType(col)
	if !strings.Contains(got, "CHECK (age >= 0)") {
		t.Errorf("FormatColumnType(CHECK) = %q, missing CHECK clause", got)
	}
}

func TestFormatColumnType_ForeignKey(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{
		Name:             "user_id",
		Type:             "INT",
		Nullable:         false,
		ForeignKeyTable:  "users",
		ForeignKeyColumn: "id",
		OnDeleteAction:   "CASCADE",
	}
	got := a.FormatColumnType(col)
	// MySQL adds FK constraints at the table level, not in the column definition.
	// FormatColumnType should NOT include REFERENCES for MySQL.
	if strings.Contains(got, "REFERENCES") {
		t.Errorf("FormatColumnType(FK) = %q, should NOT contain REFERENCES for MySQL", got)
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

// ── GenerateAlterColumnSQL ────────────────────────────────────────────────────

func TestGenerateAlterColumnSQL_ModifyWithoutPrimaryKey(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "id", Type: "INT", IsPrimary: true, IsAutoIncrement: true}
	sql := a.GenerateAlterColumnSQL("users", col, "")
	// MODIFY COLUMN must NOT include PRIMARY KEY — doing so on a column
	// that already has the primary key causes MySQL Error 1068.
	if strings.Contains(sql, "PRIMARY KEY") {
		t.Errorf("GenerateAlterColumnSQL must NOT contain PRIMARY KEY, got %q", sql)
	}
	if !strings.Contains(sql, "AUTO_INCREMENT") {
		t.Errorf("GenerateAlterColumnSQL missing AUTO_INCREMENT, got %q", sql)
	}
	if !strings.Contains(sql, "MODIFY COLUMN `id`") {
		t.Errorf("GenerateAlterColumnSQL missing MODIFY COLUMN, got %q", sql)
	}
}

func TestGenerateAlterColumnSQL_NonPrimaryColumn(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "email", Type: "VARCHAR(255)", Nullable: false, IsUnique: true}
	sql := a.GenerateAlterColumnSQL("users", col, "")
	if !strings.Contains(sql, "UNIQUE") {
		t.Errorf("GenerateAlterColumnSQL missing UNIQUE, got %q", sql)
	}
	if !strings.Contains(sql, "NOT NULL") {
		t.Errorf("GenerateAlterColumnSQL missing NOT NULL, got %q", sql)
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

func TestGenerateAddIndexSQL_NonUnique(t *testing.T) {
	a := newAdapter()
	idx := types.SchemaIndex{Name: "idx_name", Table: "users", Columns: []string{"name"}, Unique: false}
	sql := a.GenerateAddIndexSQL(idx)
	if strings.Contains(sql, "UNIQUE") {
		t.Errorf("GenerateAddIndexSQL(non-unique) = %q, should not contain UNIQUE", sql)
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

// ── convertTypeToMySQL ────────────────────────────────────────────────────────

func TestConvertTypeToMySQL(t *testing.T) {
	a := newAdapter()
	cases := []struct{ in, want string }{
		{"SERIAL", "INT"},
		{"BIGSERIAL", "BIGINT"},
		{"SMALLSERIAL", "SMALLINT"},
		{"BOOLEAN", "TINYINT(1)"},
		{"BOOL", "TINYINT(1)"},
		{"TIMESTAMP WITH TIME ZONE", "TIMESTAMP"},
		{"TIMESTAMPTZ", "TIMESTAMP"},
		{"INT", "INT"},
		{"VARCHAR(255)", "VARCHAR(255)"},
		{"TEXT", "TEXT"},
	}
	for _, c := range cases {
		got := a.convertTypeToMySQL(c.in)
		if got != c.want {
			t.Errorf("convertTypeToMySQL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ── security hardening 

func TestMySQLAdapter_ProviderName(t *testing.T) {
	a := newAdapter()
	if got := a.ProviderName(); got != "mysql" {
		t.Errorf("ProviderName() = %q, want mysql", got)
	}
}

func TestMySQLAdapter_QuoteIdentifier(t *testing.T) {
	a := newAdapter()
	cases := []struct{ in, want string }{
		{"users", "`users`"},
		{"order_items", "`order_items`"},
		{"tab`le", "`tab``le`"}, // embedded backtick must be doubled
	}
	for _, c := range cases {
		got := a.QuoteIdentifier(c.in)
		if got != c.want {
			t.Errorf("QuoteIdentifier(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
