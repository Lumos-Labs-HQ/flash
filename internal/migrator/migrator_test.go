package migrator

import (
	"os"
	"strings"
	"testing"
)

// ── extractUpSQL ─────────────────────────────────────────────────────────────

func TestExtractUpSQL_WithMarkers(t *testing.T) {
	content := `-- Migration: test
-- Created: 2024-01-01

-- +migrate Up
CREATE TABLE users (id SERIAL PRIMARY KEY);
ALTER TABLE users ADD COLUMN email TEXT;

-- +migrate Down
DROP TABLE users;
`
	got := extractUpSQL(content)
	if !strings.Contains(got, "CREATE TABLE users") {
		t.Errorf("missing CREATE TABLE in up SQL: %q", got)
	}
	if !strings.Contains(got, "ALTER TABLE users") {
		t.Errorf("missing ALTER TABLE in up SQL: %q", got)
	}
	if strings.Contains(got, "DROP TABLE") {
		t.Errorf("down SQL leaked into up SQL: %q", got)
	}
}

func TestExtractUpSQL_NoMarkers_ReturnsAll(t *testing.T) {
	content := "CREATE TABLE users (id SERIAL PRIMARY KEY);"
	got := extractUpSQL(content)
	if got != content {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestExtractUpSQL_EmptyUpSection(t *testing.T) {
	content := `-- +migrate Up
-- +migrate Down
DROP TABLE users;
`
	got := extractUpSQL(content)
	if strings.Contains(got, "DROP TABLE") {
		t.Errorf("down SQL leaked into up SQL: %q", got)
	}
}

func TestExtractUpSQL_CaseInsensitiveMarkers(t *testing.T) {
	content := `-- +migrate up
CREATE TABLE a (id INT);
-- +migrate down
DROP TABLE a;
`
	got := extractUpSQL(content)
	if !strings.Contains(got, "CREATE TABLE a") {
		t.Errorf("missing CREATE TABLE: %q", got)
	}
	if strings.Contains(got, "DROP TABLE") {
		t.Errorf("down SQL leaked: %q", got)
	}
}

// ── extractDownSQL (via Migrator method) ─────────────────────────────────────

func TestExtractDownSQL_WithMarkers(t *testing.T) {
	m := &Migrator{}

	// Write a temp file
	dir := t.TempDir()
	path := dir + "/migration.sql"
	content := `-- +migrate Up
CREATE TABLE users (id SERIAL PRIMARY KEY);
-- +migrate Down
DROP TABLE users;
`
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}

	got := m.extractDownSQL(path)
	if !strings.Contains(got, "DROP TABLE users") {
		t.Errorf("missing DROP TABLE in down SQL: %q", got)
	}
	if strings.Contains(got, "CREATE TABLE") {
		t.Errorf("up SQL leaked into down SQL: %q", got)
	}
}

func TestExtractDownSQL_NoFile_ReturnsEmpty(t *testing.T) {
	m := &Migrator{}
	got := m.extractDownSQL("/nonexistent/path.sql")
	if got != "" {
		t.Errorf("expected empty string for missing file, got %q", got)
	}
}

// ── splitMigrationID ─────────────────────────────────────────────────────────

func TestSplitMigrationID_Standard(t *testing.T) {
	id, name := splitMigrationID("20251204234836_add_phone_column")
	if id != "20251204234836" {
		t.Errorf("id = %q, want 20251204234836", id)
	}
	if name != "add_phone_column" {
		t.Errorf("name = %q, want add_phone_column", name)
	}
}

func TestSplitMigrationID_Short(t *testing.T) {
	id, name := splitMigrationID("short")
	if id != "short" {
		t.Errorf("id = %q, want short", id)
	}
	if name != "" {
		t.Errorf("name = %q, want empty", name)
	}
}

func TestSplitMigrationID_NoUnderscore(t *testing.T) {
	id, name := splitMigrationID("20251204234836")
	if id != "20251204234836" {
		t.Errorf("id = %q, want 20251204234836", id)
	}
	_ = name // may be empty or the full string — both acceptable
}

// ── formatMigrationFileWithDown ───────────────────────────────────────────────

func TestFormatMigrationFileWithDown_ContainsSections(t *testing.T) {
	m := &Migrator{provider: "postgresql"}
	up := []string{"CREATE TABLE users (id SERIAL PRIMARY KEY);"}
	down := []string{"DROP TABLE IF EXISTS users CASCADE;"}
	result := m.formatMigrationFileWithDown("test migration", up, down)

	if !strings.Contains(result, "-- +migrate Up") {
		t.Error("missing +migrate Up marker")
	}
	if !strings.Contains(result, "-- +migrate Down") {
		t.Error("missing +migrate Down marker")
	}
	if !strings.Contains(result, "CREATE TABLE users") {
		t.Error("missing up SQL")
	}
	if !strings.Contains(result, "DROP TABLE IF EXISTS users") {
		t.Error("missing down SQL")
	}
	if !strings.Contains(result, "-- Migration: test migration") {
		t.Error("missing migration name header")
	}
}

func TestFormatMigrationFileWithDown_NilDown(t *testing.T) {
	m := &Migrator{provider: "postgresql"}
	result := m.formatMigrationFileWithDown("empty", nil, nil)
	if !strings.Contains(result, "-- +migrate Down") {
		t.Error("missing +migrate Down section even when nil")
	}
}

// helper
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
