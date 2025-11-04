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

func ToPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	for i, word := range words {
		words[i] = strings.Title(strings.ToLower(word))
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
