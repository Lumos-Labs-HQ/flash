package sql

import (
	"context"
	"fmt"
	"sync"
)

type DBMetrics struct {
	Provider string `json:"provider"`

	ActiveConnections int `json:"active_connections"`
	IdleConnections   int `json:"idle_connections"`
	TotalConnections  int `json:"total_connections"`
	MaxConnections    int `json:"max_connections"`

	DatabaseSizeMB float64 `json:"database_size_mb"`

	RowsInserted int64 `json:"rows_inserted"`
	RowsUpdated  int64 `json:"rows_updated"`
	RowsDeleted  int64 `json:"rows_deleted"`
	RowsFetched  int64 `json:"rows_fetched"`

	CacheHitRate float64 `json:"cache_hit_rate"` // percentage 0-100

	Deadlocks int64 `json:"deadlocks"`

	ActiveQueries []ActiveQuery `json:"active_queries"`

	SlowQueries []SlowQuery `json:"slow_queries"`

	TableSizes []TableSize `json:"table_sizes"`
}

type ActiveQuery struct {
	PID      int    `json:"pid"`
	Duration string `json:"duration"`
	State    string `json:"state"`
	Query    string `json:"query"`
	User     string `json:"user"`
}

type SlowQuery struct {
	Query   string  `json:"query"`
	Calls   int64   `json:"calls"`
	TotalMS float64 `json:"total_ms"`
	MeanMS  float64 `json:"mean_ms"`
	Rows    int64   `json:"rows"`
}

type TableSize struct {
	Name     string  `json:"name"`
	SizeMB   float64 `json:"size_mb"`
	RowCount int64   `json:"row_count"`
}

func (s *Service) GetMetrics(ctx context.Context) (*DBMetrics, error) {
	provider := s.adapter.ProviderName()
	m := &DBMetrics{Provider: provider}

	switch provider {
	case "postgres", "postgresql":
		return s.getPostgresMetrics(ctx, m)
	case "mysql":
		return s.getMySQLMetrics(ctx, m)
	case "sqlite", "sqlite3":
		return s.getSQLiteMetrics(ctx, m)
	default:
		return m, nil
	}
}

func (s *Service) getPostgresMetrics(ctx context.Context, m *DBMetrics) (*DBMetrics, error) {
	var mu sync.Mutex
	var wg sync.WaitGroup

	run := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx, `
			SELECT
				COUNT(*) FILTER (WHERE state = 'active')::int AS active,
				COUNT(*) FILTER (WHERE state = 'idle')::int   AS idle,
				COUNT(*)::int                                  AS total,
				(SELECT setting::int FROM pg_settings WHERE name = 'max_connections') AS max
			FROM pg_stat_activity
			WHERE datname = current_database()`)
		if err == nil && len(r.Rows) > 0 {
			row := r.Rows[0]
			mu.Lock()
			m.ActiveConnections = toInt(row["active"])
			m.IdleConnections = toInt(row["idle"])
			m.TotalConnections = toInt(row["total"])
			m.MaxConnections = toInt(row["max"])
			mu.Unlock()
		}
	})

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx,
			`SELECT CAST(COALESCE(SUM(pg_total_relation_size(relid)),0) AS float8) / 1048576.0 AS size_mb
			 FROM pg_stat_user_tables`)
		if err == nil && len(r.Rows) > 0 {
			mu.Lock()
			m.DatabaseSizeMB = toFloat(r.Rows[0]["size_mb"])
			mu.Unlock()
		}
	})

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx, `
			SELECT
				CAST(COALESCE(SUM(n_tup_ins),0)      AS bigint) AS inserted,
				CAST(COALESCE(SUM(n_tup_upd),0)      AS bigint) AS updated,
				CAST(COALESCE(SUM(n_tup_del),0)      AS bigint) AS deleted,
				CAST(COALESCE(SUM(seq_tup_read),0)   AS bigint) AS fetched
			FROM pg_stat_user_tables`)
		if err == nil && len(r.Rows) > 0 {
			row := r.Rows[0]
			mu.Lock()
			m.RowsInserted = toInt64(row["inserted"])
			m.RowsUpdated = toInt64(row["updated"])
			m.RowsDeleted = toInt64(row["deleted"])
			m.RowsFetched = toInt64(row["fetched"])
			mu.Unlock()
		}
	})

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx,
			`SELECT CAST(COALESCE(SUM(deadlocks),0) AS bigint) AS deadlocks
			 FROM pg_stat_database WHERE datname = current_database()`)
		if err == nil && len(r.Rows) > 0 {
			mu.Lock()
			m.Deadlocks = toInt64(r.Rows[0]["deadlocks"])
			mu.Unlock()
		}
	})

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx, `
			SELECT CAST(
				CASE WHEN (SUM(heap_blks_hit)+SUM(heap_blks_read))=0 THEN 0
				     ELSE 100.0*SUM(heap_blks_hit)/NULLIF(SUM(heap_blks_hit)+SUM(heap_blks_read),0)
				END AS float8) AS hit_rate
			FROM pg_statio_user_tables`)
		if err == nil && len(r.Rows) > 0 {
			mu.Lock()
			m.CacheHitRate = toFloat(r.Rows[0]["hit_rate"])
			mu.Unlock()
		}
	})

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx, `
			SELECT pid,
				   COALESCE(EXTRACT(EPOCH FROM (now() - query_start))::int::text || 's', '0s') AS duration,
				   state,
				   LEFT(query, 120) AS query,
				   usename AS "user"
			FROM pg_stat_activity
			WHERE datname = current_database()
			  AND state != 'idle'
			  AND pid <> pg_backend_pid()
			ORDER BY query_start
			LIMIT 20`)
		if err == nil {
			var aq []ActiveQuery
			for _, row := range r.Rows {
				aq = append(aq, ActiveQuery{
					PID:      toInt(row["pid"]),
					Duration: toString(row["duration"]),
					State:    toString(row["state"]),
					Query:    toString(row["query"]),
					User:     toString(row["user"]),
				})
			}
			mu.Lock()
			m.ActiveQueries = aq
			mu.Unlock()
		}
	})

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx, `
			SELECT LEFT(query, 120) AS query, calls,
				   ROUND(total_exec_time::numeric, 2) AS total_ms,
				   ROUND(mean_exec_time::numeric, 2)  AS mean_ms,
				   rows
			FROM pg_stat_statements
			ORDER BY mean_exec_time DESC
			LIMIT 10`)
		if err == nil {
			var sq []SlowQuery
			for _, row := range r.Rows {
				sq = append(sq, SlowQuery{
					Query:   toString(row["query"]),
					Calls:   toInt64(row["calls"]),
					TotalMS: toFloat(row["total_ms"]),
					MeanMS:  toFloat(row["mean_ms"]),
					Rows:    toInt64(row["rows"]),
				})
			}
			mu.Lock()
			m.SlowQueries = sq
			mu.Unlock()
		}
	})

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx, `
			SELECT relname AS name,
				   CAST(pg_total_relation_size(relid) AS float8) / 1048576.0 AS size_mb,
				   CAST(n_live_tup AS bigint) AS row_count
			FROM pg_stat_user_tables
			ORDER BY pg_total_relation_size(relid) DESC`)
		if err == nil {
			var ts []TableSize
			for _, row := range r.Rows {
				ts = append(ts, TableSize{
					Name:     toString(row["name"]),
					SizeMB:   toFloat(row["size_mb"]),
					RowCount: toInt64(row["row_count"]),
				})
			}
			mu.Lock()
			m.TableSizes = ts
			mu.Unlock()
		}
	})

	wg.Wait()
	return m, nil
}

func (s *Service) getMySQLMetrics(ctx context.Context, m *DBMetrics) (*DBMetrics, error) {
	var mu sync.Mutex
	var wg sync.WaitGroup

	run := func(fn func()) {
		wg.Add(1)
		go func() { defer wg.Done(); fn() }()
	}

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx, `
			SELECT
				SUM(CASE WHEN Command != 'Sleep' THEN 1 ELSE 0 END) AS active,
				SUM(CASE WHEN Command  = 'Sleep' THEN 1 ELSE 0 END) AS idle,
				COUNT(*) AS total
			FROM information_schema.PROCESSLIST`)
		if err == nil && len(r.Rows) > 0 {
			row := r.Rows[0]
			mu.Lock()
			m.ActiveConnections = toInt(row["active"])
			m.IdleConnections = toInt(row["idle"])
			m.TotalConnections = toInt(row["total"])
			mu.Unlock()
		}
	})

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx, `SELECT @@max_connections AS max`)
		if err == nil && len(r.Rows) > 0 {
			mu.Lock()
			m.MaxConnections = toInt(r.Rows[0]["max"])
			mu.Unlock()
		}
	})

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx, `
			SELECT COALESCE(ROUND(SUM(data_length + index_length) / 1048576, 4), 0) AS size_mb
			FROM information_schema.TABLES
			WHERE table_schema = DATABASE()`)
		if err == nil && len(r.Rows) > 0 {
			mu.Lock()
			m.DatabaseSizeMB = toFloat(r.Rows[0]["size_mb"])
			mu.Unlock()
		}
	})

	run(func() {
		vars, err := s.adapter.ExecuteQuery(ctx, `SHOW GLOBAL STATUS WHERE Variable_name IN
			('Com_insert','Com_update','Com_delete','Com_select',
			 'Innodb_buffer_pool_read_requests','Innodb_buffer_pool_reads')`)
		if err == nil {
			vmap := make(map[string]float64)
			for _, row := range vars.Rows {
				vmap[toString(row["Variable_name"])] = toFloat(row["Value"])
			}
			reads := vmap["Innodb_buffer_pool_read_requests"]
			disk := vmap["Innodb_buffer_pool_reads"]
			mu.Lock()
			m.RowsInserted = int64(vmap["Com_insert"])
			m.RowsUpdated = int64(vmap["Com_update"])
			m.RowsDeleted = int64(vmap["Com_delete"])
			m.RowsFetched = int64(vmap["Com_select"])
			if reads > 0 {
				m.CacheHitRate = (reads - disk) / reads * 100
			}
			mu.Unlock()
		}
	})

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx, `
			SELECT ID AS pid, TIME AS duration, STATE AS state,
				   LEFT(INFO, 120) AS query, USER AS user
			FROM information_schema.PROCESSLIST
			WHERE COMMAND != 'Sleep' AND ID != CONNECTION_ID()
			ORDER BY TIME DESC LIMIT 20`)
		if err == nil {
			var aq []ActiveQuery
			for _, row := range r.Rows {
				aq = append(aq, ActiveQuery{
					PID:      toInt(row["pid"]),
					Duration: fmt.Sprintf("%ds", toInt(row["duration"])),
					State:    toString(row["state"]),
					Query:    toString(row["query"]),
					User:     toString(row["user"]),
				})
			}
			mu.Lock()
			m.ActiveQueries = aq
			mu.Unlock()
		}
	})

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx, `
			SELECT LEFT(DIGEST_TEXT, 120) AS query,
				   COUNT_STAR AS calls,
				   ROUND(SUM_TIMER_WAIT/1000000000, 2) AS total_ms,
				   ROUND(AVG_TIMER_WAIT/1000000000, 2) AS mean_ms,
				   SUM_ROWS_EXAMINED AS rows
			FROM performance_schema.events_statements_summary_by_digest
			ORDER BY AVG_TIMER_WAIT DESC LIMIT 10`)
		if err == nil {
			var sq []SlowQuery
			for _, row := range r.Rows {
				sq = append(sq, SlowQuery{
					Query:   toString(row["query"]),
					Calls:   toInt64(row["calls"]),
					TotalMS: toFloat(row["total_ms"]),
					MeanMS:  toFloat(row["mean_ms"]),
					Rows:    toInt64(row["rows"]),
				})
			}
			mu.Lock()
			m.SlowQueries = sq
			mu.Unlock()
		}
	})

	run(func() {
		r, err := s.adapter.ExecuteQuery(ctx, `
			SELECT table_name AS name,
				   COALESCE(ROUND((data_length + index_length) / 1048576, 4), 0) AS size_mb,
				   COALESCE(table_rows, 0) AS row_count
			FROM information_schema.TABLES
			WHERE table_schema = DATABASE()
			ORDER BY (data_length + index_length) DESC`)
		if err == nil {
			var ts []TableSize
			for _, row := range r.Rows {
				ts = append(ts, TableSize{
					Name:     toString(row["name"]),
					SizeMB:   toFloat(row["size_mb"]),
					RowCount: toInt64(row["row_count"]),
				})
			}
			mu.Lock()
			m.TableSizes = ts
			mu.Unlock()
		}
	})

	wg.Wait()
	return m, nil
}

func (s *Service) getSQLiteMetrics(ctx context.Context, m *DBMetrics) (*DBMetrics, error) {
	// SQLite: limited metrics — just table sizes and row counts
	tables, err := s.adapter.GetAllTableNames(ctx)
	if err != nil {
		return m, nil
	}

	var totalSize int64
	for _, table := range tables {
		if table == "_flash_migrations" {
			continue
		}
		r, err := s.adapter.ExecuteQueryWithArgs(ctx,
			fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %q", table))
		if err == nil && len(r.Rows) > 0 {
			cnt := toInt64(r.Rows[0]["cnt"])
			totalSize += cnt
			m.TableSizes = append(m.TableSizes, TableSize{
				Name:     table,
				RowCount: cnt,
			})
		}
	}

	// SQLite page count * page size = db size
	r, err := s.adapter.ExecuteQuery(ctx, "SELECT page_count * page_size / 1048576.0 AS size_mb FROM pragma_page_count(), pragma_page_size()")
	if err == nil && len(r.Rows) > 0 {
		m.DatabaseSizeMB = toFloat(r.Rows[0]["size_mb"])
	}

	m.MaxConnections = 1
	m.TotalConnections = 1
	m.ActiveConnections = 1

	return m, nil
}

// helpers
func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch x := v.(type) {
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case float64:
		return int(x)
	case float32:
		return int(x)
	case []byte:
		var n int
		_, _ = fmt.Sscanf(string(x), "%d", &n)
		return n
	case string:
		var n int
		_, _ = fmt.Sscanf(x, "%d", &n)
		return n
	}
	var n int
	_, _ = fmt.Sscanf(fmt.Sprintf("%v", v), "%d", &n)
	return n
}

func toInt64(v interface{}) int64 {
	return int64(toInt(v))
}

func toFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int64:
		return float64(x)
	case int32:
		return float64(x)
	case int:
		return float64(x)
	case []byte:
		var f float64
		_, _ = fmt.Sscanf(string(x), "%f", &f)
		return f
	case string:
		var f float64
		_, _ = fmt.Sscanf(x, "%f", &f)
		return f
	}
	var f float64
	_, _ = fmt.Sscanf(fmt.Sprintf("%v", v), "%f", &f)
	return f
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	}
	return fmt.Sprintf("%v", v)
}
