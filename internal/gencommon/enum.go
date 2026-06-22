package gencommon

import "strings"

// ExtractEnumValues extracts values from a MySQL inline ENUM column type like "enum('a','b')".
func ExtractEnumValues(columnType string) []string {
	lower := strings.ToLower(columnType)
	if !strings.HasPrefix(lower, "enum(") {
		return nil
	}

	values := lower[5 : len(lower)-1]

	var result []string
	parts := strings.Split(values, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, "'\"")
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
