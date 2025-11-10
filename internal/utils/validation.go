package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// ValidateTableReferences checks if tables referenced in queries exist in the schema
func ValidateTableReferences(sql string, schema interface{}, sourceFile string) error {
	if schema == nil {
		return nil
	}

	if sourceFile == "" {
		sourceFile = "queries"
	}

	// Extract tables using reflection to avoid import cycles
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

	// Extract table names from schema
	tableNames := make(map[string]bool)
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

	cteNames := make(map[string]bool)
	ctePattern := regexp.MustCompile(`(?i)(\w+)\s+AS\s*\(`)
	cteMatches := ctePattern.FindAllStringSubmatch(sql, -1)
	for _, match := range cteMatches {
		if len(match) > 1 {
			cteName := match[1]
			if !IsSQLKeyword(cteName) {
				cteNames[strings.ToLower(cteName)] = true
			}
		}
	}

	tablePattern := regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+(\w+)`)
	matches := tablePattern.FindAllStringSubmatch(sql, -1)

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

			return fmt.Errorf("# package FlashORM\ndb\\queries\\%s.sql:%d:%d: relation \"%s\" does not exist", sourceFile, lineNum, colPos, tableName)
		}
	}

	if foundTableRefs && len(tableNames) == 0 {
		return fmt.Errorf("# package flash\ndb\\queries\\%s.sql:1:1: no tables found in schema, but query references tables", sourceFile)
	}

	return nil
}

// ValidateColumnReferences checks if columns referenced in queries exist in the schema
func ValidateColumnReferences(sql string, schema interface{}, sourceFile string) error {
	if schema == nil {
		return nil
	}

	if sourceFile == "" {
		sourceFile = "queries"
	}

	// Extract tables and columns using reflection to avoid import cycles
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

	tableAliasPattern := regexp.MustCompile(`(?i)FROM\s+(\w+)\s+(\w+)`)
	joinPattern := regexp.MustCompile(`(?i)JOIN\s+(\w+)\s+(\w+)`)

	aliasToTable := make(map[string]string)

	matches := tableAliasPattern.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			tableName := match[1]
			alias := match[2]
			aliasToTable[strings.ToLower(alias)] = strings.ToLower(tableName)
		}
	}

	matches = joinPattern.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			tableName := match[1]
			alias := match[2]
			aliasToTable[strings.ToLower(alias)] = strings.ToLower(tableName)
		}
	}

	// First, check qualified column references (table.column)
	columnRefPattern := regexp.MustCompile(`(?i)(\w+)\.(\w+)`)
	columnRefs := columnRefPattern.FindAllStringSubmatch(sql, -1)

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

	// Build a set of all known table aliases to skip them in validation
	knownAliases := make(map[string]bool)
	for alias := range aliasToTable {
		knownAliases[alias] = true
	}

	// Also add single-letter aliases that are commonly used (p, u, c, etc.)
	// These are extracted from FROM and JOIN clauses
	aliasExtractPattern := regexp.MustCompile(`(?i)(?:FROM|JOIN)\s+(\w+)(?:\s+(?:AS\s+)?(\w+))?`)
	aliasMatches := aliasExtractPattern.FindAllStringSubmatch(sql, -1)
	for _, match := range aliasMatches {
		if len(match) >= 3 && match[2] != "" {
			// Has explicit alias
			knownAliases[strings.ToLower(match[2])] = true
		}
	}

	// Now check unqualified column references in WHERE, SET, ORDER BY, GROUP BY, HAVING clauses
	// Get the primary table from the query
	var primaryTable *tableInfo
	fromPattern := regexp.MustCompile(`(?i)\bFROM\s+(\w+)`)
	if fromMatch := fromPattern.FindStringSubmatch(sql); len(fromMatch) > 1 {
		tableName := strings.ToLower(fromMatch[1])
		primaryTable = tables[tableName]
	}

	// Also check INSERT/UPDATE tables
	if primaryTable == nil {
		insertPattern := regexp.MustCompile(`(?i)\b(?:INSERT\s+INTO|UPDATE)\s+(\w+)`)
		if insertMatch := insertPattern.FindStringSubmatch(sql); len(insertMatch) > 1 {
			tableName := strings.ToLower(insertMatch[1])
			primaryTable = tables[tableName]
		}
	}

	// Only validate unqualified columns for simple queries without JOINs
	// For complex queries with JOINs, qualified column references are mandatory and already validated above
	hasJoin := regexp.MustCompile(`(?i)\bJOIN\b`).MatchString(sql)

	if primaryTable != nil && !hasJoin {
		// Extract unqualified column names from WHERE, SET, ORDER BY, GROUP BY, HAVING clauses
		// NOTE: We do NOT validate SELECT clause here because it's already validated by the query parser
		clausePatterns := []*regexp.Regexp{
			regexp.MustCompile(`(?i)\bWHERE\s+(.*?)(?:\s+(?:LIMIT|ORDER|GROUP|HAVING|;|$))`),
			regexp.MustCompile(`(?i)\bSET\s+(.*?)(?:\s+(?:WHERE|;|$))`),
			regexp.MustCompile(`(?i)\bORDER\s+BY\s+(.*?)(?:\s+(?:LIMIT|;|$))`),
			regexp.MustCompile(`(?i)\bGROUP\s+BY\s+(.*?)(?:\s+(?:HAVING|ORDER|LIMIT|;|$))`),
			regexp.MustCompile(`(?i)\bHAVING\s+(.*?)(?:\s+(?:ORDER|LIMIT|;|$))`),
		}

		// Precompile regex for numbers/parameters to avoid recompiling in loop
		paramRegex := regexp.MustCompile(`^\d+$|^\$\d+$|\?`)

		for _, pattern := range clausePatterns {
			if matches := pattern.FindStringSubmatch(sql); len(matches) > 1 {
				clauseText := matches[1]

				// Extract potential column names (word boundaries, not preceded by table.)
				// This regex matches words that are not preceded by a dot and not SQL keywords
				unqualifiedColPattern := regexp.MustCompile(`\b(\w+)\b`)
				colMatches := unqualifiedColPattern.FindAllString(clauseText, -1)

				for _, colName := range colMatches {
					colLower := strings.ToLower(colName)

					// Skip SQL keywords, operators, and common functions
					if IsSQLKeyword(colName) ||
						colLower == "true" || colLower == "false" || colLower == "null" ||
						colLower == "and" || colLower == "or" || colLower == "not" ||
						strings.Contains(clauseText, colName+"(") { // Skip functions
						continue
					}

					// Skip if it's a number or parameter
					if paramRegex.MatchString(colName) {
						continue
					}

					// Skip if it's a known table alias
					if knownAliases[colLower] {
						continue
					}

					// Check if this column exists in the primary table
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
