package validation

import "strings"

// stripOverClauses removes all OVER(...) blocks from SQL to prevent
// ORDER BY inside window functions from being matched by clause-level regexes.
func stripOverClauses(sql string) string {
	sqlUpper := strings.ToUpper(sql)
	var result strings.Builder
	result.Grow(len(sql))
	i := 0
	for i < len(sql) {
		if i+4 <= len(sql) && sqlUpper[i:i+4] == "OVER" {
			next := i + 4
			for next < len(sql) && (sql[next] == ' ' || sql[next] == '\t' || sql[next] == '\n') {
				next++
			}
			if next < len(sql) && sql[next] == '(' {
				result.WriteString(sql[i:next])
				depth := 0
				for next < len(sql) {
					if sql[next] == '(' {
						depth++
					} else if sql[next] == ')' {
						depth--
						if depth == 0 {
							result.WriteByte(')')
							next++
							break
						}
					}
					next++
				}
				i = next
				continue
			}
		}
		result.WriteByte(sql[i])
		i++
	}
	return result.String()
}

// stripSubqueryBlocks replaces parenthesized subquery blocks with empty parens
// so clause-level regexes don't match keywords inside subqueries.
func stripSubqueryBlocks(sql string) string {
	sqlUpper := strings.ToUpper(sql)
	var result strings.Builder
	result.Grow(len(sql))
	i := 0
	for i < len(sql) {
		if sql[i] == '(' {
			after := i + 1
			for after < len(sql) && (sql[after] == ' ' || sql[after] == '\t' || sql[after] == '\n') {
				after++
			}
			isSubQuery := false
			if after+6 <= len(sql) && sqlUpper[after:after+6] == "SELECT" &&
				(after+6 >= len(sql) || !isAlphaNum(sql[after+6])) {
				isSubQuery = true
			}
			if !isSubQuery && after+4 <= len(sql) && sqlUpper[after:after+4] == "WITH" &&
				(after+4 >= len(sql) || !isAlphaNum(sql[after+4])) {
				isSubQuery = true
			}
			if isSubQuery {
				result.WriteString("(...)")
				depth := 1
				i++
				for i < len(sql) && depth > 0 {
					if sql[i] == '(' {
						depth++
					} else if sql[i] == ')' {
						depth--
					}
					i++
				}
				continue
			}
		}
		result.WriteByte(sql[i])
		i++
	}
	return result.String()
}

// stripStringLiterals removes single-quoted SQL literals so their contents
// don't get falsely flagged as column references.
func stripStringLiterals(sql string) string {
	var result strings.Builder
	result.Grow(len(sql))
	inString := false
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if ch == '\'' {
			if inString && i+1 < len(sql) && sql[i+1] == '\'' {
				result.WriteString("''")
				i++
				continue
			}
			inString = !inString
			result.WriteString("''")
			continue
		}
		if !inString {
			result.WriteByte(ch)
		}
	}
	return result.String()
}

// isAlphaNum checks if a byte is an alphanumeric character or underscore
func isAlphaNum(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}
