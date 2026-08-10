package validation

import (
	"strings"
	"testing"
)

// The validation package intentionally does NOT import parser (import cycle),
// and ValidateColumnReferences / ValidateTableReferences read the schema via
// reflection. Any struct with these field names works as a schema stand-in.
type vCol struct {
	Name     string
	Type     string
	Nullable bool
}
type vTable struct {
	Name    string
	Columns []*vCol
}
type vSchema struct {
	Tables []*vTable
}

func usersSchema() *vSchema {
	return &vSchema{Tables: []*vTable{{
		Name: "users",
		Columns: []*vCol{
			{Name: "id"}, {Name: "name"}, {Name: "email"}, {Name: "created_at"},
		},
	}}}
}

func twoTableSchema() *vSchema {
	return &vSchema{Tables: []*vTable{
		{Name: "users", Columns: []*vCol{{Name: "id"}, {Name: "email"}}},
		{Name: "posts", Columns: []*vCol{{Name: "id"}, {Name: "user_id"}, {Name: "title"}}},
	}}
}

// ---------------------------------------------------------------------------
// ValidateColumnReferences — qualified (alias.column) refs
// ---------------------------------------------------------------------------

func TestValidateColumnReferences_QualifiedValid(t *testing.T) {
	if err := ValidateColumnReferences("SELECT u.id, u.email FROM users u", usersSchema(), "q"); err != nil {
		t.Errorf("valid qualified columns should pass, got: %v", err)
	}
}

func TestValidateColumnReferences_QualifiedInvalid(t *testing.T) {
	err := ValidateColumnReferences("SELECT u.nope FROM users u", usersSchema(), "q")
	if err == nil {
		t.Fatal("expected error for u.nope, got nil")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error should name the bad column, got: %v", err)
	}
}

func TestValidateColumnReferences_JoinAliasesResolved(t *testing.T) {
	sql := "SELECT u.email, p.title FROM users u JOIN posts p ON p.user_id = u.id"
	if err := ValidateColumnReferences(sql, twoTableSchema(), "q"); err != nil {
		t.Errorf("valid join columns should pass, got: %v", err)
	}
}

func TestValidateColumnReferences_JoinBadColumn(t *testing.T) {
	sql := "SELECT u.email, p.nonexistent FROM users u JOIN posts p ON p.user_id = u.id"
	if err := ValidateColumnReferences(sql, twoTableSchema(), "q"); err == nil {
		t.Error("expected error for p.nonexistent in a join, got nil")
	}
}

// ---------------------------------------------------------------------------
// ValidateColumnReferences — unqualified refs in clauses
// ---------------------------------------------------------------------------

// PASSES today: WHERE is followed by LIMIT so the clause regex matches.
func TestValidateColumnReferences_UnqualifiedInvalid_WithLimit(t *testing.T) {
	err := ValidateColumnReferences("SELECT id FROM users WHERE nonexistent = $1 LIMIT 10", usersSchema(), "q")
	if err == nil {
		t.Fatal("expected error for unqualified 'nonexistent', got nil")
	}
}

// Same query, but WHERE is the LAST clause (no LIMIT/;). The clause regex ends
// with `(?:\s+(?:LIMIT|ORDER|GROUP|HAVING|;|$))`, which requires whitespace
// before end-of-string, so a trimmed trailing WHERE is never matched and the
// bad column is silently accepted. Documents the validation gap.
func TestValidateColumnReferences_UnqualifiedInvalid_TrailingWhere(t *testing.T) {
	err := ValidateColumnReferences("SELECT id FROM users WHERE nonexistent = $1", usersSchema(), "q")
	if err == nil {
		t.Fatal("expected error for unqualified 'nonexistent' in a trailing WHERE, got nil (validation gap)")
	}
}

func TestValidateColumnReferences_UnqualifiedValid_TrailingWhere(t *testing.T) {
	if err := ValidateColumnReferences("SELECT id FROM users WHERE email = $1", usersSchema(), "q"); err != nil {
		t.Errorf("valid unqualified column should pass, got: %v", err)
	}
}

func TestValidateColumnReferences_UnqualifiedInvalid_TrailingSemicolon(t *testing.T) {
	err := ValidateColumnReferences("SELECT id FROM users WHERE nonexistent = $1;", usersSchema(), "q")
	if err == nil {
		t.Fatal("expected error for unqualified 'nonexistent' before ';', got nil")
	}
}

// ---------------------------------------------------------------------------
// ValidateColumnReferences — things that must NOT be flagged
// ---------------------------------------------------------------------------

func TestValidateColumnReferences_NilSchemaSkips(t *testing.T) {
	if err := ValidateColumnReferences("SELECT anything FROM nowhere WHERE x = 1", nil, "q"); err != nil {
		t.Errorf("nil schema must skip validation, got: %v", err)
	}
}

func TestValidateColumnReferences_UnionSkips(t *testing.T) {
	sql := "SELECT id FROM users UNION SELECT bogus FROM users"
	if err := ValidateColumnReferences(sql, usersSchema(), "q"); err != nil {
		t.Errorf("UNION queries are intentionally skipped, got: %v", err)
	}
}

func TestValidateColumnReferences_StringLiteralNotFlagged(t *testing.T) {
	sql := "SELECT id FROM users WHERE email = 'not_a_column' LIMIT 1"
	if err := ValidateColumnReferences(sql, usersSchema(), "q"); err != nil {
		t.Errorf("string literal must not be treated as a column, got: %v", err)
	}
}

func TestValidateColumnReferences_ParamsNotFlagged(t *testing.T) {
	sql := "SELECT id FROM users WHERE email = $1 LIMIT 1"
	if err := ValidateColumnReferences(sql, usersSchema(), "q"); err != nil {
		t.Errorf("bind param must not be treated as a column, got: %v", err)
	}
}

func TestValidateColumnReferences_UnknownTableSkipped(t *testing.T) {
	// Column qualified by a table not in the schema -> cannot validate, must skip.
	sql := "SELECT o.anything FROM orders o"
	if err := ValidateColumnReferences(sql, usersSchema(), "q"); err != nil {
		t.Errorf("unknown table reference should be skipped, got: %v", err)
	}
}
