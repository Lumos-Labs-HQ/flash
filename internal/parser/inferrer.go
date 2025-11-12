package parser

import (
	"fmt"
	"regexp"
	"strings"
)

type TypeInferrer struct {
	cache map[string]string
}

func NewTypeInferrer() *TypeInferrer {
	return &TypeInferrer{
		cache: make(map[string]string, 64),
	}
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
				// Return raw SQL type from schema
				return col.Type
			}
		}
	}

	aggregatePattern := fmt.Sprintf(`(?i)\b(count|sum|avg|max|min|total)_?\w*\s*[<>=!]+\s*\$%d|\$%d\s*[<>=!]+\s*\b(count|sum|avg|max|min|total)_?\w*`, paramIndex, paramIndex)
	if matched, _ := regexp.MatchString(aggregatePattern, sql); matched {
		return "INTEGER"
	}

	numericAliasPattern := fmt.Sprintf(`(?i)\w*\.(count|sum|avg|total|min|max|num|qty|quantity|amount)_?\w*\s*[<>=!]+\s*\$%d|\$%d\s*[<>=!]+\s*\w*\.(count|sum|avg|total|min|max|num|qty|quantity|amount)_?\w*`, paramIndex, paramIndex)
	if matched, _ := regexp.MatchString(numericAliasPattern, sql); matched {
		return "INTEGER"
	}

	wherePattern := fmt.Sprintf(`(?i)WHERE\s+(?:\w+\.)?(\w+)\s*=\s*\$%d`, paramIndex)
	whereRe := regexp.MustCompile(wherePattern)
	if match := whereRe.FindStringSubmatch(sql); len(match) > 1 {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, match[1]) {
				// Return raw SQL type from schema
				return col.Type
			}
		}
	}

	if strings.Contains(strings.ToUpper(sql), "INSERT") {
		insertColRegex := regexp.MustCompile(`(?i)INSERT\s+INTO\s+\w+\s*\(([\s\S]*?)\)\s*VALUES`)
		if match := insertColRegex.FindStringSubmatch(sql); len(match) > 1 {
			colNames := strings.Split(match[1], ",")
			if paramIndex <= len(colNames) {
				colName := strings.TrimSpace(colNames[paramIndex-1])
				for _, col := range table.Columns {
					if strings.EqualFold(col.Name, colName) {
						// Return raw SQL type from schema
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
				// Return raw SQL type from schema
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
				// Return raw SQL type from schema
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
					// Return raw SQL type from schema
					return col.Type
				}
			}
		}
	}

	datePattern := fmt.Sprintf(`(?i)(created_at|updated_at|date|time)\s*[<>=]+\s*\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(datePattern, sql); matched {
		return "TIMESTAMP"
	}

	return "TEXT" // Default to TEXT (generic string type)
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

	setPattern := fmt.Sprintf(`(?i)SET\s+(\w+)\s*=\s*\$%d`, paramIndex)
	setRe := regexp.MustCompile(setPattern)
	if match := setRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	limitPattern := fmt.Sprintf(`(?i)LIMIT\s+\$%d`, paramIndex)
	if matched, _ := regexp.MatchString(limitPattern, sql); matched {
		return "limit"
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

	compPattern := fmt.Sprintf(`(?i)(\w+)\s*[<>=]+\s*\$%d`, paramIndex)
	compRe := regexp.MustCompile(compPattern)
	if match := compRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	return fmt.Sprintf("param%d", paramIndex)
}
