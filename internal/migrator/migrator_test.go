package migrator

import (
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/database/mysql"
	"github.com/Lumos-Labs-HQ/flash/internal/database/postgres"
	"github.com/Lumos-Labs-HQ/flash/internal/database/sqlite"
	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ── dropTableSQL provider-specific tests ──────────────────────────────────────

func TestGenerateSQLFromDiff_SQLiteDropTable(t *testing.T) {
	adapter := sqlite.New()
	m := &Migrator{adapter: adapter, provider: "sqlite"}

	diff := &types.SchemaDiff{
		DroppedTables: []string{"users"},
	}
	sql, hasSQL := m.generateSQLFromDiff(diff, "test")
	if !hasSQL {
		t.Fatal("expected executable SQL")
	}
	// SQLite: double quotes, no CASCADE
	if !contains(sql, `DROP TABLE IF EXISTS "users";`) {
		t.Errorf("SQLite drop table missing expected syntax:\n%s", sql)
	}
	if contains(sql, "CASCADE") {
		t.Error("SQLite DROP TABLE should NOT contain CASCADE")
	}
}

func TestGenerateSQLFromDiff_MySQLDropTable(t *testing.T) {
	adapter := mysql.New()
	m := &Migrator{adapter: adapter, provider: "mysql"}

	diff := &types.SchemaDiff{
		DroppedTables: []string{"users"},
	}
	sql, hasSQL := m.generateSQLFromDiff(diff, "test")
	if !hasSQL {
		t.Fatal("expected executable SQL")
	}
	// MySQL: backticks, no CASCADE
	if !contains(sql, "DROP TABLE IF EXISTS `users`;") {
		t.Errorf("MySQL drop table missing expected syntax:\n%s", sql)
	}
	if contains(sql, "CASCADE") {
		t.Error("MySQL DROP TABLE should NOT contain CASCADE")
	}
	if contains(sql, `"users"`) {
		t.Error("MySQL DROP TABLE should use backticks, not double quotes")
	}
}

func TestGenerateSQLFromDiff_PostgresDropTable(t *testing.T) {
	adapter := postgres.New()
	m := &Migrator{adapter: adapter, provider: "postgresql"}

	diff := &types.SchemaDiff{
		DroppedTables: []string{"users"},
	}
	sql, hasSQL := m.generateSQLFromDiff(diff, "test")
	if !hasSQL {
		t.Fatal("expected executable SQL")
	}
	// PostgreSQL: double quotes, WITH CASCADE
	if !contains(sql, `DROP TABLE IF EXISTS "users" CASCADE;`) {
		t.Errorf("PostgreSQL drop table missing expected syntax:\n%s", sql)
	}
}

// ── generateSQLFromDiff new table tests ───────────────────────────────────────

func TestGenerateSQLFromDiff_MySQLNewTable(t *testing.T) {
	adapter := mysql.New()
	m := &Migrator{adapter: adapter, provider: "mysql"}

	diff := &types.SchemaDiff{
		NewTables: []types.SchemaTable{
			{
				Name: "users",
				Columns: []types.SchemaColumn{
					{Name: "id", Type: "INT", IsPrimary: true, IsAutoIncrement: true},
					{Name: "email", Type: "VARCHAR(255)", Nullable: false},
				},
			},
		},
	}
	sql, hasSQL := m.generateSQLFromDiff(diff, "test")
	if !hasSQL {
		t.Fatal("expected executable SQL")
	}
	if !contains(sql, "CREATE TABLE IF NOT EXISTS `users`") {
		t.Errorf("MySQL CREATE TABLE missing:\n%s", sql)
	}
	if !contains(sql, "PRIMARY KEY") {
		t.Errorf("MySQL CREATE TABLE missing PRIMARY KEY:\n%s", sql)
	}
}

func TestGenerateSQLFromDiff_PostgresNewTable(t *testing.T) {
	adapter := postgres.New()
	m := &Migrator{adapter: adapter, provider: "postgresql"}

	diff := &types.SchemaDiff{
		NewTables: []types.SchemaTable{
			{
				Name: "users",
				Columns: []types.SchemaColumn{
					{Name: "id", Type: "SERIAL", IsPrimary: true},
					{Name: "email", Type: "VARCHAR(255)", Nullable: false},
				},
			},
		},
	}
	sql, hasSQL := m.generateSQLFromDiff(diff, "test")
	if !hasSQL {
		t.Fatal("expected executable SQL")
	}
	if !contains(sql, `CREATE TABLE IF NOT EXISTS "users"`) {
		t.Errorf("PostgreSQL CREATE TABLE missing:\n%s", sql)
	}
	if !contains(sql, "PRIMARY KEY") {
		t.Errorf("PostgreSQL CREATE TABLE missing PRIMARY KEY:\n%s", sql)
	}
}

// ── generateSQLFromDiff modified column tests ─────────────────────────────────

func TestGenerateSQLFromDiff_MySQLModifiedColumn(t *testing.T) {
	adapter := mysql.New()
	m := &Migrator{adapter: adapter, provider: "mysql"}

	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{
			{
				Name: "users",
				ModifiedColumns: []types.ColumnDiff{
					{
						Name:      "email",
						OldType:   "VARCHAR(100)",
						NewType:   "VARCHAR(255)",
						OldColumn: types.SchemaColumn{Name: "email", Type: "VARCHAR(100)", Nullable: false},
						NewColumn: types.SchemaColumn{Name: "email", Type: "VARCHAR(255)", Nullable: false},
					},
				},
			},
		},
	}
	sql, hasSQL := m.generateSQLFromDiff(diff, "test")
	if !hasSQL {
		t.Fatal("expected executable SQL")
	}
	// Up: ALTER TABLE MODIFY COLUMN with new type
	if !contains(sql, "MODIFY COLUMN `email`") {
		t.Errorf("MySQL MODIFY COLUMN missing:\n%s", sql)
	}
	// Down: ALTER TABLE MODIFY COLUMN reverting to old type
	if !contains(sql, "MODIFY COLUMN `email`") {
		t.Errorf("MySQL down MODIFY COLUMN missing:\n%s", sql)
	}
}

func TestGenerateSQLFromDiff_MySQLModifiedColumn_NoPrimaryKeyInAlter(t *testing.T) {
	adapter := mysql.New()
	m := &Migrator{adapter: adapter, provider: "mysql"}

	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{
			{
				Name: "users",
				ModifiedColumns: []types.ColumnDiff{
					{
						Name:      "id",
						OldType:   "INT",
						NewType:   "BIGINT",
						OldColumn: types.SchemaColumn{Name: "id", Type: "INT", IsPrimary: true, IsAutoIncrement: true},
						NewColumn: types.SchemaColumn{Name: "id", Type: "BIGINT", IsPrimary: true, IsAutoIncrement: true},
					},
				},
			},
		},
	}
	sql, hasSQL := m.generateSQLFromDiff(diff, "test")
	if !hasSQL {
		t.Fatal("expected executable SQL")
	}
	// MODIFY COLUMN must NOT contain PRIMARY KEY — it would cause Error 1068.
	if contains(sql, "MODIFY COLUMN `id` BIGINT PRIMARY KEY") {
		t.Errorf("MySQL MODIFY COLUMN should NOT contain PRIMARY KEY:\n%s", sql)
	}
	if !contains(sql, "AUTO_INCREMENT") {
		t.Errorf("MySQL MODIFY COLUMN should still contain AUTO_INCREMENT:\n%s", sql)
	}
}

func TestGenerateSQLFromDiff_PostgresModifiedColumn(t *testing.T) {
	adapter := postgres.New()
	m := &Migrator{adapter: adapter, provider: "postgresql"}

	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{
			{
				Name: "users",
				ModifiedColumns: []types.ColumnDiff{
					{
						Name:      "email",
						OldType:   "VARCHAR(100)",
						NewType:   "TEXT",
						OldColumn: types.SchemaColumn{Name: "email", Type: "VARCHAR(100)", Nullable: false},
						NewColumn: types.SchemaColumn{Name: "email", Type: "TEXT", Nullable: false},
					},
				},
			},
		},
	}
	sql, hasSQL := m.generateSQLFromDiff(diff, "test")
	if !hasSQL {
		t.Fatal("expected executable SQL")
	}
	if !contains(sql, `ALTER TABLE "users" ALTER COLUMN "email" TYPE TEXT;`) {
		t.Errorf("PostgreSQL ALTER COLUMN TYPE missing:\n%s", sql)
	}
}

func TestGenerateSQLFromDiff_PostgresModifiedColumn_NeverSerial(t *testing.T) {
	adapter := postgres.New()
	m := &Migrator{adapter: adapter, provider: "postgresql"}

	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{
			{
				Name: "users",
				ModifiedColumns: []types.ColumnDiff{
					{
						Name:      "id",
						OldType:   "INTEGER",
						NewType:   "SERIAL",
						OldColumn: types.SchemaColumn{Name: "id", Type: "INTEGER", IsPrimary: true},
						NewColumn: types.SchemaColumn{Name: "id", Type: "SERIAL", IsPrimary: true},
					},
				},
			},
		},
	}
	sql, hasSQL := m.generateSQLFromDiff(diff, "test")
	if !hasSQL {
		t.Fatal("expected executable SQL")
	}
	// SERIAL is invalid in ALTER COLUMN TYPE — should use INTEGER instead.
	if contains(sql, "TYPE SERIAL") {
		t.Errorf("PostgreSQL ALTER COLUMN TYPE must NOT contain SERIAL:\n%s", sql)
	}
	if !contains(sql, "TYPE INTEGER") {
		t.Errorf("PostgreSQL ALTER COLUMN TYPE should use INTEGER:\n%s", sql)
	}
}

// ── hasSignificantSQLiteModifications tests ───────────────────────────────────

func TestHasSignificantSQLiteModifications_True(t *testing.T) {
	adapter := sqlite.New()
	m := &Migrator{adapter: adapter, provider: "sqlite"}

	tableDiff := types.TableDiff{
		Name: "users",
		ModifiedColumns: []types.ColumnDiff{
			{
				Name:      "age",
				OldType:   "TEXT",
				NewType:   "INTEGER",
				OldColumn: types.SchemaColumn{Name: "age", Type: "TEXT"},
				NewColumn: types.SchemaColumn{Name: "age", Type: "INTEGER"},
			},
		},
	}
	if !m.hasSignificantSQLiteModifications(tableDiff) {
		t.Error("TEXT → INTEGER is a significant modification")
	}
}

func TestHasSignificantSQLiteModifications_False_Cosmetic(t *testing.T) {
	adapter := sqlite.New()
	m := &Migrator{adapter: adapter, provider: "sqlite"}

	tableDiff := types.TableDiff{
		Name: "users",
		ModifiedColumns: []types.ColumnDiff{
			{
				Name:      "email",
				OldType:   "TEXT",
				NewType:   "VARCHAR(255)",
				OldColumn: types.SchemaColumn{Name: "email", Type: "TEXT"},
				NewColumn: types.SchemaColumn{Name: "email", Type: "VARCHAR(255)"},
			},
		},
	}
	if m.hasSignificantSQLiteModifications(tableDiff) {
		t.Error("TEXT → VARCHAR(255) is a cosmetic change, not significant")
	}
}

// ── generateSQLFromDiff empty migration ───────────────────────────────────────

func TestGenerateSQLFromDiff_EmptyDiff(t *testing.T) {
	adapter := sqlite.New()
	m := &Migrator{adapter: adapter, provider: "sqlite"}

	diff := &types.SchemaDiff{}
	sql, hasSQL := m.generateSQLFromDiff(diff, "test")
	if hasSQL {
		t.Error("empty diff should produce no executable SQL")
	}
	if sql == "" {
		t.Fatal("empty diff should still produce a migration template")
	}
	if !contains(sql, "-- No migration statements") {
		t.Errorf("empty diff should produce empty migration template:\n%s", sql)
	}
}

// ── SQLite table recreation (existing test, kept for coverage) ────────────────

func TestGenerateSQLiteTableRecreateSQL(t *testing.T) {
	adapter := sqlite.New()
	m := &Migrator{
		adapter:  adapter,
		provider: "sqlite",
	}

	oldTable := types.SchemaTable{
		Name: "users",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "INTEGER", IsPrimary: true},
			{Name: "name", Type: "TEXT", Nullable: false},
			{Name: "email", Type: "TEXT", Nullable: false, IsUnique: true},
		},
		Indexes: []types.SchemaIndex{
			{Name: "idx_email", Table: "users", Columns: []string{"email"}, Unique: true},
		},
	}

	newTable := types.SchemaTable{
		Name: "users",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "INTEGER", IsPrimary: true},
			{Name: "name", Type: "TEXT", Nullable: false},
			{Name: "email", Type: "VARCHAR(255)", Nullable: false, IsUnique: true},
		},
		Indexes: []types.SchemaIndex{
			{Name: "idx_email", Table: "users", Columns: []string{"email"}, Unique: true},
		},
	}

	sql := m.generateSQLiteTableRecreateSQL(oldTable, newTable)
	if sql == "" {
		t.Fatal("expected non-empty recreation SQL")
	}

	required := []string{
		"PRAGMA foreign_keys=OFF;",
		`CREATE TABLE "users_new"`,
		`INSERT INTO "users_new"`,
		`DROP TABLE "users";`,
		`ALTER TABLE "users_new" RENAME TO "users";`,
		`CREATE UNIQUE INDEX "idx_email"`,
		"PRAGMA foreign_keys=ON;",
	}
	for _, r := range required {
		if !contains(sql, r) {
			t.Errorf("missing expected statement: %q\nGenerated SQL:\n%s", r, sql)
		}
	}
}

func TestGenerateSQLFromDiff_SQLiteSkipsRedundantAlterTable(t *testing.T) {
	adapter := sqlite.New()
	m := &Migrator{
		adapter:  adapter,
		provider: "sqlite",
	}

	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{
			{
				Name: "users",
				OldTable: types.SchemaTable{
					Name: "users",
					Columns: []types.SchemaColumn{
						{Name: "id", Type: "INTEGER", IsPrimary: true},
						{Name: "name", Type: "TEXT", Nullable: false},
					},
				},
				NewTable: types.SchemaTable{
					Name: "users",
					Columns: []types.SchemaColumn{
						{Name: "id", Type: "INTEGER", IsPrimary: true},
						{Name: "name", Type: "TEXT", Nullable: false},
						{Name: "is_active", Type: "BOOLEAN", Nullable: false, Default: "1"},
					},
				},
				ModifiedColumns: []types.ColumnDiff{
					{
						Name:      "name",
						OldType:   "TEXT",
						NewType:   "INTEGER",
						OldColumn: types.SchemaColumn{Name: "name", Type: "TEXT", Nullable: false},
						NewColumn: types.SchemaColumn{Name: "name", Type: "INTEGER", Nullable: false},
					},
				},
				NewColumns: []types.SchemaColumn{
					{Name: "is_active", Type: "BOOLEAN", Nullable: false, Default: "1"},
				},
			},
		},
	}

	sql, _ := m.generateSQLFromDiff(diff, "test")

	if !contains(sql, `CREATE TABLE "users_new"`) {
		t.Error("expected table recreation SQL")
	}
	if contains(sql, `ALTER TABLE "users" ADD COLUMN`) {
		t.Error("migration should NOT contain redundant ALTER TABLE ADD COLUMN when table recreation happens")
	}
}
