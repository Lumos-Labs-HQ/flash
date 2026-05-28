package common

import (
	"fmt"
	"regexp"
	"strings"
)

// identifierSegment matches a single valid SQL identifier segment.
var identifierSegment = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ValidateIdentifier checks that a string is a safe single-segment SQL identifier.
// Rejects dotted names — use ValidateQualifiedIdentifier for those.
func ValidateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if len(name) > 128 {
		return fmt.Errorf("identifier too long (max 128 chars)")
	}
	if !identifierSegment.MatchString(name) {
		return fmt.Errorf("invalid identifier: %q", name)
	}
	return nil
}

// ValidateQualifiedIdentifier checks that a string is a safe qualified SQL identifier
// like "table" or "schema.table". Each segment must be a valid identifier.
func ValidateQualifiedIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	parts := strings.Split(name, ".")
	if len(parts) > 3 {
		return fmt.Errorf("identifier has too many segments: %q", name)
	}
	for _, part := range parts {
		if err := ValidateIdentifier(part); err != nil {
			return fmt.Errorf("invalid identifier segment %q in %q: %w", part, name, err)
		}
	}
	return nil
}
