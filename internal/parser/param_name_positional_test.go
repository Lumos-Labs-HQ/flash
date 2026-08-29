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

func TestInferParamNameInsertPositional(t *testing.T) {
	cases := []struct {
		name       string
		sql        string
		paramIndex int
		want       string
	}{
		{
			name:       "dollar params map to columns",
			sql:        "INSERT INTO users (email, name) VALUES ($1, $2)",
			paramIndex: 1,
			want:       "email",
		},
		{
			name:       "question marks map to columns",
			sql:        "INSERT INTO users (name, email) VALUES (?, ?)",
			paramIndex: 2,
			want:       "email",
		},
		{
			name:       "literal slot does not shift mapping",
			sql:        "INSERT INTO mounts (mount_type, service_type, host_path) VALUES (?, 'DATABASE', ?)",
			paramIndex: 2,
			want:       "host_path",
		},
		{
			name:       "trailing literal slots",
			sql:        "INSERT INTO service_compose (service_id, compose_type, compose_file, compose_path, suffix) VALUES (?, ?, ?, '', '')",
			paramIndex: 3,
			want:       "compose_file",
		},
		{
			name:       "INSERT OR IGNORE",
			sql:        "INSERT OR IGNORE INTO group_policy (group_id, policy_id) VALUES (?, ?)",
			paramIndex: 2,
			want:       "policy_id",
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

func TestInferParamNameUnionBranches(t *testing.T) {
	sql := "SELECT x FROM service_application sa JOIN services s ON s.id = sa.service_id WHERE sa.registry_id = ? UNION ALL SELECT x FROM t WHERE sa.rollback_registry_id = ? UNION ALL SELECT x FROM t WHERE sa.build_registry_id = ?"
	ti := &TypeInferrer{}
	for i, want := range []string{"registry_id", "rollback_registry_id", "build_registry_id"} {
		if got := ti.InferParamName(sql, i+1); got != want {
			t.Errorf("param %d = %q, want %q", i+1, got, want)
		}
	}
}
