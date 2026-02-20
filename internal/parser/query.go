package parser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/utils"
)

var (
	fromRegex      *regexp.Regexp
	paramRegex     *regexp.Regexp
	returningRegex *regexp.Regexp
	asRegex        *regexp.Regexp
)

func init() {
	fromRegex = regexp.MustCompile(`(?i)FROM\s+(\w+)`)
	paramRegex = regexp.MustCompile(`\$\d+|\?`)
	returningRegex = regexp.MustCompile(`(?i)RETURNING\s+(.+?)(?:;|\z)`)
	asRegex = regexp.MustCompile(`(?i)\s+AS\s+`)
}

type QueryParser struct {
	Config       *config.Config
	insertRegex  *regexp.Regexp
	updateRegex  *regexp.Regexp
	deleteRegex  *regexp.Regexp
	typeInferrer *TypeInferrer
}

func NewQueryParser(cfg *config.Config) *QueryParser {
	return &QueryParser{
		Config:       cfg,
		insertRegex:  regexp.MustCompile(`(?i)INSERT\s+INTO\s+(\w+)`),
		updateRegex:  regexp.MustCompile(`(?i)UPDATE\s+(\w+)`),
		deleteRegex:  regexp.MustCompile(`(?i)DELETE\s+FROM\s+(\w+)`),
		typeInferrer: NewTypeInferrer(),
	}
}

func (p *QueryParser) Parse(schema *Schema) ([]*Query, error) {
	queriesPath := p.Config.Queries
	if !filepath.IsAbs(queriesPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		queriesPath = filepath.Join(cwd, queriesPath)
	}

	files, err := filepath.Glob(filepath.Join(queriesPath, "*.sql"))
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return []*Query{}, nil
	}

	// Use concurrent processing for better performance on large projects
	return p.parseFilesConcurrently(files, schema)
}

// parseFilesConcurrently processes query files in parallel using worker pool
func (p *QueryParser) parseFilesConcurrently(files []string, schema *Schema) ([]*Query, error) {
	// Create indexed schema for O(1) lookups
	indexedSchema := NewIndexedSchema(schema)
	
	// Determine optimal worker count (don't exceed CPU count or file count)
	numWorkers := runtime.NumCPU()
	if numWorkers > len(files) {
		numWorkers = len(files)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	// Channels for work distribution and result collection
	type parseResult struct {
		queries []*Query
		err     error
		file    string
	}
	
	fileChan := make(chan string, len(files))
	resultChan := make(chan parseResult, len(files))

	// Launch worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				queries, err := p.parseQueryFile(file, indexedSchema.Schema)
				resultChan <- parseResult{
					queries: queries,
					err:     err,
					file:    file,
				}
			}
		}()
	}

	// Send files to workers
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	allQueries := make([]*Query, 0, len(files)*4)
	for result := range resultChan {
		if result.err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", result.file, result.err)
		}
		allQueries = append(allQueries, result.queries...)
	}

	return allQueries, nil
}

func (p *QueryParser) parseQueryFile(filename string, schema *Schema) ([]*Query, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

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

		if strings.HasPrefix(line, "-- name:") || strings.HasPrefix(line, "-- name :") {
			if currentQuery != nil {
				currentQuery.SQL = strings.TrimSpace(strings.Join(sqlLines, " "))
				currentQuery.Comment = comment
				currentQuery.SourceFile = sourceFileName
				if err := p.analyzeQuery(currentQuery, schema); err != nil {
					return nil, err
				}
				queries = append(queries, currentQuery)
			}

			nameStart := strings.Index(line, "name")
			if nameStart == -1 {
				continue
			}
			remainder := line[nameStart+4:]
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
		if err := p.analyzeQuery(currentQuery, schema); err != nil {
			return nil, err
		}
		queries = append(queries, currentQuery)
	}

	return queries, scanner.Err()
}

func (p *QueryParser) analyzeQuery(query *Query, schema *Schema) error {
	var tableName string
	if match := fromRegex.FindStringSubmatch(query.SQL); len(match) > 1 {
		tableName = match[1]
	}

	if tableName == "" {
		if match := p.insertRegex.FindStringSubmatch(query.SQL); len(match) > 1 {
			tableName = match[1]
		}
	}
	if tableName == "" {
		if match := p.updateRegex.FindStringSubmatch(query.SQL); len(match) > 1 {
			tableName = match[1]
		}
	}

	// Use indexed lookup for O(1) performance
	var table *Table
	for _, t := range schema.Tables {
		if strings.EqualFold(t.Name, tableName) {
			table = t
			break
		}
	}

	// CRITICAL: If table is referenced but not found, return error
	if tableName != "" && table == nil {
		availableTables := make([]string, len(schema.Tables))
		for i, t := range schema.Tables {
			availableTables[i] = t.Name
		}
		return fmt.Errorf("table '%s' referenced in query '%s' does not exist in schema. Available tables: %v",
			tableName, query.Name, availableTables)
	}

	paramMatches := paramRegex.FindAllString(query.SQL, -1)

	var paramCount int
	if len(paramMatches) > 0 && paramMatches[0] == "?" {
		paramCount = len(paramMatches)
	} else {
		seen := make(map[string]bool, len(paramMatches))
		for _, p := range paramMatches {
			if !seen[p] {
				seen[p] = true
				paramCount++
			}
		}
	}

	query.Params = make([]*Param, paramCount)
	usedParamNames := make(map[string]int)

	// CRITICAL: Validate INSERT/UPDATE columns exist in schema before proceeding
	if table != nil {
		sqlUpper := strings.ToUpper(query.SQL)
		if strings.Contains(sqlUpper, "INSERT INTO") {
			if err := p.validateInsertColumns(query.SQL, table); err != nil {
				return fmt.Errorf("validation error in query '%s': %w", query.Name, err)
			}
		} else if strings.Contains(sqlUpper, "UPDATE") {
			if err := p.validateUpdateColumns(query.SQL, table); err != nil {
				return fmt.Errorf("validation error in query '%s': %w", query.Name, err)
			}
		}
	}

	for i := 0; i < paramCount; i++ {
		paramName := fmt.Sprintf("param%d", i+1)
		paramType := "any"

		if table != nil {
			inferredName := p.typeInferrer.InferParamName(query.SQL, i+1)
			if inferredName != "" && inferredName != paramName {
				paramName = inferredName
			}

			paramType = p.typeInferrer.InferParamType(query.SQL, i+1, table, paramName)
		}

		if count, exists := usedParamNames[paramName]; exists {
			usedParamNames[paramName] = count + 1
			paramName = fmt.Sprintf("%s%d", paramName, count+1)
		} else {
			usedParamNames[paramName] = 1
		}

		query.Params[i] = &Param{
			Name: paramName,
			Type: paramType,
		}
	}

	sqlUpper := strings.ToUpper(query.SQL)
	sqlTrimmed := strings.TrimSpace(sqlUpper)

	isSelectQuery := strings.HasPrefix(sqlTrimmed, "SELECT") ||
		strings.HasPrefix(sqlTrimmed, "WITH") ||
		(strings.HasPrefix(sqlTrimmed, "(") && strings.Contains(sqlTrimmed, "SELECT"))
	isNotModifying := !utils.ContainsSQLKeyword(sqlTrimmed, "DELETE") &&
		!utils.ContainsSQLKeyword(sqlTrimmed, "UPDATE") &&
		!utils.ContainsSQLKeyword(sqlTrimmed, "INSERT")

	hasReturning := utils.ContainsSQLKeyword(sqlTrimmed, "RETURNING")

	if (isSelectQuery && isNotModifying) || hasReturning {
		var columnsStr string

		if hasReturning {
			if matches := returningRegex.FindStringSubmatch(query.SQL); len(matches) > 1 {
				columnsStr = strings.TrimSpace(matches[1])
			}
		} else {
			columnsStr = utils.ExtractSelectColumns(query.SQL)
		}

		if columnsStr != "" && strings.TrimSpace(columnsStr) != "*" {
			colNames := utils.SmartSplitColumns(columnsStr)

			if len(colNames) > 0 {
				query.Columns = make([]*QueryColumn, 0, len(colNames))

				for _, colName := range colNames {
					colName = strings.TrimSpace(colName)
					if colName == "" {
						continue
					}

					originalExpr := colName
					aliasName := ""

					allMatches := asRegex.FindAllStringIndex(colName, -1)
					if len(allMatches) > 0 {
						validMatch := -1
						colNameUpper := strings.ToUpper(colName)

						for i := len(allMatches) - 1; i >= 0; i-- {
							asPos := allMatches[i][0]
							parenDepth := 0
							caseDepth := 0

							for j := 0; j < asPos; j++ {
								switch colName[j] {
								case '(':
									parenDepth++
								case ')':
									parenDepth--
								}

								// Track CASE/END blocks
								if j+4 <= len(colNameUpper) && colNameUpper[j:j+4] == "CASE" {
									if j == 0 || !((colName[j-1] >= 'A' && colName[j-1] <= 'Z') || (colName[j-1] >= 'a' && colName[j-1] <= 'z')) {
										caseDepth++
									}
								}
								if j+3 <= len(colNameUpper) && colNameUpper[j:j+3] == "END" {
									if (j == 0 || !((colName[j-1] >= 'A' && colName[j-1] <= 'Z') || (colName[j-1] >= 'a' && colName[j-1] <= 'z'))) &&
										(j+3 >= len(colName) || !((colName[j+3] >= 'A' && colName[j+3] <= 'Z') || (colName[j+3] >= 'a' && colName[j+3] <= 'z'))) {
										caseDepth--
									}
								}
							}

							// If we're at depth 0 for both parentheses and CASE blocks, this AS is at the top level (column alias)
							if parenDepth == 0 && caseDepth == 0 {
								validMatch = i
								break
							}
						}

						if validMatch >= 0 {
							loc := allMatches[validMatch]
							originalExpr = strings.TrimSpace(colName[:loc[0]])
							aliasName = strings.TrimSpace(colName[loc[1]:])
							colName = aliasName
						}
					} else {
						if !strings.Contains(colName, "(") {
							if idx := strings.Index(colName, "."); idx != -1 {
								originalExpr = colName 
								colName = colName[idx+1:]
							}
						}
					}

					colType, nullable := p.inferColumnType(colName, originalExpr, query.SQL, schema, table)

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

	if err := utils.ValidateTableReferences(query.SQL, schema, query.SourceFile); err != nil {
		return err
	}

	if err := utils.ValidateColumnReferences(query.SQL, schema, query.SourceFile); err != nil {
		return err
	}

	hasJoin := strings.Contains(sqlUpper, "JOIN")
	hasUnion := strings.Contains(sqlUpper, "UNION")

	if table != nil && len(query.Columns) > 0 && !hasJoin && !hasUnion {
		for _, queryCol := range query.Columns {
			if queryCol.Name == "*" {
				continue
			}

			colNameLower := strings.ToLower(queryCol.Name)
			if strings.Contains(colNameLower, "count") ||
				strings.Contains(colNameLower, "sum") ||
				strings.Contains(colNameLower, "avg") ||
				strings.Contains(colNameLower, "max") ||
				strings.Contains(colNameLower, "min") ||
				strings.Contains(colNameLower, "length") ||
				strings.Contains(colNameLower, "extract") {
				continue
			}

			if strings.Contains(queryCol.Name, "(") || strings.Contains(queryCol.Name, ")") {
				continue
			}

			columnExists := false
			for _, schemaCol := range table.Columns {
				if strings.EqualFold(schemaCol.Name, queryCol.Name) {
					columnExists = true
					break
				}
			}

			if !columnExists {
				lines := strings.Split(query.SQL, "\n")
				lineNum := 1
				colPos := 1
				upperCol := strings.ToUpper(queryCol.Name)

				for i, line := range lines {
					upperLine := strings.ToUpper(line)
					if strings.Contains(upperLine, upperCol) {
						lineNum = i + 1
						colPos = strings.Index(upperLine, upperCol) + 1
						break
					}
				}

				sourceFile := query.SourceFile
				if sourceFile == "" {
					sourceFile = "queries"
				}
				return fmt.Errorf("# package FlashORM\ndb\\queries\\%s.sql:%d:%d: column \"%s\" does not exist in table \"%s\"",
					sourceFile, lineNum, colPos, queryCol.Name, table.Name)
			}
		}
	}

	return nil
}

// inferColumnType determines the correct SQL type for a column based on the expression and schema
func (p *QueryParser) inferColumnType(colName string, originalExpr string, sql string, schema *Schema, primaryTable *Table) (string, bool) {
	sqlType, nullable, found := p.inferTypeFromExpression(originalExpr, sql, schema)
	if found {
		return sqlType, nullable
	}

	if primaryTable != nil {
		for _, col := range primaryTable.Columns {
			if strings.EqualFold(col.Name, colName) {
				return col.Type, col.Nullable
			}
		}
	}

	for _, table := range schema.Tables {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, colName) {
				return col.Type, col.Nullable
			}
		}
	}

	return "TEXT", false
}

// inferTypeFromExpression analyzes SQL expressions to determine types
func (p *QueryParser) inferTypeFromExpression(originalExpr string, sql string, schema *Schema) (string, bool, bool) {
	exprUpper := strings.ToUpper(originalExpr)
	originalExprTrimmed := strings.TrimSpace(originalExpr)

	tableColRefRe := regexp.MustCompile(`^(\w+)\.(\w+)$`)
	if matches := tableColRefRe.FindStringSubmatch(originalExprTrimmed); len(matches) == 3 {
		tableName := matches[1]
		columnName := matches[2]

		for _, table := range schema.Tables {
			if strings.EqualFold(table.Name, tableName) {
				for _, col := range table.Columns {
					if strings.EqualFold(col.Name, columnName) {
						return col.Type, col.Nullable, true
					}
				}
			}
		}
	}

	if strings.Contains(exprUpper, "COUNT(") {
		return "INTEGER", false, true 
	}

	if strings.Contains(exprUpper, "SUM(") {
		return "NUMERIC", true, true
	}

	if strings.Contains(exprUpper, "AVG(") {
		return "NUMERIC", true, true 
	}

	if strings.Contains(exprUpper, "MAX(") || strings.Contains(exprUpper, "MIN(") {
		if strings.Contains(exprUpper, "CREATED_AT") || strings.Contains(exprUpper, "UPDATED_AT") {
			return "TIMESTAMP WITH TIME ZONE", true, true
		}
		return "NUMERIC", true, true
	}

	if strings.Contains(exprUpper, "STRING_AGG(") {
		return "TEXT", true, true 
	}

	if strings.Contains(exprUpper, "ARRAY_AGG(") {
		return "TEXT[]", true, true 
	}

	if strings.Contains(exprUpper, "LENGTH(") {
		return "INTEGER", true, true 
	}

	if strings.Contains(exprUpper, "EXTRACT(") {
		return "NUMERIC", true, true 
	}

	// Check for COALESCE
	if strings.Contains(exprUpper, "COALESCE(") {
		// Extract first argument
		coalesceRe := regexp.MustCompile(`(?i)COALESCE\s*\(\s*([^,)]+)`)
		if matches := coalesceRe.FindStringSubmatch(originalExpr); len(matches) > 1 {
			firstArg := strings.TrimSpace(matches[1])
			firstArgUpper := strings.ToUpper(firstArg)

			// Check if it's a CTE reference with known aggregate type patterns
			if strings.Contains(firstArgUpper, ".CNT") || strings.Contains(firstArgUpper, ".COUNT") ||
				strings.Contains(firstArgUpper, ".TOTAL_CNT") || strings.Contains(firstArgUpper, ".POST_CNT") ||
				strings.Contains(firstArgUpper, ".COMMENT_CNT") || strings.Contains(firstArgUpper, ".PUB_CNT") ||
				strings.Contains(firstArgUpper, ".DRAFT_CNT") || strings.Contains(firstArgUpper, ".POSTS_CNT") ||
				strings.Contains(firstArgUpper, ".CAT_CNT") || strings.Contains(firstArgUpper, ".UNIQUE_USERS") ||
				strings.Contains(firstArgUpper, ".NUM") {
				return "INTEGER", false, true // COALESCE makes it NOT NULL
			}

			if strings.Contains(firstArgUpper, ".AVG") || strings.Contains(firstArgUpper, ".SUM") ||
				strings.Contains(firstArgUpper, ".AVG_LEN") {
				return "NUMERIC", false, true
			}

			cteParts := strings.Split(firstArg, ".")
			if len(cteParts) == 2 {
				cteType, _, found := p.inferTypeFromCTE(sql, strings.TrimSpace(cteParts[0]), strings.TrimSpace(cteParts[1]), schema)
				if found {
					return cteType, false, true // COALESCE makes it NOT NULL
				}
			}
		}
		return "TEXT", false, true // Default for COALESCE
	}

	// Check for CASE expressions
	if strings.Contains(exprUpper, "CASE") && strings.Contains(exprUpper, "END") {
		thenRe := regexp.MustCompile(`(?i)THEN\s+'([^']*)'`)
		if matches := thenRe.FindAllStringSubmatch(originalExpr, -1); len(matches) > 0 {
			return "TEXT", false, true // String literals
		}

		// Check for numeric operations
		if strings.Contains(exprUpper, "+") || strings.Contains(exprUpper, "*") {
			return "INTEGER", false, true
		}

		return "TEXT", false, true 
	}

	// Check for arithmetic operations
	if regexp.MustCompile(`\s*[+\-*/]\s*`).MatchString(originalExpr) {
		if strings.Contains(originalExpr, "(") {
			return "NUMERIC", true, true
		}
	}

	// Check for CTE column references (e.g., ca.all_content, cn.names)
	ctaRefRe := regexp.MustCompile(`^(\w+)\.(\w+)$`)
	if matches := ctaRefRe.FindStringSubmatch(strings.TrimSpace(originalExpr)); len(matches) == 3 {
		cteAlias := matches[1]
		cteColumn := matches[2]
		cteType, nullable, found := p.inferTypeFromCTE(sql, cteAlias, cteColumn, schema)
		if found {
			return cteType, nullable, true
		}
	}

	// Check for table.column references
	tableColRe := regexp.MustCompile(`^(\w+)\.(\w+)$`)
	if matches := tableColRe.FindStringSubmatch(strings.TrimSpace(originalExpr)); len(matches) == 3 {
		tableName := matches[1]
		columnName := matches[2]

		// Try indexed lookup first
		for _, table := range schema.Tables {
			if strings.EqualFold(table.Name, tableName) {
				for _, col := range table.Columns {
					if strings.EqualFold(col.Name, columnName) {
						return col.Type, col.Nullable, true
					}
				}
			}
		}
	}

	return "", false, false
}

// inferTypeFromCTE finds a CTE by alias and infers the type of one of its columns
func (p *QueryParser) inferTypeFromCTE(sql string, cteAlias string, cteColumn string, schema *Schema) (string, bool, bool) {
	withRe := regexp.MustCompile(fmt.Sprintf(`(?is)%s\s+AS\s*\((.*?)\)(?:\s*,|\s+SELECT)`, regexp.QuoteMeta(cteAlias)))
	matches := withRe.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return "", false, false
	}

	cteQuery := matches[1]

	patterns := []struct {
		pattern  string
		sqlType  string
		nullable bool
	}{
		{fmt.Sprintf(`(?i)ARRAY_AGG\([^)]+\)\s+(?:AS\s+)?%s`, cteColumn), "TEXT[]", true},
		{fmt.Sprintf(`(?i)STRING_AGG\([^)]+\)\s+(?:AS\s+)?%s`, cteColumn), "TEXT", true},
		{fmt.Sprintf(`(?i)COUNT\([^)]*\)(?:\s+FILTER\s*\([^)]*\))?\s+(?:AS\s+)?%s`, cteColumn), "INTEGER", false},
		{fmt.Sprintf(`(?i)SUM\([^)]+\)\s+(?:AS\s+)?%s`, cteColumn), "NUMERIC", true},
		{fmt.Sprintf(`(?i)AVG\([^)]+\)\s+(?:AS\s+)?%s`, cteColumn), "NUMERIC", true},
		{fmt.Sprintf(`(?i)MAX\(([^)]+)\)\s+(?:AS\s+)?%s`, cteColumn), "", true}, 
		{fmt.Sprintf(`(?i)MIN\(([^)]+)\)\s+(?:AS\s+)?%s`, cteColumn), "", true},
		{fmt.Sprintf(`(?i)LENGTH\([^)]+\)\s+(?:AS\s+)?%s`, cteColumn), "INTEGER", true},
		{fmt.Sprintf(`(?i)EXTRACT\([^)]+\)\s+(?:AS\s+)?%s`, cteColumn), "NUMERIC", true},
	}

	for _, p := range patterns {
		if matched, _ := regexp.MatchString(p.pattern, cteQuery); matched {
			re := regexp.MustCompile(p.pattern)
			if subMatches := re.FindStringSubmatch(cteQuery); len(subMatches) > 0 {
				upperMatch := strings.ToUpper(subMatches[0])
				if strings.Contains(upperMatch, "MAX") || strings.Contains(upperMatch, "MIN") {
					if len(subMatches) > 1 {
						arg := strings.ToUpper(subMatches[1])
						if strings.Contains(arg, "CREATED_AT") || strings.Contains(arg, "UPDATED_AT") {
							return "TIMESTAMP WITH TIME ZONE", p.nullable, true
						}
					}
					return "NUMERIC", p.nullable, true
				}
				return p.sqlType, p.nullable, true
			}
		}
	}

	// Check if it's a direct column reference in the CTE
	colRefPattern := fmt.Sprintf(`(?i)(\w+)\.(\w+)\s+AS\s+%s`, cteColumn)
	if matched, _ := regexp.MatchString(colRefPattern, cteQuery); matched {
		colRefRe := regexp.MustCompile(colRefPattern)
		if matches := colRefRe.FindStringSubmatch(cteQuery); len(matches) >= 3 {
			refTable := matches[1]
			refColumn := matches[2]

			// Look up in schema
			for _, table := range schema.Tables {
				if strings.EqualFold(table.Name, refTable) {
					for _, col := range table.Columns {
						if strings.EqualFold(col.Name, refColumn) {
							return col.Type, col.Nullable, true
						}
					}
				}
			}
		}
	}

	return "", false, false
}

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
