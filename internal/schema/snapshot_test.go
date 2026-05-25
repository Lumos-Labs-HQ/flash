package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func TestLoadSchemaSnapshot_MissingFile(t *testing.T) {
	snap, err := LoadSchemaSnapshot("/nonexistent/path/snap.json")
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if snap != nil {
		t.Error("expected nil snapshot for missing file")
	}
}

func TestLoadSchemaSnapshot_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "snap.json")
	if err := os.WriteFile(path, []byte("not-json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadSchemaSnapshot(path)
	if err == nil {
		t.Fatal("expected error for corrupted snapshot")
	}
}

func TestSaveAndLoadSchemaSnapshot_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".flash", "schema_snapshot.json")

	tables := []types.SchemaTable{
		{
			Name: "users",
			Columns: []types.SchemaColumn{
				{Name: "id", Type: "INTEGER", IsPrimary: true},
				{Name: "email", Type: "TEXT", Nullable: false, IsUnique: true},
			},
			Indexes: []types.SchemaIndex{
				{Name: "idx_email", Table: "users", Columns: []string{"email"}, Unique: true},
			},
		},
	}

	enums := []types.SchemaEnum{
		{Name: "status", Values: []string{"active", "inactive"}},
	}

	if err := SaveSchemaSnapshot(path, tables, enums); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	snap, err := LoadSchemaSnapshot(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if snap == nil {
		t.Fatal("expected snapshot, got nil")
	}
	if snap.Version != snapshotVersion {
		t.Errorf("version = %q, want %q", snap.Version, snapshotVersion)
	}
	if len(snap.Tables) != 1 {
		t.Fatalf("tables = %d, want 1", len(snap.Tables))
	}
	if snap.Tables[0].Name != "users" {
		t.Errorf("table name = %q, want users", snap.Tables[0].Name)
	}
	if len(snap.Tables[0].Columns) != 2 {
		t.Fatalf("columns = %d, want 2", len(snap.Tables[0].Columns))
	}
	if len(snap.Enums) != 1 {
		t.Fatalf("enums = %d, want 1", len(snap.Enums))
	}
}

func TestGenerateSchemaDiff_UsesSnapshot(t *testing.T) {
	sm := newSM()
	ctx := t.Context()

	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "snap.json")
	schemaPath := filepath.Join(tmpDir, "schema.sql")

	// 1. Create a snapshot representing schema v1 (users with id, name)
	snapTables := []types.SchemaTable{
		{
			Name: "users",
			Columns: []types.SchemaColumn{
				{Name: "id", Type: "INTEGER", IsPrimary: true},
				{Name: "name", Type: "TEXT", Nullable: false},
			},
		},
	}
	if err := SaveSchemaSnapshot(snapshotPath, snapTables, nil); err != nil {
		t.Fatal(err)
	}

	// 2. Target schema adds email column
	schemaSQL := `CREATE TABLE "users" (
		"id" INTEGER PRIMARY KEY,
		"name" TEXT NOT NULL,
		"email" TEXT UNIQUE NOT NULL
	);`
	if err := os.WriteFile(schemaPath, []byte(schemaSQL), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Diff should use snapshot, not the empty DB
	diff, err := sm.GenerateSchemaDiff(ctx, schemaPath, snapshotPath)
	if err != nil {
		t.Fatalf("diff failed: %v", err)
	}

	if len(diff.NewTables) != 0 {
		t.Errorf("expected 0 new tables (users exists in snapshot), got %d", len(diff.NewTables))
	}
	if len(diff.ModifiedTables) != 1 {
		t.Fatalf("expected 1 modified table, got %d", len(diff.ModifiedTables))
	}
	mod := diff.ModifiedTables[0]
	if mod.Name != "users" {
		t.Errorf("expected table 'users', got %q", mod.Name)
	}
	if len(mod.NewColumns) != 1 {
		t.Fatalf("expected 1 new column, got %d", len(mod.NewColumns))
	}
	if mod.NewColumns[0].Name != "email" {
		t.Errorf("expected new column 'email', got %q", mod.NewColumns[0].Name)
	}
}

func TestGenerateSchemaDiff_DetectsModifiedColumn(t *testing.T) {
	sm := newSM()
	ctx := t.Context()

	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "snap.json")
	schemaPath := filepath.Join(tmpDir, "schema.sql")

	// Snapshot has email as TEXT (match what the parser produces)
	snapTables := []types.SchemaTable{
		{
			Name: "users",
			Columns: []types.SchemaColumn{
				{Name: "id", Type: "INTEGER", Nullable: true, IsPrimary: true},
				{Name: "email", Type: "TEXT", Nullable: false, IsUnique: true},
			},
		},
	}
	if err := SaveSchemaSnapshot(snapshotPath, snapTables, nil); err != nil {
		t.Fatal(err)
	}

	// Target schema changes email to VARCHAR(255)
	schemaSQL := `CREATE TABLE "users" (
		"id" INTEGER PRIMARY KEY,
		"email" VARCHAR(255) NOT NULL UNIQUE
	);`
	if err := os.WriteFile(schemaPath, []byte(schemaSQL), 0644); err != nil {
		t.Fatal(err)
	}

	diff, err := sm.GenerateSchemaDiff(ctx, schemaPath, snapshotPath)
	if err != nil {
		t.Fatalf("diff failed: %v", err)
	}

	if len(diff.ModifiedTables) != 1 {
		t.Fatalf("expected 1 modified table, got %d", len(diff.ModifiedTables))
	}
	mod := diff.ModifiedTables[0]
	if len(mod.ModifiedColumns) != 1 {
		t.Fatalf("expected 1 modified column, got %d", len(mod.ModifiedColumns))
	}
	colDiff := mod.ModifiedColumns[0]
	if colDiff.Name != "email" {
		t.Errorf("expected modified column 'email', got %q", colDiff.Name)
	}
	if colDiff.OldType != "TEXT" {
		t.Errorf("expected old type 'TEXT', got %q", colDiff.OldType)
	}
	if colDiff.NewType != "VARCHAR(255)" {
		t.Errorf("expected new type 'VARCHAR(255)', got %q", colDiff.NewType)
	}
	if !strings.Contains(strings.Join(colDiff.Changes, ""), "type changed") {
		t.Errorf("expected changes to mention type change, got %v", colDiff.Changes)
	}
}

