package validation

import (
	"strings"
	"testing"
)

// ValidateTableReferences reads the schema by reflection (same vSchema/vTable
// mirror used by column_refs_test.go, reused here) and flags FROM/JOIN tables
// that don't exist. These tests cover both the cases it handles correctly
// (guards) and a false-positive it produces on valid SQL (bug).

func TestValidateTableReferences_ValidSingleTable(t *testing.T) {
	if err := ValidateTableReferences("SELECT id, email FROM users", usersSchema(), "q"); err != nil {
		t.Errorf("known table must pass, got: %v", err)
	}
}

func TestValidateTableReferences_ValidWithAlias(t *testing.T) {
	if err := ValidateTableReferences("SELECT u.id FROM users u", usersSchema(), "q"); err != nil {
		t.Errorf("known table with alias must pass, got: %v", err)
	}
}

func TestValidateTableReferences_ValidJoin(t *testing.T) {
	sql := "SELECT u.id, p.title FROM users u JOIN posts p ON p.user_id = u.id"
	if err := ValidateTableReferences(sql, twoTableSchema(), "q"); err != nil {
		t.Errorf("known joined tables must pass, got: %v", err)
	}
}

func TestValidateTableReferences_UnknownFromErrors(t *testing.T) {
	err := ValidateTableReferences("SELECT id FROM nonexistent", usersSchema(), "q")
	if err == nil {
		t.Fatal("expected error for unknown FROM table, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should name the missing table, got: %v", err)
	}
}

func TestValidateTableReferences_UnknownJoinErrors(t *testing.T) {
	sql := "SELECT u.id FROM users u JOIN ghosts g ON g.user_id = u.id"
	err := ValidateTableReferences(sql, twoTableSchema(), "q")
	if err == nil {
		t.Fatal("expected error for unknown JOIN table, got nil")
	}
	if !strings.Contains(err.Error(), "ghosts") {
		t.Errorf("error should name the missing joined table, got: %v", err)
	}
}

// Keyspace-qualified reference to a known table must resolve after stripping
// the prefix ("myapp.users" -> "users").
func TestValidateTableReferences_KeyspaceQualifiedResolved(t *testing.T) {
	if err := ValidateTableReferences("SELECT id FROM myapp.users", usersSchema(), "q"); err != nil {
		t.Errorf("keyspace-qualified known table must resolve, got: %v", err)
	}
}

// A CTE name is not a real table and must not be flagged.
func TestValidateTableReferences_CTENotFlagged(t *testing.T) {
	sql := "WITH recent AS (SELECT id FROM users) SELECT * FROM recent"
	if err := ValidateTableReferences(sql, usersSchema(), "q"); err != nil {
		t.Errorf("CTE reference must not be flagged as a missing table, got: %v", err)
	}
}

func TestValidateTableReferences_NilSchemaSkips(t *testing.T) {
	if err := ValidateTableReferences("SELECT * FROM whatever", nil, "q"); err != nil {
		t.Errorf("nil schema must skip validation, got: %v", err)
	}
}

// Trailing semicolon must not be swept into the captured table name.
func TestValidateTableReferences_TrailingSemicolonKnown(t *testing.T) {
	if err := ValidateTableReferences("SELECT id FROM users;", usersSchema(), "q"); err != nil {
		t.Errorf("known table before ';' must pass, got: %v", err)
	}
}

// BUG: the FROM/JOIN capture is `[^\s;]+`, which includes a comma. In a
// comma-separated (old-style) join the first table is captured WITH its
// trailing comma ("users,"), which is not in the schema, so the validator
// reports a relation-does-not-exist error for a table that DOES exist —
// a false positive on valid SQL.
func TestValidateTableReferences_CommaJoinFalsePositive(t *testing.T) {
	err := ValidateTableReferences("SELECT u.id FROM users, posts", twoTableSchema(), "q")
	if err != nil {
		t.Errorf("comma-join over existing tables must not error, got false positive: %v", err)
	}
}
