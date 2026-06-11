package clickhouse

import (
	"context"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func (a *Adapter) GetAllTableNames(ctx context.Context) ([]string, error) {
	db := a.currentDatabase(ctx)
	rows, err := a.db.QueryContext(ctx,
		`SELECT name FROM system.tables WHERE database = ? AND name NOT LIKE '_flash_%' ORDER BY name`, db)
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

func (a *Adapter) GetCurrentSchema(ctx context.Context) ([]types.SchemaTable, error) {
	return a.PullCompleteSchema(ctx)
}

func (a *Adapter) GetCurrentEnums(_ context.Context) ([]types.SchemaEnum, error) {
	return nil, nil
}

func (a *Adapter) GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error) {
	db := a.currentDatabase(ctx)
	rows, err := a.db.QueryContext(ctx,
		`SELECT name, type, default_expression, is_in_primary_key FROM system.columns WHERE database = ? AND table = ? ORDER BY position`,
		db, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []types.SchemaColumn
	for rows.Next() {
		var name, chType, defaultExpr string
		var isPK uint8
		if err := rows.Scan(&name, &chType, &defaultExpr, &isPK); err != nil {
			continue
		}
		cols = append(cols, types.SchemaColumn{
			Name:      name,
			Type:      chType,
			Nullable:  strings.HasPrefix(strings.ToLower(chType), "nullable("),
			Default:   defaultExpr,
			IsPrimary: isPK > 0,
		})
	}
	return cols, rows.Err()
}

func (a *Adapter) GetTableIndexes(_ context.Context, _ string) ([]types.SchemaIndex, error) {
	return nil, nil
}

func (a *Adapter) PullCompleteSchema(ctx context.Context) ([]types.SchemaTable, error) {
	db := a.currentDatabase(ctx)
	rows, err := a.db.QueryContext(ctx,
		`SELECT table, name, type, default_expression, is_in_primary_key
		 FROM system.columns
		 WHERE database = ? AND table NOT LIKE '_flash_%'
		 ORDER BY table, position`,
		db)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tableMap := make(map[string]*types.SchemaTable)
	var order []string

	for rows.Next() {
		var tableName, colName, chType, defaultExpr string
		var isPK uint8
		if err := rows.Scan(&tableName, &colName, &chType, &defaultExpr, &isPK); err != nil {
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
			Default:   defaultExpr,
			IsPrimary: isPK > 0,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tables := make([]types.SchemaTable, 0, len(order))
	for _, name := range order {
		tables = append(tables, *tableMap[name])
	}
	return tables, nil
}
