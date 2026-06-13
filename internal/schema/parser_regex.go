package schema

import (
	"regexp"
)

var (
	// Matches: CREATE TABLE users ( | CREATE TABLE "myapp"."posts" ( | CREATE TABLE IF NOT EXISTS visa_app.applications (
	// Captures the full table name including quotes and dots via \S+ (non-whitespace run before opening paren).
	tableRegex = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s*\(`)
	enumRegex  = regexp.MustCompile(`(?i)CREATE\s+TYPE\s+(?:"?(\w+)"?|(\w+))\s+AS\s+ENUM\s*\(\s*([^)]+)\s*\)`)

	// Index — captures (UNIQUE? INDEX name ON table(cols))
	indexRegex      = regexp.MustCompile(`(?i)CREATE\s+(UNIQUE\s+)?INDEX\s+(?:CONCURRENTLY\s+)?(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s+ON\s+(\S+)\s*\(([^)]+)\)`)
	indexOrderRegex = regexp.MustCompile(`(?i)\s+(ASC|DESC)$`)
	indexWhereRegex = regexp.MustCompile(`(?i)\s+WHERE\s+(.+)$`)
	indexUsingRegex = regexp.MustCompile(`(?i)\s+USING\s+(\w+)`)

	createTableStmtRegex = regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE`)
	createIndexStmtRegex = regexp.MustCompile(`(?i)^\s*CREATE\s+(UNIQUE\s+)?INDEX`)
	createTypeStmtRegex  = regexp.MustCompile(`(?i)^\s*CREATE\s+TYPE\s+\w+\s+AS\s+ENUM`)
	createKeyspaceRegex  = regexp.MustCompile(`(?i)CREATE\s+KEYSPACE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s+WITH\s+REPLICATION\s*=\s*(\{[^}]+\})(?:\s+AND\s+DURABLE_WRITES\s*=\s*(true|false))?`)
	createViewRegex      = regexp.MustCompile(`(?i)CREATE\s+(?:MATERIALIZED\s+)?VIEW\s+(?:IF\s+NOT\s+EXISTS\s+)?(\S+)\s+AS\s+SELECT`)
	createViewStmtRegex  = regexp.MustCompile(`(?i)^\s*CREATE\s+(?:MATERIALIZED\s+)?VIEW`)
	createUDTStmtRegex   = regexp.MustCompile(`(?i)^\s*CREATE\s+TYPE\s+\S+\s*\(`)
	createUDTFullRegex   = regexp.MustCompile(`(?i)CREATE\s+TYPE\s+(\S+)\s*\(([\s\S]*?)\)`)

	commentRegex    = regexp.MustCompile(`--.*|/\*[\s\S]*?\*/`)
	whitespaceRegex = regexp.MustCompile(`\s+`)
	enumValueRegex  = regexp.MustCompile(`'([^']+)'`)
)
