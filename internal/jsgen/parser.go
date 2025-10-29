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
	Name    string
	SQL     string
	Cmd     string
	Comment string
	Params  []*Param
	Columns []*QueryColumn
}

type Param struct {
	Name string
	Type string
}

type QueryColumn struct {
	Name  string
	Type  string
	Table string
}

func (g *Generator) parseSchema() (*Schema, error) {
	regexOnce.Do(initRegex)

	if g.cachedSchema != nil {
		return g.cachedSchema, nil
	}

	schema := &Schema{Tables: []*Table{}}

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
			tables := g.parseCreateTables(string(content))
			schema.Tables = append(schema.Tables, tables...)
		}
	} else {
		content, err := os.ReadFile(schemaPath)
		if err != nil {
			return schema, nil
		}
		tables := g.parseCreateTables(string(content))
		schema.Tables = append(schema.Tables, tables...)
	}

	return schema, nil
}

func (g *Generator) parseCreateTables(sql string) []*Table {
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

			table.Columns = append(table.Columns, &Column{
				Name:     parts[0],
				Type:     parts[1],
				Nullable: !strings.Contains(lineUpper, "NOT NULL"),
			})
		}

		if len(table.Columns) > 0 {
			tables = append(tables, table)
		}
	}

	return tables
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

func (g *Generator) parseQueries() ([]*Query, error) {
	regexOnce.Do(initRegex)

	queriesPath := g.Config.Queries
	if !filepath.IsAbs(queriesPath) {
		cwd, _ := os.Getwd()
		queriesPath = filepath.Join(cwd, queriesPath)
	}

	schema := g.cachedSchema
	if schema == nil {
		schema, _ = g.parseSchema()
	}

	files, err := filepath.Glob(filepath.Join(queriesPath, "*.sql"))
	if err != nil {
		return nil, err
	}

	queries := make([]*Query, 0, len(files)*4)
	for _, file := range files {
		fileQueries, err := g.parseQueryFile(file, schema)
		if err != nil {
			continue
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

		if strings.HasPrefix(line, "-- name:") {
			if currentQuery != nil {
				currentQuery.SQL = strings.TrimSpace(strings.Join(sqlLines, " "))
				currentQuery.Comment = comment
				g.analyzeQuery(currentQuery, schema)
				queries = append(queries, currentQuery)
			}

			parts := strings.Fields(line[8:])
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
		g.analyzeQuery(currentQuery, schema)
		queries = append(queries, currentQuery)
	}

	return queries, scanner.Err()
}

func (g *Generator) analyzeQuery(query *Query, schema *Schema) {
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

	// Only parse columns for SELECT queries, not for DELETE/UPDATE/INSERT
	sqlUpper := strings.ToUpper(query.SQL)
	if strings.HasPrefix(strings.TrimSpace(sqlUpper), "SELECT") &&
		!strings.HasPrefix(strings.TrimSpace(sqlUpper), "DELETE") &&
		!strings.HasPrefix(strings.TrimSpace(sqlUpper), "UPDATE") &&
		!strings.HasPrefix(strings.TrimSpace(sqlUpper), "INSERT") {
		columnsStr := extractSelectColumns(query.SQL)
		if columnsStr != "" {
			if strings.TrimSpace(columnsStr) != "*" {
				colNames := smartSplitColumns(columnsStr)
				query.Columns = make([]*QueryColumn, 0, len(colNames))

				asRegex := regexp.MustCompile(`(?i)\s+AS\s+`)

				for _, colName := range colNames {
					colName = strings.TrimSpace(colName)

					if loc := asRegex.FindStringIndex(colName); loc != nil {
						colName = strings.TrimSpace(colName[loc[1]:])
					} else {
						if !strings.Contains(colName, "(") {
							if idx := strings.Index(colName, "."); idx != -1 {
								colName = colName[idx+1:]
							}
						}
					}

					query.Columns = append(query.Columns, &QueryColumn{
						Name:  colName,
						Type:  "string",
						Table: tableName,
					})
				}
			} else {
				query.Columns = []*QueryColumn{{
					Name:  "*",
					Type:  "string",
					Table: tableName,
				}}
			}
		}
	}
}

func extractSelectColumns(sql string) string {
	sqlUpper := strings.ToUpper(sql)
	selectIdx := strings.Index(sqlUpper, "SELECT")
	if selectIdx == -1 {
		return ""
	}

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
			if strings.EqualFold(col.Name, paramName) {
				return g.mapSQLTypeToJS(col.Type)
			}
		}
	}

	wherePattern := fmt.Sprintf(`(?i)WHERE\s+(\w+)\s*=\s*\$%d`, paramIndex)
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

	return "any"
}

func (g *Generator) inferParamName(sql string, paramIndex int) string {
	wherePattern := fmt.Sprintf(`(?i)WHERE\s+(\w+)\s*=\s*\$%d`, paramIndex)
	whereRe := regexp.MustCompile(wherePattern)
	if match := whereRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	if strings.Contains(strings.ToUpper(sql), "INSERT") {
		if match := insertColRegex.FindStringSubmatch(sql); len(match) > 1 {
			colNames := strings.Split(match[1], ",")
			if paramIndex <= len(colNames) {
				return strings.TrimSpace(colNames[paramIndex-1])
			}
		}
	}

	setPattern := fmt.Sprintf(`(?i)SET\s+(\w+)\s*=\s*\$%d`, paramIndex)
	setRe := regexp.MustCompile(setPattern)
	if match := setRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	return fmt.Sprintf("param%d", paramIndex)
}
