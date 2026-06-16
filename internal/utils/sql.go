package utils

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Pre-compiled regex patterns for SQL parsing (performance optimization)
var (
	blockCommentRegex   = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	insertIntoRegex     = regexp.MustCompile(`(?i)INSERT\s+INTO\s+(\S+)`)
	fromTableRegex      = regexp.MustCompile(`(?i)FROM\s+(\S+)`)
	updateTableRegex    = regexp.MustCompile(`(?i)UPDATE\s+(\S+)`)
	deleteFromRegex     = regexp.MustCompile(`(?i)DELETE\s+FROM\s+(\S+)`)
	modifyingQueryRegex = regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE)\b`)
)

// ExtractTableName extracts the primary table name from a SQL query.
// Works with INSERT INTO, SELECT FROM, UPDATE, and DELETE FROM statements.
// Uses word-boundary matching to avoid matching function-internal FROM (e.g. EXTRACT(EPOCH FROM ...)).
func ExtractTableName(sql string) string {
	sqlUpper := strings.ToUpper(sql)

	if strings.Contains(sqlUpper, "INSERT INTO") {
		if matches := insertIntoRegex.FindStringSubmatch(sql); len(matches) > 1 {
			return matches[1]
		}
	}

	// Match FROM/UPDATE/DELETE as standalone keywords (word boundary), not inside functions.
	// Strip parentheses content first to avoid matching EXTRACT(EPOCH FROM ...).
	stripped := StripParenthesizedContent(sqlUpper)
	if matches := fromTableRegex.FindStringSubmatch(stripped); len(matches) > 1 {
		return strings.ToLower(matches[1])
	}

	if matches := updateTableRegex.FindStringSubmatch(stripped); len(matches) > 1 {
		return strings.ToLower(matches[1])
	}

	if matches := deleteFromRegex.FindStringSubmatch(stripped); len(matches) > 1 {
		return strings.ToLower(matches[1])
	}

	return ""
}

// StripParenthesizedContent replaces everything between balanced parentheses with spaces.
func StripParenthesizedContent(s string) string {
	var result strings.Builder
	result.Grow(len(s))
	depth := 0
	for _, ch := range s {
		if ch == '(' {
			depth++
		} else if ch == ')' {
			if depth > 0 {
				depth--
			}
		} else if depth == 0 {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// IsModifyingQuery returns true if the SQL contains INSERT, UPDATE, or DELETE.
func IsModifyingQuery(sql string) bool {
	return modifyingQueryRegex.MatchString(sql)
}

func RemoveComments(sql string) string {
	var result strings.Builder
	result.Grow(len(sql)) // Pre-allocate buffer

	start := 0
	for i := 0; i < len(sql); i++ {
		// Check for line comment
		if i+1 < len(sql) && sql[i] == '-' && sql[i+1] == '-' {
			result.WriteString(sql[start:i])
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			if i < len(sql) {
				result.WriteByte('\n')
			}
			start = i + 1
		}
	}
	result.WriteString(sql[start:])

	// Remove block comments
	return blockCommentRegex.ReplaceAllString(result.String(), "")
}

// SplitColumns splits a comma-separated column string, respecting parentheses depth.
// This handles cases like "col1, COALESCE(a, b), col2" correctly.
// Also handles CQL angle-bracket types like map<text,text> by tracking <> depth
// only outside parentheses (inside parens, < is a SQL comparison operator).
func SplitColumns(columnsStr string) []string {
	result := make([]string, 0, 8)
	var current strings.Builder
	current.Grow(64)
	parenDepth := 0
	angleDepth := 0
	inString := false

	for i := 0; i < len(columnsStr); i++ {
		ch := columnsStr[i]
		if inString {
			current.WriteByte(ch)
			if ch == '\'' {
				if i+1 < len(columnsStr) && columnsStr[i+1] == '\'' {
					i++
					current.WriteByte(columnsStr[i])
				} else {
					inString = false
				}
			}
			continue
		}
		switch ch {
		case '\'':
			inString = true
			current.WriteByte(ch)
		case '(':
			parenDepth++
			current.WriteByte(ch)
		case ')':
			parenDepth--
			current.WriteByte(ch)
		case '<':
			// Only track angle brackets at depth 0 (CQL types: frozen<type>, map<k,v>).
			// Inside parens, < is a comparison operator (e.g. age < 18, ARRAY[0,10)).
			if parenDepth == 0 {
				angleDepth++
			}
			current.WriteByte(ch)
		case '>':
			if parenDepth == 0 && angleDepth > 0 {
				angleDepth--
			}
			current.WriteByte(ch)
		case ',':
			if parenDepth == 0 && angleDepth == 0 {
				result = append(result, current.String())
				current.Reset()
				current.Grow(64)
			} else {
				current.WriteByte(ch)
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// SmartSplitColumns is an alias for SplitColumns for backward compatibility.
// Deprecated: Use SplitColumns directly.
func SmartSplitColumns(columnsStr string) []string {
	return SplitColumns(columnsStr)
}

func ExtractSelectColumns(sql string) string {
	sqlUpper := strings.ToUpper(sql)
	sqlTrimmed := strings.TrimSpace(sqlUpper)

	if strings.HasPrefix(sqlTrimmed, "(") {
		parenDepth := 0
		for i := 0; i < len(sqlUpper)-6; i++ {
			switch sql[i] {
			case '(':
				parenDepth++
			case ')':
				parenDepth--
			case 'S', 's':
				if parenDepth == 1 && i+6 <= len(sqlUpper) {
					if sqlUpper[i:i+6] == "SELECT" {
						if (i == 0 || !isAlphaNum(sql[i-1])) &&
							(i+6 >= len(sql) || !isAlphaNum(sql[i+6])) {
							return extractColumnsFromSelect(sql, i)
						}
					}
				}
			}
		}
	}

	if strings.HasPrefix(sqlTrimmed, "WITH") {
		var selectPositions []int
		parenDepth := 0

		for i := 0; i < len(sqlUpper)-6; i++ {
			switch sql[i] {
			case '(':
				parenDepth++
			case ')':
				parenDepth--
			case 'S', 's':
				if parenDepth == 0 && i+6 <= len(sqlUpper) {
					if sqlUpper[i:i+6] == "SELECT" {
						if (i == 0 || !isAlphaNum(sql[i-1])) &&
							(i+6 >= len(sql) || !isAlphaNum(sql[i+6])) {
							selectPositions = append(selectPositions, i)
						}
					}
				}
			}
		}

		if len(selectPositions) > 0 {
			selectIdx := selectPositions[len(selectPositions)-1]
			return extractColumnsFromSelect(sql, selectIdx)
		}
	}

	selectIdx := strings.Index(sqlUpper, "SELECT")
	if selectIdx == -1 {
		return ""
	}

	return extractColumnsFromSelect(sql, selectIdx)
}

func extractColumnsFromSelect(sql string, selectIdx int) string {
	sqlUpper := strings.ToUpper(sql)
	start := selectIdx + 6
	for start < len(sql) && (sql[start] == ' ' || sql[start] == '\t' || sql[start] == '\n') {
		start++
	}

	// Skip DISTINCT ON (col1, col2) — it's not part of the column list
	restUpper := sqlUpper[start:]
	if strings.HasPrefix(restUpper, "DISTINCT") {
		distinctEnd := start + 8 // "DISTINCT"
		after := distinctEnd
		for after < len(sql) && (sql[after] == ' ' || sql[after] == '\t' || sql[after] == '\n') {
			after++
		}
		if after+2 < len(sql) && strings.ToUpper(sql[after:after+2]) == "ON" {
			// Skip "ON (col1, col2)"
			after += 2
			for after < len(sql) && (sql[after] == ' ' || sql[after] == '\t' || sql[after] == '\n') {
				after++
			}
			if after < len(sql) && sql[after] == '(' {
				depth := 0
				for after < len(sql) {
					if sql[after] == '(' {
						depth++
					} else if sql[after] == ')' {
						depth--
						if depth == 0 {
							after++
							break
						}
					}
					after++
				}
				start = after
				for start < len(sql) && (sql[start] == ' ' || sql[start] == '\t' || sql[start] == '\n') {
					start++
				}
			}
		}
	}

	parenDepth := 0
	fromIdx := -1

	for i := start; i < len(sql); i++ {
		switch sql[i] {
		case '(':
			parenDepth++
		case ')':
			parenDepth--
		case 'F', 'f':
			if parenDepth == 0 && i+4 <= len(sql) {
				if sqlUpper[i:i+4] == "FROM" {
					if (i == 0 || !isAlphaNum(sql[i-1])) &&
						(i+4 >= len(sql) || !isAlphaNum(sql[i+4])) {
						fromIdx = i
						break
					}
				}
			}
		case ';':
			if parenDepth == 0 && fromIdx == -1 {
				return strings.TrimSpace(sql[start:i])
			}
		}

		if fromIdx != -1 {
			break
		}
	}

	if fromIdx != -1 {
		return strings.TrimSpace(sql[start:fromIdx])
	}

	end := len(sql)
	for i := start; i < len(sql); i++ {
		if sql[i] == ';' {
			end = i
			break
		}
	}

	return strings.TrimSpace(sql[start:end])
}

func ContainsSQLKeyword(sql, keyword string) bool {
	keyword = strings.ToUpper(keyword)
	sql = strings.ToUpper(sql)

	index := 0
	for {
		pos := strings.Index(sql[index:], keyword)
		if pos == -1 {
			return false
		}

		absPos := index + pos
		beforeOK := absPos == 0 || !isAlphaNum(sql[absPos-1])
		afterPos := absPos + len(keyword)
		afterOK := afterPos >= len(sql) || !isAlphaNum(sql[afterPos])

		if beforeOK && afterOK {
			return true
		}

		index = absPos + 1
	}
}

func isAlphaNum(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// SQL keywords map
var sqlKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "JOIN": true, "INNER": true,
	"LEFT": true, "RIGHT": true, "OUTER": true, "ON": true, "AND": true,
	"OR": true, "NOT": true, "IN": true, "LIKE": true, "ILIKE": true, "SIMILAR": true,
	"BETWEEN": true, "IS": true, "NULL": true, "GROUP": true, "BY": true, "HAVING": true,
	"ORDER": true, "ASC": true, "DESC": true, "LIMIT": true, "OFFSET": true,
	"INSERT": true, "UPDATE": true, "DELETE": true, "CREATE": true, "DROP": true,
	"ALTER": true, "TABLE": true, "INDEX": true, "VIEW": true, "AS": true,
	"DISTINCT": true, "COUNT": true, "SUM": true, "AVG": true, "MIN": true,
	"MAX": true, "CASE": true, "WHEN": true, "THEN": true, "ELSE": true,
	"END": true, "WITH": true, "RECURSIVE": true, "ANY": true, "ALL": true,
	"EXISTS": true, "NULLS": true, "LAST": true, "FIRST": true, "FILTER": true,
	"OVER": true, "PARTITION": true, "ROWS": true, "RANGE": true, "FOLLOWING": true,
	"PRECEDING": true, "UNBOUNDED": true, "CURRENT": true, "ROW": true,
	"LATERAL": true, "PLAINTO_TSQUERY": true, "TS_RANK": true,
	"UNNEST": true, "INTERVAL": true,
	"EXCEPT": true, "INTERSECT": true, "UNION": true, "RETURNING": true,
	"CONFLICT": true, "DO": true, "NOTHING": true, "EXCLUDED": true,
}

func IsSQLKeyword(word string) bool {
	return sqlKeywords[strings.ToUpper(word)]
}

func ValidateSchemaSyntax(content, filePath string) error {
	lines := strings.Split(content, "\n")

	inCreateTable := false
	tableStartLine := 0
	parenDepth := 0

	for lineNum, line := range lines {
		lineNumber := lineNum + 1
		trimmed := strings.TrimSpace(line)
		upper := strings.ToUpper(trimmed)

		if strings.Contains(upper, "CREATE TABLE") {
			inCreateTable = true
			tableStartLine = lineNumber
			parenDepth = 0
		}

		// Track paren depth inside CREATE TABLE
		if inCreateTable {
			for _, ch := range line {
				switch ch {
				case '(':
					parenDepth++
				case ')':
					parenDepth--
				}
			}
		}

		// End of CREATE TABLE: look for ");
		if inCreateTable && strings.Contains(trimmed, ");") {
			for i := lineNum - 1; i >= 0; i-- {
				prevLine := strings.TrimSpace(lines[i])
				prevUpper := strings.ToUpper(prevLine)
				if prevLine == "" {
					continue
				}
				// CQL-compatible: skip PRIMARY KEY lines (trailing commas required in CQL)
				if strings.HasPrefix(prevUpper, "PRIMARY KEY") {
					break
				}
				if strings.HasSuffix(prevLine, ",") {
					relPath := filepath.Base(filePath)
					return fmt.Errorf("# package FlashORM\n%s:%d:2: syntax error at or near \")\"", relPath, lineNumber)
				}
				break
			}
			inCreateTable = false
			parenDepth = 0
		}

		// Only check for unmatched ')' when NOT inside a CREATE TABLE.
		// CQL uses nested parens for composite partition keys like
		// PRIMARY KEY ((col1, col2), col3) which naturally resolves
		// multiple paren levels in one line.
		if parenDepth < 0 && !inCreateTable {
			relPath := filepath.Base(filePath)
			return fmt.Errorf("# package flash\n%s:%d:2: syntax error: unexpected ')'", relPath, lineNumber)
		}
	}

	if inCreateTable && parenDepth > 0 {
		relPath := filepath.Base(filePath)
		return fmt.Errorf("# package flash\n%s:%d:2: syntax error: unclosed CREATE TABLE statement", relPath, tableStartLine)
	}

	return nil
}
