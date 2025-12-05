package pull

import (
	"fmt"
	"regexp"
	"strings"
)

// extractTableSQL extracts the CREATE TABLE statement for a specific table from content
func (s *Service) extractTableSQL(content, tableName string) string {
	startPattern := fmt.Sprintf(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?["'\x60]?%s["'\x60]?\s*\(`, regexp.QuoteMeta(tableName))
	startRe := regexp.MustCompile(startPattern)
	startMatch := startRe.FindStringIndex(content)
	if startMatch == nil {
		return ""
	}

	// Find the matching closing parenthesis
	start := startMatch[0]
	parenStart := startMatch[1] - 1
	depth := 0
	endPos := -1

	for i := parenStart; i < len(content); i++ {
		if content[i] == '(' {
			depth++
		} else if content[i] == ')' {
			depth--
			if depth == 0 {
				endPos = i + 1
				break
			}
		}
	}

	if endPos == -1 {
		return ""
	}

	// Find the semicolon after the closing paren
	semiPos := strings.Index(content[endPos:], ";")
	if semiPos != -1 {
		endPos = endPos + semiPos + 1
	}

	tableSQL := content[start:endPos]

	// Also capture any CREATE INDEX statements that follow
	remaining := content[endPos:]
	indexPattern := regexp.MustCompile(`(?i)^\s*(CREATE\s+(?:UNIQUE\s+)?INDEX\s+[^;]+;)`)
	for {
		match := indexPattern.FindStringSubmatch(remaining)
		if match == nil {
			break
		}
		tableSQL += "\n" + strings.TrimSpace(match[1])
		remaining = remaining[len(match[0]):]
	}

	return strings.TrimSpace(tableSQL)
}

// replaceTableInContent replaces a table definition in the content
func (s *Service) replaceTableInContent(content, tableName, newTableSQL string) string {
	existingSQL := s.extractTableSQL(content, tableName)
	if existingSQL == "" {
		return content
	}
	return strings.Replace(content, existingSQL, newTableSQL, 1)
}

// compareTableSQL compares two table SQL definitions (ignoring whitespace differences)
func (s *Service) compareTableSQL(sql1, sql2 string) bool {
	normalize := func(sql string) string {
		sql = strings.ToLower(sql)
		sql = regexp.MustCompile(`\s+`).ReplaceAllString(sql, " ")
		sql = strings.TrimSpace(sql)
		return sql
	}
	return normalize(sql1) == normalize(sql2)
}

// isFileCommentedOut checks if a file is already fully commented out
func (s *Service) isFileCommentedOut(content string) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "--") {
			return false
		}
	}
	return true
}

func (s *Service) commentOutFile(content, tableName string) string {
	var sb strings.Builder

	sb.WriteString("-- ============================================================\n")
	sb.WriteString(fmt.Sprintf("-- TABLE DROPPED: '%s' no longer exists in database\n", tableName))
	sb.WriteString("-- This file has been commented out by 'flash pull'\n")
	sb.WriteString("-- You can delete this file or uncomment to recreate the table\n")
	sb.WriteString("-- ============================================================\n\n")

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			sb.WriteString("\n")
		} else if strings.HasPrefix(trimmed, "--") {
			sb.WriteString(line + "\n")
		} else {
			sb.WriteString("-- " + line + "\n")
		}
	}

	return sb.String()
}
