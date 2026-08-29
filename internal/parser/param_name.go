package parser

import (
	"fmt"
	"regexp"
	"strings"
)

func (ti *TypeInferrer) InferParamName(sql string, paramIndex int) string {
	// INSERT statements: map the paramIndex-th ? to the VALUES slot (column)
	// it sits in. Handles literal slots ('VALUES (?, ''LIT'', ?...)') that the
	// old positional column-list lookup misnamed.
	if name := inferInsertName(sql, paramIndex); name != "" {
		return name
	}

	// col = ANY($N) — param name is the column name (it's an array param)
	anyNameRe := regexp.MustCompile(fmt.Sprintf(`(?i)(?:\w+\.)?(\w+)\s*=\s*ANY\s*\(\s*\$%d`, paramIndex))
	if match := anyNameRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// $N = ANY(col) or $N = ANY(alias.col) — reverse form, param on left side
	anyRevRe := regexp.MustCompile(fmt.Sprintf(`(?i)\$%d\s*=\s*ANY\s*\(\s*(?:\w+\.)?(\w+)\s*\)`, paramIndex))
	if match := anyRevRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	if strings.Contains(sql, "?") {
		// ? || col or col || ? in SELECT — concatenation prefix/suffix
		concatRe := regexp.MustCompile(`\?\s*\|\|\s*(\w+)`)
		if match := concatRe.FindStringSubmatch(sql); len(match) > 1 {
			beforeMatch := sql[:strings.Index(sql, match[0])]
			if strings.Count(beforeMatch, "?")+1 == paramIndex {
				return match[1] + "_prefix"
			}
		}

		// ? = ANY(col) or ? = ANY(alias.col) — reverse ANY pattern with ? params
		anyQRe := regexp.MustCompile(`\?\s*=\s*ANY\s*\(\s*(?:\w+\.)?(\w+)\s*\)`)
		if allAnyMatches := anyQRe.FindAllStringSubmatchIndex(sql, -1); len(allAnyMatches) > 0 {
			for _, loc := range allAnyMatches {
				beforeMatch := sql[:loc[0]]
				if strings.Count(beforeMatch, "?")+1 == paramIndex {
					return anyQRe.FindStringSubmatch(sql[loc[0]:loc[1]])[1]
				}
			}
		}

		// col = ANY(?) — forward ANY pattern with ? params
		anyFwdQRe := regexp.MustCompile(`(?i)(?:\w+\.)?(\w+)\s*=\s*ANY\s*\(\s*\?\s*\)`)
		if allFwdMatches := anyFwdQRe.FindAllStringSubmatchIndex(sql, -1); len(allFwdMatches) > 0 {
			for _, loc := range allFwdMatches {
				beforeMatch := sql[:loc[0]]
				// Count ? up to the ? inside ANY(...) — the ? is inside the parentheses
				matchStr := sql[loc[0]:loc[1]]
				qMarkPos := strings.Index(matchStr, "?")
				totalBefore := strings.Count(beforeMatch, "?") + strings.Count(matchStr[:qMarkPos], "?") + 1
				if paramIndex == totalBefore {
					return anyFwdQRe.FindStringSubmatch(matchStr)[1]
				}
			}
		}

		// Positional naming: map the paramIndex-th ? to the column it targets
		// by scanning the SET clause and the trailing WHERE clause in order.
		// The previous pattern-list lookups miscounted ?s wrapped in
		// expressions (COALESCE(?, col), CASE WHEN ?=?, func(?)), shifting
		// every later name — e.g. "SET a=?, b=COALESCE(?,b) WHERE id=?"
		// generated (a, id, param2) instead of (a, b, id).
		if name := inferPositionalName(sql, paramIndex); name != "" {
			return name
		}

		// LIMIT ? — count total params before LIMIT to find if this ? is LIMIT
		if regexp.MustCompile(`(?i)LIMIT\s+\?`).MatchString(sql) {
			beforeLimit := regexp.MustCompile(`(?i)LIMIT\s+\?`).Split(sql, 2)[0]
			totalBefore := strings.Count(beforeLimit, "?")
			if paramIndex == totalBefore+1 {
				return "limit"
			}
		}

		// OFFSET ? — count total params before OFFSET
		if regexp.MustCompile(`(?i)OFFSET\s+\?`).MatchString(sql) {
			beforeOffset := regexp.MustCompile(`(?i)OFFSET\s+\?`).Split(sql, 2)[0]
			totalBefore := strings.Count(beforeOffset, "?")
			if paramIndex == totalBefore+1 {
				return "offset"
			}
		}
	}

	// $N || col (concatenation prefix)
	concatDollarRe := regexp.MustCompile(fmt.Sprintf(`\$%d\s*\|\|\s*(\w+)`, paramIndex))
	if match := concatDollarRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1] + "_prefix"
	}

	// jsonb_set / json_set function params: jsonb_set(expr, $1, $2) → "path", "value"
	// The first arg to jsonb_set can be complex (e.g. COALESCE(col, '{}'))
	jsonbSetPathRe := regexp.MustCompile(fmt.Sprintf(`(?i)jsonb?_set\s*\(.*\$%d`, paramIndex))
	if jsonbSetPathRe.MatchString(sql) {
		// Find all $N params inside jsonb_set(...)
		jsonbSetFullRe := regexp.MustCompile(`(?i)jsonb?_set\s*\(`)
		if loc := jsonbSetFullRe.FindStringIndex(sql); loc != nil {
			// Extract params after the function name, counting from first $N
			afterFn := sql[loc[1]:]
			paramRe := regexp.MustCompile(`\$(\d+)`)
			params := paramRe.FindAllStringSubmatch(afterFn, -1)
			if len(params) >= 2 {
				if fmt.Sprintf("$%d", paramIndex) == params[0][0] {
					return "path"
				}
				if fmt.Sprintf("$%d", paramIndex) == params[1][0] {
					return "value"
				}
			}
		}
	}

	// jsonb_insert(expr, $1, $2) → "path", "value"
	jsonbInsertPathRe := regexp.MustCompile(fmt.Sprintf(`(?i)jsonb?_insert\s*\(.*\$%d`, paramIndex))
	if jsonbInsertPathRe.MatchString(sql) {
		jsonbInsertFullRe := regexp.MustCompile(`(?i)jsonb?_insert\s*\(`)
		if loc := jsonbInsertFullRe.FindStringIndex(sql); loc != nil {
			afterFn := sql[loc[1]:]
			paramRe := regexp.MustCompile(`\$(\d+)`)
			params := paramRe.FindAllStringSubmatch(afterFn, -1)
			if len(params) >= 2 {
				if fmt.Sprintf("$%d", paramIndex) == params[0][0] {
					return "path"
				}
				if fmt.Sprintf("$%d", paramIndex) == params[1][0] {
					return "value"
				}
			}
		}
	}

	// func(col) = $N or func(col)::type = $N (function-wrapped column comparison)
	funcColRe := regexp.MustCompile(fmt.Sprintf(`(?i)(?:WHERE|AND|OR)\s*\(?\s*\w+\s*\(\s*(?:\w+\.)?(\w+)\s*\)(?:::\w+)?\s*=\s*\$%d\b`, paramIndex))
	if match := funcColRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	wherePattern := fmt.Sprintf(`(?i)(?:WHERE|AND|OR)\s*\(?\s*(?:\w+\.)?(\w+)\s*=\s*\$%d\b`, paramIndex)
	whereRe := regexp.MustCompile(wherePattern)
	if match := whereRe.FindStringSubmatch(sql); len(match) > 1 {
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

	// ILIKE / LIKE / SIMILAR TO
	likePattern := fmt.Sprintf(`(?i)(?:\w+\.)?(\w+)\s+(?:I?LIKE|SIMILAR\s+TO|NOT\s+I?LIKE)\s+\S*\$%d\b`, paramIndex)
	likeRe := regexp.MustCompile(likePattern)
	if match := likeRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}
	// ILIKE/LIKE with concatenation: col ILIKE '%' || $N || '%'
	likeConcatPattern := fmt.Sprintf(`(?i)(?:\w+\.)?(\w+)\s+(?:I?LIKE|NOT\s+I?LIKE)\s+.*?\$%d\b`, paramIndex)
	likeConcatRe := regexp.MustCompile(likeConcatPattern)
	if match := likeConcatRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// Interval expressions: ($N || ' days')::INTERVAL or INTERVAL $N or ($N || ' hours')::INTERVAL
	intervalPattern := fmt.Sprintf(`(?i)\(\s*\$%d\s*\|\|\s*'[^']*'\s*\)\s*::\s*INTERVAL`, paramIndex)
	if regexp.MustCompile(intervalPattern).MatchString(sql) {
		return "days"
	}
	// Also: NOW() - INTERVAL '$N days' or INTERVAL '$N' DAY
	intervalPattern2 := fmt.Sprintf(`(?i)INTERVAL\s+\$%d\b`, paramIndex)
	if regexp.MustCompile(intervalPattern2).MatchString(sql) {
		return "days"
	}

	setPattern := fmt.Sprintf(`(?i)SET\s+(\w+)\s*=\s*\$%d`, paramIndex)
	setRe := regexp.MustCompile(setPattern)
	if match := setRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// SET col = col + $N or SET col = col - $N (counter increment/decrement)
	setCounterRe := regexp.MustCompile(fmt.Sprintf(`(?i)(\w+)\s*=\s*\w+\s*[+\-]\s*\$%d`, paramIndex))
	if match := setCounterRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1] + "_delta"
	}

	// SET col = COALESCE($N, col) — "update if not null" pattern
	setCoalesceRe := regexp.MustCompile(fmt.Sprintf(`(?i)(\w+)\s*=\s*COALESCE\s*\(\s*\$%d\b`, paramIndex))
	if match := setCoalesceRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
	}

	// Multi-assignment SET: ..., col = $N (not just first SET position)
	setAnyRe := regexp.MustCompile(fmt.Sprintf(`(?i)(\w+)\s*=\s*\$%d\b`, paramIndex))
	if match := setAnyRe.FindStringSubmatch(sql); len(match) > 1 {
		return match[1]
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

	// $N = ANY(col) or $N = ANY(alias.col)
	anyRe := regexp.MustCompile(fmt.Sprintf(`(?i)\$%d\s*=\s*ANY\s*\(\s*(?:\w+\.)?(\w+)\s*\)`, paramIndex))
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
	// Handles: WHERE (SELECT ...) = $N, WHERE ( SELECT ... ) = $N
	subqueryThresholdRe := regexp.MustCompile(fmt.Sprintf(`(?i)(?:WHERE|AND|OR)\s+\(\s*SELECT[\s\S]*?\)\s*[><=!]+\s*\$%d`, paramIndex))
	if subqueryThresholdRe.MatchString(sql) {
		return "count"
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

// inferInsertName maps the paramIndex-th ? of an INSERT to the column of the
// VALUES slot it occupies. Literals and expressions in sibling slots do not
// shift the mapping. INSERT..SELECT returns "" (params belong to the SELECT).
func inferInsertName(sql string, paramIndex int) string {
	re := regexp.MustCompile(`(?is)INSERT\s+(?:OR\s+\w+\s+)?INTO\s+\S+\s*\(([^)]*)\)\s*(?:VALUES|ROW\s+VALUES)\s*\(`)
	loc := re.FindStringSubmatchIndex(sql)
	if loc == nil {
		return ""
	}
	cols := splitTopLevelCommas(sql[loc[2]:loc[3]])

	valuesStart := loc[1] // just after the "VALUES (" opening paren
	depth := 1            // already inside the VALUES paren
	end := -1
	for i := valuesStart; i < len(sql); i++ {
		switch sql[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				end = i
			}
		case '\'', '"':
			// skip string literal
			quote := sql[i]
			for i++; i < len(sql) && sql[i] != quote; i++ {
				if sql[i] == '\\' {
					i++
				}
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return ""
	}
	slots := splitTopLevelCommas(sql[valuesStart:end])
	if len(slots) != len(cols) {
		return ""
	}

	qSeen := 0
	for i, slot := range slots {
		// $N-style: the slot references $paramIndex directly.
		if regexp.MustCompile(fmt.Sprintf(`\$%d\b`, paramIndex)).MatchString(slot) {
			return strings.TrimSpace(cols[i])
		}
		// ?-style: count ? occurrences across slots in order.
		for j := 0; j < strings.Count(slot, "?"); j++ {
			qSeen++
			if qSeen == paramIndex {
				return strings.TrimSpace(cols[i])
			}
		}
	}
	return ""
}

// splitTopLevelCommas splits on commas outside parentheses/quotes/brackets.
func splitTopLevelCommas(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case '\'', '"':
			quote := s[i]
			for i++; i < len(s) && s[i] != quote; i++ {
				if s[i] == '\\' {
					i++
				}
			}
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// inferPositionalName maps the paramIndex-th ? placeholder to the column it
// targets. It locates the placeholder inside the UPDATE ... SET clause or the
// trailing WHERE clause and names it after the assigned/compared column.
// Expression-wrapped placeholders (COALESCE(?, col), func(?)) inherit the
// column name; counter assignments (col = col + ?) get a _delta suffix.
// Placeholders it cannot attribute (CASE WHEN ?=?, bare SELECT ?) return ""
// so the caller keeps the ordered paramN fallback.
func inferPositionalName(sql string, paramIndex int) string {
	upper := strings.ToUpper(sql)

	setStart, setEnd := -1, -1
	if setLoc := regexp.MustCompile(`\bSET\s`).FindStringIndex(upper); setLoc != nil {
		setStart = setLoc[1]
		setEnd = len(sql)
		if w := strings.LastIndex(upper, "WHERE"); w > setStart {
			setEnd = w
		}
	}

	qPositions := make([]int, 0, 8)
	for i := 0; i < len(sql); i++ {
		if sql[i] == '?' {
			qPositions = append(qPositions, i)
		}
	}
	if paramIndex > len(qPositions) {
		return ""
	}
	pos := qPositions[paramIndex-1]

	if setStart >= 0 && pos >= setStart && pos < setEnd {
		return nameForSetPlaceholder(sql[setStart:pos])
	}

	// Resolve against the WHERE clause nearest BEFORE this placeholder, so
	// UNION/multi-SELECT statements map each branch's params to its own
	// WHERE columns (LastIndex-WHERE only sees the final branch).
	if w := strings.LastIndex(upper[:pos], "WHERE"); w >= 0 {
		start := w + 5
		end := len(sql)
		if e := regexp.MustCompile(`\b(LIMIT|ORDER\s+BY|GROUP\s+BY|HAVING|ALLOW\s+FILTERING)\b`).FindStringIndex(upper[start:]); e != nil {
			end = start + e[0]
		}
		if pos < end {
			return nameForWherePlaceholder(sql[start:pos])
		}
	}
	return ""
}

// nameForSetPlaceholder names a SET-clause ? from the clause text preceding it.
func nameForSetPlaceholder(prefix string) string {
	// col = col + ? / col = col - ? → counter delta
	if m := regexp.MustCompile(`(\w+)\s*=\s*\w+\s*[+\-]\s*$`).FindStringSubmatch(prefix); len(m) > 1 {
		return m[1] + "_delta"
	}
	// col = ?, col = COALESCE(?, ...), col = func(func(?...
	if m := regexp.MustCompile(`(\w+)\s*=\s*(?:\w+\s*\(\s*)*$`).FindStringSubmatch(prefix); len(m) > 1 {
		return m[1]
	}
	return ""
}

// nameForWherePlaceholder names a WHERE-clause ? from the clause text preceding it.
func nameForWherePlaceholder(prefix string) string {
	// col = ?, col = COALESCE(?, col), col = func(... func(?, alias.col = ?
	if m := regexp.MustCompile(`(?:\w+\.)?(\w+)\s*=\s*(?:\w+\s*\(\s*)*$`).FindStringSubmatch(prefix); len(m) > 1 {
		return m[1]
	}
	// col >= ? / col <= ? / col != ? / col > ? / col < ?
	if m := regexp.MustCompile(`(?:\w+\.)?(\w+)\s*(?:>=|<=|!=|>|<)\s*$`).FindStringSubmatch(prefix); len(m) > 1 {
		return m[1]
	}
	// col LIKE ? / col ILIKE '%' || ? (concatenated search)
	if m := regexp.MustCompile(`(?:\w+\.)?(\w+)\s+(?:NOT\s+)?I?LIKE\s+.*$`).FindStringSubmatch(prefix); len(m) > 1 {
		return m[1]
	}
	// col CONTAINS ? (ScyllaDB)
	if m := regexp.MustCompile(`(?:\w+\.)?(\w+)\s+CONTAINS\s+$`).FindStringSubmatch(prefix); len(m) > 1 {
		return m[1]
	}
	return ""
}
