package migrator

import (
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/database/sqlite"
	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

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

	// Check key statements are present
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
				// ModifiedColumns with a REAL type change (TEXT → INTEGER) triggers table recreation
				ModifiedColumns: []types.ColumnDiff{
					{
						Name:    "name",
						OldType: "TEXT",
						NewType: "INTEGER",
						OldColumn: types.SchemaColumn{Name: "name", Type: "TEXT", Nullable: false},
						NewColumn: types.SchemaColumn{Name: "name", Type: "INTEGER", Nullable: false},
					},
				},
				// NewColumns would normally trigger ALTER TABLE ADD COLUMN
				NewColumns: []types.SchemaColumn{
					{Name: "is_active", Type: "BOOLEAN", Nullable: false, Default: "1"},
				},
			},
		},
	}

	sql, _ := m.generateSQLFromDiff(diff, "test")

	// Should contain table recreation
	if !contains(sql, `CREATE TABLE "users_new"`) {
		t.Error("expected table recreation SQL")
	}

	// Must NOT contain redundant ALTER TABLE ADD COLUMN since recreation handles it
	if contains(sql, `ALTER TABLE "users" ADD COLUMN`) {
		t.Error("migration should NOT contain redundant ALTER TABLE ADD COLUMN when table recreation happens")
	}
}
