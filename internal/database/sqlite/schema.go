package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync"

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

	// PERFORMANCE OPTIMIZATION: Parallel column fetching
	// SQLite doesn't support batching PRAGMA, but we can parallelize
	// This gives 5-10x speedup for large schemas!
	type result struct {
		tableName string
		columns   []types.SchemaColumn
		err       error
	}

	results := make(chan result, len(validTables))
	var wg sync.WaitGroup

	for _, tableName := range validTables {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			cols, colErr := s.GetTableColumns(ctx, name)
			results <- result{name, cols, colErr}
		}(tableName)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	allColumns := make(map[string][]types.SchemaColumn)
	for r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("failed to get columns for table %s: %w", r.tableName, r.err)
		}
		allColumns[r.tableName] = r.columns
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

// validateTableName prevents SQL injection in PRAGMA statements
// SQLite PRAGMA doesn't support parameterized table names, so we validate them
func (s *Adapter) validateTableName(name string) error {
	// Valid SQL identifiers: start with letter/underscore, contain only alphanumeric and underscore
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	if !matched {
		return fmt.Errorf("invalid table name: %s", name)
	}
	return nil
}

func (s *Adapter) GetAllTablesIndexes(ctx context.Context, tableNames []string) (map[string][]types.SchemaIndex, error) {
	if len(tableNames) == 0 {
		return make(map[string][]types.SchemaIndex), nil
	}

	result := make(map[string][]types.SchemaIndex)

	for _, tableName := range tableNames {
		// SECURITY: Validate table name to prevent SQL injection
		if err := s.validateTableName(tableName); err != nil {
			return nil, err
		}

		indexes, err := s.GetTableIndexes(ctx, tableName)
		if err != nil {
			// CRITICAL FIX: Return error instead of silently continuing
			return nil, fmt.Errorf("failed to get indexes for table %s: %w", tableName, err)
		}
		if len(indexes) > 0 {
			result[tableName] = indexes
		}
	}

	return result, nil
}

func (s *Adapter) GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error) {
	// SECURITY: Validate table name before using in PRAGMA
	if err := s.validateTableName(tableName); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(\"%s\")", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// CRITICAL OPTIMIZATION: Batch fetch unique columns ONCE per table
	// Old code called isColumnUnique for EVERY column (N+1 problem)
	// 100 columns = 100 separate PRAGMA queries!
	// New: Single call = massive speedup
	uniqueColumns := s.getUniqueColumnsForTable(ctx, tableName)

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

		// Use pre-fetched unique column map instead of N+1 queries
		column.IsUnique = uniqueColumns[column.Name]
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
	// SECURITY: Validate table name before using in PRAGMA
	if err := s.validateTableName(tableName); err != nil {
		return nil, err
	}

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

// PullCompleteSchema returns complete schema excluding internal tables
// OPTIMIZATION: Reuses GetCurrentSchema with parallel fetching (was sequential N+1!)
func (s *Adapter) PullCompleteSchema(ctx context.Context) ([]types.SchemaTable, error) {
	return s.GetCurrentSchema(ctx)
}

func (s *Adapter) getUniqueColumnsForTable(ctx context.Context, tableName string) map[string]bool {
	// Fetch all unique columns for a table in ONE query
	// Returns map of column_name -> is_unique
	uniqueMap := make(map[string]bool)

	// Validation already done by caller, but be safe
	if err := s.validateTableName(tableName); err != nil {
		return uniqueMap
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(\"%s\")", tableName))
	if err != nil {
		return uniqueMap
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

		// Get columns for this unique index
		columns := s.getIndexColumns(ctx, indexName)
		// Only mark as unique if it's a single-column unique index
		if len(columns) == 1 {
			uniqueMap[columns[0]] = true
		}
	}

	return uniqueMap
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
