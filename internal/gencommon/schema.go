package gencommon

import (
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/parser"
	"github.com/Lumos-Labs-HQ/flash/internal/utils"
)

// SchemaExpander provides schema-based query column expansion and model type detection.
type SchemaExpander struct {
	Tables []*parser.Table
}

// NewSchemaExpander creates a SchemaExpander from a parsed schema.
func NewSchemaExpander(schema *parser.Schema) *SchemaExpander {
	if schema == nil {
		return &SchemaExpander{}
	}
	return &SchemaExpander{Tables: schema.Tables}
}

// ExpandWildcardColumns expands SELECT * columns. If the query has only one
// column which is "*", it replaces it with all columns from the matched schema table.
func (e *SchemaExpander) ExpandWildcardColumns(query *parser.Query) []*parser.QueryColumn {
	// Multiple explicitly listed columns — no expansion needed
	if len(query.Columns) > 1 {
		return query.Columns
	}
	// Single non-wildcard column — no expansion
	if len(query.Columns) == 1 && query.Columns[0].Name != "*" {
		return query.Columns
	}
	// No tables in schema — can't expand
	if e == nil || len(e.Tables) == 0 {
		return query.Columns
	}

	tableName := utils.ExtractTableName(query.SQL)
	if tableName == "" {
		return query.Columns
	}
	// Strip keyspace prefix for lookup
	if idx := strings.LastIndex(tableName, "."); idx >= 0 {
		tableName = tableName[idx+1:]
	}

	for _, t := range e.Tables {
		name := t.Name
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			name = name[idx+1:]
		}
		if !strings.EqualFold(name, tableName) {
			continue
		}
		expanded := make([]*parser.QueryColumn, 0, len(t.Columns))
		for _, col := range t.Columns {
			if col.Name == "*" || strings.HasPrefix(col.Name, "/*") {
				continue
			}
			expanded = append(expanded, &parser.QueryColumn{
				Name:     col.Name,
				Type:     col.Type,
				Table:    t.Name,
				Nullable: col.Nullable,
			})
		}
		return expanded
	}
	return query.Columns
}

// ModelTypeForQuery returns the model struct name if the query columns exactly
// match a table in the schema — enabling reuse of the model type instead of
// generating a duplicate row struct.
func (e *SchemaExpander) ModelTypeForQuery(query *parser.Query, columns []*parser.QueryColumn) string {
	if e == nil || len(columns) == 0 || len(e.Tables) == 0 {
		return ""
	}
	tableName := utils.ExtractTableName(query.SQL)
	if tableName == "" {
		return ""
	}
	// Strip keyspace prefix for lookup
	if idx := strings.LastIndex(tableName, "."); idx >= 0 {
		tableName = tableName[idx+1:]
	}

	for _, t := range e.Tables {
		name := t.Name
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			name = name[idx+1:]
		}
		if !strings.EqualFold(name, tableName) || len(t.Columns) != len(columns) {
			continue
		}
		match := true
		for i, col := range columns {
			if !strings.EqualFold(col.Name, t.Columns[i].Name) {
				match = false
				break
			}
		}
		if match {
			return utils.ToPascalCase(tableName)
		}
	}
	return ""
}
