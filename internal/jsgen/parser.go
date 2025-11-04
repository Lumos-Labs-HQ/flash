package jsgen

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var (
	createTableRegex *regexp.Regexp
	fromRegex        *regexp.Regexp
	paramRegex       *regexp.Regexp
	insertColRegex   *regexp.Regexp
	regexOnce        sync.Once
)

func initRegex() {
	createTableRegex = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s*\(([\s\S]*?)\);`)
	fromRegex = regexp.MustCompile(`(?i)FROM\s+(\w+)`)
	paramRegex = regexp.MustCompile(`\$\d+|\?`)
	insertColRegex = regexp.MustCompile(`(?i)INSERT\s+INTO\s+\w+\s*\(([\s\S]*?)\)\s*VALUES`)
}

type Schema struct {
	Tables []*Table
	Enums  []*Enum
}

type Enum struct {
	Name   string
	Values []string
}

type Table struct {
	Name    string
	Columns []*Column
}

type Column struct {
	Name     string
	Type     string
	Nullable bool
}

type Query struct {
	Name       string
	SQL        string
	Cmd        string
	Comment    string
	Params     []*Param
	Columns    []*QueryColumn
	SourceFile string // Name of the SQL file this query came from (without path/extension)
}

type Param struct {
	Name string
	Type string
}

type QueryColumn struct {
	Name     string
	Type     string
	Table    string
	Nullable bool
}

// removeComments removes SQL comments (-- and /* */) from the input
func removeComments(sql string) string {
	var result strings.Builder
	lines := strings.Split(sql, "\n")

	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx != -1 {
			line = line[:idx]
		}
		result.WriteString(line)
		result.WriteString("\n")
	}

	cleaned := result.String()
	blockCommentRegex := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	return blockCommentRegex.ReplaceAllString(cleaned, "")
}

func (g *Generator) ParseSchema() (*Schema, error) {
	regexOnce.Do(initRegex)

	schema := &Schema{
		Tables: []*Table{},
		Enums:  []*Enum{},
	}

	schemaPath := g.Config.SchemaPath
	if !filepath.IsAbs(schemaPath) {
		cwd, _ := os.Getwd()
		schemaPath = filepath.Join(cwd, schemaPath)
	}

	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		schemaDir := filepath.Dir(schemaPath)
		files, err := filepath.Glob(filepath.Join(schemaDir, "*.sql"))
		if err != nil || len(files) == 0 {
			return schema, nil
		}

		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}

			if err := validateSchemaSyntax(string(content), file); err != nil {
				return nil, err
			}

			tables := g.parseCreateTables(string(content))
			schema.Tables = append(schema.Tables, tables...)
			enums := g.parseCreateEnums(string(content))
			schema.Enums = append(schema.Enums, enums...)
		}
	} else {
		content, err := os.ReadFile(schemaPath)
		if err != nil {
			return schema, nil
		}

		if err := validateSchemaSyntax(string(content), schemaPath); err != nil {
			return nil, err
		}

		tables := g.parseCreateTables(string(content))
		schema.Tables = append(schema.Tables, tables...)
		enums := g.parseCreateEnums(string(content))
		schema.Enums = append(schema.Enums, enums...)
	}

	return schema, nil
}

func (g *Generator) parseCreateTables(sql string) []*Table {
	// Remove SQL comments before parsing
	sql = removeComments(sql)

	tables := make([]*Table, 0, 8)
	matches := createTableRegex.FindAllStringSubmatch(sql, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		table := &Table{
			Name:    match[1],
			Columns: make([]*Column, 0, 16),
		}

		lines := splitColumns(match[2])
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			lineUpper := strings.ToUpper(line)
			if strings.HasPrefix(lineUpper, "PRIMARY") ||
				strings.HasPrefix(lineUpper, "FOREIGN") ||
				strings.HasPrefix(lineUpper, "UNIQUE") ||
				strings.HasPrefix(lineUpper, "CHECK") ||
				strings.HasPrefix(lineUpper, "CONSTRAINT") ||
				strings.HasPrefix(lineUpper, "INDEX") ||
				strings.HasPrefix(lineUpper, "KEY") {
				continue
			}

			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}

			// Check if column is nullable
			// PRIMARY KEY and SERIAL columns are always non-nullable
			isNullable := !strings.Contains(lineUpper, "NOT NULL") &&
				!strings.Contains(lineUpper, "PRIMARY KEY") &&
				!strings.Contains(strings.ToUpper(parts[1]), "SERIAL")

			table.Columns = append(table.Columns, &Column{
				Name:     parts[0],
				Type:     parts[1],
				Nullable: isNullable,
			})
		}

		if len(table.Columns) > 0 {
			tables = append(tables, table)
		}
	}

	return tables
}

func (g *Generator) parseCreateEnums(sql string) []*Enum {
	sql = removeComments(sql)

	enums := make([]*Enum, 0)
	enumRegex := regexp.MustCompile(`(?i)CREATE\s+TYPE\s+(\w+)\s+AS\s+ENUM\s*\(\s*([^)]+)\s*\)`)
	matches := enumRegex.FindAllStringSubmatch(sql, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		enumName := match[1]
		valuesStr := match[2]

		// Parse enum values
		var values []string
		for _, v := range strings.Split(valuesStr, ",") {
			v = strings.TrimSpace(v)
			v = strings.Trim(v, "'\"")
			if v != "" {
				values = append(values, v)
			}
		}

		if len(values) > 0 {
			enums = append(enums, &Enum{
				Name:   enumName,
				Values: values,
			})
		}
	}

	return enums
}

func splitColumns(columnsStr string) []string {
	var result []string
	var current strings.Builder
	parenDepth := 0

	for _, char := range columnsStr {
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

func (g *Generator) ParseQueries(schema *Schema) ([]*Query, error) {
	regexOnce.Do(initRegex)

	queriesPath := g.Config.Queries
	if !filepath.IsAbs(queriesPath) {
		cwd, _ := os.Getwd()
		queriesPath = filepath.Join(cwd, queriesPath)
	}

	files, err := filepath.Glob(filepath.Join(queriesPath, "*.sql"))
	if err != nil {
		return nil, err
	}

	queries := make([]*Query, 0, len(files)*4)
	for _, file := range files {
		fileQueries, err := g.parseQueryFile(file, schema)
		if err != nil {
			return nil, err
		}
		queries = append(queries, fileQueries...)
	}

	return queries, nil
}

func (g *Generator) parseQueryFile(filename string, schema *Schema) ([]*Query, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Extract source file name without path and extension
	baseName := filepath.Base(filename)
	sourceFileName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	queries := []*Query{}
	scanner := bufio.NewScanner(file)

	var currentQuery *Query
	var sqlLines []string
	var comment string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		// Support both "-- name:" and "-- name :" formats
		if strings.HasPrefix(line, "-- name:") || strings.HasPrefix(line, "-- name :") {
			if currentQuery != nil {
				currentQuery.SQL = strings.TrimSpace(strings.Join(sqlLines, " "))
				currentQuery.Comment = comment
				currentQuery.SourceFile = sourceFileName
				if err := g.analyzeQuery(currentQuery, schema); err != nil {
					return nil, err
				}
				queries = append(queries, currentQuery)
			}

			// Extract the part after "-- name" (with or without space before colon)
			nameStart := strings.Index(line, "name")
			if nameStart == -1 {
				continue
			}
			remainder := line[nameStart+4:] // Skip "name"
			remainder = strings.TrimLeft(remainder, " :")

			parts := strings.Fields(remainder)
			if len(parts) >= 2 {
				currentQuery = &Query{
					Name: parts[0],
					Cmd:  parts[1],
				}
				sqlLines = []string{}
				comment = ""
			}
		} else if strings.HasPrefix(line, "--") {
			comment = strings.TrimPrefix(line, "--")
			comment = strings.TrimSpace(comment)
		} else if currentQuery != nil {
			sqlLines = append(sqlLines, line)
		}
	}

	if currentQuery != nil {
		currentQuery.SQL = strings.TrimSpace(strings.Join(sqlLines, " "))
		currentQuery.Comment = comment
		currentQuery.SourceFile = sourceFileName
		if err := g.analyzeQuery(currentQuery, schema); err != nil {
			return nil, err
		}
		queries = append(queries, currentQuery)
	}

	return queries, scanner.Err()
}

func (g *Generator) analyzeQuery(query *Query, schema *Schema) error {
	var tableName string
	if match := fromRegex.FindStringSubmatch(query.SQL); len(match) > 1 {
		tableName = match[1]
	}

	if tableName == "" {
		if match := g.insertRegex.FindStringSubmatch(query.SQL); len(match) > 1 {
			tableName = match[1]
		}
	}
	if tableName == "" {
		if match := g.updateRegex.FindStringSubmatch(query.SQL); len(match) > 1 {
			tableName = match[1]
		}
	}

	var table *Table
	for _, t := range schema.Tables {
		if strings.EqualFold(t.Name, tableName) {
			table = t
			break
		}
	}

	paramMatches := paramRegex.FindAllString(query.SQL, -1)
	seen := make(map[string]bool, len(paramMatches))
	uniqueParams := make([]string, 0, len(paramMatches))

	for _, p := range paramMatches {
		if !seen[p] {
			seen[p] = true
			uniqueParams = append(uniqueParams, p)
		}
	}

	query.Params = make([]*Param, len(uniqueParams))
	for i := range uniqueParams {
		paramName := fmt.Sprintf("param%d", i+1)
		paramType := "any"

		if table != nil {
			inferredName := g.inferParamName(query.SQL, i+1)
			if inferredName != "" && inferredName != paramName {
				paramName = inferredName
			}

			paramType = g.inferParamType(query.SQL, i+1, table, paramName)
		}

		query.Params[i] = &Param{
			Name: paramName,
			Type: paramType,
		}
	}

	// Parse columns for SELECT queries (including CTEs)
	sqlUpper := strings.ToUpper(query.SQL)
	sqlTrimmed := strings.TrimSpace(sqlUpper)

	isSelectQuery := strings.HasPrefix(sqlTrimmed, "SELECT") || strings.HasPrefix(sqlTrimmed, "WITH")

	isNotModifying := !containsSQLKeyword(sqlTrimmed, "DELETE") &&
		!containsSQLKeyword(sqlTrimmed, "UPDATE") &&
		!containsSQLKeyword(sqlTrimmed, "INSERT")

	if isSelectQuery && isNotModifying {
		columnsStr := extractSelectColumns(query.SQL)

		// Try to parse columns
		if columnsStr != "" && strings.TrimSpace(columnsStr) != "*" {
			colNames := smartSplitColumns(columnsStr)

			if len(colNames) > 0 {
				query.Columns = make([]*QueryColumn, 0, len(colNames))

				asRegex := regexp.MustCompile(`(?i)\s+AS\s+`)

				for _, colName := range colNames {
					colName = strings.TrimSpace(colName)
					if colName == "" {
						continue
					}

					if loc := asRegex.FindStringIndex(colName); loc != nil {
						colName = strings.TrimSpace(colName[loc[1]:])
					} else {
						if !strings.Contains(colName, "(") {
							if idx := strings.Index(colName, "."); idx != -1 {
								colName = colName[idx+1:]
							}
						}
					}

					// Look up actual column type from schema
					colType := "string"
					nullable := false
					if table != nil {
						for _, col := range table.Columns {
							if strings.EqualFold(col.Name, colName) {
								colType = col.Type
								nullable = col.Nullable
								break
							}
						}
					}

					query.Columns = append(query.Columns, &QueryColumn{
						Name:     colName,
						Type:     colType,
						Table:    tableName,
						Nullable: nullable,
					})
				}
			}
		}

		if len(query.Columns) == 0 {
			query.Columns = []*QueryColumn{{
				Name:  "*",
				Type:  "string",
				Table: tableName,
			}}
		}
	}

	if err := validateTableReferences(query, schema); err != nil {
		return err
	}

	if err := validateColumnReferences(query, schema); err != nil {
		return err
	}

	return nil
}
func extractSelectColumns(sql string) string {
	sqlUpper := strings.ToUpper(sql)

	// For CTE queries, find the main SELECT after the CTE definitions
	if strings.HasPrefix(strings.TrimSpace(sqlUpper), "WITH") {
		// Find all SELECT positions
		var selectPositions []int
		parenDepth := 0

		for i := 0; i < len(sqlUpper)-6; i++ {
			switch sql[i] {
			case '(':
				parenDepth++
			case ')':
				parenDepth--
			case 'S', 's':
				if parenDepth == 0 && i+6 <= len(sqlUpper) {
					if strings.ToUpper(sql[i:i+6]) == "SELECT" {
						if (i == 0 || !isAlphaNum(sql[i-1])) &&
							(i+6 >= len(sql) || !isAlphaNum(sql[i+6])) {
							selectPositions = append(selectPositions, i)
						}
					}
				}
			}
		}

		// Use the last SELECT (main query)
		if len(selectPositions) > 0 {
			selectIdx := selectPositions[len(selectPositions)-1]
			return extractColumnsFromSelect(sql, selectIdx)
		}
	}

	// Regular SELECT query
	selectIdx := strings.Index(sqlUpper, "SELECT")
	if selectIdx == -1 {
		return ""
	}

	return extractColumnsFromSelect(sql, selectIdx)
}

func extractColumnsFromSelect(sql string, selectIdx int) string {
	start := selectIdx + 6
	for start < len(sql) && (sql[start] == ' ' || sql[start] == '\t' || sql[start] == '\n') {
		start++
	}

	parenDepth := 0
	fromIdx := -1

	for i := start; i < len(sql); i++ {
		switch sql[i] {
		case '(':
			parenDepth++
		case ')':
			parenDepth--
		case 'F', 'f':
			if parenDepth == 0 && i+4 <= len(sql) {
				potential := strings.ToUpper(sql[i:min(i+4, len(sql))])
				if potential == "FROM" {
					if (i == 0 || !isAlphaNum(sql[i-1])) &&
						(i+4 >= len(sql) || !isAlphaNum(sql[i+4])) {
						fromIdx = i
						break
					}
				}
			}
		case ';':
			if parenDepth == 0 && fromIdx == -1 {
				return strings.TrimSpace(sql[start:i])
			}
		}

		if fromIdx != -1 {
			break
		}
	}

	if fromIdx != -1 {
		return strings.TrimSpace(sql[start:fromIdx])
	}

	end := len(sql)
	for i := start; i < len(sql); i++ {
		if sql[i] == ';' {
			end = i
			break
		}
	}

	return strings.TrimSpace(sql[start:end])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isAlphaNum(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// containsSQLKeyword checks if a SQL keyword appears as a standalone word (not part of column names)
func containsSQLKeyword(sql, keyword string) bool {
	keyword = strings.ToUpper(keyword)
	sql = strings.ToUpper(sql)

	index := 0
	for {
		pos := strings.Index(sql[index:], keyword)
		if pos == -1 {
			return false
		}

		absPos := index + pos

		// Check if it's a word boundary before the keyword
		beforeOK := absPos == 0 || !isAlphaNum(sql[absPos-1])

		// Check if it's a word boundary after the keyword
		afterPos := absPos + len(keyword)
		afterOK := afterPos >= len(sql) || !isAlphaNum(sql[afterPos])

		if beforeOK && afterOK {
			return true
		}

		index = absPos + 1
	}
}

// smartSplitColumns splits column names by comma, respecting parentheses nesting
func smartSplitColumns(columnsStr string) []string {
	var result []string
	var current strings.Builder
	parenDepth := 0

	for _, char := range columnsStr {
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

func (g *Generator) inferParamType(sql string, paramIndex int, table *Table, paramName string) string {
	if paramName != "" && paramName != fmt.Sprintf("param%d", paramIndex) {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, paramName) ||
				strings.EqualFold(col.Name, strings.TrimSuffix(strings.TrimSuffix(paramName, "_start"), "_end")) {
				return g.mapSQLTypeToJS(col.Type)
			}
		}
	}

	aggregatePattern := fmt.Sprintf(`(?i)\b(count|sum|avg|max|min|total)_?\w*\s*[<>=!]+\s*\$%d|\$%d\s*[<>=!]+\s*\b(count|sum|avg|max|min|total)_?\w*`, paramIndex, paramIndex)
	if matched, _ := regexp.MatchString(aggregatePattern, sql); matched {
		return "number"
	}

	numericAliasPattern := fmt.Sprintf(`(?i)\w*\.(count|sum|avg|total|min|max|num|qty|quantity|amount)_?\w*\s*[<>=!]+\s*\$%d|\$%d\s*[<>=!]+\s*\w*\.(count|sum|avg|total|min|max|num|qty|quantity|amount)_?\w*`, paramIndex, paramIndex)
	if matched, _ := regexp.MatchString(numericAliasPattern, sql); matched {
		return "number"
	}

	wherePattern := fmt.Sprintf(`(?i)WHERE\s+(?:\w+\.)?(\w+)\s*=\s*\$%d`, paramIndex)
	whereRe := regexp.MustCompile(wherePattern)
	if match := whereRe.FindStringSubmatch(sql); len(match) > 1 {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, match[1]) {
				return g.mapSQLTypeToJS(col.Type)
			}
		}
	}

	if strings.Contains(strings.ToUpper(sql), "INSERT") {
		if match := insertColRegex.FindStringSubmatch(sql); len(match) > 1 {
			colNames := strings.Split(match[1], ",")
			if paramIndex <= len(colNames) {
				colName := strings.TrimSpace(colNames[paramIndex-1])
				for _, col := range table.Columns {
					if strings.EqualFold(col.Name, colName) {
						return g.mapSQLTypeToJS(col.Type)
					}
				}
			}
		}
	}

	setPattern := fmt.Sprintf(`(?i)SET\s+(\w+)\s*=\s*\$%d`, paramIndex)
	setRe := regexp.MustCompile(setPattern)
	if match := setRe.FindStringSubmatch(sql); len(match) > 1 {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, match[1]) {
				return g.mapSQLTypeToJS(col.Type)
			}
		}
	}

	// Check for LIMIT clause
	limitPattern := fmt.Sprintf(`(?i)LIMIT\s+\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(limitPattern, sql); matched {
		return "number"
	}

	// Check for OFFSET clause
	offsetPattern := fmt.Sprintf(`(?i)OFFSET\s+\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(offsetPattern, sql); matched {
		return "number"
	}

	// Check for BETWEEN clause (dates)
	betweenPattern := fmt.Sprintf(`(?i)(\w+)\s+BETWEEN\s+\$%d`, paramIndex)
	betweenRe := regexp.MustCompile(betweenPattern)
	if match := betweenRe.FindStringSubmatch(sql); len(match) > 1 {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, match[1]) {
				return g.mapSQLTypeToJS(col.Type)
			}
		}
	}

	betweenEndPattern := fmt.Sprintf(`(?i)BETWEEN\s+\$\d+\s+AND\s+\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(betweenEndPattern, sql); matched {
		betweenStartRe := regexp.MustCompile(`(?i)(\w+)\s+BETWEEN`)
		if match := betweenStartRe.FindStringSubmatch(sql); len(match) > 1 {
			for _, col := range table.Columns {
				if strings.EqualFold(col.Name, match[1]) {
					return g.mapSQLTypeToJS(col.Type)
				}
			}
		}
	}

	// Check for date comparisons
	datePattern := fmt.Sprintf(`(?i)(created_at|updated_at|date|time)\s*[<>=]+\s*\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(datePattern, sql); matched {
		return "Date | string"
	}

	return "any"
}

func (g *Generator) inferParamName(sql string, paramIndex int) string {
	// WHERE clause
	wherePattern := fmt.Sprintf(`(?i)WHERE\s+(?:\w+\.)?(\w+)\s*=\s*\$%d`, paramIndex)
	whereRe := regexp.MustCompile(wherePattern)
	if match := whereRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// INSERT clause
	if strings.Contains(strings.ToUpper(sql), "INSERT") {
		if match := insertColRegex.FindStringSubmatch(sql); len(match) > 1 {
			colNames := strings.Split(match[1], ",")
			if paramIndex <= len(colNames) {
				return strings.TrimSpace(colNames[paramIndex-1])
			}
		}
	}

	// SET clause
	setPattern := fmt.Sprintf(`(?i)SET\s+(\w+)\s*=\s*\$%d`, paramIndex)
	setRe := regexp.MustCompile(setPattern)
	if match := setRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// LIMIT clause
	limitPattern := fmt.Sprintf(`(?i)LIMIT\s+\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(limitPattern, sql); matched {
		return "limit"
	}

	// BETWEEN clause
	betweenPattern := fmt.Sprintf(`(?i)(\w+)\s+BETWEEN\s+\$%d`, paramIndex)
	betweenRe := regexp.MustCompile(betweenPattern)
	if match := betweenRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1] + "_start"
	}

	betweenEndPattern := fmt.Sprintf(`(?i)BETWEEN\s+\$\d+\s+AND\s+\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(betweenEndPattern, sql); matched {
		betweenStartRe := regexp.MustCompile(`(?i)(\w+)\s+BETWEEN`)
		if match := betweenStartRe.FindStringSubmatch(sql); len(match) > 1 {
			return match[1] + "_end"
		}
	}

	// Comparison operators
	compPattern := fmt.Sprintf(`(?i)(\w+)\s*[<>=]+\s*\$%d`, paramIndex)
	compRe := regexp.MustCompile(compPattern)
	if match := compRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	return fmt.Sprintf("param%d", paramIndex)
}

// validateSchemaSyntax checks for common SQL syntax errors in schema files
func validateSchemaSyntax(content, filePath string) error {
	lines := strings.Split(content, "\n")

	// Track CREATE TABLE blocks
	inCreateTable := false
	tableStartLine := 0
	parenDepth := 0

	for lineNum, line := range lines {
		lineNumber := lineNum + 1
		trimmed := strings.TrimSpace(line)

		if strings.Contains(strings.ToUpper(trimmed), "CREATE TABLE") {
			inCreateTable = true
			tableStartLine = lineNumber
			parenDepth = 0
		}

		for _, ch := range line {
			switch ch {
			case '(':
				parenDepth++
			case ')':
				parenDepth--
			}
		}

		// Check for trailing comma before closing paren (common error)
		if inCreateTable && parenDepth == 0 && strings.Contains(trimmed, ");") {
			for i := lineNum - 1; i >= 0; i-- {
				prevLine := strings.TrimSpace(lines[i])
				if prevLine == "" {
					continue
				}
				if strings.HasSuffix(prevLine, ",") {
					relPath := filepath.Base(filePath)
					return fmt.Errorf("# package graft\n%s:%d:2: syntax error at or near \")\"", relPath, lineNumber)
				}
				break
			}
			inCreateTable = false
		}

		if parenDepth < 0 {
			relPath := filepath.Base(filePath)
			return fmt.Errorf("# package graft\n%s:%d:2: syntax error: unexpected ')'", relPath, lineNumber)
		}
	}

	if inCreateTable && parenDepth > 0 {
		relPath := filepath.Base(filePath)
		return fmt.Errorf("# package graft\n%s:%d:2: syntax error: unclosed CREATE TABLE statement", relPath, tableStartLine)
	}

	return nil
}

// validateTableReferences checks if tables referenced in queries exist in the schema
func validateTableReferences(query *Query, schema *Schema) error {
	if query == nil || schema == nil {
		return nil
	}

	// Extract all CTE (Common Table Expression) names from WITH clause
	// CTEs can be defined as: WITH cte1 AS (...), cte2 AS (...), ...
	cteNames := make(map[string]bool)

	// Find all occurrences of "word AS (" pattern which indicates a CTE definition
	ctePattern := regexp.MustCompile(`(?i)(\w+)\s+AS\s*\(`)
	cteMatches := ctePattern.FindAllStringSubmatch(query.SQL, -1)
	for _, match := range cteMatches {
		if len(match) > 1 {
			cteName := match[1]
			// Exclude SQL keywords like SELECT AS (...)
			if !isSQLKeyword(cteName) {
				cteNames[strings.ToLower(cteName)] = true
			}
		}
	}

	tablePattern := regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+(\w+)`)
	matches := tablePattern.FindAllStringSubmatch(query.SQL, -1)

	// Track if we found any table references
	foundTableRefs := false

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		tableName := match[1]

		if isSQLKeyword(tableName) {
			continue
		}

		// Skip if this is a CTE
		if cteNames[strings.ToLower(tableName)] {
			continue
		}

		foundTableRefs = true

		// Check if table exists in schema
		tableExists := false
		for _, t := range schema.Tables {
			if strings.EqualFold(t.Name, tableName) {
				tableExists = true
				break
			}
		}

		if !tableExists {
			lines := strings.Split(query.SQL, "\n")
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

			return fmt.Errorf("# package graft\ndb\\queries\\users.sql:%d:%d: relation \"%s\" does not exist", lineNum, colPos, tableName)
		}
	}

	// If query references tables but schema is empty, that's an error
	if foundTableRefs && len(schema.Tables) == 0 {
		return fmt.Errorf("# package graft\ndb\\queries\\users.sql:1:1: no tables found in schema, but query references tables")
	}

	return nil
}

// validateColumnReferences checks if columns referenced in queries exist in the schema
func validateColumnReferences(query *Query, schema *Schema) error {
	if query == nil || schema == nil {
		return nil
	}

	tableAliasPattern := regexp.MustCompile(`(?i)FROM\s+(\w+)\s+(\w+)`)
	joinPattern := regexp.MustCompile(`(?i)JOIN\s+(\w+)\s+(\w+)`)

	// Build table alias to table name mapping
	aliasToTable := make(map[string]string)

	matches := tableAliasPattern.FindAllStringSubmatch(query.SQL, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			tableName := match[1]
			alias := match[2]
			aliasToTable[alias] = tableName
		}
	}

	matches = joinPattern.FindAllStringSubmatch(query.SQL, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			tableName := match[1]
			alias := match[2]
			aliasToTable[alias] = tableName
		}
	}

	// Check column references in SELECT, WHERE, GROUP BY clauses
	columnRefPattern := regexp.MustCompile(`(?i)(\w+)\.(\w+)`)
	columnRefs := columnRefPattern.FindAllStringSubmatch(query.SQL, -1)

	for _, ref := range columnRefs {
		if len(ref) < 3 {
			continue
		}

		tableOrAlias := ref[1]
		columnName := ref[2]

		if isSQLKeyword(tableOrAlias) || isSQLKeyword(columnName) {
			continue
		}

		tableName := tableOrAlias
		if realTable, ok := aliasToTable[tableOrAlias]; ok {
			tableName = realTable
		}

		// Find the table in schema
		var table *Table
		for _, t := range schema.Tables {
			if strings.EqualFold(t.Name, tableName) {
				table = t
				break
			}
		}

		if table == nil {
			continue // Table not found in schema,
		}

		columnExists := false
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, columnName) {
				columnExists = true
				break
			}
		}

		if !columnExists {
			lines := strings.Split(query.SQL, "\n")
			lineNum := 0
			colPos := 0
			for i, line := range lines {
				if strings.Contains(line, ref[0]) {
					lineNum = i + 1
					colPos = strings.Index(line, ref[0]) + len(tableOrAlias) + 1
					break
				}
			}

			return fmt.Errorf("# package graft\ndb\\queries\\users.sql:%d:%d: column reference \"%s\" not found", lineNum, colPos, columnName)
		}
	}

	return nil
}

// isSQLKeyword checks if a word is a SQL keyword
func isSQLKeyword(word string) bool {
	keywords := []string{
		"SELECT", "FROM", "WHERE", "JOIN", "INNER", "LEFT", "RIGHT", "OUTER",
		"ON", "AND", "OR", "NOT", "IN", "LIKE", "BETWEEN", "IS", "NULL",
		"GROUP", "BY", "HAVING", "ORDER", "ASC", "DESC", "LIMIT", "OFFSET",
		"INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "TABLE",
		"INDEX", "VIEW", "AS", "DISTINCT", "COUNT", "SUM", "AVG", "MIN", "MAX",
		"CASE", "WHEN", "THEN", "ELSE", "END", "WITH", "RECURSIVE",
	}

	wordUpper := strings.ToUpper(word)
	for _, kw := range keywords {
		if wordUpper == kw {
			return true
		}
	}
	return false
}
