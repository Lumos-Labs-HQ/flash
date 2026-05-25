package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/database/sqlite"
	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func TestRealTypeChange_SQLite(t *testing.T) {
	adapter := sqlite.New()
	sm := NewSchemaManager(adapter)
	ctx := t.Context()

	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "snap.json")
	schemaPath := filepath.Join(tmpDir, "schema.sql")

	// Snapshot uses the SAME parser format as the schema file would produce
	snapTables := []types.SchemaTable{
		{
			Name: "users",
			Columns: []types.SchemaColumn{
				{Name: "id", Type: "INTEGER", Nullable: true, IsPrimary: true},
				{Name: "name", Type: "TEXT", Nullable: false},
			},
		},
	}
	SaveSchemaSnapshot(snapshotPath, snapTables, nil)

	// Schema changes TEXT to INTEGER (REAL type change)
	schemaSQL := `CREATE TABLE "users" (
		"id" INTEGER PRIMARY KEY,
		"name" INTEGER NOT NULL
	);`
	os.WriteFile(schemaPath, []byte(schemaSQL), 0644)

	diff, err := sm.GenerateSchemaDiff(ctx, schemaPath, snapshotPath)
	if err != nil {
		t.Fatalf("diff failed: %v", err)
	}

	for _, mod := range diff.ModifiedTables {
		for _, c := range mod.ModifiedColumns {
			fmt.Printf("Modified: %s (%s -> %s) changes=%v\n", c.Name, c.OldType, c.NewType, c.Changes)
		}
	}

	if len(diff.ModifiedTables) != 1 {
		t.Fatalf("expected 1 modified table, got %d", len(diff.ModifiedTables))
	}
	mod := diff.ModifiedTables[0]
	if len(mod.ModifiedColumns) != 1 {
		t.Fatalf("expected 1 modified column, got %d", len(mod.ModifiedColumns))
	}
	if mod.ModifiedColumns[0].Name != "name" {
		t.Errorf("expected modified column 'name', got %q", mod.ModifiedColumns[0].Name)
	}
}

func TestCosmeticTypeDiff_SQLite(t *testing.T) {
	adapter := sqlite.New()
	sm := NewSchemaManager(adapter)
	ctx := t.Context()

	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "snap.json")
	schemaPath := filepath.Join(tmpDir, "schema.sql")

	snapTables := []types.SchemaTable{
		{
			Name: "users",
			Columns: []types.SchemaColumn{
				{Name: "id", Type: "INTEGER", Nullable: true, IsPrimary: true},
				{Name: "email", Type: "TEXT", Nullable: false, IsUnique: true},
				{Name: "password_hash", Type: "TEXT", Nullable: false},
				{Name: "last_login", Type: "TEXT", Nullable: true},
				{Name: "created_at", Type: "TEXT", Nullable: false, Default: "CURRENT_TIMESTAMP"},
			},
		},
	}
	SaveSchemaSnapshot(snapshotPath, snapTables, nil)

	// Schema uses VARCHAR(255) and DATETIME — cosmetic differences in SQLite
	schemaSQL := `CREATE TABLE "users" (
		"id" INTEGER PRIMARY KEY,
		"email" VARCHAR(255) NOT NULL UNIQUE,
		"password_hash" VARCHAR(255) NOT NULL,
		"last_login" DATETIME,
		"created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		"is_active" BOOLEAN NOT NULL DEFAULT 1
	);`
	os.WriteFile(schemaPath, []byte(schemaSQL), 0644)

	diff, err := sm.GenerateSchemaDiff(ctx, schemaPath, snapshotPath)
	if err != nil {
		t.Fatalf("diff failed: %v", err)
	}

	if len(diff.ModifiedTables) != 1 {
		t.Fatalf("expected 1 modified table, got %d", len(diff.ModifiedTables))
	}
	mod := diff.ModifiedTables[0]

	// Cosmetic type differences ARE detected at the diff level (exact string comparison)
	if len(mod.ModifiedColumns) != 4 {
		t.Fatalf("expected 4 cosmetic modified columns at diff level, got %d", len(mod.ModifiedColumns))
	}

	// But only the truly new column should appear in NewColumns
	if len(mod.NewColumns) != 1 {
		t.Fatalf("expected 1 new column, got %d", len(mod.NewColumns))
	}
	if mod.NewColumns[0].Name != "is_active" {
		t.Errorf("expected new column 'is_active', got %q", mod.NewColumns[0].Name)
	}
}
