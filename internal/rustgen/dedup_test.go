package rustgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
)

// TestGenerateDedupsIdenticalRowAndParamsStructs covers cross-file dedup:
// identical Row shapes across query files must be emitted once, with later
// occurrences as `pub type X = Y;` aliases importing the canonical struct.
func TestGenerateDedupsIdenticalRowAndParamsStructs(t *testing.T) {
	root := t.TempDir()
	schemaDir := filepath.Join(root, "db", "schema")
	queryDir := filepath.Join(root, "db", "queries")
	outDir := filepath.Join(root, "src", "flash_gen")
	for _, dir := range []string{schemaDir, queryDir, outDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	schemaSQL := `CREATE TABLE accounts (
	id INTEGER PRIMARY KEY,
	email TEXT NOT NULL,
	display_name TEXT,
	is_active INTEGER NOT NULL
);
CREATE TABLE extra_settings (
	id INTEGER PRIMARY KEY,
	account_id INTEGER NOT NULL,
	flag INTEGER NOT NULL
);`
	if err := os.WriteFile(filepath.Join(schemaDir, "schema.sql"), []byte(schemaSQL), 0644); err != nil {
		t.Fatal(err)
	}

	// alpha.sql defines the canonical row shape; beta.sql repeats the exact
	// same SELECT shape and an identical params shape across files.
	alphaSQL := `-- name: AlphaList :many
SELECT id AS id, email AS email, display_name AS display_name FROM accounts WHERE email = ?;

-- name: AlphaUpsert :exec
INSERT INTO accounts (email, display_name, is_active) VALUES (?, ?, ?);`
	betaSQL := `-- name: BetaList :many
SELECT id AS id, email AS email, display_name AS display_name FROM accounts WHERE email = ?;

-- name: BetaUpsert :exec
INSERT INTO accounts (email, display_name, is_active) VALUES (?, ?, ?);`
	if err := os.WriteFile(filepath.Join(queryDir, "alpha.sql"), []byte(alphaSQL), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(queryDir, "beta.sql"), []byte(betaSQL), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SchemaDir:  schemaDir,
		SchemaPath: filepath.Join(schemaDir, "schema.sql"),
		Queries:    queryDir,
		Database:   config.Database{Provider: "sqlite"},
		Gen:        config.Gen{Rust: config.RustGen{Enabled: true, Out: outDir, Driver: "sqlx"}},
	}
	if err := New(cfg).Generate(); err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	alpha, err := os.ReadFile(filepath.Join(outDir, "alpha.rs"))
	if err != nil {
		t.Fatal(err)
	}
	beta, err := os.ReadFile(filepath.Join(outDir, "beta.rs"))
	if err != nil {
		t.Fatal(err)
	}

	// Alpha owns the canonical structs.
	if !strings.Contains(string(alpha), "pub struct AlphaListRow {") {
		t.Errorf("alpha.rs missing canonical AlphaListRow struct:\n%s", alpha)
	}
	if !strings.Contains(string(alpha), "pub struct AlphaUpsertParams {") {
		t.Errorf("alpha.rs missing canonical AlphaUpsertParams struct:\n%s", alpha)
	}

	// Beta aliases to the canonical structs and imports them.
	if !strings.Contains(string(beta), "pub type BetaListRow = AlphaListRow;") {
		t.Errorf("beta.rs missing BetaListRow alias:\n%s", beta)
	}
	if !strings.Contains(string(beta), "pub type BetaUpsertParams = AlphaUpsertParams;") {
		t.Errorf("beta.rs missing BetaUpsertParams alias:\n%s", beta)
	}
	if !strings.Contains(string(beta), "use super::alpha::AlphaListRow;") ||
		!strings.Contains(string(beta), "use super::alpha::AlphaUpsertParams;") {
		t.Errorf("beta.rs missing imports of canonical structs:\n%s", beta)
	}
	if strings.Contains(string(beta), "pub struct BetaListRow {") {
		t.Errorf("beta.rs must not re-emit the duplicate BetaListRow struct:\n%s", beta)
	}

	// Methods must reference the aliased types so callers keep using their own
	// module's name.
	if !strings.Contains(string(beta), "Vec<BetaListRow>") {
		t.Errorf("beta.rs method must return Vec<BetaListRow>:\n%s", beta)
	}
	if !strings.Contains(string(beta), "params: &BetaUpsertParams") {
		t.Errorf("beta.rs method must take &BetaUpsertParams:\n%s", beta)
	}
}

// TestGenerateDedupSameFile verifies that identical rows within ONE file also
// dedup (first canonical, later aliases) without emitting imports.
func TestGenerateDedupSameFile(t *testing.T) {
	root := t.TempDir()
	schemaDir := filepath.Join(root, "db", "schema")
	queryDir := filepath.Join(root, "db", "queries")
	outDir := filepath.Join(root, "src", "flash_gen")
	for _, dir := range []string{schemaDir, queryDir, outDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	schemaSQL := `CREATE TABLE widgets (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	weight REAL,
	hidden INTEGER NOT NULL
);`
	if err := os.WriteFile(filepath.Join(schemaDir, "schema.sql"), []byte(schemaSQL), 0644); err != nil {
		t.Fatal(err)
	}
	queriesSQL := `-- name: WidgetsActive :many
SELECT id AS id, name AS name, weight AS weight FROM widgets WHERE weight > 0.5;

-- name: WidgetsHeavy :many
SELECT id AS id, name AS name, weight AS weight FROM widgets WHERE weight > 10.0;`
	if err := os.WriteFile(filepath.Join(queryDir, "widgets.sql"), []byte(queriesSQL), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SchemaDir:  schemaDir,
		SchemaPath: filepath.Join(schemaDir, "schema.sql"),
		Queries:    queryDir,
		Database:   config.Database{Provider: "sqlite"},
		Gen:        config.Gen{Rust: config.RustGen{Enabled: true, Out: outDir, Driver: "sqlx"}},
	}
	if err := New(cfg).Generate(); err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	out, err := os.ReadFile(filepath.Join(outDir, "widgets.rs"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "pub struct WidgetsActiveRow {") {
		t.Errorf("missing canonical WidgetsActiveRow:\n%s", out)
	}
	if !strings.Contains(string(out), "pub type WidgetsHeavyRow = WidgetsActiveRow;") {
		t.Errorf("missing WidgetsHeavyRow alias:\n%s", out)
	}
	if strings.Contains(string(out), "pub struct WidgetsHeavyRow {") {
		t.Errorf("duplicate WidgetsHeavyRow struct must not be emitted:\n%s", out)
	}
	if !strings.Contains(string(out), "Vec<WidgetsHeavyRow>") {
		t.Errorf("widgets_heavy must still return Vec<WidgetsHeavyRow>:\n%s", out)
	}
}

// TestGenerateDedupDifferentShapeNoAlias verifies near-identical shapes that
// differ in one field type do NOT alias.
func TestGenerateDedupDifferentShapeNoAlias(t *testing.T) {
	root := t.TempDir()
	schemaDir := filepath.Join(root, "db", "schema")
	queryDir := filepath.Join(root, "db", "queries")
	outDir := filepath.Join(root, "src", "flash_gen")
	for _, dir := range []string{schemaDir, queryDir, outDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	schemaSQL := `CREATE TABLE gadgets (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	weight REAL,
	count INTEGER
);`
	if err := os.WriteFile(filepath.Join(schemaDir, "schema.sql"), []byte(schemaSQL), 0644); err != nil {
		t.Fatal(err)
	}
	queriesSQL := `-- name: GadgetsA :many
SELECT id AS id, name AS name, weight AS weight FROM gadgets WHERE weight > 0.5;

-- name: GadgetsB :many
SELECT id AS id, name AS name, count AS count FROM gadgets WHERE count > 3;`
	if err := os.WriteFile(filepath.Join(queryDir, "gadgets.sql"), []byte(queriesSQL), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SchemaDir:  schemaDir,
		SchemaPath: filepath.Join(schemaDir, "schema.sql"),
		Queries:    queryDir,
		Database:   config.Database{Provider: "sqlite"},
		Gen:        config.Gen{Rust: config.RustGen{Enabled: true, Out: outDir, Driver: "sqlx"}},
	}
	if err := New(cfg).Generate(); err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	out, err := os.ReadFile(filepath.Join(outDir, "gadgets.rs"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "pub struct GadgetsARow {") ||
		!strings.Contains(string(out), "pub struct GadgetsBRow {") {
		t.Errorf("both distinct structs must be emitted:\n%s", out)
	}
	if strings.Contains(string(out), "pub type GadgetsBRow =") {
		t.Errorf("different shape must not alias:\n%s", out)
	}
}
