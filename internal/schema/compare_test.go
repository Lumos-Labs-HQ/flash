package schema

import (
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/database/postgres"
	"github.com/Lumos-Labs-HQ/flash/internal/database/sqlite"
	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

// ── columnsEqual with nil adapter (exact comparison) ──────────────────────────

func TestColumnsEqual_NilAdapter_ExactMatch(t *testing.T) {
	sm := NewSchemaManager(nil)
	a := types.SchemaColumn{Name: "id", Type: "INT", Nullable: false, IsPrimary: true}
	b := types.SchemaColumn{Name: "id", Type: "INT", Nullable: false, IsPrimary: true}
	if !sm.columnsEqual(a, b) {
		t.Error("expected equal columns")
	}
}

func TestColumnsEqual_NilAdapter_TypeMismatch(t *testing.T) {
	sm := NewSchemaManager(nil)
	a := types.SchemaColumn{Name: "id", Type: "INT"}
	b := types.SchemaColumn{Name: "id", Type: "TEXT"}
	if sm.columnsEqual(a, b) {
		t.Error("expected different types to not be equal")
	}
}

func TestColumnsEqual_NilAdapter_NullableMismatch(t *testing.T) {
	sm := NewSchemaManager(nil)
	a := types.SchemaColumn{Name: "name", Type: "TEXT", Nullable: true}
	b := types.SchemaColumn{Name: "name", Type: "TEXT", Nullable: false}
	if sm.columnsEqual(a, b) {
		t.Error("expected different nullable to not be equal")
	}
}

// ── columnsEqual with PostgreSQL adapter (type normalization) ─────────────────

func TestColumnsEqual_Postgres_SerialVsInteger(t *testing.T) {
	sm := NewSchemaManager(postgres.New())
	// Schema parser extracts "SERIAL" from CREATE TABLE
	schemaCol := types.SchemaColumn{Name: "id", Type: "SERIAL", IsPrimary: true}
	// DB introspection returns "INTEGER" for a SERIAL column
	dbCol := types.SchemaColumn{Name: "id", Type: "INTEGER", IsPrimary: true}
	if !sm.columnsEqual(schemaCol, dbCol) {
		t.Error("SERIAL and INTEGER should be equal for PostgreSQL")
	}
}

func TestColumnsEqual_Postgres_BigserialVsBigint(t *testing.T) {
	sm := NewSchemaManager(postgres.New())
	schemaCol := types.SchemaColumn{Name: "id", Type: "BIGSERIAL", IsPrimary: true}
	dbCol := types.SchemaColumn{Name: "id", Type: "BIGINT", IsPrimary: true}
	if !sm.columnsEqual(schemaCol, dbCol) {
		t.Error("BIGSERIAL and BIGINT should be equal for PostgreSQL")
	}
}

func TestColumnsEqual_Postgres_VarcharVsText(t *testing.T) {
	sm := NewSchemaManager(postgres.New())
	// These are different types and should NOT be normalized to equal
	schemaCol := types.SchemaColumn{Name: "name", Type: "VARCHAR(255)", Nullable: false}
	dbCol := types.SchemaColumn{Name: "name", Type: "TEXT", Nullable: false}
	if sm.columnsEqual(schemaCol, dbCol) {
		t.Error("VARCHAR(255) and TEXT should NOT be equal for PostgreSQL")
	}
}

// ── columnsEqual PK nullable normalization ────────────────────────────────────

func TestColumnsEqual_PKImpliesNotNull(t *testing.T) {
	sm := NewSchemaManager(postgres.New())
	// Schema parser leaves Nullable=true when no explicit NOT NULL is present,
	// even for PRIMARY KEY columns.
	schemaCol := types.SchemaColumn{Name: "id", Type: "INTEGER", Nullable: true, IsPrimary: true}
	// DB introspection reports Nullable=false because PK implies NOT NULL.
	dbCol := types.SchemaColumn{Name: "id", Type: "INTEGER", Nullable: false, IsPrimary: true}
	if !sm.columnsEqual(schemaCol, dbCol) {
		t.Error("PK column with Nullable=true should equal PK column with Nullable=false")
	}
}

func TestColumnsEqual_PKImpliesNotNull_NonPK(t *testing.T) {
	sm := NewSchemaManager(postgres.New())
	// For non-PK columns, nullable should be compared normally.
	schemaCol := types.SchemaColumn{Name: "name", Type: "TEXT", Nullable: true}
	dbCol := types.SchemaColumn{Name: "name", Type: "TEXT", Nullable: false}
	if sm.columnsEqual(schemaCol, dbCol) {
		t.Error("non-PK columns with different nullable should NOT be equal")
	}
}

func TestColumnsEqual_PKImpliesNotNull_BothTrue(t *testing.T) {
	sm := NewSchemaManager(postgres.New())
	// Both have IsPrimary=true and Nullable=true — should match.
	a := types.SchemaColumn{Name: "id", Type: "INTEGER", Nullable: true, IsPrimary: true}
	b := types.SchemaColumn{Name: "id", Type: "INTEGER", Nullable: true, IsPrimary: true}
	if !sm.columnsEqual(a, b) {
		t.Error("identical PK columns should be equal")
	}
}

// ── columnsEqual default values ───────────────────────────────────────────────

func TestColumnsEqual_DefaultMismatch(t *testing.T) {
	sm := NewSchemaManager(nil)
	a := types.SchemaColumn{Name: "status", Type: "TEXT", Default: "active"}
	b := types.SchemaColumn{Name: "status", Type: "TEXT", Default: "inactive"}
	if sm.columnsEqual(a, b) {
		t.Error("columns with different defaults should NOT be equal")
	}
}

func TestColumnsEqual_DefaultEmptyVsNullString(t *testing.T) {
	sm := NewSchemaManager(nil)
	// Empty default vs explicit NULL default.
	a := types.SchemaColumn{Name: "name", Type: "TEXT", Default: ""}
	b := types.SchemaColumn{Name: "name", Type: "TEXT", Default: "NULL"}
	if sm.columnsEqual(a, b) {
		t.Error("columns with '' and 'NULL' defaults should NOT be equal")
	}
}

// ── columnsEqual foreign keys ─────────────────────────────────────────────────

func TestColumnsEqual_FKMatch(t *testing.T) {
	sm := NewSchemaManager(nil)
	a := types.SchemaColumn{
		Name: "user_id", Type: "INTEGER",
		ForeignKeyTable: "users", ForeignKeyColumn: "id", OnDeleteAction: "CASCADE",
	}
	b := types.SchemaColumn{
		Name: "user_id", Type: "INTEGER",
		ForeignKeyTable: "users", ForeignKeyColumn: "id", OnDeleteAction: "CASCADE",
	}
	if !sm.columnsEqual(a, b) {
		t.Error("columns with identical FK should be equal")
	}
}

func TestColumnsEqual_FKTableMismatch(t *testing.T) {
	sm := NewSchemaManager(nil)
	a := types.SchemaColumn{Name: "user_id", Type: "INTEGER", ForeignKeyTable: "users"}
	b := types.SchemaColumn{Name: "user_id", Type: "INTEGER", ForeignKeyTable: "admins"}
	if sm.columnsEqual(a, b) {
		t.Error("columns with different FK tables should NOT be equal")
	}
}

func TestColumnsEqual_FKActionMismatch(t *testing.T) {
	sm := NewSchemaManager(nil)
	a := types.SchemaColumn{Name: "user_id", Type: "INTEGER", ForeignKeyTable: "users", ForeignKeyColumn: "id", OnDeleteAction: "CASCADE"}
	b := types.SchemaColumn{Name: "user_id", Type: "INTEGER", ForeignKeyTable: "users", ForeignKeyColumn: "id", OnDeleteAction: "SET NULL"}
	if sm.columnsEqual(a, b) {
		t.Error("columns with different ON DELETE actions should NOT be equal")
	}
}

// ── columnsEqual with SQLite adapter (cosmetic type normalization) ────────────

func TestColumnsEqual_Sqlite_VarcharVsText(t *testing.T) {
	sm := NewSchemaManager(sqlite.New())
	// SQLite MapColumnType normalizes VARCHAR(255) → TEXT
	schemaCol := types.SchemaColumn{Name: "name", Type: "VARCHAR(255)", Nullable: false}
	dbCol := types.SchemaColumn{Name: "name", Type: "TEXT", Nullable: false}
	if !sm.columnsEqual(schemaCol, dbCol) {
		t.Error("VARCHAR(255) and TEXT should be equal for SQLite")
	}
}

func TestColumnsEqual_Sqlite_IntVsInteger(t *testing.T) {
	sm := NewSchemaManager(sqlite.New())
	schemaCol := types.SchemaColumn{Name: "id", Type: "INT", Nullable: false, IsPrimary: true}
	dbCol := types.SchemaColumn{Name: "id", Type: "INTEGER", Nullable: false, IsPrimary: true}
	if !sm.columnsEqual(schemaCol, dbCol) {
		t.Error("INT and INTEGER should be equal for SQLite")
	}
}

// ── columnsEqual unique constraint ────────────────────────────────────────────

func TestColumnsEqual_UniqueMismatch(t *testing.T) {
	sm := NewSchemaManager(nil)
	a := types.SchemaColumn{Name: "email", Type: "TEXT", IsUnique: true}
	b := types.SchemaColumn{Name: "email", Type: "TEXT", IsUnique: false}
	if sm.columnsEqual(a, b) {
		t.Error("columns with different unique should NOT be equal")
	}
}

// ── columnsEqual name mismatch ────────────────────────────────────────────────

func TestColumnsEqual_NameMismatch(t *testing.T) {
	sm := NewSchemaManager(nil)
	a := types.SchemaColumn{Name: "email", Type: "TEXT"}
	b := types.SchemaColumn{Name: "email_address", Type: "TEXT"}
	if sm.columnsEqual(a, b) {
		t.Error("columns with different names should NOT be equal")
	}
}
