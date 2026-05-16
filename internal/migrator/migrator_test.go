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
