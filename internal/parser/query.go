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
	cteNameRegex   *regexp.Regexp
	// Pre-compiled for inferTypeFromExpression
	windowFuncRe    *regexp.Regexp
	numericCTEColRe *regexp.Regexp
	pgCastRe        *regexp.Regexp
)

func init() {
	fromRegex = regexp.MustCompile(`(?i)\bFROM\s+([^\s;]+)`)
	paramRegex = regexp.MustCompile(`\$\d+|\?`)
	returningRegex = regexp.MustCompile(`(?i)RETURNING\s+(.+?)(?:;|\z)`)
	asRegex = regexp.MustCompile(`(?i)\s+AS\s+`)
	cteNameRegex = regexp.MustCompile(`(?i)(\w+)\s+AS\s*\(`)
	windowFuncRe = regexp.MustCompile(`(?i)^(ROW_NUMBER|RANK|DENSE_RANK|NTILE|PERCENT_RANK|CUME_DIST|LEAD|LAG|FIRST_VALUE|LAST_VALUE)\s*\(`)
	numericCTEColRe = regexp.MustCompile(`(?i)\.(cnt|count|total|total_posts|published_posts|draft_posts|total_comments|posts_commented_on|categories_used|engagement_score|num|qty|quantity|amount|unique_\w+)`)
	pgCastRe = regexp.MustCompile(`(?i)::[a-zA-Z][a-zA-Z0-9_]*(\([^)]*\))?$`)
}

// stripPGCast removes PostgreSQL cast suffix like ::TEXT or ::NUMERIC(10,2)
func stripPGCast(expr string) string {
	return pgCastRe.ReplaceAllString(strings.TrimSpace(expr), "")
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
		insertRegex:  regexp.MustCompile(`(?i)INSERT\s+INTO\s+([^\s;]+)`),
		updateRegex:  regexp.MustCompile(`(?i)UPDATE\s+([^\s;]+)`),
		deleteRegex:  regexp.MustCompile(`(?i)DELETE\s+FROM\s+([^\s;]+)`),
		typeInferrer: NewTypeInferrer(),
	}
}

func (p *QueryParser) Parse(schema *Schema) ([]*Query, error) {
	// Inject schema into inferrer for cross-table type resolution
	p.typeInferrer = NewTypeInferrerWithSchema(schema)

	queriesPath := p.Config.Queries
	if !filepath.IsAbs(queriesPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		queriesPath = filepath.Join(cwd, queriesPath)
	}

	files, _ := filepath.Glob(filepath.Join(queriesPath, "*.sql"))
	cqlFiles, _ := filepath.Glob(filepath.Join(queriesPath, "*.cql"))
	files = append(files, cqlFiles...)

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
	// Rewrite col IN ($1, $2, ...) → col = ANY($1) with a single array param,
	// renumbering all subsequent params. Only for PostgreSQL-style $N params.
	query.SQL = rewriteINListToANY(query.SQL)

	var tableName string
	// Strip parenthesized content to avoid matching function-internal FROM (e.g. EXTRACT(EPOCH FROM ...))
	cleaned := utils.StripParenthesizedContent(query.SQL)
	if match := fromRegex.FindStringSubmatch(cleaned); len(match) > 1 {
		tableName = stripIdentQuotes(match[1])
	}

	if tableName == "" {
		if match := p.insertRegex.FindStringSubmatch(query.SQL); len(match) > 1 {
			tableName = stripIdentQuotes(match[1])
		}
	}
	if tableName == "" {
		if match := p.updateRegex.FindStringSubmatch(query.SQL); len(match) > 1 {
			tableName = stripIdentQuotes(match[1])
		}
	}

	// Build CTE name set — CTEs are query-local, not tables in schema
	cteNames := make(map[string]bool)
	for _, m := range cteNameRegex.FindAllStringSubmatch(query.SQL, -1) {
		if len(m) > 1 {
			cteNames[strings.ToLower(m[1])] = true
		}
	}

	// If the extracted table name is actually a CTE, skip schema validation
	if cteNames[strings.ToLower(tableName)] {
		tableName = ""
	}

	// Normalize keyspace-qualified names: "ks"."tbl" → ks.tbl, ks.tbl → ks.tbl
	// Match against schema table names which may be keyspace-qualified or plain.
	var table *Table
	for _, t := range schema.Tables {
		if matchesTableName(t.Name, tableName) {
			table = t
			break
		}
	}

	// Fallback: strip keyspace prefix and retry.
	// e.g. query says "myapp.users" but schema has "users" (ScyllaDB single-keyspace mode)
	if table == nil && tableName != "" {
		if dotIdx := strings.LastIndex(tableName, "."); dotIdx >= 0 {
			stripped := tableName[dotIdx+1:]
			for _, t := range schema.Tables {
				if strings.EqualFold(t.Name, stripped) {
					table = t
					break
				}
			}
		}
	}

	// Return an error when a referenced table is missing from the schema.
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
	// Extract ordered actual param numbers from the SQL so we map
	orderedParamNums := extractOrderedParamNums(query.SQL)

	// Validate INSERT/UPDATE columns exist in the schema.
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
		// Use the actual $N number from the SQL if available,
		// falling back to i+1 for ?-style parameters.
		paramNum := i + 1
		if i < len(orderedParamNums) && orderedParamNums[i] > 0 {
			paramNum = orderedParamNums[i]
		}
		paramName := fmt.Sprintf("param%d", i+1)
		paramType := "any"

		// Infer param name from SQL regardless of table availability
		inferredName := p.typeInferrer.InferParamName(query.SQL, paramNum)
		if inferredName != "" && inferredName != paramName {
			paramName = inferredName
		}

		if table != nil {
			paramType = p.typeInferrer.InferParamType(query.SQL, paramNum, table, paramName)
		}

		if count, exists := usedParamNames[paramName]; exists {
			usedParamNames[paramName] = count + 1
			paramName = fmt.Sprintf("%s%d", paramName, count+1)
		} else {
			usedParamNames[paramName] = 1
		}

		query.Params[i] = &Param{
			Name:     paramName,
			Type:     paramType,
			ParamNum: paramNum,
		}
	}

	// Renumber $N placeholders to sequential $1, $2, ... so generated
	// when $1 is absent from the query.
	if len(orderedParamNums) > 0 {
		query.SQL = renumberParams(query.SQL, orderedParamNums)
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

					// isComputed: true if the original expression involves anything
					// beyond a simple column reference (function calls, operators, etc.)
					bareRefRe := regexp.MustCompile(`^\w+(\.\w+)?$`)
					isComputed := originalExpr != "" && !bareRefRe.MatchString(originalExpr)

					query.Columns = append(query.Columns, &QueryColumn{
						Name:         colName,
						Type:         colType,
						Table:        tableName,
						Nullable:     nullable,
						IsComputed:   isComputed,
						OriginalExpr: originalExpr,
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

	hasJoin := strings.Contains(sqlUpper, "JOIN")
	hasUnion := strings.Contains(sqlUpper, "UNION")
	hasAggregate := strings.Contains(sqlUpper, "GROUP BY") ||
		strings.Contains(sqlUpper, "COUNT(") ||
		strings.Contains(sqlUpper, "SUM(") ||
		strings.Contains(sqlUpper, "AVG(") ||
		strings.Contains(sqlUpper, "MAX(") ||
		strings.Contains(sqlUpper, "MIN(") ||
		strings.Contains(sqlUpper, " FILTER ") ||
		strings.Contains(sqlUpper, "OVER(") ||
		strings.Contains(sqlUpper, " OVER ")

	// Table references are always validated — catches real typos.
	// Column *reference* validation (qualified refs + WHERE clause) runs for all queries.
	// Only the return-column existence check (alias vs schema) is skipped for complex queries
	// because aggregates/window functions/CTEs produce computed aliases not in any table.
	if err := utils.ValidateTableReferences(query.SQL, schema, query.SourceFile); err != nil {
		return err
	}
	if err := utils.ValidateColumnReferences(query.SQL, schema, query.SourceFile); err != nil {
		return err
	}

	if table != nil && len(query.Columns) > 0 && !hasJoin && !hasUnion && !hasAggregate {
		for _, queryCol := range query.Columns {
			if queryCol.Name == "*" {
				continue
			}

			if strings.Contains(queryCol.Name, "(") || strings.Contains(queryCol.Name, ")") {
				continue
			}

			// Skip column existence check when alias differs from the original
			// expression — e.g. "id AS post_id", "preferences->'key' AS pref_value"
			if queryCol.IsComputed || (queryCol.OriginalExpr != "" && !strings.EqualFold(queryCol.Name, queryCol.OriginalExpr) &&
				!strings.HasSuffix(strings.ToLower(queryCol.OriginalExpr), "."+strings.ToLower(queryCol.Name))) {
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

	// Check CTE aliased columns — bare name doesn't match primaryTable but
	// might resolve through a CTE in the SQL
	if bareColType, bareNullable, bareFound := p.inferBareCTEColumnType(sql, colName, schema); bareFound {
		return bareColType, bareNullable
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
	// Strip PostgreSQL cast suffix (e.g. ::NUMERIC(10,2) or ::TEXT) before analysis
	expr := stripPGCast(originalExpr)
	exprUpper := strings.ToUpper(expr)
	exprTrimmed := strings.TrimSpace(expr)

	tableColRefRe := regexp.MustCompile(`^(\w+)\.(\w+)$`)
	if matches := tableColRefRe.FindStringSubmatch(exprTrimmed); len(matches) == 3 {
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
		// Resolve via table alias (e.g. "u" → "users")
		for _, table := range schema.Tables {
			if strings.HasPrefix(strings.ToLower(table.Name), strings.ToLower(tableName)) ||
				(len(tableName) == 1 && strings.HasPrefix(strings.ToLower(table.Name), strings.ToLower(tableName))) {
				for _, col := range table.Columns {
					if strings.EqualFold(col.Name, columnName) {
						return col.Type, col.Nullable, true
					}
				}
			}
		}
		// Try CTE resolution (e.g. "ps.total_views" where "ps" is a CTE alias)
		cteType, nullable, found := p.inferTypeFromCTE(sql, tableName, columnName, schema)
		if found {
			return cteType, nullable, true
		}
	}

	// Window functions → INTEGER (ROW_NUMBER, RANK, DENSE_RANK, etc.)
	if windowFuncRe.MatchString(exprUpper) {
		return "INTEGER", false, true
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
		if strings.Contains(exprUpper, "_AT") || strings.Contains(exprUpper, "_DATE") {
			return "TIMESTAMP WITH TIME ZONE", true, true
		}
		return "NUMERIC", true, true
	}

	// STRING_AGG / ARRAY_AGG — use HasPrefix check so ORDER BY inside doesn't break it
	if strings.HasPrefix(exprUpper, "STRING_AGG(") || strings.Contains(exprUpper, "STRING_AGG(") {
		return "TEXT", true, true
	}
	if strings.HasPrefix(exprUpper, "ARRAY_AGG(") || strings.Contains(exprUpper, "ARRAY_AGG(") {
		return "TEXT[]", true, true
	}

	// ARRAY_LENGTH — check before generic LENGTH to avoid false match
	if strings.Contains(exprUpper, "ARRAY_LENGTH(") {
		return "INTEGER", false, true
	}
	if strings.Contains(exprUpper, "LENGTH(") {
		return "INTEGER", true, true
	}
	if strings.Contains(exprUpper, "EXTRACT(") {
		return "NUMERIC", true, true
	}
	if strings.Contains(exprUpper, "NULLIF(") {
		return "TEXT", true, true
	}
	// ROUND(x, d) → NUMERIC
	if strings.HasPrefix(exprUpper, "ROUND(") {
		return "NUMERIC", true, true
	}
	// TS_RANK → REAL (numeric)
	if strings.Contains(exprUpper, "TS_RANK(") {
		return "NUMERIC", true, true
	}
	// COALESCE(agg, literal) — common pattern: COALESCE(SUM(...), 0)
	if strings.Contains(exprUpper, "COALESCE(") {
		// Check if first arg is an aggregate
		if strings.Contains(exprUpper, "COALESCE(SUM(") || strings.Contains(exprUpper, "COALESCE(AVG(") {
			return "NUMERIC", true, true
		}
		if strings.Contains(exprUpper, "COALESCE(COUNT(") {
			return "INTEGER", true, true
		}
		if strings.Contains(exprUpper, "COALESCE(MAX(") || strings.Contains(exprUpper, "COALESCE(MIN(") {
			if strings.Contains(exprUpper, "_AT") || strings.Contains(exprUpper, "_DATE") {
				return "TIMESTAMP WITH TIME ZONE", true, true
			}
			return "NUMERIC", true, true
		}
	}
	// Subquery expression: (SELECT agg(...) FROM ...)
	if strings.HasPrefix(exprTrimmed, "(") && strings.Contains(exprUpper, "SELECT") {
		if strings.Contains(exprUpper, "COUNT(") {
			return "INTEGER", true, true
		}
		if strings.Contains(exprUpper, "SUM(") || strings.Contains(exprUpper, "AVG(") {
			return "NUMERIC", true, true
		}
		return "TEXT", true, true
	}

	if strings.Contains(exprUpper, "COALESCE(") {
		coalesceRe := regexp.MustCompile(`(?i)COALESCE\s*\(\s*([^,)]+)`)
		if matches := coalesceRe.FindStringSubmatch(expr); len(matches) > 1 {
			firstArg := strings.TrimSpace(matches[1])
			firstArgUpper := strings.ToUpper(firstArg)

			// Numeric CTE alias columns
			if numericCTEColRe.MatchString(firstArgUpper) {
				return "INTEGER", false, true
			}
			if strings.Contains(firstArgUpper, ".AVG") || strings.Contains(firstArgUpper, ".SUM") ||
				strings.Contains(firstArgUpper, ".AVG_") {
				return "NUMERIC", false, true
			}

			cteParts := strings.Split(firstArg, ".")
			if len(cteParts) == 2 {
				cteType, _, found := p.inferTypeFromCTE(sql, strings.TrimSpace(cteParts[0]), strings.TrimSpace(cteParts[1]), schema)
				if found {
					return cteType, false, true
				}
			}
		}
		return "TEXT", false, true
	}
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

	// Check for CTE column references (e.g., ups.total_posts, ucs.last_comment_date)
	ctaRefRe := regexp.MustCompile(`^(\w+)\.(\w+)$`)
	if matches := ctaRefRe.FindStringSubmatch(exprTrimmed); len(matches) == 3 {
		cteAlias := matches[1]
		cteColumn := matches[2]
		// Try CTE lookup first (handles aliases like ups → user_post_stats)
		cteType, nullable, found := p.inferTypeFromCTE(sql, cteAlias, cteColumn, schema)
		if found {
			return cteType, nullable, true
		}
		// Fall through to real table lookup below
	}

	// Try resolving bare column names through CTEs
	// e.g. "base_score" in a query that has "user_scores" CTE with "u.score AS base_score"
	if bareCol := strings.TrimSpace(exprTrimmed); bareCol != "" && !strings.Contains(bareCol, ".") &&
		!strings.Contains(bareCol, "(") && !strings.Contains(bareCol, " ") {
		cteType, nullable, found := p.inferBareCTEColumnType(sql, bareCol, schema)
		if found {
			return cteType, nullable, true
		}
	}

	// Check for table.column references against real schema tables
	tableColRe := regexp.MustCompile(`^(\w+)\.(\w+)$`)
	if matches := tableColRe.FindStringSubmatch(exprTrimmed); len(matches) == 3 {
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

	return "", false, false
}

// inferBareCTEColumnType resolves a bare column name (no table prefix) by scanning
// all CTE aliases in the SQL. For each CTE alias, it tries to find the column name
// in that CTE's body and resolve its type back to the source table.
func (p *QueryParser) inferBareCTEColumnType(sql string, columnName string, schema *Schema) (string, bool, bool) {
	// Find all CTE aliases: "alias AS ("
	cteNameRe := regexp.MustCompile(`(?i)(\w+)\s+AS\s*\(`)
	cteMatches := cteNameRe.FindAllStringSubmatch(sql, -1)
	for _, m := range cteMatches {
		if len(m) < 2 {
			continue
		}
		cteAlias := m[1]
		if utils.IsSQLKeyword(cteAlias) {
			continue
		}
		t, n, ok := p.inferTypeFromCTE(sql, cteAlias, columnName, schema)
		if ok {
			return t, n, true
		}
	}
	return "", false, false
}

// inferTypeFromCTE finds a CTE by alias or name and infers the type of one of its columns.
func (p *QueryParser) inferTypeFromCTE(sql string, cteAlias string, cteColumn string, schema *Schema) (string, bool, bool) {
	// Try direct match: "cteAlias AS (...)"
	if t, n, ok := p.inferTypeFromCTEBody(sql, cteAlias, cteColumn, schema); ok {
		return t, n, ok
	}

	// Resolve outer alias → CTE name via "cteName alias" in FROM/JOIN
	// e.g. "FROM user_post_stats ups" → alias "ups" → CTE "user_post_stats"
	aliasRe := regexp.MustCompile(`(?i)(?:FROM|JOIN)\s+(\w+)\s+` + regexp.QuoteMeta(cteAlias) + `\b`)
	if m := aliasRe.FindStringSubmatch(sql); len(m) > 1 {
		if t, n, ok := p.inferTypeFromCTEBody(sql, m[1], cteColumn, schema); ok {
			return t, n, ok
		}
	}

	return "", false, false
}

// inferTypeFromCTEBody searches for cteName AS (...) and infers the column type from the CTE body.
func (p *QueryParser) inferTypeFromCTEBody(sql string, cteName string, cteColumn string, schema *Schema) (string, bool, bool) {
	// Find "cteName AS (" position
	searchRe := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(cteName) + `\s+AS\s*\(`)
	loc := searchRe.FindStringIndex(sql)
	if loc == nil {
		return "", false, false
	}
	// Extract balanced content between the opening ( and its matching )
	openPos := strings.Index(sql[loc[1]-1:], "(") + loc[1] - 1
	cteQuery := extractBalancedParens(sql, openPos)
	if cteQuery == "" {
		return "", false, false
	}

	// Match aggregate functions — use [^,)]+ but allow balanced parens via suffix match
	// Using a broader pattern that handles ORDER BY inside aggregates
	aggPatterns := []struct {
		re       *regexp.Regexp
		sqlType  string
		nullable bool
	}{
		{regexp.MustCompile(fmt.Sprintf(`(?i)COUNT\([^)]*\)(?:\s+FILTER\s*\([^)]*\))?\s+(?:AS\s+)?%s\b`, cteColumn)), "INTEGER", false},
		{regexp.MustCompile(fmt.Sprintf(`(?i)SUM\([^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "NUMERIC", true},
		{regexp.MustCompile(fmt.Sprintf(`(?i)AVG\([^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "NUMERIC", true},
		{regexp.MustCompile(fmt.Sprintf(`(?i)LENGTH\([^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "INTEGER", true},
		{regexp.MustCompile(fmt.Sprintf(`(?i)EXTRACT\([^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "NUMERIC", true},
		// STRING_AGG and ARRAY_AGG may contain ORDER BY inside — match up to the AS alias
		{regexp.MustCompile(fmt.Sprintf(`(?i)STRING_AGG\b.+?\)\s+(?:AS\s+)?%s\b`, cteColumn)), "TEXT", true},
		{regexp.MustCompile(fmt.Sprintf(`(?i)ARRAY_AGG\b.+?\)\s+(?:AS\s+)?%s\b`, cteColumn)), "TEXT[]", true},
		// COALESCE-wrapped aggregates: COALESCE(SUM(...), 0) AS col
		{regexp.MustCompile(fmt.Sprintf(`(?i)COALESCE\s*\(\s*SUM\([^)]*\)[^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "NUMERIC", true},
		{regexp.MustCompile(fmt.Sprintf(`(?i)COALESCE\s*\(\s*AVG\([^)]*\)[^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "NUMERIC", true},
		{regexp.MustCompile(fmt.Sprintf(`(?i)COALESCE\s*\(\s*COUNT\([^)]*\)[^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "INTEGER", false},
		// ROUND(x, d) AS col
		{regexp.MustCompile(fmt.Sprintf(`(?i)ROUND\([^)]+\)\s+(?:AS\s+)?%s\b`, cteColumn)), "NUMERIC", true},
		// Integer literal: 0 AS depth — simple 0 as depth
		{regexp.MustCompile(fmt.Sprintf(`(?i)\b(\d+)\s+(?:AS\s+)?%s\b`, cteColumn)), "INTEGER", false},
		// Arithmetic expression: ct.depth + 1 AS depth
		{regexp.MustCompile(fmt.Sprintf(`(?i)(\w+\.\w+|\w+|\d+)\s*\+\s*\d+\s+(?:AS\s+)?%s\b`, cteColumn)), "INTEGER", false},
		// ARRAY_LENGTH in CTE
		{regexp.MustCompile(fmt.Sprintf(`(?i)ARRAY_LENGTH\([^)]+\)\s+(?:AS\s+)?%s\b`, cteColumn)), "INTEGER", false},
	}

	// Special case: subquery aggregates in CTE bodies
	// (SELECT COUNT(*) FROM ...) as follower_count
	_ = cteColumn
	subQueryAggRe := regexp.MustCompile(`(?i)\(\s*SELECT\s+(COUNT|SUM|AVG)\(`)
	if m := subQueryAggRe.FindStringSubmatch(cteQuery); len(m) > 1 {
		agg := strings.ToUpper(m[1])
		switch agg {
		case "COUNT":
			return "INTEGER", false, true
		case "SUM", "AVG":
			return "NUMERIC", true, true
		}
	}
	// Generic subquery: (SELECT ...) AS col
	if strings.Contains(cteQuery, fmt.Sprintf("AS %s", cteColumn)) ||
		strings.Contains(cteQuery, fmt.Sprintf("as %s", cteColumn)) {
		subMatch := regexp.MustCompile(`(?i)\(\s*SELECT\s+(COUNT|SUM|AVG|MAX|MIN)\(`)
		if sm := subMatch.FindStringSubmatch(cteQuery); len(sm) > 1 {
			agg := strings.ToUpper(sm[1])
			switch agg {
			case "COUNT":
				return "INTEGER", false, true
			case "SUM", "AVG":
				return "NUMERIC", true, true
			case "MAX", "MIN":
				return "TIMESTAMP WITH TIME ZONE", true, true
			}
		}
	}
	// ARRAY_LENGTH in CTE: ARRAY_LENGTH(col, 1) AS tag_count
	if strings.Contains(cteQuery, "ARRAY_LENGTH") &&
		strings.Contains(cteQuery, fmt.Sprintf("AS %s", cteColumn)) {
		return "INTEGER", false, true
	}

	for _, ap := range aggPatterns {
		if ap.re.MatchString(cteQuery) {
			return ap.sqlType, ap.nullable, true
		}
	}

	// MAX/MIN — inherit from argument
	maxMinRe := regexp.MustCompile(fmt.Sprintf(`(?i)(MAX|MIN)\(([^)]+)\)\s+(?:AS\s+)?%s\b`, cteColumn))
	if m := maxMinRe.FindStringSubmatch(cteQuery); len(m) > 2 {
		arg := strings.ToUpper(m[2])
		if strings.Contains(arg, "CREATED_AT") || strings.Contains(arg, "UPDATED_AT") ||
			strings.Contains(arg, "_AT") || strings.Contains(arg, "DATE") {
			return "TIMESTAMP WITH TIME ZONE", true, true
		}
		return "NUMERIC", true, true
	}

	// Direct column reference: table.col AS cteColumn or col AS cteColumn
	colRefRe := regexp.MustCompile(fmt.Sprintf(`(?i)(?:(\w+)\.)?(\w+)\s+[Aa][Ss]\s+%s\b`, cteColumn))
	if m := colRefRe.FindStringSubmatch(cteQuery); len(m) >= 3 {
		refTable := m[1]
		refColumn := m[2]
		for _, table := range schema.Tables {
			// Match by full table name OR by table alias (single letter or short name)
			tableMatches := strings.EqualFold(table.Name, refTable)
			// Also match if refTable is an alias prefix of the table name (e.g. "u" for "users")
			if !tableMatches && refTable != "" {
				tableMatches = strings.HasPrefix(strings.ToLower(table.Name), strings.ToLower(refTable))
			}
			if tableMatches || refTable == "" {
				for _, col := range table.Columns {
					if strings.EqualFold(col.Name, refColumn) {
						return col.Type, col.Nullable, true
					}
				}
			}
		}
	}

	// Bare column in CTE (no alias): SELECT col FROM table — col matches cteColumn
	bareColRe := regexp.MustCompile(fmt.Sprintf(`(?i)(?:^|,|\s)(\w+)\.(%s)\b`, cteColumn))
	if m := bareColRe.FindStringSubmatch(cteQuery); len(m) >= 3 {
		refTable := m[1]
		refColumn := m[2]
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

// stripIdentQuotes removes surrounding double-quotes and backticks from identifiers.
func stripIdentQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '`' && s[len(s)-1] == '`') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// matchesTableName compares a schema table name with a query table reference.
// Handles keyspace-qualified names: "ks"."tbl" = ks.tbl
func matchesTableName(schemaName, queryName string) bool {
	// Exact match
	if strings.EqualFold(schemaName, queryName) {
		return true
	}
	// If the query reference is ks.tbl, extract just the table part and match
	if dotIdx := strings.LastIndex(queryName, "."); dotIdx >= 0 {
		tbl := queryName[dotIdx+1:]
		// Match against plain table name
		if strings.EqualFold(schemaName, tbl) {
			return true
		}
		// Match against ks.tbl form
		if strings.EqualFold(schemaName, queryName) {
			return true
		}
	}
	// Schema name might be ks.tbl, query might be plain tbl
	if dotIdx := strings.LastIndex(schemaName, "."); dotIdx >= 0 {
		tbl := schemaName[dotIdx+1:]
		if strings.EqualFold(tbl, queryName) {
			return true
		}
	}
	return false
}

// getColumnNames returns a list of column names from a table for error messages
func (p *QueryParser) getColumnNames(table *Table) []string {
	var names []string
	for _, col := range table.Columns {
		names = append(names, col.Name)
	}
	return names
}

// extractBalancedParens extracts the content between the opening paren at startPos
// and its matching closing paren, returning the inner content without the parens.
func extractBalancedParens(s string, startPos int) string {
	if startPos >= len(s) || s[startPos] != '(' {
		return ""
	}
	depth := 0
	for i := startPos; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[startPos+1 : i]
			}
		}
	}
	return ""
}

// renumberParams rewrites $N placeholders from original numbers to sequential
func renumberParams(sql string, orderedNums []int) string {
	mapping := make(map[int]int, len(orderedNums))
	for i, orig := range orderedNums {
		mapping[orig] = i + 1
	}
	re := regexp.MustCompile(`\$(\d+)`)
	return re.ReplaceAllStringFunc(sql, func(match string) string {
		var n int
		if _, err := fmt.Sscanf(match[1:], "%d", &n); err != nil {
			return match
		}
		if newNum, ok := mapping[n]; ok {
			return fmt.Sprintf("$%d", newNum)
		}
		return match
	})
}

// extractOrderedParamNums returns deduped ordered $N numbers from SQL.
func extractOrderedParamNums(sql string) []int {
	re := regexp.MustCompile(`\$(\d+)`)
	matches := re.FindAllStringSubmatch(sql, -1)
	seen := make(map[int]bool, len(matches))
	result := make([]int, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 {
			var n int
			if _, err := fmt.Sscanf(m[1], "%d", &n); err != nil {
				continue
			}
			if !seen[n] {
				seen[n] = true
				result = append(result, n)
			}
		}
	}
	return result
}

// rewriteINListToANY rewrites `col IN ($1, $2, $3)` → `col = ANY($1)` and
// renumbers all subsequent $N params so they stay sequential.
func rewriteINListToANY(sql string) string {
	inRe := regexp.MustCompile(`(?i)(\w+)\s+IN\s*\(\s*(\$\d+(?:\s*,\s*\$\d+)*)\s*\)`)
	numRe := regexp.MustCompile(`\$(\d+)`)

	type inSpan struct {
		start, end int
		col        string
		nums       []int // original $N numbers in this IN list
	}

	var spans []inSpan
	for _, loc := range inRe.FindAllStringSubmatchIndex(sql, -1) {
		paramsStr := sql[loc[4]:loc[5]]
		var nums []int
		for _, m := range numRe.FindAllStringSubmatch(paramsStr, -1) {
			var n int
			fmt.Sscanf(m[1], "%d", &n)
			nums = append(nums, n)
		}
		if len(nums) < 2 {
			continue
		}
		spans = append(spans, inSpan{loc[0], loc[1], sql[loc[2]:loc[3]], nums})
	}
	if len(spans) == 0 {
		return sql
	}

	// Build a remapping: original $N → new $N
	// Each IN list keeps only its first param; the rest are removed.
	// Collect all "removed" param numbers in sorted order.
	removed := map[int]bool{}
	for _, s := range spans {
		for _, n := range s.nums[1:] {
			removed[n] = true
		}
	}

	// For each original $N, compute its new number (subtract count of removed nums < N)
	newNum := func(orig int) int {
		shift := 0
		for r := range removed {
			if r < orig {
				shift++
			}
		}
		return orig - shift
	}

	// Replace spans in reverse order (to preserve offsets)
	for i := len(spans) - 1; i >= 0; i-- {
		s := spans[i]
		replacement := fmt.Sprintf("%s = ANY($%d)", s.col, newNum(s.nums[0]))
		sql = sql[:s.start] + replacement + sql[s.end:]
	}

	// Renumber all remaining $N params (high-to-low to avoid collisions)
	for n := 100; n >= 1; n-- {
		if removed[n] {
			continue
		}
		nn := newNum(n)
		if nn != n {
			sql = strings.ReplaceAll(sql, fmt.Sprintf("$%d", n), fmt.Sprintf("$%d", nn))
		}
	}

	return sql
}
