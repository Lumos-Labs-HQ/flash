package javagen

import (
	"strings"
	"testing"

	"os"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

func newGen(provider string) *Generator {
	return New(&config.Config{
		SchemaDir: "db/schema",
		Queries:   "db/queries/",
		Database:  config.Database{Provider: provider},
		Gen:       config.Gen{Java: config.JavaGen{Enabled: true, Out: "flash_gen"}},
	})
}

// ── sqlTypeToJava ─────────────────────────────────────────────────────────────

func TestSQLTypeToJava_BasicTypes(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}

	cases := []struct {
		sql      string
		nullable bool
		want     string
	}{
		{"INTEGER", false, "int"},
		{"INTEGER", true, "Integer"},
		{"BIGINT", false, "long"},
		{"BIGINT", true, "Long"},
		{"SERIAL", false, "int"},
		{"TEXT", false, "String"},
		{"VARCHAR(255)", false, "String"},
		{"BOOLEAN", false, "boolean"},
		{"BOOLEAN", true, "Boolean"},
		{"TIMESTAMP", false, "LocalDateTime"},
		{"DATE", false, "LocalDateTime"},
		{"FLOAT", false, "float"},
		{"DOUBLE PRECISION", false, "double"},
		{"NUMERIC", false, "double"},
		{"DECIMAL", false, "double"},
		{"UUID", false, "UUID"},
		{"JSONB", false, "java.util.Map<String, Object>"},
		{"BYTEA", false, "byte[]"},
	}

	for _, c := range cases {
		got := g.sqlTypeToJava(c.sql, c.nullable)
		if got != c.want {
			t.Errorf("sqlTypeToJava(%q, nullable=%v) = %q, want %q", c.sql, c.nullable, got, c.want)
		}
	}
}

func TestSQLTypeToJava_ArrayType(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}
	got := g.sqlTypeToJava("TEXT[]", false)
	if got != "java.util.List<String>" {
		t.Errorf("sqlTypeToJava(TEXT[]) = %q, want java.util.List<String>", got)
	}
}

func TestSQLTypeToJava_EnumType(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{
		Enums: []*parser.Enum{{Name: "status", Values: []string{"active", "inactive"}}},
	}
	got := g.sqlTypeToJava("status", false)
	if got != "Status" {
		t.Errorf("sqlTypeToJava(enum) = %q, want Status", got)
	}
}

func TestSQLTypeToJava_ClickHouseTypes(t *testing.T) {
	g := newGen("clickhouse")
	g.schema = &parser.Schema{}
	cases := []struct{ sql, want string }{
		{"UInt64", "long"},
		{"Int32", "int"},
		{"Float64", "double"},
		{"String", "String"},
	}
	for _, c := range cases {
		got := g.sqlTypeToJava(c.sql, false)
		if got != c.want {
			t.Errorf("sqlTypeToJava(%q) = %q, want %q", c.sql, got, c.want)
		}
	}
}

// ── modelTypeForQuery ─────────────────────────────────────────────────────────

func TestModelTypeForQuery_MatchesTable(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{
		Tables: []*parser.Table{{
			Name:    "users",
			Columns: []*parser.Column{{Name: "id"}, {Name: "email"}},
		}},
	}
	cols := []*parser.QueryColumn{{Name: "id"}, {Name: "email"}}
	query := &parser.Query{SQL: "SELECT id, email FROM users"}
	got := g.modelTypeForQuery(query, cols)
	if got != "Users" {
		t.Errorf("modelTypeForQuery = %q, want Users", got)
	}
}

func TestModelTypeForQuery_ColumnMismatch(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{
		Tables: []*parser.Table{{
			Name:    "users",
			Columns: []*parser.Column{{Name: "id"}, {Name: "email"}},
		}},
	}
	cols := []*parser.QueryColumn{{Name: "id"}}
	query := &parser.Query{SQL: "SELECT id FROM users"}
	got := g.modelTypeForQuery(query, cols)
	if got != "" {
		t.Errorf("partial columns should return empty, got %q", got)
	}
}

// ── expandWildcardColumns ─────────────────────────────────────────────────────

func TestExpandWildcardColumns_Expands(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{
		Tables: []*parser.Table{{
			Name: "products",
			Columns: []*parser.Column{
				{Name: "id", Type: "INTEGER"},
				{Name: "name", Type: "TEXT"},
				{Name: "price", Type: "DECIMAL"},
			},
		}},
	}
	query := &parser.Query{
		SQL:     "SELECT * FROM products",
		Columns: []*parser.QueryColumn{{Name: "*", Table: "products"}},
	}
	got := g.expandWildcardColumns(query)
	if len(got) != 3 {
		t.Errorf("expanded = %d cols, want 3", len(got))
	}
}

// ── javaPackage ───────────────────────────────────────────────────────────────

func TestJavaPackage(t *testing.T) {
	cases := []struct{ pkg, out, want string }{
		{"", "flash_gen", "flash_gen"},
		{"", "my-gen", "my_gen"},
		{"com.example.db", "flash_gen", "com.example.db"},
	}
	for _, c := range cases {
		cfg := &config.JavaGen{Package: c.pkg, Out: c.out}
		got := javaPackage(cfg)
		if got != c.want {
			t.Errorf("javaPackage(pkg=%q out=%q) = %q, want %q", c.pkg, c.out, got, c.want)
		}
	}
}

// ── generateModels smoke test ─────────────────────────────────────────────────

func TestGenerateModels_Smoke(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{
		Tables: []*parser.Table{{
			Name: "orders",
			Columns: []*parser.Column{
				{Name: "id", Type: "BIGINT", Nullable: false},
				{Name: "total", Type: "DECIMAL", Nullable: false},
				{Name: "note", Type: "TEXT", Nullable: true},
			},
		}},
		Enums: []*parser.Enum{{Name: "state", Values: []string{"open", "closed"}}},
	}

	dir := t.TempDir()
	g.Config.Gen.Java.Out = dir

	if err := g.generateModels(); err != nil {
		t.Fatalf("generateModels: %v", err)
	}

	// Each type gets its own file
	record, err := os.ReadFile(dir + "/Orders.java")
	if err != nil {
		t.Fatalf("Orders.java not created: %v", err)
	}
	src := string(record)
	if !strings.Contains(src, "public record Orders(") {
		t.Error("missing record Orders")
	}
	if !strings.Contains(src, "long id") {
		t.Error("missing long field id")
	}
	if !strings.Contains(src, "double total") {
		t.Error("missing double field total")
	}
	if !strings.Contains(src, "String note") {
		t.Error("missing String field note")
	}

	enumFile, err := os.ReadFile(dir + "/State.java")
	if err != nil {
		t.Fatalf("State.java not created: %v", err)
	}
	enumSrc := string(enumFile)
	if !strings.Contains(enumSrc, "public enum State") {
		t.Error("missing enum State")
	}
	if !strings.Contains(enumSrc, "OPEN,") {
		t.Error("missing enum value OPEN,")
	}
	if !strings.Contains(enumSrc, "CLOSED;") {
		t.Error("missing enum value CLOSED;")
	}
}

func TestGenerateModels_UUIDImport(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{
		Tables: []*parser.Table{{
			Name:    "tokens",
			Columns: []*parser.Column{{Name: "id", Type: "UUID"}},
		}},
	}
	dir := t.TempDir()
	g.Config.Gen.Java.Out = dir
	if err := g.generateModels(); err != nil {
		t.Fatalf("generateModels: %v", err)
	}
	src, _ := os.ReadFile(dir + "/Tokens.java")
	if !strings.Contains(string(src), "import java.util.UUID;") {
		t.Error("missing UUID import")
	}
}

// ── generateQueryMethod smoke test ────────────────────────────────────────────

func TestGenerateQueryMethod_OneRow(t *testing.T) {
	g := newGen("postgresql")
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
		Name: "GetUser",
		SQL:  "SELECT id, email FROM users WHERE id = $1",
		Cmd:  ":one",
		Params: []*parser.Param{
			{Name: "id", Type: "INTEGER"},
		},
		Columns: []*parser.QueryColumn{
			{Name: "id", Type: "INTEGER"},
			{Name: "email", Type: "TEXT"},
		},
	}

	var w strings.Builder
	g.generateQueryMethod(&w, query)
	src := w.String()

	if !strings.Contains(src, "public Users getUser(") {
		t.Error("missing method signature: public Users getUser(")
	}
	if !strings.Contains(src, "int id") {
		t.Error("missing int param")
	}
	if !strings.Contains(src, "throws java.sql.SQLException") {
		t.Error("missing throws clause for JDBC")
	}
}

func TestGenerateQueryMethod_Many(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}

	query := &parser.Query{
		Name: "ListProducts",
		SQL:  "SELECT id, name FROM products",
		Cmd:  ":many",
		Columns: []*parser.QueryColumn{
			{Name: "id", Type: "INTEGER"},
			{Name: "name", Type: "TEXT"},
		},
	}

	var w strings.Builder
	g.generateQueryMethod(&w, query)
	src := w.String()

	if !strings.Contains(src, "java.util.List<") {
		t.Error("missing List return type")
	}
}

func TestGenerateQueryMethod_ExecResult(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}

	query := &parser.Query{
		Name:    "CreateUser",
		SQL:     "INSERT INTO users (email) VALUES ($1)",
		Cmd:     ":execresult",
		Params:  []*parser.Param{{Name: "email", Type: "TEXT"}},
		Columns: []*parser.QueryColumn{},
	}

	var w strings.Builder
	g.generateQueryMethod(&w, query)
	src := w.String()

	if !strings.Contains(src, "public long createUser(") {
		t.Error("execresult should return long")
	}
	if !strings.Contains(src, "getGeneratedKeys") {
		t.Error("execresult should fetch generated keys")
	}
}

func TestGenerateQueryMethod_Exec(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}

	query := &parser.Query{
		Name:    "DeleteUser",
		SQL:     "DELETE FROM users WHERE id = $1",
		Cmd:     ":exec",
		Params:  []*parser.Param{{Name: "id", Type: "INTEGER"}},
		Columns: []*parser.QueryColumn{},
	}

	var w strings.Builder
	g.generateQueryMethod(&w, query)
	src := w.String()

	if !strings.Contains(src, "public void deleteUser(") {
		t.Error("exec method should return void")
	}
}

func TestGenerateQueryMethod_Scylla(t *testing.T) {
	g := newGen("scylla")
	g.schema = &parser.Schema{}

	query := &parser.Query{
		Name:    "CreateEvent",
		SQL:     "INSERT INTO events (id) VALUES (?)",
		Cmd:     ":exec",
		Params:  []*parser.Param{{Name: "id", Type: "uuid"}},
		Columns: []*parser.QueryColumn{},
	}

	var w strings.Builder
	g.generateQueryMethod(&w, query)
	src := w.String()

	if !strings.Contains(src, "session.execute") {
		t.Error("Scylla method should call session.execute")
	}
	if strings.Contains(src, "throws java.sql.SQLException") {
		t.Error("Scylla method should not have JDBC throws clause")
	}
}

func TestGenerateQueryMethod_JooqDriver(t *testing.T) {
	g := New(&config.Config{
		Database: config.Database{Provider: "postgresql"},
		Gen:      config.Gen{Java: config.JavaGen{Enabled: true, Out: "flash_gen", Driver: "jooq"}},
	})
	g.schema = &parser.Schema{}

	query := &parser.Query{
		Name:    "ListAll",
		SQL:     "SELECT id FROM users",
		Cmd:     ":many",
		Columns: []*parser.QueryColumn{{Name: "id", Type: "INTEGER"}},
	}

	var w strings.Builder
	g.generateQueryMethod(&w, query)
	src := w.String()

	if !strings.Contains(src, "ctx.fetch(") {
		t.Error("jOOQ method should use ctx.fetch")
	}
}

func TestGenerateQueryMethod_MySQLParamConversion(t *testing.T) {
	g := newGen("mysql")
	g.schema = &parser.Schema{}

	query := &parser.Query{
		Name:    "GetUser",
		SQL:     "SELECT id FROM users WHERE id = $1",
		Cmd:     ":one",
		Params:  []*parser.Param{{Name: "id", Type: "INTEGER"}},
		Columns: []*parser.QueryColumn{{Name: "id", Type: "INTEGER"}},
	}

	var w strings.Builder
	g.generateQueryMethod(&w, query)
	src := w.String()

	// $1 should be converted to ? for JDBC
	if strings.Contains(src, "$1") {
		t.Error("MySQL JDBC SQL should have $1 converted to ?")
	}
	if !strings.Contains(src, "?") {
		t.Error("MySQL JDBC SQL should contain ?")
	}
}
