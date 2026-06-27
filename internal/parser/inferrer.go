package parser

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

type TypeInferrer struct {
	mu     sync.RWMutex
	cache  map[string]string
	schema *Schema
}

func NewTypeInferrer() *TypeInferrer {
	return &TypeInferrer{cache: make(map[string]string, 64)}
}

func NewTypeInferrerWithSchema(schema *Schema) *TypeInferrer {
	return &TypeInferrer{cache: make(map[string]string, 64), schema: schema}
}

// InferParamTypeByName infers a param type from its name alone, without a table schema.
// Used for CTE queries where the table can't be resolved.
func (ti *TypeInferrer) InferParamTypeByName(paramName string) string {
	n := strings.ToLower(paramName)
	switch n {
	case "limit", "offset", "count", "min_count", "count_threshold", "id", "age":
		return "INTEGER"
	}
	if strings.HasSuffix(n, "_count") || strings.HasSuffix(n, "_sum") ||
		strings.HasSuffix(n, "_total") || strings.HasSuffix(n, "_num") ||
		strings.HasSuffix(n, "_id") || strings.HasSuffix(n, "_age") {
		return "INTEGER"
	}
	if strings.Contains(n, "score") || strings.Contains(n, "rating") || strings.Contains(n, "avg") {
		return "DOUBLE PRECISION"
	}
	if strings.Contains(n, "is_") || strings.HasPrefix(n, "is_") || n == "active" || n == "featured" || n == "pinned" {
		return "BOOLEAN"
	}
	return "TEXT"
}

func (ti *TypeInferrer) InferParamType(sql string, paramIndex int, table *Table, paramName string) string {
	cacheKey := fmt.Sprintf("%s:%d:%s", table.Name, paramIndex, paramName)

	ti.mu.RLock()
	cached, ok := ti.cache[cacheKey]
	ti.mu.RUnlock()
	if ok {
		return cached
	}

	result := ti.inferParamTypeInternal(sql, paramIndex, table, paramName)

	if result != "" {
		ti.mu.Lock()
		ti.cache[cacheKey] = result
		ti.mu.Unlock()
	}

	return result
}

func (ti *TypeInferrer) inferParamTypeInternal(sql string, paramIndex int, table *Table, paramName string) string {
	// Well-known param names that always have fixed types
	nameLower := strings.ToLower(paramName)
	switch nameLower {
	case "limit", "offset", "count", "min_count", "count_threshold":
		return "INTEGER"
	}
	// Any param named *_count, *_sum, *_total is numeric
	if strings.HasSuffix(nameLower, "_count") || strings.HasSuffix(nameLower, "_sum") ||
		strings.HasSuffix(nameLower, "_total") || strings.HasSuffix(nameLower, "_num") {
		return "INTEGER"
	}

	// col = ANY($N) must be checked before name-based lookup, as the param name
	// is the column name but the type is col.Type[] (an array), not col.Type.
	anyArrayRe := regexp.MustCompile(fmt.Sprintf(`(?i)(?:\w+\.)?(\w+)\s*=\s*ANY\s*\(\s*\$%d`, paramIndex))
	if match := anyArrayRe.FindStringSubmatch(sql); len(match) > 1 {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, match[1]) {
				return col.Type + "[]"
			}
		}
	}

	if paramName != "" && paramName != fmt.Sprintf("param%d", paramIndex) {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, paramName) ||
				strings.EqualFold(col.Name, strings.TrimSuffix(strings.TrimSuffix(paramName, "_start"), "_end")) {
				return col.Type
			}
		}
		// Cross-table lookup: param name may refer to a column in another table (subquery joins)
		if ti.schema != nil {
			baseName := strings.TrimSuffix(strings.TrimSuffix(paramName, "_start"), "_end")
			for _, t := range ti.schema.Tables {
				for _, col := range t.Columns {
					if strings.EqualFold(col.Name, baseName) {
						return col.Type
					}
				}
			}
		}
	}

	aggregatePattern := fmt.Sprintf(`(?i)\b(count|sum|avg|max|min|total)_?\w*\s*[<>=!]+\s*\$%d|\$%d\s*[<>=!]+\s*\b(count|sum|avg|max|min|total)_?\w*|\b\w+_count\b\s*[<>=!]+\s*\$%d|\b\w+_sum\b\s*[<>=!]+\s*\$%d`, paramIndex, paramIndex, paramIndex, paramIndex)
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
		insertColRegex := regexp.MustCompile(`(?i)INSERT\s+INTO\s+\S+\s*\(([\s\S]*?)\)\s*VALUES`)
		allInsertCols := []string{}
		for _, match := range insertColRegex.FindAllStringSubmatch(sql, -1) {
			for _, c := range strings.Split(match[1], ",") {
				allInsertCols = append(allInsertCols, strings.TrimSpace(c))
			}
		}
		if paramIndex <= len(allInsertCols) {
			colName := allInsertCols[paramIndex-1]
			for _, col := range table.Columns {
				if strings.EqualFold(col.Name, colName) {
					return col.Type
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
	// SET col = COALESCE($N, col)
	setCoalesceRe := regexp.MustCompile(fmt.Sprintf(`(?i)(\w+)\s*=\s*COALESCE\s*\(\s*\$%d\b`, paramIndex))
	if match := setCoalesceRe.FindStringSubmatch(sql); len(match) > 1 {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, match[1]) {
				return col.Type
			}
		}
	}
	// SET with ? params — extract by using same logic as InferParamName
	if strings.Contains(sql, "?") {
		setColPattern := regexp.MustCompile(`(?i)SET\s+([\s\S]*?)(?:WHERE|$)`)
		if setMatch := setColPattern.FindStringSubmatch(sql); len(setMatch) > 1 {
			colPattern := regexp.MustCompile(`(\w+)\s*=\s*\?`)
			matches := colPattern.FindAllStringSubmatch(setMatch[1], -1)
			if paramIndex <= len(matches) {
				colName := matches[paramIndex-1][1]
				for _, col := range table.Columns {
					if strings.EqualFold(col.Name, colName) {
						return col.Type
					}
				}
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
	// Check for INSERT statement first — collect ALL column names from every INSERT in multi-statement SQL
	insertColRegex := regexp.MustCompile(`(?i)INSERT\s+INTO\s+\S+\s*\(([\s\S]*?)\)\s*VALUES`)
	allInsertCols := []string{}
	for _, match := range insertColRegex.FindAllStringSubmatch(sql, -1) {
		for _, c := range strings.Split(match[1], ",") {
			allInsertCols = append(allInsertCols, strings.TrimSpace(c))
		}
	}
	if paramIndex <= len(allInsertCols) {
		return allInsertCols[paramIndex-1]
	}

	// col = ANY($N) — param name is the column name (it's an array param)
	anyNameRe := regexp.MustCompile(fmt.Sprintf(`(?i)(?:\w+\.)?(\w+)\s*=\s*ANY\s*\(\s*\$%d`, paramIndex))
	if match := anyNameRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	if strings.Contains(sql, "?") {
		// SET clause with ? params: SET col = ?, col2 = ?, col = col + ?
		setColPattern := regexp.MustCompile(`(?i)SET\s+([\s\S]*?)(?:WHERE|$)`)
		if setMatch := setColPattern.FindStringSubmatch(sql); len(setMatch) > 1 {
			setClause := setMatch[1]
			// Match both: direct (col = ?) and counter (col = col + ? or col = col - ?)
			colPattern := regexp.MustCompile(`(?i)(\w+)\s*=\s*(?:\w+\s*[+\-]\s*)?\?`)
			allSetMatches := colPattern.FindAllStringSubmatch(setClause, -1)
			setCols := []string{}
			for _, m := range allSetMatches {
				setCols = append(setCols, m[1])
			}
			if paramIndex <= len(setCols) {
				name := setCols[paramIndex-1]
				// For counter pattern (col = col +/- ?) append _delta
				counterCheck := regexp.MustCompile(fmt.Sprintf(`(?i)%s\s*=\s*\w+\s*[+\-]\s*\?`, regexp.QuoteMeta(name)))
				if counterCheck.MatchString(setClause) {
					return name + "_delta"
				}
				return name
			}
			// Offset index past SET params for WHERE matching
			paramIndex = paramIndex - len(setCols)
		}

		// WHERE clause with ? params
		whereRegex := regexp.MustCompile(`(?is)WHERE\s+([\s\S]+?)(?:LIMIT|ORDER|GROUP|HAVING|ALLOW FILTERING|$)`)
		if whereMatch := whereRegex.FindStringSubmatch(sql); len(whereMatch) > 1 {
			whereClause := whereMatch[1]
			colPattern := regexp.MustCompile(`(?i)(\w+)\s*=\s*\?`)
			matches := colPattern.FindAllStringSubmatch(whereClause, -1)
			if paramIndex <= len(matches) && len(matches[paramIndex-1]) > 1 {
				return matches[paramIndex-1][1]
			}

			// CONTAINS ? pattern
			containsPattern := regexp.MustCompile(`(?i)(\w+)\s+CONTAINS\s+\?`)
			allContains := containsPattern.FindAllStringSubmatch(whereClause, -1)
			// Only match if this paramIndex maps to a CONTAINS position
			if paramIndex <= len(allContains) {
				return allContains[paramIndex-1][1]
			}
			// Also match >= AND <= BETWEEN-style
			whereParamIndex := paramIndex - len(matches)
			rangePattern := regexp.MustCompile(`(?i)(\w+)\s*(>=|<=|>|<)\s*\?`)
			rangeMatches := rangePattern.FindAllStringSubmatch(whereClause, -1)
			if whereParamIndex > 0 && whereParamIndex <= len(rangeMatches) {
				return rangeMatches[whereParamIndex-1][1]
			}
		}

		// LIMIT ? — count total params before LIMIT to find if this ? is LIMIT
		if regexp.MustCompile(`(?i)LIMIT\s+\?`).MatchString(sql) {
			beforeLimit := regexp.MustCompile(`(?i)LIMIT\s+\?`).Split(sql, 2)[0]
			totalBefore := strings.Count(beforeLimit, "?")
			if paramIndex == totalBefore+1 {
				return "limit"
			}
		}

		// SET col = col + ? (counter increment) → use the column name
		counterRe := regexp.MustCompile(`(?i)(\w+)\s*=\s*(\w+)\s*\+\s*\?`)
		for _, m := range counterRe.FindAllStringSubmatch(sql, -1) {
			if strings.EqualFold(m[1], m[2]) {
				// Find position of this ? in the SQL
				idx := strings.Index(sql, m[0])
				if idx >= 0 {
					pos := strings.Count(sql[:idx+len(m[0])], "?")
					if paramIndex == pos {
						return m[1]
					}
				}
			}
		}
	}

	wherePattern := fmt.Sprintf(`(?i)(?:WHERE|AND|OR)\s*\(?\s*(?:\w+\.)?(\w+)\s*=\s*\$%d\b`, paramIndex)
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

	// SET col = COALESCE($N, col) — "update if not null" pattern
	setCoalesceRe := regexp.MustCompile(fmt.Sprintf(`(?i)(\w+)\s*=\s*COALESCE\s*\(\s*\$%d\b`, paramIndex))
	if match := setCoalesceRe.FindStringSubmatch(sql); len(match) > 1 {
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

	compPattern := fmt.Sprintf(`(?i)(?:WHERE|AND|OR)\s+(?:\w+\.)?(\w+)\s*([<>=!]+)\s*\$%d`, paramIndex)
	compRe := regexp.MustCompile(compPattern)
	if match := compRe.FindStringSubmatch(sql); len(match) > 2 {
		col := match[1]
		op := match[2]
		// Detect range pattern: same col with opposite operator on another param
		otherRangeRe := regexp.MustCompile(fmt.Sprintf(`(?i)%s\s*[<>=!]+\s*\$\d+`, regexp.QuoteMeta(col)))
		if len(otherRangeRe.FindAllString(sql, -1)) > 1 {
			if op == ">=" || op == ">" {
				return col + "_start"
			}
			if op == "<=" || op == "<" {
				return col + "_end"
			}
		}
		return col
	}

	// COALESCE(col, ...) op $N
	coalesceRe := regexp.MustCompile(fmt.Sprintf(`(?i)COALESCE\s*\(\s*(\w+)[^)]*\)\s*[><=!]+\s*\$%d`, paramIndex))
	if match := coalesceRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// col @> $N, col && $N, col || $N (jsonb/array operators)
	jsonbOpRe := regexp.MustCompile(fmt.Sprintf(`(?i)(\w+)\s*(?:@>|&&|\|\|)\s*\$%d`, paramIndex))
	if match := jsonbOpRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// col ->> $N IS NOT NULL — jsonb key existence check, name as "key"
	jsonbKeyRe := regexp.MustCompile(fmt.Sprintf(`(?i)(\w+)\s*->>\s*\$%d`, paramIndex))
	if match := jsonbKeyRe.FindStringSubmatch(sql); len(match) > 1 {
		return "key"
	}

	// $N = ANY(col)
	anyRe := regexp.MustCompile(fmt.Sprintf(`(?i)\$%d\s*=\s*ANY\s*\(\s*(\w+)\s*\)`, paramIndex))
	if match := anyRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// array_append(col, $N) / array_remove(col, $N)
	arrFnRe := regexp.MustCompile(fmt.Sprintf(`(?i)array_(?:append|remove)\s*\(\s*(\w+)\s*,\s*\$%d\b`, paramIndex))
	if match := arrFnRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// col->>'key' = $N or col->'key' = $N (jsonb field access)
	arrowRe := regexp.MustCompile(fmt.Sprintf(`(?i)(\w+)->>?\S+\s*=\s*\$%d`, paramIndex))
	if match := arrowRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// id = ANY($N::type) — cast variant
	anyWithCastRe := regexp.MustCompile(fmt.Sprintf(`(?i)(?:\w+\.)?(\w+)\s*=\s*ANY\s*\(\s*\$%d`, paramIndex))
	if match := anyWithCastRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// HAVING COUNT(*) > $N or HAVING col op $N
	havingRe := regexp.MustCompile(fmt.Sprintf(`(?i)HAVING\s+(?:\w+\s*\(.*?\)\s*)?(?:(\w+)\s*)?[><=!]+\s*\$%d`, paramIndex))
	if match := havingRe.FindStringSubmatch(sql); len(match) > 1 && match[1] != "" {
		return "count_threshold"
	}
	if regexp.MustCompile(fmt.Sprintf(`(?i)HAVING\s+.*\$%d`, paramIndex)).MatchString(sql) {
		return "count_threshold"
	}

	// WHERE (subquery) > $N — correlated subquery threshold
	subqueryThresholdRe := regexp.MustCompile(fmt.Sprintf(`(?i)(?:WHERE|AND|OR)\s+\(SELECT[\s\S]*?\)\s*[><=!]+\s*\$%d`, paramIndex))
	if subqueryThresholdRe.MatchString(sql) {
		return "min_count"
	}

	// plainto_tsquery(..., $N) or to_tsquery(..., $N) — full-text search
	tsqueryRe := regexp.MustCompile(fmt.Sprintf(`(?i)(?:plainto_tsquery|to_tsquery|phraseto_tsquery)\s*\([^,)]+,\s*\$%d\b`, paramIndex))
	if tsqueryRe.MatchString(sql) {
		return "search_query"
	}

	// WHERE col IN ($1, $2, ...) — name each as col_1, col_2
	inListRe := regexp.MustCompile(`(?i)(?:WHERE|AND|OR)\s+(?:\w+\.)?(\w+)\s+IN\s*\(([^)]+)\)`)
	if inMatch := inListRe.FindStringSubmatch(sql); len(inMatch) > 2 {
		colName := inMatch[1]
		for pos, part := range strings.Split(inMatch[2], ",") {
			if regexp.MustCompile(fmt.Sprintf(`\$%d\b`, paramIndex)).MatchString(strings.TrimSpace(part)) {
				return fmt.Sprintf("%s%d", colName, pos+1)
			}
		}
	}

	// CTE: WHERE alias.col > $N — strip table alias and try col name
	ctePropRe := regexp.MustCompile(fmt.Sprintf(`(?i)(?:WHERE|AND|OR)\s+\w+\.(\w+)\s*[><=!]+\s*\$%d`, paramIndex))
	if match := ctePropRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// SET col = ROW($1, $2, ...) — name params as col_field1, col_field2
	rowSetRe := regexp.MustCompile(`(?i)SET\s+(\w+)\s*=\s*ROW\s*\(([^)]+)\)`)
	if rowSetMatch := rowSetRe.FindStringSubmatch(sql); len(rowSetMatch) > 2 {
		colName := rowSetMatch[1]
		for pos, part := range strings.Split(rowSetMatch[2], ",") {
			if regexp.MustCompile(fmt.Sprintf(`\$%d\b`, paramIndex)).MatchString(strings.TrimSpace(part)) {
				return fmt.Sprintf("%s_field%d", colName, pos+1)
			}
		}
	}

	// Generic: col = $N anywhere in the SQL (catches multi-column SET clauses etc.)
	genericColRe := regexp.MustCompile(fmt.Sprintf(`(?i)(\w+)\s*=\s*\$%d\b`, paramIndex))
	if match := genericColRe.FindStringSubmatch(sql); len(match) > 1 {
		name := strings.ToLower(match[1])
		// Avoid returning SQL keywords as param names
		if name != "true" && name != "false" && name != "null" {
			return match[1]
		}
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
