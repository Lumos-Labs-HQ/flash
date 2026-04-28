package postgres

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
		{"character varying", "VARCHAR"},
		{"varchar", "VARCHAR"},
		{"integer", "INTEGER"},
		{"int4", "INTEGER"},
		{"bigint", "BIGINT"},
		{"boolean", "BOOLEAN"},
		{"bool", "BOOLEAN"},
		{"text", "TEXT"},
		{"uuid", "UUID"},
		{"jsonb", "JSONB"},
		{"timestamp with time zone", "TIMESTAMP WITH TIME ZONE"},
		{"timestamptz", "TIMESTAMP WITH TIME ZONE"},
		{"UNKNOWN_TYPE", "UNKNOWN_TYPE"},
	}
	for _, c := range cases {
		got := a.MapColumnType(c.in)
		if got != c.want {
			t.Errorf("MapColumnType(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ── FormatColumnType ──────────────────────────────────────────────────────────

func TestFormatColumnType_PrimaryKey(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "id", Type: "SERIAL", IsPrimary: true}
	got := a.FormatColumnType(col)
	if !strings.Contains(got, "PRIMARY KEY") {
		t.Errorf("FormatColumnType(PK) = %q, missing PRIMARY KEY", got)
	}
}

func TestFormatColumnType_NotNull(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "email", Type: "TEXT", Nullable: false}
	got := a.FormatColumnType(col)
	if !strings.Contains(got, "NOT NULL") {
		t.Errorf("FormatColumnType(NOT NULL) = %q, missing NOT NULL", got)
	}
}

func TestFormatColumnType_Unique(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "email", Type: "TEXT", IsUnique: true, Nullable: false}
	got := a.FormatColumnType(col)
	if !strings.Contains(got, "UNIQUE") {
		t.Errorf("FormatColumnType(UNIQUE) = %q, missing UNIQUE", got)
	}
}

func TestFormatColumnType_ForeignKey(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{
		Name: "user_id", Type: "INTEGER",
		ForeignKeyTable: "users", ForeignKeyColumn: "id",
		OnDeleteAction: "CASCADE", Nullable: false,
	}
	got := a.FormatColumnType(col)
	if !strings.Contains(got, "REFERENCES") || !strings.Contains(got, "CASCADE") {
		t.Errorf("FormatColumnType(FK) = %q", got)
	}
}

func TestFormatColumnType_Default(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "active", Type: "BOOLEAN", Default: "true", Nullable: true}
	got := a.FormatColumnType(col)
	if !strings.Contains(got, "DEFAULT true") {
		t.Errorf("FormatColumnType(DEFAULT) = %q, missing DEFAULT true", got)
	}
}

// ── GenerateCreateTableSQL ────────────────────────────────────────────────────

func TestGenerateCreateTableSQL_Basic(t *testing.T) {
	a := newAdapter()
	table := types.SchemaTable{
		Name: "users",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "SERIAL", IsPrimary: true},
			{Name: "email", Type: "TEXT", Nullable: false},
		},
	}
	sql := a.GenerateCreateTableSQL(table)
	for _, want := range []string{`CREATE TABLE IF NOT EXISTS "users"`, `"id"`, `"email"`, "PRIMARY KEY"} {
		if !strings.Contains(sql, want) {
			t.Errorf("GenerateCreateTableSQL missing %q:\n%s", want, sql)
		}
	}
}

func TestGenerateCreateTableSQL_WithFK(t *testing.T) {
	a := newAdapter()
	table := types.SchemaTable{
		Name: "posts",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "SERIAL", IsPrimary: true},
			{Name: "user_id", Type: "INTEGER", ForeignKeyTable: "users", ForeignKeyColumn: "id", OnDeleteAction: "CASCADE"},
		},
	}
	sql := a.GenerateCreateTableSQL(table)
	if !strings.Contains(sql, "FOREIGN KEY") || !strings.Contains(sql, "CASCADE") {
		t.Errorf("GenerateCreateTableSQL(FK) missing FK clause:\n%s", sql)
	}
}

// ── GenerateAddColumnSQL ──────────────────────────────────────────────────────

func TestGenerateAddColumnSQL(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "phone", Type: "TEXT", Nullable: true}
	sql := a.GenerateAddColumnSQL("users", col)
	if !strings.Contains(sql, `ALTER TABLE "users"`) || !strings.Contains(sql, `"phone"`) {
		t.Errorf("GenerateAddColumnSQL = %q", sql)
	}
	if !strings.Contains(sql, "IF NOT EXISTS") {
		t.Errorf("GenerateAddColumnSQL missing IF NOT EXISTS: %q", sql)
	}
}

// ── GenerateDropColumnSQL ─────────────────────────────────────────────────────

func TestGenerateDropColumnSQL(t *testing.T) {
	a := newAdapter()
	sql := a.GenerateDropColumnSQL("users", "phone")
	if !strings.Contains(sql, `ALTER TABLE "users"`) || !strings.Contains(sql, `"phone"`) {
		t.Errorf("GenerateDropColumnSQL = %q", sql)
	}
	if !strings.Contains(sql, "DROP COLUMN") {
		t.Errorf("GenerateDropColumnSQL missing DROP COLUMN: %q", sql)
	}
}

// ── GenerateAddIndexSQL ───────────────────────────────────────────────────────

func TestGenerateAddIndexSQL_Unique(t *testing.T) {
	a := newAdapter()
	idx := types.SchemaIndex{Name: "idx_users_email", Table: "users", Columns: []string{"email"}, Unique: true}
	sql := a.GenerateAddIndexSQL(idx)
	if !strings.Contains(sql, "UNIQUE INDEX") {
		t.Errorf("GenerateAddIndexSQL(unique) = %q, missing UNIQUE INDEX", sql)
	}
	if !strings.Contains(sql, `"idx_users_email"`) {
		t.Errorf("GenerateAddIndexSQL missing index name: %q", sql)
	}
}

func TestGenerateAddIndexSQL_NonUnique(t *testing.T) {
	a := newAdapter()
	idx := types.SchemaIndex{Name: "idx_posts_user", Table: "posts", Columns: []string{"user_id"}, Unique: false}
	sql := a.GenerateAddIndexSQL(idx)
	if strings.Contains(sql, "UNIQUE") {
		t.Errorf("GenerateAddIndexSQL(non-unique) should not contain UNIQUE: %q", sql)
	}
}

// ── GenerateDropIndexSQL ──────────────────────────────────────────────────────

func TestGenerateDropIndexSQL(t *testing.T) {
	a := newAdapter()
	idx := types.SchemaIndex{Name: "idx_users_email", Table: "users"}
	sql := a.GenerateDropIndexSQL(idx)
	if !strings.Contains(sql, "DROP INDEX") || !strings.Contains(sql, "idx_users_email") {
		t.Errorf("GenerateDropIndexSQL = %q", sql)
	}
}
