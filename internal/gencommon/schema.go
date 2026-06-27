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

// ExpandWildcardColumns expands SELECT * columns. If the query has a
// column which is "*", it replaces it with all columns from the matched schema table.
func (e *SchemaExpander) ExpandWildcardColumns(query *parser.Query) []*parser.QueryColumn {
	if e == nil || len(e.Tables) == 0 || len(query.Columns) == 0 {
		return query.Columns
	}

	// Check if any column is a wildcard
	hasWildcard := false
	for _, col := range query.Columns {
		if col.Name == "*" {
			hasWildcard = true
			break
		}
	}
	if !hasWildcard {
		return query.Columns
	}

	tableName := utils.ExtractTableName(query.SQL)
	if tableName == "" {
		return query.Columns
	}
	if idx := strings.LastIndex(tableName, "."); idx >= 0 {
		tableName = tableName[idx+1:]
	}

	var matchedTable *parser.Table
	for _, t := range e.Tables {
		name := t.Name
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			name = name[idx+1:]
		}
		if strings.EqualFold(name, tableName) {
			matchedTable = t
			break
		}
	}
	if matchedTable == nil {
		return query.Columns
	}

	// Expand: replace each "*" with all table columns, keep non-wildcard columns
	expanded := make([]*parser.QueryColumn, 0, len(matchedTable.Columns)+len(query.Columns))
	for _, col := range query.Columns {
		if col.Name == "*" {
			for _, tc := range matchedTable.Columns {
				if tc.Name == "*" || strings.HasPrefix(tc.Name, "/*") {
					continue
				}
				expanded = append(expanded, &parser.QueryColumn{
					Name:     tc.Name,
					Type:     tc.Type,
					Table:    matchedTable.Name,
					Nullable: tc.Nullable,
				})
			}
		} else {
			expanded = append(expanded, col)
		}
	}
	return expanded
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
