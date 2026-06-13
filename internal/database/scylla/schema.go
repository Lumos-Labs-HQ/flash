package scylla

import (
	"context"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func (a *Adapter) GetAllTableNames(ctx context.Context) ([]string, error) {
	ks := a.currentKeyspace()
	iter := a.session.Query(
		`SELECT table_name FROM system_schema.tables WHERE keyspace_name = ? ALLOW FILTERING`, ks,
	).IterContext(ctx)
	defer iter.Close()

	var names []string
	var table string
	for iter.Scan(&table) {
		if !strings.HasPrefix(table, "_flash_") {
			names = append(names, table)
		}
	}
	if err := iter.Close(); err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, err
	}
	return names, nil
}

func (a *Adapter) GetCurrentSchema(ctx context.Context) ([]types.SchemaTable, error) {
	return a.PullCompleteSchema(ctx)
}

func (a *Adapter) GetCurrentEnums(_ context.Context) ([]types.SchemaEnum, error) {
	return nil, nil
}

func (a *Adapter) GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error) {
	ks := a.currentKeyspace()
	iter := a.session.Query(
		`SELECT column_name, type, kind FROM system_schema.columns WHERE keyspace_name = ? AND table_name = ? ALLOW FILTERING`,
		ks, tableName,
	).IterContext(ctx)
	defer iter.Close()

	var cols []types.SchemaColumn
	var name, cqlType, kind string
	for iter.Scan(&name, &cqlType, &kind) {
		cols = append(cols, types.SchemaColumn{
			Name:      name,
			Type:      cqlType,
			Nullable:  kind != "partition_key",
			IsPrimary: kind == "partition_key" || kind == "clustering",
		})
	}
	return cols, iter.Close()
}

func (a *Adapter) GetTableIndexes(_ context.Context, _ string) ([]types.SchemaIndex, error) {
	return nil, nil
}

func (a *Adapter) PullCompleteSchema(ctx context.Context) ([]types.SchemaTable, error) {
	ks := a.currentKeyspace()
	iter := a.session.Query(
		`SELECT table_name, column_name, type, kind FROM system_schema.columns WHERE keyspace_name = ? ALLOW FILTERING`, ks,
	).IterContext(ctx)
	defer iter.Close()

	tableMap := make(map[string]*types.SchemaTable)
	var order []string

	var tableName, colName, cqlType, kind string
	for iter.Scan(&tableName, &colName, &cqlType, &kind) {
		if strings.HasPrefix(tableName, "_flash_") {
			continue
		}
		if _, ok := tableMap[tableName]; !ok {
			tableMap[tableName] = &types.SchemaTable{Name: tableName}
			order = append(order, tableName)
		}
		tableMap[tableName].Columns = append(tableMap[tableName].Columns, types.SchemaColumn{
			Name:      colName,
			Type:      cqlType,
			Nullable:  kind != "partition_key",
			IsPrimary: kind == "partition_key" || kind == "clustering",
		})
	}
	if err := iter.Close(); err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return nil, nil
		}
		return nil, err
	}

	tables := make([]types.SchemaTable, 0, len(order))
	for _, name := range order {
		tables = append(tables, *tableMap[name])
	}
	return tables, nil
}
