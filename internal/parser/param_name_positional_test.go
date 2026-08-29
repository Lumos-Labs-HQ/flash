package parser

import "testing"

func TestInferParamNamePositional(t *testing.T) {
	cases := []struct {
		name       string
		sql        string
		paramIndex int
		want       string
	}{
		{
			name:       "mixed direct and COALESCE in SET",
			sql:        "UPDATE deployments SET status = ?, error_message = COALESCE(?, error_message) WHERE id = ?",
			paramIndex: 2,
			want:       "error_message",
		},
		{
			name:       "WHERE id after COALESCE SET",
			sql:        "UPDATE deployments SET status = ?, error_message = COALESCE(?, error_message) WHERE id = ?",
			paramIndex: 3,
			want:       "id",
		},
		{
			name:       "all COALESCE SET plus WHERE id",
			sql:        "UPDATE services SET a = COALESCE(?, a), b = COALESCE(?, b) WHERE id = ?",
			paramIndex: 3,
			want:       "id",
		},
		{
			name:       "plain SET stays direct",
			sql:        "UPDATE deployments SET status = ? WHERE id = ?",
			paramIndex: 1,
			want:       "status",
		},
		{
			name:       "counter assignment gets delta suffix",
			sql:        "UPDATE deployments SET failures = failures + ? WHERE id = ?",
			paramIndex: 1,
			want:       "failures_delta",
		},
		{
			name:       "COALESCE filter in WHERE",
			sql:        "SELECT id FROM deployments WHERE status = COALESCE(?, status) AND service_id = COALESCE(?, service_id) LIMIT ?",
			paramIndex: 2,
			want:       "service_id",
		},
		{
			name:       "CASE WHEN comparison falls back to ordered param",
			sql:        "UPDATE deployments SET status = CASE WHEN ? = ? THEN 'A' ELSE 'B' END WHERE id = ?",
			paramIndex: 1,
			want:       "param1",
		},
		{
			name:       "CASE WHEN keeps WHERE id mapping",
			sql:        "UPDATE deployments SET status = CASE WHEN ? = ? THEN 'A' ELSE 'B' END WHERE id = ?",
			paramIndex: 3,
			want:       "id",
		},
		{
			name:       "NULL-or-not predicate stays ordered",
			sql:        "SELECT COUNT(*) FROM domains WHERE (? IS NULL OR id != ?)",
			paramIndex: 1,
			want:       "param1",
		},
		{
			name:       "NULL-or-not second param maps to id",
			sql:        "SELECT COUNT(*) FROM domains WHERE (? IS NULL OR id != ?)",
			paramIndex: 2,
			want:       "id",
		},
		{
			name:       "range comparison",
			sql:        "SELECT id FROM deployments WHERE id > ? AND id < ? LIMIT ?",
			paramIndex: 2,
			want:       "id",
		},
	}

	ti := &TypeInferrer{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ti.InferParamName(tc.sql, tc.paramIndex)
			if got != tc.want {
				t.Fatalf("InferParamName(%q, %d) = %q, want %q", tc.sql, tc.paramIndex, got, tc.want)
			}
		})
	}
}
