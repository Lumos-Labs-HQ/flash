package studio

import (
	"context"
	"fmt"
	"strings"

	"github.com/Lumos-Labs-HQ/graft/internal/database"
)

type Service struct {
	adapter database.DatabaseAdapter
	ctx     context.Context
}

func NewService(adapter database.DatabaseAdapter) *Service {
	return &Service{
		adapter: adapter,
		ctx:     context.Background(),
	}
}

func (s *Service) GetTables() ([]TableInfo, error) {
	tables, err := s.adapter.GetAllTableNames(s.ctx)
	if err != nil {
		return nil, err
	}

	var result []TableInfo

	// Use goroutines for parallel count fetching (much faster)
	type tableCount struct {
		name  string
		count int
	}

	countChan := make(chan tableCount, len(tables))

	for _, table := range tables {
		if table == "_graft_migrations" {
			continue
		}

		go func(tableName string) {
			count := 0
			// Fast count using COUNT(*) query
			count, err := s.adapter.GetTableRowCount(s.ctx, tableName)
			if err != nil {
				count = 0
			}
			countChan <- tableCount{name: tableName, count: count}
		}(table)
	}

	// Collect results
	tableMap := make(map[string]int)
	for i := 0; i < len(tables)-1; i++ { // -1 for _graft_migrations
		tc := <-countChan
		tableMap[tc.name] = tc.count
	}

	// Build result in order
	for _, table := range tables {
		if table == "_graft_migrations" {
			continue
		}
		result = append(result, TableInfo{
			Name:     table,
			RowCount: tableMap[table],
		})
	}

	return result, nil
}

func (s *Service) GetTableData(tableName string, page, limit int) (*TableData, error) {
	schema, err := s.adapter.GetTableColumns(s.ctx, tableName)
	if err != nil {
		return nil, err
	}

	columns := make([]ColumnInfo, len(schema))
	for i, col := range schema {
		columns[i] = ColumnInfo{
			Name:             col.Name,
			Type:             col.Type,
			Nullable:         col.Nullable,
			PrimaryKey:       col.IsPrimary,
			Default:          col.Default,
			AutoIncrement:    col.IsAutoIncrement, // NEW: Pass auto-increment info
			ForeignKeyTable:  col.ForeignKeyTable,
			ForeignKeyColumn: col.ForeignKeyColumn,
		}
	}

	offset := (page - 1) * limit
	rows, err := s.getRows(tableName, limit, offset)
	if err != nil {
		return nil, err
	}

	total, _ := s.getRowCount(tableName)

	return &TableData{
		Columns: columns,
		Rows:    rows,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}, nil
}

func (s *Service) SaveChanges(tableName string, changes []RowChange) error {
	// Get primary key column
	schema, err := s.adapter.GetTableColumns(s.ctx, tableName)
	if err != nil {
		return err
	}

	pkColumn := "id"
	for _, col := range schema {
		if col.IsPrimary {
			pkColumn = col.Name
			break
		}
	}

	// Execute each change
	for _, change := range changes {
		if change.Action == "update" {
			// Build and execute UPDATE query
			query := fmt.Sprintf("UPDATE %s SET %s = '%s' WHERE %s = '%s'",
				tableName, change.Column, change.Value, pkColumn, change.RowID)

			if err := s.adapter.ExecuteMigration(s.ctx, query); err != nil {
				return fmt.Errorf("failed to update %s.%s: %w", tableName, change.Column, err)
			}
		}
	}

	return nil
}

func (s *Service) DeleteRows(tableName string, rowIDs []string) error {
	// Get primary key column
	schema, err := s.adapter.GetTableColumns(s.ctx, tableName)
	if err != nil {
		return err
	}

	pkColumn := "id"
	for _, col := range schema {
		if col.IsPrimary {
			pkColumn = col.Name
			break
		}
	}

	// Delete each row
	for _, rowID := range rowIDs {
		query := fmt.Sprintf("DELETE FROM %s WHERE %s = '%s'", tableName, pkColumn, rowID)
		if err := s.adapter.ExecuteMigration(s.ctx, query); err != nil {
			return fmt.Errorf("failed to delete row %s: %w", rowID, err)
		}
	}

	return nil
}

func (s *Service) AddRow(tableName string, data map[string]any) error {
	if len(data) == 0 {
		return fmt.Errorf("no data provided")
	}

	columns := []string{}
	values := []string{}

	for col, val := range data {
		columns = append(columns, col)
		// Format value with proper escaping
		if val == nil {
			values = append(values, "NULL")
		} else {
			// Escape single quotes and wrap in quotes for strings
			strVal := fmt.Sprintf("%v", val)
			escapedVal := strings.ReplaceAll(strVal, "'", "''")
			values = append(values, fmt.Sprintf("'%s'", escapedVal))
		}
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		joinColumns(columns),
		strings.Join(values, ", "))

	return s.adapter.ExecuteMigration(s.ctx, query)
}

func (s *Service) DeleteRow(tableName, rowID string) error {
	return s.deleteRow(tableName, rowID)
}

// Helper methods
func (s *Service) getRowCount(tableName string) (int, error) {
	data, err := s.adapter.GetTableData(s.ctx, tableName)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (s *Service) getRows(tableName string, limit, offset int) ([]map[string]any, error) {
	data, err := s.adapter.GetTableData(s.ctx, tableName)
	if err != nil {
		return nil, err
	}

	start := offset
	end := offset + limit
	if start > len(data) {
		return []map[string]any{}, nil
	}
	if end > len(data) {
		end = len(data)
	}

	return data[start:end], nil
}

func (s *Service) updateRow(tableName string, change RowChange) error {
	query := fmt.Sprintf("UPDATE %s SET %s = $1 WHERE id = $2",
		tableName, change.Column)
	return s.adapter.ExecuteMigration(s.ctx, query)
}

func (s *Service) deleteRow(tableName, rowID string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", tableName)
	return s.adapter.ExecuteMigration(s.ctx, query)
}

func joinColumns(cols []string) string {
	result := ""
	for i, col := range cols {
		if i > 0 {
			result += ", "
		}
		result += col
	}
	return result
}

func placeholders(n int) string {
	result := ""
	for i := 1; i <= n; i++ {
		if i > 1 {
			result += ", "
		}
		result += fmt.Sprintf("$%d", i)
	}
	return result
}

// GetSchemaVisualization returns schema for visualization
func (s *Service) GetSchemaVisualization() (map[string]any, error) {
	tables, err := s.adapter.GetCurrentSchema(s.ctx)
	if err != nil {
		return nil, err
	}

	enums, _ := s.adapter.GetCurrentEnums(s.ctx)

	// Build nodes (tables)
	nodes := []map[string]any{}
	nodeIndex := make(map[string]string)

	for i, table := range tables {
		nodeID := fmt.Sprintf("table-%d", i)
		nodeIndex[table.Name] = nodeID

		// Prepare columns info
		columns := []map[string]any{}
		for _, col := range table.Columns {
			columns = append(columns, map[string]any{
				"name":      col.Name,
				"type":      col.Type,
				"isPrimary": col.IsPrimary,
				"isForeign": col.ForeignKeyTable != "",
			})
		}

		nodes = append(nodes, map[string]any{
			"id": nodeID,
			"data": map[string]any{
				"label":   table.Name,
				"columns": columns,
			},
			"position": map[string]int{
				"x": 100 + (i%4)*300,
				"y": 100 + (i/4)*250,
			},
		})
	}

	// Build edges (relationships)
	edges := []map[string]any{}
	for _, table := range tables {
		sourceID := nodeIndex[table.Name]
		for _, col := range table.Columns {
			if col.ForeignKeyTable != "" {
				if targetID, ok := nodeIndex[col.ForeignKeyTable]; ok {
					edges = append(edges, map[string]any{
						"id":     fmt.Sprintf("%s-%s", sourceID, targetID),
						"source": sourceID,
						"target": targetID,
						"label":  col.Name,
					})
				}
			}
		}
	}

	return map[string]any{
		"nodes": nodes,
		"edges": edges,
		"enums": enums,
	}, nil
}

// ExecuteSQL executes a raw SQL query
func (s *Service) ExecuteSQL(query string) (*TableData, error) {
	query = strings.TrimSpace(query)

	// Check if it's a SELECT query or other query that returns data
	queryUpper := strings.ToUpper(query)
	isSelectQuery := strings.HasPrefix(queryUpper, "SELECT") ||
		strings.HasPrefix(queryUpper, "SHOW") ||
		strings.HasPrefix(queryUpper, "DESCRIBE") ||
		strings.HasPrefix(queryUpper, "EXPLAIN") ||
		strings.HasPrefix(queryUpper, "WITH")

	if isSelectQuery {
		// Execute as a query and return results
		result, err := s.adapter.ExecuteQuery(s.ctx, query)
		if err != nil {
			return nil, fmt.Errorf("query execution failed: %w", err)
		}

		// Convert to TableData format with ordered columns
		columns := make([]ColumnInfo, len(result.Columns))
		for i, col := range result.Columns {
			columns[i] = ColumnInfo{
				Name: col,
				Type: "TEXT", // We don't have type info from query results
			}
		}

		return &TableData{
			Columns: columns,
			Rows:    result.Rows,
			Total:   len(result.Rows),
			Page:    1,
			Limit:   len(result.Rows),
		}, nil
	} else {
		// Execute as a migration (INSERT, UPDATE, DELETE, CREATE, etc.)
		if err := s.adapter.ExecuteMigration(s.ctx, query); err != nil {
			return nil, fmt.Errorf("query execution failed: %w", err)
		}

		// Return success message
		return &TableData{
			Columns: []ColumnInfo{},
			Rows:    []map[string]any{},
			Total:   0,
			Page:    1,
			Limit:   0,
		}, nil
	}
}
