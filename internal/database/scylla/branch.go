package scylla

import (
	"context"
	"fmt"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func (a *Adapter) CreateBranchSchema(ctx context.Context, branchName string) error {
	ks := sanitizeKeyspace(branchName)
	return a.session.Query(fmt.Sprintf(
		`CREATE KEYSPACE IF NOT EXISTS "%s" WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1}`, ks,
	)).ExecContext(ctx)
}

func (a *Adapter) DropBranchSchema(ctx context.Context, branchName string) error {
	ks := sanitizeKeyspace(branchName)
	return a.session.Query(fmt.Sprintf(`DROP KEYSPACE IF EXISTS "%s"`, ks)).ExecContext(ctx)
}

func (a *Adapter) CloneSchemaToBranch(ctx context.Context, sourceDB, targetDB string) error {
	if err := a.DropBranchSchema(ctx, targetDB); err != nil {
		return err
	}
	if err := a.CreateBranchSchema(ctx, targetDB); err != nil {
		return err
	}

	// Fetch column definitions from source keyspace
	iter := a.session.Query(
		`SELECT table_name, column_name, type, kind FROM system_schema.columns WHERE keyspace_name = ? ALLOW FILTERING`, sourceDB,
	).IterContext(ctx)
	defer iter.Close()

	type colInfo struct {
		kind    string
		cqlType string
	}
	tableMap := make(map[string]map[string]colInfo)
	tableOrder := []string{}

	var tblName, colName, cqlType, kind string
	for iter.Scan(&tblName, &colName, &cqlType, &kind) {
		if strings.HasPrefix(tblName, "_flash_") {
			continue
		}
		if _, ok := tableMap[tblName]; !ok {
			tableMap[tblName] = make(map[string]colInfo)
			tableOrder = append(tableOrder, tblName)
		}
		tableMap[tblName][colName] = colInfo{kind: kind, cqlType: cqlType}
	}
	if err := iter.Close(); err != nil {
		return err
	}

	for _, tbl := range tableOrder {
		cols := tableMap[tbl]
		var pkPart, clusterPart, colDefs []string
		for col, info := range cols {
			colDefs = append(colDefs, fmt.Sprintf(`"%s" %s`, col, info.cqlType))
			switch info.kind {
			case "partition_key":
				pkPart = append(pkPart, `"`+col+`"`)
			case "clustering":
				clusterPart = append(clusterPart, `"`+col+`"`)
			}
		}
		var pk string
		if len(clusterPart) > 0 {
			pk = fmt.Sprintf("(%s), %s", strings.Join(pkPart, ", "), strings.Join(clusterPart, ", "))
		} else {
			pk = strings.Join(pkPart, ", ")
		}
		colDefs = append(colDefs, fmt.Sprintf("PRIMARY KEY (%s)", pk))
		ddl := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s"."%s" (%s)`, targetDB, tbl, strings.Join(colDefs, ", "))
		if err := a.session.Query(ddl).ExecContext(ctx); err != nil {
			return fmt.Errorf("failed to clone table %s: %w", tbl, err)
		}
	}
	return nil
}

func (a *Adapter) GetSchemaForBranch(ctx context.Context, branchDB string) ([]types.SchemaTable, error) {
	ks := sanitizeKeyspace(branchDB)
	iter := a.session.Query(
		`SELECT table_name, column_name, type, kind FROM system_schema.columns WHERE keyspace_name = ? ALLOW FILTERING`, ks,
	).IterContext(ctx)
	defer iter.Close()

	tableMap := make(map[string]*types.SchemaTable)
	var order []string

	var tableName, colName, cqlType, kind string
	for iter.Scan(&tableName, &colName, &cqlType, &kind) {
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
		return nil, err
	}

	tables := make([]types.SchemaTable, 0, len(order))
	for _, n := range order {
		tables = append(tables, *tableMap[n])
	}
	return tables, nil
}

func (a *Adapter) SetActiveSchema(_ context.Context, schemaName string) error {
	a.keyspace = sanitizeKeyspace(schemaName)
	return nil
}

func (a *Adapter) GetTableNamesInSchema(ctx context.Context, dbName string) ([]string, error) {
	ks := sanitizeKeyspace(dbName)
	iter := a.session.Query(
		`SELECT table_name FROM system_schema.tables WHERE keyspace_name = ? ALLOW FILTERING`, ks,
	).IterContext(ctx)
	defer iter.Close()

	var names []string
	var n string
	for iter.Scan(&n) {
		names = append(names, n)
	}
	return names, iter.Close()
}
