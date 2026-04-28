package common

import (
	"testing"
)

func TestParseSQLStatements_Single(t *testing.T) {
	stmts := ParseSQLStatements("SELECT 1")
	if len(stmts) != 1 || stmts[0] != "SELECT 1" {
		t.Errorf("got %v", stmts)
	}
}

func TestParseSQLStatements_Multiple(t *testing.T) {
	sql := `CREATE TABLE a (id INT);
CREATE TABLE b (id INT);
CREATE TABLE c (id INT);`
	stmts := ParseSQLStatements(sql)
	if len(stmts) != 3 {
		t.Errorf("stmts = %d, want 3: %v", len(stmts), stmts)
	}
}

func TestParseSQLStatements_StripsLineComments(t *testing.T) {
	sql := `-- this is a comment
CREATE TABLE users (id INT);`
	stmts := ParseSQLStatements(sql)
	if len(stmts) != 1 {
		t.Errorf("stmts = %d, want 1: %v", len(stmts), stmts)
	}
}

func TestParseSQLStatements_SemicolonInsideString(t *testing.T) {
	// The semicolon inside the string literal must NOT split the statement.
	sql := `INSERT INTO t (msg) VALUES ('hello; world')`
	stmts := ParseSQLStatements(sql)
	if len(stmts) != 1 {
		t.Errorf("stmts = %d, want 1 (semicolon inside string): %v", len(stmts), stmts)
	}
}

func TestParseSQLStatements_Empty(t *testing.T) {
	stmts := ParseSQLStatements("")
	if len(stmts) != 0 {
		t.Errorf("stmts = %d, want 0", len(stmts))
	}
}

func TestParseSQLStatements_OnlyComments(t *testing.T) {
	stmts := ParseSQLStatements("-- just a comment\n-- another comment")
	if len(stmts) != 0 {
		t.Errorf("stmts = %d, want 0: %v", len(stmts), stmts)
	}
}

func TestParseSQLStatements_TrailingSemicolon(t *testing.T) {
	stmts := ParseSQLStatements("SELECT 1;")
	if len(stmts) != 1 {
		t.Errorf("stmts = %d, want 1", len(stmts))
	}
}
