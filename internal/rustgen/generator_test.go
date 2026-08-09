package rustgen

import (
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/gencommon"
	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

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
		{"  SELECT   id   FROM   users  ", "SELECT id FROM users"},
		{`INSERT INTO users (name) VALUES ('hello\\world')`, `INSERT INTO users (name) VALUES ('hello\\\\world')`},
	}

	for _, tt := range tests {
		got := escapeSQLForRust(tt.input)
		if got != tt.want {
			t.Errorf("escapeSQLForRust(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToRustStructName_PascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GetUser", "GetUser"},
		{"GetUserByEmail", "GetUserByEmail"},
		{"GetUserAgeStats", "GetUserAgeStats"},
		{"ListUsersWithPostCount", "ListUsersWithPostCount"},
		{"get_user", "GetUser"},
		{"get_user_by_email", "GetUserByEmail"},
		{"create_user", "CreateUser"},
	}

	for _, tt := range tests {
		got := toRustStructName(tt.input)
		if got != tt.want {
			t.Errorf("toRustStructName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGetDBType(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{"postgresql", "sqlx::Postgres"},
		{"postgres", "sqlx::Postgres"},
		{"mysql", "sqlx::MySql"},
		{"sqlite", "sqlx::Sqlite"},
	}

	for _, tt := range tests {
		g := newGen(tt.provider)
		got := g.getDBType()
		if got != tt.want {
			t.Errorf("getDBType(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

func TestGetExecResultType(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{"postgresql", "sqlx::postgres::PgQueryResult"},
		{"mysql", "sqlx::mysql::MySqlQueryResult"},
		{"sqlite", "sqlx::sqlite::SqliteQueryResult"},
	}

	for _, tt := range tests {
		g := newGen(tt.provider)
		got := g.getExecResultType()
		if got != tt.want {
			t.Errorf("getExecResultType(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

func TestSQLTypeToRust_DateTimeTypes(t *testing.T) {
	g := newGen("postgresql")

	tests := []struct {
		sqlType  string
		nullable bool
		want     string
	}{
		{"DATE", false, "chrono::NaiveDate"},
		{"DATE", true, "Option<chrono::NaiveDate>"},
		{"TIME", false, "chrono::NaiveTime"},
		{"TIME WITHOUT TIME ZONE", false, "chrono::NaiveTime"},
		{"TIMESTAMP WITHOUT TIME ZONE", false, "NaiveDateTime"},
		{"TIMESTAMP WITH TIME ZONE", false, "DateTime<Utc>"},
		{"DATETIME", false, "NaiveDateTime"},
	}

	for _, tt := range tests {
		got := g.sqlTypeToRust(tt.sqlType, tt.nullable)
		if got != tt.want {
			t.Errorf("sqlTypeToRust(%q, %v) = %q, want %q", tt.sqlType, tt.nullable, got, tt.want)
		}
	}
}

func TestSQLTypeToRust_NetworkAndBinaryTypes(t *testing.T) {
	g := newGen("postgresql")

	tests := []struct {
		sqlType string
		want    string
	}{
		{"INET", "ipnetwork::IpNetwork"},
		{"CIDR", "ipnetwork::IpNetwork"},
		{"MACADDR", "mac_address::MacAddress"},
		{"MACADDR8", "mac_address::MacAddress"},
		{"BYTEA", "Vec<u8>"},
		{"BLOB", "Vec<u8>"},
		{"BINARY(16)", "Vec<u8>"},
		{"VARBINARY(255)", "Vec<u8>"},
	}

	for _, tt := range tests {
		got := g.sqlTypeToRust(tt.sqlType, false)
		if got != tt.want {
			t.Errorf("sqlTypeToRust(%q) = %q, want %q", tt.sqlType, got, tt.want)
		}
	}
}

func TestSQLTypeToRust_CharacterTypes(t *testing.T) {
	g := newGen("postgresql")

	tests := []struct {
		sqlType string
		want    string
	}{
		{"TEXT", "String"},
		{"CITEXT", "String"},
		{"NAME", "String"},
		{"VARCHAR(100)", "String"},
		{"CHAR(1)", "String"},
		{"CHARACTER VARYING(255)", "String"},
	}

	for _, tt := range tests {
		got := g.sqlTypeToRust(tt.sqlType, false)
		if got != tt.want {
			t.Errorf("sqlTypeToRust(%q) = %q, want %q", tt.sqlType, got, tt.want)
		}
	}
}

func TestSQLTypeToRust_IntegerVariants(t *testing.T) {
	g := newGen("postgresql")

	tests := []struct {
		sqlType string
		want    string
	}{
		{"INT", "i32"},
		{"INT2", "i16"},
		{"INT4", "i32"},
		{"INT8", "i64"},
		{"SMALLSERIAL", "i16"},
		{"SERIAL", "i32"},
		{"BIGSERIAL", "i64"},
	}

	for _, tt := range tests {
		got := g.sqlTypeToRust(tt.sqlType, false)
		if got != tt.want {
			t.Errorf("sqlTypeToRust(%q) = %q, want %q", tt.sqlType, got, tt.want)
		}
	}
}

func TestSQLTypeToRust_NumericWithPrecision(t *testing.T) {
	g := newGen("postgresql")

	tests := []struct {
		sqlType string
		want    string
	}{
		{"NUMERIC(10,2)", "Decimal"},
		{"DECIMAL(5,3)", "Decimal"},
		{"NUMERIC", "Decimal"},
	}

	for _, tt := range tests {
		got := g.sqlTypeToRust(tt.sqlType, false)
		if got != tt.want {
			t.Errorf("sqlTypeToRust(%q) = %q, want %q", tt.sqlType, got, tt.want)
		}
	}
}

func TestSQLTypeToRust_ArrayOfArrays(t *testing.T) {
	g := newGen("postgresql")

	tests := []struct {
		sqlType  string
		nullable bool
		want     string
	}{
		{"BIGINT[]", false, "Vec<i64>"},
		{"BOOLEAN[]", false, "Vec<bool>"},
		{"UUID[]", false, "Vec<Uuid>"},
		{"JSONB[]", false, "Vec<serde_json::Value>"},
		{"REAL[]", true, "Option<Vec<f32>>"},
	}

	for _, tt := range tests {
		got := g.sqlTypeToRust(tt.sqlType, tt.nullable)
		if got != tt.want {
			t.Errorf("sqlTypeToRust(%q, %v) = %q, want %q", tt.sqlType, tt.nullable, got, tt.want)
		}
	}
}

func TestSQLTypeToRust_UnknownType_FallsBackToString(t *testing.T) {
	g := newGen("postgresql")

	got := g.sqlTypeToRust("UNKNOWN_TYPE", false)
	if got != "String" {
		t.Errorf("sqlTypeToRust(UNKNOWN_TYPE) = %q, want String (fallback)", got)
	}

	got = g.sqlTypeToRust("UNKNOWN_TYPE", true)
	if got != "Option<String>" {
		t.Errorf("sqlTypeToRust(UNKNOWN_TYPE, nullable) = %q, want Option<String>", got)
	}
}

func TestParamTypeToRust_References(t *testing.T) {
	g := newGen("postgresql")

	tests := []struct {
		sqlType string
		want    string
	}{
		{"TEXT", "&str"},
		{"VARCHAR(255)", "&str"},
		{"CITEXT", "&str"},
		{"BYTEA", "&[u8]"},
		{"TEXT[]", "&Vec<String>"},
		{"INTEGER[]", "&Vec<i32>"},
		// Value types — no reference
		{"INTEGER", "i32"},
		{"BIGINT", "i64"},
		{"BOOLEAN", "bool"},
		{"UUID", "Uuid"},
		{"REAL", "f32"},
		{"DOUBLE PRECISION", "f64"},
		{"NUMERIC", "Decimal"},
	}

	for _, tt := range tests {
		got := g.paramTypeToRust(tt.sqlType)
		if got != tt.want {
			t.Errorf("paramTypeToRust(%q) = %q, want %q", tt.sqlType, got, tt.want)
		}
	}
}

func TestConvertSQL_MySQL_MultipleParams(t *testing.T) {
	g := newGen("mysql")
	sql := "INSERT INTO users (name, email, age) VALUES ($1, $2, $3) RETURNING id"
	want := "INSERT INTO users (name, email, age) VALUES (?, ?, ?) RETURNING id"
	got := g.convertSQL(sql)
	if got != want {
		t.Errorf("convertSQL(mysql multi) = %q, want %q", got, want)
	}
}

func TestConvertSQL_PostgreSQL_HighParams(t *testing.T) {
	g := newGen("postgresql")
	sql := "INSERT INTO t (a,b,c,d,e) VALUES ($1,$2,$3,$4,$5)"
	got := g.convertSQL(sql)
	if got != sql {
		t.Errorf("convertSQL(pg high params) should not modify, got %q", got)
	}
}

func TestCollectImports_EmptyQueries(t *testing.T) {
	g := newGen("postgresql")
	imports := g.collectImports([]*parser.Query{})
	if len(imports) != 0 {
		t.Errorf("collectImports(empty) should return empty, got %v", imports)
	}
}

func TestCollectImports_WithTimestamp(t *testing.T) {
	g := newGen("postgresql")
	queries := []*parser.Query{
		{
			Columns: []*parser.QueryColumn{
				{Name: "created_at", Type: "TIMESTAMPTZ", Nullable: false},
			},
		},
	}
	imports := g.collectImports(queries)
	found := false
	for _, imp := range imports {
		if imp == "chrono::{DateTime, Utc}" {
			found = true
		}
	}
	if !found {
		t.Errorf("collectImports with TIMESTAMPTZ should include chrono, got %v", imports)
	}
}

func TestCollectImports_WithUUID(t *testing.T) {
	g := newGen("postgresql")
	queries := []*parser.Query{
		{
			Columns: []*parser.QueryColumn{
				{Name: "id", Type: "UUID", Nullable: false},
			},
		},
	}
	imports := g.collectImports(queries)
	found := false
	for _, imp := range imports {
		if imp == "uuid::Uuid" {
			found = true
		}
	}
	if !found {
		t.Errorf("collectImports with UUID should include uuid, got %v", imports)
	}
}

func TestCollectImports_WithDecimalAndJSON(t *testing.T) {
	g := newGen("postgresql")
	queries := []*parser.Query{
		{
			Columns: []*parser.QueryColumn{
				{Name: "amount", Type: "NUMERIC", Nullable: false},
				{Name: "data", Type: "JSONB", Nullable: true},
			},
		},
	}
	imports := g.collectImports(queries)
	hasDecimal := false
	hasJson := false
	for _, imp := range imports {
		if imp == "rust_decimal::Decimal" {
			hasDecimal = true
		}
		if imp == "serde_json::Value" {
			hasJson = true
		}
	}
	if !hasDecimal {
		t.Errorf("collectImports with NUMERIC should include rust_decimal, got %v", imports)
	}
	if !hasJson {
		t.Errorf("collectImports with JSONB should include serde_json, got %v", imports)
	}
}

func TestGetOneReturnType_MatchesModel(t *testing.T) {
	g := newGen("postgresql")
	g.expander = gencommon.NewSchemaExpander(g.schema)

	// Query columns match the "users" table exactly
	q := &parser.Query{
		Name: "GetUser",
		SQL:  "SELECT id, name, email, created_at FROM users WHERE id = $1",
	}
	columns := []*parser.QueryColumn{
		{Name: "id", Type: "SERIAL", Nullable: false},
		{Name: "name", Type: "VARCHAR(255)", Nullable: false},
		{Name: "email", Type: "VARCHAR(255)", Nullable: false},
		{Name: "created_at", Type: "TIMESTAMPTZ", Nullable: false},
	}

	got := g.getOneReturnType(q, columns)
	// Should return the model type "Users"
	if got != "Users" {
		t.Errorf("getOneReturnType (matching model) = %q, want Users", got)
	}
}

func TestGetOneReturnType_EmptyColumns(t *testing.T) {
	g := newGen("postgresql")
	q := &parser.Query{Name: "DeleteUser", SQL: "DELETE FROM users WHERE id = $1"}
	got := g.getOneReturnType(q, nil)
	if got != "()" {
		t.Errorf("getOneReturnType (empty) = %q, want ()", got)
	}
}

func TestGetOneReturnType_SingleColumn(t *testing.T) {
	g := newGen("postgresql")
	q := &parser.Query{Name: "CountUsers", SQL: "SELECT COUNT(*) FROM users"}
	columns := []*parser.QueryColumn{
		{Name: "count", Type: "BIGINT", Nullable: false},
	}
	got := g.getOneReturnType(q, columns)
	if got != "i64" {
		t.Errorf("getOneReturnType (single BIGINT) = %q, want i64", got)
	}
}

func TestGetOneReturnType_CustomRow(t *testing.T) {
	g := newGen("postgresql")
	q := &parser.Query{
		Name: "GetUserWithPostCount",
		SQL:  "SELECT u.id, u.name, COUNT(p.id) AS post_count FROM users u LEFT JOIN posts p ON p.user_id = u.id",
	}
	columns := []*parser.QueryColumn{
		{Name: "id", Type: "INTEGER", Nullable: false},
		{Name: "name", Type: "VARCHAR(255)", Nullable: false},
		{Name: "post_count", Type: "BIGINT", Nullable: false},
	}
	got := g.getOneReturnType(q, columns)
	if got != "GetUserWithPostCountRow" {
		t.Errorf("getOneReturnType (custom) = %q, want GetUserWithPostCountRow", got)
	}
}

func TestMacroArgs_NoParams(t *testing.T) {
	g := newGen("postgresql")
	q := &parser.Query{
		Name:   "ListUsers",
		SQL:    "SELECT * FROM users",
		Params: []*parser.Param{},
	}
	got := g.macroArgs(q)
	if got != "" {
		t.Errorf("macroArgs (no params) = %q, want empty string", got)
	}
}

func TestMacroArgs_TwoParams(t *testing.T) {
	g := newGen("postgresql")
	q := &parser.Query{
		Name: "GetUser",
		SQL:  "SELECT * FROM users WHERE id = $1 AND name = $2",
		Params: []*parser.Param{
			{Name: "id", Type: "INTEGER"},
			{Name: "name", Type: "VARCHAR"},
		},
	}
	got := g.macroArgs(q)
	if got != ", id, name" {
		t.Errorf("macroArgs (two params) = %q, want %q", got, ", id, name")
	}
}

func TestMacroArgs_ManyParams_UsesParamsStruct(t *testing.T) {
	g := newGen("postgresql")
	q := &parser.Query{
		Name: "CreateUser",
		SQL:  "INSERT INTO users (name, email, age) VALUES ($1, $2, $3)",
		Params: []*parser.Param{
			{Name: "name", Type: "VARCHAR"},
			{Name: "email", Type: "VARCHAR"},
			{Name: "age", Type: "INTEGER"},
		},
	}
	got := g.macroArgs(q)
	if got != ", params.name, params.email, params.age" {
		t.Errorf("macroArgs (3 params) = %q, want %q", got, ", params.name, params.email, params.age")
	}
}

func TestWriteMacroExecBody_Exec(t *testing.T) {
	g := newGen("postgresql")
	g.Config.Gen.Rust.Macros = true

	q := &parser.Query{
		Name: "DeleteUser",
		SQL:  "DELETE FROM users WHERE id = $1",
		Params: []*parser.Param{
			{Name: "id", Type: "INTEGER"},
		},
	}
	w := gencommon.GetBuilder()
	defer gencommon.PutBuilder(w)
	g.writeMacroExecBody(w, q, "DELETE FROM users WHERE id = $1", ":exec")
	output := w.String()
	if !contains(output, "sqlx::query!") {
		t.Errorf("macro exec body should contain sqlx::query!, got:\n%s", output)
	}
	if !contains(output, "Ok(())") {
		t.Errorf("macro exec body should contain Ok(()), got:\n%s", output)
	}
}

func TestWriteMacroOneBody_MultiColumn(t *testing.T) {
	g := newGen("postgresql")
	g.Config.Gen.Rust.Macros = true
	g.expander = gencommon.NewSchemaExpander(g.schema)

	q := &parser.Query{
		Name: "GetUserStats",
		SQL:  "SELECT id, name FROM users WHERE id = $1",
		Params: []*parser.Param{
			{Name: "id", Type: "INTEGER"},
		},
		Columns: []*parser.QueryColumn{
			{Name: "id", Type: "INTEGER", Nullable: false},
			{Name: "name", Type: "VARCHAR(255)", Nullable: false},
		},
	}
	columns := q.Columns
	w := gencommon.GetBuilder()
	defer gencommon.PutBuilder(w)
	g.writeMacroOneBody(w, q, "SELECT id, name FROM users WHERE id = $1", columns)
	output := w.String()
	if !contains(output, "sqlx::query_as!") {
		t.Errorf("macro one body (multi col) should contain sqlx::query_as!, got:\n%s", output)
	}
	if !contains(output, "GetUserStatsRow") {
		t.Errorf("macro one body should reference row struct, got:\n%s", output)
	}
	if !contains(output, "fetch_one") {
		t.Errorf("macro one body should use fetch_one, got:\n%s", output)
	}
}

func TestWriteMacroOneBody_SingleColumn(t *testing.T) {
	g := newGen("postgresql")
	g.Config.Gen.Rust.Macros = true
	g.expander = gencommon.NewSchemaExpander(g.schema)

	q := &parser.Query{
		Name: "CountUsers",
		SQL:  "SELECT COUNT(*) FROM users",
		Columns: []*parser.QueryColumn{
			{Name: "count", Type: "BIGINT", Nullable: false},
		},
	}
	columns := q.Columns
	w := gencommon.GetBuilder()
	defer gencommon.PutBuilder(w)
	g.writeMacroOneBody(w, q, "SELECT COUNT(*) FROM users", columns)
	output := w.String()
	if !contains(output, "sqlx::query_scalar!") {
		t.Errorf("macro one body (single col) should contain sqlx::query_scalar!, got:\n%s", output)
	}
}

func TestWriteMacroManyBody_MultiColumn(t *testing.T) {
	g := newGen("postgresql")
	g.Config.Gen.Rust.Macros = true
	g.expander = gencommon.NewSchemaExpander(g.schema)

	q := &parser.Query{
		Name: "ListUserStats",
		SQL:  "SELECT id, name FROM users",
		Columns: []*parser.QueryColumn{
			{Name: "id", Type: "INTEGER", Nullable: false},
			{Name: "name", Type: "VARCHAR(255)", Nullable: false},
		},
	}
	columns := q.Columns
	w := gencommon.GetBuilder()
	defer gencommon.PutBuilder(w)
	g.writeMacroManyBody(w, q, "SELECT id, name FROM users", columns)
	output := w.String()
	if !contains(output, "sqlx::query_as!") {
		t.Errorf("macro many body (multi col) should contain sqlx::query_as!, got:\n%s", output)
	}
	if !contains(output, "fetch_all") {
		t.Errorf("macro many body should use fetch_all, got:\n%s", output)
	}
}

func TestComputeConfigChecksum_MacrosChangesChecksum(t *testing.T) {
	g1 := newGen("postgresql")
	g1.Config.Gen.Rust.Macros = false
	hash1 := g1.computeConfigChecksum()

	g2 := newGen("postgresql")
	g2.Config.Gen.Rust.Macros = true
	hash2 := g2.computeConfigChecksum()

	if hash1 == hash2 {
		t.Error("changing macros flag should produce different config checksum")
	}
}
