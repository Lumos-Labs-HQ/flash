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
	fromRegex  *regexp.Regexp
	paramRegex *regexp.Regexp
)

func init() {
	fromRegex = regexp.MustCompile(`(?i)FROM\s+(\w+)`)
	paramRegex = regexp.MustCompile(`\$\d+|\?`)
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

	// For PostgreSQL-style ($1, $2), we need unique params
	// For SQLite-style (?), each ? is a separate parameter
	var paramCount int
	if len(paramMatches) > 0 && paramMatches[0] == "?" {
		// SQLite style - count all occurrences
		paramCount = len(paramMatches)
	} else {
		// PostgreSQL style - count unique $n
		seen := make(map[string]bool, len(paramMatches))
		for _, p := range paramMatches {
			if !seen[p] {
				seen[p] = true
				paramCount++
			}
		}
	}

	query.Params = make([]*Param, paramCount)
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

		query.Params[i] = &Param{
			Name: paramName,
			Type: paramType,
		}
	}

	sqlUpper := strings.ToUpper(query.SQL)
	sqlTrimmed := strings.TrimSpace(sqlUpper)

	isSelectQuery := strings.HasPrefix(sqlTrimmed, "SELECT") || strings.HasPrefix(sqlTrimmed, "WITH")
	isNotModifying := !utils.ContainsSQLKeyword(sqlTrimmed, "DELETE") &&
		!utils.ContainsSQLKeyword(sqlTrimmed, "UPDATE") &&
		!utils.ContainsSQLKeyword(sqlTrimmed, "INSERT")

	hasReturning := utils.ContainsSQLKeyword(sqlTrimmed, "RETURNING")

	// Extract columns from SELECT queries or RETURNING clauses
	if (isSelectQuery && isNotModifying) || hasReturning {
		var columnsStr string

		if hasReturning {
			// Extract columns from RETURNING clause
			returningRegex := regexp.MustCompile(`(?i)RETURNING\s+(.+?)(?:;|\z)`)
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

	if err := utils.ValidateTableReferences(query.SQL, schema, query.SourceFile); err != nil {
		return err
	}

	if err := utils.ValidateColumnReferences(query.SQL, schema, query.SourceFile); err != nil {
		return err
	}

	// Validate that SELECT columns exist in schema
	// Skip this for queries with JOINs since they span multiple tables and use qualified columns
	hasJoin := strings.Contains(strings.ToUpper(query.SQL), "JOIN")

	if table != nil && len(query.Columns) > 0 && !hasJoin {
		for _, queryCol := range query.Columns {
			if queryCol.Name == "*" {
				continue
			}

			// Skip aggregate functions and expressions
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

			// Skip if it contains parentheses (function call or expression)
			if strings.Contains(queryCol.Name, "(") || strings.Contains(queryCol.Name, ")") {
				continue
			}

			// Check if column exists in table
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
