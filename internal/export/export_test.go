package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func makeBackup(tables map[string][]map[string]interface{}) types.BackupData {
	t := make(map[string]interface{}, len(tables))
	for k, v := range tables {
		t[k] = v
	}
	return types.BackupData{
		Timestamp: "2024-01-01 00:00:00",
		Version:   "1.0",
		Tables:    t,
		Comment:   "test",
	}
}

// ── exportToJSON ──────────────────────────────────────────────────────────────

func TestExportToJSON_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	data := makeBackup(map[string][]map[string]interface{}{
		"users": {{"id": 1, "email": "a@b.com"}},
	})

	path, err := exportToJSON(data, dir)
	if err != nil {
		t.Fatalf("exportToJSON error: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("output file not created: %s", path)
	}
	if !strings.HasSuffix(path, ".json") {
		t.Errorf("expected .json extension, got %q", path)
	}
}

func TestExportToJSON_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	data := makeBackup(map[string][]map[string]interface{}{
		"users": {{"id": 1, "name": "Alice"}},
	})

	path, err := exportToJSON(data, dir)
	if err != nil {
		t.Fatalf("exportToJSON error: %v", err)
	}

	raw, _ := os.ReadFile(path)
	var parsed types.BackupData
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

func TestExportToJSON_EmptyTables(t *testing.T) {
	dir := t.TempDir()
	data := makeBackup(nil)
	path, err := exportToJSON(data, dir)
	if err != nil {
		t.Fatalf("exportToJSON error: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path even for empty data")
	}
}

// ── exportToCSV ───────────────────────────────────────────────────────────────

func TestExportToCSV_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	data := makeBackup(map[string][]map[string]interface{}{
		"users": {{"id": 1, "email": "a@b.com"}, {"id": 2, "email": "c@d.com"}},
	})

	path, err := exportToCSV(data, dir)
	if err != nil {
		t.Fatalf("exportToCSV error: %v", err)
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Errorf("expected directory at %s", path)
	}
}

func TestExportToCSV_CreatesCSVFile(t *testing.T) {
	dir := t.TempDir()
	data := makeBackup(map[string][]map[string]interface{}{
		"users": {{"id": 1, "email": "a@b.com"}},
	})

	csvDir, err := exportToCSV(data, dir)
	if err != nil {
		t.Fatalf("exportToCSV error: %v", err)
	}

	entries, _ := os.ReadDir(csvDir)
	hasCSV := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".csv") {
			hasCSV = true
			break
		}
	}
	if !hasCSV {
		t.Error("no .csv file created")
	}
}

func TestExportToCSV_SkipsEmptyTables(t *testing.T) {
	dir := t.TempDir()
	data := makeBackup(map[string][]map[string]interface{}{
		"empty_table": {},
	})

	csvDir, err := exportToCSV(data, dir)
	if err != nil {
		t.Fatalf("exportToCSV error: %v", err)
	}

	entries, _ := os.ReadDir(csvDir)
	for _, e := range entries {
		if e.Name() == "empty_table.csv" {
			t.Error("empty table should not produce a CSV file")
		}
	}
}

// ── exportToSQLite ────────────────────────────────────────────────────────────

func TestExportToSQLite_CreatesDB(t *testing.T) {
	dir := t.TempDir()
	data := makeBackup(map[string][]map[string]interface{}{
		"users": {{"id": "1", "email": "a@b.com"}},
	})

	path, err := exportToSQLite(nil, nil, data, dir)
	if err != nil {
		t.Fatalf("exportToSQLite error: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("SQLite file not created: %s", path)
	}
	if !strings.HasSuffix(path, ".db") {
		t.Errorf("expected .db extension, got %q", path)
	}
}

// ── buildColumnDefs / buildInsertSQL ─────────────────────────────────────────

func TestBuildColumnDefs(t *testing.T) {
	defs := buildColumnDefs([]string{"id", "email", "name"})
	for _, col := range []string{"id", "email", "name"} {
		if !strings.Contains(defs, col) {
			t.Errorf("buildColumnDefs missing %q: %s", col, defs)
		}
	}
}

func TestBuildInsertSQL(t *testing.T) {
	sql := buildInsertSQL("users", []string{"id", "email"})
	if !strings.Contains(sql, "INSERT INTO users") {
		t.Errorf("missing INSERT INTO users: %s", sql)
	}
	if !strings.Contains(sql, "id") || !strings.Contains(sql, "email") {
		t.Errorf("missing columns: %s", sql)
	}
	if strings.Count(sql, "?") != 2 {
		t.Errorf("expected 2 placeholders, got: %s", sql)
	}
}

// ── exportPath creation ───────────────────────────────────────────────────────

func TestExportToJSON_CreatesExportDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new", "nested", "dir")
	data := makeBackup(nil)
	_, err := exportToJSON(data, dir)
	if err != nil {
		t.Fatalf("should create nested dirs: %v", err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("export directory not created")
	}
}
