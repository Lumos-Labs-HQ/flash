package rustgen

import (
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

func newGen(provider string) *Generator {
	cfg := &config.Config{
		SchemaDir: "db/schema",
		Queries:   "db/queries/",
		Database:  config.Database{Provider: provider},
		Gen: config.Gen{
			Rust: config.RustGen{
				Enabled: true,
				Out:     "src/flash_gen",
				Driver:  "sqlx",
			},
		},
	}
	cfg.Gen.Rust.Driver = "sqlx"
	return &Generator{
		Config: cfg,
		schema: &parser.Schema{
			Tables: []*parser.Table{
				{
					Name: "users",
					Columns: []*parser.Column{
						{Name: "id", Type: "SERIAL", Nullable: false},
						{Name: "name", Type: "VARCHAR(255)", Nullable: false},
						{Name: "email", Type: "VARCHAR(255)", Nullable: false},
						{Name: "created_at", Type: "TIMESTAMPTZ", Nullable: false},
					},
				},
			},
			Enums: []*parser.Enum{
				{Name: "user_role", Values: []string{"admin", "user", "moderator"}},
			},
		},
	}
}

func TestSQLTypeToRust_BasicTypes(t *testing.T) {
	g := newGen("postgresql")

	tests := []struct {
		sqlType  string
		nullable bool
		want     string
	}{
		{"INTEGER", false, "i32"},
		{"INTEGER", true, "Option<i32>"},
		{"BIGINT", false, "i64"},
		{"SMALLINT", false, "i16"},
		{"TEXT", false, "String"},
		{"VARCHAR(255)", false, "String"},
		{"BOOLEAN", false, "bool"},
		{"BOOLEAN", true, "Option<bool>"},
		{"UUID", false, "Uuid"},
		{"UUID", true, "Option<Uuid>"},
		{"TIMESTAMPTZ", false, "DateTime<Utc>"},
		{"TIMESTAMP", false, "NaiveDateTime"},
		{"JSONB", false, "serde_json::Value"},
		{"JSONB", true, "Option<serde_json::Value>"},
		{"BYTEA", false, "Vec<u8>"},
		{"REAL", false, "f32"},
		{"DOUBLE PRECISION", false, "f64"},
		{"NUMERIC", false, "Decimal"},
		{"SERIAL", false, "i32"},
		{"BIGSERIAL", false, "i64"},
	}

	for _, tt := range tests {
		got := g.sqlTypeToRust(tt.sqlType, tt.nullable)
		if got != tt.want {
			t.Errorf("sqlTypeToRust(%q, %v) = %q, want %q", tt.sqlType, tt.nullable, got, tt.want)
		}
	}
}

func TestSQLTypeToRust_ArrayType(t *testing.T) {
	g := newGen("postgresql")

	got := g.sqlTypeToRust("TEXT[]", false)
	if got != "Vec<String>" {
		t.Errorf("sqlTypeToRust(TEXT[]) = %q, want Vec<String>", got)
	}

	got = g.sqlTypeToRust("INTEGER[]", true)
	if got != "Option<Vec<i32>>" {
		t.Errorf("sqlTypeToRust(INTEGER[], nullable) = %q, want Option<Vec<i32>>", got)
	}
}

func TestSQLTypeToRust_EnumType(t *testing.T) {
	g := newGen("postgresql")

	got := g.sqlTypeToRust("user_role", false)
	if got != "UserRole" {
		t.Errorf("sqlTypeToRust(user_role) = %q, want UserRole", got)
	}

	got = g.sqlTypeToRust("user_role", true)
	if got != "Option<UserRole>" {
		t.Errorf("sqlTypeToRust(user_role, nullable) = %q, want Option<UserRole>", got)
	}
}

func TestConvertSQL_PostgreSQL_NoChange(t *testing.T) {
	g := newGen("postgresql")
	sql := "SELECT id, name FROM users WHERE id = $1"
	got := g.convertSQL(sql)
	if got != sql {
		t.Errorf("convertSQL(postgres) = %q, want %q", got, sql)
	}
}

func TestConvertSQL_MySQL_QuestionMark(t *testing.T) {
	g := newGen("mysql")
	sql := "SELECT id, name FROM users WHERE id = $1 AND email = $2"
	want := "SELECT id, name FROM users WHERE id = ? AND email = ?"
	got := g.convertSQL(sql)
	if got != want {
		t.Errorf("convertSQL(mysql) = %q, want %q", got, want)
	}
}

func TestConvertSQL_SQLite_QuestionMark(t *testing.T) {
	g := newGen("sqlite")
	sql := "SELECT id, name FROM users WHERE id = $1"
	want := "SELECT id, name FROM users WHERE id = ?"
	got := g.convertSQL(sql)
	if got != want {
		t.Errorf("convertSQL(sqlite) = %q, want %q", got, want)
	}
}

func TestParamTypeToRust(t *testing.T) {
	g := newGen("postgresql")

	tests := []struct {
		sqlType string
		want    string
	}{
		{"TEXT", "&str"},
		{"VARCHAR(255)", "&str"},
		{"INTEGER", "i32"},
		{"BIGINT", "i64"},
		{"UUID", "Uuid"},
		{"BYTEA", "&[u8]"},
		{"BOOLEAN", "bool"},
	}

	for _, tt := range tests {
		got := g.paramTypeToRust(tt.sqlType)
		if got != tt.want {
			t.Errorf("paramTypeToRust(%q) = %q, want %q", tt.sqlType, got, tt.want)
		}
	}
}

func TestToRustModuleName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users.sql", "users"},
		{"user_queries.sql", "user_queries"},
		{"UserPosts.sql", "user_posts"},
		{"path/to/MyQueries.sql", "my_queries"},
	}

	for _, tt := range tests {
		got := toRustModuleName(tt.input)
		if got != tt.want {
			t.Errorf("toRustModuleName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToRustFnName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GetUser", "get_user"},
		{"CreateUser", "create_user"},
		{"getUserByEmail", "get_user_by_email"},
	}

	for _, tt := range tests {
		got := toRustFnName(tt.input)
		if got != tt.want {
			t.Errorf("toRustFnName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGetPoolType(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{"postgresql", "sqlx::PgPool"},
		{"mysql", "sqlx::MySqlPool"},
		{"sqlite", "sqlx::SqlitePool"},
	}

	for _, tt := range tests {
		g := newGen(tt.provider)
		got := g.getPoolType()
		if got != tt.want {
			t.Errorf("getPoolType(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

func TestEscapeSQLForRust(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`SELECT "id" FROM users`, `SELECT \"id\" FROM users`},
		{"SELECT id\n  FROM users", "SELECT id FROM users"},
		{`SELECT id FROM users WHERE name = 'te\"st'`, `SELECT id FROM users WHERE name = 'te\\\"st'`},
	}

	for _, tt := range tests {
		got := escapeSQLForRust(tt.input)
		if got != tt.want {
			t.Errorf("escapeSQLForRust(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
