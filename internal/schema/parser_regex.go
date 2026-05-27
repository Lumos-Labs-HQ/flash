package schema

import (
	"regexp"
)

// Pre-compile regexes once at init to avoid recompilation on every parse.

var (
	// Table and type parsing
	tableRegex = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:"?(\w+)"?|(\w+)|` + "`" + `(\w+)` + "`" + `)\s*\(`)
	enumRegex  = regexp.MustCompile(`(?i)CREATE\s+TYPE\s+(?:"?(\w+)"?|(\w+))\s+AS\s+ENUM\s*\(\s*([^)]+)\s*\)`)
	
	// Index parsing — captures up to the column list; WHERE is extracted separately
	indexRegex     = regexp.MustCompile(`(?i)CREATE\s+(UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:"?(\w+)"?|(\w+))\s+ON\s+(?:"?(\w+)"?|(\w+))\s*\(\s*([^)]+)\s*\)`)
	indexOrderRegex = regexp.MustCompile(`(?i)\s+(ASC|DESC)$`)
	indexWhereRegex = regexp.MustCompile(`(?i)\s+WHERE\s+(.+)$`)
	
	// Statement detection
	createTableStmtRegex = regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE`)
	createIndexStmtRegex = regexp.MustCompile(`(?i)^\s*CREATE\s+(UNIQUE\s+)?INDEX`)
	createTypeStmtRegex  = regexp.MustCompile(`(?i)^\s*CREATE\s+TYPE\s+\w+\s+AS\s+ENUM`)
	
	// Cleaning
	commentRegex     = regexp.MustCompile(`--.*|/\*[\s\S]*?\*/`)
	whitespaceRegex  = regexp.MustCompile(`\s+`)
	enumValueRegex   = regexp.MustCompile(`'([^']+)'`)
)
