package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// SchemaColumn represents a column that can be validated
type SchemaColumn interface {
	GetName() string
}

// SchemaTable represents a table that can be validated
type SchemaTable interface {
	GetName() string
	GetColumns() []SchemaColumn
}

// Schema represents a schema that can be validated
type Schema interface {
	GetTables() []SchemaTable
}

// SimpleColumn is a wrapper for basic column validation
type SimpleColumn struct {
	Name string
}

func (c SimpleColumn) GetName() string { return c.Name }

// SimpleTable is a wrapper for basic table validation
type SimpleTable struct {
	Name    string
	Columns []SimpleColumn
}

func (t SimpleTable) GetName() string { return t.Name }
func (t SimpleTable) GetColumns() []SchemaColumn {
	cols := make([]SchemaColumn, len(t.Columns))
	for i, c := range t.Columns {
		cols[i] = c
	}
	return cols
}

var (
	ctePatternRegex            *regexp.Regexp
	tablePatternRegex          *regexp.Regexp
	tableAliasPatternRegex     *regexp.Regexp
	joinPatternRegex           *regexp.Regexp
	columnRefPatternRegex      *regexp.Regexp
	aliasExtractPatternRegex   *regexp.Regexp
	fromPatternRegex           *regexp.Regexp
	insertPatternRegex         *regexp.Regexp
	joinCheckRegex             *regexp.Regexp
	whereClauseRegex           *regexp.Regexp
	setClauseRegex             *regexp.Regexp
	orderByClauseRegex         *regexp.Regexp
	groupByClauseRegex         *regexp.Regexp
	havingClauseRegex          *regexp.Regexp
	paramCheckRegex            *regexp.Regexp
	unqualifiedColPatternRegex *regexp.Regexp
	selectAliasRegex           *regexp.Regexp
)

func init() {
	ctePatternRegex = regexp.MustCompile(`(?i)(\w+)\s+AS\s*\(`)
	tablePatternRegex = regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+(\w+)`)
	tableAliasPatternRegex = regexp.MustCompile(`(?i)FROM\s+(\w+)\s+(\w+)`)
	joinPatternRegex = regexp.MustCompile(`(?i)JOIN\s+(\w+)\s+(\w+)`)
	columnRefPatternRegex = regexp.MustCompile(`(?i)(\w+)\.(\w+)`)
	aliasExtractPatternRegex = regexp.MustCompile(`(?i)(?:FROM|JOIN)\s+(\w+)(?:\s+(?:AS\s+)?(\w+))?`)
	fromPatternRegex = regexp.MustCompile(`(?i)\bFROM\s+(\w+)`)
	insertPatternRegex = regexp.MustCompile(`(?i)\b(?:INSERT\s+INTO|UPDATE)\s+(\w+)`)
	joinCheckRegex = regexp.MustCompile(`(?i)\bJOIN\b`)
	whereClauseRegex = regexp.MustCompile(`(?i)\bWHERE\s+(.*?)(?:\s+(?:LIMIT|ORDER|GROUP|HAVING|;|$))`)
	setClauseRegex = regexp.MustCompile(`(?i)\bSET\s+(.*?)(?:\s+(?:WHERE|;|$))`)
	orderByClauseRegex = regexp.MustCompile(`(?i)\bORDER\s+BY\s+(.*?)(?:\s+(?:LIMIT|;|$))`)
	groupByClauseRegex = regexp.MustCompile(`(?i)\bGROUP\s+BY\s+(.*?)(?:\s+(?:HAVING|ORDER|LIMIT|;|$))`)
	havingClauseRegex = regexp.MustCompile(`(?i)\bHAVING\s+(.*?)(?:\s+(?:ORDER|LIMIT|;|$))`)
	paramCheckRegex = regexp.MustCompile(`^\d+$|^\$\d+$|\?`)
	unqualifiedColPatternRegex = regexp.MustCompile(`\b(\w+)\b`)
	selectAliasRegex = regexp.MustCompile(`(?i)\s+AS\s+(\w+)`)
}

// ValidateTableReferences checks if tables referenced in queries exist in the schema
// Uses type assertion for performance instead of reflection
func ValidateTableReferences(sql string, schema interface{}, sourceFile string) error {
	if schema == nil {
		return nil
	}

	if sourceFile == "" {
		sourceFile = "queries"
	}

	// Try to extract table names using type assertion
	tableNames := extractTableNamesFromSchema(schema)
	if tableNames == nil {
		return nil // Cannot extract, skip validation
	}

	cteNames := make(map[string]bool, 4)
	cteMatches := ctePatternRegex.FindAllStringSubmatch(sql, -1)
	for _, match := range cteMatches {
		if len(match) > 1 {
			cteName := match[1]
			if !IsSQLKeyword(cteName) {
				cteNames[strings.ToLower(cteName)] = true
			}
		}
	}

	matches := tablePatternRegex.FindAllStringSubmatch(sql, -1)

	foundTableRefs := false

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		tableName := match[1]

		if IsSQLKeyword(tableName) {
			continue
		}

		if cteNames[strings.ToLower(tableName)] {
			continue
		}

		foundTableRefs = true

		// Check if table exists in schema
		tableExists := tableNames[strings.ToLower(tableName)]

		if !tableExists {
			lines := strings.Split(sql, "\n")
			lineNum := 1
			colPos := 1
			upperTable := strings.ToUpper(tableName)

			for i, line := range lines {
				upperLine := strings.ToUpper(line)
				if strings.Contains(upperLine, upperTable) {
					lineNum = i + 1
					colPos = strings.Index(upperLine, upperTable) + 1
					break
				}
			}

			return fmt.Errorf("# package flash\ndb\\queries\\%s.sql:%d:%d: relation \"%s\" does not exist", sourceFile, lineNum, colPos, tableName)
		}
	}

	if foundTableRefs && len(tableNames) == 0 {
		return fmt.Errorf("# package flash\ndb\\queries\\%s.sql:1:1: no tables found in schema, but query references tables", sourceFile)
	}

	return nil
}

// stripOverClauses removes all OVER(...) blocks from SQL to prevent
// ORDER BY inside window functions from being matched by clause-level regexes.
func stripOverClauses(sql string) string {
	sqlUpper := strings.ToUpper(sql)
	var result strings.Builder
	result.Grow(len(sql))
	i := 0
	for i < len(sql) {
		// Look for OVER with optional whitespace, then '('
		if i+4 <= len(sql) && sqlUpper[i:i+4] == "OVER" {
			next := i + 4
			for next < len(sql) && (sql[next] == ' ' || sql[next] == '\t' || sql[next] == '\n') {
				next++
			}
			if next < len(sql) && sql[next] == '(' {
				// Found OVER (...), skip the parenthesized content
				result.WriteString(sql[i:next])
				depth := 0
				for next < len(sql) {
					if sql[next] == '(' {
						depth++
					} else if sql[next] == ')' {
						depth--
						if depth == 0 {
							result.WriteByte(')')
							next++
							break
						}
					}
					next++
				}
				i = next
				continue
			}
		}
		result.WriteByte(sql[i])
		i++
	}
	return result.String()
}

// stripSubqueryBlocks replaces parenthesized subquery blocks — typically
// (SELECT ... FROM ... WHERE ...) — with empty parens so clause-level
// regexes don't match keywords (WHERE, ORDER BY, etc.) inside subqueries.
func stripSubqueryBlocks(sql string) string {
	sqlUpper := strings.ToUpper(sql)
	var result strings.Builder
	result.Grow(len(sql))
	i := 0
	for i < len(sql) {
		if sql[i] == '(' {
			after := i + 1
			for after < len(sql) && (sql[after] == ' ' || sql[after] == '\t' || sql[after] == '\n') {
				after++
			}
			isSubQuery := false
			if after+6 <= len(sql) && sqlUpper[after:after+6] == "SELECT" &&
				(after+6 >= len(sql) || !isAlphaNum(sql[after+6])) {
				isSubQuery = true
			}
			if !isSubQuery && after+4 <= len(sql) && sqlUpper[after:after+4] == "WITH" &&
				(after+4 >= len(sql) || !isAlphaNum(sql[after+4])) {
				isSubQuery = true
			}
			if isSubQuery {
				result.WriteString("(...)")
				depth := 1
				i++
				for i < len(sql) && depth > 0 {
					if sql[i] == '(' {
						depth++
					} else if sql[i] == ')' {
						depth--
					}
					i++
				}
				continue
			}
		}
		result.WriteByte(sql[i])
		i++
	}
	return result.String()
}

// stripStringLiterals removes single-quoted SQL literals so their contents
// don't get falsely flagged as column references.
func stripStringLiterals(sql string) string {
	var result strings.Builder
	result.Grow(len(sql))
	inString := false
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if ch == '\'' {
			if inString && i+1 < len(sql) && sql[i+1] == '\'' {
				// Escaped quote
				result.WriteString("''")
				i++
				continue
			}
			inString = !inString
			result.WriteString("''") // placeholder
			continue
		}
		if !inString {
			result.WriteByte(ch)
		}
	}
	return result.String()
}

// extractTableNamesFromSchema extracts table names from various schema types
func extractTableNamesFromSchema(schema interface{}) map[string]bool {
	// Try known interface types first (fast path)
	if s, ok := schema.(interface {
		GetTables() []interface{ GetName() string }
	}); ok {
		tables := s.GetTables()
		names := make(map[string]bool, len(tables))
		for _, t := range tables {
			names[strings.ToLower(t.GetName())] = true
		}
		return names
	}

	// Use type switch for known patterns
	switch s := schema.(type) {
	case interface{ GetTableNames() map[string]bool }:
		return s.GetTableNames()
	default:
		// Fallback: try to access via type assertion for common schema patterns
		return extractTableNamesViaReflection(s)
	}
}

// extractTableNamesViaReflection is the fallback that uses reflection
func extractTableNamesViaReflection(schema interface{}) map[string]bool {
	schemaVal := reflect.ValueOf(schema)
	if schemaVal.Kind() == reflect.Ptr {
		schemaVal = schemaVal.Elem()
	}

	if schemaVal.Kind() != reflect.Struct {
		return nil
	}

	tablesField := schemaVal.FieldByName("Tables")
	if !tablesField.IsValid() || tablesField.Kind() != reflect.Slice {
		return nil
	}

	tableNames := make(map[string]bool, tablesField.Len())
	for i := 0; i < tablesField.Len(); i++ {
		tablePtr := tablesField.Index(i)
		if tablePtr.Kind() == reflect.Ptr {
			tablePtr = tablePtr.Elem()
		}
		if tablePtr.Kind() == reflect.Struct {
			nameField := tablePtr.FieldByName("Name")
			if nameField.IsValid() && nameField.Kind() == reflect.String {
				tableNames[strings.ToLower(nameField.String())] = true
			}
		}
	}
	return tableNames
}

// ValidateColumnReferences checks if columns referenced in queries exist in the schema
func ValidateColumnReferences(sql string, schema interface{}, sourceFile string) error {
	if schema == nil {
		return nil
	}

	if sourceFile == "" {
		sourceFile = "queries"
	}

	sqlUpper := strings.ToUpper(sql)
	if strings.Contains(sqlUpper, "UNION") {
		return nil
	}

	// Pre-process SQL: strip subqueries and OVER(...) blocks so clause-level
	// regexes don't match keywords inside subqueries or window functions.
	outerSQL := stripSubqueryBlocks(stripOverClauses(sql))

	schemaVal := reflect.ValueOf(schema)
	if schemaVal.Kind() == reflect.Ptr {
		schemaVal = schemaVal.Elem()
	}

	if schemaVal.Kind() != reflect.Struct {
		return nil
	}

	tablesField := schemaVal.FieldByName("Tables")
	if !tablesField.IsValid() || tablesField.Kind() != reflect.Slice {
		return nil
	}

	// Build table structure with columns
	type tableInfo struct {
		name    string
		columns map[string]bool
	}

	tables := make(map[string]*tableInfo)
	for i := 0; i < tablesField.Len(); i++ {
		tablePtr := tablesField.Index(i)
		if tablePtr.Kind() == reflect.Ptr {
			tablePtr = tablePtr.Elem()
		}
		if tablePtr.Kind() == reflect.Struct {
			nameField := tablePtr.FieldByName("Name")
			columnsField := tablePtr.FieldByName("Columns")

			if nameField.IsValid() && nameField.Kind() == reflect.String {
				tableName := strings.ToLower(nameField.String())
				tblInfo := &tableInfo{
					name:    nameField.String(),
					columns: make(map[string]bool),
				}

				if columnsField.IsValid() && columnsField.Kind() == reflect.Slice {
					for j := 0; j < columnsField.Len(); j++ {
						colPtr := columnsField.Index(j)
						if colPtr.Kind() == reflect.Ptr {
							colPtr = colPtr.Elem()
						}
						if colPtr.Kind() == reflect.Struct {
							colNameField := colPtr.FieldByName("Name")
							if colNameField.IsValid() && colNameField.Kind() == reflect.String {
								tblInfo.columns[strings.ToLower(colNameField.String())] = true
							}
						}
					}
				}
				tables[tableName] = tblInfo
			}
		}
	}

	aliasToTable := make(map[string]string, 4) // Pre-allocate

	matches := tableAliasPatternRegex.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			tableName := match[1]
			alias := match[2]
			aliasToTable[strings.ToLower(alias)] = strings.ToLower(tableName)
		}
	}

	matches = joinPatternRegex.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			tableName := match[1]
			alias := match[2]
			aliasToTable[strings.ToLower(alias)] = strings.ToLower(tableName)
		}
	}

	columnRefs := columnRefPatternRegex.FindAllStringSubmatch(sql, -1)

	for _, ref := range columnRefs {
		if len(ref) < 3 {
			continue
		}

		tableOrAlias := ref[1]
		columnName := ref[2]

		if IsSQLKeyword(tableOrAlias) || IsSQLKeyword(columnName) {
			continue
		}

		tableName := strings.ToLower(tableOrAlias)
		if realTable, ok := aliasToTable[tableName]; ok {
			tableName = realTable
		}

		table, tableExists := tables[tableName]
		if !tableExists {
			continue
		}

		columnExists := table.columns[strings.ToLower(columnName)]

		if !columnExists {
			lines := strings.Split(sql, "\n")
			lineNum := 0
			colPos := 0
			for i, line := range lines {
				if strings.Contains(line, ref[0]) {
					lineNum = i + 1
					colPos = strings.Index(line, ref[0]) + len(tableOrAlias) + 1
					break
				}
			}

			return fmt.Errorf("# package flash\ndb\\queries\\%s.sql:%d:%d: column reference \"%s\" not found in table \"%s\"", sourceFile, lineNum, colPos, columnName, table.name)
		}
	}

	knownAliases := make(map[string]bool)
	for alias := range aliasToTable {
		knownAliases[alias] = true
	}

	aliasMatches := aliasExtractPatternRegex.FindAllStringSubmatch(sql, -1)
	for _, match := range aliasMatches {
		if len(match) >= 3 && match[2] != "" {
			knownAliases[strings.ToLower(match[2])] = true
		}
	}

	// Extract SELECT aliases (e.g. RANK() OVER (...) AS rank, COUNT(*) AS cnt)
	// so ORDER BY rank, GROUP BY cnt don't get falsely flagged as missing columns.
	selectAliasMatches := selectAliasRegex.FindAllStringSubmatch(sql, -1)
	for _, match := range selectAliasMatches {
		if len(match) >= 2 {
			alias := strings.ToLower(match[1])
			if !IsSQLKeyword(alias) {
				knownAliases[alias] = true
			}
		}
	}

	var primaryTable *tableInfo
	if fromMatch := fromPatternRegex.FindStringSubmatch(outerSQL); len(fromMatch) > 1 {
		tableName := strings.ToLower(fromMatch[1])
		primaryTable = tables[tableName]
	}

	if primaryTable == nil {
		if insertMatch := insertPatternRegex.FindStringSubmatch(outerSQL); len(insertMatch) > 1 {
			tableName := strings.ToLower(insertMatch[1])
			primaryTable = tables[tableName]
		}
	}

	hasJoin := joinCheckRegex.MatchString(outerSQL)

	if primaryTable != nil && !hasJoin {
		clausePatterns := []*regexp.Regexp{
			whereClauseRegex,
			setClauseRegex,
			orderByClauseRegex,
			groupByClauseRegex,
			havingClauseRegex,
		}

		for _, pattern := range clausePatterns {
			if matches := pattern.FindStringSubmatch(outerSQL); len(matches) > 1 {
				clauseText := matches[1]

				// Strip SQL string literals so quoted values
				// (e.g. status = 'published') don't get flagged as column names.
				clauseText = stripStringLiterals(clauseText)

				colMatches := unqualifiedColPatternRegex.FindAllString(clauseText, -1)

				for _, colName := range colMatches {
					colLower := strings.ToLower(colName)

					if IsSQLKeyword(colName) ||
						colLower == "true" || colLower == "false" || colLower == "null" ||
						colLower == "and" || colLower == "or" || colLower == "not" ||
						strings.Contains(clauseText, colName+"(") { // Skip functions
						continue
					}

					if paramCheckRegex.MatchString(colName) {
						continue
					}

					if knownAliases[colLower] {
						continue
					}

					if !primaryTable.columns[colLower] {
						lines := strings.Split(sql, "\n")
						lineNum := 1
						colPos := 1
						upperCol := strings.ToUpper(colName)

						for i, line := range lines {
							upperLine := strings.ToUpper(line)
							if strings.Contains(upperLine, upperCol) {
								lineNum = i + 1
								colPos = strings.Index(upperLine, upperCol) + 1
								break
							}
						}

						return fmt.Errorf("# package flash\ndb\\queries\\%s.sql:%d:%d: column \"%s\" does not exist in table \"%s\"",
							sourceFile, lineNum, colPos, colName, primaryTable.name)
					}
				}
			}
		}
	}

	return nil
}
