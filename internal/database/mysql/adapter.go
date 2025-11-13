package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/database/common"
	"github.com/Masterminds/squirrel"
	_ "github.com/go-sql-driver/mysql"
)

type Adapter struct {
	db *sql.DB
	qb squirrel.StatementBuilderType
}

var typeMap = map[string]string{
	"varchar": "VARCHAR", "char": "CHAR",
	"text": "TEXT", "longtext": "TEXT", "mediumtext": "TEXT", "tinytext": "TEXT",
	"int": "INT", "integer": "INT", "bigint": "BIGINT", "smallint": "SMALLINT", "tinyint": "TINYINT",
	"boolean": "BOOLEAN", "bool": "BOOLEAN",
	"datetime": "DATETIME", "timestamp": "TIMESTAMP", "date": "DATE", "time": "TIME",
	"decimal": "DECIMAL", "numeric": "DECIMAL", "float": "FLOAT", "double": "DOUBLE",
	"json": "JSON", "blob": "BLOB", "binary": "BINARY", "varbinary": "VARBINARY",
}

func New() *Adapter {
	return &Adapter{
		qb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
	}
}

func (m *Adapter) Connect(ctx context.Context, url string) error {
	dsn := url
	if strings.HasPrefix(url, "mysql://") {
		dsn = strings.TrimPrefix(url, "mysql://")

		atIndex := strings.Index(dsn, "@")
		if atIndex > 0 {
			credentials := dsn[:atIndex]
			remainder := dsn[atIndex+1:]

			slashIndex := strings.Index(remainder, "/")
			if slashIndex > 0 {
				hostPort := remainder[:slashIndex]
				dbAndParams := remainder[slashIndex+1:]

				dbAndParams = strings.ReplaceAll(dbAndParams, "ssl-mode=REQUIRED", "tls=skip-verify")
				dbAndParams = strings.ReplaceAll(dbAndParams, "ssl-mode=DISABLED", "tls=false")
				dbAndParams = strings.ReplaceAll(dbAndParams, "ssl-mode=VERIFY_CA", "tls=true")
				dbAndParams = strings.ReplaceAll(dbAndParams, "ssl-mode=VERIFY_IDENTITY", "tls=true")
				dbAndParams = strings.ReplaceAll(dbAndParams, "sslmode=require", "tls=skip-verify")
				dbAndParams = strings.ReplaceAll(dbAndParams, "sslmode=disable", "tls=false")
				dbAndParams = strings.ReplaceAll(dbAndParams, "sslmode=verify-ca", "tls=true")
				dbAndParams = strings.ReplaceAll(dbAndParams, "sslmode=verify-full", "tls=true")

				dsn = fmt.Sprintf("%s@tcp(%s)/%s", credentials, hostPort, dbAndParams)
			}
		}
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open MySQL connection: %w", err)
	}
	db.SetMaxOpenConns(2)                   
	db.SetMaxIdleConns(0)                   
	db.SetConnMaxLifetime(15 * time.Minute) 
	db.SetConnMaxIdleTime(3 * time.Minute)  

	m.db = db
	return nil
}

func (m *Adapter) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *Adapter) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

func (m *Adapter) CreateMigrationsTable(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS _flash_migrations (
		id VARCHAR(255) PRIMARY KEY,
		checksum VARCHAR(64) NOT NULL,
		finished_at TIMESTAMP NULL,
		migration_name VARCHAR(255) NOT NULL,
		logs TEXT,
		rolled_back_at TIMESTAMP NULL,
		started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		applied_steps_count INTEGER NOT NULL DEFAULT 0
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	_, err := m.db.ExecContext(ctx, query)
	return err
}

func (m *Adapter) EnsureMigrationTableCompatibility(ctx context.Context) error {
	exists, err := m.columnExists("_flash_migrations", "logs")
	if err != nil {
		return err
	}
	if !exists {
		_, err = m.db.ExecContext(ctx, "ALTER TABLE _flash_migrations ADD COLUMN logs TEXT")
	}
	return err
}

func (m *Adapter) CleanupBrokenMigrationRecords(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, `
	DELETE FROM _flash_migrations 
		WHERE finished_at IS NULL AND started_at < DATE_SUB(NOW(), INTERVAL 1 HOUR)
	`)
	return err
}

func (m *Adapter) GetAppliedMigrations(ctx context.Context) (map[string]*time.Time, error) {
	applied := make(map[string]*time.Time)
	query := m.qb.Select("id", "finished_at").From("_flash_migrations").
		Where(squirrel.NotEq{"finished_at": nil}).OrderBy("started_at")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := m.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var finishedAt *time.Time
		if err := rows.Scan(&id, &finishedAt); err != nil {
			continue
		}
		applied[id] = finishedAt
	}
	return applied, nil
}

func (m *Adapter) RecordMigration(ctx context.Context, migrationID, name, checksum string) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
	INSERT INTO _flash_migrations (id, migration_name, checksum, started_at, finished_at, applied_steps_count)
		VALUES (?, ?, ?, NOW(), NOW(), 1)
	`, migrationID, name, checksum)

	if err != nil {
		return err
	}
	return tx.Commit()
}

func (m *Adapter) ExecuteMigration(ctx context.Context, migrationSQL string) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	statements := common.ParseSQLStatements(migrationSQL)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		_, err := tx.ExecContext(ctx, stmt)
		if err != nil {
			return fmt.Errorf("failed to execute statement '%s': %w", stmt, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	return nil
}

func (m *Adapter) ExecuteQuery(ctx context.Context, query string) (*common.QueryResult, error) {
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return &common.QueryResult{
		Columns: columns,
		Rows:    results,
	}, nil
}

func (m *Adapter) MapColumnType(dbType string) string {
	if mapped, exists := typeMap[strings.ToLower(dbType)]; exists {
		return mapped
	}
	return strings.ToUpper(dbType)
}
