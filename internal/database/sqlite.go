package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Rana718/Graft/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteAdapter struct {
	db *sql.DB
}

func NewSQLiteAdapter() *SQLiteAdapter {
	return &SQLiteAdapter{}
}

func (s *SQLiteAdapter) Connect(ctx context.Context, url string) error {
	// Remove sqlite:// prefix if present
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

func (s *SQLiteAdapter) CreateMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS _graft_migrations (
			id TEXT PRIMARY KEY,
			checksum TEXT NOT NULL,
			finished_at TIMESTAMP,
			migration_name TEXT NOT NULL,
			logs TEXT,
			rolled_back_at TIMESTAMP,
			started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			applied_steps_count INTEGER NOT NULL DEFAULT 0
		);`

	_, err := s.db.ExecContext(ctx, query)
	return err
}

func (s *SQLiteAdapter) EnsureMigrationTableCompatibility(ctx context.Context) error {
	// Check if logs column exists, add if missing
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0 FROM pragma_table_info('_graft_migrations') 
		WHERE name = 'logs'
	`).Scan(&exists)

	if err != nil {
		return err
	}

	if !exists {
		_, err = s.db.ExecContext(ctx, "ALTER TABLE _graft_migrations ADD COLUMN logs TEXT")
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLiteAdapter) CleanupBrokenMigrationRecords(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM _graft_migrations 
		WHERE finished_at IS NULL 
		AND started_at < datetime('now', '-1 hour')
	`)
	return err
}

func (s *SQLiteAdapter) GetAppliedMigrations(ctx context.Context) (map[string]*time.Time, error) {
	applied := make(map[string]*time.Time)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, finished_at 
		FROM _graft_migrations 
		WHERE finished_at IS NOT NULL
		ORDER BY started_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var finishedAt *time.Time
		if err := rows.Scan(&id, &finishedAt); err != nil {
			return nil, err
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

	// Insert migration record
	_, err = tx.ExecContext(ctx, `
		INSERT INTO _graft_migrations (id, migration_name, checksum, started_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`, migrationID, name, checksum)
	if err != nil {
		return err
	}

	// Mark as finished
	_, err = tx.ExecContext(ctx, `
		UPDATE _graft_migrations 
		SET finished_at = CURRENT_TIMESTAMP, applied_steps_count = 1
		WHERE id = ?
	`, migrationID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteAdapter) ExecuteMigration(ctx context.Context, migrationSQL string) error {
	_, err := s.db.ExecContext(ctx, migrationSQL)
	return err
}

func (s *SQLiteAdapter) GetCurrentSchema(ctx context.Context) ([]types.SchemaTable, error) {
	tables := []types.SchemaTable{}

	rows, err := s.db.QueryContext(ctx, `
		SELECT name FROM sqlite_master 
		WHERE type = 'table' 
		AND name != '_graft_migrations'
		AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}

		columns, err := s.GetTableColumns(ctx, tableName)
		if err != nil {
			return nil, err
		}

		indexes, err := s.GetTableIndexes(ctx, tableName)
		if err != nil {
			return nil, err
		}

		tables = append(tables, types.SchemaTable{
			Name:    tableName,
			Columns: columns,
			Indexes: indexes,
		})
	}

	return tables, nil
}

func (s *SQLiteAdapter) GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error) {
	columns := []types.SchemaColumn{}

	// Get the CREATE TABLE statement for constraint analysis
	var createSQL string
	err := s.db.QueryRowContext(ctx,
		"SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&createSQL)
	if err != nil {
		// If we can't get the CREATE statement, continue with basic info
		createSQL = ""
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var column types.SchemaColumn
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int

		err := rows.Scan(&cid, &column.Name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			return nil, err
		}

		column.Type = s.MapColumnType(dataType)
		column.Nullable = notNull == 0
		column.IsPrimary = pk > 0

		if defaultValue.Valid {
			column.Default = defaultValue.String
		}

		// Check for UNIQUE constraint in CREATE TABLE statement
		if createSQL != "" {
			column.IsUnique = s.isColumnUniqueFromSQL(createSQL, column.Name)
		}

		columns = append(columns, column)
	}

	// Fallback: Check for unique constraints using index information
	for i := range columns {
		if !columns[i].IsUnique {
			unique, err := s.isColumnUnique(ctx, tableName, columns[i].Name)
			if err != nil {
				log.Printf("Warning: failed to check unique constraint for column %s: %v", columns[i].Name, err)
			}
			columns[i].IsUnique = unique
		}
	}

	return columns, nil
}

// isColumnUniqueFromSQL parses CREATE TABLE statement to check for UNIQUE constraints
func (s *SQLiteAdapter) isColumnUniqueFromSQL(createSQL, columnName string) bool {
	if createSQL == "" {
		return false
	}

	// Convert to lowercase for case-insensitive matching
	sqlLower := strings.ToLower(createSQL)
	columnLower := strings.ToLower(columnName)

	// Look for column definitions with UNIQUE keyword
	// Pattern: `columnName` ... UNIQUE or columnName ... UNIQUE
	patterns := []string{
		fmt.Sprintf("`%s`", columnLower),
		fmt.Sprintf("\"%s\"", columnLower),
		fmt.Sprintf("'%s'", columnLower),
		columnLower,
	}

	for _, pattern := range patterns {
		// Find the column definition
		index := strings.Index(sqlLower, pattern)
		if index == -1 {
			continue
		}

		// Look for UNIQUE keyword after the column name but before the next column or constraint
		remaining := sqlLower[index:]
		nextComma := strings.Index(remaining, ",")
		nextParen := strings.Index(remaining, ")")

		// Determine the end of this column definition
		endPos := len(remaining)
		if nextComma != -1 && (nextParen == -1 || nextComma < nextParen) {
			endPos = nextComma
		} else if nextParen != -1 {
			endPos = nextParen
		}

		columnDef := remaining[:endPos]
		if strings.Contains(columnDef, "unique") {
			return true
		}
	}

	return false
}

// isColumnUnique checks if a column has a unique constraint in SQLite
func (s *SQLiteAdapter) isColumnUnique(ctx context.Context, tableName, columnName string) (bool, error) {
	// Check for unique indexes on single columns
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var seq int
		var indexName string
		var unique int
		var origin string
		var partial int

		err := rows.Scan(&seq, &indexName, &unique, &origin, &partial)
		if err != nil {
			continue
		}

		// Skip non-unique indexes
		if unique == 0 {
			continue
		}

		// Check if this unique index is on a single column matching our column
		colRows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_info(%s)", indexName))
		if err != nil {
			continue
		}

		var columns []string
		for colRows.Next() {
			var seqno int
			var cid int
			var name string

			err := colRows.Scan(&seqno, &cid, &name)
			if err != nil {
				continue
			}
			columns = append(columns, name)
		}
		colRows.Close()

		// If this is a single-column unique index on our column, return true
		if len(columns) == 1 && columns[0] == columnName {
			return true, nil
		}
	}

	return false, nil
}

func (s *SQLiteAdapter) GetTableIndexes(ctx context.Context, tableName string) ([]types.SchemaIndex, error) {
	indexes := []types.SchemaIndex{}

	// Get index list
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var seq int
		var indexName string
		var unique int
		var origin string
		var partial int

		err := rows.Scan(&seq, &indexName, &unique, &origin, &partial)
		if err != nil {
			return nil, err
		}

		// Skip auto-created indexes
		if origin == "pk" {
			continue
		}

		// Get index columns
		colRows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_info(%s)", indexName))
		if err != nil {
			continue
		}

		var columns []string
		for colRows.Next() {
			var seqno, cid int
			var name string

			if err := colRows.Scan(&seqno, &cid, &name); err != nil {
				continue
			}
			columns = append(columns, name)
		}
		colRows.Close()

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
	var tables []string

	rows, err := s.db.QueryContext(ctx, `
		SELECT name FROM sqlite_master 
		WHERE type = 'table' 
		AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

func (s *SQLiteAdapter) CheckTableExists(ctx context.Context, tableName string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0 FROM sqlite_master 
		WHERE type = 'table' AND name = ?
	`, tableName).Scan(&exists)
	return exists, err
}

func (s *SQLiteAdapter) CheckColumnExists(ctx context.Context, tableName, columnName string) (bool, error) {
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

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			continue
		}

		if name == columnName {
			return true, nil
		}
	}

	return false, nil
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

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			continue
		}

		if name == columnName {
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
		var id int
		var seq int
		var table, from, to, onUpdate, onDelete, match string

		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			continue
		}

		// SQLite doesn't store constraint names, so we check by referenced table
		if strings.Contains(constraintName, table) {
			return true, nil
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
		var origin string
		var partial int

		if err := rows.Scan(&seq, &indexName, &unique, &origin, &partial); err != nil {
			continue
		}

		if indexName == constraintName && unique == 1 {
			return true, nil
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
			return nil, err
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
	if err != nil {
		log.Printf("Error dropping table %s: %v", tableName, err)
		return err
	}
	return nil
}

func (s *SQLiteAdapter) GenerateCreateTableSQL(table types.SchemaTable) string {
	var builder strings.Builder
	var foreignKeys []string

	builder.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n", table.Name))

	for i, column := range table.Columns {
		if i > 0 {
			builder.WriteString(",\n")
		}
		builder.WriteString(fmt.Sprintf("    `%s` %s", column.Name, s.FormatColumnType(column)))

		// Collect foreign key constraints for table-level definition
		if column.ForeignKeyTable != "" && column.ForeignKeyColumn != "" {
			fkConstraint := fmt.Sprintf("FOREIGN KEY (`%s`) REFERENCES `%s`(`%s`)",
				column.Name, column.ForeignKeyTable, column.ForeignKeyColumn)
			if column.OnDeleteAction != "" {
				fkConstraint += fmt.Sprintf(" ON DELETE %s", column.OnDeleteAction)
			}
			foreignKeys = append(foreignKeys, fkConstraint)
		}
	}

	// Add foreign key constraints
	for _, fk := range foreignKeys {
		builder.WriteString(",\n    ")
		builder.WriteString(fk)
	}

	builder.WriteString("\n);")
	return builder.String()
}

func (s *SQLiteAdapter) GenerateAddColumnSQL(tableName string, column types.SchemaColumn) string {
	// SQLite doesn't have sophisticated conditional logic, but we can check in application layer
	// For now, generate standard SQL - the application should check if column exists before calling this
	return fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s;",
		tableName, column.Name, s.FormatColumnType(column))
}

func (s *SQLiteAdapter) GenerateDropColumnSQL(tableName, columnName string) string {
	// SQLite doesn't support DROP COLUMN directly.
	// The proper way is to recreate the table without the column.
	// This generates a comment indicating manual intervention is needed.
	return fmt.Sprintf("-- SQLite does not support DROP COLUMN. To remove column '%s' from table '%s', you need to:\n-- 1. CREATE TABLE %s_temp AS SELECT (all columns except %s) FROM %s;\n-- 2. DROP TABLE %s;\n-- 3. ALTER TABLE %s_temp RENAME TO %s;",
		columnName, tableName, tableName, columnName, tableName, tableName, tableName, tableName)
}

func (s *SQLiteAdapter) GenerateAddIndexSQL(index types.SchemaIndex) string {
	uniqueStr := ""
	if index.Unique {
		uniqueStr = "UNIQUE "
	}

	columnsStr := "`" + strings.Join(index.Columns, "`, `") + "`"
	return fmt.Sprintf("CREATE %sINDEX `%s` ON `%s` (%s);",
		uniqueStr, index.Name, index.Table, columnsStr)
}

func (s *SQLiteAdapter) GenerateDropIndexSQL(indexName string) string {
	return fmt.Sprintf("DROP INDEX IF EXISTS `%s`;", indexName)
}

func (s *SQLiteAdapter) MapColumnType(dbType string) string {
	switch strings.ToLower(dbType) {
	case "varchar", "text", "char":
		return "TEXT"
	case "int", "integer", "bigint", "smallint", "tinyint":
		return "INTEGER"
	case "real", "double", "float":
		return "REAL"
	case "blob":
		return "BLOB"
	case "numeric", "decimal":
		return "NUMERIC"
	case "boolean", "bool":
		return "INTEGER" // SQLite uses INTEGER for boolean
	case "date", "datetime", "timestamp":
		return "TEXT" // SQLite stores dates as TEXT
	default:
		return strings.ToUpper(dbType)
	}
}

func (s *SQLiteAdapter) FormatColumnType(column types.SchemaColumn) string {
	var parts []string

	// Add the base type
	parts = append(parts, column.Type)

	// Add PRIMARY KEY constraint with AUTOINCREMENT for INTEGER
	if column.IsPrimary {
		if strings.ToUpper(column.Type) == "INTEGER" {
			parts = append(parts, "PRIMARY KEY AUTOINCREMENT")
		} else {
			parts = append(parts, "PRIMARY KEY")
		}
	}

	// Add UNIQUE constraint (but not if it's already PRIMARY KEY)
	if column.IsUnique && !column.IsPrimary {
		parts = append(parts, "UNIQUE")
	}

	// Add constraints
	if !column.Nullable {
		parts = append(parts, "NOT NULL")
	}

	// Add default value
	if column.Default != "" {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", column.Default))
	}

	return strings.Join(parts, " ")
}
