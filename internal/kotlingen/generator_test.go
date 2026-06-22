package kotlingen

import (
	"os"
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
		Gen:       config.Gen{Kotlin: config.KotlinGen{Enabled: true, Out: "flash_gen"}},
	})
}

// ── sqlTypeToKotlin ───────────────────────────────────────────────────────────

func TestSQLTypeToKotlin_BasicTypes(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}

	cases := []struct {
		sql      string
		nullable bool
		want     string
	}{
		{"INTEGER", false, "Int"},
		{"BIGINT", false, "Long"},
		{"SERIAL", false, "Int"},
		{"TEXT", false, "String"},
		{"VARCHAR(255)", false, "String"},
		{"BOOLEAN", false, "Boolean"},
		{"TIMESTAMP", false, "LocalDateTime"},
		{"DATE", false, "LocalDateTime"},
		{"FLOAT", false, "Float"},
		{"DOUBLE PRECISION", false, "Double"},
		{"NUMERIC", false, "Double"},
		{"DECIMAL", false, "Double"},
		{"UUID", false, "UUID"},
		{"JSONB", false, "String"},
		{"BYTEA", false, "ByteArray"},
		// nullable
		{"INTEGER", true, "Int?"},
		{"TEXT", true, "String?"},
		{"BOOLEAN", true, "Boolean?"},
		{"UUID", true, "UUID?"},
	}

	for _, c := range cases {
		got := g.sqlTypeToKotlin(c.sql, c.nullable)
		if got != c.want {
			t.Errorf("sqlTypeToKotlin(%q, nullable=%v) = %q, want %q", c.sql, c.nullable, got, c.want)
		}
	}
}

func TestSQLTypeToKotlin_ArrayType(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}
	got := g.sqlTypeToKotlin("TEXT[]", false)
	if got != "List<String>" {
		t.Errorf("sqlTypeToKotlin(TEXT[]) = %q, want List<String>", got)
	}
}

func TestSQLTypeToKotlin_EnumType(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{
		Enums: []*parser.Enum{{Name: "status", Values: []string{"active", "inactive"}}},
	}
	got := g.sqlTypeToKotlin("status", false)
	if got != "Status" {
		t.Errorf("sqlTypeToKotlin(enum) = %q, want Status", got)
	}
	gotNullable := g.sqlTypeToKotlin("status", true)
	if gotNullable != "Status?" {
		t.Errorf("sqlTypeToKotlin(enum nullable) = %q, want Status?", gotNullable)
	}
}

func TestSQLTypeToKotlin_ScyllaCollections(t *testing.T) {
	g := newGen("scylla")
	g.schema = &parser.Schema{}
	cases := []struct{ sql, want string }{
		{"set<text>", "Set<String>"},
		{"list<int>", "List<Int>"},
		{"map<text, int>", "Map<String, Int>"},
		{"uuid", "UUID"},
		{"set<uuid>", "Set<java.util.UUID>"},
		{"list<uuid>", "List<java.util.UUID>"},
	}
	for _, c := range cases {
		got := g.sqlTypeToKotlin(c.sql, false)
		if got != c.want {
			t.Errorf("sqlTypeToKotlin(%q) = %q, want %q", c.sql, got, c.want)
		}
	}
}

func TestSQLTypeToKotlin_ClickHouseTypes(t *testing.T) {
	g := newGen("clickhouse")
	g.schema = &parser.Schema{}
	cases := []struct{ sql, want string }{
		{"UInt64", "Long"},
		{"Int32", "Int"},
		{"Float64", "Double"},
		{"String", "String"},
	}
	for _, c := range cases {
		got := g.sqlTypeToKotlin(c.sql, false)
		if got != c.want {
			t.Errorf("sqlTypeToKotlin(%q) = %q, want %q", c.sql, got, c.want)
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
	query := &parser.Query{SQL: "SELECT id, email FROM users"}
	cols := []*parser.QueryColumn{{Name: "id"}, {Name: "email"}}
	got := g.modelTypeForQuery(query, cols)
	if got != "Users" {
		t.Errorf("modelTypeForQuery = %q, want Users", got)
	}
}

func TestModelTypeForQuery_NoMatch(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{
		Tables: []*parser.Table{{
			Name:    "users",
			Columns: []*parser.Column{{Name: "id"}, {Name: "email"}},
		}},
	}
	query := &parser.Query{SQL: "SELECT id FROM users"}
	cols := []*parser.QueryColumn{{Name: "id"}}
	got := g.modelTypeForQuery(query, cols)
	if got != "" {
		t.Errorf("partial column match should return empty, got %q", got)
	}
}

// ── expandWildcardColumns ─────────────────────────────────────────────────────

func TestExpandWildcardColumns_Expands(t *testing.T) {
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
		SQL:     "SELECT * FROM users",
		Columns: []*parser.QueryColumn{{Name: "*", Table: "users"}},
	}
	got := g.expandWildcardColumns(query)
	if len(got) != 2 {
		t.Errorf("expanded = %d cols, want 2", len(got))
	}
}

func TestExpandWildcardColumns_NonWildcard(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}
	query := &parser.Query{
		SQL:     "SELECT id FROM users",
		Columns: []*parser.QueryColumn{{Name: "id", Type: "INTEGER"}},
	}
	got := g.expandWildcardColumns(query)
	if len(got) != 1 || got[0].Name != "id" {
		t.Errorf("non-wildcard should pass through unchanged: %v", got)
	}
}

// ── toKotlinPackage ───────────────────────────────────────────────────────────

func TestToKotlinPackage(t *testing.T) {
	cases := []struct{ pkg, out, want string }{
		{"", "flash_gen", "flash_gen"},
		{"", "my-gen", "my_gen"},
		{"com.example.db", "flash_gen", "com.example.db"},
	}
	for _, c := range cases {
		cfg := &config.KotlinGen{Package: c.pkg, Out: c.out}
		got := toKotlinPackage(cfg)
		if got != c.want {
			t.Errorf("toKotlinPackage(pkg=%q out=%q) = %q, want %q", c.pkg, c.out, got, c.want)
		}
	}
}

// ── generateModels smoke test ─────────────────────────────────────────────────

func TestGenerateModels_Smoke(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{
		Tables: []*parser.Table{{
			Name: "users",
			Columns: []*parser.Column{
				{Name: "id", Type: "SERIAL", Nullable: false},
				{Name: "email", Type: "TEXT", Nullable: false},
				{Name: "bio", Type: "TEXT", Nullable: true},
			},
		}},
		Enums: []*parser.Enum{{Name: "role", Values: []string{"admin", "user"}}},
	}

	dir := t.TempDir()
	g.Config.Gen.Kotlin.Out = dir

	if err := g.generateModels(); err != nil {
		t.Fatalf("generateModels: %v", err)
	}

	data, err := os.ReadFile(dir + "/Models.kt")
	if err != nil {
		t.Fatalf("Models.kt not created: %v", err)
	}
	src := string(data)

	if !strings.Contains(src, "data class Users(") {
		t.Error("missing data class Users")
	}
	if !strings.Contains(src, "val email: String") {
		t.Error("missing non-nullable String field")
	}
	if !strings.Contains(src, "val bio: String?") {
		t.Error("missing nullable String? field")
	}
	if !strings.Contains(src, "enum class Role") {
		t.Error("missing enum class Role")
	}
	if !strings.Contains(src, "ADMIN") {
		t.Error("missing enum value ADMIN")
	}
}

func TestGenerateModels_ScyllaUUID(t *testing.T) {
	g := newGen("scylla")
	g.schema = &parser.Schema{
		Tables: []*parser.Table{{
			Name:    "events",
			Columns: []*parser.Column{{Name: "id", Type: "uuid"}},
		}},
	}
	dir := t.TempDir()
	g.Config.Gen.Kotlin.Out = dir
	if err := g.generateModels(); err != nil {
		t.Fatalf("generateModels: %v", err)
	}
	src, _ := os.ReadFile(dir + "/Models.kt")
	if !strings.Contains(string(src), "import java.util.UUID") {
		t.Error("missing UUID import for uuid column")
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

	if !strings.Contains(src, "fun getUser(") {
		t.Error("missing method getUser")
	}
	if !strings.Contains(src, "id: Int") {
		t.Error("missing Int param")
	}
	if !strings.Contains(src, "Users?") {
		t.Error("missing nullable return type")
	}
}

func TestGenerateQueryMethod_Many(t *testing.T) {
	g := newGen("postgresql")
	g.schema = &parser.Schema{}

	query := &parser.Query{
		Name: "ListUsers",
		SQL:  "SELECT id, email FROM users",
		Cmd:  ":many",
		Columns: []*parser.QueryColumn{
			{Name: "id", Type: "INTEGER"},
			{Name: "email", Type: "TEXT"},
		},
	}

	var w strings.Builder
	g.generateQueryMethod(&w, query)
	src := w.String()

	if !strings.Contains(src, "fun listUsers(") {
		t.Error("missing method listUsers")
	}
	if !strings.Contains(src, "List<") {
		t.Error("missing List return type")
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

	if !strings.Contains(src, ": Unit") {
		t.Error("exec method should return Unit")
	}
}

func TestGenerateQueryMethod_ScyllaExec(t *testing.T) {
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
}

// need os for TestGenerateModels_Smoke — already imported above