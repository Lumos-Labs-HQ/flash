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
	)).WithContext(ctx).Exec()
}

func (a *Adapter) DropBranchSchema(ctx context.Context, branchName string) error {
	ks := sanitizeKeyspace(branchName)
	return a.session.Query(fmt.Sprintf(`DROP KEYSPACE IF EXISTS "%s"`, ks)).WithContext(ctx).Exec()
}

func (a *Adapter) CloneSchemaToBranch(ctx context.Context, sourceDB, targetDB string) error {
	if err := a.DropBranchSchema(ctx, targetDB); err != nil {
		return err
	}
	if err := a.CreateBranchSchema(ctx, targetDB); err != nil {
		return err
	}

	iter := a.session.Query(
		`SELECT table_name FROM system_schema.tables WHERE keyspace_name = ? ALLOW FILTERING`, sourceDB,
	).WithContext(ctx).Iter()
	defer iter.Close()

	var tables []string
	var t string
	for iter.Scan(&t) {
		if !strings.HasPrefix(t, "_flash_") {
			tables = append(tables, t)
		}
	}
	if err := iter.Close(); err != nil {
		return err
	}

	for _, tbl := range tables {
		q := fmt.Sprintf(`CREATE TABLE "%s"."%s" AS SELECT * FROM "%s"."%s" WHERE 1=0`,
			targetDB, tbl, sourceDB, tbl)
		if err := a.session.Query(q).WithContext(ctx).Exec(); err != nil {
			return fmt.Errorf("failed to clone table %s: %w", tbl, err)
		}
	}
	return nil
}

func (a *Adapter) GetSchemaForBranch(ctx context.Context, branchDB string) ([]types.SchemaTable, error) {
	ks := sanitizeKeyspace(branchDB)
	iter := a.session.Query(
		`SELECT table_name, column_name, type, kind FROM system_schema.columns WHERE keyspace_name = ? ALLOW FILTERING`, ks,
	).WithContext(ctx).Iter()
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
	ks := sanitizeKeyspace(schemaName)
	a.keyspace = ks
	return nil
}

func (a *Adapter) GetTableNamesInSchema(ctx context.Context, dbName string) ([]string, error) {
	ks := sanitizeKeyspace(dbName)
	iter := a.session.Query(
		`SELECT table_name FROM system_schema.tables WHERE keyspace_name = ? ALLOW FILTERING`, ks,
	).WithContext(ctx).Iter()
	defer iter.Close()

	var names []string
	var n string
	for iter.Scan(&n) {
		names = append(names, n)
	}
	return names, iter.Close()
}
