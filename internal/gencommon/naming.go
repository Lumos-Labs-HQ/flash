package gencommon

import (
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/utils"
)

// QueryPascal returns the query name as PascalCase, preserving existing casing.
func QueryPascal(name string) string {
	if name == "" {
		return name
	}
	if name[0] >= 'A' && name[0] <= 'Z' {
		return name
	}
	return utils.ToPascalCase(name)
}

// ToCamelCase converts PascalCase or snake_case to camelCase.
func ToCamelCase(name string) string {
	if name == "" {
		return name
	}
	if name[0] >= 'A' && name[0] <= 'Z' {
		return strings.ToLower(name[:1]) + name[1:]
	}
	pascal := utils.ToPascalCase(name)
	if pascal == "" {
		return pascal
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}
