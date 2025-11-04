package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/Rana718/Graft/internal/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

type PostgresAdapter struct {
	pool *pgxpool.Pool
	qb   squirrel.StatementBuilderType
}

// PostgreSQL type mappings
var pgTypeMap = map[string]string{
	"character varying": "VARCHAR", "varchar": "VARCHAR",
	"character": "CHAR", "char": "CHAR", "text": "TEXT",
	"integer": "INTEGER", "int4": "INTEGER", "bigint": "BIGINT", "int8": "BIGINT",
	"smallint": "SMALLINT", "int2": "SMALLINT", "boolean": "BOOLEAN", "bool": "BOOLEAN",
	"timestamp with time zone": "TIMESTAMP WITH TIME ZONE", "timestamptz": "TIMESTAMP WITH TIME ZONE",
	"timestamp without time zone": "TIMESTAMP", "timestamp": "TIMESTAMP",
	"date": "DATE", "time": "TIME", "numeric": "NUMERIC", "decimal": "NUMERIC",
	"real": "REAL", "float4": "REAL", "double precision": "DOUBLE PRECISION", "float8": "DOUBLE PRECISION",
	"uuid": "UUID", "json": "JSON", "jsonb": "JSONB",
}

func NewPostgresAdapter() *PostgresAdapter {
	return &PostgresAdapter{
		qb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (p *PostgresAdapter) Connect(ctx context.Context, url string) error {
	// Configure pool for Supabase pooler compatibility
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return fmt.Errorf("failed to parse connection URL: %w", err)
	}

	// Use exec mode for pooler compatibility (Supabase, PgBouncer)
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec

	// Optimize pool settings
	config.MaxConns = 5
	config.MinConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	p.pool = pool
	return nil
}

func (p *PostgresAdapter) Close() error {
	if p.pool != nil {
		p.pool.Close()
	}
	return nil
}

func (p *PostgresAdapter) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Migration table operations
func (p *PostgresAdapter) CreateMigrationsTable(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS _graft_migrations (
		id VARCHAR(255) PRIMARY KEY,
		checksum VARCHAR(64) NOT NULL,
		finished_at TIMESTAMP WITH TIME ZONE,
		migration_name VARCHAR(255) NOT NULL,
		logs TEXT,
		rolled_back_at TIMESTAMP WITH TIME ZONE,
		started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		applied_steps_count INTEGER NOT NULL DEFAULT 0
	)`
	_, err := p.pool.Exec(ctx, query)
	return err
}

func (p *PostgresAdapter) EnsureMigrationTableCompatibility(ctx context.Context) error {
	exists, err := p.columnExists("_graft_migrations", "logs")
	if err != nil {
		return err
	}
	if !exists {
		_, err = p.pool.Exec(ctx, "ALTER TABLE _graft_migrations ADD COLUMN logs TEXT")
	}
	return err
}

func (p *PostgresAdapter) CleanupBrokenMigrationRecords(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `
		DELETE FROM _graft_migrations 
		WHERE finished_at IS NULL AND started_at < NOW() - INTERVAL '1 hour'
	`)
	return err
}

// Core migration operations
func (p *PostgresAdapter) GetAppliedMigrations(ctx context.Context) (map[string]*time.Time, error) {
	applied := make(map[string]*time.Time)

	rows, err := p.pool.Query(ctx, `
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
			continue
		}
		applied[id] = finishedAt
	}
	return applied, nil
}

func (p *PostgresAdapter) RecordMigration(ctx context.Context, migrationID, name, checksum string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO _graft_migrations (id, migration_name, checksum, started_at, finished_at, applied_steps_count)
		VALUES ($1, $2, $3, NOW(), NOW(), 1)
	`, migrationID, name, checksum)

	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (p *PostgresAdapter) ExecuteMigration(ctx context.Context, migrationSQL string) error {
	// Start a transaction for the entire migration
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Will be ignored if already committed

	// Parse SQL statements properly handling multi-line statements
	statements := p.parseSQLStatements(migrationSQL)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue // Skip empty statements
		}

		_, err := tx.Exec(ctx, stmt)
		if err != nil {
			return fmt.Errorf("failed to execute statement '%s': %w", stmt, err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	return nil
}

// parseSQLStatements properly parses SQL statements handling multi-line CREATE TABLE statements
func (p *PostgresAdapter) parseSQLStatements(sql string) []string {
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
func (p *PostgresAdapter) GetCurrentSchema(ctx context.Context) ([]types.SchemaTable, error) {
	tableNames, err := p.GetAllTableNames(ctx)
	if err != nil {
		return nil, err
	}

	var tables []types.SchemaTable
	for _, name := range tableNames {
		if name == "_graft_migrations" {
			continue
		}

		columns, err := p.GetTableColumns(ctx, name)
		if err != nil {
			return nil, err
		}

		indexes, err := p.GetTableIndexes(ctx, name)
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

func (p *PostgresAdapter) GetCurrentEnums(ctx context.Context) ([]types.SchemaEnum, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT 
			t.typname as enum_name,
			e.enumlabel as enum_value
		FROM pg_type t
		JOIN pg_enum e ON t.oid = e.enumtypid
		JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = 'public'
		ORDER BY t.typname, e.enumsortorder
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	enumMap := make(map[string][]string)
	for rows.Next() {
		var enumName, enumValue string
		if err := rows.Scan(&enumName, &enumValue); err != nil {
			return nil, err
		}
		enumMap[enumName] = append(enumMap[enumName], enumValue)
	}

	var enums []types.SchemaEnum
	for name, values := range enumMap {
		enums = append(enums, types.SchemaEnum{
			Name:   name,
			Values: values,
		})
	}

	return enums, nil
}

func (p *PostgresAdapter) GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT 
			c.column_name, c.udt_name, c.is_nullable, c.column_default,
			c.character_maximum_length, c.numeric_precision, c.numeric_scale,
			CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_primary_key,
			CASE WHEN uq.column_name IS NOT NULL THEN true ELSE false END as is_unique,
			fk.foreign_table_name,
			fk.foreign_column_name,
			fk.on_delete_action
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku ON tc.constraint_name = ku.constraint_name
			WHERE tc.table_name = $1 AND tc.constraint_type = 'PRIMARY KEY'
		) pk ON c.column_name = pk.column_name
		LEFT JOIN (
			SELECT ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku ON tc.constraint_name = ku.constraint_name
			WHERE tc.table_name = $1 AND tc.constraint_type = 'UNIQUE'
		) uq ON c.column_name = uq.column_name
		LEFT JOIN (
			SELECT kcu.column_name, ccu.table_name AS foreign_table_name, ccu.column_name AS foreign_column_name, rc.delete_rule AS on_delete_action
			FROM information_schema.table_constraints AS tc
			JOIN information_schema.key_column_usage AS kcu ON tc.constraint_name = kcu.constraint_name
			JOIN information_schema.constraint_column_usage AS ccu ON ccu.constraint_name = tc.constraint_name
			JOIN information_schema.referential_constraints AS rc ON tc.constraint_name = rc.constraint_name
			WHERE tc.table_name = $1 AND tc.constraint_type = 'FOREIGN KEY'
		) fk ON c.column_name = fk.column_name
		WHERE c.table_name = $1 AND c.table_schema = 'public'
		ORDER BY c.ordinal_position
	`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []types.SchemaColumn
	for rows.Next() {
		var column types.SchemaColumn
		var udtName, isNullable string
		var columnDefault sql.NullString
		var charMaxLength, numericPrecision, numericScale sql.NullInt64
		var isPrimary, isUnique bool
		var fkTable, fkColumn, onDelete sql.NullString

		err := rows.Scan(&column.Name, &udtName, &isNullable, &columnDefault,
			&charMaxLength, &numericPrecision, &numericScale, &isPrimary, &isUnique, &fkTable, &fkColumn, &onDelete)
		if err != nil {
			return nil, err
		}

		column.Type = p.formatPostgresType(udtName, charMaxLength, numericPrecision, numericScale)
		column.Nullable = isNullable == "YES"
		column.IsPrimary = isPrimary
		column.IsUnique = isUnique
		if columnDefault.Valid {
			column.Default = p.cleanDefaultValue(columnDefault.String)
		}
		if fkTable.Valid {
			column.ForeignKeyTable = fkTable.String
		}
		if fkColumn.Valid {
			column.ForeignKeyColumn = fkColumn.String
		}
		if onDelete.Valid {
			column.OnDeleteAction = onDelete.String
		}

		columns = append(columns, column)
	}
	return columns, nil
}

func (p *PostgresAdapter) GetTableIndexes(ctx context.Context, tableName string) ([]types.SchemaIndex, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT indexname, indexdef
		FROM pg_indexes
		WHERE tablename = $1 AND schemaname = 'public' AND indexname NOT LIKE '%_pkey'
	`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []types.SchemaIndex
	for rows.Next() {
		var indexName, indexDef string
		if err := rows.Scan(&indexName, &indexDef); err != nil {
			continue
		}

		index := types.SchemaIndex{
			Name:   indexName,
			Table:  tableName,
			Unique: strings.Contains(strings.ToUpper(indexDef), "UNIQUE"),
		}

		if start := strings.Index(indexDef, "("); start != -1 {
			if end := strings.Index(indexDef[start:], ")"); end != -1 {
				columnsStr := indexDef[start+1 : start+end]
				for _, col := range strings.Split(columnsStr, ",") {
					index.Columns = append(index.Columns, strings.TrimSpace(col))
				}
			}
		}
		indexes = append(indexes, index)
	}
	return indexes, nil
}

func (p *PostgresAdapter) GetAllTableNames(ctx context.Context) ([]string, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT table_name FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
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
			continue
		}
		tables = append(tables, tableName)
	}
	return tables, nil
}

// Helper methods for existence checks
func (p *PostgresAdapter) tableExists(tableName string) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = $1 AND table_schema = 'public'
		)
	`, tableName).Scan(&exists)
	return exists, err
}

func (p *PostgresAdapter) columnExists(tableName, columnName string) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = $1 AND column_name = $2
		)
	`, tableName, columnName).Scan(&exists)
	return exists, err
}

func (p *PostgresAdapter) constraintExists(tableName, constraintName, constraintType string) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints 
			WHERE table_name = $1 AND constraint_name = $2 AND constraint_type = $3 AND table_schema = 'public'
		)
	`, tableName, constraintName, constraintType).Scan(&exists)
	return exists, err
}

func (p *PostgresAdapter) CheckTableExists(ctx context.Context, tableName string) (bool, error) {
	return p.tableExists(tableName)
}

func (p *PostgresAdapter) CheckColumnExists(ctx context.Context, tableName, columnName string) (bool, error) {
	return p.columnExists(tableName, columnName)
}

func (p *PostgresAdapter) CheckNotNullConstraint(ctx context.Context, tableName, columnName string) (bool, error) {
	var isNullable string
	err := p.pool.QueryRow(ctx, `
		SELECT is_nullable FROM information_schema.columns 
		WHERE table_name = $1 AND column_name = $2 AND table_schema = 'public'
	`, tableName, columnName).Scan(&isNullable)
	if err != nil {
		return false, err
	}
	return isNullable == "NO", nil
}

func (p *PostgresAdapter) CheckForeignKeyConstraint(ctx context.Context, tableName, constraintName string) (bool, error) {
	return p.constraintExists(tableName, constraintName, "FOREIGN KEY")
}

func (p *PostgresAdapter) CheckUniqueConstraint(ctx context.Context, tableName, constraintName string) (bool, error) {
	return p.constraintExists(tableName, constraintName, "UNIQUE")
}

func (p *PostgresAdapter) GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error) {
	columnInfo, err := p.GetTableColumns(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get column info: %w", err)
	}
	
	var selectCols []string
	for _, col := range columnInfo {
		if strings.Contains(col.Type, "_") || 
		   (!strings.Contains(col.Type, "int") && 
		    !strings.Contains(col.Type, "varchar") && 
		    !strings.Contains(col.Type, "text") &&
		    !strings.Contains(col.Type, "bool") &&
		    !strings.Contains(col.Type, "timestamp") &&
		    !strings.Contains(col.Type, "date") &&
		    !strings.Contains(col.Type, "numeric") &&
		    !strings.Contains(col.Type, "float") &&
		    !strings.Contains(col.Type, "serial")) {
			// Cast enum to text
			selectCols = append(selectCols, fmt.Sprintf("\"%s\"::text", col.Name))
		} else {
			selectCols = append(selectCols, fmt.Sprintf("\"%s\"", col.Name))
		}
	}
	
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(selectCols, ", "), tableName)
	
	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		fmt.Printf("ERROR querying table %s: %v\n", tableName, err)
		return nil, err
	}
	defer rows.Close()

	columns := rows.FieldDescriptions()
	var result []map[string]interface{}

	rowNum := 0
	for rows.Next() {
		rowNum++
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			fmt.Printf("ERROR scanning row %d in table %s: %v\n", rowNum, tableName, err)
			return result, nil
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			colName := string(col.Name)
			
			switch v := val.(type) {
			case []byte:
				row[colName] = string(v)
			case string:
				row[colName] = v
			case nil:
				row[colName] = nil
			case int, int8, int16, int32, int64:
				row[colName] = v
			case uint, uint8, uint16, uint32, uint64:
				row[colName] = v
			case float32, float64:
				row[colName] = v
			case bool:
				row[colName] = v
			default:
				row[colName] = fmt.Sprintf("%v", v)
			}
		}
		result = append(result, row)
	}
	
	if err := rows.Err(); err != nil {
		fmt.Printf("ERROR after reading rows from %s: %v\n", tableName, err)
		return result, err
	}

	// fmt.Printf("Successfully fetched %d rows from %s\n", len(result), tableName)  its here for dev reference
	return result, nil
}

func (p *PostgresAdapter) DropTable(ctx context.Context, tableName string) error {
	_, err := p.pool.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName))
	return err
}

// SQL Generation
func (p *PostgresAdapter) GenerateCreateTableSQL(table types.SchemaTable) string {
	var lines []string
	var foreignKeys []string

	for _, column := range table.Columns {
		if column.ForeignKeyTable != "" && column.ForeignKeyColumn != "" {
			fk := fmt.Sprintf("  FOREIGN KEY (\"%s\") REFERENCES \"%s\"(\"%s\")",
				column.Name, column.ForeignKeyTable, column.ForeignKeyColumn)
			if column.OnDeleteAction != "" {
				fk += fmt.Sprintf(" ON DELETE %s", column.OnDeleteAction)
			}
			foreignKeys = append(foreignKeys, fk)
		}
	}

	lines = append(lines, fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\" (", table.Name))

	for i, column := range table.Columns {
		comma := ","
		if i == len(table.Columns)-1 && len(foreignKeys) == 0 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf("  \"%s\" %s%s", column.Name, p.FormatColumnType(column), comma))
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

func (p *PostgresAdapter) GenerateAddColumnSQL(tableName string, column types.SchemaColumn) string {
	return fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN IF NOT EXISTS \"%s\" %s;",
		tableName, column.Name, p.FormatColumnType(column))
}

func (p *PostgresAdapter) GenerateDropColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE \"%s\" DROP COLUMN IF EXISTS \"%s\";", tableName, columnName)
}

func (p *PostgresAdapter) GenerateAddIndexSQL(index types.SchemaIndex) string {
	unique := ""
	if index.Unique {
		unique = "UNIQUE "
	}
	columns := strings.Join(index.Columns, ", ")
	return fmt.Sprintf("CREATE %sINDEX \"%s\" ON \"%s\" (%s);", unique, index.Name, index.Table, columns)
}

func (p *PostgresAdapter) GenerateDropIndexSQL(indexName string) string {
	return fmt.Sprintf("DROP INDEX IF EXISTS \"%s\";", indexName)
}

// Type mapping and formatting
func (p *PostgresAdapter) MapColumnType(dbType string) string {
	if mapped, exists := pgTypeMap[strings.ToLower(dbType)]; exists {
		return mapped
	}
	return strings.ToUpper(dbType)
}

func (p *PostgresAdapter) FormatColumnType(column types.SchemaColumn) string {
	var parts []string
	parts = append(parts, column.Type)

	if column.IsPrimary {
		parts = append(parts, "PRIMARY KEY")
	}

	if column.IsUnique && !column.IsPrimary {
		parts = append(parts, "UNIQUE")
	}

	if !column.Nullable && !column.IsPrimary {
		parts = append(parts, "NOT NULL")
	}

	if column.ForeignKeyTable != "" && column.ForeignKeyColumn != "" {
		parts = append(parts, fmt.Sprintf("REFERENCES \"%s\"(\"%s\")", column.ForeignKeyTable, column.ForeignKeyColumn))
		if column.OnDeleteAction != "" {
			parts = append(parts, fmt.Sprintf("ON DELETE %s", column.OnDeleteAction))
		}
	}

	if column.Default != "" && !strings.Contains(column.Default, "nextval") {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", column.Default))
	}

	return strings.Join(parts, " ")
}

func (p *PostgresAdapter) formatPostgresType(udtName string, charMaxLength, numericPrecision, numericScale sql.NullInt64) string {
	switch udtName {
	case "varchar", "character varying":
		if charMaxLength.Valid {
			return fmt.Sprintf("VARCHAR(%d)", charMaxLength.Int64)
		}
		return "VARCHAR"
	case "bpchar", "character":
		if charMaxLength.Valid {
			return fmt.Sprintf("CHAR(%d)", charMaxLength.Int64)
		}
		return "CHAR"
	case "numeric":
		if numericPrecision.Valid && numericScale.Valid {
			return fmt.Sprintf("NUMERIC(%d,%d)", numericPrecision.Int64, numericScale.Int64)
		} else if numericPrecision.Valid {
			return fmt.Sprintf("NUMERIC(%d)", numericPrecision.Int64)
		}
		return "NUMERIC"
	case "timestamptz":
		return "TIMESTAMP WITH TIME ZONE"
	case "timestamp":
		return "TIMESTAMP"
	default:
		// Check if this is a known built-in type
		if mapped, exists := pgTypeMap[strings.ToLower(udtName)]; exists {
			return mapped
		}
		// For custom types (like enums), keep original case
		return udtName
	}
}

func (p *PostgresAdapter) cleanDefaultValue(defaultVal string) string {
	if defaultVal == "" {
		return ""
	}

	// Remove PostgreSQL type casts like 'value'::enum_type or value::type
	if idx := strings.Index(defaultVal, "::"); idx != -1 {
		// Extract everything before the ::
		value := strings.TrimSpace(defaultVal[:idx])

		// Special case: if it's nextval, remove entirely
		if strings.Contains(strings.ToLower(value), "nextval") {
			return ""
		}

		// Special case: if it's NOW() or timestamp functions
		if strings.Contains(strings.ToUpper(value), "NOW()") || strings.Contains(strings.ToUpper(value), "CURRENT_TIMESTAMP") {
			return "NOW()"
		}

		// For everything else, return the value part (keeps quotes if present)
		return value
	}

	// Handle special cases without type casts
	upper := strings.ToUpper(defaultVal)
	if strings.Contains(upper, "NEXTVAL") {
		return "" // Remove serial defaults
	}
	if strings.Contains(upper, "NOW()") || strings.Contains(upper, "CURRENT_TIMESTAMP") {
		return "NOW()"
	}
	if upper == "TRUE" || upper == "FALSE" {
		return upper // Normalize boolean to uppercase
	}

	return defaultVal
}

func (p *PostgresAdapter) PullCompleteSchema(ctx context.Context) ([]types.SchemaTable, error) {
	query := `
	SELECT 
		c.table_name,
		c.column_name,
		c.udt_name,
		c.is_nullable,
		c.column_default,
		c.character_maximum_length,
		c.numeric_precision,
		c.numeric_scale,
		c.ordinal_position,
		CASE WHEN pk.column_name IS NOT NULL THEN 'PRIMARY KEY' ELSE NULL END as is_primary,
		CASE WHEN uq.column_name IS NOT NULL THEN 'UNIQUE' ELSE NULL END as is_unique,
		fk.foreign_table_name,
		fk.foreign_column_name,
		fk.delete_rule
	FROM information_schema.columns c
	LEFT JOIN (
		SELECT kcu.table_name, kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name 
			AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY' AND tc.table_schema = 'public'
	) pk ON c.table_name = pk.table_name AND c.column_name = pk.column_name
	LEFT JOIN (
		SELECT kcu.table_name, kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name 
			AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'UNIQUE' AND tc.table_schema = 'public'
	) uq ON c.table_name = uq.table_name AND c.column_name = uq.column_name
	LEFT JOIN (
		SELECT 
			kcu.table_name, 
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			rc.delete_rule
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name 
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu 
			ON tc.constraint_name = ccu.constraint_name 
			AND tc.table_schema = ccu.table_schema
		JOIN information_schema.referential_constraints rc 
			ON tc.constraint_name = rc.constraint_name 
			AND tc.table_schema = rc.constraint_schema
		WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_schema = 'public'
	) fk ON c.table_name = fk.table_name AND c.column_name = fk.column_name
	WHERE c.table_schema = 'public' 
		AND c.table_name NOT LIKE '_graft_%'
	ORDER BY c.table_name, c.ordinal_position`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schema: %w", err)
	}
	defer rows.Close()

	tableMap := make(map[string]*types.SchemaTable)
	columnsSeen := make(map[string]map[string]bool) // table -> column -> seen

	for rows.Next() {
		var tableName, columnName, udtName, isNullable string
		var ordinalPosition int
		var columnDefault, isPrimary, isUnique, foreignTable, foreignColumn, deleteRule sql.NullString
		var charMaxLength, numericPrecision, numericScale sql.NullInt64

		err := rows.Scan(&tableName, &columnName, &udtName, &isNullable, &columnDefault,
			&charMaxLength, &numericPrecision, &numericScale, &ordinalPosition, &isPrimary, &isUnique,
			&foreignTable, &foreignColumn, &deleteRule)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Initialize table if not exists
		if _, exists := tableMap[tableName]; !exists {
			tableMap[tableName] = &types.SchemaTable{
				Name:    tableName,
				Columns: []types.SchemaColumn{},
			}
			columnsSeen[tableName] = make(map[string]bool)
		}

		// Skip if column already processed for this table
		if columnsSeen[tableName][columnName] {
			continue
		}
		columnsSeen[tableName][columnName] = true

		// Format column type properly
		columnType := p.formatPullColumnType(udtName, charMaxLength, numericPrecision, numericScale, columnDefault.String, isPrimary.Valid)

		column := types.SchemaColumn{
			Name:      columnName,
			Type:      columnType,
			Nullable:  isNullable == "YES",
			Default:   p.formatDefaultValue(columnDefault.String),
			IsPrimary: isPrimary.Valid,
			IsUnique:  isUnique.Valid,
		}

		if foreignTable.Valid && foreignColumn.Valid {
			column.ForeignKeyTable = foreignTable.String
			column.ForeignKeyColumn = foreignColumn.String
			if deleteRule.Valid {
				column.OnDeleteAction = deleteRule.String
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

func (p *PostgresAdapter) formatPullColumnType(dataType string, charMaxLength, numericPrecision, numericScale sql.NullInt64, defaultValue string, isPrimary bool) string {
	switch dataType {
	case "int4", "integer":
		if isPrimary && strings.Contains(defaultValue, "nextval(") {
			return "SERIAL"
		}
		return "INT"
	case "int8", "bigint":
		if isPrimary && strings.Contains(defaultValue, "nextval(") {
			return "BIGSERIAL"
		}
		return "BIGINT"
	case "varchar", "character varying":
		if charMaxLength.Valid {
			return fmt.Sprintf("VARCHAR(%d)", charMaxLength.Int64)
		}
		return "VARCHAR(255)"
	case "bpchar", "character":
		if charMaxLength.Valid {
			return fmt.Sprintf("CHAR(%d)", charMaxLength.Int64)
		}
		return "CHAR"
	case "text":
		return "TEXT"
	case "bool", "boolean":
		return "BOOLEAN"
	case "timestamp":
		return "TIMESTAMP"
	case "timestamptz":
		return "TIMESTAMP WITH TIME ZONE"
	case "date":
		return "DATE"
	case "time":
		return "TIME"
	case "numeric":
		if numericPrecision.Valid && numericScale.Valid {
			return fmt.Sprintf("NUMERIC(%d,%d)", numericPrecision.Int64, numericScale.Int64)
		} else if numericPrecision.Valid {
			return fmt.Sprintf("NUMERIC(%d)", numericPrecision.Int64)
		}
		return "NUMERIC"
	default:
		return strings.ToUpper(dataType)
	}
}

func (p *PostgresAdapter) formatDefaultValue(defaultValue string) string {
	if defaultValue == "" {
		return ""
	}

	// Skip sequence defaults for SERIAL columns
	if strings.Contains(defaultValue, "nextval(") {
		return ""
	}

	// Format common defaults
	if strings.Contains(strings.ToLower(defaultValue), "now()") {
		return "NOW()"
	}

	return defaultValue
}
