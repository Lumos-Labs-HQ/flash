package migrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/Lumos-Labs-HQ/flash/internal/database/mysql"
	"github.com/Lumos-Labs-HQ/flash/internal/database/postgres"
	"github.com/Lumos-Labs-HQ/flash/internal/database/scylla"
	"github.com/Lumos-Labs-HQ/flash/internal/database/sqlite"
	"github.com/Lumos-Labs-HQ/flash/internal/types"
	"github.com/Lumos-Labs-HQ/flash/internal/utils"
)

// ── shared fixture helpers ────────────────────────────────────────────────────

func migratorFor(provider string) *Migrator {
	var adapter database.DatabaseAdapter
	switch provider {
	case "mysql":
		adapter = mysql.New()
	case "postgresql":
		adapter = postgres.New()
	case "scylla":
		adapter = scylla.New()
	default:
		adapter = sqlite.New()
	}
	return &Migrator{
		adapter:       adapter,
		provider:      provider,
		fileUtils:     &utils.FileUtils{},
		inputUtils:    &utils.InputUtils{},
		conflictUtils: &utils.ConflictUtils{},
	}
}

// generateFor renders the migration file for diff and returns the whole file
// plus whether it contained executable SQL.
func generateFor(m *Migrator, diff *types.SchemaDiff) (string, bool) {
	return m.generateSQLFromDiff(diff, "edge")
}

// upSection / downSection split a formatted migration file into its halves.
func upSection(t *testing.T, file string) string {
	t.Helper()
	idx := strings.Index(file, "-- +migrate Up")
	if idx < 0 {
		t.Fatalf("migration file missing '-- +migrate Up' marker:\n%s", file)
	}
	end := strings.Index(file, "-- +migrate Down")
	if end < 0 {
		t.Fatalf("migration file missing '-- +migrate Down' marker:\n%s", file)
	}
	return file[idx:end]
}

func downSection(t *testing.T, file string) string {
	t.Helper()
	idx := strings.Index(file, "-- +migrate Down")
	if idx < 0 {
		t.Fatalf("migration file missing '-- +migrate Down' marker:\n%s", file)
	}
	return file[idx:]
}

// orderOf returns the index of each fragment inside s, failing when missing.
func orderOf(t *testing.T, s string, fragments ...string) []int {
	t.Helper()
	idxs := make([]int, len(fragments))
	for i, f := range fragments {
		idx := strings.Index(s, f)
		if idx < 0 {
			t.Fatalf("fragment %q not found in:\n%s", f, s)
		}
		idxs[i] = idx
	}
	return idxs
}

func assertAscending(t *testing.T, name string, idxs []int) {
	t.Helper()
	for i := 1; i < len(idxs); i++ {
		if idxs[i-1] >= idxs[i] {
			t.Errorf("%s: fragments out of expected order (positions %v)", name, idxs)
		}
	}
}

// ── enum edge cases ───────────────────────────────────────────────────────────

func TestGenerateSQLFromDiff_PostgresNewEnum_GuardsAndEscapes(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		NewEnums: []types.SchemaEnum{
			{Name: "status", Values: []string{"active", "it's on"}},
		},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("new enum must produce executable SQL")
	}
	if !contains(file, "CREATE TYPE \"status\" AS ENUM ('active', 'it''s on')") {
		t.Errorf("enum values must be single-quote escaped:\n%s", file)
	}
	if !contains(file, "IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'status')") {
		t.Errorf("Postgres enum creation must be guarded against re-creation:\n%s", file)
	}
	// DOWN must drop the enum.
	if !contains(downSection(t, file), "DROP TYPE IF EXISTS \"status\";") {
		t.Errorf("enum down must drop the type:\n%s", file)
	}
}

func TestGenerateSQLFromDiff_EnumNameQuoteEscaping(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		NewEnums: []types.SchemaEnum{{Name: `we"ird`, Values: []string{"a"}}},
	}
	file, _ := generateFor(m, diff)
	// A double quote inside the enum name must be doubled for the identifier.
	if !contains(file, `CREATE TYPE "we""ird" AS ENUM ('a')`) {
		t.Errorf("double quote in enum name must be escaped by doubling:\n%s", file)
	}
}

func TestGenerateSQLFromDiff_SQLEnumSkippedOnSQLiteMySQL(t *testing.T) {
	diff := &types.SchemaDiff{
		NewEnums:     []types.SchemaEnum{{Name: "status", Values: []string{"a"}}},
		DroppedEnums: []string{"gone"},
	}
	for _, provider := range []string{"sqlite", "mysql"} {
		m := migratorFor(provider)
		file, hasSQL := generateFor(m, diff)
		if hasSQL {
			t.Errorf("%s: standalone enums must not generate SQL (handled inline/skipped)", provider)
		}
		if contains(file, "CREATE TYPE") || contains(file, "DROP TYPE") {
			t.Errorf("%s: enum SQL leaked:\n%s", provider, file)
		}
	}
}

func TestGenerateSQLFromDiff_PostgresModifiedEnum(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		ModifiedEnums: []types.EnumDiff{{Name: "status", AddValues: []string{"archived", "it's"}}},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("enum value addition must produce SQL")
	}
	if !contains(file, `ALTER TYPE "status" ADD VALUE IF NOT EXISTS 'archived';`) {
		t.Errorf("missing ADD VALUE for 'archived':\n%s", file)
	}
	if !contains(file, `ALTER TYPE "status" ADD VALUE IF NOT EXISTS 'it''s';`) {
		t.Errorf("added enum value must be quote-escaped:\n%s", file)
	}
	// Down cannot remove values (Postgres limitation) — must say so as a comment, not SQL.
	if !contains(downSection(t, file), "-- Cannot remove enum value") {
		t.Errorf("down section must document the irreversible enum change:\n%s", file)
	}
}

func TestGenerateSQLFromDiff_ModifiedEnumSkippedOnSQLite(t *testing.T) {
	m := migratorFor("sqlite")
	diff := &types.SchemaDiff{ModifiedEnums: []types.EnumDiff{{Name: "status", AddValues: []string{"x"}}}}
	if _, hasSQL := generateFor(m, diff); hasSQL {
		t.Error("SQLite has no enums — modified enum must be a no-op")
	}
}

// ── rename edge cases ─────────────────────────────────────────────────────────

func TestGenerateSQLFromDiff_RenamedTable_AllProviders(t *testing.T) {
	diff := &types.SchemaDiff{
		RenamedTables: []types.RenameOp{{OldName: "users", NewName: "members"}},
	}
	cases := []struct {
		provider string
		up       string
		down     string
	}{
		{"postgresql", `ALTER TABLE "users" RENAME TO "members";`, `ALTER TABLE "members" RENAME TO "users";`},
		{"sqlite", `ALTER TABLE "users" RENAME TO "members";`, `ALTER TABLE "members" RENAME TO "users";`},
		{"mysql", "RENAME TABLE `users` TO `members`;", "RENAME TABLE `members` TO `users`;"},
	}
	for _, c := range cases {
		m := migratorFor(c.provider)
		file, hasSQL := generateFor(m, diff)
		if !hasSQL {
			t.Fatalf("%s: rename must produce executable SQL", c.provider)
		}
		if !contains(upSection(t, file), c.up) {
			t.Errorf("%s: missing up %q:\n%s", c.provider, c.up, file)
		}
		if !contains(downSection(t, file), c.down) {
			t.Errorf("%s: missing down %q:\n%s", c.provider, c.down, file)
		}
	}
}

func TestGenerateSQLFromDiff_RenamedColumn_PostgresAndSQLite(t *testing.T) {
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name:           "users",
			RenamedColumns: []types.RenameOp{{Table: "users", OldName: "fullname", NewName: "display_name"}},
			NewTable: types.SchemaTable{Columns: []types.SchemaColumn{
				{Name: "display_name", Type: "TEXT"},
			}},
		}},
	}
	for _, provider := range []string{"postgresql", "sqlite"} {
		m := migratorFor(provider)
		file, hasSQL := generateFor(m, diff)
		if !hasSQL {
			t.Fatalf("%s: column rename must produce SQL", provider)
		}
		wantUp := `ALTER TABLE "users" RENAME COLUMN "fullname" TO "display_name";`
		wantDown := `ALTER TABLE "users" RENAME COLUMN "display_name" TO "fullname";`
		if !contains(upSection(t, file), wantUp) {
			t.Errorf("%s: missing up %q:\n%s", provider, wantUp, file)
		}
		if !contains(downSection(t, file), wantDown) {
			t.Errorf("%s: missing down %q:\n%s", provider, wantDown, file)
		}
	}
}

func TestGenerateSQLFromDiff_RenamedColumn_MySQLIncludesType(t *testing.T) {
	m := migratorFor("mysql")
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name:           "users",
			RenamedColumns: []types.RenameOp{{Table: "users", OldName: "fullname", NewName: "display_name"}},
			NewTable: types.SchemaTable{Columns: []types.SchemaColumn{
				{Name: "display_name", Type: "VARCHAR(120)"},
			}},
		}},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("MySQL rename must produce SQL")
	}
	if !contains(file, "ALTER TABLE `users` CHANGE COLUMN `fullname` `display_name` VARCHAR(120);") {
		t.Errorf("MySQL CHANGE COLUMN must carry the new column type:\n%s", file)
	}
}

func TestGenerateSQLFromDiff_RenamedColumn_MySQLMissingTypeFallsBackToText(t *testing.T) {
	m := migratorFor("mysql")
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name:           "users",
			RenamedColumns: []types.RenameOp{{Table: "users", OldName: "a", NewName: "b"}},
			// NewTable has no matching column -> colType fallback "TEXT"
			NewTable: types.SchemaTable{Columns: []types.SchemaColumn{{Name: "unrelated", Type: "INT"}}},
		}},
	}
	file, _ := generateFor(m, diff)
	if !contains(file, "CHANGE COLUMN `a` `b` TEXT;") {
		t.Errorf("rename without type info must fall back to TEXT:\n%s", file)
	}
}

// ── column modification edge cases ────────────────────────────────────────────

func TestGenerateSQLFromDiff_NullableChange_Postgres(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name: "users",
			ModifiedColumns: []types.ColumnDiff{
				{
					Name: "email", OldType: "TEXT", NewType: "TEXT",
					OldColumn:       types.SchemaColumn{Name: "email", Type: "TEXT", Nullable: true},
					NewColumn:       types.SchemaColumn{Name: "email", Type: "TEXT", Nullable: false},
					NullableChanged: true,
				},
			},
		}},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("nullable change must produce SQL")
	}
	up := upSection(t, file)
	down := downSection(t, file)
	// Tighten: NOT NULL added in up, removed in down.
	if !contains(up, `ALTER TABLE "users" ALTER COLUMN "email" SET NOT NULL;`) {
		t.Errorf("up must SET NOT NULL:\n%s", up)
	}
	if !contains(down, `ALTER TABLE "users" ALTER COLUMN "email" DROP NOT NULL;`) {
		t.Errorf("down must DROP NOT NULL:\n%s", down)
	}
}

func TestGenerateSQLFromDiff_NullableLoosened_Postgres(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name: "users",
			ModifiedColumns: []types.ColumnDiff{
				{
					Name: "age", OldType: "INT", NewType: "INT",
					OldColumn:       types.SchemaColumn{Name: "age", Type: "INT", Nullable: false},
					NewColumn:       types.SchemaColumn{Name: "age", Type: "INT", Nullable: true},
					NullableChanged: true,
				},
			},
		}},
	}
	file, _ := generateFor(m, diff)
	if !contains(upSection(t, file), `ALTER COLUMN "age" DROP NOT NULL;`) {
		t.Errorf("up must DROP NOT NULL when the column becomes nullable:\n%s", file)
	}
	if !contains(downSection(t, file), `ALTER COLUMN "age" SET NOT NULL;`) {
		t.Errorf("down must SET NOT NULL when reverting:\n%s", file)
	}
}

func TestGenerateSQLFromDiff_DefaultChange_Postgres(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name: "posts",
			ModifiedColumns: []types.ColumnDiff{
				{
					Name: "status", OldType: "TEXT", NewType: "TEXT",
					OldColumn:      types.SchemaColumn{Name: "status", Type: "TEXT", Default: ""},
					NewColumn:      types.SchemaColumn{Name: "status", Type: "TEXT", Default: "'draft'"},
					DefaultChanged: true,
				},
			},
		}},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("default change must produce SQL")
	}
	if !contains(upSection(t, file), `ALTER COLUMN "status" SET DEFAULT 'draft';`) {
		t.Errorf("up must SET DEFAULT:\n%s", file)
	}
	if !contains(downSection(t, file), `ALTER COLUMN "status" DROP DEFAULT;`) {
		t.Errorf("down must DROP DEFAULT when there was none before:\n%s", file)
	}
}

func TestGenerateSQLFromDiff_DefaultRemoved_Postgres(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name: "posts",
			ModifiedColumns: []types.ColumnDiff{
				{
					Name: "views", OldType: "INT", NewType: "INT",
					OldColumn:      types.SchemaColumn{Name: "views", Type: "INT", Default: "0"},
					NewColumn:      types.SchemaColumn{Name: "views", Type: "INT", Default: ""},
					DefaultChanged: true,
				},
			},
		}},
	}
	file, _ := generateFor(m, diff)
	if !contains(upSection(t, file), `ALTER COLUMN "views" DROP DEFAULT;`) {
		t.Errorf("up must DROP DEFAULT:\n%s", file)
	}
	if !contains(downSection(t, file), `ALTER COLUMN "views" SET DEFAULT 0;`) {
		t.Errorf("down must restore the old default:\n%s", file)
	}
}

func TestGenerateSQLFromDiff_CheckConstraintChange_Postgres(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name: "users",
			ModifiedColumns: []types.ColumnDiff{
				{
					Name: "age", OldType: "INT", NewType: "INT",
					OldColumn: types.SchemaColumn{Name: "age", Type: "INT", Check: "age >= 0"},
					NewColumn: types.SchemaColumn{Name: "age", Type: "INT", Check: "age >= 18"},
				},
			},
		}},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("check change must produce SQL")
	}
	if !contains(file, `DROP CONSTRAINT IF EXISTS "users_age_check";`) {
		t.Errorf("old CHECK must be dropped first:\n%s", file)
	}
	if !contains(file, `ADD CONSTRAINT "users_age_check" CHECK (age >= 18);`) {
		t.Errorf("new CHECK must be added:\n%s", file)
	}
}

func TestGenerateSQLFromDiff_UniqueAdded_Postgres(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name: "users",
			ModifiedColumns: []types.ColumnDiff{
				{
					Name: "email", OldType: "TEXT", NewType: "TEXT",
					OldColumn: types.SchemaColumn{Name: "email", Type: "TEXT"},
					NewColumn: types.SchemaColumn{Name: "email", Type: "TEXT", IsUnique: true},
				},
			},
		}},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("unique addition must produce SQL")
	}
	if !contains(upSection(t, file), `ADD CONSTRAINT "users_email_key" UNIQUE ("email");`) {
		t.Errorf("up must add unique constraint:\n%s", file)
	}
	if !contains(downSection(t, file), `DROP CONSTRAINT IF EXISTS "users_email_key";`) {
		t.Errorf("down must drop unique constraint:\n%s", file)
	}
}

func TestGenerateSQLFromDiff_UniqueRemoved_MySQL(t *testing.T) {
	m := migratorFor("mysql")
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name: "users",
			ModifiedColumns: []types.ColumnDiff{
				{
					Name: "email", OldType: "VARCHAR(255)", NewType: "VARCHAR(255)",
					OldColumn: types.SchemaColumn{Name: "email", Type: "VARCHAR(255)", IsUnique: true},
					NewColumn: types.SchemaColumn{Name: "email", Type: "VARCHAR(255)"},
				},
			},
		}},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("unique removal must produce SQL")
	}
	if !contains(upSection(t, file), "DROP INDEX `users_email_key`;") {
		t.Errorf("up must drop the unique index:\n%s", file)
	}
	if !contains(downSection(t, file), "ADD UNIQUE INDEX `users_email_key` (`email`);") {
		t.Errorf("down must re-add the unique index:\n%s", file)
	}
}

func TestGenerateSQLFromDiff_AddAndDropColumn_Postgres(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name: "users",
			NewColumns: []types.SchemaColumn{
				{Name: "nickname", Type: "TEXT", Default: "''"},
			},
			DroppedColumns: []types.SchemaColumn{
				{Name: "legacy_flag", Type: "BOOLEAN", Default: "false"},
			},
		}},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("add/drop columns must produce SQL")
	}
	up := upSection(t, file)
	down := downSection(t, file)
	if !contains(up, `ALTER TABLE "users" ADD COLUMN IF NOT EXISTS "nickname"`) {
		t.Errorf("up must add nickname:\n%s", up)
	}
	if !contains(up, `ALTER TABLE "users" DROP COLUMN IF EXISTS "legacy_flag";`) {
		t.Errorf("up must drop legacy_flag:\n%s", up)
	}
	// Down reverses: drop nickname, re-add legacy_flag with its old definition.
	if !contains(down, `DROP COLUMN IF EXISTS "nickname"`) {
		t.Errorf("down must drop nickname:\n%s", down)
	}
	if !contains(down, `ADD COLUMN IF NOT EXISTS "legacy_flag"`) {
		t.Errorf("down must re-add legacy_flag:\n%s", down)
	}
}

// ── ordering invariants ───────────────────────────────────────────────────────

func TestGenerateSQLFromDiff_DownRunsInReverseOrder(t *testing.T) {
	// Two tables created in the up section; the down section must drop them
	// in reverse so FK dependencies unwind correctly.
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		NewTables: []types.SchemaTable{
			{Name: "authors", Columns: []types.SchemaColumn{{Name: "id", Type: "SERIAL", IsPrimary: true}}},
			{Name: "books", Columns: []types.SchemaColumn{
				{Name: "id", Type: "SERIAL", IsPrimary: true},
				{Name: "author_id", Type: "INTEGER", ForeignKeyTable: "authors", ForeignKeyColumn: "id"},
			}},
		},
	}
	file, _ := generateFor(m, diff)
	up := upSection(t, file)
	down := downSection(t, file)

	upIdx := orderOf(t, up, `CREATE TABLE IF NOT EXISTS "authors"`, `CREATE TABLE IF NOT EXISTS "books"`)
	assertAscending(t, "up: authors before books", upIdx)

	downIdx := orderOf(t, down, `DROP TABLE IF EXISTS "books"`, `DROP TABLE IF EXISTS "authors"`)
	assertAscending(t, "down: books before authors", downIdx)
}

func TestGenerateSQLFromDiff_ExtensionBeforeTables_FunctionsAfter(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		NewTables: []types.SchemaTable{
			{Name: "events", Columns: []types.SchemaColumn{{Name: "id", Type: "SERIAL", IsPrimary: true}}},
		},
		NewRawStatements: []string{
			"CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"",
			"CREATE DOMAIN postal_code AS TEXT CHECK (VALUE ~ '^\\d{5}$')",
			"CREATE FUNCTION audit_log() RETURNS trigger AS $$ BEGIN RETURN NEW; END $$ LANGUAGE plpgsql",
		},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("raw statements must produce SQL")
	}
	up := upSection(t, file)
	idx := orderOf(t, up,
		"CREATE EXTENSION",
		"CREATE DOMAIN",
		`CREATE TABLE IF NOT EXISTS "events"`,
		"CREATE FUNCTION",
	)
	assertAscending(t, "extensions < domains < tables < functions", idx)
}

func TestGenerateSQLFromDiff_IndexesSkippedWhenAutoGenerated(t *testing.T) {
	// sqlite_autoindex_* indexes are internal — they must never be emitted.
	m := migratorFor("sqlite")
	diff := &types.SchemaDiff{
		NewTables: []types.SchemaTable{
			{
				Name:    "users",
				Columns: []types.SchemaColumn{{Name: "id", Type: "INTEGER", IsPrimary: true}, {Name: "email", Type: "TEXT", IsUnique: true}},
				Indexes: []types.SchemaIndex{
					{Name: "sqlite_autoindex_users_1", Table: "users", Columns: []string{"email"}, Unique: true},
					{Name: "idx_email", Table: "users", Columns: []string{"email"}},
				},
			},
		},
	}
	file, _ := generateFor(m, diff)
	if contains(file, "sqlite_autoindex") {
		t.Errorf("internal sqlite autoindex must not be emitted:\n%s", file)
	}
	if !contains(file, `CREATE INDEX "idx_email"`) {
		t.Errorf("explicit index must be emitted:\n%s", file)
	}
}

// ── views ─────────────────────────────────────────────────────────────────────

func TestGenerateSQLFromDiff_ViewPassthroughAndDown(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		NewTables: []types.SchemaTable{
			{
				Name: "active_users",
				Columns: []types.SchemaColumn{
					{Name: "/* VIEW */", Type: "CREATE VIEW active_users AS SELECT id, email FROM users WHERE is_active = true"},
				},
			},
		},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("view must produce SQL")
	}
	if !contains(upSection(t, file), "CREATE VIEW active_users AS SELECT id, email FROM users") {
		t.Errorf("view SQL must be passed through verbatim:\n%s", file)
	}
	if !contains(downSection(t, file), `DROP VIEW IF EXISTS "active_users";`) {
		t.Errorf("view down must drop the view:\n%s", file)
	}
}

func TestGenerateSQLFromDiff_MaterializedViewDown(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		NewTables: []types.SchemaTable{
			{
				Name: "user_stats",
				Columns: []types.SchemaColumn{
					{Name: "/* MATERIALIZED VIEW */", Type: "CREATE MATERIALIZED VIEW user_stats AS SELECT user_id, COUNT(*) FROM posts GROUP BY user_id"},
				},
			},
		},
	}
	file, _ := generateFor(m, diff)
	if !contains(downSection(t, file), `DROP MATERIALIZED VIEW IF EXISTS "user_stats";`) {
		t.Errorf("materialized view down must use DROP MATERIALIZED VIEW:\n%s", file)
	}
	// Tables must be created before views are defined.
	up := upSection(t, file)
	_ = up // ordering vs. tables is covered by the raw-statements test; here we pin the down shape
}

// ── ScyllaDB keyspaces and UDTs ───────────────────────────────────────────────

func TestGenerateSQLFromDiff_ScyllaKeyspaceAndUDT(t *testing.T) {
	m := migratorFor("scylla")
	durable := false
	diff := &types.SchemaDiff{
		NewKeyspaces: []types.SchemaKeyspace{
			{Name: "app_ks", Replication: "{'class': 'SimpleStrategy', 'replication_factor': 1}", DurableWrites: &durable},
		},
		NewUDTs: []types.SchemaUDT{
			{Name: "app_ks.address", Fields: []types.SchemaUDTField{{Name: "street", Type: "text"}, {Name: "zip", Type: "int"}}},
		},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("keyspace + UDT must produce SQL")
	}
	up := upSection(t, file)
	if !contains(up, `CREATE KEYSPACE IF NOT EXISTS "app_ks" WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1} AND DURABLE_WRITES = false;`) {
		t.Errorf("keyspace up must include replication + durable writes:\n%s", up)
	}
	if !contains(up, "CREATE TYPE IF NOT EXISTS app_ks.address (street text, zip int);") {
		t.Errorf("UDT up missing:\n%s", up)
	}
	// Keyspace first, then UDT (UDTs live inside keyspaces).
	idx := orderOf(t, up, "CREATE KEYSPACE", "CREATE TYPE")
	assertAscending(t, "keyspace before UDT", idx)
	// Down: reverse.
	down := downSection(t, file)
	if !contains(down, `DROP KEYSPACE IF EXISTS "app_ks";`) {
		t.Errorf("keyspace down missing:\n%s", down)
	}
	if !contains(down, "DROP TYPE IF EXISTS app_ks.address;") {
		t.Errorf("UDT down missing:\n%s", down)
	}
}

// ── SQLite table recreation edge cases ────────────────────────────────────────

func TestGenerateSQLiteTableRecreateSQL_DropsColumnInCopy(t *testing.T) {
	m := migratorFor("sqlite")
	oldTable := types.SchemaTable{
		Name: "t",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "INTEGER", IsPrimary: true},
			{Name: "keep", Type: "TEXT"},
			{Name: "drop_me", Type: "TEXT"},
		},
	}
	newTable := types.SchemaTable{
		Name: "t",
		Columns: []types.SchemaColumn{
			{Name: "id", Type: "INTEGER", IsPrimary: true},
			{Name: "keep", Type: "TEXT"},
			{Name: "added", Type: "TEXT"},
		},
	}
	sql := m.generateSQLiteTableRecreateSQL(oldTable, newTable)
	// Data copy must reference ONLY the common columns — copying a dropped
	// column (or a missing new one) would fail at apply time.
	if !contains(sql, `INSERT INTO "t_new" ("id", "keep") SELECT "id", "keep" FROM "t";`) {
		t.Errorf("copy must use exactly the common columns:\n%s", sql)
	}
	if contains(sql, "drop_me") || contains(sql, `"added"`) && contains(sql, "SELECT \"id\", \"keep\", \"added\"") {
		t.Errorf("copy must not reference dropped/added columns:\n%s", sql)
	}
}

func TestGenerateSQLiteTableRecreateSQL_NoCommonColumnsSkipsInsert(t *testing.T) {
	m := migratorFor("sqlite")
	oldTable := types.SchemaTable{Name: "t", Columns: []types.SchemaColumn{{Name: "a", Type: "INT"}}}
	newTable := types.SchemaTable{Name: "t", Columns: []types.SchemaColumn{{Name: "b", Type: "INT"}}}
	sql := m.generateSQLiteTableRecreateSQL(oldTable, newTable)
	if contains(sql, "INSERT INTO") {
		t.Errorf("no common columns -> INSERT must be skipped:\n%s", sql)
	}
	// The recreate sequence must still be complete.
	for _, want := range []string{"PRAGMA foreign_keys=OFF;", `CREATE TABLE "t_new"`, `DROP TABLE "t";`, `ALTER TABLE "t_new" RENAME TO "t";`, "PRAGMA foreign_keys=ON;"} {
		if !contains(sql, want) {
			t.Errorf("missing %q in:\n%s", want, sql)
		}
	}
}

func TestHasSignificantSQLiteModifications_AllFacets(t *testing.T) {
	m := migratorFor("sqlite")
	col := func(mut func(*types.SchemaColumn)) types.TableDiff {
		oldC := types.SchemaColumn{Name: "c", Type: "TEXT"}
		newC := types.SchemaColumn{Name: "c", Type: "TEXT"}
		mut(&newC)
		return types.TableDiff{
			ModifiedColumns: []types.ColumnDiff{{
				Name: "c", OldType: "TEXT", NewType: newC.Type,
				OldColumn: oldC, NewColumn: newC,
			}},
		}
	}
	significant := []types.TableDiff{
		col(func(c *types.SchemaColumn) { c.Nullable = true }),           // nullability
		col(func(c *types.SchemaColumn) { c.Default = "'x'" }),           // default
		col(func(c *types.SchemaColumn) { c.IsPrimary = true }),          // PK
		col(func(c *types.SchemaColumn) { c.IsUnique = true }),           // unique
		col(func(c *types.SchemaColumn) { c.Check = "length(c) > 0" }),   // check
		col(func(c *types.SchemaColumn) { c.ForeignKeyTable = "other" }), // FK table
		col(func(c *types.SchemaColumn) { c.ForeignKeyColumn = "id" }),   // FK column
		// Real type change.
		{ModifiedColumns: []types.ColumnDiff{{
			Name: "c", OldType: "TEXT", NewType: "INTEGER",
			OldColumn: types.SchemaColumn{Name: "c", Type: "TEXT"},
			NewColumn: types.SchemaColumn{Name: "c", Type: "INTEGER"},
		}}},
	}
	for i, td := range significant {
		if !m.hasSignificantSQLiteModifications(td) {
			t.Errorf("case %d: change should be significant: %+v", i, td.ModifiedColumns[0])
		}
	}
	// Cosmetic: TEXT -> VARCHAR(255) maps to the same affinity.
	cosmetic := types.TableDiff{
		ModifiedColumns: []types.ColumnDiff{{
			Name: "c", OldType: "TEXT", NewType: "VARCHAR(255)",
			OldColumn: types.SchemaColumn{Name: "c", Type: "TEXT"},
			NewColumn: types.SchemaColumn{Name: "c", Type: "VARCHAR(255)"},
		}},
	}
	if m.hasSignificantSQLiteModifications(cosmetic) {
		t.Error("TEXT -> VARCHAR(255) is cosmetic for SQLite")
	}
}

func TestGenerateSQLFromDiff_SQLiteCosmeticChangeNoRecreate(t *testing.T) {
	m := migratorFor("sqlite")
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name: "users",
			ModifiedColumns: []types.ColumnDiff{{
				Name: "name", OldType: "TEXT", NewType: "VARCHAR(255)",
				OldColumn: types.SchemaColumn{Name: "name", Type: "TEXT"},
				NewColumn: types.SchemaColumn{Name: "name", Type: "VARCHAR(255)"},
			}},
			OldTable: types.SchemaTable{Name: "users", Columns: []types.SchemaColumn{{Name: "name", Type: "TEXT"}}},
			NewTable: types.SchemaTable{Name: "users", Columns: []types.SchemaColumn{{Name: "name", Type: "VARCHAR(255)"}}},
		}},
	}
	file, hasSQL := generateFor(m, diff)
	if hasSQL && contains(file, "users_new") {
		t.Errorf("cosmetic type change must not trigger table recreation:\n%s", file)
	}
}

// ── migration file format invariants ──────────────────────────────────────────

func TestFormatMigrationFile_MarkersAndTerminators(t *testing.T) {
	m := migratorFor("postgresql")
	file := m.formatMigrationFileWithDown("test", []string{
		`ALTER TABLE "users" ADD COLUMN "x" INTEGER`,
		"-- just a comment, no semicolon needed",
	}, []string{
		`ALTER TABLE "users" DROP COLUMN "x"`,
	})

	if !contains(file, "-- Migration: test\n") {
		t.Errorf("migration header missing:\n%s", file)
	}
	if !contains(file, "-- +migrate Up") || !contains(file, "-- +migrate Down") {
		t.Errorf("up/down markers missing:\n%s", file)
	}
	// Non-comment statements get a trailing semicolon.
	if !contains(file, `ALTER TABLE "users" ADD COLUMN "x" INTEGER;`) {
		t.Errorf("statements must be terminated with ';':\n%s", file)
	}
	// Known quirk: the UP formatter appends ';' to ANY line lacking one —
	// including comments (the DOWN formatter skips comment lines). The
	// semicolon lands inside the comment text so it is harmless, but this
	// pins the inconsistency so a future cleanup is deliberate.
	if !contains(file, "-- just a comment, no semicolon needed;") {
		t.Errorf("UP section currently appends ';' to comment lines too (documents quirk):\n%s", file)
	}
}

func TestFormatMigrationFile_EmptyDown(t *testing.T) {
	m := migratorFor("sqlite")
	file := m.formatMigrationFileWithDown("t", []string{"CREATE TABLE t (id INT)"}, nil)
	if !contains(file, "-- Add rollback statements here") {
		t.Errorf("empty down must carry the rollback placeholder:\n%s", file)
	}
}

func TestFormatMigrationFile_EmptyUp(t *testing.T) {
	m := migratorFor("sqlite")
	file := m.formatMigrationFileWithDown("t", nil, nil)
	if !contains(file, "-- No migration statements") {
		t.Errorf("empty up must be explicit:\n%s", file)
	}
}

// ── extractDownSQL ────────────────────────────────────────────────────────────

func writeMigrationFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "20260101000000_test.sql")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractDownSQL_BasicSplit(t *testing.T) {
	m := migratorFor("sqlite")
	path := writeMigrationFile(t, `-- Migration: test

-- +migrate Up
CREATE TABLE a (id INT);
CREATE TABLE b (id INT);

-- +migrate Down
DROP TABLE b;
DROP TABLE a;
`)
	got := m.extractDownSQL(path)
	if strings.TrimSpace(got) != "DROP TABLE b;\nDROP TABLE a;" {
		t.Errorf("extractDownSQL = %q", got)
	}
}

func TestExtractDownSQL_CaseInsensitiveMarkers(t *testing.T) {
	m := migratorFor("sqlite")
	path := writeMigrationFile(t, "-- +MIGRATE UP\nCREATE TABLE a (id INT);\n\n-- +Migrate Down\nDROP TABLE a;\n")
	got := m.extractDownSQL(path)
	if !contains(got, "DROP TABLE a;") {
		t.Errorf("markers must match case-insensitively, got %q", got)
	}
	if contains(got, "CREATE TABLE") {
		t.Errorf("up statements must not leak into down, got %q", got)
	}
}

func TestExtractDownSQL_MissingDownSection(t *testing.T) {
	m := migratorFor("sqlite")
	path := writeMigrationFile(t, "-- +migrate Up\nCREATE TABLE a (id INT);\n")
	if got := m.extractDownSQL(path); got != "" {
		t.Errorf("no down section must yield empty string, got %q", got)
	}
}

func TestExtractDownSQL_MissingFile(t *testing.T) {
	m := migratorFor("sqlite")
	if got := m.extractDownSQL(filepath.Join(t.TempDir(), "missing.sql")); got != "" {
		t.Errorf("missing file must yield empty string, got %q", got)
	}
}

func TestExtractDownSQL_SqlInsideDollarQuotes(t *testing.T) {
	// A function body containing the literal words "+migrate Down" inside $$
	// must not end the up section.
	m := migratorFor("postgresql")
	path := writeMigrationFile(t, `-- +migrate Up
CREATE FUNCTION f() RETURNS void AS $$
BEGIN
  -- +migrate Down inside a string is not a real marker
  RAISE NOTICE 'hi';
END
$$ LANGUAGE plpgsql;

-- +migrate Down
DROP FUNCTION f();
`)
	got := m.extractDownSQL(path)
	if !contains(got, "DROP FUNCTION f();") {
		t.Errorf("down section must be found after $$ body, got %q", got)
	}
}

// ── rename detection data flow (schema-level) ────────────────────────────────

func TestGenerateSQLFromDiff_ColumnRenameBeforeAddDrop(t *testing.T) {
	// When a column is renamed AND others added, renames must come first so
	// data is preserved before any structural churn.
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name:           "users",
			RenamedColumns: []types.RenameOp{{Table: "users", OldName: "name", NewName: "full_name"}},
			NewColumns:     []types.SchemaColumn{{Name: "nickname", Type: "TEXT"}},
			NewTable: types.SchemaTable{Columns: []types.SchemaColumn{
				{Name: "full_name", Type: "TEXT"},
				{Name: "nickname", Type: "TEXT"},
			}},
		}},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("expected SQL")
	}
	up := upSection(t, file)
	idx := orderOf(t, up,
		`RENAME COLUMN "name" TO "full_name"`,
		`ADD COLUMN IF NOT EXISTS "nickname"`,
	)
	assertAscending(t, "rename before add", idx)
}

func TestGenerateSQLFromDiff_TableRenameBeforeCreates(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		RenamedTables: []types.RenameOp{{OldName: "users", NewName: "members"}},
		NewTables: []types.SchemaTable{
			{Name: "audit", Columns: []types.SchemaColumn{{Name: "id", Type: "SERIAL", IsPrimary: true}}},
		},
	}
	file, _ := generateFor(m, diff)
	up := upSection(t, file)
	idx := orderOf(t, up,
		`ALTER TABLE "users" RENAME TO "members"`,
		`CREATE TABLE IF NOT EXISTS "audit"`,
	)
	assertAscending(t, "renames before creates", idx)
}

// ── enum + table interplay ────────────────────────────────────────────────────

func TestGenerateSQLFromDiff_EnumCreatedBeforeTableUsingIt(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		NewEnums: []types.SchemaEnum{{Name: "post_status", Values: []string{"draft", "live"}}},
		NewTables: []types.SchemaTable{
			{Name: "posts", Columns: []types.SchemaColumn{
				{Name: "id", Type: "SERIAL", IsPrimary: true},
				{Name: "status", Type: "post_status"},
			}},
		},
	}
	file, _ := generateFor(m, diff)
	up := upSection(t, file)
	idx := orderOf(t, up, "CREATE TYPE", `CREATE TABLE IF NOT EXISTS "posts"`)
	assertAscending(t, "enum before table", idx)
}

// ── empty diff / guards ───────────────────────────────────────────────────────

func TestGenerateSQLFromDiff_OnlyRenamedTable(t *testing.T) {
	m := migratorFor("sqlite")
	diff := &types.SchemaDiff{RenamedTables: []types.RenameOp{{OldName: "a", NewName: "b"}}}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("rename alone is executable")
	}
	if !contains(upSection(t, file), `ALTER TABLE "a" RENAME TO "b";`) {
		t.Errorf("rename missing:\n%s", file)
	}
}

func TestGenerateSQLFromDiff_DroppedIndexesEmitted(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		DroppedIndexes: []types.SchemaIndex{{Name: "idx_old", Table: "users", Columns: []string{"email"}}},
	}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("dropped index is executable")
	}
	if !contains(upSection(t, file), `DROP INDEX IF EXISTS "idx_old";`) {
		t.Errorf("dropped index missing from up:\n%s", file)
	}
}

func TestGenerateSQLFromDiff_NewIndexDownDrops(t *testing.T) {
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{
		NewIndexes: []types.SchemaIndex{{Name: "idx_new", Table: "users", Columns: []string{"email"}}},
	}
	file, _ := generateFor(m, diff)
	if !contains(upSection(t, file), `CREATE INDEX "idx_new"`) {
		t.Errorf("new index missing from up:\n%s", file)
	}
	if !contains(downSection(t, file), `DROP INDEX IF EXISTS "idx_new";`) {
		t.Errorf("new index down missing:\n%s", file)
	}
}

// ── MySQL-specific down for modified column ───────────────────────────────────

func TestGenerateSQLFromDiff_MySQLTypeChangeDownReverts(t *testing.T) {
	m := migratorFor("mysql")
	diff := &types.SchemaDiff{
		ModifiedTables: []types.TableDiff{{
			Name: "users",
			ModifiedColumns: []types.ColumnDiff{{
				Name:      "email",
				OldType:   "VARCHAR(100)",
				NewType:   "VARCHAR(255)",
				OldColumn: types.SchemaColumn{Name: "email", Type: "VARCHAR(100)", Nullable: false},
				NewColumn: types.SchemaColumn{Name: "email", Type: "VARCHAR(255)", Nullable: false},
			}},
		}},
	}
	file, _ := generateFor(m, diff)
	if !contains(upSection(t, file), "MODIFY COLUMN `email` VARCHAR(255)") {
		t.Errorf("up must modify to new type:\n%s", file)
	}
	if !contains(downSection(t, file), "MODIFY COLUMN `email` VARCHAR(100)") {
		t.Errorf("down must revert to old type:\n%s", file)
	}
}

// ── snapshot-less down for dropped tables ─────────────────────────────────────

func TestGenerateSQLFromDiff_DroppedTableDownWithoutSnapshot(t *testing.T) {
	// No schemaManager -> no restore info. Down must still contain the drop in
	// the UP section and must not crash; down simply lacks the CREATE.
	m := migratorFor("postgresql")
	diff := &types.SchemaDiff{DroppedTables: []string{"legacy"}}
	file, hasSQL := generateFor(m, diff)
	if !hasSQL {
		t.Fatal("drop table is executable")
	}
	if !contains(upSection(t, file), `DROP TABLE IF EXISTS "legacy" CASCADE;`) {
		t.Errorf("up must drop the table:\n%s", file)
	}
}

// ── extractRefTables (materialized view dependency extraction) ───────────────

func TestExtractRefTables(t *testing.T) {
	tests := []struct {
		sql  string
		want []string
	}{
		{
			sql:  "CREATE MATERIALIZED VIEW v AS SELECT id FROM users JOIN posts ON posts.user_id = users.id",
			want: []string{"users", "POSTS"}, // JOIN names are captured from the uppercased SQL — documents quirk
		},
		{
			sql:  "CREATE MATERIALIZED VIEW v AS SELECT * FROM a, b",
			want: []string{"a,"}, // comma sticks to the FROM capture — documents current behavior
		},
		{
			sql:  "SELECT 1",
			want: nil,
		},
	}
	for _, tt := range tests {
		got := extractRefTables(tt.sql)
		if len(got) != len(tt.want) {
			t.Errorf("extractRefTables(%q) = %v, want %v", tt.sql, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("extractRefTables(%q)[%d] = %q, want %q", tt.sql, i, got[i], tt.want[i])
			}
		}
	}
}

func TestIsTableInNewTables_QualifiedNames(t *testing.T) {
	newTables := []types.SchemaTable{
		{Name: "app.users"},
		{Name: "posts"},
	}
	cases := []struct {
		name string
		want bool
	}{
		{"users", true},        // bare name matches qualified table (suffix rule)
		{"app.users", true},    // exact
		{"other.users", false}, // both qualified with different prefixes — no match (correct)
		{"comments", false},
	}
	for _, c := range cases {
		if got := isTableInNewTables(c.name, newTables); got != c.want {
			t.Errorf("isTableInNewTables(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}
