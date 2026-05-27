package gencommon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

// ── GenerationCache ───────────────────────────────────────────────────────────

func TestNewGenerationCache_Defaults(t *testing.T) {
	// Run in a temp dir so it doesn't pick up a real cache file.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	c := NewGenerationCache()
	if c.Version != "1.0" {
		t.Errorf("Version = %q, want 1.0", c.Version)
	}
	if c.QueryFileChecksums == nil {
		t.Error("QueryFileChecksums is nil")
	}
}

func TestGenerationCache_ShouldRegenerateAll_SchemaChanged(t *testing.T) {
	c := &GenerationCache{
		SchemaChecksum: "old",
		ConfigChecksum: "cfg",
	}
	if !c.ShouldRegenerateAll("new", "cfg") {
		t.Error("should regenerate when schema changed")
	}
}

func TestGenerationCache_ShouldRegenerateAll_ConfigChanged(t *testing.T) {
	c := &GenerationCache{
		SchemaChecksum: "schema",
		ConfigChecksum: "old",
	}
	if !c.ShouldRegenerateAll("schema", "new") {
		t.Error("should regenerate when config changed")
	}
}

func TestGenerationCache_ShouldRegenerateAll_NoChange(t *testing.T) {
	c := &GenerationCache{
		SchemaChecksum: "schema",
		ConfigChecksum: "cfg",
	}
	if c.ShouldRegenerateAll("schema", "cfg") {
		t.Error("should NOT regenerate when nothing changed")
	}
}

func TestGenerationCache_ShouldRegenerateQuery_NewFile(t *testing.T) {
	c := &GenerationCache{QueryFileChecksums: map[string]string{}}
	if !c.ShouldRegenerateQuery("users.sql", "abc123") {
		t.Error("new file should require regeneration")
	}
}

func TestGenerationCache_ShouldRegenerateQuery_Changed(t *testing.T) {
	c := &GenerationCache{QueryFileChecksums: map[string]string{"users.sql": "old"}}
	if !c.ShouldRegenerateQuery("users.sql", "new") {
		t.Error("changed file should require regeneration")
	}
}

func TestGenerationCache_ShouldRegenerateQuery_Unchanged(t *testing.T) {
	c := &GenerationCache{QueryFileChecksums: map[string]string{"users.sql": "abc"}}
	if c.ShouldRegenerateQuery("users.sql", "abc") {
		t.Error("unchanged file should NOT require regeneration")
	}
}

func TestGenerationCache_UpdateAndRead(t *testing.T) {
	c := &GenerationCache{
		QueryFileChecksums:     map[string]string{},
		QueryTableDeps:         map[string][]string{},
		GeneratedFileChecksums: map[string]string{},
	}

	c.UpdateSchemaChecksum("schema123")
	c.UpdateConfigChecksum("cfg456")
	c.UpdateQueryChecksum("users.sql", "qhash")
	c.UpdateQueryDependencies("users.sql", []string{"users", "posts"})
	c.UpdateGeneratedFileChecksum("flash_gen/users.go", "genhash")
	c.MarkGeneration()

	if c.SchemaChecksum != "schema123" {
		t.Errorf("SchemaChecksum = %q", c.SchemaChecksum)
	}
	if c.ConfigChecksum != "cfg456" {
		t.Errorf("ConfigChecksum = %q", c.ConfigChecksum)
	}
	if c.QueryFileChecksums["users.sql"] != "qhash" {
		t.Errorf("QueryFileChecksums[users.sql] = %q", c.QueryFileChecksums["users.sql"])
	}
	if c.LastGeneration.IsZero() {
		t.Error("LastGeneration should not be zero after MarkGeneration")
	}
}

func TestGenerationCache_GetAffectedQueries(t *testing.T) {
	c := &GenerationCache{
		QueryTableDeps: map[string][]string{
			"users.sql": {"users"},
			"posts.sql": {"posts", "users"},
			"other.sql": {"orders"},
		},
	}
	affected := c.GetAffectedQueries([]string{"users"})
	affectedSet := map[string]bool{}
	for _, f := range affected {
		affectedSet[f] = true
	}
	if !affectedSet["users.sql"] {
		t.Error("users.sql should be affected")
	}
	if !affectedSet["posts.sql"] {
		t.Error("posts.sql should be affected (depends on users)")
	}
	if affectedSet["other.sql"] {
		t.Error("other.sql should NOT be affected")
	}
}

func TestGenerationCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	c := &GenerationCache{
		Version:                "1.0",
		SchemaChecksum:         "s1",
		ConfigChecksum:         "c1",
		QueryFileChecksums:     map[string]string{"q.sql": "h1"},
		QueryTableDeps:         map[string][]string{"q.sql": {"users"}},
		GeneratedFileChecksums: map[string]string{},
	}
	c.MarkGeneration()

	if err := c.Save(); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	c2 := &GenerationCache{
		QueryFileChecksums:     map[string]string{},
		QueryTableDeps:         map[string][]string{},
		GeneratedFileChecksums: map[string]string{},
	}
	if err := c2.Load(); err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if c2.SchemaChecksum != "s1" {
		t.Errorf("SchemaChecksum = %q, want s1", c2.SchemaChecksum)
	}
	if c2.QueryFileChecksums["q.sql"] != "h1" {
		t.Errorf("QueryFileChecksums[q.sql] = %q, want h1", c2.QueryFileChecksums["q.sql"])
	}
}

func TestGenerationCache_Clear(t *testing.T) {
	c := &GenerationCache{
		SchemaChecksum:         "s",
		ConfigChecksum:         "c",
		QueryFileChecksums:     map[string]string{"f": "h"},
		QueryTableDeps:         map[string][]string{"f": {"t"}},
		GeneratedFileChecksums: map[string]string{"g": "h"},
	}
	c.MarkGeneration()
	c.Clear()

	if c.SchemaChecksum != "" || c.ConfigChecksum != "" {
		t.Error("checksums should be empty after Clear")
	}
	if len(c.QueryFileChecksums) != 0 {
		t.Error("QueryFileChecksums should be empty after Clear")
	}
	if !c.LastGeneration.IsZero() {
		t.Error("LastGeneration should be zero after Clear")
	}
}

// ── ComputeFileChecksum ───────────────────────────────────────────────────────

func TestComputeFileChecksum_Deterministic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sql")
	os.WriteFile(path, []byte("SELECT 1;"), 0644)

	h1, err := ComputeFileChecksum(path)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	h2, _ := ComputeFileChecksum(path)
	if h1 != h2 {
		t.Errorf("checksum not deterministic: %q != %q", h1, h2)
	}
}

func TestComputeFileChecksum_DifferentContent(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.sql")
	p2 := filepath.Join(dir, "b.sql")
	os.WriteFile(p1, []byte("SELECT 1;"), 0644)
	os.WriteFile(p2, []byte("SELECT 2;"), 0644)

	h1, _ := ComputeFileChecksum(p1)
	h2, _ := ComputeFileChecksum(p2)
	if h1 == h2 {
		t.Error("different files should have different checksums")
	}
}

func TestComputeFileChecksum_MissingFile(t *testing.T) {
	_, err := ComputeFileChecksum("/nonexistent/path.sql")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// ── ComputeSchemaChecksum ─────────────────────────────────────────────────────

func TestComputeSchemaChecksum_Empty(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	c := &GenerationCache{}
	hash, err := c.ComputeSchemaChecksum(dir)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if hash != "" {
		t.Errorf("empty dir hash = %q, want empty", hash)
	}
}

func TestComputeSchemaChecksum_WithFiles(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(filepath.Join(dir, "users.sql"), []byte("CREATE TABLE users (id INT);"), 0644)

	c := &GenerationCache{}
	hash, err := c.ComputeSchemaChecksum(dir)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash for dir with SQL files")
	}
}

// ── ExtractTableDependencies ──────────────────────────────────────────────────

func TestExtractTableDependencies(t *testing.T) {
	queries := []*parser.Query{
		{SQL: "SELECT * FROM users WHERE id = $1", Columns: []*parser.QueryColumn{{Name: "id", Table: "users"}}},
		{SQL: "INSERT INTO posts (title) VALUES ($1)"},
	}
	deps := ExtractTableDependencies(queries)
	depSet := map[string]bool{}
	for _, d := range deps {
		depSet[d] = true
	}
	if !depSet["users"] {
		t.Error("users should be in dependencies")
	}
	if !depSet["posts"] {
		t.Error("posts should be in dependencies")
	}
}

// ── DetectSchemaChanges ───────────────────────────────────────────────────────

func TestDetectSchemaChanges_NewTable(t *testing.T) {
	old := &parser.Schema{Tables: []*parser.Table{}}
	new := &parser.Schema{Tables: []*parser.Table{{Name: "users", Columns: []*parser.Column{{Name: "id"}}}}}
	changed := DetectSchemaChanges(old, new)
	if len(changed) != 1 || changed[0] != "users" {
		t.Errorf("changed = %v, want [users]", changed)
	}
}

func TestDetectSchemaChanges_DeletedTable(t *testing.T) {
	old := &parser.Schema{Tables: []*parser.Table{{Name: "old_table"}}}
	new := &parser.Schema{Tables: []*parser.Table{}}
	changed := DetectSchemaChanges(old, new)
	if len(changed) != 1 || changed[0] != "old_table" {
		t.Errorf("changed = %v, want [old_table]", changed)
	}
}

func TestDetectSchemaChanges_ModifiedColumn(t *testing.T) {
	old := &parser.Schema{Tables: []*parser.Table{{
		Name:    "users",
		Columns: []*parser.Column{{Name: "id", Type: "INT"}},
	}}}
	new := &parser.Schema{Tables: []*parser.Table{{
		Name:    "users",
		Columns: []*parser.Column{{Name: "id", Type: "BIGINT"}},
	}}}
	changed := DetectSchemaChanges(old, new)
	if len(changed) != 1 || changed[0] != "users" {
		t.Errorf("changed = %v, want [users]", changed)
	}
}

func TestDetectSchemaChanges_NoChange(t *testing.T) {
	schema := &parser.Schema{Tables: []*parser.Table{{
		Name:    "users",
		Columns: []*parser.Column{{Name: "id", Type: "INT", Nullable: false}},
	}}}
	changed := DetectSchemaChanges(schema, schema)
	if len(changed) != 0 {
		t.Errorf("changed = %v, want empty", changed)
	}
}

func TestDetectSchemaChanges_NilInputs(t *testing.T) {
	if changed := DetectSchemaChanges(nil, nil); len(changed) != 0 {
		t.Errorf("nil inputs: changed = %v, want empty", changed)
	}
}
