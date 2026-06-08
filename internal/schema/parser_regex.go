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

	// ALTER TABLE statements for schema-level changes
	alterTableRenameColRegex   = regexp.MustCompile(`(?i)ALTER\s+TABLE\s+(?:"?(\w+)"?)\s+RENAME\s+COLUMN\s+(?:"?(\w+)"?)\s+TO\s+(?:"?(\w+)"?)`)
	alterTableRenameTableRegex = regexp.MustCompile(`(?i)ALTER\s+TABLE\s+(?:"?(\w+)"?)\s+RENAME\s+TO\s+(?:"?(\w+)"?)`)
	alterTableSetNotNullRegex  = regexp.MustCompile(`(?i)ALTER\s+TABLE\s+(?:"?(\w+)"?)\s+ALTER\s+COLUMN\s+(?:"?(\w+)"?)\s+(SET\s+NOT\s+NULL|DROP\s+NOT\s+NULL)`)
	alterTableSetDefaultRegex  = regexp.MustCompile(`(?i)ALTER\s+TABLE\s+(?:"?(\w+)"?)\s+ALTER\s+COLUMN\s+(?:"?(\w+)"?)\s+SET\s+DEFAULT\s+(.+)`)
	alterTableDropDefaultRegex = regexp.MustCompile(`(?i)ALTER\s+TABLE\s+(?:"?(\w+)"?)\s+ALTER\s+COLUMN\s+(?:"?(\w+)"?)\s+DROP\s+DEFAULT`)

	commentRegex    = regexp.MustCompile(`--.*|/\*[\s\S]*?\*/`)
	whitespaceRegex = regexp.MustCompile(`\s+`)
	enumValueRegex  = regexp.MustCompile(`'([^']+)'`)
)
