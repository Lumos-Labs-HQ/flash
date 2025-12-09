package schema

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

// SQL Parsing helpers
func (sm *SchemaManager) cleanSQL(sql string) string {
	commentRegex := regexp.MustCompile(`--.*|/\*[\s\S]*?\*/`)
	sql = commentRegex.ReplaceAllString(sql, "")

	whitespaceRegex := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(whitespaceRegex.ReplaceAllString(sql, " "))
}

func (sm *SchemaManager) splitStatements(sql string) []string {
	statements := strings.Split(sql, ";")
	result := make([]string, 0, len(statements))

	for _, stmt := range statements {
		if stmt = strings.TrimSpace(stmt); stmt != "" {
			result = append(result, stmt)
		}
	}
	return result
}

func (sm *SchemaManager) isCreateTableStatement(stmt string) bool {
	matched, _ := regexp.MatchString(`(?i)^\s*CREATE\s+TABLE`, stmt)
	return matched
}

func (sm *SchemaManager) isCreateIndexStatement(stmt string) bool {
	matched, _ := regexp.MatchString(`(?i)^\s*CREATE\s+(UNIQUE\s+)?INDEX`, stmt)
	return matched
}

func (sm *SchemaManager) parseCreateIndexStatement(stmt string) (types.SchemaIndex, error) {
	indexRegex := regexp.MustCompile(`(?i)CREATE\s+(UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:"?(\w+)"?|(\w+))\s+ON\s+(?:"?(\w+)"?|(\w+))\s*\(\s*([^)]+)\s*\)`)
	matches := indexRegex.FindStringSubmatch(stmt)

	if len(matches) < 7 {
		return types.SchemaIndex{}, fmt.Errorf("could not parse CREATE INDEX statement: %s", stmt)
	}

	isUnique := strings.TrimSpace(matches[1]) != ""

	// Extract index name (could be in matches[2] or matches[3])
	indexName := matches[2]
	if indexName == "" {
		indexName = matches[3]
	}

	// Extract table name (could be in matches[4] or matches[5])
	tableName := matches[4]
	if tableName == "" {
		tableName = matches[5]
	}

	// Extract columns
	columnsStr := matches[6]
	columnParts := strings.Split(columnsStr, ",")
	var columns []string
	for _, col := range columnParts {
		// Clean up column name (remove quotes, ASC/DESC, etc.)
		col = strings.TrimSpace(col)
		col = strings.Trim(col, `"'`)
		col = regexp.MustCompile(`(?i)\s+(ASC|DESC)$`).ReplaceAllString(col, "")
		col = strings.TrimSpace(col)
		if col != "" {
			columns = append(columns, col)
		}
	}

	return types.SchemaIndex{
		Name:    indexName,
		Table:   tableName,
		Columns: columns,
		Unique:  isUnique,
	}, nil
}

func (sm *SchemaManager) isCreateTypeStatement(stmt string) bool {
	matched, _ := regexp.MatchString(`(?i)^\s*CREATE\s+TYPE\s+\w+\s+AS\s+ENUM`, stmt)
	return matched
}

func (sm *SchemaManager) parseCreateTypeStatement(stmt string) (types.SchemaEnum, error) {
	// Match: CREATE TYPE enum_name AS ENUM ('value1', 'value2', ...)
	enumRegex := regexp.MustCompile(`(?i)CREATE\s+TYPE\s+(?:"?(\w+)"?|(\w+))\s+AS\s+ENUM\s*\(\s*([^)]+)\s*\)`)
	matches := enumRegex.FindStringSubmatch(stmt)

	if len(matches) < 4 {
		return types.SchemaEnum{}, fmt.Errorf("could not parse CREATE TYPE statement: %s", stmt)
	}

	// Extract enum name
	enumName := matches[1]
	if enumName == "" {
		enumName = matches[2]
	}

	// Extract values
	valuesStr := matches[3]
	valueRegex := regexp.MustCompile(`'([^']+)'`)
	valueMatches := valueRegex.FindAllStringSubmatch(valuesStr, -1)

	values := make([]string, 0, len(valueMatches))
	for _, match := range valueMatches {
		if len(match) > 1 {
			values = append(values, match[1])
		}
	}

	return types.SchemaEnum{
		Name:   enumName,
		Values: values,
	}, nil
}

func (sm *SchemaManager) parseCreateTableStatement(stmt string) (types.SchemaTable, error) {
	tableRegex := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:"?(\w+)"?|(\w+)|\x60(\w+)\x60)\s*\(`)
	matches := tableRegex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return types.SchemaTable{}, fmt.Errorf("could not extract table name from: %s", stmt)
	}

	tableName := sm.extractTableName(matches)
	if tableName == "" {
		return types.SchemaTable{}, fmt.Errorf("could not extract table name")
	}

	start, end := strings.Index(stmt, "("), strings.LastIndex(stmt, ")")
	if start == -1 || end == -1 {
		return types.SchemaTable{}, fmt.Errorf("invalid CREATE TABLE syntax")
	}

	columns, foreignKeys, err := sm.parseColumnDefinitionsAndConstraints(stmt[start+1 : end])
	if err != nil {
		return types.SchemaTable{}, err
	}

	sm.applyForeignKeys(columns, foreignKeys)

	return types.SchemaTable{
		Name:    tableName,
		Columns: columns,
		Indexes: []types.SchemaIndex{},
	}, nil
}

func (sm *SchemaManager) extractTableName(matches []string) string {
	for i := 1; i < len(matches); i++ {
		if matches[i] != "" {
			return matches[i]
		}
	}
	return ""
}

func (sm *SchemaManager) applyForeignKeys(columns []types.SchemaColumn, foreignKeys []foreignKeyConstraint) {
	for _, fk := range foreignKeys {
		for i := range columns {
			if columns[i].Name == fk.ColumnName {
				columns[i].ForeignKeyTable = fk.ReferencedTable
				columns[i].ForeignKeyColumn = fk.ReferencedColumn
				columns[i].OnDeleteAction = fk.OnDeleteAction
				break
			}
		}
	}
}

func (sm *SchemaManager) parseColumnDefinitionsAndConstraints(columnDefs string) ([]types.SchemaColumn, []foreignKeyConstraint, error) {
	var columns []types.SchemaColumn
	var foreignKeys []foreignKeyConstraint

	for _, colDef := range sm.splitColumnDefinitions(columnDefs) {
		if colDef = strings.TrimSpace(colDef); colDef == "" {
			continue
		}

		if sm.isTableConstraint(colDef) {
			if fk := sm.parseForeignKeyConstraint(colDef); fk != nil {
				foreignKeys = append(foreignKeys, *fk)
			}
			continue
		}

		column, err := sm.parseColumnDefinition(colDef)
		if err != nil {
			return nil, nil, err
		}
		columns = append(columns, column)
	}

	return columns, foreignKeys, nil
}

func (sm *SchemaManager) parseForeignKeyConstraint(constraint string) *foreignKeyConstraint {
	fkRegex := regexp.MustCompile(`(?i)FOREIGN\s+KEY\s*\(\s*(\w+)\s*\)\s+REFERENCES\s+(\w+)\s*\(\s*(\w+)\s*\)(?:\s+ON\s+DELETE\s+(CASCADE|SET\s+NULL|RESTRICT|NO\s+ACTION))?`)
	matches := fkRegex.FindStringSubmatch(constraint)

	if len(matches) >= 4 {
		fk := &foreignKeyConstraint{
			ColumnName:       matches[1],
			ReferencedTable:  matches[2],
			ReferencedColumn: matches[3],
		}
		if len(matches) >= 5 && matches[4] != "" {
			fk.OnDeleteAction = strings.ToUpper(matches[4])
		}
		return fk
	}
	return nil
}

func (sm *SchemaManager) splitColumnDefinitions(defs string) []string {
	var result []string
	var current strings.Builder
	parenLevel := 0

	for _, char := range defs {
		switch char {
		case '(':
			parenLevel++
			current.WriteRune(char)
		case ')':
			parenLevel--
			current.WriteRune(char)
		case ',':
			if parenLevel == 0 {
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

func (sm *SchemaManager) isTableConstraint(def string) bool {
	def = strings.ToUpper(strings.TrimSpace(def))
	prefixes := []string{"PRIMARY KEY", "FOREIGN KEY", "UNIQUE", "CHECK", "CONSTRAINT"}

	for _, prefix := range prefixes {
		if strings.HasPrefix(def, prefix) {
			return true
		}
	}
	return false
}

func (sm *SchemaManager) parseColumnDefinition(colDef string) (types.SchemaColumn, error) {
	colDef = strings.TrimSpace(colDef)

	// Extract column name (first token)
	spaceIdx := strings.IndexAny(colDef, " \t")
	if spaceIdx == -1 {
		return types.SchemaColumn{}, fmt.Errorf("invalid column definition: %s", colDef)
	}

	colName := strings.Trim(colDef[:spaceIdx], `"`)
	rest := strings.TrimSpace(colDef[spaceIdx+1:])

	if rest == "" {
		return types.SchemaColumn{}, fmt.Errorf("invalid column definition (no type): %s", colDef)
	}

	column := types.SchemaColumn{
		Name:     colName,
		Nullable: true,
	}

	// Extract type - handle parentheses for types like DECIMAL(10, 2)
	restUpper := strings.ToUpper(rest)

	// Handle multi-word types first
	if strings.HasPrefix(restUpper, "TIMESTAMP WITH TIME ZONE") {
		column.Type = "TIMESTAMP WITH TIME ZONE"
	} else if strings.HasPrefix(restUpper, "TIMESTAMP WITHOUT TIME ZONE") {
		column.Type = "TIMESTAMP WITHOUT TIME ZONE"
	} else if strings.HasPrefix(restUpper, "DOUBLE PRECISION") {
		column.Type = "DOUBLE PRECISION"
	} else if strings.HasPrefix(restUpper, "CHARACTER VARYING") {
		column.Type = "CHARACTER VARYING"
	} else {
		// Extract type including parentheses content
		parenDepth := 0
		typeEnd := 0
		for i, ch := range rest {
			if ch == '(' {
				parenDepth++
			} else if ch == ')' {
				parenDepth--
				if parenDepth == 0 {
					typeEnd = i + 1
					break
				}
			} else if parenDepth == 0 && (ch == ' ' || ch == '\t') {
				typeEnd = i
				break
			}
		}

		if typeEnd == 0 {
			typeEnd = len(rest)
		}

		column.Type = rest[:typeEnd]
	}

	sm.parseColumnConstraints(&column, colDef)
	return column, nil
}

func (sm *SchemaManager) parseColumnConstraints(column *types.SchemaColumn, colDef string) {
	defUpper := strings.ToUpper(colDef)

	constraints := map[string]func(){
		"NOT NULL":       func() { column.Nullable = false },
		"PRIMARY KEY":    func() { column.IsPrimary = true },
		"UNIQUE":         func() { column.IsUnique = true },
		"AUTOINCREMENT":  func() { column.IsPrimary = true },
		"AUTO_INCREMENT": func() { column.IsPrimary = true },
		"SERIAL":         func() { column.IsPrimary = true },
		"BIGSERIAL":      func() { column.IsPrimary = true },
		"SMALLSERIAL":    func() { column.IsPrimary = true },
	}

	for constraint, action := range constraints {
		if strings.Contains(defUpper, constraint) {
			action()
		}
	}

	referencesRegex := regexp.MustCompile(`(?i)REFERENCES\s+(\w+)\s*\(\s*(\w+)\s*\)`)
	if matches := referencesRegex.FindStringSubmatch(colDef); len(matches) >= 3 {
		column.ForeignKeyTable = matches[1]
		column.ForeignKeyColumn = matches[2]

		onDeleteRegex := regexp.MustCompile(`(?i)ON\s+DELETE\s+(CASCADE|SET\s+NULL|RESTRICT|NO\s+ACTION)`)
		if onDeleteMatches := onDeleteRegex.FindStringSubmatch(colDef); len(onDeleteMatches) >= 2 {
			column.OnDeleteAction = strings.ToUpper(onDeleteMatches[1])
		}
	}

	defaultRegex := regexp.MustCompile(`(?i)\bDEFAULT\s+([^,\s]+|'[^']*'|\([^)]*\))`)
	if matches := defaultRegex.FindStringSubmatch(colDef); len(matches) > 1 {
		column.Default = matches[1]
	}
}
