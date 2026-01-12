package parser

import (
	"strings"
	"sync"
)

// IndexedSchema wraps Schema with fast lookup indices
type IndexedSchema struct {
	*Schema
	tableIndex  map[string]*Table              // lowercase table name → table
	columnIndex map[string]map[string]*Column  // table name → (column name → column)
	mu          sync.RWMutex                   // protects indices during concurrent access
}

// NewIndexedSchema creates an indexed schema from a regular schema
func NewIndexedSchema(schema *Schema) *IndexedSchema {
	if schema == nil {
		return &IndexedSchema{
			Schema:      &Schema{Tables: []*Table{}, Enums: []*Enum{}},
			tableIndex:  make(map[string]*Table),
			columnIndex: make(map[string]map[string]*Column),
		}
	}

	idx := &IndexedSchema{
		Schema:      schema,
		tableIndex:  make(map[string]*Table, len(schema.Tables)),
		columnIndex: make(map[string]map[string]*Column, len(schema.Tables)),
	}

	// Build indices
	for _, table := range schema.Tables {
		tableKey := strings.ToLower(table.Name)
		idx.tableIndex[tableKey] = table

		// Build column index for this table
		colMap := make(map[string]*Column, len(table.Columns))
		for _, col := range table.Columns {
			colKey := strings.ToLower(col.Name)
			colMap[colKey] = col
		}
		idx.columnIndex[tableKey] = colMap
	}

	return idx
}

// GetTable performs O(1) case-insensitive table lookup
func (idx *IndexedSchema) GetTable(tableName string) *Table {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.tableIndex[strings.ToLower(tableName)]
}

// GetColumn performs O(1) case-insensitive column lookup
func (idx *IndexedSchema) GetColumn(tableName, columnName string) *Column {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	
	tableKey := strings.ToLower(tableName)
	colMap, ok := idx.columnIndex[tableKey]
	if !ok {
		return nil
	}
	return colMap[strings.ToLower(columnName)]
}

// TableExists checks if a table exists (O(1))
func (idx *IndexedSchema) TableExists(tableName string) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	_, exists := idx.tableIndex[strings.ToLower(tableName)]
	return exists
}

// ColumnExists checks if a column exists in a table (O(1))
func (idx *IndexedSchema) ColumnExists(tableName, columnName string) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	
	tableKey := strings.ToLower(tableName)
	colMap, ok := idx.columnIndex[tableKey]
	if !ok {
		return false
	}
	_, exists := colMap[strings.ToLower(columnName)]
	return exists
}

// GetTableNames returns all table names (useful for iteration)
func (idx *IndexedSchema) GetTableNames() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	
	names := make([]string, 0, len(idx.tableIndex))
	for name := range idx.tableIndex {
		names = append(names, name)
	}
	return names
}

// Refresh rebuilds the indices (call after schema modifications)
func (idx *IndexedSchema) Refresh() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Clear old indices
	idx.tableIndex = make(map[string]*Table, len(idx.Schema.Tables))
	idx.columnIndex = make(map[string]map[string]*Column, len(idx.Schema.Tables))

	// Rebuild
	for _, table := range idx.Schema.Tables {
		tableKey := strings.ToLower(table.Name)
		idx.tableIndex[tableKey] = table

		colMap := make(map[string]*Column, len(table.Columns))
		for _, col := range table.Columns {
			colKey := strings.ToLower(col.Name)
			colMap[colKey] = col
		}
		idx.columnIndex[tableKey] = colMap
	}
}
