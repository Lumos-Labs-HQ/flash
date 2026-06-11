package parser

import (
	"fmt"
	"regexp"
	"strings"
)

type TypeInferrer struct {
	cache  map[string]string
	schema *Schema
}

func NewTypeInferrer() *TypeInferrer {
	return &TypeInferrer{cache: make(map[string]string, 64)}
}

func NewTypeInferrerWithSchema(schema *Schema) *TypeInferrer {
	return &TypeInferrer{cache: make(map[string]string, 64), schema: schema}
}

func (ti *TypeInferrer) InferParamType(sql string, paramIndex int, table *Table, paramName string) string {
	cacheKey := fmt.Sprintf("%s:%d:%s", table.Name, paramIndex, paramName)
	if cached, ok := ti.cache[cacheKey]; ok {
		return cached
	}

	result := ti.inferParamTypeInternal(sql, paramIndex, table, paramName)

	if result != "" {
		ti.cache[cacheKey] = result
	}

	return result
}

func (ti *TypeInferrer) inferParamTypeInternal(sql string, paramIndex int, table *Table, paramName string) string {
	if paramName != "" && paramName != fmt.Sprintf("param%d", paramIndex) {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, paramName) ||
				strings.EqualFold(col.Name, strings.TrimSuffix(strings.TrimSuffix(paramName, "_start"), "_end")) {
				return col.Type
			}
		}
	}

	aggregatePattern := fmt.Sprintf(`(?i)\b(count|sum|avg|max|min|total)_?\w*\s*[<>=!]+\s*\$%d|\$%d\s*[<>=!]+\s*\b(count|sum|avg|max|min|total)_?\w*`, paramIndex, paramIndex)
	if matched, _ := regexp.MatchString(aggregatePattern, sql); matched {
		return "INTEGER"
	}

	// CTE alias numeric column comparisons: ups.total_posts > $1
	numericAliasPattern := fmt.Sprintf(`(?i)\w+\.(total_posts|published_posts|draft_posts|total_comments|posts_commented_on|categories_used|engagement_score|count|sum|avg|total|min|max|num|qty|quantity|amount|cnt|total_cnt|post_cnt|comment_cnt|pub_cnt|draft_cnt|posts_cnt|cat_cnt)\s*[<>=!]+\s*\$%d|\$%d\s*[<>=!]+\s*\w+\.(total_posts|published_posts|draft_posts|total_comments|posts_commented_on|categories_used|engagement_score|count|sum|avg|total|min|max|num|qty|quantity|amount|cnt)`, paramIndex, paramIndex)
	if matched, _ := regexp.MatchString(numericAliasPattern, sql); matched {
		return "INTEGER"
	}

	coalescePattern := fmt.Sprintf(`(?i)COALESCE\([^)]*\.(cnt|count|sum|avg|total|total_\w+|post\w*|comment\w*|pub\w*|draft\w*|posts\w*|cat\w*|unique\w*|engagement\w*|categories\w*)[^)]*\)\s*[<>=!]+\s*\$%d|\$%d\s*[<>=!]+\s*COALESCE\([^)]*\.(cnt|count|sum|avg|total)`, paramIndex, paramIndex)
	if matched, _ := regexp.MatchString(coalescePattern, sql); matched {
		return "INTEGER"
	}

	wherePattern := fmt.Sprintf(`(?i)WHERE\s+(?:\w+\.)?(\w+)\s*=\s*\$%d`, paramIndex)
	whereRe := regexp.MustCompile(wherePattern)
	if match := whereRe.FindStringSubmatch(sql); len(match) > 1 {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, match[1]) {
				return col.Type
			}
		}
	}

	// ILIKE / SIMILAR TO / LIKE patterns: WHERE col ILIKE $N
	likePattern := fmt.Sprintf(`(?i)(?:WHERE|AND|OR)\s*\(?\s*(?:\w+\.)?(\w+)\s+(?:I?LIKE|SIMILAR\s+TO|NOT\s+I?LIKE)\s+\$%d`, paramIndex)
	likeRe := regexp.MustCompile(likePattern)
	if match := likeRe.FindStringSubmatch(sql); len(match) > 1 {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, match[1]) {
				return col.Type
			}
		}
		return "TEXT" // default for LIKE params
	}

	if strings.Contains(strings.ToUpper(sql), "INSERT") {
		insertColRegex := regexp.MustCompile(`(?i)INSERT\s+INTO\s+\w+\s*\(([\s\S]*?)\)\s*VALUES`)
		if match := insertColRegex.FindStringSubmatch(sql); len(match) > 1 {
			colNames := strings.Split(match[1], ",")
			if paramIndex <= len(colNames) {
				colName := strings.TrimSpace(colNames[paramIndex-1])
				for _, col := range table.Columns {
					if strings.EqualFold(col.Name, colName) {
						return col.Type
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
				return col.Type
			}
		}
	}

	limitPattern := fmt.Sprintf(`(?i)LIMIT\s+\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(limitPattern, sql); matched {
		return "INTEGER"
	}

	offsetPattern := fmt.Sprintf(`(?i)OFFSET\s+\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(offsetPattern, sql); matched {
		return "INTEGER"
	}

	betweenPattern := fmt.Sprintf(`(?i)(\w+)\s+BETWEEN\s+\$%d`, paramIndex)
	betweenRe := regexp.MustCompile(betweenPattern)
	if match := betweenRe.FindStringSubmatch(sql); len(match) > 1 {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, match[1]) {
				return col.Type
			}
		}
	}

	betweenEndPattern := fmt.Sprintf(`(?i)BETWEEN\s+\$\d+\s+AND\s+\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(betweenEndPattern, sql); matched {
		betweenStartRe := regexp.MustCompile(`(?i)(\w+)\s+BETWEEN`)
		if match := betweenStartRe.FindStringSubmatch(sql); len(match) > 1 {
			for _, col := range table.Columns {
				if strings.EqualFold(col.Name, match[1]) {
					return col.Type
				}
			}
		}
	}

	datePattern := fmt.Sprintf(`(?i)(created_at|updated_at|deleted_at|published_at|date|time)\s*[<>=]+\s*\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(datePattern, sql); matched {
		return "TIMESTAMP"
	}

	// WHERE alias.col > $N — unqualified comparison fallback in primary table
	compQualPattern := fmt.Sprintf(`(?i)(?:(\w+)\.)?(\w+)\s*[<>=!]+\s*\$%d`, paramIndex)
	compQualRe := regexp.MustCompile(compQualPattern)
	if match := compQualRe.FindStringSubmatch(sql); len(match) > 1 {
		tableQual := match[1]
		colName := match[2]
		// Search primary table first
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, colName) {
				return col.Type
			}
		}
		// Cross-table lookup
		if ti.schema != nil {
			for _, t := range ti.schema.Tables {
				for _, col := range t.Columns {
					if strings.EqualFold(col.Name, colName) {
						return col.Type
					}
				}
			}
			// CTE resolution: ct.depth — if there's a table qualifier, try resolving via CTE
			if tableQual != "" {
				if cteType, _, found := ti.resolveCTEColumn(sql, tableQual, colName); found {
					return cteType
				}
			}
		}
	}

	return "TEXT"
}

func (ti *TypeInferrer) InferParamName(sql string, paramIndex int) string {
	// Check for INSERT statement first (works for both ? and $n)
	if strings.Contains(strings.ToUpper(sql), "INSERT") {
		insertColRegex := regexp.MustCompile(`(?i)INSERT\s+INTO\s+\w+\s*\(([\s\S]*?)\)\s*VALUES`)
		if match := insertColRegex.FindStringSubmatch(sql); len(match) > 1 {
			colNames := strings.Split(match[1], ",")
			if paramIndex <= len(colNames) {
				return strings.TrimSpace(colNames[paramIndex-1])
			}
		}
	}

	if strings.Contains(sql, "?") {
		whereRegex := regexp.MustCompile(`(?i)WHERE\s+(.+?)(?:LIMIT|ORDER|GROUP|HAVING|$)`)
		if whereMatch := whereRegex.FindStringSubmatch(sql); len(whereMatch) > 1 {
			whereClause := whereMatch[1]
			colPattern := regexp.MustCompile(`(?i)(\w+)\s*=\s*\?`)
			matches := colPattern.FindAllStringSubmatch(whereClause, -1)
			if paramIndex <= len(matches) && len(matches[paramIndex-1]) > 1 {
				return matches[paramIndex-1][1]
			}
		}
	}

	wherePattern := fmt.Sprintf(`(?i)WHERE\s+(?:\w+\.)?(\w+)\s*=\s*\$%d`, paramIndex)
	whereRe := regexp.MustCompile(wherePattern)
	if match := whereRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// ILIKE / LIKE / SIMILAR TO
	likePattern := fmt.Sprintf(`(?i)(?:WHERE|AND|OR)\s*\(?\s*(?:\w+\.)?(\w+)\s+(?:I?LIKE|SIMILAR\s+TO|NOT\s+I?LIKE)\s+\$%d`, paramIndex)
	likeRe := regexp.MustCompile(likePattern)
	if match := likeRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	setPattern := fmt.Sprintf(`(?i)SET\s+(\w+)\s*=\s*\$%d`, paramIndex)
	setRe := regexp.MustCompile(setPattern)
	if match := setRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	limitPattern := fmt.Sprintf(`(?i)LIMIT\s+\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(limitPattern, sql); matched {
		return "limit"
	}

	offsetPattern := fmt.Sprintf(`(?i)OFFSET\s+\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(offsetPattern, sql); matched {
		return "offset"
	}

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

	compPattern := fmt.Sprintf(`(?i)(?:WHERE|AND|OR)\s+(?:\w+\.)?(\w+)\s*[<>=!]+\s*\$%d`, paramIndex)
	compRe := regexp.MustCompile(compPattern)
	if match := compRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	return fmt.Sprintf("param%d", paramIndex)
}

// resolveCTEColumn resolves the type of a CTE column (e.g. "ct" → "depth")
// by scanning the SQL for "ct AS (" and finding "0 as depth" or "ct.depth + 1" inside.
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
