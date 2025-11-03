package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/Rana718/Graft/internal/types"
	_ "github.com/go-sql-driver/mysql"
)

type MySQLAdapter struct {
	db *sql.DB
	qb squirrel.StatementBuilderType
}

// Type mappings
var typeMap = map[string]string{
	"varchar": "VARCHAR", "char": "CHAR",
	"text": "TEXT", "longtext": "TEXT", "mediumtext": "TEXT", "tinytext": "TEXT",
	"int": "INT", "integer": "INT", "bigint": "BIGINT", "smallint": "SMALLINT", "tinyint": "TINYINT",
	"boolean": "BOOLEAN", "bool": "BOOLEAN",
	"datetime": "DATETIME", "timestamp": "TIMESTAMP", "date": "DATE", "time": "TIME",
	"decimal": "DECIMAL", "numeric": "DECIMAL", "float": "FLOAT", "double": "DOUBLE",
	"json": "JSON", "blob": "BLOB", "binary": "BINARY", "varbinary": "VARBINARY",
}

func NewMySQLAdapter() *MySQLAdapter {
	return &MySQLAdapter{
		qb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
	}
}

func (m *MySQLAdapter) Connect(ctx context.Context, url string) error {
	db, err := sql.Open("mysql", url)
	if err != nil {
		return fmt.Errorf("failed to open MySQL connection: %w", err)
	}
	m.db = db
	return nil
}

func (m *MySQLAdapter) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *MySQLAdapter) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

// Migration table operations
func (m *MySQLAdapter) CreateMigrationsTable(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS _graft_migrations (
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

func (m *MySQLAdapter) EnsureMigrationTableCompatibility(ctx context.Context) error {
	exists, err := m.columnExists("_graft_migrations", "logs")
	if err != nil {
		return err
	}
	if !exists {
		_, err = m.db.ExecContext(ctx, "ALTER TABLE _graft_migrations ADD COLUMN logs TEXT")
	}
	return err
}

func (m *MySQLAdapter) CleanupBrokenMigrationRecords(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, `
		DELETE FROM _graft_migrations 
		WHERE finished_at IS NULL AND started_at < DATE_SUB(NOW(), INTERVAL 1 HOUR)
	`)
	return err
}

// Core migration operations
func (m *MySQLAdapter) GetAppliedMigrations(ctx context.Context) (map[string]*time.Time, error) {
	applied := make(map[string]*time.Time)
	query := m.qb.Select("id", "finished_at").From("_graft_migrations").
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

func (m *MySQLAdapter) RecordMigration(ctx context.Context, migrationID, name, checksum string) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO _graft_migrations (id, migration_name, checksum, started_at, finished_at, applied_steps_count)
		VALUES (?, ?, ?, NOW(), NOW(), 1)
	`, migrationID, name, checksum)

	if err != nil {
		return err
	}
	return tx.Commit()
}

func (m *MySQLAdapter) ExecuteMigration(ctx context.Context, migrationSQL string) error {
	// Start a transaction for the entire migration
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Will be ignored if already committed

	// Parse SQL statements properly handling multi-line statements
	statements := m.parseSQLStatements(migrationSQL)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue // Skip empty statements
		}

		_, err := tx.ExecContext(ctx, stmt)
		if err != nil {
			return fmt.Errorf("failed to execute statement '%s': %w", stmt, err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	return nil
}

// parseSQLStatements properly parses SQL statements handling multi-line CREATE TABLE statements
func (m *MySQLAdapter) parseSQLStatements(sql string) []string {
	var statements []string
	var currentStatement strings.Builder
	var inParentheses int
	var inQuotes bool
	var quoteChar rune

	lines := strings.Split(sql, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		// Process each character to track parentheses and quotes
		for i, char := range line {
			switch {
			case !inQuotes && (char == '"' || char == '\'' || char == '`'):
				inQuotes = true
				quoteChar = char
				currentStatement.WriteRune(char)
			case inQuotes && char == quoteChar:
				// Check if it's escaped
				if i > 0 && rune(line[i-1]) == '\\' {
					currentStatement.WriteRune(char)
				} else {
					inQuotes = false
					currentStatement.WriteRune(char)
				}
			case !inQuotes && char == '(':
				inParentheses++
				currentStatement.WriteRune(char)
			case !inQuotes && char == ')':
				inParentheses--
				currentStatement.WriteRune(char)
			case !inQuotes && char == ';' && inParentheses == 0:
				// End of statement
				stmt := strings.TrimSpace(currentStatement.String())
				if stmt != "" {
					statements = append(statements, stmt)
				}
				currentStatement.Reset()
			default:
				currentStatement.WriteRune(char)
			}
		}

		// Add newline if we're still building a statement
		if currentStatement.Len() > 0 {
			currentStatement.WriteRune('\n')
		}
	}

	// Add any remaining statement
	if currentStatement.Len() > 0 {
		stmt := strings.TrimSpace(currentStatement.String())
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}

	return statements
}

// Schema introspection
func (m *MySQLAdapter) GetCurrentSchema(ctx context.Context) ([]types.SchemaTable, error) {
	tableNames, err := m.GetAllTableNames(ctx)
	if err != nil {
		return nil, err
	}

	var tables []types.SchemaTable
	for _, name := range tableNames {
		if name == "_graft_migrations" {
			continue
		}

		columns, err := m.GetTableColumns(ctx, name)
		if err != nil {
			return nil, err
		}

		indexes, err := m.GetTableIndexes(ctx, name)
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

func (m *MySQLAdapter) GetCurrentEnums(ctx context.Context) ([]types.SchemaEnum, error) {
	// MySQL doesn't have native ENUM types like PostgreSQL
	return []types.SchemaEnum{}, nil
}

func (m *MySQLAdapter) GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT 
			c.column_name, c.data_type, c.is_nullable, c.column_default,
			c.character_maximum_length, c.numeric_precision, c.numeric_scale, c.column_type,
			CASE WHEN c.column_key = 'PRI' THEN 1 ELSE 0 END as is_primary_key,
			CASE WHEN c.column_key = 'UNI' THEN 1 ELSE 0 END as is_unique
		FROM information_schema.columns c
		WHERE c.table_name = ? AND c.table_schema = DATABASE()
		ORDER BY c.ordinal_position
	`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []types.SchemaColumn
	for rows.Next() {
		var column types.SchemaColumn
		var dataType, isNullable, columnType string
		var columnDefault sql.NullString
		var charMaxLength, numericPrecision, numericScale sql.NullInt64
		var isPrimary, isUnique int

		err := rows.Scan(&column.Name, &dataType, &isNullable, &columnDefault,
			&charMaxLength, &numericPrecision, &numericScale, &columnType,
			&isPrimary, &isUnique)
		if err != nil {
			return nil, err
		}

		column.Type = m.formatMySQLType(dataType, columnType, charMaxLength, numericPrecision, numericScale)
		column.Nullable = isNullable == "YES"
		column.IsPrimary = isPrimary == 1
		column.IsUnique = isUnique == 1
		if columnDefault.Valid {
			column.Default = columnDefault.String
		}

		columns = append(columns, column)
	}
	return columns, nil
}

func (m *MySQLAdapter) GetTableIndexes(ctx context.Context, tableName string) ([]types.SchemaIndex, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT index_name, column_name, non_unique
		FROM information_schema.statistics
		WHERE table_name = ? AND table_schema = DATABASE() AND index_name != 'PRIMARY'
		ORDER BY index_name, seq_in_index
	`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*types.SchemaIndex)
	for rows.Next() {
		var indexName, columnName string
		var nonUnique int

		if err := rows.Scan(&indexName, &columnName, &nonUnique); err != nil {
			return nil, err
		}

		if idx, exists := indexMap[indexName]; exists {
			idx.Columns = append(idx.Columns, columnName)
		} else {
			indexMap[indexName] = &types.SchemaIndex{
				Name:    indexName,
				Table:   tableName,
				Columns: []string{columnName},
				Unique:  nonUnique == 0,
			}
		}
	}

	var indexes []types.SchemaIndex
	for _, idx := range indexMap {
		indexes = append(indexes, *idx)
	}
	return indexes, nil
}

func (m *MySQLAdapter) GetAllTableNames(ctx context.Context) ([]string, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT table_name FROM information_schema.tables 
		WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}
	return tables, nil
}

// Table/column existence checks - consolidated helper
func (m *MySQLAdapter) tableExists(tableName string) (bool, error) {
	var exists bool
	err := m.db.QueryRow(`
		SELECT COUNT(*) > 0 FROM information_schema.tables 
		WHERE table_name = ? AND table_schema = DATABASE()
	`, tableName).Scan(&exists)
	return exists, err
}

func (m *MySQLAdapter) columnExists(tableName, columnName string) (bool, error) {
	var exists bool
	err := m.db.QueryRow(`
		SELECT COUNT(*) > 0 FROM information_schema.columns 
		WHERE table_name = ? AND column_name = ? AND table_schema = DATABASE()
	`, tableName, columnName).Scan(&exists)
	return exists, err
}

func (m *MySQLAdapter) CheckTableExists(ctx context.Context, tableName string) (bool, error) {
	return m.tableExists(tableName)
}

func (m *MySQLAdapter) CheckColumnExists(ctx context.Context, tableName, columnName string) (bool, error) {
	return m.columnExists(tableName, columnName)
}

func (m *MySQLAdapter) CheckNotNullConstraint(ctx context.Context, tableName, columnName string) (bool, error) {
	var isNullable string
	err := m.db.QueryRowContext(ctx, `
		SELECT is_nullable FROM information_schema.columns 
		WHERE table_name = ? AND column_name = ? AND table_schema = DATABASE()
	`, tableName, columnName).Scan(&isNullable)
	if err != nil {
		return false, err
	}
	return isNullable == "NO", nil
}

func (m *MySQLAdapter) CheckForeignKeyConstraint(ctx context.Context, tableName, constraintName string) (bool, error) {
	return m.checkConstraint(tableName, constraintName, "FOREIGN KEY")
}

func (m *MySQLAdapter) CheckUniqueConstraint(ctx context.Context, tableName, constraintName string) (bool, error) {
	return m.checkConstraint(tableName, constraintName, "UNIQUE")
}

// Helper for constraint checking
func (m *MySQLAdapter) checkConstraint(tableName, constraintName, constraintType string) (bool, error) {
	var exists bool
	err := m.db.QueryRow(`
		SELECT COUNT(*) > 0 FROM information_schema.table_constraints 
		WHERE table_name = ? AND constraint_name = ? AND constraint_type = ? AND table_schema = DATABASE()
	`, tableName, constraintName, constraintType).Scan(&exists)
	return exists, err
}

func (m *MySQLAdapter) GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error) {
	rows, err := m.db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM `%s`", tableName))
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

func (m *MySQLAdapter) DropTable(ctx context.Context, tableName string) error {
	_, err := m.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName))
	return err
}

// SQL Generation
func (m *MySQLAdapter) GenerateCreateTableSQL(table types.SchemaTable) string {
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
		lines = append(lines, fmt.Sprintf("  `%s` %s%s", column.Name, m.FormatColumnType(column), comma))
	}

	for i, fk := range foreignKeys {
		comma := ","
		if i == len(foreignKeys)-1 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf("%s%s", fk, comma))
	}

	lines = append(lines, ") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;")
	return strings.Join(lines, "\n")
}

func (m *MySQLAdapter) GenerateAddColumnSQL(tableName string, column types.SchemaColumn) string {
	return fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s;",
		tableName, column.Name, m.FormatColumnType(column))
}

func (m *MySQLAdapter) GenerateDropColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`;", tableName, columnName)
}

func (m *MySQLAdapter) GenerateAddIndexSQL(index types.SchemaIndex) string {
	unique := ""
	if index.Unique {
		unique = "UNIQUE "
	}
	columns := "`" + strings.Join(index.Columns, "`, `") + "`"
	return fmt.Sprintf("CREATE %sINDEX `%s` ON `%s` (%s);", unique, index.Name, index.Table, columns)
}

func (m *MySQLAdapter) GenerateDropIndexSQL(indexName string) string {
	return fmt.Sprintf("DROP INDEX `%s`;", indexName)
}

// Type mapping and formatting
func (m *MySQLAdapter) MapColumnType(dbType string) string {
	if mapped, exists := typeMap[strings.ToLower(dbType)]; exists {
		return mapped
	}
	return strings.ToUpper(dbType)
}

func (m *MySQLAdapter) FormatColumnType(column types.SchemaColumn) string {
	var parts []string
	parts = append(parts, column.Type)

	if column.IsPrimary {
		parts = append(parts, "PRIMARY KEY")
		if strings.Contains(strings.ToUpper(column.Type), "INT") {
			parts = append(parts, "AUTO_INCREMENT")
		}
	}

	if column.IsUnique && !column.IsPrimary {
		parts = append(parts, "UNIQUE")
	}

	if !column.Nullable && !column.IsPrimary {
		parts = append(parts, "NOT NULL")
	}

	if column.Default != "" {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", column.Default))
	}

	return strings.Join(parts, " ")
}

func (m *MySQLAdapter) formatMySQLType(dataType, columnType string, charMaxLength, numericPrecision, numericScale sql.NullInt64) string {
	switch dataType {
	case "varchar":
		if charMaxLength.Valid {
			return fmt.Sprintf("VARCHAR(%d)", charMaxLength.Int64)
		}
		return "VARCHAR(255)"
	case "char":
		if charMaxLength.Valid {
			return fmt.Sprintf("CHAR(%d)", charMaxLength.Int64)
		}
		return "CHAR(1)"
	case "decimal":
		if numericPrecision.Valid && numericScale.Valid {
			return fmt.Sprintf("DECIMAL(%d,%d)", numericPrecision.Int64, numericScale.Int64)
		} else if numericPrecision.Valid {
			return fmt.Sprintf("DECIMAL(%d)", numericPrecision.Int64)
		}
		return "DECIMAL"
	default:
		if columnType != "" {
			return strings.ToUpper(columnType)
		}
		return m.MapColumnType(dataType)
	}
}

func (m *MySQLAdapter) PullCompleteSchema(ctx context.Context) ([]types.SchemaTable, error) {
	query := `
	SELECT
		c.TABLE_NAME,
		c.COLUMN_NAME,
		c.COLUMN_TYPE,
		c.IS_NULLABLE,
		c.COLUMN_DEFAULT,
		c.EXTRA,
		c.ORDINAL_POSITION,
		CASE WHEN c.COLUMN_KEY = 'PRI' THEN 'PRIMARY KEY' ELSE NULL END as is_primary,
		CASE WHEN c.COLUMN_KEY = 'UNI' THEN 'UNIQUE' ELSE NULL END as is_unique,
		k.REFERENCED_TABLE_NAME AS REFERENCES_TABLE,
		k.REFERENCED_COLUMN_NAME AS REFERENCES_COLUMN,
		r.DELETE_RULE AS ON_DELETE
	FROM INFORMATION_SCHEMA.COLUMNS c
	LEFT JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE k
		ON c.TABLE_SCHEMA = k.TABLE_SCHEMA
		AND c.TABLE_NAME = k.TABLE_NAME
		AND c.COLUMN_NAME = k.COLUMN_NAME
		AND k.REFERENCED_TABLE_NAME IS NOT NULL
	LEFT JOIN INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS r
		ON k.CONSTRAINT_NAME = r.CONSTRAINT_NAME
		AND k.TABLE_SCHEMA = r.CONSTRAINT_SCHEMA
	WHERE c.TABLE_SCHEMA = DATABASE()
		AND c.TABLE_NAME NOT LIKE '_graft_%'
	ORDER BY c.TABLE_NAME, c.ORDINAL_POSITION`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schema: %w", err)
	}
	defer rows.Close()

	tableMap := make(map[string]*types.SchemaTable)
	columnsSeen := make(map[string]map[string]bool)

	for rows.Next() {
		var tableName, columnName, columnType, isNullable string
		var ordinalPosition int
		var columnDefault, extra, isPrimary, isUnique, referencesTable, referencesColumn, onDelete sql.NullString

		err := rows.Scan(&tableName, &columnName, &columnType, &isNullable, &columnDefault,
			&extra, &ordinalPosition, &isPrimary, &isUnique, &referencesTable, &referencesColumn, &onDelete)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if _, exists := tableMap[tableName]; !exists {
			tableMap[tableName] = &types.SchemaTable{
				Name:    tableName,
				Columns: []types.SchemaColumn{},
			}
			columnsSeen[tableName] = make(map[string]bool)
		}

		if columnsSeen[tableName][columnName] {
			continue
		}
		columnsSeen[tableName][columnName] = true

		// Format column type
		formattedType := m.formatMySQLPullType(columnType)

		column := types.SchemaColumn{
			Name:      columnName,
			Type:      formattedType,
			Nullable:  isNullable == "YES",
			Default:   m.formatMySQLDefault(columnDefault.String, formattedType),
			IsPrimary: isPrimary.Valid,
			IsUnique:  isUnique.Valid,
		}

		if referencesTable.Valid && referencesColumn.Valid {
			column.ForeignKeyTable = referencesTable.String
			column.ForeignKeyColumn = referencesColumn.String
			if onDelete.Valid {
				column.OnDeleteAction = onDelete.String
			}
		}

		tableMap[tableName].Columns = append(tableMap[tableName].Columns, column)
	}

	var tables []types.SchemaTable
	for _, table := range tableMap {
		tables = append(tables, *table)
	}

	return tables, nil
}

func (m *MySQLAdapter) formatMySQLPullType(columnType string) string {
	columnType = strings.ToUpper(columnType)

	// Handle common MySQL types
	if strings.HasPrefix(columnType, "INT(") {
		return "INT"
	}
	if strings.HasPrefix(columnType, "BIGINT(") {
		return "BIGINT"
	}
	if strings.HasPrefix(columnType, "SMALLINT(") {
		return "SMALLINT"
	}
	if strings.HasPrefix(columnType, "TINYINT(1)") {
		return "BOOLEAN"
	}
	if strings.HasPrefix(columnType, "TINYINT(") {
		return "TINYINT"
	}

	return columnType
}

func (m *MySQLAdapter) formatMySQLDefault(defaultValue, columnType string) string {
	if defaultValue == "" {
		return ""
	}

	// Handle MySQL specific defaults
	if strings.Contains(strings.ToLower(defaultValue), "current_timestamp") {
		return "CURRENT_TIMESTAMP"
	}

	return defaultValue
}
