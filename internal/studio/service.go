package studio

import (
	"context"
	"fmt"
	"strings"

	"github.com/Rana718/Graft/internal/database"
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
	for _, table := range tables {
		if table == "_graft_migrations" {
			continue
		}

		// Get count asynchronously for better performance
		count := 0
		data, err := s.adapter.GetTableData(s.ctx, table)
		if err == nil {
			count = len(data)
		}
		
		result = append(result, TableInfo{
			Name:     table,
			RowCount: count,
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
			Name:       col.Name,
			Type:       col.Type,
			Nullable:   col.Nullable,
			PrimaryKey: col.IsPrimary,
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
	columns := []string{}
	values := []any{}

	for col, val := range data {
		columns = append(columns, col)
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		joinColumns(columns),
		placeholders(len(values)))

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

func (s *Service) GetSchemaVisualization() (map[string]interface{}, error) {
	tables, err := s.adapter.GetAllTableNames(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get table names: %w", err)
	}

	nodes := []map[string]interface{}{}
	edges := []map[string]interface{}{}
	
	edgeId := 0
	tableIndex := 0

	for _, tableName := range tables {
		if tableName == "_graft_migrations" || tableName == "graft_migrations" {
			continue
		}

		columns, err := s.adapter.GetTableColumns(s.ctx, tableName)
		if err != nil {
			fmt.Printf("Error getting columns for table %s: %v\n", tableName, err)
			continue
		}

		columnData := []map[string]interface{}{}
		for _, col := range columns {
			columnData = append(columnData, map[string]interface{}{
				"name":      col.Name,
				"type":      col.Type,
				"isPrimary": col.IsPrimary,
				"isForeign": col.ForeignKeyTable != "",
			})

			if col.ForeignKeyTable != "" {
				edgeId++
				edges = append(edges, map[string]interface{}{
					"id":     fmt.Sprintf("e%d", edgeId),
					"source": tableName,
					"target": col.ForeignKeyTable,
					"label":  col.Name,
				})
			}
		}

		// Better positioning: 2 columns, more spacing
		col := tableIndex % 2
		row := tableIndex / 2
		posX := 150 + (col * 550)
		posY := 100 + (row * 400)

		nodes = append(nodes, map[string]interface{}{
			"id":   tableName,
			"type": "table",
			"data": map[string]interface{}{
				"label":   tableName,
				"columns": columnData,
			},
			"position": map[string]interface{}{
				"x": posX,
				"y": posY,
			},
		})
		
		tableIndex++
	}

	return map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	}, nil
}

func (s *Service) ExecuteSQL(query string) (*TableData, error) {
	// Basic SQL injection protection - only allow SELECT
	query = strings.TrimSpace(query)
	if !strings.HasPrefix(strings.ToUpper(query), "SELECT") {
		return nil, fmt.Errorf("only SELECT queries are allowed")
	}

	// Execute raw query - we'll use ExecuteMigration as a workaround
	// In production, you'd want a proper Query method in the adapter interface
	
	// For now, return a simple message
	// TODO: Implement proper raw query execution in database adapters
	return nil, fmt.Errorf("SQL query execution not yet implemented - coming soon!")
}
