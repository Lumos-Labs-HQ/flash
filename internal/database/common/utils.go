package common

import (
	"strings"
)

type QueryResult struct {
	Columns []string
	Rows    []map[string]interface{}
}

// ParseSQLStatements splits SQL migration text into individual statements,
// correctly handling string literals, comments, and PostgreSQL dollar-quoted
// blocks so that semicolons inside them do not terminate statements.
func ParseSQLStatements(sql string) []string {
	var statements []string
	var current strings.Builder

	const (
		normal = iota
		singleQuote
		doubleQuote
		backtick
		lineComment
		blockComment
		dollarQuote
	)
	state := normal
	var dollarTag string

	for i := 0; i < len(sql); {
		ch := sql[i]

		switch state {
		case lineComment:
			if ch == '\n' {
				state = normal
			}
			i++
			continue

		case blockComment:
			if ch == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				state = normal
				i += 2
				continue
			}
			i++
			continue

		case singleQuote:
			if ch == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					current.WriteString("''")
					i += 2
					continue
				}
				state = normal
			}
			current.WriteByte(ch)
			i++
			continue

		case doubleQuote:
			if ch == '"' {
				if i+1 < len(sql) && sql[i+1] == '"' {
					current.WriteString("\"\"")
					i += 2
					continue
				}
				state = normal
			}
			current.WriteByte(ch)
			i++
			continue

		case backtick:
			if ch == '`' {
				if i+1 < len(sql) && sql[i+1] == '`' {
					current.WriteString("``")
					i += 2
					continue
				}
				state = normal
			}
			current.WriteByte(ch)
			i++
			continue

		case dollarQuote:
			if ch == '$' {
				// Try to find a matching closing tag
				tagEnd := i + 1
				for tagEnd < len(sql) && sql[tagEnd] != '$' {
					if !isDollarTagChar(sql[tagEnd]) {
						break
					}
					tagEnd++
				}
				if tagEnd < len(sql) && sql[tagEnd] == '$' {
					tag := sql[i : tagEnd+1]
					if tag == dollarTag {
						state = normal
						current.WriteString(tag)
						i = tagEnd + 1
						continue
					}
				}
			}
			current.WriteByte(ch)
			i++
			continue
		}

		// state == normal
		if ch == '-' && i+1 < len(sql) && sql[i+1] == '-' {
			state = lineComment
			i += 2
			continue
		}

		if ch == '/' && i+1 < len(sql) && sql[i+1] == '*' {
			state = blockComment
			i += 2
			continue
		}

		if ch == '\'' {
			state = singleQuote
			current.WriteByte(ch)
			i++
			continue
		}

		if ch == '"' {
			state = doubleQuote
			current.WriteByte(ch)
			i++
			continue
		}

		if ch == '`' {
			state = backtick
			current.WriteByte(ch)
			i++
			continue
		}

		if ch == '$' {
			tagEnd := i + 1
			for tagEnd < len(sql) && isDollarTagChar(sql[tagEnd]) {
				tagEnd++
			}
			if tagEnd < len(sql) && sql[tagEnd] == '$' {
				dollarTag = sql[i : tagEnd+1]
				state = dollarQuote
				current.WriteString(dollarTag)
				i = tagEnd + 1
				continue
			}
		}

		if ch == ';' {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			i++
			continue
		}

		current.WriteByte(ch)
		i++
	}

	if current.Len() > 0 {
		stmt := strings.TrimSpace(current.String())
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}

	return statements
}

func isDollarTagChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}
