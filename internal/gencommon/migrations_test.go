package gencommon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEmbeddedMigrations(t *testing.T) {
	dir := t.TempDir()
	content := "-- +migrate Up\nCREATE TABLE users (id INTEGER);\nINSERT INTO users VALUES (1);\n-- +migrate Down\nDROP TABLE users;\n"
	path := filepath.Join(dir, "20260825010101_init.sql")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadEmbeddedMigrations(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "20260825010101_init" || got[0].Name != "init" {
		t.Fatalf("unexpected migrations: %#v", got)
	}
	if len(got[0].Statements) != 2 {
		t.Fatalf("statements = %d, want 2", len(got[0].Statements))
	}
	if got[0].Checksum == "" {
		t.Fatal("checksum should be populated")
	}
}

func TestLoadEmbeddedMigrations_MissingDir(t *testing.T) {
	got, err := LoadEmbeddedMigrations(filepath.Join(t.TempDir(), "missing"))
	if err != nil || len(got) != 0 {
		t.Fatalf("got %#v, %v; want empty migrations", got, err)
	}
}
