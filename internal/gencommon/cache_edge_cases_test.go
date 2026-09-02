package gencommon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

// ── ShouldRegenerateQuery / ShouldRegenerateAll (edge complements) ────────────

func TestShouldRegenerateAll_MixedChanges(t *testing.T) {
	c := NewGenerationCacheWithDir(t.TempDir())
	c.UpdateSchemaChecksum("s1")
	c.UpdateConfigChecksum("c1")
	if !c.ShouldRegenerateAll("s2", "c1") {
		t.Error("schema change must trigger full regen")
	}
	if !c.ShouldRegenerateAll("s1", "c2") {
		t.Error("config change must trigger full regen")
	}
	if c.ShouldRegenerateAll("s1", "c1") {
		t.Error("no changes must not trigger full regen")
	}
}

// A fresh cache has empty checksums. When the computed schema hash is also
// empty (e.g. empty schema dir), ShouldRegenerateAll returns false — the
// empty==empty comparison makes an empty-schema first run look "unchanged".
// Documents the quirk; with a non-empty schema dir the hash is non-empty and
// a full regen is triggered correctly.
func TestShouldRegenerateAll_FreshCache(t *testing.T) {
	c := NewGenerationCacheWithDir(t.TempDir())
	if c.ShouldRegenerateAll("", "") {
		t.Error("empty-vs-empty must NOT trigger full regen (documents quirk)")
	}
	if !c.ShouldRegenerateAll("somehash", "cfg") {
		t.Error("fresh cache with a real schema hash must fully regenerate")
	}
}

// ── GetAffectedQueries (edge complements) ─────────────────────────────────────

func TestGetAffectedQueries_MultiTableAndMissing(t *testing.T) {
	c := NewGenerationCacheWithDir(t.TempDir())
	c.UpdateQueryDependencies("users.sql", []string{"users", "posts"})
	c.UpdateQueryDependencies("audit.sql", []string{"audit_log"})
	c.UpdateQueryDependencies("none.sql", []string{"other"})

	affected := c.GetAffectedQueries([]string{"users"})
	if len(affected) != 1 || affected[0] != "users.sql" {
		t.Errorf("affected = %v, want [users.sql]", affected)
	}

	affected = c.GetAffectedQueries([]string{"users", "audit_log"})
	if len(affected) != 2 {
		t.Errorf("affected = %v, want 2 files", affected)
	}

	if affected := c.GetAffectedQueries([]string{"missing"}); len(affected) != 0 {
		t.Errorf("unaffected tables must return empty, got %v", affected)
	}
	if affected := c.GetAffectedQueries(nil); len(affected) != 0 {
		t.Errorf("nil changed-tables must return empty, got %v", affected)
	}
}

// ── DetectSchemaChanges ───────────────────────────────────────────────────────

func mkTable(name string, cols ...*parser.Column) *parser.Table {
	tbl := &parser.Table{Name: name}
	for _, c := range cols {
		tbl.Columns = append(tbl.Columns, c)
	}
	return tbl
}

func TestDetectSchemaChanges_AllMutationKinds(t *testing.T) {
	old := &parser.Schema{
		Tables: []*parser.Table{
			mkTable("users",
				&parser.Column{Name: "id", Type: "SERIAL"},
				&parser.Column{Name: "email", Type: "TEXT"},
			),
		},
	}

	// Identical schema -> no changes
	same := &parser.Schema{
		Tables: []*parser.Table{
			mkTable("users",
				&parser.Column{Name: "id", Type: "SERIAL"},
				&parser.Column{Name: "email", Type: "TEXT"},
			),
		},
	}
	if changed := DetectSchemaChanges(old, same); len(changed) != 0 {
		t.Errorf("identical schemas must yield no changes, got %v", changed)
	}

	// New table
	withNew := &parser.Schema{
		Tables: []*parser.Table{
			old.Tables[0],
			mkTable("posts", &parser.Column{Name: "id", Type: "SERIAL"}),
		},
	}
	if changed := DetectSchemaChanges(old, withNew); len(changed) != 1 || changed[0] != "posts" {
		t.Errorf("new table not detected: %v", changed)
	}

	// Deleted table
	if changed := DetectSchemaChanges(withNew, old); len(changed) != 1 || changed[0] != "posts" {
		t.Errorf("deleted table not detected: %v", changed)
	}

	// Column added
	colAdded := &parser.Schema{
		Tables: []*parser.Table{
			mkTable("users",
				&parser.Column{Name: "id", Type: "SERIAL"},
				&parser.Column{Name: "email", Type: "TEXT"},
				&parser.Column{Name: "age", Type: "INT"},
			),
		},
	}
	if changed := DetectSchemaChanges(old, colAdded); len(changed) != 1 || changed[0] != "users" {
		t.Errorf("added column not detected: %v", changed)
	}

	// Column type changed
	typeChanged := &parser.Schema{
		Tables: []*parser.Table{
			mkTable("users",
				&parser.Column{Name: "id", Type: "SERIAL"},
				&parser.Column{Name: "email", Type: "VARCHAR(255)"},
			),
		},
	}
	if changed := DetectSchemaChanges(old, typeChanged); len(changed) != 1 || changed[0] != "users" {
		t.Errorf("type change not detected: %v", changed)
	}

	// Column removed (count differs)
	colRemoved := &parser.Schema{
		Tables: []*parser.Table{
			mkTable("users", &parser.Column{Name: "id", Type: "SERIAL"}),
		},
	}
	if changed := DetectSchemaChanges(old, colRemoved); len(changed) != 1 || changed[0] != "users" {
		t.Errorf("removed column not detected: %v", changed)
	}
}

func TestDetectSchemaChanges_NilInputsEdge(t *testing.T) {
	if changed := DetectSchemaChanges(nil, &parser.Schema{}); changed != nil {
		t.Errorf("nil old schema must yield nil, got %v", changed)
	}
	if changed := DetectSchemaChanges(&parser.Schema{}, nil); changed != nil {
		t.Errorf("nil new schema must yield nil, got %v", changed)
	}
}

// Column order changes but content is equal: current tableChanged uses column
// maps so reordering alone should NOT report a change.
func TestDetectSchemaChanges_ColumnReorderNotAChange(t *testing.T) {
	a := &parser.Schema{Tables: []*parser.Table{mkTable("t",
		&parser.Column{Name: "a", Type: "INT"},
		&parser.Column{Name: "b", Type: "TEXT"},
	)}}
	b := &parser.Schema{Tables: []*parser.Table{mkTable("t",
		&parser.Column{Name: "b", Type: "TEXT"},
		&parser.Column{Name: "a", Type: "INT"},
	)}}
	if changed := DetectSchemaChanges(a, b); len(changed) != 0 {
		t.Errorf("column reordering must not be a change (name-keyed compare), got %v", changed)
	}
}

// ── ComputeSchemaChecksum ─────────────────────────────────────────────────────

func TestComputeSchemaChecksum_DeterministicAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.sql"), []byte("CREATE TABLE a (id INT);"), 0644)
	c1 := NewGenerationCacheWithDir(t.TempDir())
	c2 := NewGenerationCacheWithDir(t.TempDir())
	h1, _ := c1.ComputeSchemaChecksum(dir)
	h2, _ := c2.ComputeSchemaChecksum(dir)
	if h1 == "" {
		t.Fatal("checksum must not be empty for a non-empty schema dir")
	}
	if h1 != h2 {
		t.Errorf("checksum must be deterministic across cache instances: %s vs %s", h1, h2)
	}
}

func TestComputeSchemaChecksum_ContentMutationDetected(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.sql"), []byte("CREATE TABLE a (id INT);"), 0644)
	c := NewGenerationCacheWithDir(t.TempDir())
	h1, _ := c.ComputeSchemaChecksum(dir)
	os.WriteFile(filepath.Join(dir, "a.sql"), []byte("CREATE TABLE a (id BIGINT);"), 0644)
	h2, _ := c.ComputeSchemaChecksum(dir)
	if h1 == h2 {
		t.Error("content change must change the checksum")
	}
}

func TestComputeSchemaChecksum_FileRenameDetected(t *testing.T) {
	// The filename is part of the hash — renaming a file (same content) must
	// change the checksum so dependent generators regenerate.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.sql"), []byte("CREATE TABLE a (id INT);"), 0644)
	c := NewGenerationCacheWithDir(t.TempDir())
	h1, _ := c.ComputeSchemaChecksum(dir)
	os.Rename(filepath.Join(dir, "a.sql"), filepath.Join(dir, "b.sql"))
	h2, _ := c.ComputeSchemaChecksum(dir)
	if h1 == h2 {
		t.Error("file rename must change the checksum (filename is hashed)")
	}
}

func TestComputeSchemaChecksum_NoFilesIsEmpty(t *testing.T) {
	dir := t.TempDir()
	c := NewGenerationCacheWithDir(t.TempDir())
	h, err := c.ComputeSchemaChecksum(dir)
	if err != nil {
		t.Fatalf("empty dir must not error: %v", err)
	}
	if h != "" {
		t.Errorf("empty dir checksum = %q, want empty string", h)
	}
}

// ── Clear ─────────────────────────────────────────────────────────────────────

func TestClear_WipesAllState(t *testing.T) {
	dir := t.TempDir()
	c := NewGenerationCacheWithDir(dir)
	c.UpdateQueryChecksum("f.sql", "h")
	c.UpdateSchemaChecksum("sh")
	c.UpdateConfigChecksum("ch")
	c.UpdateQueryDependencies("f.sql", []string{"t"})
	c.UpdateGeneratedFileChecksum("f.go", "gh")
	c.MarkGeneration()

	c.Clear()

	if c.SchemaChecksum != "" || c.ConfigChecksum != "" {
		t.Error("checksums not cleared")
	}
	if len(c.QueryFileChecksums) != 0 || len(c.QueryTableDeps) != 0 || len(c.GeneratedFileChecksums) != 0 {
		t.Error("maps not cleared")
	}
	if !c.LastGeneration.IsZero() {
		t.Error("LastGeneration not cleared")
	}
}

// ── Save/Load round-trips ─────────────────────────────────────────────────────

func TestSaveLoad_FullRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := NewGenerationCacheWithDir(dir)
	c.UpdateQueryChecksum("f.sql", "h1")
	c.UpdateQueryChecksum("g.sql", "h2")
	c.UpdateSchemaChecksum("sh")
	c.UpdateConfigChecksum("ch")
	c.UpdateQueryDependencies("f.sql", []string{"users"})
	c.UpdateGeneratedFileChecksum("f.go", "gh")
	c.MarkGeneration()

	if err := c.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded := NewGenerationCacheWithDir(dir)
	if loaded.QueryFileChecksums["f.sql"] != "h1" || loaded.QueryFileChecksums["g.sql"] != "h2" {
		t.Errorf("query checksums not persisted: %v", loaded.QueryFileChecksums)
	}
	if loaded.SchemaChecksum != "sh" || loaded.ConfigChecksum != "ch" {
		t.Errorf("schema/config checksums not persisted")
	}
	if len(loaded.QueryTableDeps["f.sql"]) != 1 || loaded.QueryTableDeps["f.sql"][0] != "users" {
		t.Errorf("deps not persisted: %v", loaded.QueryTableDeps)
	}
	if loaded.GeneratedFileChecksums["f.go"] != "gh" {
		t.Errorf("generated checksums not persisted")
	}
	if loaded.LastGeneration.IsZero() {
		t.Error("LastGeneration not persisted")
	}
}

func TestSaveLoad_DefaultDir(t *testing.T) {
	// NewGenerationCache() defaults OutDir to "flash_gen" — cache must be
	// written under ./flash_gen relative to cwd.
	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	c := NewGenerationCache()
	c.UpdateQueryChecksum("f.sql", "h")
	if err := c.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "flash_gen", ".flash_cache.json")); err != nil {
		t.Errorf("default cache file missing: %v", err)
	}
}

func TestLoad_MissingFileIsNotAnError(t *testing.T) {
	c := NewGenerationCacheWithDir(t.TempDir())
	if err := c.Load(); err != nil {
		t.Errorf("missing cache file must not error, got %v", err)
	}
}

func TestLoad_DirectoryAsCacheFile(t *testing.T) {
	// A directory where the cache file should be — Load must fail loudly
	// rather than silently misbehave.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, cacheFileName), 0755); err != nil {
		t.Fatal(err)
	}
	c := NewGenerationCacheWithDir(dir)
	if err := c.Load(); err == nil {
		t.Error("directory-in-place-of-cache-file must return an error")
	}
}

// ── ComputeFileChecksum ───────────────────────────────────────────────────────

func TestComputeFileChecksum_StableAndDistinct(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	os.WriteFile(p, []byte("hello"), 0644)
	h1, err := ComputeFileChecksum(p)
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := ComputeFileChecksum(p)
	if h1 != h2 {
		t.Errorf("checksum not stable: %s vs %s", h1, h2)
	}
	os.WriteFile(p, []byte("world"), 0644)
	h3, _ := ComputeFileChecksum(p)
	if h1 == h3 {
		t.Error("different content must produce different checksums")
	}
	if len(h1) != 64 { // SHA256 hex
		t.Errorf("checksum length = %d, want 64", len(h1))
	}
}

func TestComputeFileChecksum_AbsentPathErrors(t *testing.T) {
	if _, err := ComputeFileChecksum(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("missing file must error")
	}
}

// ── Cross-generator isolation ─────────────────────────────────────────────────

func TestCacheIsolation_BetweenOutputDirs(t *testing.T) {
	// Two generators with different out dirs must not see each other's caches.
	root := t.TempDir()
	dirA := filepath.Join(root, "gen_go")
	dirB := filepath.Join(root, "gen_js")

	a := NewGenerationCacheWithDir(dirA)
	a.UpdateQueryChecksum("f.sql", "go-hash")
	a.MarkGeneration()
	if err := a.Save(); err != nil {
		t.Fatal(err)
	}

	b := NewGenerationCacheWithDir(dirB)
	if _, ok := b.QueryFileChecksums["f.sql"]; ok {
		t.Error("cache from dir A leaked into dir B")
	}
}
