package clickhouse

import (
	"context"
	"fmt"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func (a *Adapter) CreateBranchSchema(ctx context.Context, branchName string) error {
	_, err := a.db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", branchName))
	return err
}

func (a *Adapter) DropBranchSchema(ctx context.Context, branchName string) error {
	_, err := a.db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", branchName))
	return err
}

func (a *Adapter) CloneSchemaToBranch(ctx context.Context, sourceDB, targetDB string) error {
	if err := a.DropBranchSchema(ctx, targetDB); err != nil {
		return err
	}
	if err := a.CreateBranchSchema(ctx, targetDB); err != nil {
		return err
	}

	rows, err := a.db.QueryContext(ctx,
		`SELECT name FROM system.tables WHERE database = ? AND name NOT LIKE '_flash_%'`, sourceDB)
	if err != nil {
		return err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			continue
		}
		tables = append(tables, t)
	}

	for _, t := range tables {
		q := fmt.Sprintf("CREATE TABLE `%s`.`%s` AS `%s`.`%s`", targetDB, t, sourceDB, t)
		if _, err := a.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("failed to clone table %s: %w", t, err)
		}
	}
	return nil
}

func (a *Adapter) GetSchemaForBranch(ctx context.Context, branchDB string) ([]types.SchemaTable, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT table, name, type, is_in_primary_key FROM system.columns WHERE database = ? ORDER BY table, position`, branchDB)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tableMap := make(map[string]*types.SchemaTable)
	var order []string
	for rows.Next() {
		var tableName, colName, chType string
		var isPK uint8
		if err := rows.Scan(&tableName, &colName, &chType, &isPK); err != nil {
			continue
		}
		if _, ok := tableMap[tableName]; !ok {
			tableMap[tableName] = &types.SchemaTable{Name: tableName}
			order = append(order, tableName)
		}
		tableMap[tableName].Columns = append(tableMap[tableName].Columns, types.SchemaColumn{
			Name:      colName,
			Type:      chType,
			Nullable:  strings.HasPrefix(strings.ToLower(chType), "nullable("),
			IsPrimary: isPK > 0,
		})
	}

	tables := make([]types.SchemaTable, 0, len(order))
	for _, n := range order {
		tables = append(tables, *tableMap[n])
	}
	return tables, rows.Err()
}

func (a *Adapter) SetActiveSchema(_ context.Context, _ string) error {
	return nil
}

func (a *Adapter) GetTableNamesInSchema(ctx context.Context, dbName string) ([]string, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT name FROM system.tables WHERE database = ? ORDER BY name`, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			continue
		}
		names = append(names, n)
	}
	return names, rows.Err()
}
