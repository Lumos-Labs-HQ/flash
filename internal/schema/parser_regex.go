package schema

import (
	"regexp"
)

var (
	tableRegex = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:"?(\w+)"?|(\w+)|` + "`" + `(\w+)` + "`" + `)\s*\(`)
	enumRegex  = regexp.MustCompile(`(?i)CREATE\s+TYPE\s+(?:"?(\w+)"?|(\w+))\s+AS\s+ENUM\s*\(\s*([^)]+)\s*\)`)

	// Index — captures method (USING ...) and WHERE separately
	indexRegex      = regexp.MustCompile(`(?i)CREATE\s+(UNIQUE\s+)?INDEX\s+(?:CONCURRENTLY\s+)?(?:IF\s+NOT\s+EXISTS\s+)?(?:"?(\w+)"?|(\w+))\s+ON\s+(?:"?(\w+)"?|(\w+))\s*\(([^)]+)\)`)
	indexOrderRegex = regexp.MustCompile(`(?i)\s+(ASC|DESC)$`)
	indexWhereRegex = regexp.MustCompile(`(?i)\s+WHERE\s+(.+)$`)
	indexUsingRegex = regexp.MustCompile(`(?i)\s+USING\s+(\w+)`)

	createTableStmtRegex = regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE`)
	createIndexStmtRegex = regexp.MustCompile(`(?i)^\s*CREATE\s+(UNIQUE\s+)?INDEX`)
	createTypeStmtRegex  = regexp.MustCompile(`(?i)^\s*CREATE\s+TYPE\s+\w+\s+AS\s+ENUM`)

	commentRegex    = regexp.MustCompile(`--.*|/\*[\s\S]*?\*/`)
	whitespaceRegex = regexp.MustCompile(`\s+`)
	enumValueRegex  = regexp.MustCompile(`'([^']+)'`)
)
