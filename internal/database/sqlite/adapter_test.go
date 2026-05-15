package sqlite

import (
	"context"
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

// connectMemory opens an in-memory SQLite database.
func connectMemory(t *testing.T) *Adapter {
	t.Helper()
	a := New()
	if err := a.Connect(context.Background(), ":memory:"); err != nil {
		t.Fatalf("Connect(:memory:) error: %v", err)
	}
	t.Cleanup(func() { a.Close() })
	return a
}

// ── MapColumnType ─────────────────────────────────────────────────────────────

func TestMapColumnType(t *testing.T) {
	a := New()
	cases := []struct{ in, want string }{
		{"varchar", "TEXT"},
		{"text", "TEXT"},
		{"int", "INTEGER"},
		{"integer", "INTEGER"},
		{"bigint", "INTEGER"},
		{"real", "REAL"},
		{"float", "REAL"},
		{"blob", "BLOB"},
		{"boolean", "INTEGER"},
		{"bool", "INTEGER"},
		{"date", "TEXT"},
		{"datetime", "TEXT"},
		{"timestamp", "TEXT"},
		{"UNKNOWN", "UNKNOWN"},
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
	a := New()
	col := types.SchemaColumn{Name: "id", Type: "INTEGER", IsPrimary: true}
	got := a.FormatColumnType(col)
	if !strings.Contains(got, "PRIMARY KEY") {
		t.Errorf("FormatColumnType(PK) = %q, missing PRIMARY KEY", got)
	}
}

func TestFormatColumnType_NotNull(t *testing.T) {
	a := New()
	col := types.SchemaColumn{Name: "name", Type: "TEXT", Nullable: false}
	got := a.FormatColumnType(col)
	if !strings.Contains(got, "NOT NULL") {
		t.Errorf("FormatColumnType(NOT NULL) = %q, missing NOT NULL", got)
	}
}

// ── GenerateCreateTableSQL ────────────────────────────────────────────────────

func TestGenerateCreateTableSQL(t *testing.T) {
	a := New()
	table := types.SchemaTable{
		Name: "users",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "INTEGER", IsPrimary: true},
			{Name: "email", Type: "TEXT", Nullable: false},
			{Name: "bio", Type: "TEXT", Nullable: true},
		},
	}
	sql := a.GenerateCreateTableSQL(table)
	for _, want := range []string{"CREATE TABLE", "users", "id", "email", "PRIMARY KEY"} {
		if !strings.Contains(sql, want) {
			t.Errorf("GenerateCreateTableSQL missing %q:\n%s", want, sql)
		}
	}
}

// ── GenerateAddColumnSQL ──────────────────────────────────────────────────────

func TestGenerateAddColumnSQL(t *testing.T) {
	a := New()
	col := types.SchemaColumn{Name: "phone", Type: "TEXT", Nullable: true}
	sql := a.GenerateAddColumnSQL("users", col)
	if !strings.Contains(sql, "ALTER TABLE") || !strings.Contains(sql, "phone") {
		t.Errorf("GenerateAddColumnSQL = %q", sql)
	}
}

// ── GenerateDropColumnSQL ─────────────────────────────────────────────────────

func TestGenerateDropColumnSQL(t *testing.T) {
	a := New()
	sql := a.GenerateDropColumnSQL("users", "phone")
	if !strings.Contains(sql, "DROP COLUMN") || !strings.Contains(sql, "phone") {
		t.Errorf("GenerateDropColumnSQL = %q", sql)
	}
}

// ── GenerateAddIndexSQL ───────────────────────────────────────────────────────

func TestGenerateAddIndexSQL_Unique(t *testing.T) {
	a := New()
	idx := types.SchemaIndex{Name: "idx_email", Table: "users", Columns: []string{"email"}, Unique: true}
	sql := a.GenerateAddIndexSQL(idx)
	if !strings.Contains(sql, "UNIQUE") || !strings.Contains(sql, "idx_email") {
		t.Errorf("GenerateAddIndexSQL(unique) = %q", sql)
	}
}

func TestGenerateDropIndexSQL(t *testing.T) {
	a := New()
	idx := types.SchemaIndex{Name: "idx_email", Table: "users"}
	sql := a.GenerateDropIndexSQL(idx)
	if !strings.Contains(sql, "DROP INDEX") || !strings.Contains(sql, "idx_email") {
		t.Errorf("GenerateDropIndexSQL = %q", sql)
	}
}

// ── validateTableName ─────────────────────────────────────────────────────────

func TestValidateTableName_Valid(t *testing.T) {
	a := New()
	for _, name := range []string{"users", "order_items", "_flash_migrations", "t1"} {
		if err := a.validateTableName(name); err != nil {
			t.Errorf("validateTableName(%q) unexpected error: %v", name, err)
		}
	}
}

func TestValidateTableName_Invalid(t *testing.T) {
	a := New()
	for _, name := range []string{"users; DROP TABLE users", "1invalid", "has space", "has-dash"} {
		if err := a.validateTableName(name); err == nil {
			t.Errorf("validateTableName(%q) expected error, got nil", name)
		}
	}
}

// ── Fresh DB behaviour ──────────────────────────────────────────────────────

func TestSQLiteAdapter_GetAppliedMigrations_FreshDB(t *testing.T) {
	ctx := context.Background()
	a := connectMemory(t)

	// On a fresh database, _flash_migrations doesn't exist yet.
	// GetAppliedMigrations should return an empty map without error.
	applied, err := a.GetAppliedMigrations(ctx)
	if err != nil {
		t.Fatalf("GetAppliedMigrations on fresh DB should not error: %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("expected 0 applied migrations on fresh DB, got %d", len(applied))
	}
}

// ── Full lifecycle (in-memory SQLite) ─────────────────────────────────────────

func TestSQLiteAdapter_Lifecycle(t *testing.T) {
	ctx := context.Background()
	a := connectMemory(t)

	// Ping
	if err := a.Ping(ctx); err != nil {
		t.Fatalf("Ping error: %v", err)
	}

	// Create migrations table
	if err := a.CreateMigrationsTable(ctx); err != nil {
		t.Fatalf("CreateMigrationsTable error: %v", err)
	}

	// Execute a migration
	migSQL := `CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		email TEXT NOT NULL,
		name TEXT
	);`
	if err := a.ExecuteMigration(ctx, migSQL); err != nil {
		t.Fatalf("ExecuteMigration error: %v", err)
	}

	// Record migration
	if err := a.RecordMigration(ctx, "20240101_init", "init", "checksum1"); err != nil {
		t.Fatalf("RecordMigration error: %v", err)
	}

	// GetAppliedMigrations
	applied, err := a.GetAppliedMigrations(ctx)
	if err != nil {
		t.Fatalf("GetAppliedMigrations error: %v", err)
	}
	if _, ok := applied["20240101_init"]; !ok {
		t.Error("migration not found in applied list")
	}

	// GetAllTableNames
	tables, err := a.GetAllTableNames(ctx)
	if err != nil {
		t.Fatalf("GetAllTableNames error: %v", err)
	}
	found := false
	for _, tbl := range tables {
		if tbl == "users" {
			found = true
		}
	}
	if !found {
		t.Errorf("users table not found in %v", tables)
	}

	// CheckTableExists
	exists, err := a.CheckTableExists(ctx, "users")
	if err != nil || !exists {
		t.Errorf("CheckTableExists(users) = %v, %v", exists, err)
	}
	notExists, _ := a.CheckTableExists(ctx, "nonexistent")
	if notExists {
		t.Error("nonexistent table should not exist")
	}

	// CheckColumnExists
	colExists, err := a.CheckColumnExists(ctx, "users", "email")
	if err != nil || !colExists {
		t.Errorf("CheckColumnExists(users.email) = %v, %v", colExists, err)
	}

	// CheckNotNullConstraint
	notNull, err := a.CheckNotNullConstraint(ctx, "users", "email")
	if err != nil || !notNull {
		t.Errorf("CheckNotNullConstraint(users.email) = %v, %v", notNull, err)
	}
	nullable, _ := a.CheckNotNullConstraint(ctx, "users", "name")
	if nullable {
		t.Error("name column should be nullable")
	}

	// ExecuteQuery
	result, err := a.ExecuteQuery(ctx, "SELECT 1 AS val")
	if err != nil {
		t.Fatalf("ExecuteQuery error: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Errorf("ExecuteQuery rows = %d, want 1", len(result.Rows))
	}

	// GetTableData (empty table)
	data, err := a.GetTableData(ctx, "users")
	if err != nil {
		t.Fatalf("GetTableData error: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("GetTableData on empty table = %d rows, want 0", len(data))
	}

	// GetTableRowCount
	count, err := a.GetTableRowCount(ctx, "users")
	if err != nil || count != 0 {
		t.Errorf("GetTableRowCount = %d, %v", count, err)
	}

	// GetCurrentSchema
	schema, err := a.GetCurrentSchema(ctx)
	if err != nil {
		t.Fatalf("GetCurrentSchema error: %v", err)
	}
	foundUsers := false
	for _, tbl := range schema {
		if tbl.Name == "users" {
			foundUsers = true
			if len(tbl.Columns) == 0 {
				t.Error("users table has no columns in schema")
			}
		}
	}
	if !foundUsers {
		t.Error("users not found in GetCurrentSchema")
	}

	// RemoveMigrationRecord
	if err := a.RemoveMigrationRecord(ctx, "20240101_init"); err != nil {
		t.Fatalf("RemoveMigrationRecord error: %v", err)
	}
	applied2, _ := a.GetAppliedMigrations(ctx)
	if _, ok := applied2["20240101_init"]; ok {
		t.Error("migration should be removed")
	}

	// DropTable
	if err := a.DropTable(ctx, "users"); err != nil {
		t.Fatalf("DropTable error: %v", err)
	}
	exists2, _ := a.CheckTableExists(ctx, "users")
	if exists2 {
		t.Error("users table should be dropped")
	}
}

func TestSQLiteAdapter_ExecuteAndRecordMigration(t *testing.T) {
	ctx := context.Background()
	a := connectMemory(t)

	if err := a.CreateMigrationsTable(ctx); err != nil {
		t.Fatalf("CreateMigrationsTable: %v", err)
	}

	sql := `CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT NOT NULL);`
	if err := a.ExecuteAndRecordMigration(ctx, "20240102_posts", "create posts", "chk", sql); err != nil {
		t.Fatalf("ExecuteAndRecordMigration: %v", err)
	}

	applied, _ := a.GetAppliedMigrations(ctx)
	if _, ok := applied["20240102_posts"]; !ok {
		t.Error("migration not recorded")
	}

	exists, _ := a.CheckTableExists(ctx, "posts")
	if !exists {
		t.Error("posts table not created")
	}
}

func TestSQLiteAdapter_CleanupBrokenMigrationRecords(t *testing.T) {
	ctx := context.Background()
	a := connectMemory(t)
	if err := a.CreateMigrationsTable(ctx); err != nil {
		t.Fatalf("CreateMigrationsTable: %v", err)
	}
	// Should not error even on empty table
	if err := a.CleanupBrokenMigrationRecords(ctx); err != nil {
		t.Errorf("CleanupBrokenMigrationRecords error: %v", err)
	}
}

func TestSQLiteAdapter_GetAllTableRowCounts(t *testing.T) {
	ctx := context.Background()
	a := connectMemory(t)
	a.ExecuteMigration(ctx, "CREATE TABLE a (id INTEGER PRIMARY KEY); CREATE TABLE b (id INTEGER PRIMARY KEY);")

	counts, err := a.GetAllTableRowCounts(ctx, []string{"a", "b"})
	if err != nil {
		t.Fatalf("GetAllTableRowCounts error: %v", err)
	}
	if counts["a"] != 0 || counts["b"] != 0 {
		t.Errorf("counts = %v, want all 0", counts)
	}
}
