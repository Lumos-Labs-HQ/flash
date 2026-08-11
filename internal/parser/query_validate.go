package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// validateInsertColumns validates that all columns in an INSERT statement exist in the table
func (p *QueryParser) validateInsertColumns(sql string, table *Table) error {
	insertRegex := regexp.MustCompile(`(?i)INSERT\s+INTO\s+[\w"]+\s*\(([^)]+)\)\s*VALUES\s*\(([^)]+)\)`)
	matches := insertRegex.FindStringSubmatch(sql)

	if len(matches) < 3 {
		return nil
	}

	columnsStr := matches[1]
	valuesStr := matches[2]

	columnNames := strings.Split(columnsStr, ",")
	valueParams := strings.Split(valuesStr, ",")

	if len(columnNames) != len(valueParams) {
		return fmt.Errorf("column-value count mismatch: %d columns but %d values provided",
			len(columnNames), len(valueParams))
	}

	validColumns := make(map[string]bool)
	for _, col := range table.Columns {
		validColumns[strings.ToLower(col.Name)] = true
	}

	var invalidColumns []string
	for _, colName := range columnNames {
		colName = strings.TrimSpace(colName)
		colName = strings.Trim(colName, `"'`)
		colName = strings.ToLower(colName)

		if !validColumns[colName] {
			invalidColumns = append(invalidColumns, colName)
		}
	}

	if len(invalidColumns) > 0 {
		return fmt.Errorf("column(s) %v do not exist in table '%s'. Available columns: %v",
			invalidColumns, table.Name, p.getColumnNames(table))
	}

	return nil
}

// validateUpdateColumns validates that all columns in an UPDATE SET clause exist in the table
func (p *QueryParser) validateUpdateColumns(sql string, table *Table) error {
	updateRegex := regexp.MustCompile(`(?i)UPDATE\s+[\w"]+\s+SET\s+(.+?)(?:\s+WHERE|\s+RETURNING|$)`)
	matches := updateRegex.FindStringSubmatch(sql)

	if len(matches) < 2 {
		return nil
	}

	setClause := matches[1]
	assignments := p.splitSetClause(setClause)

	validColumns := make(map[string]bool)
	for _, col := range table.Columns {
		validColumns[strings.ToLower(col.Name)] = true
	}

	var invalidColumns []string
	for _, assignment := range assignments {
		parts := strings.SplitN(assignment, "=", 2)
		if len(parts) < 1 {
			continue
		}

		colName := strings.TrimSpace(parts[0])
		colName = strings.Trim(colName, `"'`)
		colName = strings.ToLower(colName)

		if !validColumns[colName] {
			invalidColumns = append(invalidColumns, colName)
		}
	}

	if len(invalidColumns) > 0 {
		return fmt.Errorf("column(s) %v do not exist in table '%s'. Available columns: %v",
			invalidColumns, table.Name, p.getColumnNames(table))
	}

	return nil
}

// splitSetClause splits SET clause by comma, respecting parentheses
func (p *QueryParser) splitSetClause(setClause string) []string {
	var result []string
	var current strings.Builder
	parenDepth := 0

	for _, char := range setClause {
		switch char {
		case '(':
			parenDepth++
			current.WriteRune(char)
		case ')':
			parenDepth--
			current.WriteRune(char)
		case ',':
			if parenDepth == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// getColumnNames returns a list of column names from a table for error messages
func (p *QueryParser) getColumnNames(table *Table) []string {
	var names []string
	for _, col := range table.Columns {
		names = append(names, col.Name)
	}
	return names
}
