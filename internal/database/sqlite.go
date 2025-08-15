package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/Rana718/Graft/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteAdapter struct {
	db *sql.DB
	qb squirrel.StatementBuilderType
}

// SQLite type mappings
var sqliteTypeMap = map[string]string{
	"varchar": "TEXT", "text": "TEXT", "char": "TEXT",
	"int": "INTEGER", "integer": "INTEGER", "bigint": "INTEGER", "smallint": "INTEGER", "tinyint": "INTEGER",
	"real": "REAL", "double": "REAL", "float": "REAL",
	"blob": "BLOB", "numeric": "NUMERIC", "decimal": "NUMERIC",
	"boolean": "INTEGER", "bool": "INTEGER",
	"date": "TEXT", "datetime": "TEXT", "timestamp": "TEXT",
}

func NewSQLiteAdapter() *SQLiteAdapter {
	return &SQLiteAdapter{
		qb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
	}
}

func (s *SQLiteAdapter) Connect(ctx context.Context, url string) error {
	dbPath := strings.TrimPrefix(url, "sqlite://")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open SQLite connection: %w", err)
	}
	s.db = db
	return nil
}

func (s *SQLiteAdapter) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteAdapter) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Migration table operations
func (s *SQLiteAdapter) CreateMigrationsTable(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS _graft_migrations (
		id TEXT PRIMARY KEY,
		checksum TEXT NOT NULL,
		finished_at TIMESTAMP,
		migration_name TEXT NOT NULL,
		logs TEXT,
		rolled_back_at TIMESTAMP,
		started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		applied_steps_count INTEGER NOT NULL DEFAULT 0
	)`
	_, err := s.db.ExecContext(ctx, query)
	return err
}

func (s *SQLiteAdapter) EnsureMigrationTableCompatibility(ctx context.Context) error {
	exists, err := s.columnExists("_graft_migrations", "logs")
	if err != nil {
		return err
	}
	if !exists {
		_, err = s.db.ExecContext(ctx, "ALTER TABLE _graft_migrations ADD COLUMN logs TEXT")
	}
	return err
}

func (s *SQLiteAdapter) CleanupBrokenMigrationRecords(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM _graft_migrations WHERE finished_at IS NULL AND started_at < datetime('now', '-1 hour')")
	return err
}

// Core migration operations
func (s *SQLiteAdapter) GetAppliedMigrations(ctx context.Context) (map[string]*time.Time, error) {
	applied := make(map[string]*time.Time)
	query := s.qb.Select("id", "finished_at").From("_graft_migrations").
		Where(squirrel.NotEq{"finished_at": nil}).OrderBy("started_at")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, sql, args...)
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

func (s *SQLiteAdapter) RecordMigration(ctx context.Context, migrationID, name, checksum string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO _graft_migrations (id, migration_name, checksum, started_at, finished_at, applied_steps_count)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
	`, migrationID, name, checksum)

	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteAdapter) ExecuteMigration(ctx context.Context, migrationSQL string) error {
	_, err := s.db.ExecContext(ctx, migrationSQL)
	return err
}

// Schema introspection
func (s *SQLiteAdapter) GetCurrentSchema(ctx context.Context) ([]types.SchemaTable, error) {
	tableNames, err := s.GetAllTableNames(ctx)
	if err != nil {
		return nil, err
	}

	var tables []types.SchemaTable
	for _, name := range tableNames {
		if name == "_graft_migrations" {
			continue
		}

		columns, err := s.GetTableColumns(ctx, name)
		if err != nil {
			return nil, err
		}

		indexes, err := s.GetTableIndexes(ctx, name)
		if err != nil {
			return nil, err
		}

		tables = append(tables, types.SchemaTable{
			Name:    name,
			Columns: columns,
			Indexes: indexes,
		})
	}
	return tables, nil
}

func (s *SQLiteAdapter) GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []types.SchemaColumn
	for rows.Next() {
		var cid int
		var column types.SchemaColumn
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int

		err := rows.Scan(&cid, &column.Name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}

		column.Type = s.MapColumnType(dataType)
		column.Nullable = notNull == 0
		column.IsPrimary = pk > 0
		if defaultValue.Valid {
			column.Default = defaultValue.String
		}

		column.IsUnique, _ = s.isColumnUnique(ctx, tableName, column.Name)
		columns = append(columns, column)
	}
	return columns, nil
}

func (s *SQLiteAdapter) GetTableIndexes(ctx context.Context, tableName string) ([]types.SchemaIndex, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []types.SchemaIndex
	for rows.Next() {
		var seq int
		var indexName string
		var unique int
		var origin, partial string

		err := rows.Scan(&seq, &indexName, &unique, &origin, &partial)
		if err != nil || origin == "pk" {
			continue
		}

		// Get columns for this index
		columns := s.getIndexColumns(ctx, indexName)
		if len(columns) > 0 {
			indexes = append(indexes, types.SchemaIndex{
				Name:    indexName,
				Table:   tableName,
				Columns: columns,
				Unique:  unique == 1,
			})
		}
	}
	return indexes, nil
}

func (s *SQLiteAdapter) GetAllTableNames(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err == nil {
			tables = append(tables, tableName)
		}
	}
	return tables, nil
}

// Helper methods
func (s *SQLiteAdapter) tableExists(tableName string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(
		"SELECT COUNT(*) > 0 FROM sqlite_master WHERE type = 'table' AND name = ?",
		tableName).Scan(&exists)
	return exists, err
}

func (s *SQLiteAdapter) columnExists(tableName, columnName string) (bool, error) {
	rows, err := s.db.QueryContext(context.Background(), fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err == nil && name == columnName {
			return true, nil
		}
	}
	return false, nil
}

func (s *SQLiteAdapter) isColumnUnique(ctx context.Context, tableName, columnName string) (bool, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var seq int
		var indexName string
		var unique int
		var origin, partial string

		err := rows.Scan(&seq, &indexName, &unique, &origin, &partial)
		if err != nil || unique == 0 {
			continue
		}

		columns := s.getIndexColumns(ctx, indexName)
		if len(columns) == 1 && columns[0] == columnName {
			return true, nil
		}
	}
	return false, nil
}

func (s *SQLiteAdapter) getIndexColumns(ctx context.Context, indexName string) []string {
	colRows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_info(%s)", indexName))
	if err != nil {
		return nil
	}
	defer colRows.Close()

	var columns []string
	for colRows.Next() {
		var seqno, cid int
		var name string
		if err := colRows.Scan(&seqno, &cid, &name); err == nil {
			columns = append(columns, name)
		}
	}
	return columns
}

func (s *SQLiteAdapter) CheckTableExists(ctx context.Context, tableName string) (bool, error) {
	return s.tableExists(tableName)
}

func (s *SQLiteAdapter) CheckColumnExists(ctx context.Context, tableName, columnName string) (bool, error) {
	return s.columnExists(tableName, columnName)
}

func (s *SQLiteAdapter) CheckNotNullConstraint(ctx context.Context, tableName, columnName string) (bool, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err == nil && name == columnName {
			return notNull == 1, nil
		}
	}
	return false, nil
}

func (s *SQLiteAdapter) CheckForeignKeyConstraint(ctx context.Context, tableName, constraintName string) (bool, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA foreign_key_list(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, seq int
		var table, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err == nil {
			if strings.Contains(constraintName, table) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *SQLiteAdapter) CheckUniqueConstraint(ctx context.Context, tableName, constraintName string) (bool, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var seq int
		var indexName string
		var unique int
		var origin, partial string
		if err := rows.Scan(&seq, &indexName, &unique, &origin, &partial); err == nil {
			if indexName == constraintName && unique == 1 {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *SQLiteAdapter) GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM `%s`", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		result = append(result, row)
	}
	return result, nil
}

func (s *SQLiteAdapter) DropTable(ctx context.Context, tableName string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName))
	return err
}

// SQL Generation
func (s *SQLiteAdapter) GenerateCreateTableSQL(table types.SchemaTable) string {
	var lines []string
	var foreignKeys []string

	for _, column := range table.Columns {
		if column.ForeignKeyTable != "" && column.ForeignKeyColumn != "" {
			fk := fmt.Sprintf("  FOREIGN KEY (`%s`) REFERENCES `%s`(`%s`)",
				column.Name, column.ForeignKeyTable, column.ForeignKeyColumn)
			if column.OnDeleteAction != "" {
				fk += fmt.Sprintf(" ON DELETE %s", column.OnDeleteAction)
			}
			foreignKeys = append(foreignKeys, fk)
		}
	}

	lines = append(lines, fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (", table.Name))

	for i, column := range table.Columns {
		comma := ","
		if i == len(table.Columns)-1 && len(foreignKeys) == 0 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf("  `%s` %s%s", column.Name, s.FormatColumnType(column), comma))
	}

	for i, fk := range foreignKeys {
		comma := ","
		if i == len(foreignKeys)-1 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf("%s%s", fk, comma))
	}

	lines = append(lines, ");")
	return strings.Join(lines, "\n")
}

func (s *SQLiteAdapter) GenerateAddColumnSQL(tableName string, column types.SchemaColumn) string {
	return fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s;",
		tableName, column.Name, s.FormatColumnType(column))
}

func (s *SQLiteAdapter) GenerateDropColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf("-- SQLite doesn't support DROP COLUMN. Manual steps required for %s.%s", tableName, columnName)
}

func (s *SQLiteAdapter) GenerateAddIndexSQL(index types.SchemaIndex) string {
	unique := ""
	if index.Unique {
		unique = "UNIQUE "
	}
	columns := "`" + strings.Join(index.Columns, "`, `") + "`"
	return fmt.Sprintf("CREATE %sINDEX `%s` ON `%s` (%s);", unique, index.Name, index.Table, columns)
}

func (s *SQLiteAdapter) GenerateDropIndexSQL(indexName string) string {
	return fmt.Sprintf("DROP INDEX IF EXISTS `%s`;", indexName)
}

// Type mapping and formatting
func (s *SQLiteAdapter) MapColumnType(dbType string) string {
	if mapped, exists := sqliteTypeMap[strings.ToLower(dbType)]; exists {
		return mapped
	}
	return strings.ToUpper(dbType)
}

func (s *SQLiteAdapter) FormatColumnType(column types.SchemaColumn) string {
	parts := []string{column.Type}

	if column.IsPrimary {
		if strings.ToUpper(column.Type) == "INTEGER" {
			parts = append(parts, "PRIMARY KEY AUTOINCREMENT")
		} else {
			parts = append(parts, "PRIMARY KEY")
		}
	}

	if column.IsUnique && !column.IsPrimary {
		parts = append(parts, "UNIQUE")
	}

	if !column.Nullable {
		parts = append(parts, "NOT NULL")
	}

	if column.Default != "" {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", column.Default))
	}

	return strings.Join(parts, " ")
}
