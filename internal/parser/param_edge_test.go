package parser

import (
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
)

// analyzeSmoke runs analyzeQuery over the standard smoke schema and returns
// the analyzed query — the entry point for param/column assertions.
func analyzeSmoke(t *testing.T, name, sql string) *Query {
	t.Helper()
	p := NewQueryParser(&config.Config{Database: config.Database{Provider: "postgresql"}})
	schema := &Schema{Tables: []*Table{
		{Name: "users", Columns: []*Column{
			{Name: "id", Type: "SERIAL"},
			{Name: "email", Type: "VARCHAR(255)"},
			{Name: "name", Type: "TEXT"},
			{Name: "age", Type: "INTEGER"},
			{Name: "is_active", Type: "BOOLEAN"},
			{Name: "rating", Type: "DOUBLE PRECISION"},
			{Name: "tags", Type: "TEXT[]"},
			{Name: "settings", Type: "JSONB"},
			{Name: "created_at", Type: "TIMESTAMP WITH TIME ZONE"},
			{Name: "updated_at", Type: "TIMESTAMP"},
		}},
		{Name: "posts", Columns: []*Column{
			{Name: "id", Type: "SERIAL"},
			{Name: "user_id", Type: "INTEGER"},
			{Name: "title", Type: "TEXT"},
			{Name: "body", Type: "TEXT"},
			{Name: "view_count", Type: "INTEGER"},
			{Name: "status", Type: "VARCHAR(20)"},
			{Name: "published_at", Type: "TIMESTAMP WITH TIME ZONE"},
		}},
	}}
	q := &Query{Name: name, SQL: sql}
	// Mirror Parse()'s wiring: analyzeQuery relies on the inferrer having the
	// schema for cross-table type lookups.
	p.typeInferrer = NewTypeInferrerWithSchema(schema)
	if err := p.analyzeQuery(q, schema); err != nil {
		t.Fatalf("analyzeQuery(%s): %v", name, err)
	}
	return q
}

func mustParam(t *testing.T, q *Query, idx int) *Param {
	t.Helper()
	if idx >= len(q.Params) {
		t.Fatalf("query %s: param index %d out of range (%d params: %s)", q.Name, idx, len(q.Params), paramSummary(q))
	}
	return q.Params[idx]
}

// wantParam asserts name and type in one call.
func wantParam(t *testing.T, q *Query, idx int, name, typ string) {
	t.Helper()
	p := mustParam(t, q, idx)
	if p.Name != name {
		t.Errorf("%s param[%d].Name = %q, want %q (all: %s)", q.Name, idx, p.Name, name, paramSummary(q))
	}
	if typ != "" && p.Type != typ {
		t.Errorf("%s param[%d].Type = %q, want %q", q.Name, idx, p.Type, typ)
	}
}

// ── Param name inference: the "params wrong name on regx" regression class ──

func TestParamNames_WHERE(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE email = $1 AND age = $2 AND is_active = $3")
	wantParam(t, q, 0, "email", "VARCHAR(255)")
	wantParam(t, q, 1, "age", "INTEGER")
	wantParam(t, q, 2, "is_active", "BOOLEAN")
}

func TestParamNames_OrConditions(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE email = $1 OR email = $2")
	wantParam(t, q, 0, "email", "VARCHAR(255)")
	// dedup: second param with the same inferred name → email2
	wantParam(t, q, 1, "email2", "VARCHAR(255)")
}

func TestParamNames_DedupTripleCollision(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE email = $1 OR email = $2 OR email = $3")
	wantParam(t, q, 0, "email", "")
	wantParam(t, q, 1, "email2", "")
	wantParam(t, q, 2, "email3", "")
}

func TestParamNames_QualifiedColumns(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT u.id FROM users u WHERE u.email = $1 AND u.age = $2")
	wantParam(t, q, 0, "email", "")
	wantParam(t, q, 1, "age", "")
}

func TestParamNames_UpdateSet(t *testing.T) {
	q := analyzeSmoke(t, "q", "UPDATE users SET name = $1, age = $2 WHERE id = $3")
	wantParam(t, q, 0, "name", "TEXT")
	wantParam(t, q, 1, "age", "INTEGER")
	wantParam(t, q, 2, "id", "SERIAL")
}

func TestParamNames_UpdateSetCOALESCE(t *testing.T) {
	// The "COALESCE issues" bug class: SET col = COALESCE($N, col)
	q := analyzeSmoke(t, "q", "UPDATE users SET name = COALESCE($1, name), age = COALESCE($2, age) WHERE id = $3")
	wantParam(t, q, 0, "name", "TEXT")
	wantParam(t, q, 1, "age", "INTEGER")
	wantParam(t, q, 2, "id", "SERIAL")
}

func TestParamNames_InsertValues(t *testing.T) {
	q := analyzeSmoke(t, "q", "INSERT INTO users (email, name, age) VALUES ($1, $2, $3)")
	wantParam(t, q, 0, "email", "VARCHAR(255)")
	wantParam(t, q, 1, "name", "TEXT")
	wantParam(t, q, 2, "age", "INTEGER")
}

func TestParamNames_InsertValuesWithLiteralSlots(t *testing.T) {
	// A literal slot between params must not shift names ("map INSERT VALUES
	// params to columns through literal slots" fix). $1 fills slot 1 (email),
	// $2 fills slot 3 (age), $3 fills slot 4 (is_active) — the literal
	// occupies slot 2 (name).
	q := analyzeSmoke(t, "q", "INSERT INTO users (email, name, age, is_active) VALUES ($1, 'literal', $2, $3)")
	wantParam(t, q, 0, "email", "VARCHAR(255)")
	wantParam(t, q, 1, "age", "INTEGER")
	wantParam(t, q, 2, "is_active", "BOOLEAN")
}

func TestParamNames_LikeAndILike(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE name ILIKE $1 AND email LIKE $2")
	wantParam(t, q, 0, "name", "TEXT")
	wantParam(t, q, 1, "email", "VARCHAR(255)")
}

func TestParamNames_LimitOffset(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM users LIMIT $1 OFFSET $2")
	wantParam(t, q, 0, "limit", "BIGINT")
	wantParam(t, q, 1, "offset", "BIGINT")
}

func TestParamNames_Between(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE age BETWEEN $1 AND $2")
	wantParam(t, q, 0, "age_start", "")
	wantParam(t, q, 1, "age_end", "")
}

func TestParamNames_RangeComparison(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE age >= $1 AND age <= $2")
	wantParam(t, q, 0, "age_start", "")
	wantParam(t, q, 1, "age_end", "")
}

func TestParamNames_INListRewrittenToANY(t *testing.T) {
	// IN ($1,$2,$3) → id = ANY($1), params collapse to one array param,
	// and subsequent params renumber.
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE id IN ($1, $2, $3) AND email = $4")
	wantParam(t, q, 0, "id", "SERIAL[]")
	wantParam(t, q, 1, "email", "VARCHAR(255)")
	if !strings.Contains(q.SQL, "ANY($1)") {
		t.Errorf("SQL should contain ANY($1) after rewrite, got %q", q.SQL)
	}
	if !strings.Contains(q.SQL, "email = $2") {
		t.Errorf("SQL should renumber $4 → $2, got %q", q.SQL)
	}
}

func TestParamNames_INListSingleParamNotRewritten(t *testing.T) {
	// Single-element IN lists are NOT rewritten to ANY. Current behavior names
	// the param "id1" via the IN-list naming pattern (col + position), and the
	// "id1" name matches no schema column so the type falls back to TEXT —
	// a name-matching weakness worth pinning.
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE id IN ($1)")
	if strings.Contains(q.SQL, "ANY(") {
		t.Errorf("single-element IN list must not be rewritten: %q", q.SQL)
	}
	wantParam(t, q, 0, "id1", "TEXT")
}

func TestParamNames_ANYArray(t *testing.T) {
	// tags is TEXT[] and `tags = ANY($1)` compares an array column against a
	// set — current behavior types the param as the column type plus [].
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE tags = ANY($1)")
	if len(q.Params) != 1 {
		t.Fatalf("params = %d, want 1", len(q.Params))
	}
	p := q.Params[0]
	if p.Name != "tags" {
		t.Errorf("Name = %q, want tags", p.Name)
	}
	if p.Type != "TEXT[][]" && p.Type != "TEXT[]" {
		t.Errorf("Type = %q, want TEXT[] family", p.Type)
	}
}

func TestParamNames_ANYReverseForm(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE $1 = ANY(tags)")
	wantParam(t, q, 0, "tags", "TEXT")
}

func TestParamNames_FallthroughParamN(t *testing.T) {
	// The generic `col = $N` pattern still matches the first param in an
	// arithmetic expression (id = $1); only truly unmatched params fall back
	// to paramN. The second has no pattern → param2.
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE id = $1 + $2 * 100")
	wantParam(t, q, 0, "id", "SERIAL")
	wantParam(t, q, 1, "param2", "TEXT")
}

func TestParamNames_RepeatedDollarParamDedup(t *testing.T) {
	// $2 appears twice → ONE param emitted, SQL unchanged semantics
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE id = $1 OR email = $1")
	if len(q.Params) != 1 {
		t.Fatalf("params = %d, want 1 (repeated $N deduped)", len(q.Params))
	}
	wantParam(t, q, 0, "id", "SERIAL")
}

func TestParamNames_NonSequentialNumbersRenumbered(t *testing.T) {
	// Params written as $3, $1 → ParamNum reflects actual numbers and SQL
	// is renumbered to sequential.
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE age = $3 AND email = $1")
	if len(q.Params) != 2 {
		t.Fatalf("params = %d, want 2", len(q.Params))
	}
	wantParam(t, q, 0, "age", "INTEGER")
	wantParam(t, q, 1, "email", "VARCHAR(255)")
	if !strings.Contains(q.SQL, "$1") && !strings.Contains(q.SQL, "$2") {
		t.Errorf("SQL should be renumbered to $1/$2, got %q", q.SQL)
	}
}

func TestParamNames_HavingCount(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT user_id FROM posts GROUP BY user_id HAVING COUNT(*) > $1")
	// The dedicated HAVING pattern names it count_threshold, typed BIGINT.
	wantParam(t, q, 0, "count_threshold", "BIGINT")
}

func TestParamNames_JsonbOperators(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE settings @> $1")
	wantParam(t, q, 0, "settings", "JSONB")
}

func TestParamNames_JsonbArrow(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE settings->>'theme' = $1")
	wantParam(t, q, 0, "settings", "JSONB")
}

func TestParamNames_UnionBranches(t *testing.T) {
	// "resolve UNION branch params against their own WHERE clauses" fix:
	// each branch's param must be attributed to its own column.
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE email = $1 UNION SELECT id FROM posts WHERE title = $2")
	wantParam(t, q, 0, "email", "VARCHAR(255)")
	wantParam(t, q, 1, "title", "TEXT")
}

func TestParamNames_JoinOnParams(t *testing.T) {
	// "attribute INSERT..SELECT, JOIN ON, and wrapped-comparison params" fix.
	q := analyzeSmoke(t, "q", "SELECT u.id FROM users u JOIN posts p ON p.user_id = u.id WHERE u.age = $1 AND p.view_count = $2")
	wantParam(t, q, 0, "age", "INTEGER")
	wantParam(t, q, 1, "view_count", "INTEGER")
}

func TestParamNames_WrappedComparison(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE (age = $1) AND (email = $2)")
	wantParam(t, q, 0, "age", "INTEGER")
	wantParam(t, q, 1, "email", "VARCHAR(255)")
}

func TestParamNames_InsertSelect(t *testing.T) {
	q := analyzeSmoke(t, "q", "INSERT INTO posts (user_id, title) SELECT user_id, title FROM posts WHERE view_count = $1")
	wantParam(t, q, 0, "view_count", "INTEGER")
}

func TestParamNames_CTEQuery(t *testing.T) {
	q := analyzeSmoke(t, "q", `WITH stats AS (SELECT user_id, COUNT(*) AS post_count FROM posts GROUP BY user_id)
SELECT user_id FROM stats WHERE post_count > $1 LIMIT $2`)
	wantParam(t, q, 0, "post_count", "BIGINT")
	wantParam(t, q, 1, "limit", "BIGINT")
}

func TestParamNames_DateSuffix(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM posts WHERE published_at = $1")
	wantParam(t, q, 0, "published_at", "TIMESTAMP WITH TIME ZONE")
}

// ── ?-style (MySQL/SQLite) params ────────────────────────────────────────────

func TestParamNames_QuestionMarkInsert(t *testing.T) {
	q := analyzeSmoke(t, "q", "INSERT INTO users (email, name) VALUES (?, ?)")
	wantParam(t, q, 0, "email", "VARCHAR(255)")
	wantParam(t, q, 1, "name", "TEXT")
}

func TestParamNames_QuestionMarkUpdate(t *testing.T) {
	q := analyzeSmoke(t, "q", "UPDATE users SET name = ?, age = ? WHERE id = ?")
	wantParam(t, q, 0, "name", "TEXT")
	wantParam(t, q, 1, "age", "INTEGER")
	wantParam(t, q, 2, "id", "SERIAL")
}

func TestParamNames_QuestionMarkWhere(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM users WHERE email = ? AND age = ?")
	wantParam(t, q, 0, "email", "")
	wantParam(t, q, 1, "age", "")
}

func TestParamNames_QuestionMarkLimit(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id FROM users LIMIT ?")
	wantParam(t, q, 0, "limit", "BIGINT")
}

// ── InferParamTypeByName: name-only type fallback (CTE/no-table path) ───────

func TestInferParamTypeByName_Matrix(t *testing.T) {
	ti := NewTypeInferrer()
	cases := []struct{ name, want string }{
		{"limit", "BIGINT"},
		{"offset", "BIGINT"},
		{"count", "BIGINT"},
		{"min_count", "BIGINT"},
		{"post_count", "BIGINT"},
		{"total_sum", "BIGINT"},
		{"cart_total", "BIGINT"},
		{"item_num", "BIGINT"},
		{"user_age", "INTEGER"},
		{"id", "INTEGER"},
		{"age", "INTEGER"},
		{"relevance_score", "DOUBLE PRECISION"},
		{"product_rating", "DOUBLE PRECISION"},
		{"session_avg", "DOUBLE PRECISION"},
		{"is_active", "BOOLEAN"},
		{"active", "BOOLEAN"},
		{"featured", "BOOLEAN"},
		{"pinned", "BOOLEAN"},
		{"user_id", "TEXT"}, // _id NOT forced INTEGER (could be UUID)
		{"unknown_thing", "TEXT"},
		{"", "TEXT"},
	}
	for _, tc := range cases {
		if got := ti.InferParamTypeByName(tc.name); got != tc.want {
			t.Errorf("InferParamTypeByName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// ── Param type via schema lookup vs name heuristics ordering ────────────────

func TestParamTypes_SchemaWinsOverNameHeuristics(t *testing.T) {
	// Column named like a heuristic (view_count) but typed INTEGER in the
	// schema: the schema lookup must win over the *_count→BIGINT heuristic.
	// This was the exact "type validation not matching" bug class.
	q := analyzeSmoke(t, "q", "SELECT id FROM posts WHERE view_count = $1")
	wantParam(t, q, 0, "view_count", "INTEGER")
}

// ── Column extraction assertions ─────────────────────────────────────────────

func TestColumns_BasicNamesAndTypes(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT id, email, name FROM users WHERE id = $1")
	if len(q.Columns) != 3 {
		t.Fatalf("columns = %d, want 3", len(q.Columns))
	}
	want := map[string]string{"id": "SERIAL", "email": "VARCHAR(255)", "name": "TEXT"}
	for _, col := range q.Columns {
		if w, ok := want[col.Name]; ok && col.Type != w {
			t.Errorf("column %s type = %q, want %q", col.Name, col.Type, w)
		}
	}
}

func TestColumns_ASAlias(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT email AS contact_email, age * 2 AS double_age FROM users WHERE id = $1")
	if len(q.Columns) != 2 {
		t.Fatalf("columns = %d, want 2", len(q.Columns))
	}
	if q.Columns[0].Name != "contact_email" {
		t.Errorf("col0 name = %q, want contact_email", q.Columns[0].Name)
	}
	if q.Columns[0].OriginalExpr != "email" {
		t.Errorf("col0 OriginalExpr = %q, want email", q.Columns[0].OriginalExpr)
	}
	if q.Columns[1].Name != "double_age" {
		t.Errorf("col1 name = %q, want double_age", q.Columns[1].Name)
	}
	if !q.Columns[1].IsComputed {
		t.Error("double_age should be IsComputed")
	}
	if q.Columns[0].IsComputed {
		t.Error("email AS contact_email should not be IsComputed")
	}
}

func TestColumns_QualifiedWildcard(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT u.* FROM users u WHERE u.id = $1")
	if len(q.Columns) != 1 {
		t.Fatalf("columns = %d, want 1 (wildcard preserved for expansion)", len(q.Columns))
	}
	if q.Columns[0].Name != "*" || q.Columns[0].Table != "u" {
		t.Errorf("wildcard column = {%s %s}, want {*, u}", q.Columns[0].Name, q.Columns[0].Table)
	}
}

func TestColumns_BareWildcard(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT * FROM users WHERE id = $1")
	if len(q.Columns) != 1 {
		t.Fatalf("columns = %d, want 1", len(q.Columns))
	}
	if q.Columns[0].Name != "*" || q.Columns[0].Table != "users" {
		t.Errorf("wildcard = {%s %s}, want {*, users}", q.Columns[0].Name, q.Columns[0].Table)
	}
}

func TestColumns_Distinct(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT DISTINCT email FROM users WHERE id = $1")
	if len(q.Columns) != 1 || q.Columns[0].Name != "email" {
		t.Fatalf("columns = %+v, want single email", q.Columns)
	}
}

func TestColumns_ExpressionTypes(t *testing.T) {
	q := analyzeSmoke(t, "q", `SELECT COUNT(*) AS total, MAX(age) AS oldest, AVG(rating) AS avg_rating,
		bool_or(is_active) AS any_active, string_agg(name, ', ') AS names, COALESCE(rating, 0) AS rating_or_zero
		FROM users GROUP BY is_active`)
	if len(q.Columns) != 6 {
		t.Fatalf("columns = %d, want 6: %+v", len(q.Columns), q.Columns)
	}
	byName := map[string]*QueryColumn{}
	for _, c := range q.Columns {
		byName[c.Name] = c
	}
	if byName["total"] == nil || byName["total"].Type != "BIGINT" {
		t.Errorf("COUNT(*) type = %+v, want BIGINT", byName["total"])
	}
	if byName["oldest"] == nil || byName["oldest"].Type != "INTEGER" {
		t.Errorf("MAX(age) type = %+v, want INTEGER", byName["oldest"])
	}
	if byName["avg_rating"] == nil || byName["avg_rating"].Type != "NUMERIC" {
		t.Errorf("AVG(rating) type = %+v, want NUMERIC", byName["avg_rating"])
	}
}

func TestColumns_JsonbArrowType(t *testing.T) {
	q := analyzeSmoke(t, "q", "SELECT settings->>'theme' AS theme FROM users WHERE id = $1")
	if len(q.Columns) != 1 {
		t.Fatalf("columns = %d, want 1", len(q.Columns))
	}
	if q.Columns[0].Name != "theme" {
		t.Errorf("name = %q, want theme", q.Columns[0].Name)
	}
	if !q.Columns[0].IsComputed {
		t.Error("settings->>'theme' should be computed")
	}
}

func TestColumns_WrappedSelect(t *testing.T) {
	q := analyzeSmoke(t, "q", "(SELECT id, email FROM users WHERE id = $1)")
	if len(q.Columns) != 2 {
		t.Fatalf("columns = %d, want 2", len(q.Columns))
	}
}

func TestColumns_WithCTEColumns(t *testing.T) {
	q := analyzeSmoke(t, "q", `WITH stats AS (SELECT user_id, COUNT(*) AS post_count FROM posts GROUP BY user_id)
SELECT user_id, post_count FROM stats WHERE post_count > $1`)
	if len(q.Columns) != 2 {
		t.Fatalf("columns = %d, want 2: %+v", len(q.Columns), q.Columns)
	}
	if q.Columns[1].Name != "post_count" {
		t.Errorf("col1 = %q, want post_count", q.Columns[1].Name)
	}
}

func TestColumns_ReturningClause(t *testing.T) {
	q := analyzeSmoke(t, "q", "UPDATE users SET name = $1 WHERE id = $2 RETURNING id, updated_at")
	names := []string{}
	for _, c := range q.Columns {
		names = append(names, c.Name)
	}
	if len(names) != 2 || names[0] != "id" || names[1] != "updated_at" {
		t.Errorf("RETURNING columns = %v, want [id updated_at]", names)
	}
}
