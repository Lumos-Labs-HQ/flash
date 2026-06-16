package utils

import "strings"

func Capitalize(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func Uncapitalize(s string) string {
	if s == "" {
		return ""
	}
	s = Capitalize(s)
	return strings.ToLower(s[:1]) + s[1:]
}

// toTitleCase converts a word to title case (first letter uppercase, rest lowercase).
// This replaces the deprecated strings.Title function.
func toTitleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func ToPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.'
	})
	for i, word := range words {
		words[i] = toTitleCase(word)
	}
	return strings.Join(words, "")
}

func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// goKeywords is the set of Go reserved keywords that cannot be used as identifiers.
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// SafeGoIdent ensures a string is a valid Go identifier by appending "_"
// to reserved keywords.
func SafeGoIdent(s string) string {
	if goKeywords[s] {
		return s + "_"
	}
	return s
}
