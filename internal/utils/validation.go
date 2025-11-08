package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateTableReferences checks if tables referenced in queries exist in the schema
func ValidateTableReferences(sql string, schema interface{}) error {
	type Table struct {
		Name string
	}
	type Schema struct {
		Tables []*Table
	}

	s, ok := schema.(*Schema)
	if !ok {
		// Try to convert from parser.Schema
		return nil
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

		tableExists := false
		for _, t := range s.Tables {
			if strings.EqualFold(t.Name, tableName) {
				tableExists = true
				break
			}
		}

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

			return fmt.Errorf("# package FlashORM\ndb\\queries\\users.sql:%d:%d: relation \"%s\" does not exist", lineNum, colPos, tableName)
		}
	}

	if foundTableRefs && len(s.Tables) == 0 {
		return fmt.Errorf("# package flash\ndb\\queries\\users.sql:1:1: no tables found in schema, but query references tables")
	}

	return nil
}

// ValidateColumnReferences checks if columns referenced in queries exist in the schema
func ValidateColumnReferences(sql string, schema interface{}) error {
	type Column struct {
		Name string
	}
	type Table struct {
		Name    string
		Columns []*Column
	}
	type Schema struct {
		Tables []*Table
	}

	s, ok := schema.(*Schema)
	if !ok {
		return nil
	}

	tableAliasPattern := regexp.MustCompile(`(?i)FROM\s+(\w+)\s+(\w+)`)
	joinPattern := regexp.MustCompile(`(?i)JOIN\s+(\w+)\s+(\w+)`)

	aliasToTable := make(map[string]string)

	matches := tableAliasPattern.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			tableName := match[1]
			alias := match[2]
			aliasToTable[alias] = tableName
		}
	}

	matches = joinPattern.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			tableName := match[1]
			alias := match[2]
			aliasToTable[alias] = tableName
		}
	}

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

		tableName := tableOrAlias
		if realTable, ok := aliasToTable[tableOrAlias]; ok {
			tableName = realTable
		}

		var table *Table
		for _, t := range s.Tables {
			if strings.EqualFold(t.Name, tableName) {
				table = t
				break
			}
		}

		if table == nil {
			continue
		}

		columnExists := false
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, columnName) {
				columnExists = true
				break
			}
		}

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

			return fmt.Errorf("# package flash\ndb\\queries\\users.sql:%d:%d: column reference \"%s\" not found", lineNum, colPos, columnName)
		}
	}

	return nil
}
