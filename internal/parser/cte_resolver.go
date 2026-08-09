package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/utils"
)

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
		{regexp.MustCompile(fmt.Sprintf(`(?i)COUNT\([^)]*\)(?:\s+FILTER\s*\([^)]*\))?\s+(?:AS\s+)?%s\b`, cteColumn)), "BIGINT", false},
		{regexp.MustCompile(fmt.Sprintf(`(?i)SUM\([^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "BIGINT", true},
		{regexp.MustCompile(fmt.Sprintf(`(?i)AVG\([^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "NUMERIC", true},
		{regexp.MustCompile(fmt.Sprintf(`(?i)LENGTH\([^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "INTEGER", true},
		{regexp.MustCompile(fmt.Sprintf(`(?i)EXTRACT\([^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "NUMERIC", true},
		// STRING_AGG and ARRAY_AGG may contain ORDER BY inside — match up to the AS alias
		{regexp.MustCompile(fmt.Sprintf(`(?i)STRING_AGG\b.+?\)\s+(?:AS\s+)?%s\b`, cteColumn)), "TEXT", true},
		{regexp.MustCompile(fmt.Sprintf(`(?i)ARRAY_AGG\b.+?\)\s+(?:AS\s+)?%s\b`, cteColumn)), "TEXT[]", true},
		// COALESCE-wrapped aggregates: COALESCE(SUM(...), 0) AS col
		{regexp.MustCompile(fmt.Sprintf(`(?i)COALESCE\s*\(\s*SUM\([^)]*\)[^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "BIGINT", false},
		{regexp.MustCompile(fmt.Sprintf(`(?i)COALESCE\s*\(\s*AVG\([^)]*\)[^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "NUMERIC", false},
		{regexp.MustCompile(fmt.Sprintf(`(?i)COALESCE\s*\(\s*COUNT\([^)]*\)[^)]*\)\s+(?:AS\s+)?%s\b`, cteColumn)), "BIGINT", false},
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
			return "BIGINT", false, true
		case "SUM":
			return "BIGINT", true, true
		case "AVG":
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
				return "BIGINT", false, true
			case "SUM":
				return "BIGINT", true, true
			case "AVG":
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

	// Bare column without table prefix in CTE: SELECT title, created_at FROM posts
	// The column name matches the cteColumn and we resolve type from the FROM table
	fromRe := regexp.MustCompile(`(?i)FROM\s+(\w+)`)
	if fromMatch := fromRe.FindStringSubmatch(cteQuery); len(fromMatch) > 1 {
		fromTable := fromMatch[1]
		for _, table := range schema.Tables {
			if strings.EqualFold(table.Name, fromTable) {
				for _, col := range table.Columns {
					if strings.EqualFold(col.Name, cteColumn) {
						return col.Type, col.Nullable, true
					}
				}
			}
		}
	}

	return "", false, false
}

// resolveCTEColumn resolves the type of a CTE column (e.g. "ct" → "depth")
// by scanning the SQL for "ct AS (" and finding "0 as depth" or "ct.depth + 1" inside.
// NOTE: This is a method on *TypeInferrer, not *QueryParser.
func (ti *TypeInferrer) resolveCTEColumn(sql string, cteAlias string, colName string) (string, bool, bool) {
	if ti.schema == nil {
		return "", false, false
	}

	// Find CTE definition: "cteAlias AS ("
	searchRe := regexp.MustCompile(fmt.Sprintf(`(?i)%s\s+AS\s*\(`, regexp.QuoteMeta(cteAlias)))
	loc := searchRe.FindStringIndex(sql)
	if loc == nil {
		// Try resolving via FROM alias: "FROM cteName cteAlias"
		aliasRe := regexp.MustCompile(fmt.Sprintf(`(?i)(?:FROM|JOIN)\s+(\w+)\s+%s\b`, regexp.QuoteMeta(cteAlias)))
		if am := aliasRe.FindStringSubmatch(sql); len(am) > 1 {
			return ti.resolveCTEColumn(sql, am[1], colName)
		}
		return "", false, false
	}

	// Extract CTE body (balanced parens)
	openPos := strings.Index(sql[loc[1]-1:], "(") + loc[1] - 1
	cteBody := extractBalancedParens(sql, openPos)
	if cteBody == "" {
		return "", false, false
	}

	// Detect arithmetic: ct.depth + 1 AS depth
	arithRe := regexp.MustCompile(fmt.Sprintf(`(?i)\+.*?\s+(?:AS\s+)?%s\b`, regexp.QuoteMeta(colName)))
	if arithRe.MatchString(cteBody) {
		return "INTEGER", false, true
	}

	// Integer literal: 0 as depth
	intRe := regexp.MustCompile(fmt.Sprintf(`(?i)\b\d+\s+(?:AS\s+)?%s\b`, regexp.QuoteMeta(colName)))
	if intRe.MatchString(cteBody) {
		return "INTEGER", false, true
	}

	// Simple column alias: score AS base_score
	colRefRe := regexp.MustCompile(fmt.Sprintf(`(?i)(?:(\w+)\.)?(\w+)\s+[Aa][Ss]\s+%s\b`, regexp.QuoteMeta(colName)))
	if m := colRefRe.FindStringSubmatch(cteBody); len(m) >= 3 {
		srcCol := m[2]
		for _, t := range ti.schema.Tables {
			for _, c := range t.Columns {
				if strings.EqualFold(c.Name, srcCol) {
					return c.Type, c.Nullable, true
				}
			}
		}
	}

	return "", false, false
}
