package parser

import (
	"regexp"
	"strings"
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

// extractAggInnerColumn extracts the column reference from an aggregate like MIN(age) or MAX(t.created_at)
func extractAggInnerColumn(expr string, aggName string) string {
	re := regexp.MustCompile(`(?i)` + aggName + `\s*\(\s*([^)]+)\s*\)`)
	if m := re.FindStringSubmatch(expr); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// resolveColumnType looks up a column reference (e.g. "age" or "t.age") in the schema
func resolveColumnType(colRef string, schema *Schema) string {
	if schema == nil {
		return ""
	}
	// Handle qualified ref: table.column or alias.column
	parts := strings.Split(colRef, ".")
	var colName string
	if len(parts) == 2 {
		colName = strings.TrimSpace(parts[1])
	} else {
		colName = strings.TrimSpace(parts[0])
	}
	// Search all tables for the column
	for _, table := range schema.Tables {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, colName) {
				return col.Type
			}
		}
	}
	return ""
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
	// Check if there's an explicit PostgreSQL cast (e.g. ::INTEGER, ::TEXT)
	// If so, the cast determines the final type (nullable stays from the inner expression)
	castRe := regexp.MustCompile(`(?i)::([a-zA-Z][a-zA-Z0-9_]*)(\([^)]*\))?$`)
	if castMatch := castRe.FindStringSubmatch(strings.TrimSpace(originalExpr)); len(castMatch) > 1 {
		castType := strings.ToUpper(castMatch[1])
		// Determine nullability from the inner expression
		inner := stripPGCast(originalExpr)
		innerUpper := strings.ToUpper(inner)
		nullable := strings.Contains(innerUpper, "AVG(") || strings.Contains(innerUpper, "SUM(") ||
			strings.Contains(innerUpper, "MAX(") || strings.Contains(innerUpper, "MIN(") ||
			strings.Contains(innerUpper, "NULLIF(")
		return castType, nullable, true
	}

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
		// Fallback: resolve column name across all tables (handles LATERAL/subquery aliases)
		// Mark as nullable since unresolved alias likely comes from a LEFT JOIN
		for _, table := range schema.Tables {
			for _, col := range table.Columns {
				if strings.EqualFold(col.Name, columnName) {
					return col.Type, true, true
				}
			}
		}
	}

	// Window functions → BIGINT (ROW_NUMBER, RANK, DENSE_RANK return bigint in PostgreSQL)
	if windowFuncRe.MatchString(exprUpper) {
		return "BIGINT", false, true
	}

	// EXISTS(...) always returns BOOLEAN
	if strings.Contains(exprUpper, "EXISTS(") {
		return "BOOLEAN", false, true
	}

	if strings.Contains(exprUpper, "COUNT(") {
		return "BIGINT", false, true
	}
	if strings.Contains(exprUpper, "SUM(") {
		// SUM of integer types returns BIGINT in PostgreSQL, SUM of numeric returns NUMERIC
		innerCol := extractAggInnerColumn(expr, "SUM")
		if innerCol != "" {
			if colType := resolveColumnType(innerCol, schema); colType != "" {
				colUpper := strings.ToUpper(colType)
				if colUpper == "INTEGER" || colUpper == "INT" || colUpper == "SMALLINT" || colUpper == "BIGINT" || colUpper == "SERIAL" || colUpper == "BIGSERIAL" {
					return "BIGINT", true, true
				}
			}
		}
		// Default: SUM returns NUMERIC (for decimal/float types)
		return "NUMERIC", true, true
	}
	if strings.Contains(exprUpper, "AVG(") {
		return "NUMERIC", true, true
	}
	if strings.Contains(exprUpper, "MAX(") || strings.Contains(exprUpper, "MIN(") {
		if strings.Contains(exprUpper, "_AT") || strings.Contains(exprUpper, "_DATE") {
			return "TIMESTAMP WITH TIME ZONE", true, true
		}
		// MIN/MAX preserve the type of the inner column
		aggName := "MIN"
		if strings.Contains(exprUpper, "MAX(") {
			aggName = "MAX"
		}
		innerCol := extractAggInnerColumn(expr, aggName)
		if innerCol != "" {
			if colType := resolveColumnType(innerCol, schema); colType != "" {
				return colType, true, true
			}
		}
		return "NUMERIC", true, true
	}

	// JSONB aggregate functions
	if strings.Contains(exprUpper, "JSONB_AGG(") || strings.Contains(exprUpper, "JSON_AGG(") ||
		strings.Contains(exprUpper, "JSONB_BUILD_OBJECT(") || strings.Contains(exprUpper, "JSONB_BUILD_ARRAY(") ||
		strings.Contains(exprUpper, "TO_JSONB(") {
		return "JSONB", true, true
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
		if strings.Contains(exprUpper, "COALESCE(SUM(") {
			// COALESCE(SUM(int_col), 0) → BIGINT in PostgreSQL
			return "BIGINT", false, true
		}
		if strings.Contains(exprUpper, "COALESCE(AVG(") {
			return "NUMERIC", false, true
		}
		if strings.Contains(exprUpper, "COALESCE(COUNT(") {
			return "BIGINT", false, true
		}
		if strings.Contains(exprUpper, "COALESCE(MAX(") || strings.Contains(exprUpper, "COALESCE(MIN(") {
			if strings.Contains(exprUpper, "_AT") || strings.Contains(exprUpper, "_DATE") {
				return "TIMESTAMP WITH TIME ZONE", false, true
			}
			return "NUMERIC", false, true
		}
	}
	// Subquery expression: (SELECT agg(...) FROM ...) or (SELECT col FROM table ...)
	if strings.HasPrefix(exprTrimmed, "(") && strings.Contains(exprUpper, "SELECT") {
		if strings.Contains(exprUpper, "COUNT(") {
			return "BIGINT", true, true
		}
		if strings.Contains(exprUpper, "SUM(") {
			return "BIGINT", true, true
		}
		if strings.Contains(exprUpper, "AVG(") {
			return "NUMERIC", true, true
		}
		if strings.Contains(exprUpper, "EXISTS(") {
			return "BOOLEAN", true, true
		}
		// Try to resolve the selected column type from schema
		// Pattern: (SELECT col_name FROM table_name WHERE ...)
		subColRe := regexp.MustCompile(`(?i)\(\s*SELECT\s+(?:\w+\.)?(\w+)\s+FROM\s+(\w+)`)
		if m := subColRe.FindStringSubmatch(exprTrimmed); len(m) > 2 {
			colName := m[1]
			tableName := m[2]
			if schema != nil {
				for _, table := range schema.Tables {
					if strings.EqualFold(table.Name, tableName) {
						for _, col := range table.Columns {
							if strings.EqualFold(col.Name, colName) {
								return col.Type, true, true // nullable because subquery may return no rows
							}
						}
					}
				}
			}
		}
		return "TEXT", true, true
	}

	if strings.Contains(exprUpper, "COALESCE(") {
		coalesceRe := regexp.MustCompile(`(?i)COALESCE\s*\(\s*([^,)]+)`)
		if matches := coalesceRe.FindStringSubmatch(expr); len(matches) > 1 {
			firstArg := strings.TrimSpace(matches[1])
			firstArgUpper := strings.ToUpper(firstArg)

			// Numeric CTE alias columns (these are typically COUNT/SUM results → BIGINT)
			if numericCTEColRe.MatchString(firstArgUpper) {
				return "BIGINT", false, true
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
				// Try resolving via table alias in schema (covers LATERAL join aliases
				// where the alias refers to a subquery column with known aggregate type)
				colName := strings.TrimSpace(cteParts[1])
				for _, table := range schema.Tables {
					for _, col := range table.Columns {
						if strings.EqualFold(col.Name, colName) {
							return col.Type, true, true
						}
					}
				}
			}
		}

		// Check for explicit ::type cast on any COALESCE argument as type hint.
		// e.g., COALESCE(a.attachments, '[]'::jsonb) → JSONB
		coalesceCastRe := regexp.MustCompile(`(?i)COALESCE\s*\([^)]*::\s*([a-zA-Z][a-zA-Z0-9_]*)`)
		if castMatch := coalesceCastRe.FindStringSubmatch(expr); len(castMatch) > 1 {
			return strings.ToUpper(castMatch[1]), true, true
		}

		// Check if COALESCE wraps a jsonb_agg or json_agg call
		if strings.Contains(exprUpper, "JSONB_AGG(") || strings.Contains(exprUpper, "JSON_AGG(") {
			return "JSONB", true, true
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
