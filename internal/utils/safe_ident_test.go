package utils

import "testing"

// SafeGoIdent escapes Go keywords so generated identifiers compile.

func TestSafeGoIdent_EscapesAllKeywords(t *testing.T) {
	// The complete set of Go reserved keywords.
	keywords := []string{
		"break", "case", "chan", "const", "continue",
		"default", "defer", "else", "fallthrough", "for",
		"func", "go", "goto", "if", "import",
		"interface", "map", "package", "range", "return",
		"select", "struct", "switch", "type", "var",
	}
	for _, kw := range keywords {
		if got := SafeGoIdent(kw); got != kw+"_" {
			t.Errorf("SafeGoIdent(%q) = %q, want %q (keyword must be escaped)", kw, got, kw+"_")
		}
	}
}

func TestSafeGoIdent_LeavesValidIdentsUnchanged(t *testing.T) {
	// Ordinary identifiers, and capitalized look-alikes of keywords (which are
	// NOT keywords in Go and are valid identifiers), must be returned as-is.
	for _, id := range []string{"id", "userId", "email", "typeName", "Struct", "Type", "myFunc"} {
		if got := SafeGoIdent(id); got != id {
			t.Errorf("SafeGoIdent(%q) = %q, want unchanged", id, got)
		}
	}
}
