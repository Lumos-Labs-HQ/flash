package gencommon

import (
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

func TestModelTypeForQuery(t *testing.T) {
	expander := NewSchemaExpander(&parser.Schema{
		Tables: []*parser.Table{
			{
				Name: "preview_deployments",
				Columns: []*parser.Column{
					{Name: "id", Type: "BIGINT", Nullable: false},
					{Name: "source_branch", Type: "TEXT", Nullable: false},
					{Name: "target_branch", Type: "TEXT", Nullable: false},
					{Name: "source_owner", Type: "TEXT", Nullable: true},
					{Name: "source_repository", Type: "TEXT", Nullable: true},
				},
			},
		},
	})

	cols := func(names ...string) []*parser.QueryColumn {
		types := map[string]struct {
			typ      string
			nullable bool
		}{
			"id":                {"BIGINT", false},
			"source_branch":     {"TEXT", false},
			"target_branch":     {"TEXT", false},
			"source_owner":      {"TEXT", true},
			"source_repository": {"TEXT", true},
		}
		out := make([]*parser.QueryColumn, 0, len(names))
		for _, n := range names {
			m := types[n]
			out = append(out, &parser.QueryColumn{Name: n, Type: m.typ, Nullable: m.nullable})
		}
		return out
	}

	cases := []struct {
		name    string
		sql     string
		columns []*parser.QueryColumn
		want    string
	}{
		{
			name:    "same order matches",
			sql:     "SELECT * FROM preview_deployments WHERE id = ?",
			columns: cols("id", "source_branch", "target_branch", "source_owner", "source_repository"),
			want:    "PreviewDeployments",
		},
		{
			name:    "reordered columns still match",
			sql:     "SELECT id AS id, source_branch, source_owner, source_repository, target_branch FROM preview_deployments WHERE id = ?",
			columns: cols("id", "source_branch", "source_owner", "source_repository", "target_branch"),
			want:    "PreviewDeployments",
		},
		{
			name:    "missing column does not match",
			sql:     "SELECT id, source_branch FROM preview_deployments WHERE id = ?",
			columns: cols("id", "source_branch"),
			want:    "",
		},
		{
			name: "nullability mismatch does not match",
			sql:  "SELECT id, source_branch, target_branch, source_owner, source_repository FROM preview_deployments WHERE id = ?",
			columns: []*parser.QueryColumn{
				{Name: "id", Type: "BIGINT", Nullable: false},
				{Name: "source_branch", Type: "TEXT", Nullable: false},
				{Name: "target_branch", Type: "TEXT", Nullable: false},
				{Name: "source_owner", Type: "TEXT", Nullable: false},
				{Name: "source_repository", Type: "TEXT", Nullable: true},
			},
			want: "",
		},
		{
			name: "type mismatch does not match",
			sql:  "SELECT id, source_branch, target_branch, source_owner, source_repository FROM preview_deployments WHERE id = ?",
			columns: []*parser.QueryColumn{
				{Name: "id", Type: "TEXT", Nullable: false},
				{Name: "source_branch", Type: "TEXT", Nullable: false},
				{Name: "target_branch", Type: "TEXT", Nullable: false},
				{Name: "source_owner", Type: "TEXT", Nullable: true},
				{Name: "source_repository", Type: "TEXT", Nullable: true},
			},
			want: "",
		},
		{
			name:    "extra unknown column does not match",
			sql:     "SELECT id, source_branch, target_branch, source_owner, source_repository, extra FROM preview_deployments WHERE id = ?",
			columns: append(cols("id", "source_branch", "target_branch", "source_owner", "source_repository"), &parser.QueryColumn{Name: "extra", Type: "TEXT"}),
			want:    "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := &parser.Query{SQL: tc.sql}
			if got := expander.ModelTypeForQueryByName(q, tc.columns); got != tc.want {
				t.Fatalf("ModelTypeForQueryByName(%q) = %q, want %q", tc.sql, got, tc.want)
			}
		})
	}
}

func TestModelTypeForQueryPositional(t *testing.T) {
	expander := NewSchemaExpander(&parser.Schema{
		Tables: []*parser.Table{
			{
				Name: "tags",
				Columns: []*parser.Column{
					{Name: "id", Type: "BIGINT", Nullable: false},
					{Name: "name", Type: "TEXT", Nullable: false},
				},
			},
		},
	})

	// Same order matches
	q := &parser.Query{SQL: "SELECT id, name FROM tags"}
	cols := []*parser.QueryColumn{
		{Name: "id", Type: "BIGINT", Nullable: false},
		{Name: "name", Type: "TEXT", Nullable: false},
	}
	if got := expander.ModelTypeForQuery(q, cols); got != "Tags" {
		t.Errorf("same order: got %q, want %q", got, "Tags")
	}
	// Different order does NOT match (positional matcher for ORMs that scan by position)
	colsReordered := []*parser.QueryColumn{
		{Name: "name", Type: "TEXT", Nullable: false},
		{Name: "id", Type: "BIGINT", Nullable: false},
	}
	if got := expander.ModelTypeForQuery(q, colsReordered); got != "" {
		t.Errorf("reordered: got %q, want empty", got)
	}
}
