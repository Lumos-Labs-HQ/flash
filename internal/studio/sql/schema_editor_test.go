package sql

import (
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/database/sqlite"
	"context"
)

// newTestService creates a Service backed by an in-memory SQLite adapter.
func newTestService(t *testing.T) *Service {
	t.Helper()
	a := sqlite.New()
	if err := a.Connect(context.Background(), ":memory:"); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { a.Close() })
	return NewService(a, &config.Config{Database: config.Database{Provider: "sqlite", URLEnv: "STUDIO_DB_URL"}})
}

// ── generateSQL ───────────────────────────────────────────────────────────────

func TestGenerateSQL_AddColumn(t *testing.T) {
	s := newTestService(t)
	change := &SchemaChange{
		Type:  "add_column",
		Table: "users",
		Column: &ColumnChange{
			Name:     "phone",
			Type:     "TEXT",
			Nullable: true,
		},
	}
	sql := s.generateSQL(change)
	if !strings.Contains(sql, "ALTER TABLE") || !strings.Contains(sql, "phone") {
		t.Errorf("add_column SQL = %q", sql)
	}
}

func TestGenerateSQL_DropColumn(t *testing.T) {
	s := newTestService(t)
	change := &SchemaChange{
		Type:   "drop_column",
		Table:  "users",
		Column: &ColumnChange{Name: "phone"},
	}
	sql := s.generateSQL(change)
	if !strings.Contains(sql, "DROP COLUMN") || !strings.Contains(sql, "phone") {
		t.Errorf("drop_column SQL = %q", sql)
	}
}

func TestGenerateSQL_DropTable(t *testing.T) {
	s := newTestService(t)
	change := &SchemaChange{Type: "drop_table", Table: "old_table"}
	sql := s.generateSQL(change)
	if !strings.Contains(sql, "DROP TABLE") || !strings.Contains(sql, "old_table") {
		t.Errorf("drop_table SQL = %q", sql)
	}
}

func TestGenerateSQL_CreateTable(t *testing.T) {
	s := newTestService(t)
	change := &SchemaChange{
		Type:  "create_table",
		Table: "products",
		TableDef: &TableDefinition{
			Name: "products",
			Columns: []ColumnChange{
				{Name: "id", Type: "INTEGER", IsPrimary: true},
				{Name: "name", Type: "TEXT", Nullable: false},
			},
		},
	}
	sql := s.generateSQL(change)
	if !strings.Contains(sql, "CREATE TABLE") || !strings.Contains(sql, "products") {
		t.Errorf("create_table SQL = %q", sql)
	}
}

func TestGenerateSQL_DropEnum(t *testing.T) {
	s := newTestService(t)
	change := &SchemaChange{Type: "drop_enum", EnumName: "status"}
	sql := s.generateSQL(change)
	if !strings.Contains(sql, "DROP TYPE") || !strings.Contains(sql, "status") {
		t.Errorf("drop_enum SQL = %q", sql)
	}
}

func TestGenerateSQL_RawSQL(t *testing.T) {
	s := newTestService(t)
	raw := "SELECT 1;"
	change := &SchemaChange{Type: "raw", SQL: raw}
	sql := s.generateSQL(change)
	if sql != raw {
		t.Errorf("raw SQL = %q, want %q", sql, raw)
	}
}

// ── sanitizeDefaultValue ──────────────────────────────────────────────────────

func TestSanitizeDefaultValue_Empty(t *testing.T) {
	if got := sanitizeDefaultValue("", "TEXT"); got != "" {
		t.Errorf("empty default = %q, want empty", got)
	}
}

func TestSanitizeDefaultValue_Null(t *testing.T) {
	if got := sanitizeDefaultValue("NULL", "TEXT"); got != "NULL" {
		t.Errorf("NULL default = %q, want NULL", got)
	}
}

func TestSanitizeDefaultValue_SQLFunction(t *testing.T) {
	for _, fn := range []string{"NOW()", "CURRENT_TIMESTAMP", "TRUE", "FALSE"} {
		got := sanitizeDefaultValue(fn, "TEXT")
		if got != fn {
			t.Errorf("sanitizeDefaultValue(%q) = %q, want %q", fn, got, fn)
		}
	}
}

func TestSanitizeDefaultValue_NumericType(t *testing.T) {
	got := sanitizeDefaultValue("42", "INTEGER")
	if got != "42" {
		t.Errorf("numeric default = %q, want 42", got)
	}
}

func TestSanitizeDefaultValue_StringWrapped(t *testing.T) {
	got := sanitizeDefaultValue("hello", "TEXT")
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("string default = %q, want quoted", got)
	}
}

func TestSanitizeDefaultValue_AlreadyQuoted(t *testing.T) {
	got := sanitizeDefaultValue("'already quoted'", "TEXT")
	if got != "'already quoted'" {
		t.Errorf("already-quoted default = %q", got)
	}
}

func TestSanitizeDefaultValue_SQLInjectionEscaped(t *testing.T) {
	got := sanitizeDefaultValue("it's a test", "TEXT")
	if strings.Contains(got, "it's") && !strings.Contains(got, "it''s") {
		t.Errorf("single quote not escaped: %q", got)
	}
}

// ── PreviewSchemaChange ───────────────────────────────────────────────────────

func TestPreviewSchemaChange(t *testing.T) {
	s := newTestService(t)
	change := &SchemaChange{
		Type:   "add_column",
		Table:  "users",
		Column: &ColumnChange{Name: "bio", Type: "TEXT", Nullable: true},
	}
	preview, err := s.PreviewSchemaChange(change)
	if err != nil {
		t.Fatalf("PreviewSchemaChange error: %v", err)
	}
	if preview.SQL == "" {
		t.Error("preview SQL should not be empty")
	}
	if len(preview.Changes) == 0 {
		t.Error("preview Changes should not be empty")
	}
}
