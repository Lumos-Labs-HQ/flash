package schema

import (
	"os"
	"path/filepath"
	"strings"
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

// ── Date/Time type parsing ────────────────────────────────────────────────────

func TestParseAllDateTimeTypes_PostgreSQL(t *testing.T) {
	sm := newSM()
	sql := `CREATE TABLE events (
		id SERIAL PRIMARY KEY,
		ts_tz    TIMESTAMP WITH TIME ZONE NOT NULL,
		ts_notz  TIMESTAMP WITHOUT TIME ZONE,
		ts_plain TIMESTAMP,
		ts_tz2   TIMESTAMPTZ,
		d        DATE,
		t        TIME,
		t_tz     TIME WITH TIME ZONE,
		iv       INTERVAL,
		ts_def   TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]string{
		"ts_tz":    "TIMESTAMP WITH TIME ZONE",
		"ts_notz":  "TIMESTAMP WITHOUT TIME ZONE",
		"ts_plain": "TIMESTAMP",
		"ts_tz2":   "TIMESTAMPTZ",
		"d":        "DATE",
		"t":        "TIME",
		"t_tz":     "TIME WITH TIME ZONE",
		"iv":       "INTERVAL",
		"ts_def":   "TIMESTAMP WITH TIME ZONE",
	}

	cols := make(map[string]string)
	for _, c := range table.Columns {
		cols[c.Name] = c.Type
	}

	for name, expectedType := range want {
		got, ok := cols[name]
		if !ok {
			t.Errorf("column %q not found", name)
			continue
		}
		if got != expectedType {
			t.Errorf("col %q: type = %q, want %q", name, got, expectedType)
		}
	}

	// ts_tz should be NOT NULL
	for _, c := range table.Columns {
		if c.Name == "ts_tz" && c.Nullable {
			t.Error("ts_tz should be NOT NULL")
		}
		if c.Name == "ts_def" && c.Default == "" {
			t.Error("ts_def should have DEFAULT NOW()")
		}
	}
}

func TestParseAllDateTimeTypes_MySQL(t *testing.T) {
	sm := newSM()
	sql := `CREATE TABLE events (
		id INT PRIMARY KEY AUTO_INCREMENT,
		dt   DATETIME NOT NULL,
		dt6  DATETIME(6),
		ts   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		ts6  TIMESTAMP(6),
		d    DATE,
		t    TIME,
		yr   YEAR
	)`
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]string{
		"dt":  "DATETIME",
		"dt6": "DATETIME(6)",
		"ts":  "TIMESTAMP",
		"ts6": "TIMESTAMP(6)",
		"d":   "DATE",
		"t":   "TIME",
		"yr":  "YEAR",
	}

	cols := make(map[string]string)
	for _, c := range table.Columns {
		cols[c.Name] = c.Type
	}

	for name, expectedType := range want {
		got, ok := cols[name]
		if !ok {
			t.Errorf("column %q not found", name)
			continue
		}
		if got != expectedType {
			t.Errorf("col %q: type = %q, want %q", name, got, expectedType)
		}
	}
}

func TestParseAllDateTimeTypes_SQLite(t *testing.T) {
	sm := newSM()
	// SQLite stores dates as TEXT, REAL, or INTEGER — all are valid
	sql := `CREATE TABLE events (
		id   INTEGER PRIMARY KEY AUTOINCREMENT,
		ts   TEXT NOT NULL,
		dt   TEXT,
		d    TEXT DEFAULT (date('now')),
		unix INTEGER
	)`
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(table.Columns) != 5 {
		t.Errorf("columns = %d, want 5", len(table.Columns))
	}
}

// ── Numeric precision types ───────────────────────────────────────────────────

func TestParseNumericTypes(t *testing.T) {
	sm := newSM()
	sql := `CREATE TABLE financials (
		id       SERIAL PRIMARY KEY,
		price    DECIMAL(10,2) NOT NULL,
		qty      NUMERIC(8,3),
		amount   FLOAT,
		rate     DOUBLE PRECISION,
		score    REAL,
		big      BIGINT,
		small    SMALLINT,
		tiny     TINYINT,
		pct      NUMERIC(5,4) DEFAULT 0.0000
	)`
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]string{
		"price":  "DECIMAL(10,2)",
		"qty":    "NUMERIC(8,3)",
		"amount": "FLOAT",
		"rate":   "DOUBLE PRECISION",
		"score":  "REAL",
		"big":    "BIGINT",
		"small":  "SMALLINT",
		"tiny":   "TINYINT",
		"pct":    "NUMERIC(5,4)",
	}

	cols := make(map[string]string)
	for _, c := range table.Columns {
		cols[c.Name] = c.Type
	}
	for name, expectedType := range want {
		if got := cols[name]; got != expectedType {
			t.Errorf("col %q: type = %q, want %q", name, got, expectedType)
		}
	}
}

// ── CHECK constraint parsing ──────────────────────────────────────────────────

func TestParseCheckConstraints(t *testing.T) {
	sm := newSM()
	sql := `CREATE TABLE products (
		id    SERIAL PRIMARY KEY,
		price NUMERIC(10,2) NOT NULL CHECK (price >= 0),
		qty   INTEGER CHECK (qty > 0 AND qty <= 10000),
		code  VARCHAR(10) CHECK (code ~ '^[A-Z]{3}[0-9]{4}$')
	)`
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := make(map[string]string)
	for _, c := range table.Columns {
		checks[c.Name] = c.Check
	}

	if checks["price"] == "" {
		t.Error("price should have CHECK constraint")
	}
	if checks["qty"] == "" {
		t.Error("qty should have CHECK constraint")
	}
}

// ── Schema diff: nullable/default changes ─────────────────────────────────────

func TestSchemaDiff_NullableAndDefaultChange(t *testing.T) {
	sm := newSM()

	current := []types.SchemaTable{{
		Name: "users",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "SERIAL", IsPrimary: true},
			{Name: "email", Type: "TEXT", Nullable: true},
			{Name: "score", Type: "INTEGER", Nullable: false, Default: "0"},
		},
	}}

	target := []types.SchemaTable{{
		Name: "users",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "SERIAL", IsPrimary: true},
			{Name: "email", Type: "TEXT", Nullable: false},                    // made NOT NULL
			{Name: "score", Type: "INTEGER", Nullable: false, Default: "100"}, // default changed
		},
	}}

	diff := sm.compareSchemas(current, target, nil, nil, nil)

	if len(diff.ModifiedTables) != 1 {
		t.Fatalf("expected 1 modified table, got %d", len(diff.ModifiedTables))
	}
	mods := diff.ModifiedTables[0].ModifiedColumns
	if len(mods) != 2 {
		t.Fatalf("expected 2 modified columns, got %d: %v", len(mods), mods)
	}

	for _, col := range mods {
		switch col.Name {
		case "email":
			if !col.NullableChanged {
				t.Error("email NullableChanged should be true")
			}
		case "score":
			if !col.DefaultChanged {
				t.Error("score DefaultChanged should be true")
			}
		}
	}
}

// ── Schema diff: UNIQUE change ────────────────────────────────────────────────

func TestSchemaDiff_UniqueChange(t *testing.T) {
	sm := newSM()

	current := []types.SchemaTable{{
		Name: "users",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "SERIAL", IsPrimary: true},
			{Name: "email", Type: "TEXT", IsUnique: false},
		},
	}}
	target := []types.SchemaTable{{
		Name: "users",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "SERIAL", IsPrimary: true},
			{Name: "email", Type: "TEXT", IsUnique: true},
		},
	}}

	diff := sm.compareSchemas(current, target, nil, nil, nil)
	if len(diff.ModifiedTables) != 1 || len(diff.ModifiedTables[0].ModifiedColumns) != 1 {
		t.Fatal("expected 1 modified column (email uniqueness change)")
	}
}

// ── Schema diff: enum ADD VALUE ───────────────────────────────────────────────

func TestSchemaDiff_EnumAddValue(t *testing.T) {
	sm := newSM()
	current := []types.SchemaEnum{{Name: "status", Values: []string{"active", "inactive"}}}
	target := []types.SchemaEnum{{Name: "status", Values: []string{"active", "inactive", "pending"}}}

	diff := sm.compareSchemas(nil, nil, current, target, nil)
	if len(diff.ModifiedEnums) != 1 {
		t.Fatalf("expected 1 modified enum, got %d", len(diff.ModifiedEnums))
	}
	if len(diff.ModifiedEnums[0].AddValues) != 1 || diff.ModifiedEnums[0].AddValues[0] != "pending" {
		t.Errorf("expected pending to be added, got %v", diff.ModifiedEnums[0].AddValues)
	}
}

// ── Multi-word types preserved through parse/compare cycle ───────────────────

func TestMultiWordTypesRoundTrip(t *testing.T) {
	sm := newSM()
	sql := `CREATE TABLE t (
		a TIMESTAMP WITH TIME ZONE NOT NULL,
		b TIMESTAMP WITHOUT TIME ZONE,
		c DOUBLE PRECISION,
		d CHARACTER VARYING(100),
		e TIME WITH TIME ZONE
	)`
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// compareSchemas with identical target — no diffs
	tables := []types.SchemaTable{table}
	diff := sm.compareSchemas(tables, tables, nil, nil, nil)
	if len(diff.ModifiedTables) != 0 {
		t.Errorf("identical schema should produce no diffs, got %v", diff.ModifiedTables)
	}
}

// ── GENERATED column ──────────────────────────────────────────────────────────

func TestParseGeneratedColumn(t *testing.T) {
	sm := newSM()
	sql := `CREATE TABLE products (
		id    SERIAL PRIMARY KEY,
		price NUMERIC(10,2) NOT NULL,
		tax   NUMERIC(10,2) GENERATED ALWAYS AS (price * 0.1) STORED
	)`
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range table.Columns {
		if c.Name == "tax" && c.Generated != "" {
			found = true
		}
	}
	if !found {
		t.Error("tax column should have Generated expression")
	}
}

// ── ON UPDATE FK action ───────────────────────────────────────────────────────

func TestParseOnUpdateFK(t *testing.T) {
	sm := newSM()
	sql := `CREATE TABLE orders (
		id      SERIAL PRIMARY KEY,
		user_id INTEGER REFERENCES users(id) ON DELETE CASCADE ON UPDATE RESTRICT
	)`
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range table.Columns {
		if c.Name == "user_id" {
			if c.OnDeleteAction != "CASCADE" {
				t.Errorf("OnDeleteAction = %q, want CASCADE", c.OnDeleteAction)
			}
			if c.OnUpdateAction != "RESTRICT" {
				t.Errorf("OnUpdateAction = %q, want RESTRICT", c.OnUpdateAction)
			}
		}
	}
}

// ── Date/Time type parsing ────────────────────────────────────────────────────


func TestParseUsersTableFromExampleSchema(t *testing.T) {
	sql := `
CREATE TABLE IF NOT EXISTS users (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    address     VARCHAR(255),
    isadmin     BOOLEAN NOT NULL DEFAULT FALSE,
    age         INT CHECK (age >= 0),
    age_range   INT4RANGE GENERATED ALWAYS AS (
                    CASE WHEN age IS NULL THEN NULL
                         WHEN age < 18  THEN '[0,18)'::int4range
                         WHEN age < 35  THEN '[18,35)'::int4range
                         WHEN age < 55  THEN '[35,55)'::int4range
                         ELSE                '[55,)'::int4range
                    END
                ) STORED,
    bio         VARCHAR(500),
    email       VARCHAR(255) UNIQUE NOT NULL,
    preferences JSONB DEFAULT '{"theme":"light","notifications":true}',
    role        VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);`
	sm := NewSchemaManager(nil)
	table, err := sm.parseCreateTableStatement(sql)
	if err != nil {
		t.Fatalf("parseCreateTableStatement failed: %v", err)
	}
	if table.Name != "users" {
		t.Errorf("expected table name 'users', got %q", table.Name)
	}

	expectedCols := map[string]string{
		"id":          "SERIAL",
		"name":        "VARCHAR(255)",
		"address":     "VARCHAR(255)",
		"isadmin":     "BOOLEAN",
		"age":         "INT",
		"age_range":   "INT4RANGE",
		"bio":         "VARCHAR(500)",
		"email":       "VARCHAR(255)",
		"preferences": "JSONB",
		"role":        "VARCHAR(50)",
		"created_at":  "TIMESTAMP WITH TIME ZONE",
	}
	if len(table.Columns) != len(expectedCols) {
		t.Errorf("expected %d columns, got %d", len(expectedCols), len(table.Columns))
	}
	for _, col := range table.Columns {
		expectedType, exists := expectedCols[col.Name]
		if !exists {
			t.Errorf("unexpected column %q (type %q)", col.Name, col.Type)
		} else if col.Type != expectedType {
			t.Errorf("column %q: expected type %q, got %q", col.Name, expectedType, col.Type)
		}
	}
	for _, col := range table.Columns {
		if col.Name == "age_range" {
			if col.Generated == "" {
				t.Error("age_range should have Generated expression")
			}
			if !strings.Contains(col.Generated, "CASE WHEN") {
				t.Errorf("age_range Generated should contain CASE, got: %s", col.Generated)
			}
		}
	}
}
