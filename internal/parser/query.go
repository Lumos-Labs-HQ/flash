package parser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
		cwd, _ := os.Getwd()
		queriesPath = filepath.Join(cwd, queriesPath)
	}

	files, err := filepath.Glob(filepath.Join(queriesPath, "*.sql"))
	if err != nil {
		return nil, err
	}

	queries := make([]*Query, 0, len(files)*4)
	for _, file := range files {
		fileQueries, err := p.parseQueryFile(file, schema)
		if err != nil {
			return nil, err
		}
		queries = append(queries, fileQueries...)
	}

	return queries, nil
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

	var table *Table
	for _, t := range schema.Tables {
		if strings.EqualFold(t.Name, tableName) {
			table = t
			break
		}
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

	hasJoin := strings.Contains(strings.ToUpper(query.SQL), "JOIN")
	hasUnion := strings.Contains(strings.ToUpper(query.SQL), "UNION")

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

				for i, line := range lines {
					if strings.Contains(strings.ToUpper(line), strings.ToUpper(queryCol.Name)) {
						lineNum = i + 1
						upperLine := strings.ToUpper(line)
						upperCol := strings.ToUpper(queryCol.Name)
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

		// Look up in schema
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
				if strings.Contains(strings.ToUpper(subMatches[0]), "MAX") || strings.Contains(strings.ToUpper(subMatches[0]), "MIN") {
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
