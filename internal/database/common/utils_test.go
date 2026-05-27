package common

import (
	"strings"
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

func TestParseSQLStatements_PostgresDollarQuote(t *testing.T) {
	sql := `DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'job_status') THEN
        CREATE TYPE "job_status" AS ENUM ('pending', 'queued');
    END IF;
END $$;
CREATE TABLE users (id INT);`

	stmts := ParseSQLStatements(sql)
	if len(stmts) != 2 {
		t.Fatalf("stmts = %d, want 2: %v", len(stmts), stmts)
	}
	if !strings.Contains(stmts[0], "DO $$") {
		t.Errorf("first stmt missing DO $$ block: %s", stmts[0])
	}
	if !strings.Contains(stmts[0], "END $$") {
		t.Errorf("first stmt missing END $$: %s", stmts[0])
	}
	if stmts[1] != "CREATE TABLE users (id INT)" {
		t.Errorf("second stmt = %q, want CREATE TABLE users (id INT)", stmts[1])
	}
}

func TestParseSQLStatements_PostgresTaggedDollarQuote(t *testing.T) {
	sql := `CREATE OR REPLACE FUNCTION foo() RETURNS TEXT AS $func$
BEGIN
    RETURN 'hello; world';
END;
$func$;
SELECT 1;`

	stmts := ParseSQLStatements(sql)
	if len(stmts) != 2 {
		t.Fatalf("stmts = %d, want 2: %v", len(stmts), stmts)
	}
	if !strings.Contains(stmts[0], "$func$") {
		t.Errorf("first stmt missing $func$ block: %s", stmts[0])
	}
	if stmts[1] != "SELECT 1" {
		t.Errorf("second stmt = %q, want SELECT 1", stmts[1])
	}
}

func TestParseSQLStatements_BlockComment(t *testing.T) {
	sql := `/* multi
line comment */
CREATE TABLE a (id INT);`
	stmts := ParseSQLStatements(sql)
	if len(stmts) != 1 {
		t.Errorf("stmts = %d, want 1: %v", len(stmts), stmts)
	}
}

func TestParseSQLStatements_BlockCommentWithSemicolon(t *testing.T) {
	sql := `/* comment with; semicolon */
CREATE TABLE a (id INT);`
	stmts := ParseSQLStatements(sql)
	if len(stmts) != 1 {
		t.Errorf("stmts = %d, want 1: %v", len(stmts), stmts)
	}
}
