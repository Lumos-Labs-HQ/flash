package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func newSM() *SchemaManager {
	// Parse methods don't call the adapter — safe to pass nil.
	return NewSchemaManager(nil)
}

// ── parseCreateTableStatement ────────────────────────────────────────────────

func TestParseCreateTable_Basic(t *testing.T) {
	sm := newSM()
	sql := `CREATE TABLE users (
		id SERIAL PRIMARY KEY,
		email VARCHAR(255) NOT NULL,
		name TEXT,
		age INTEGER DEFAULT 0
	)`
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if table.Name != "users" {
		t.Errorf("name = %q, want users", table.Name)
	}
	if len(table.Columns) != 4 {
		t.Fatalf("columns = %d, want 4", len(table.Columns))
	}
	if !table.Columns[0].IsPrimary {
		t.Error("id should be primary key")
	}
	if table.Columns[1].Nullable {
		t.Error("email should be NOT NULL")
	}
	if !table.Columns[2].Nullable {
		t.Error("name should be nullable")
	}
	if table.Columns[3].Default != "0" {
		t.Errorf("age default = %q, want 0", table.Columns[3].Default)
	}
}

func TestParseCreateTable_ForeignKey(t *testing.T) {
	sm := newSM()
	sql := `CREATE TABLE posts (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title TEXT NOT NULL
	)`
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	uid := table.Columns[1]
	if uid.ForeignKeyTable != "users" {
		t.Errorf("FK table = %q, want users", uid.ForeignKeyTable)
	}
	if uid.ForeignKeyColumn != "id" {
		t.Errorf("FK column = %q, want id", uid.ForeignKeyColumn)
	}
	if uid.OnDeleteAction != "CASCADE" {
		t.Errorf("ON DELETE = %q, want CASCADE", uid.OnDeleteAction)
	}
}

func TestParseCreateTable_IfNotExists(t *testing.T) {
	sm := newSM()
	sql := `CREATE TABLE IF NOT EXISTS products (id SERIAL PRIMARY KEY, name TEXT NOT NULL)`
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if table.Name != "products" {
		t.Errorf("name = %q, want products", table.Name)
	}
}

func TestParseCreateTable_QuotedName(t *testing.T) {
	sm := newSM()
	sql := `CREATE TABLE "order_items" (id SERIAL PRIMARY KEY, qty INTEGER NOT NULL)`
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if table.Name != "order_items" {
		t.Errorf("name = %q, want order_items", table.Name)
	}
}

func TestParseCreateTable_TableConstraintFK(t *testing.T) {
	sm := newSM()
	sql := `CREATE TABLE comments (
		id SERIAL PRIMARY KEY,
		post_id INTEGER NOT NULL,
		body TEXT NOT NULL,
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
	)`
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	postID := table.Columns[1]
	if postID.ForeignKeyTable != "posts" {
		t.Errorf("FK table = %q, want posts", postID.ForeignKeyTable)
	}
}

// ── parseCreateIndexStatement ────────────────────────────────────────────────

func TestParseCreateIndex_Unique(t *testing.T) {
	sm := newSM()
	sql := `CREATE UNIQUE INDEX idx_users_email ON users (email)`
	idx, err := sm.parseCreateIndexStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !idx.Unique {
		t.Error("index should be unique")
	}
	if idx.Name != "idx_users_email" {
		t.Errorf("name = %q, want idx_users_email", idx.Name)
	}
	if idx.Table != "users" {
		t.Errorf("table = %q, want users", idx.Table)
	}
	if len(idx.Columns) != 1 || idx.Columns[0] != "email" {
		t.Errorf("columns = %v, want [email]", idx.Columns)
	}
}

func TestParseCreateIndex_Composite(t *testing.T) {
	sm := newSM()
	sql := `CREATE INDEX idx_posts_user_created ON posts (user_id, created_at)`
	idx, err := sm.parseCreateIndexStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.Unique {
		t.Error("index should not be unique")
	}
	if len(idx.Columns) != 2 {
		t.Fatalf("columns = %d, want 2", len(idx.Columns))
	}
}

// ── parseCreateTypeStatement ─────────────────────────────────────────────────

func TestParseCreateType_Enum(t *testing.T) {
	sm := newSM()
	sql := `CREATE TYPE status AS ENUM ('active', 'inactive', 'pending')`
	enum, err := sm.parseCreateTypeStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enum.Name != "status" {
		t.Errorf("name = %q, want status", enum.Name)
	}
	if len(enum.Values) != 3 {
		t.Fatalf("values = %d, want 3", len(enum.Values))
	}
}

// ── ParseSchemaDir ───────────────────────────────────────────────────────────

func TestParseSchemaDir_MultiFile(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "users.sql"), []byte(`
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL
		);
	`), 0644)

	os.WriteFile(filepath.Join(dir, "posts.sql"), []byte(`
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL
		);
		CREATE INDEX idx_posts_user ON posts (user_id);
	`), 0644)

	sm := newSM()
	tables, enums, indexes, err := sm.ParseSchemaDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 2 {
		t.Errorf("tables = %d, want 2", len(tables))
	}
	if len(enums) != 0 {
		t.Errorf("enums = %d, want 0", len(enums))
	}
	if len(indexes) != 1 {
		t.Errorf("standalone indexes = %d, want 1", len(indexes))
	}
	// users must come before posts (FK dependency ordering)
	if tables[0].Name != "users" {
		t.Errorf("first table = %q, want users (FK ordering)", tables[0].Name)
	}
}

func TestParseSchemaDir_CircularFKError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "schema.sql"), []byte(`
		CREATE TABLE a (id SERIAL PRIMARY KEY, b_id INTEGER REFERENCES b(id));
		CREATE TABLE b (id SERIAL PRIMARY KEY, a_id INTEGER REFERENCES a(id));
	`), 0644)

	sm := newSM()
	_, _, _, err := sm.ParseSchemaDir(dir)
	if err == nil {
		t.Error("expected circular FK error, got nil")
	}
}

func TestParseSchemaDir_MissingReferencedTable(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "schema.sql"), []byte(`
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES nonexistent(id)
		);
	`), 0644)

	sm := newSM()
	_, _, _, err := sm.ParseSchemaDir(dir)
	if err == nil {
		t.Error("expected missing-table error, got nil")
	}
}

func TestParseSchemaDir_EnumAndTable(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "schema.sql"), []byte(`
		CREATE TYPE role AS ENUM ('admin', 'user', 'guest');
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			role role NOT NULL
		);
	`), 0644)

	sm := newSM()
	tables, enums, _, err := sm.ParseSchemaDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 1 {
		t.Errorf("tables = %d, want 1", len(tables))
	}
	if len(enums) != 1 || enums[0].Name != "role" {
		t.Errorf("enums = %v, want [{role [admin user guest]}]", enums)
	}
}

// ── compareSchemas ───────────────────────────────────────────────────────────

func TestCompareSchemas_NewTable(t *testing.T) {
	sm := newSM()
	diff := sm.compareSchemas(
		nil,
		[]types.SchemaTable{{Name: "users", Columns: []types.SchemaColumn{{Name: "id", Type: "SERIAL", IsPrimary: true}}}},
		nil, nil, nil,
	)
	if len(diff.NewTables) != 1 {
		t.Errorf("NewTables = %d, want 1", len(diff.NewTables))
	}
	if len(diff.DroppedTables) != 0 {
		t.Errorf("DroppedTables = %d, want 0", len(diff.DroppedTables))
	}
}

func TestCompareSchemas_DroppedTable(t *testing.T) {
	sm := newSM()
	diff := sm.compareSchemas(
		[]types.SchemaTable{{Name: "old_table", Columns: []types.SchemaColumn{{Name: "id", Type: "SERIAL"}}}},
		nil, nil, nil, nil,
	)
	if len(diff.DroppedTables) != 1 || diff.DroppedTables[0] != "old_table" {
		t.Errorf("DroppedTables = %v, want [old_table]", diff.DroppedTables)
	}
}

func TestCompareSchemas_AddColumn(t *testing.T) {
	sm := newSM()
	base := []types.SchemaColumn{{Name: "id", Type: "SERIAL", IsPrimary: true}}
	diff := sm.compareSchemas(
		[]types.SchemaTable{{Name: "users", Columns: base}},
		[]types.SchemaTable{{Name: "users", Columns: append(base, types.SchemaColumn{Name: "email", Type: "TEXT"})}},
		nil, nil, nil,
	)
	if len(diff.ModifiedTables) != 1 {
		t.Fatalf("ModifiedTables = %d, want 1", len(diff.ModifiedTables))
	}
	if len(diff.ModifiedTables[0].NewColumns) != 1 || diff.ModifiedTables[0].NewColumns[0].Name != "email" {
		t.Errorf("NewColumns = %v", diff.ModifiedTables[0].NewColumns)
	}
}

func TestCompareSchemas_DropColumn(t *testing.T) {
	sm := newSM()
	diff := sm.compareSchemas(
		[]types.SchemaTable{{Name: "users", Columns: []types.SchemaColumn{
			{Name: "id", Type: "SERIAL", IsPrimary: true},
			{Name: "phone", Type: "TEXT", Nullable: true},
		}}},
		[]types.SchemaTable{{Name: "users", Columns: []types.SchemaColumn{
			{Name: "id", Type: "SERIAL", IsPrimary: true},
		}}},
		nil, nil, nil,
	)
	if len(diff.ModifiedTables) != 1 {
		t.Fatalf("ModifiedTables = %d, want 1", len(diff.ModifiedTables))
	}
	dropped := diff.ModifiedTables[0].DroppedColumns
	if len(dropped) != 1 || dropped[0].Name != "phone" {
		t.Errorf("DroppedColumns = %v, want [phone]", dropped)
	}
}

func TestCompareSchemas_NewEnum(t *testing.T) {
	sm := newSM()
	diff := sm.compareSchemas(nil, nil,
		nil,
		[]types.SchemaEnum{{Name: "status", Values: []string{"active", "inactive"}}},
		nil,
	)
	if len(diff.NewEnums) != 1 || diff.NewEnums[0].Name != "status" {
		t.Errorf("NewEnums = %v, want [{status ...}]", diff.NewEnums)
	}
}

func TestCompareSchemas_DroppedEnum(t *testing.T) {
	sm := newSM()
	diff := sm.compareSchemas(nil, nil,
		[]types.SchemaEnum{{Name: "old_status", Values: []string{"a"}}},
		nil,
		nil,
	)
	if len(diff.DroppedEnums) != 1 || diff.DroppedEnums[0] != "old_status" {
		t.Errorf("DroppedEnums = %v, want [old_status]", diff.DroppedEnums)
	}
}

func TestCompareSchemas_NewIndex_StandaloneInjected(t *testing.T) {
	sm := newSM()
	standaloneIdx := types.SchemaIndex{Name: "idx_users_email", Table: "users", Columns: []string{"email"}, Unique: true}
	diff := sm.compareSchemas(
		[]types.SchemaTable{{Name: "users", Columns: []types.SchemaColumn{{Name: "id", Type: "SERIAL", IsPrimary: true}}}},
		[]types.SchemaTable{{Name: "users", Columns: []types.SchemaColumn{{Name: "id", Type: "SERIAL", IsPrimary: true}}}},
		nil, nil,
		[]types.SchemaIndex{standaloneIdx},
	)
	if len(diff.NewIndexes) != 1 || diff.NewIndexes[0].Name != "idx_users_email" {
		t.Errorf("NewIndexes = %v, want [idx_users_email]", diff.NewIndexes)
	}
}

func TestCompareSchemas_NoChanges(t *testing.T) {
	sm := newSM()
	tables := []types.SchemaTable{{Name: "users", Columns: []types.SchemaColumn{
		{Name: "id", Type: "SERIAL", IsPrimary: true},
		{Name: "email", Type: "TEXT", Nullable: false},
	}}}
	diff := sm.compareSchemas(tables, tables, nil, nil, nil)
	if len(diff.NewTables) != 0 || len(diff.DroppedTables) != 0 || len(diff.ModifiedTables) != 0 {
		t.Errorf("expected empty diff, got %+v", diff)
	}
}

// ── splitColumnDefinitions ───────────────────────────────────────────────────

func TestSplitColumnDefinitions_NestedParens(t *testing.T) {
	sm := newSM()
	// DECIMAL(10, 2) must not be split at the inner comma
	input := `id SERIAL PRIMARY KEY, price DECIMAL(10, 2) NOT NULL, name TEXT`
	parts := sm.splitColumnDefinitions(input)
	if len(parts) != 3 {
		t.Errorf("parts = %d, want 3: %v", len(parts), parts)
	}
}

func TestParseCreateIndex_Partial(t *testing.T) {
	sm := newSM()
	sql := `CREATE INDEX idx_orgs_slug ON organizations (slug) WHERE deleted_at IS NULL`
	idx, err := sm.parseCreateIndexStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.Where != "deleted_at IS NULL" {
		t.Errorf("where = %q, want 'deleted_at IS NULL'", idx.Where)
	}
}

func TestParseColumnDefinition_Check(t *testing.T) {
	sm := newSM()
	sql := `CREATE TABLE ai_conversations (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		type TEXT NOT NULL CHECK (type IN ('error_assist', 'spec_gen'))
	);`
	tables, _, _, err := sm.parseSchemaContentWithIndexes(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	var typeCol *types.SchemaColumn
	for i := range tables[0].Columns {
		if tables[0].Columns[i].Name == "type" {
			typeCol = &tables[0].Columns[i]
			break
		}
	}
	if typeCol == nil {
		t.Fatal("type column not found")
	}
	if typeCol.Check != "type IN ('error_assist', 'spec_gen')" {
		t.Errorf("check = %q, want 'type IN ('error_assist', 'spec_gen')'", typeCol.Check)
	}
}

func TestCompareIndexes_SkipsNewTableIndexes(t *testing.T) {
	sm := newSM()
	current := []types.SchemaTable{
		{Name: "users", Columns: []types.SchemaColumn{{Name: "id", Type: "INT", IsPrimary: true}}},
	}
	target := []types.SchemaTable{
		{
			Name: "posts",
			Columns: []types.SchemaColumn{
				{Name: "id", Type: "INT", IsPrimary: true},
				{Name: "user_id", Type: "INT"},
			},
			Indexes: []types.SchemaIndex{
				{Name: "idx_posts_user", Table: "posts", Columns: []string{"user_id"}},
			},
		},
	}

	diff := sm.compareSchemas(current, target, nil, nil, nil)

	if len(diff.NewTables) != 1 {
		t.Fatalf("expected 1 new table, got %d", len(diff.NewTables))
	}
	if len(diff.NewIndexes) != 0 {
		t.Errorf("expected 0 new indexes (index belongs to new table), got %d: %v", len(diff.NewIndexes), diff.NewIndexes)
	}
}

func TestCompareSchemas_PreservesDependencyOrder(t *testing.T) {
	sm := newSM()
	current := []types.SchemaTable{}
	// In real usage target comes from ParseSchemaPath which sorts by dependencies.
	// We simulate that pre-sorted order here: users (no deps) before jobs (FK to users).
	target := []types.SchemaTable{
		{
			Name: "users",
			Columns: []types.SchemaColumn{
				{Name: "id", Type: "INT", IsPrimary: true},
			},
		},
		{
			Name: "jobs",
			Columns: []types.SchemaColumn{
				{Name: "id", Type: "INT", IsPrimary: true},
				{Name: "user_id", Type: "INT", ForeignKeyTable: "users", ForeignKeyColumn: "id"},
			},
		},
	}

	diff := sm.compareSchemas(current, target, nil, nil, nil)

	if len(diff.NewTables) != 2 {
		t.Fatalf("expected 2 new tables, got %d", len(diff.NewTables))
	}
	// users should come before jobs because jobs has a FK to users
	if diff.NewTables[0].Name != "users" {
		t.Errorf("expected first table to be 'users' (no deps), got %q", diff.NewTables[0].Name)
	}
	if diff.NewTables[1].Name != "jobs" {
		t.Errorf("expected second table to be 'jobs' (depends on users), got %q", diff.NewTables[1].Name)
	}
}
