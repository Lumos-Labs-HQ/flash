package mongodb

import (
	"context"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

// MongoDB adapter is a stub (no ORM support). Tests verify the stub contract:
// all methods return zero values without panicking.

func newAdapter() *Adapter {
	return New()
}

func TestMongoAdapter_StubMethods_NoPanic(t *testing.T) {
	ctx := context.Background()
	a := newAdapter()

	// All stub methods must return nil/zero without panicking.
	a.CreateMigrationsTable(ctx)
	a.EnsureMigrationTableCompatibility(ctx)
	a.CleanupBrokenMigrationRecords(ctx)
	a.GetAppliedMigrations(ctx)
	a.RecordMigration(ctx, "id", "name", "checksum")
	a.RemoveMigrationRecord(ctx, "id")
	a.ExecuteMigration(ctx, "SELECT 1")
	a.ExecuteAndRecordMigration(ctx, "id", "name", "checksum", "SELECT 1")
	a.ExecuteQuery(ctx, "SELECT 1")
	a.GetCurrentSchema(ctx)
	a.GetCurrentEnums(ctx)
	a.GetTableIndexes(ctx, "users")
	a.PullCompleteSchema(ctx)
	a.CheckTableExists(ctx, "users")
	a.CheckColumnExists(ctx, "users", "id")
	a.CheckNotNullConstraint(ctx, "users", "id")
	a.CheckForeignKeyConstraint(ctx, "users", "fk")
	a.CheckUniqueConstraint(ctx, "users", "uq")
	a.DropTable(ctx, "users")
	a.DropEnum(ctx, "status")
	a.CreateBranchSchema(ctx, "branch")
	a.DropBranchSchema(ctx, "branch")
	a.CloneSchemaToBranch(ctx, "src", "dst")
	a.GetSchemaForBranch(ctx, "branch")
	a.SetActiveSchema(ctx, "schema")
	a.GetTableNamesInSchema(ctx, "schema")
}

func TestMongoAdapter_SQLGeneration_ReturnsEmpty(t *testing.T) {
	a := newAdapter()
	table := types.SchemaTable{Name: "users", Columns: []types.SchemaColumn{{Name: "id", Type: "string"}}}
	col := types.SchemaColumn{Name: "email", Type: "string"}
	idx := types.SchemaIndex{Name: "idx", Table: "users", Columns: []string{"id"}}

	if sql := a.GenerateCreateTableSQL(table); sql != "" {
		t.Errorf("GenerateCreateTableSQL = %q, want empty", sql)
	}
	if sql := a.GenerateAddColumnSQL("users", col); sql != "" {
		t.Errorf("GenerateAddColumnSQL = %q, want empty", sql)
	}
	if sql := a.GenerateDropColumnSQL("users", "email"); sql != "" {
		t.Errorf("GenerateDropColumnSQL = %q, want empty", sql)
	}
	if sql := a.GenerateAddIndexSQL(idx); sql != "" {
		t.Errorf("GenerateAddIndexSQL = %q, want empty", sql)
	}
	if sql := a.GenerateDropIndexSQL(idx); sql != "" {
		t.Errorf("GenerateDropIndexSQL = %q, want empty", sql)
	}
}

func TestMongoAdapter_MapColumnType(t *testing.T) {
	a := newAdapter()
	if got := a.MapColumnType("string"); got != "string" {
		t.Errorf("MapColumnType = %q, want string", got)
	}
}

func TestMongoAdapter_FormatColumnType(t *testing.T) {
	a := newAdapter()
	col := types.SchemaColumn{Name: "id", Type: "ObjectId"}
	if got := a.FormatColumnType(col); got != "ObjectId" {
		t.Errorf("FormatColumnType = %q, want ObjectId", got)
	}
}

func TestMongoAdapter_Close(t *testing.T) {
	a := newAdapter()
	if err := a.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}
}
