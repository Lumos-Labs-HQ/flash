package turso

import (
	"context"
	"fmt"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func (a *Adapter) CreateBranchSchema(ctx context.Context, branchName string) error {
	return fmt.Errorf("branching for Turso must be done via the Turso API or dashboard: https://docs.turso.tech/features/branching")
}

func (a *Adapter) DropBranchSchema(ctx context.Context, branchName string) error {
	return fmt.Errorf("branching for Turso must be done via the Turso API or dashboard: https://docs.turso.tech/features/branching")
}

func (a *Adapter) CloneSchemaToBranch(ctx context.Context, sourceSchema, targetSchema string) error {
	return fmt.Errorf("branching for Turso must be done via the Turso API or dashboard: https://docs.turso.tech/features/branching")
}

func (a *Adapter) GetSchemaForBranch(ctx context.Context, branchSchema string) ([]types.SchemaTable, error) {
	return nil, fmt.Errorf("branching for Turso must be done via the Turso API or dashboard: https://docs.turso.tech/features/branching")
}

func (a *Adapter) SetActiveSchema(ctx context.Context, schemaName string) error {
	if schemaName == "" || schemaName == "main" {
		return nil
	}
	return fmt.Errorf("branching for Turso must be done via the Turso API or dashboard: https://docs.turso.tech/features/branching")
}

func (a *Adapter) GetTableNamesInSchema(ctx context.Context, schemaName string) ([]string, error) {
	if schemaName == "" || schemaName == "main" {
		return a.GetAllTableNames(ctx)
	}
	return nil, fmt.Errorf("branching for Turso must be done via the Turso API or dashboard: https://docs.turso.tech/features/branching")
}
