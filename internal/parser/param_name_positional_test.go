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

func TestInferParamNameInsertSelect(t *testing.T) {
	cases := []struct {
		name       string
		sql        string
		paramIndex int
		want       string
	}{
		{
			name: "select list slots map to insert columns",
			sql: `INSERT INTO services (service_type, name, app_name, app_status, env_var, created_at, updated_at)
SELECT base.service_type, ?, ?, 'IDLE', ?, CAST(? AS BIGINT), CAST(? AS BIGINT)
FROM services base
JOIN service_application sa ON sa.service_id = base.id
WHERE base.id = ?
RETURNING id`,
			paramIndex: 1,
			want:       "name",
		},
		{
			name: "second question slot maps to second column",
			sql: `INSERT INTO services (service_type, name, app_name, app_status, env_var, created_at, updated_at)
SELECT base.service_type, ?, ?, 'IDLE', ?, CAST(? AS BIGINT), CAST(? AS BIGINT)
FROM services base
JOIN service_application sa ON sa.service_id = base.id
WHERE base.id = ?
RETURNING id`,
			paramIndex: 2,
			want:       "app_name",
		},
		{
			name: "question after literal slots does not shift mapping",
			sql: `INSERT INTO services (service_type, name, app_name, app_status, env_var, created_at, updated_at)
SELECT base.service_type, ?, ?, 'IDLE', ?, CAST(? AS BIGINT), CAST(? AS BIGINT)
FROM services base
JOIN service_application sa ON sa.service_id = base.id
WHERE base.id = ?
RETURNING id`,
			paramIndex: 3,
			want:       "env_var",
		},
		{
			name: "CAST question inside select list maps to column",
			sql: `INSERT INTO services (service_type, name, app_name, app_status, env_var, created_at, updated_at)
SELECT base.service_type, ?, ?, 'IDLE', ?, CAST(? AS BIGINT), CAST(? AS BIGINT)
FROM services base
JOIN service_application sa ON sa.service_id = base.id
WHERE base.id = ?
RETURNING id`,
			paramIndex: 4,
			want:       "created_at",
		},
		{
			name: "trailing CAST question maps to last column",
			sql: `INSERT INTO services (service_type, name, app_name, app_status, env_var, created_at, updated_at)
SELECT base.service_type, ?, ?, 'IDLE', ?, CAST(? AS BIGINT), CAST(? AS BIGINT)
FROM services base
JOIN service_application sa ON sa.service_id = base.id
WHERE base.id = ?
RETURNING id`,
			paramIndex: 5,
			want:       "updated_at",
		},
		{
			name: "where clause param of insert select still maps to column",
			sql: `INSERT INTO services (service_type, name, app_name, app_status, env_var, created_at, updated_at)
SELECT base.service_type, ?, ?, 'IDLE', ?, CAST(? AS BIGINT), CAST(? AS BIGINT)
FROM services base
JOIN service_application sa ON sa.service_id = base.id
WHERE base.id = ?
RETURNING id`,
			paramIndex: 6,
			want:       "id",
		},
		{
			name: "clone row first column cast maps to service_id",
			sql: `INSERT INTO service_git (service_id, source_type, watch_paths, created_at, updated_at)
SELECT CAST(? AS BIGINT), source_type, watch_paths, CAST(? AS BIGINT), CAST(? AS BIGINT)
FROM service_git WHERE service_id = ?`,
			paramIndex: 1,
			want:       "service_id",
		},
		{
			name: "insert select where param maps to where column",
			sql: `INSERT INTO service_git (service_id, source_type, watch_paths, created_at, updated_at)
SELECT CAST(? AS BIGINT), source_type, watch_paths, CAST(? AS BIGINT), CAST(? AS BIGINT)
FROM service_git WHERE service_id = ?`,
			paramIndex: 4,
			want:       "service_id",
		},
		{
			name:       "dollar params in insert select map to columns",
			sql:        "INSERT INTO services (name, app_name) SELECT $1, $2 FROM services WHERE id = $3",
			paramIndex: 2,
			want:       "app_name",
		},
		{
			name:       "insert select without column list stays unnamed",
			sql:        "INSERT INTO services SELECT name, ? FROM services WHERE id = ?",
			paramIndex: 1,
			want:       "param1",
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

func TestInferParamNameJoinOnClause(t *testing.T) {
	sql := "SELECT n.id AS id, n.config_json AS config_json FROM notifications n JOIN organization o ON o.id = n.organization_id LEFT JOIN organization_members m ON m.organization_id = o.id AND m.user_id = ? WHERE n.notification_type = ? AND (o.owner_id = ? OR m.user_id IS NOT NULL) ORDER BY n.id LIMIT 1"
	ti := &TypeInferrer{}
	for i, want := range []string{"user_id", "notification_type", "owner_id"} {
		if got := ti.InferParamName(sql, i+1); got != want {
			t.Errorf("param %d = %q, want %q", i+1, got, want)
		}
	}
}

func TestInferParamNameFuncWrappedComparison(t *testing.T) {
	cases := []struct {
		name       string
		sql        string
		paramIndex int
		want       string
	}{
		{
			name:       "lower(col) = lower(?) first",
			sql:        "SELECT g.id FROM service_git g WHERE lower(g.repository) = lower(?) AND lower(g.owner) = lower(?) AND g.branch = ?",
			paramIndex: 1,
			want:       "repository",
		},
		{
			name:       "lower(col) = lower(?) second",
			sql:        "SELECT g.id FROM service_git g WHERE lower(g.repository) = lower(?) AND lower(g.owner) = lower(?) AND g.branch = ?",
			paramIndex: 2,
			want:       "owner",
		},
		{
			name:       "bare comparison after func-wrapped pair still maps",
			sql:        "SELECT g.id FROM service_git g WHERE lower(g.repository) = lower(?) AND lower(g.owner) = lower(?) AND g.branch = ?",
			paramIndex: 3,
			want:       "branch",
		},
		{
			name:       "func(col) = ? bare right side",
			sql:        "SELECT g.id FROM service_git g WHERE lower(g.owner) = ?",
			paramIndex: 1,
			want:       "owner",
		},
		{
			name:       "func(col) = func($N) dollar style",
			sql:        "SELECT g.id FROM service_git g WHERE lower(g.owner) = lower($1)",
			paramIndex: 1,
			want:       "owner",
		},
		{
			name:       "CAST(json_each.value AS TEXT) = ? names json_value",
			sql:        "SELECT COUNT(*) AS count FROM services CROSS JOIN json_each(services.network_ids) WHERE services.service_type = 'APPLICATION' AND CAST(json_each.value AS TEXT) = ?;",
			paramIndex: 1,
			want:       "json_value",
		},
		{
			name:       "CAST(table col AS TEXT) = ? names column",
			sql:        "SELECT COUNT(*) AS count FROM services WHERE CAST(services.name AS TEXT) = ?;",
			paramIndex: 1,
			want:       "name",
		},
		{
			name:       "CAST(col AS INTEGER) = $N dollar style",
			sql:        "SELECT COUNT(*) AS count FROM services WHERE CAST(services.replicas AS INTEGER) = $1;",
			paramIndex: 1,
			want:       "replicas",
		},
		{
			name:       "col IN (SELECT ... FROM json_each(?))",
			sql:        "SELECT id AS id FROM services WHERE service_type = 'APPLICATION' AND id IN (SELECT CAST(value AS INTEGER) FROM json_each(?)) ORDER BY id;",
			paramIndex: 1,
			want:       "id_list",
		},
		{
			name:       "col IN (SELECT ... FROM unnest($N)) picks nearest IN column",
			sql:        "SELECT id FROM t WHERE a IN (SELECT 1) AND id IN (SELECT CAST(x AS INTEGER) FROM json_each(?))",
			paramIndex: 1,
			want:       "id_list",
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

func TestInferParamNameLiteralComparison(t *testing.T) {
	sql := "UPDATE server_private_networks SET health_status = ?, health_error = NULLIF(?, ''), last_health_check_at = unixepoch(), consecutive_failures = CASE WHEN ? = 'HEALTHY' THEN 0 ELSE consecutive_failures + 1 END, updated_at = unixepoch() WHERE server_id = ?"
	ti := &TypeInferrer{}
	for i, want := range []string{"health_status", "health_error", "healthy", "server_id"} {
		if got := ti.InferParamName(sql, i+1); got != want {
			t.Errorf("param %d = %q, want %q", i+1, got, want)
		}
	}
}
