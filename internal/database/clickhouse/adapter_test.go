package clickhouse

import (
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func TestMapColumnType(t *testing.T) {
	a := New()
	cases := []struct{ in, want string }{
		{"String", "STRING"},
		{"UInt64", "UINT64"},
		{"Int32", "INT32"},
		{"Float64", "FLOAT64"},
		{"Nullable(String)", "STRING"},
		{"Nullable(Int64)", "INT64"},
		{"DateTime", "DATETIME"},
		{"UUID", "UUID"},
	}
	for _, c := range cases {
		if got := a.MapColumnType(c.in); got != c.want {
			t.Errorf("MapColumnType(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGenerateCreateTableSQL(t *testing.T) {
	a := New()
	table := types.SchemaTable{
		Name: "events",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "UInt64", IsPrimary: true},
			{Name: "name", Type: "String"},
			{Name: "ts", Type: "DateTime", Nullable: true},
		},
	}
	sql := a.GenerateCreateTableSQL(table)

	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS `events`",
		"ENGINE = MergeTree()",
		"ORDER BY (`id`)",
		"`id` UInt64",
		"`name` String",
		"Nullable(DateTime)",
	} {
		if !contains(sql, want) {
			t.Errorf("missing %q in:\n%s", want, sql)
		}
	}
}

func TestGenerateAddColumnSQL(t *testing.T) {
	a := New()
	col := types.SchemaColumn{Name: "email", Type: "String"}
	sql := a.GenerateAddColumnSQL("users", col)
	want := "ALTER TABLE `users` ADD COLUMN IF NOT EXISTS `email` String;"
	if sql != want {
		t.Errorf("got %q, want %q", sql, want)
	}
}

func TestGenerateDropColumnSQL(t *testing.T) {
	a := New()
	sql := a.GenerateDropColumnSQL("users", "email")
	want := "ALTER TABLE `users` DROP COLUMN IF EXISTS `email`;"
	if sql != want {
		t.Errorf("got %q, want %q", sql, want)
	}
}

func TestGenerateAlterColumnSQL(t *testing.T) {
	a := New()
	col := types.SchemaColumn{Name: "age", Type: "Int32"}
	sql := a.GenerateAlterColumnSQL("users", col, "Int16")
	want := "ALTER TABLE `users` MODIFY COLUMN `age` Int32;"
	if sql != want {
		t.Errorf("got %q, want %q", sql, want)
	}
	// same type → no-op
	if got := a.GenerateAlterColumnSQL("users", col, "Int32"); got != "" {
		t.Errorf("expected empty string for no-op, got %q", got)
	}
}

func TestGenerateIndexSQL(t *testing.T) {
	a := New()
	idx := types.SchemaIndex{Name: "idx_name", Table: "users", Columns: []string{"name"}}
	sql := a.GenerateAddIndexSQL(idx)
	if !contains(sql, "ADD INDEX") || !contains(sql, "minmax") {
		t.Errorf("unexpected index SQL: %s", sql)
	}

	uniqueIdx := types.SchemaIndex{Name: "idx_email", Table: "users", Columns: []string{"email"}, Unique: true}
	sqlUniq := a.GenerateAddIndexSQL(uniqueIdx)
	if !contains(sqlUniq, "bloom_filter") {
		t.Errorf("unique index should use bloom_filter, got: %s", sqlUniq)
	}

	dropSQL := a.GenerateDropIndexSQL(idx)
	want := "ALTER TABLE `users` DROP INDEX IF EXISTS `idx_name`;"
	if dropSQL != want {
		t.Errorf("got %q, want %q", dropSQL, want)
	}
}

func TestProviderName(t *testing.T) {
	if got := New().ProviderName(); got != "clickhouse" {
		t.Errorf("got %q, want \"clickhouse\"", got)
	}
}

func TestFormatColumnType_Nullable(t *testing.T) {
	a := New()
	col := types.SchemaColumn{Name: "x", Type: "String", Nullable: true}
	if got := a.FormatColumnType(col); got != "Nullable(String)" {
		t.Errorf("got %q", got)
	}
	// already wrapped → no double-wrap
	col2 := types.SchemaColumn{Name: "x", Type: "Nullable(String)", Nullable: true}
	if got := a.FormatColumnType(col2); got != "Nullable(String)" {
		t.Errorf("got %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
