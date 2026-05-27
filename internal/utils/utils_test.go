package utils

import (
	"testing"
)

// ── ExtractTableName ─────────────────────────────────────────────────────────

func TestExtractTableName_Select(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT * FROM users WHERE id = 1", "users"},
		{"SELECT id FROM posts ORDER BY id", "posts"},
		{"select count(*) from orders", "orders"},
	}
	for _, c := range cases {
		got := ExtractTableName(c.sql)
		if got != c.want {
			t.Errorf("ExtractTableName(%q) = %q, want %q", c.sql, got, c.want)
		}
	}
}

func TestExtractTableName_Insert(t *testing.T) {
	got := ExtractTableName("INSERT INTO users (name, email) VALUES ($1, $2)")
	if got != "users" {
		t.Errorf("got %q, want users", got)
	}
}

func TestExtractTableName_Update(t *testing.T) {
	got := ExtractTableName("UPDATE products SET price = $1 WHERE id = $2")
	if got != "products" {
		t.Errorf("got %q, want products", got)
	}
}

func TestExtractTableName_Delete(t *testing.T) {
	got := ExtractTableName("DELETE FROM sessions WHERE expires_at < NOW()")
	if got != "sessions" {
		t.Errorf("got %q, want sessions", got)
	}
}

func TestExtractTableName_Empty(t *testing.T) {
	got := ExtractTableName("SELECT 1")
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

// ── IsModifyingQuery ─────────────────────────────────────────────────────────

func TestIsModifyingQuery(t *testing.T) {
	cases := []struct {
		sql      string
		expected bool
	}{
		{"SELECT * FROM users", false},
		{"INSERT INTO users (name) VALUES ($1)", true},
		{"UPDATE users SET name = $1 WHERE id = $2", true},
		{"DELETE FROM users WHERE id = $1", true},
		{"select count(*) from users", false},
	}
	for _, c := range cases {
		got := IsModifyingQuery(c.sql)
		if got != c.expected {
			t.Errorf("IsModifyingQuery(%q) = %v, want %v", c.sql, got, c.expected)
		}
	}
}

// ── SplitColumns ─────────────────────────────────────────────────────────────

func TestSplitColumns_Simple(t *testing.T) {
	parts := SplitColumns("id, name, email")
	if len(parts) != 3 {
		t.Errorf("parts = %d, want 3: %v", len(parts), parts)
	}
}

func TestSplitColumns_NestedParens(t *testing.T) {
	// COALESCE(a, b) must not be split at the inner comma
	parts := SplitColumns("id, COALESCE(first_name, last_name), email")
	if len(parts) != 3 {
		t.Errorf("parts = %d, want 3: %v", len(parts), parts)
	}
}

func TestSplitColumns_Single(t *testing.T) {
	parts := SplitColumns("id")
	if len(parts) != 1 {
		t.Errorf("parts = %d, want 1", len(parts))
	}
}

// ── ValidateSchemaSyntax ─────────────────────────────────────────────────────

func TestValidateSchemaSyntax_Valid(t *testing.T) {
	sql := `CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	email TEXT NOT NULL
);`
	if err := ValidateSchemaSyntax(sql, "users.sql"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSchemaSyntax_TrailingComma(t *testing.T) {
	sql := `CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	email TEXT NOT NULL,
);`
	if err := ValidateSchemaSyntax(sql, "users.sql"); err == nil {
		t.Error("expected trailing-comma error, got nil")
	}
}

func TestValidateSchemaSyntax_UnclosedTable(t *testing.T) {
	sql := `CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	email TEXT NOT NULL`
	if err := ValidateSchemaSyntax(sql, "users.sql"); err == nil {
		t.Error("expected unclosed-table error, got nil")
	}
}

// ── IsSQLKeyword ─────────────────────────────────────────────────────────────

func TestIsSQLKeyword(t *testing.T) {
	if !IsSQLKeyword("SELECT") {
		t.Error("SELECT should be a keyword")
	}
	if !IsSQLKeyword("select") {
		t.Error("select (lowercase) should be a keyword")
	}
	if IsSQLKeyword("users") {
		t.Error("users should not be a keyword")
	}
}
