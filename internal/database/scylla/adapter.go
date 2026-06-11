package scylla

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apache/cassandra-gocql-driver/v2"
	"github.com/Lumos-Labs-HQ/flash/internal/database/common"
)

type Adapter struct {
	session  *gocql.Session
	cluster  *gocql.ClusterConfig
	keyspace string
}

func New() *Adapter {
	return &Adapter{}
}

var typeMap = map[string]string{
	"ascii": "ASCII", "text": "TEXT", "varchar": "VARCHAR",
	"int": "INT", "bigint": "BIGINT", "smallint": "SMALLINT",
	"tinyint": "TINYINT", "varint": "VARINT", "float": "FLOAT",
	"double": "DOUBLE", "boolean": "BOOLEAN", "bool": "BOOLEAN",
	"decimal": "DECIMAL", "counter": "COUNTER", "timestamp": "TIMESTAMP",
	"uuid": "UUID", "timeuuid": "TIMEUUID", "inet": "INET",
	"blob": "BLOB", "date": "DATE", "time": "TIME",
	"duration": "DURATION", "map": "MAP", "list": "LIST",
	"set": "SET", "tuple": "TUPLE", "frozen": "FROZEN",
}

func (a *Adapter) Connect(ctx context.Context, url string) error {
	cleanURL := strings.TrimPrefix(url, "scylla://")

	keyspace := ""
	hosts := cleanURL
	if idx := strings.Index(cleanURL, "/"); idx >= 0 {
		path := cleanURL[idx+1:]
		hosts = cleanURL[:idx]
		if qIdx := strings.Index(path, "?"); qIdx >= 0 {
			keyspace = path[:qIdx]
		} else {
			keyspace = path
		}
	}

	hostList := strings.Split(hosts, ",")
	for i := range hostList {
		hostList[i] = strings.TrimSpace(hostList[i])
	}

	cluster := gocql.NewCluster(hostList...)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 30 * time.Second
	cluster.ConnectTimeout = 10 * time.Second
	cluster.DisableInitialHostLookup = true
	cluster.IgnorePeerAddr = true

	session, err := cluster.CreateSession()
	if err != nil {
		return fmt.Errorf("failed to connect to ScyllaDB: %w", err)
	}

	a.session = session
	a.cluster = cluster
	a.keyspace = keyspace
	return nil
}

func (a *Adapter) Close() error {
	if a.session != nil {
		a.session.Close()
	}
	return nil
}

func (a *Adapter) Ping(ctx context.Context) error {
	if a.session == nil {
		return fmt.Errorf("not connected")
	}
	return a.session.Query("SELECT release_version FROM system.local").WithContext(ctx).Exec()
}

func (a *Adapter) CreateMigrationsTable(ctx context.Context) error {
	return a.session.Query(`CREATE TABLE IF NOT EXISTS _flash_migrations (
			id                  text,
			migration_name      text,
			checksum            text,
			started_at          timestamp,
			finished_at         timestamp,
			applied_steps_count int,
			PRIMARY KEY (id)
	)`).WithContext(ctx).Exec()
}

func (a *Adapter) EnsureMigrationTableCompatibility(_ context.Context) error { return nil }

func (a *Adapter) CleanupBrokenMigrationRecords(ctx context.Context) error {
	return a.session.Query(
		`DELETE FROM _flash_migrations WHERE finished_at IS NULL AND started_at < ?`,
		time.Now().Add(-1*time.Hour),
	).WithContext(ctx).Exec()
}

func (a *Adapter) GetAppliedMigrations(ctx context.Context) (map[string]*time.Time, error) {
	iter := a.session.Query(
		`SELECT id, finished_at FROM _flash_migrations WHERE finished_at IS NOT NULL`,
	).WithContext(ctx).Iter()
	defer iter.Close()

	applied := make(map[string]*time.Time)
	var id string
	var finishedAt time.Time
	for iter.Scan(&id, &finishedAt) {
		t := finishedAt
		applied[id] = &t
	}
	if err := iter.Close(); err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			return make(map[string]*time.Time), nil
		}
		return nil, err
	}
	return applied, nil
}

func (a *Adapter) RecordMigration(ctx context.Context, migrationID, name, checksum string) error {
	now := time.Now()
	return a.session.Query(
		`INSERT INTO _flash_migrations (id, migration_name, checksum, started_at, finished_at, applied_steps_count) VALUES (?, ?, ?, ?, ?, 1)`,
		migrationID, name, checksum, now, now,
	).WithContext(ctx).Exec()
}

func (a *Adapter) RemoveMigrationRecord(ctx context.Context, migrationID string) error {
	return a.session.Query(
		`DELETE FROM _flash_migrations WHERE id = ?`, migrationID,
	).WithContext(ctx).Exec()
}

func (a *Adapter) ExecuteMigration(ctx context.Context, migrationSQL string) error {
	for _, stmt := range common.ParseSQLStatements(migrationSQL) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := a.session.Query(stmt).WithContext(ctx).Exec(); err != nil {
			return fmt.Errorf("scylla: failed to execute %q: %w", stmt, err)
		}
	}
	return nil
}

func (a *Adapter) ExecuteAndRecordMigration(ctx context.Context, migrationID, name, checksum, migrationSQL string) error {
	if migrationSQL != "" {
		if err := a.ExecuteMigration(ctx, migrationSQL); err != nil {
			return err
		}
	}
	return a.RecordMigration(ctx, migrationID, name, checksum)
}

func (a *Adapter) ExecuteQuery(ctx context.Context, query string) (*common.QueryResult, error) {
	return a.ExecuteQueryWithArgs(ctx, query)
}

func (a *Adapter) ExecuteQueryWithArgs(ctx context.Context, query string, args ...interface{}) (*common.QueryResult, error) {
	iter := a.session.Query(query, args...).WithContext(ctx).Iter()
	defer iter.Close()

	cols := iter.Columns()
	if len(cols) == 0 {
		return &common.QueryResult{}, iter.Close()
	}

	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = c.Name
	}

	var results []map[string]interface{}
	for {
		row := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range row {
			ptrs[i] = &row[i]
		}
		if !iter.Scan(ptrs...) {
			break
		}
		m := make(map[string]interface{}, len(cols))
		for i, name := range colNames {
			m[name] = row[i]
		}
		results = append(results, m)
	}
	return &common.QueryResult{Columns: colNames, Rows: results}, iter.Close()
}

func (a *Adapter) ExecuteDMLWithArgs(ctx context.Context, query string, args ...interface{}) error {
	return a.session.Query(query, args...).WithContext(ctx).Exec()
}

func (a *Adapter) QuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (a *Adapter) ProviderName() string { return "scylla" }

func (a *Adapter) MapColumnType(dbType string) string {
	t := strings.TrimSpace(dbType)
	if idx := strings.Index(t, "<"); idx >= 0 {
		base := strings.ToUpper(t[:idx])
		if mapped, ok := typeMap[strings.ToLower(base)]; ok {
			return mapped + t[idx:]
		}
		return strings.ToUpper(base) + t[idx:]
	}
	if mapped, ok := typeMap[strings.ToLower(t)]; ok {
		return mapped
	}
	return strings.ToUpper(t)
}

func (a *Adapter) currentKeyspace() string {
	if a.keyspace != "" {
		return a.keyspace
	}
	return "system"
}

func sanitizeKeyspace(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}
