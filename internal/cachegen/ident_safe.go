package cachegen

import (
	"fmt"
	"strings"
)

// dedupeNames returns a copy of names where any repeat of an already-seen name
// gets a numeric suffix ("Foo", "Foo2", "Foo3", ...). Used so that two distinct
// tag strings that sanitize to the same enum member/constant identifier (e.g.
// "user-posts" and "user_posts" both -> "UserPosts") do not emit a duplicate
// member, which would fail to compile.
func dedupeNames(names []string) []string {
	used := make(map[string]int, len(names))
	out := make([]string, len(names))
	for i, n := range names {
		if c := used[n]; c > 0 {
			out[i] = fmt.Sprintf("%s%d", n, c+1)
		} else {
			out[i] = n
		}
		used[n]++
	}
	return out
}

// sanitizeUpperConst turns a tag into a valid UPPER_SNAKE_CASE identifier for
// languages whose enum members are conventionally upper-case (Python, Java,
// Kotlin). Runs of non-alphanumeric characters collapse to a single underscore,
// so "user-posts" -> "USER_POSTS" instead of the illegal "USER-POSTS".
func sanitizeUpperConst(tag string) string {
	var b strings.Builder
	prevUnderscore := false
	for _, r := range tag {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r - 32) // to upper
			prevUnderscore = false
		case (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevUnderscore = false
		default:
			if !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	s := strings.Trim(b.String(), "_")
	if s == "" {
		return "TAG"
	}
	if s[0] >= '0' && s[0] <= '9' {
		s = "TAG_" + s
	}
	return s
}

// Reserved-word sets for the non-Go languages. A key/param name that collides
// with one of these is escaped by safeParam so the generated accessor compiles.
// (Go has its own utils.SafeGoIdent.)
var (
	tsReserved = map[string]bool{
		"function": true, "class": true, "const": true, "let": true, "var": true,
		"return": true, "new": true, "typeof": true, "default": true, "enum": true,
		"interface": true, "type": true, "await": true, "async": true, "for": true,
		"if": true, "else": true, "switch": true, "case": true, "void": true,
		"in": true, "of": true, "do": true, "while": true, "delete": true,
	}
	pyReserved = map[string]bool{
		"class": true, "def": true, "return": true, "import": true, "from": true,
		"as": true, "for": true, "while": true, "if": true, "elif": true,
		"else": true, "and": true, "or": true, "not": true, "in": true,
		"is": true, "lambda": true, "none": true, "true": true, "false": true,
		"global": true, "pass": true, "yield": true, "with": true, "async": true,
		"await": true, "try": true, "except": true, "finally": true, "raise": true,
		"del": true, "nonlocal": true, "assert": true, "break": true, "continue": true,
	}
	rustReserved = map[string]bool{
		"type": true, "fn": true, "let": true, "mut": true, "match": true,
		"impl": true, "struct": true, "enum": true, "trait": true, "use": true,
		"mod": true, "pub": true, "return": true, "self": true, "super": true,
		"crate": true, "move": true, "ref": true, "where": true, "for": true,
		"loop": true, "while": true, "if": true, "else": true, "as": true,
		"const": true, "static": true, "dyn": true, "async": true, "await": true,
		"box": true, "in": true, "break": true, "continue": true, "unsafe": true,
	}
	javaReserved = map[string]bool{
		"class": true, "interface": true, "enum": true, "public": true, "private": true,
		"protected": true, "static": true, "final": true, "void": true, "int": true,
		"long": true, "short": true, "byte": true, "char": true, "boolean": true,
		"float": true, "double": true, "new": true, "return": true, "if": true,
		"else": true, "for": true, "while": true, "switch": true, "case": true,
		"default": true, "package": true, "import": true, "extends": true, "implements": true,
		"abstract": true, "this": true, "super": true, "try": true, "catch": true,
		"finally": true, "throw": true, "throws": true, "synchronized": true, "volatile": true,
	}
	kotlinReserved = map[string]bool{
		"class": true, "object": true, "interface": true, "fun": true, "val": true,
		"var": true, "return": true, "if": true, "else": true, "when": true,
		"for": true, "while": true, "is": true, "as": true, "in": true,
		"package": true, "import": true, "this": true, "super": true, "typeof": true,
		"null": true, "true": true, "false": true, "throw": true, "try": true,
		"catch": true, "finally": true, "do": true, "break": true, "continue": true,
	}
)

// safeParam escapes a parameter name that is a reserved word in the target
// language by wrapping it in underscores ("type" -> "_type_"). The wrap (rather
// than a bare prefix or suffix) keeps the escaped name from re-forming the
// keyword adjacent to surrounding syntax in any of the target languages, and
// "_x_" is a legal identifier in all of them. Non-reserved names are returned
// unchanged so ordinary params (org_id, user_id, ...) are untouched.
func safeParam(name string, reserved map[string]bool) string {
	if reserved[strings.ToLower(name)] {
		return "_" + name + "_"
	}
	return name
}
