package branch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func newMeta(t *testing.T) *MetadataManager {
	t.Helper()
	return NewMetadataManager(filepath.Join(t.TempDir(), "migrations"))
}

// ── BranchStore ───────────────────────────────────────────────────────────────

func TestBranchStore_GetBranch_Found(t *testing.T) {
	store := &BranchStore{
		Current: "main",
		Branches: []*BranchMetadata{
			{Name: "main", Schema: "public", IsDefault: true},
			{Name: "feature", Schema: "flash_branch_feature"},
		},
	}
	b := store.GetBranch("feature")
	if b == nil || b.Name != "feature" {
		t.Errorf("GetBranch(feature) = %v, want feature branch", b)
	}
}

func TestBranchStore_GetBranch_NotFound(t *testing.T) {
	store := &BranchStore{Branches: []*BranchMetadata{{Name: "main"}}}
	if b := store.GetBranch("nonexistent"); b != nil {
		t.Errorf("expected nil, got %v", b)
	}
}

func TestBranchStore_AddBranch(t *testing.T) {
	store := &BranchStore{Branches: []*BranchMetadata{{Name: "main"}}}
	if err := store.AddBranch(&BranchMetadata{Name: "feature"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.Branches) != 2 {
		t.Errorf("branches = %d, want 2", len(store.Branches))
	}
}

func TestBranchStore_AddBranch_Duplicate(t *testing.T) {
	store := &BranchStore{Branches: []*BranchMetadata{{Name: "main"}}}
	if err := store.AddBranch(&BranchMetadata{Name: "main"}); err == nil {
		t.Error("expected duplicate error, got nil")
	}
}

func TestBranchStore_RemoveBranch(t *testing.T) {
	store := &BranchStore{
		Branches: []*BranchMetadata{{Name: "main"}, {Name: "feature"}},
	}
	if err := store.RemoveBranch("feature"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.Branches) != 1 || store.GetBranch("feature") != nil {
		t.Error("feature branch still present after removal")
	}
}

func TestBranchStore_RemoveBranch_NotFound(t *testing.T) {
	store := &BranchStore{Branches: []*BranchMetadata{{Name: "main"}}}
	if err := store.RemoveBranch("nonexistent"); err == nil {
		t.Error("expected error for missing branch, got nil")
	}
}

// ── MetadataManager ───────────────────────────────────────────────────────────

func TestMetadataManager_LoadDefault_NoFile(t *testing.T) {
	m := newMeta(t)
	store, err := m.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.Current != "main" {
		t.Errorf("current = %q, want main", store.Current)
	}
	if len(store.Branches) != 1 || store.Branches[0].Name != "main" {
		t.Errorf("branches = %v", store.Branches)
	}
	if !store.Branches[0].IsDefault {
		t.Error("main branch should be default")
	}
}

func TestMetadataManager_SaveAndLoad(t *testing.T) {
	m := newMeta(t)
	store := &BranchStore{
		Current: "feature",
		Branches: []*BranchMetadata{
			{Name: "main", Schema: "public", IsDefault: true, CreatedAt: time.Now()},
			{Name: "feature", Schema: "flash_branch_feature", CreatedAt: time.Now()},
		},
	}
	if err := m.Save(store); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	loaded, err := m.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded.Current != "feature" {
		t.Errorf("current = %q, want feature", loaded.Current)
	}
	if len(loaded.Branches) != 2 {
		t.Errorf("branches = %d, want 2", len(loaded.Branches))
	}
}

func TestMetadataManager_EnsureDirectories(t *testing.T) {
	dir := t.TempDir()
	m := NewMetadataManager(filepath.Join(dir, "deep", "migrations"))
	if err := m.EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "deep", "migrations", ".flash")); os.IsNotExist(err) {
		t.Error(".flash directory not created")
	}
}

// ── SchemaDiff ────────────────────────────────────────────────────────────────

func TestSchemaDiff_IsEmpty_True(t *testing.T) {
	if !(&SchemaDiff{}).IsEmpty() {
		t.Error("empty diff should be IsEmpty() = true")
	}
}

func TestSchemaDiff_IsEmpty_False(t *testing.T) {
	if (&SchemaDiff{TablesAdded: []string{"users"}}).IsEmpty() {
		t.Error("non-empty diff should be IsEmpty() = false")
	}
}

func TestSchemaDiff_String_Empty(t *testing.T) {
	if got := (&SchemaDiff{}).String(); got != "No differences found" {
		t.Errorf("String() = %q, want 'No differences found'", got)
	}
}

func TestSchemaDiff_String_WithChanges(t *testing.T) {
	d := &SchemaDiff{
		TablesAdded:   []string{"orders"},
		TablesRemoved: []string{"legacy"},
		TablesChanged: []TableDiff{{Name: "users", ColumnsAdded: []string{"phone"}}},
	}
	s := d.String()
	for _, want := range []string{"orders", "legacy", "users", "phone"} {
		if !strings.Contains(s, want) {
			t.Errorf("String() missing %q:\n%s", want, s)
		}
	}
}

// ── compareSchemas (Manager method) ──────────────────────────────────────────

func TestManager_compareSchemas_NewTable(t *testing.T) {
	m := &Manager{}
	diff := m.compareSchemas(
		nil,
		[]types.SchemaTable{{Name: "users"}},
	)
	if len(diff.TablesAdded) != 1 || diff.TablesAdded[0] != "users" {
		t.Errorf("TablesAdded = %v, want [users]", diff.TablesAdded)
	}
}

func TestManager_compareSchemas_RemovedTable(t *testing.T) {
	m := &Manager{}
	diff := m.compareSchemas(
		[]types.SchemaTable{{Name: "old"}},
		nil,
	)
	if len(diff.TablesRemoved) != 1 || diff.TablesRemoved[0] != "old" {
		t.Errorf("TablesRemoved = %v, want [old]", diff.TablesRemoved)
	}
}

func TestManager_compareSchemas_NoChanges(t *testing.T) {
	m := &Manager{}
	tables := []types.SchemaTable{{
		Name:    "users",
		Columns: []types.SchemaColumn{{Name: "id", Type: "SERIAL"}},
	}}
	if diff := m.compareSchemas(tables, tables); !diff.IsEmpty() {
		t.Errorf("expected empty diff, got %+v", diff)
	}
}

func TestManager_compareSchemas_ColumnAdded(t *testing.T) {
	m := &Manager{}
	base := []types.SchemaColumn{{Name: "id", Type: "SERIAL"}}
	diff := m.compareSchemas(
		[]types.SchemaTable{{Name: "users", Columns: base}},
		[]types.SchemaTable{{Name: "users", Columns: append(base, types.SchemaColumn{Name: "email", Type: "TEXT"})}},
	)
	if len(diff.TablesChanged) != 1 {
		t.Fatalf("TablesChanged = %d, want 1", len(diff.TablesChanged))
	}
	if len(diff.TablesChanged[0].ColumnsAdded) != 1 || diff.TablesChanged[0].ColumnsAdded[0] != "email" {
		t.Errorf("ColumnsAdded = %v, want [email]", diff.TablesChanged[0].ColumnsAdded)
	}
}
