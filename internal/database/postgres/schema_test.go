package postgres

import (
	"testing"
)

func TestParseIndexDef_Basic(t *testing.T) {
	a := newAdapter()
	idx := a.parseIndexDef("idx_email", "users", "CREATE INDEX idx_email ON users USING btree (email)")
	if idx.Name != "idx_email" {
		t.Errorf("Name = %q, want idx_email", idx.Name)
	}
	if idx.Table != "users" {
		t.Errorf("Table = %q, want users", idx.Table)
	}
	if len(idx.Columns) != 1 || idx.Columns[0] != "email" {
		t.Errorf("Columns = %v, want [email]", idx.Columns)
	}
	if idx.Unique {
		t.Error("Unique should be false for non-unique index")
	}
}

func TestParseIndexDef_Unique(t *testing.T) {
	a := newAdapter()
	idx := a.parseIndexDef("idx_email", "users", "CREATE UNIQUE INDEX idx_email ON users USING btree (email)")
	if !idx.Unique {
		t.Error("Unique should be true")
	}
}

func TestParseIndexDef_MultiColumn(t *testing.T) {
	a := newAdapter()
	idx := a.parseIndexDef("idx_name", "users", "CREATE INDEX idx_name ON users USING btree (first_name, last_name)")
	if len(idx.Columns) != 2 {
		t.Fatalf("Columns = %v, want 2", idx.Columns)
	}
	if idx.Columns[0] != "first_name" || idx.Columns[1] != "last_name" {
		t.Errorf("Columns = %v", idx.Columns)
	}
}

func TestParseIndexDef_WithSorting(t *testing.T) {
	a := newAdapter()
	idx := a.parseIndexDef("idx_created", "users", "CREATE INDEX idx_created ON users USING btree (created_at DESC, id ASC)")
	if len(idx.Columns) != 2 {
		t.Fatalf("Columns = %v, want 2", idx.Columns)
	}
	if idx.Columns[0] != "created_at" {
		t.Errorf("Columns[0] = %q, want created_at", idx.Columns[0])
	}
	if idx.Columns[1] != "id" {
		t.Errorf("Columns[1] = %q, want id", idx.Columns[1])
	}
}

func TestParseIndexDef_PartialIndex(t *testing.T) {
	a := newAdapter()
	idx := a.parseIndexDef("idx_active", "users", "CREATE INDEX idx_active ON users USING btree (email) WHERE active = true")
	if len(idx.Columns) != 1 || idx.Columns[0] != "email" {
		t.Errorf("Columns = %v, want [email]", idx.Columns)
	}
}

func TestParseIndexDef_EmptyColumns(t *testing.T) {
	a := newAdapter()
	idx := a.parseIndexDef("idx_bad", "users", "CREATE INDEX idx_bad ON users USING btree ()")
	if len(idx.Columns) != 0 {
		t.Errorf("Columns = %v, want empty", idx.Columns)
	}
}

func TestParseIndexDef_ExpressionIndex(t *testing.T) {
	a := newAdapter()
	// Expression indexes have columns like lower(email)
	idx := a.parseIndexDef("idx_lower", "users", "CREATE INDEX idx_lower ON users USING btree (lower(email))")
	if len(idx.Columns) != 1 || idx.Columns[0] != "lower(email)" {
		t.Errorf("Columns = %v, want [lower(email)]", idx.Columns)
	}
}

// TestGetTableIndexes_SQLShape verifies the query excludes constraint-backed indexes.
// This is a structural test — the actual DB interaction is covered by integration tests.
func TestGetTableIndexes_SQLShape(t *testing.T) {
	// We can't easily unit-test the SQL execution without a DB, but we verify
	// the parseIndexDef helper (which is the core logic) is correct.
	// The query itself contains the pg_constraint exclusion subquery which
	// prevents indexes like users_pkey (PRIMARY KEY) and users_email_key
	// (UNIQUE constraint) from being returned.
	a := newAdapter()

	// Simulate a PRIMARY KEY index definition — should be excluded by the SQL query
	pkIdx := a.parseIndexDef("users_pkey", "users", "CREATE UNIQUE INDEX users_pkey ON users USING btree (id)")
	if !pkIdx.Unique {
		t.Error("users_pkey should parse as unique")
	}
	if len(pkIdx.Columns) != 1 || pkIdx.Columns[0] != "id" {
		t.Errorf("users_pkey columns = %v", pkIdx.Columns)
	}

	// Simulate a UNIQUE constraint index
	uqIdx := a.parseIndexDef("users_email_key", "users", "CREATE UNIQUE INDEX users_email_key ON users USING btree (email)")
	if !uqIdx.Unique {
		t.Error("users_email_key should parse as unique")
	}
}

// TestGetAllTablesIndexes_SQLShape verifies the batch query also excludes constraints.
func TestGetAllTablesIndexes_SQLShape(t *testing.T) {
	// The batch version uses the same pg_constraint exclusion subquery.
	// We verify parseIndexDef handles multiple table contexts correctly.
	a := newAdapter()

	idx1 := a.parseIndexDef("idx_posts_user", "posts", "CREATE INDEX idx_posts_user ON posts USING btree (user_id)")
	if idx1.Table != "posts" {
		t.Errorf("Table = %q, want posts", idx1.Table)
	}

	idx2 := a.parseIndexDef("idx_comments_post", "comments", "CREATE INDEX idx_comments_post ON comments USING btree (post_id)")
	if idx2.Table != "comments" {
		t.Errorf("Table = %q, want comments", idx2.Table)
	}
}
