package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func (s *Adapter) GetCurrentSchema(ctx context.Context) ([]types.SchemaTable, error) {
	tableNames, err := s.GetAllTableNames(ctx)
	if err != nil {
		return nil, err
	}

	var validTables []string
	for _, name := range tableNames {
		if name != "_flash_migrations" {
			validTables = append(validTables, name)
		}
	}

	if len(validTables) == 0 {
		return []types.SchemaTable{}, nil
	}

	allColumns := make(map[string][]types.SchemaColumn)
	for _, name := range validTables {
		columns, err := s.GetTableColumns(ctx, name)
		if err != nil {
			return nil, err
		}
		allColumns[name] = columns
	}

	allIndexes, err := s.GetAllTablesIndexes(ctx, validTables)
	if err != nil {
		return nil, err
	}

	var tables []types.SchemaTable
	for _, name := range validTables {
		tables = append(tables, types.SchemaTable{
			Name:    name,
			Columns: allColumns[name],
			Indexes: allIndexes[name],
		})
	}
	return tables, nil
}

func (s *Adapter) GetCurrentEnums(ctx context.Context) ([]types.SchemaEnum, error) {
	return []types.SchemaEnum{}, nil
}

func (s *Adapter) GetAllTablesIndexes(ctx context.Context, tableNames []string) (map[string][]types.SchemaIndex, error) {
	if len(tableNames) == 0 {
		return make(map[string][]types.SchemaIndex), nil
	}

	result := make(map[string][]types.SchemaIndex)

	for _, tableName := range tableNames {
		indexes, err := s.GetTableIndexes(ctx, tableName)
		if err != nil {
			continue
		}
		if len(indexes) > 0 {
			result[tableName] = indexes
		}
	}

	return result, nil
}

func (s *Adapter) GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(\"%s\")", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []types.SchemaColumn
	for rows.Next() {
		var cid int
		var column types.SchemaColumn
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int

		err := rows.Scan(&cid, &column.Name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}

		column.Type = s.MapColumnType(dataType)
		column.Nullable = notNull == 0
		column.IsPrimary = pk > 0
		column.IsAutoIncrement = pk > 0 && strings.ToUpper(dataType) == "INTEGER"

		if defaultValue.Valid {
			column.Default = defaultValue.String
		}

		column.IsUnique, _ = s.isColumnUnique(ctx, tableName, column.Name)
		columns = append(columns, column)
	}

	fkRows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA foreign_key_list(\"%s\")", tableName))
	if err == nil {
		defer fkRows.Close()

		for fkRows.Next() {
			var id, seq int
			var table, from, to, onUpdate, onDelete, match string

			err := fkRows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match)
			if err != nil {
				continue
			}

			for i := range columns {
				if columns[i].Name == from {
					columns[i].ForeignKeyTable = table
					columns[i].ForeignKeyColumn = to
					columns[i].OnDeleteAction = onDelete
					break
				}
			}
		}
	}

	return columns, nil
}

func (s *Adapter) GetTableIndexes(ctx context.Context, tableName string) ([]types.SchemaIndex, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(\"%s\")", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []types.SchemaIndex
	for rows.Next() {
		var seq int
		var indexName string
		var unique int
		var origin, partial string

		err := rows.Scan(&seq, &indexName, &unique, &origin, &partial)
		if err != nil || origin == "pk" {
			continue
		}

		columns := s.getIndexColumns(ctx, indexName)
		if len(columns) > 0 {
			indexes = append(indexes, types.SchemaIndex{
				Name:    indexName,
				Table:   tableName,
				Columns: columns,
				Unique:  unique == 1,
			})
		}
	}
	return indexes, nil
}

func (s *Adapter) GetAllTableNames(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err == nil {
			tables = append(tables, tableName)
		}
	}
	return tables, nil
}

func (s *Adapter) PullCompleteSchema(ctx context.Context) ([]types.SchemaTable, error) {
	tableQuery := `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE '_flash_%' AND name NOT LIKE 'sqlite_%'`
	rows, err := s.db.QueryContext(ctx, tableQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tableNames = append(tableNames, tableName)
	}

	var tables []types.SchemaTable
	for _, tableName := range tableNames {
		columnQuery := fmt.Sprintf("PRAGMA table_info(\"%s\")", tableName)
		columnRows, err := s.db.QueryContext(ctx, columnQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to query columns for table %s: %w", tableName, err)
		}

		var columns []types.SchemaColumn
		for columnRows.Next() {
			var cid int
			var name, dataType string
			var notNull, pk int
			var defaultValue sql.NullString

			err := columnRows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
			if err != nil {
				columnRows.Close()
				return nil, fmt.Errorf("failed to scan column info: %w", err)
			}

			column := types.SchemaColumn{
				Name:      name,
				Type:      s.formatSQLiteType(dataType),
				Nullable:  notNull == 0,
				IsPrimary: pk == 1,
				Default:   s.formatSQLiteDefault(defaultValue.String),
			}

			columns = append(columns, column)
		}
		columnRows.Close()

		fkQuery := fmt.Sprintf("PRAGMA foreign_key_list(\"%s\")", tableName)
		fkRows, err := s.db.QueryContext(ctx, fkQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to query foreign keys for table %s: %w", tableName, err)
		}

		for fkRows.Next() {
			var id, seq int
			var table, from, to, onUpdate, onDelete, match string

			err := fkRows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match)
			if err != nil {
				fkRows.Close()
				return nil, fmt.Errorf("failed to scan foreign key info: %w", err)
			}

			for i := range columns {
				if columns[i].Name == from {
					columns[i].ForeignKeyTable = table
					columns[i].ForeignKeyColumn = to
					columns[i].OnDeleteAction = onDelete
					break
				}
			}
		}
		fkRows.Close()

		tables = append(tables, types.SchemaTable{
			Name:    tableName,
			Columns: columns,
		})
	}

	return tables, nil
}

func (s *Adapter) isColumnUnique(ctx context.Context, tableName, columnName string) (bool, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(\"%s\")", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var seq int
		var indexName string
		var unique int
		var origin, partial string

		err := rows.Scan(&seq, &indexName, &unique, &origin, &partial)
		if err != nil || unique == 0 {
			continue
		}

		columns := s.getIndexColumns(ctx, indexName)
		if len(columns) == 1 && columns[0] == columnName {
			return true, nil
		}
	}
	return false, nil
}

func (s *Adapter) getIndexColumns(ctx context.Context, indexName string) []string {
	colRows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_info(\"%s\")", indexName))
	if err != nil {
		return nil
	}
	defer colRows.Close()

	var columns []string
	for colRows.Next() {
		var seqno, cid int
		var name string
		if err := colRows.Scan(&seqno, &cid, &name); err == nil {
			columns = append(columns, name)
		}
	}
	return columns
}

func (s *Adapter) formatSQLiteType(dataType string) string {
	switch strings.ToUpper(dataType) {
	case "INTEGER":
		return "INTEGER"
	case "TEXT":
		return "TEXT"
	case "REAL":
		return "REAL"
	case "BLOB":
		return "BLOB"
	case "NUMERIC":
		return "NUMERIC"
	default:
		return strings.ToUpper(dataType)
	}
}

func (s *Adapter) formatSQLiteDefault(defaultValue string) string {
	if defaultValue == "" {
		return ""
	}

	if strings.Contains(strings.ToLower(defaultValue), "current_timestamp") {
		return "CURRENT_TIMESTAMP"
	}

	return defaultValue
}
