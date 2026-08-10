package parser

import (
	"reflect"
	"testing"
)

// ── stripIdentQuotes ──────────────────────────────────────────────────────────

func TestStripIdentQuotes(t *testing.T) {
	tests := []struct{ in, want string }{
		{`"users"`, "users"},
		{"`users`", "users"},
		{"users", "users"},
		{`  "users"  `, "users"}, // trimmed first
		{`"users`, `"users`},     // unbalanced -> untouched
		{`users"`, `users"`},     // unbalanced -> untouched
		{"", ""},
		{`""`, ""},
	}
	for _, tt := range tests {
		if got := stripIdentQuotes(tt.in); got != tt.want {
			t.Errorf("stripIdentQuotes(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// ── matchesTableName ──────────────────────────────────────────────────────────

func TestMatchesTableName(t *testing.T) {
	tests := []struct {
		schema, query string
		want          bool
	}{
		{"users", "users", true},
		{"Users", "users", true},          // case-insensitive
		{"users", "public.users", true},   // query keyspace-qualified
		{"public.users", "users", true},   // schema keyspace-qualified
		{"public.users", "public.users", true},
		{"users", "posts", false},
		{"users", "public.posts", false},
		// Both qualified with DIFFERENT keyspaces => different tables, must not match
		// (matching would validate ks.users's columns against a query on other.users).
		{"ks.users", "other.users", false},
	}
	for _, tt := range tests {
		if got := matchesTableName(tt.schema, tt.query); got != tt.want {
			t.Errorf("matchesTableName(%q, %q) = %v, want %v", tt.schema, tt.query, got, tt.want)
		}
	}
}

// ── extractBalancedParens ─────────────────────────────────────────────────────

func TestExtractBalancedParens(t *testing.T) {
	tests := []struct {
		in    string
		start int
		want  string
	}{
		{"foo(a, b)", 3, "a, b"},
		{"f(a(b)c)", 1, "a(b)c"},   // nested parens balanced
		{"()", 0, ""},              // empty
		{"no paren", 0, ""},        // not a paren at start
		{"foo(unclosed", 3, ""},    // unbalanced -> ""
		{"a(b)(c)", 1, "b"},        // stops at first matching close
	}
	for _, tt := range tests {
		if got := extractBalancedParens(tt.in, tt.start); got != tt.want {
			t.Errorf("extractBalancedParens(%q, %d) = %q, want %q", tt.in, tt.start, got, tt.want)
		}
	}
}

// ── renumberParams ────────────────────────────────────────────────────────────

func TestRenumberParams(t *testing.T) {
	if got := renumberParams("SELECT $3, $1, $5", []int{3, 1, 5}); got != "SELECT $1, $2, $3" {
		t.Errorf("renumberParams = %q, want %q", got, "SELECT $1, $2, $3")
	}
	// A number not present in the ordering is left as-is.
	if got := renumberParams("a=$1 b=$9", []int{1}); got != "a=$1 b=$9" {
		t.Errorf("renumberParams unknown = %q, want %q", got, "a=$1 b=$9")
	}
	// Repeated occurrences of the same param all remap.
	if got := renumberParams("$2 and $2 and $1", []int{2, 1}); got != "$1 and $1 and $2" {
		t.Errorf("renumberParams repeated = %q, want %q", got, "$1 and $1 and $2")
	}
}

// ── extractOrderedParamNums ───────────────────────────────────────────────────

func TestExtractOrderedParamNums(t *testing.T) {
	tests := []struct {
		sql  string
		want []int
	}{
		{"SELECT $1, $2, $3", []int{1, 2, 3}},
		{"SELECT $2, $1, $2, $3", []int{2, 1, 3}}, // dedup, first-seen order
		{"no params here", []int{}},
		{"$10 $2 $10", []int{10, 2}},
	}
	for _, tt := range tests {
		got := extractOrderedParamNums(tt.sql)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("extractOrderedParamNums(%q) = %v, want %v", tt.sql, got, tt.want)
		}
	}
}

// ── attachJsonTypesToQuery ────────────────────────────────────────────────────

func TestAttachJsonTypesToQuery(t *testing.T) {
	jt := &JsonType{Column: "preferences", Name: "Preferences"}
	q := &Query{
		JsonTypes: []*JsonType{jt},
		Columns: []*QueryColumn{
			{Name: "id"},
			{Name: "preferences"},
		},
		Params: []*Param{
			{Name: "preferences", Type: "JSONB"},
			{Name: "id", Type: "BIGINT"},
		},
	}
	attachJsonTypesToQuery(q)

	// Matching return column gets the JSON def attached.
	var prefsCol *QueryColumn
	for _, c := range q.Columns {
		if c.Name == "preferences" {
			prefsCol = c
		}
	}
	if prefsCol == nil || prefsCol.JsonDef != jt {
		t.Errorf("preferences column should have JsonDef attached, got %+v", prefsCol)
	}
	// Matching param gets its type marked for JSON serialization.
	if q.Params[0].Type != "@json:Preferences" {
		t.Errorf("preferences param type = %q, want @json:Preferences", q.Params[0].Type)
	}
	// Non-matching param is untouched.
	if q.Params[1].Type != "BIGINT" {
		t.Errorf("id param type = %q, want BIGINT (unchanged)", q.Params[1].Type)
	}
}

func TestAttachJsonTypesToQuery_NoJsonTypesIsNoop(t *testing.T) {
	q := &Query{
		Columns: []*QueryColumn{{Name: "preferences"}},
		Params:  []*Param{{Name: "preferences", Type: "JSONB"}},
	}
	attachJsonTypesToQuery(q) // must not panic or mutate
	if q.Params[0].Type != "JSONB" {
		t.Errorf("param type changed unexpectedly: %q", q.Params[0].Type)
	}
}
