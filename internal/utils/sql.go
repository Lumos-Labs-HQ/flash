package utils

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var blockCommentRegex = regexp.MustCompile(`/\*[\s\S]*?\*/`)

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

func SplitColumns(columnsStr string) []string {
	result := make([]string, 0, 8) // Pre-allocate with reasonable capacity
	var current strings.Builder
	current.Grow(64) // Pre-allocate buffer for column strings
	parenDepth := 0

	for i := 0; i < len(columnsStr); i++ {
		char := columnsStr[i]
		switch char {
		case '(':
			parenDepth++
			current.WriteByte(char)
		case ')':
			parenDepth--
			current.WriteByte(char)
		case ',':
			if parenDepth == 0 {
				result = append(result, current.String())
				current.Reset()
				current.Grow(64) // Reset with pre-allocation
			} else {
				current.WriteByte(char)
			}
		default:
			current.WriteByte(char)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

func SmartSplitColumns(columnsStr string) []string {
	var result []string
	var current strings.Builder
	parenDepth := 0

	for _, char := range columnsStr {
		switch char {
		case '(':
			parenDepth++
			current.WriteRune(char)
		case ')':
			parenDepth--
			current.WriteRune(char)
		case ',':
			if parenDepth == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
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
					if strings.ToUpper(sql[i:i+6]) == "SELECT" {
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
					if strings.ToUpper(sql[i:i+6]) == "SELECT" {
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
	start := selectIdx + 6
	for start < len(sql) && (sql[start] == ' ' || sql[start] == '\t' || sql[start] == '\n') {
		start++
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
				potential := strings.ToUpper(sql[i:min(i+4, len(sql))])
				if potential == "FROM" {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SQL keywords map
var sqlKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "JOIN": true, "INNER": true,
	"LEFT": true, "RIGHT": true, "OUTER": true, "ON": true, "AND": true,
	"OR": true, "NOT": true, "IN": true, "LIKE": true, "BETWEEN": true,
	"IS": true, "NULL": true, "GROUP": true, "BY": true, "HAVING": true,
	"ORDER": true, "ASC": true, "DESC": true, "LIMIT": true, "OFFSET": true,
	"INSERT": true, "UPDATE": true, "DELETE": true, "CREATE": true, "DROP": true,
	"ALTER": true, "TABLE": true, "INDEX": true, "VIEW": true, "AS": true,
	"DISTINCT": true, "COUNT": true, "SUM": true, "AVG": true, "MIN": true,
	"MAX": true, "CASE": true, "WHEN": true, "THEN": true, "ELSE": true,
	"END": true, "WITH": true, "RECURSIVE": true,
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

		if strings.Contains(strings.ToUpper(trimmed), "CREATE TABLE") {
			inCreateTable = true
			tableStartLine = lineNumber
			parenDepth = 0
		}

		for _, ch := range line {
			switch ch {
			case '(':
				parenDepth++
			case ')':
				parenDepth--
			}
		}

		if inCreateTable && parenDepth == 0 && strings.Contains(trimmed, ");") {
			for i := lineNum - 1; i >= 0; i-- {
				prevLine := strings.TrimSpace(lines[i])
				if prevLine == "" {
					continue
				}
				if strings.HasSuffix(prevLine, ",") {
					relPath := filepath.Base(filePath)
					return fmt.Errorf("# package FlashORM\n%s:%d:2: syntax error at or near \")\"", relPath, lineNumber)
				}
				break
			}
			inCreateTable = false
		}

		if parenDepth < 0 {
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
