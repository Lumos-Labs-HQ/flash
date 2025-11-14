package sqlite

import (
	"context"
	"fmt"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func (a *Adapter) CreateBranchSchema(ctx context.Context, branchName string) error {
	return fmt.Errorf("SQLite branching requires separate database files - use file-based branching")
}

func (a *Adapter) DropBranchSchema(ctx context.Context, branchName string) error {
	return fmt.Errorf("SQLite branching requires separate database files - use file-based branching")
}

func (a *Adapter) CloneSchemaToBranch(ctx context.Context, sourceSchema, targetSchema string) error {
	return fmt.Errorf("SQLite branching requires separate database files - use file-based branching")
}

func (a *Adapter) GetSchemaForBranch(ctx context.Context, branchSchema string) ([]types.SchemaTable, error) {
	return a.GetCurrentSchema(ctx)
}

func (a *Adapter) SetActiveSchema(ctx context.Context, schemaName string) error {
	return nil
}

func (a *Adapter) GetTableNamesInSchema(ctx context.Context, schemaName string) ([]string, error) {
	return a.GetAllTableNames(ctx)
}
