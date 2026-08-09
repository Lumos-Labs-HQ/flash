package validation

import "regexp"

var (
	ctePatternRegex            *regexp.Regexp
	tablePatternRegex          *regexp.Regexp
	tableAliasPatternRegex     *regexp.Regexp
	joinPatternRegex           *regexp.Regexp
	columnRefPatternRegex      *regexp.Regexp
	aliasExtractPatternRegex   *regexp.Regexp
	fromPatternRegex           *regexp.Regexp
	insertPatternRegex         *regexp.Regexp
	joinCheckRegex             *regexp.Regexp
	whereClauseRegex           *regexp.Regexp
	setClauseRegex             *regexp.Regexp
	orderByClauseRegex         *regexp.Regexp
	groupByClauseRegex         *regexp.Regexp
	havingClauseRegex          *regexp.Regexp
	paramCheckRegex            *regexp.Regexp
	unqualifiedColPatternRegex *regexp.Regexp
	selectAliasRegex           *regexp.Regexp
)

func init() {
	ctePatternRegex = regexp.MustCompile(`(?i)(\w+)\s+AS\s*\(`)
	tablePatternRegex = regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+([^\s;]+)`)
	tableAliasPatternRegex = regexp.MustCompile(`(?i)FROM\s+([^\s;]+)\s+(\w+)`)
	joinPatternRegex = regexp.MustCompile(`(?i)JOIN\s+([^\s;]+)\s+(\w+)`)
	columnRefPatternRegex = regexp.MustCompile(`(?i)(\w+)\.(\w+)`)
	aliasExtractPatternRegex = regexp.MustCompile(`(?i)(?:FROM|JOIN)\s+([^\s;]+)(?:\s+(?:AS\s+)?(\w+))?`)
	fromPatternRegex = regexp.MustCompile(`(?i)\bFROM\s+([^\s;]+)`)
	insertPatternRegex = regexp.MustCompile(`(?i)\b(?:INSERT\s+INTO|UPDATE)\s+([^\s;]+)`)
	joinCheckRegex = regexp.MustCompile(`(?i)\bJOIN\b`)
	whereClauseRegex = regexp.MustCompile(`(?i)\bWHERE\s+(.*?)(?:\s+(?:LIMIT|ORDER|GROUP|HAVING|;|$))`)
	setClauseRegex = regexp.MustCompile(`(?i)\bSET\s+(.*?)(?:\s+(?:WHERE|;|$))`)
	orderByClauseRegex = regexp.MustCompile(`(?i)\bORDER\s+BY\s+(.*?)(?:\s+(?:LIMIT|;|$))`)
	groupByClauseRegex = regexp.MustCompile(`(?i)\bGROUP\s+BY\s+(.*?)(?:\s+(?:HAVING|ORDER|LIMIT|;|$))`)
	havingClauseRegex = regexp.MustCompile(`(?i)\bHAVING\s+(.*?)(?:\s+(?:ORDER|LIMIT|;|$))`)
	paramCheckRegex = regexp.MustCompile(`^\d+$|^\$\d+$|\?`)
	unqualifiedColPatternRegex = regexp.MustCompile(`\b(\w+)\b`)
	selectAliasRegex = regexp.MustCompile(`(?i)\s+AS\s+(\w+)`)
}
