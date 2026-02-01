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
}

// tableAccessor is a helper interface for extracting table info via type assertion
type tableAccessor interface {
	GetTableNames() map[string]bool
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

			for i, line := range lines {
				if strings.Contains(strings.ToUpper(line), strings.ToUpper(tableName)) {
					lineNum = i + 1
					upperLine := strings.ToUpper(line)
					upperTable := strings.ToUpper(tableName)
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

// extractTableNamesFromSchema extracts table names from various schema types
func extractTableNamesFromSchema(schema interface{}) map[string]bool {
	// Try known interface types first (fast path)
	if s, ok := schema.(interface{ GetTables() []interface{ GetName() string } }); ok {
		tables := s.GetTables()
		names := make(map[string]bool, len(tables))
		for _, t := range tables {
			names[strings.ToLower(t.GetName())] = true
		}
		return names
	}

	// Try struct with Tables field via type assertion on common patterns
	type tablesHolder struct {
		Tables interface{}
	}

	// Check for *parser.Schema-like structure with Tables []*Table
	type tableWithName interface {
		GetName() string
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

	if strings.Contains(strings.ToUpper(sql), "UNION") {
		return nil
	}

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

	var primaryTable *tableInfo
	if fromMatch := fromPatternRegex.FindStringSubmatch(sql); len(fromMatch) > 1 {
		tableName := strings.ToLower(fromMatch[1])
		primaryTable = tables[tableName]
	}

	if primaryTable == nil {
		if insertMatch := insertPatternRegex.FindStringSubmatch(sql); len(insertMatch) > 1 {
			tableName := strings.ToLower(insertMatch[1])
			primaryTable = tables[tableName]
		}
	}

	hasJoin := joinCheckRegex.MatchString(sql)

	if primaryTable != nil && !hasJoin {
		clausePatterns := []*regexp.Regexp{
			whereClauseRegex,
			setClauseRegex,
			orderByClauseRegex,
			groupByClauseRegex,
			havingClauseRegex,
		}

		for _, pattern := range clausePatterns {
			if matches := pattern.FindStringSubmatch(sql); len(matches) > 1 {
				clauseText := matches[1]

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

						for i, line := range lines {
							if strings.Contains(strings.ToUpper(line), strings.ToUpper(colName)) {
								lineNum = i + 1
								upperLine := strings.ToUpper(line)
								upperCol := strings.ToUpper(colName)
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
