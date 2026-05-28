package sql

import (
	"errors"
	"net/http"
	"testing"
)

// ── sanitizeError ─────────────────────────────────────────────────────────────

func TestSanitizeError(t *testing.T) {
	cases := []struct {
		err  string
		want string
	}{
		{"connection refused", "database connection error"},
		{"failed to connect to host", "database connection error"},
		{"context deadline exceeded: timeout", "request timed out"},
		{"permission denied for table users", "permission denied"},
		{"access denied for user 'root'@'localhost'", "permission denied"},
		{"pq: relation \"users\" does not exist", "internal error"},
		{"some unknown error", "internal error"},
	}
	for _, c := range cases {
		got := sanitizeError(errors.New(c.err))
		if got != c.want {
			t.Errorf("sanitizeError(%q) = %q, want %q", c.err, got, c.want)
		}
	}
}

// ── classifySQLError ──────────────────────────────────────────────────────────

func TestClassifySQLError(t *testing.T) {
	cases := []struct {
		err  string
		want int
	}{
		// syntax → 400
		{"syntax error at or near \"SELEC\"", http.StatusBadRequest},
		{"you have an error in your sql syntax near 'FORM'", http.StatusBadRequest},
		{"unrecognized token: \"@\"", http.StatusBadRequest},
		{"parse error: unexpected token", http.StatusBadRequest},
		// constraint → 400
		{"violates foreign key constraint", http.StatusBadRequest},
		{"unique constraint failed: users.email", http.StatusBadRequest},
		{"not null constraint failed: users.name", http.StatusBadRequest},
		// internal → 500
		{"connection reset by peer", http.StatusInternalServerError},
		{"pq: relation \"missing\" does not exist", http.StatusInternalServerError},
	}
	for _, c := range cases {
		got := classifySQLError(errors.New(c.err))
		if got != c.want {
			t.Errorf("classifySQLError(%q) = %d, want %d", c.err, got, c.want)
		}
	}
}
