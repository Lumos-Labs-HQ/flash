package jsgen

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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
	tables := []*Table{}

	createTableRegex := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s*\(([\s\S]*?)\);`)
	matches := createTableRegex.FindAllStringSubmatch(sql, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		tableName := match[1]
		columnsStr := match[2]

		table := &Table{
			Name:    tableName,
			Columns: []*Column{},
		}

		lines := splitColumns(columnsStr)
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

			colName := parts[0]
			colType := parts[1]
			nullable := !strings.Contains(strings.ToUpper(line), "NOT NULL")

			table.Columns = append(table.Columns, &Column{
				Name:     colName,
				Type:     colType,
				Nullable: nullable,
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
	queries := []*Query{}

	queriesPath := g.Config.Queries
	if !filepath.IsAbs(queriesPath) {
		cwd, _ := os.Getwd()
		queriesPath = filepath.Join(cwd, queriesPath)
	}

	schema, _ := g.parseSchema()

	files, err := filepath.Glob(filepath.Join(queriesPath, "*.sql"))
	if err != nil {
		return nil, err
	}

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
	fromRegex := regexp.MustCompile(`(?i)FROM\s+(\w+)`)
	if match := fromRegex.FindStringSubmatch(query.SQL); len(match) > 1 {
		tableName = match[1]
	}
	
	if tableName == "" {
		insertRegex := regexp.MustCompile(`(?i)INSERT\s+INTO\s+(\w+)`)
		if match := insertRegex.FindStringSubmatch(query.SQL); len(match) > 1 {
			tableName = match[1]
		}
	}
	if tableName == "" {
		updateRegex := regexp.MustCompile(`(?i)UPDATE\s+(\w+)`)
		if match := updateRegex.FindStringSubmatch(query.SQL); len(match) > 1 {
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

	paramRegex := regexp.MustCompile(`\$\d+|\?`)
	paramMatches := paramRegex.FindAllString(query.SQL, -1)

	seen := make(map[string]bool)
	uniqueParams := []string{}
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
			inferredName := g.inferParamName(query.SQL, i+1, table)
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

	if strings.Contains(strings.ToUpper(query.SQL), "SELECT") {
		selectRegex := regexp.MustCompile(`(?i)SELECT\s+([\s\S]*?)\s+FROM`)
		if selectMatch := selectRegex.FindStringSubmatch(query.SQL); len(selectMatch) > 1 {
			columnsStr := selectMatch[1]
			if strings.TrimSpace(columnsStr) != "*" {
				colNames := strings.Split(columnsStr, ",")
				for _, colName := range colNames {
					colName = strings.TrimSpace(colName)
					if idx := strings.Index(strings.ToUpper(colName), " AS "); idx != -1 {
						colName = colName[:idx]
					}
					if idx := strings.Index(colName, "."); idx != -1 {
						colName = colName[idx+1:]
					}

					query.Columns = append(query.Columns, &QueryColumn{
						Name:  colName,
						Type:  "string",
						Table: tableName,
					})
				}
			} else {
				query.Columns = append(query.Columns, &QueryColumn{
					Name:  "*",
					Type:  "string",
					Table: tableName,
				})
			}
		}
	}
}

func (g *Generator) inferParamType(sql string, paramIndex int, table *Table, paramName string) string {
	// If we have a parameter name, look it up in the table
	if paramName != "" && paramName != fmt.Sprintf("param%d", paramIndex) {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, paramName) {
				return g.mapSQLTypeToJS(col.Type)
			}
		}
	}
	
	// Try pattern matching as fallback
	whereRegex := regexp.MustCompile(fmt.Sprintf(`(?i)WHERE\s+(\w+)\s*=\s*\$%d`, paramIndex))
	if match := whereRegex.FindStringSubmatch(sql); len(match) > 1 {
		colName := match[1]
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, colName) {
				return g.mapSQLTypeToJS(col.Type)
			}
		}
	}

	if strings.Contains(strings.ToUpper(sql), "INSERT") {
		insertRegex := regexp.MustCompile(`(?i)INSERT\s+INTO\s+\w+\s*\(([\s\S]*?)\)\s*VALUES`)
		if match := insertRegex.FindStringSubmatch(sql); len(match) > 1 {
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

	setRegex := regexp.MustCompile(fmt.Sprintf(`(?i)SET\s+(\w+)\s*=\s*\$%d`, paramIndex))
	if match := setRegex.FindStringSubmatch(sql); len(match) > 1 {
		colName := match[1]
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, colName) {
				return g.mapSQLTypeToJS(col.Type)
			}
		}
	}

	return "any"
}

func (g *Generator) inferParamName(sql string, paramIndex int, table *Table) string {
	whereRegex := regexp.MustCompile(fmt.Sprintf(`(?i)WHERE\s+(\w+)\s*=\s*\$%d`, paramIndex))
	if match := whereRegex.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	if strings.Contains(strings.ToUpper(sql), "INSERT") {
		insertRegex := regexp.MustCompile(`(?i)INSERT\s+INTO\s+\w+\s*\(([\s\S]*?)\)\s*VALUES`)
		if match := insertRegex.FindStringSubmatch(sql); len(match) > 1 {
			colNames := strings.Split(match[1], ",")
			if paramIndex <= len(colNames) {
				return strings.TrimSpace(colNames[paramIndex-1])
			}
		}
	}

	setRegex := regexp.MustCompile(fmt.Sprintf(`(?i)SET\s+(\w+)\s*=\s*\$%d`, paramIndex))
	if match := setRegex.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	return fmt.Sprintf("param%d", paramIndex)
}
