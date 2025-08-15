package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Rana718/Graft/internal/types"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

type PostgresAdapter struct {
	pool *pgxpool.Pool
}

func NewPostgresAdapter() *PostgresAdapter {
	return &PostgresAdapter{}
}

func (p *PostgresAdapter) Connect(ctx context.Context, url string) error {
	pool, err := pgxpool.New(ctx, url)
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

func (p *PostgresAdapter) CreateMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS _graft_migrations (
			id VARCHAR(255) PRIMARY KEY,
			checksum VARCHAR(64) NOT NULL,
			finished_at TIMESTAMP WITH TIME ZONE,
			migration_name VARCHAR(255) NOT NULL,
			logs TEXT,
			rolled_back_at TIMESTAMP WITH TIME ZONE,
			started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			applied_steps_count INTEGER NOT NULL DEFAULT 0
		);`

	_, err := p.pool.Exec(ctx, query)
	return err
}

func (p *PostgresAdapter) EnsureMigrationTableCompatibility(ctx context.Context) error {
	// Check if logs column exists, add if missing
	var exists bool
	err := p.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = '_graft_migrations' 
			AND column_name = 'logs'
		)
	`).Scan(&exists)

	if err != nil {
		return err
	}

	if !exists {
		_, err = p.pool.Exec(ctx, "ALTER TABLE _graft_migrations ADD COLUMN logs TEXT")
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *PostgresAdapter) CleanupBrokenMigrationRecords(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `
		DELETE FROM _graft_migrations 
		WHERE finished_at IS NULL 
		AND started_at < NOW() - INTERVAL '1 hour'
	`)
	return err
}

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
			return nil, err
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

	// Insert migration record
	_, err = tx.Exec(ctx, `
		INSERT INTO _graft_migrations (id, migration_name, checksum, started_at)
		VALUES ($1, $2, $3, NOW())
	`, migrationID, name, checksum)
	if err != nil {
		return err
	}

	// Mark as finished
	_, err = tx.Exec(ctx, `
		UPDATE _graft_migrations 
		SET finished_at = NOW(), applied_steps_count = 1
		WHERE id = $1
	`, migrationID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (p *PostgresAdapter) ExecuteMigration(ctx context.Context, migrationSQL string) error {
	_, err := p.pool.Exec(ctx, migrationSQL)
	return err
}

func (p *PostgresAdapter) GetCurrentSchema(ctx context.Context) ([]types.SchemaTable, error) {
	tables := []types.SchemaTable{}

	rows, err := p.pool.Query(ctx, `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		AND table_name != '_graft_migrations'
		ORDER BY table_name
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

		columns, err := p.GetTableColumns(ctx, tableName)
		if err != nil {
			return nil, err
		}

		indexes, err := p.GetTableIndexes(ctx, tableName)
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

func (p *PostgresAdapter) GetTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error) {
	columns := []types.SchemaColumn{}

	rows, err := p.pool.Query(ctx, `
		SELECT 
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale,
			CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_primary_key,
			CASE WHEN uq.column_name IS NOT NULL THEN true ELSE false END as is_unique
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
		WHERE c.table_name = $1 
		AND c.table_schema = 'public'
		ORDER BY c.ordinal_position
	`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var column types.SchemaColumn
		var dataType string
		var isNullable string
		var columnDefault sql.NullString
		var charMaxLength sql.NullInt64
		var numericPrecision sql.NullInt64
		var numericScale sql.NullInt64
		var isPrimary bool
		var isUnique bool

		err := rows.Scan(
			&column.Name,
			&dataType,
			&isNullable,
			&columnDefault,
			&charMaxLength,
			&numericPrecision,
			&numericScale,
			&isPrimary,
			&isUnique,
		)
		if err != nil {
			return nil, err
		}

		column.Type = p.formatPostgresType(dataType, charMaxLength, numericPrecision, numericScale)
		column.Nullable = isNullable == "YES"
		column.IsPrimary = isPrimary
		column.IsUnique = isUnique

		if columnDefault.Valid {
			column.Default = columnDefault.String
		}

		columns = append(columns, column)
	}

	return columns, nil
}

func (p *PostgresAdapter) GetTableIndexes(ctx context.Context, tableName string) ([]types.SchemaIndex, error) {
	indexes := []types.SchemaIndex{}

	rows, err := p.pool.Query(ctx, `
		SELECT 
			indexname,
			indexdef
		FROM pg_indexes
		WHERE tablename = $1
		AND schemaname = 'public'
		AND indexname NOT LIKE '%_pkey'
	`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var indexName, indexDef string
		if err := rows.Scan(&indexName, &indexDef); err != nil {
			return nil, err
		}

		index := types.SchemaIndex{
			Name:   indexName,
			Table:  tableName,
			Unique: strings.Contains(strings.ToUpper(indexDef), "UNIQUE"),
		}

		// Extract column names from index definition
		if start := strings.Index(indexDef, "("); start != -1 {
			if end := strings.Index(indexDef[start:], ")"); end != -1 {
				columnsStr := indexDef[start+1 : start+end]
				columnNames := strings.Split(columnsStr, ",")
				for _, col := range columnNames {
					index.Columns = append(index.Columns, strings.TrimSpace(col))
				}
			}
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

func (p *PostgresAdapter) GetAllTableNames(ctx context.Context) ([]string, error) {
	var tables []string

	rows, err := p.pool.Query(ctx, `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
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

func (p *PostgresAdapter) CheckTableExists(ctx context.Context, tableName string) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = $1 AND table_schema = 'public'
		)
	`, tableName).Scan(&exists)
	return exists, err
}

func (p *PostgresAdapter) CheckColumnExists(ctx context.Context, tableName, columnName string) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = $1 AND column_name = $2 AND table_schema = 'public'
		)
	`, tableName, columnName).Scan(&exists)
	return exists, err
}

func (p *PostgresAdapter) CheckNotNullConstraint(ctx context.Context, tableName, columnName string) (bool, error) {
	var isNullable string
	err := p.pool.QueryRow(ctx, `
		SELECT is_nullable
		FROM information_schema.columns 
		WHERE table_name = $1 AND column_name = $2 AND table_schema = 'public'
	`, tableName, columnName).Scan(&isNullable)

	if err != nil {
		return false, err
	}

	return isNullable == "NO", nil
}

func (p *PostgresAdapter) CheckForeignKeyConstraint(ctx context.Context, tableName, constraintName string) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints 
			WHERE table_name = $1 
			AND constraint_name = $2 
			AND constraint_type = 'FOREIGN KEY'
			AND table_schema = 'public'
		)
	`, tableName, constraintName).Scan(&exists)
	return exists, err
}

func (p *PostgresAdapter) CheckUniqueConstraint(ctx context.Context, tableName, constraintName string) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints 
			WHERE table_name = $1 
			AND constraint_name = $2 
			AND constraint_type = 'UNIQUE'
			AND table_schema = 'public'
		)
	`, tableName, constraintName).Scan(&exists)
	return exists, err
}

func (p *PostgresAdapter) GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error) {
	rows, err := p.pool.Query(ctx, fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := rows.FieldDescriptions()
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
			row[string(col.Name)] = values[i]
		}

		result = append(result, row)
	}

	return result, nil
}

func (p *PostgresAdapter) DropTable(ctx context.Context, tableName string) error {
	_, err := p.pool.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName))
	if err != nil {
		log.Printf("Error dropping table %s: %v", tableName, err)
		return err
	}
	return nil
}

func (p *PostgresAdapter) GenerateCreateTableSQL(table types.SchemaTable) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\" (\n", table.Name))

	for i, column := range table.Columns {
		if i > 0 {
			builder.WriteString(",\n")
		}
		builder.WriteString(fmt.Sprintf("    \"%s\" %s", column.Name, p.FormatColumnType(column)))
	}

	builder.WriteString("\n);")
	return builder.String()
}

func (p *PostgresAdapter) GenerateAddColumnSQL(tableName string, column types.SchemaColumn) string {
	return fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN IF NOT EXISTS \"%s\" %s;",
		tableName, column.Name, p.FormatColumnType(column))
}

func (p *PostgresAdapter) GenerateDropColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE \"%s\" DROP COLUMN IF EXISTS \"%s\";",
		tableName, columnName)
}

func (p *PostgresAdapter) GenerateAddIndexSQL(index types.SchemaIndex) string {
	uniqueStr := ""
	if index.Unique {
		uniqueStr = "UNIQUE "
	}

	columnsStr := strings.Join(index.Columns, ", ")
	return fmt.Sprintf("CREATE %sINDEX \"%s\" ON \"%s\" (%s);",
		uniqueStr, index.Name, index.Table, columnsStr)
}

func (p *PostgresAdapter) GenerateDropIndexSQL(indexName string) string {
	return fmt.Sprintf("DROP INDEX IF EXISTS \"%s\";", indexName)
}

func (p *PostgresAdapter) MapColumnType(dbType string) string {
	switch strings.ToLower(dbType) {
	case "character varying", "varchar":
		return "VARCHAR"
	case "character", "char":
		return "CHAR"
	case "text":
		return "TEXT"
	case "integer", "int4":
		return "INTEGER"
	case "bigint", "int8":
		return "BIGINT"
	case "smallint", "int2":
		return "SMALLINT"
	case "boolean", "bool":
		return "BOOLEAN"
	case "timestamp with time zone", "timestamptz":
		return "TIMESTAMP WITH TIME ZONE"
	case "timestamp without time zone", "timestamp":
		return "TIMESTAMP"
	case "date":
		return "DATE"
	case "time":
		return "TIME"
	case "numeric", "decimal":
		return "NUMERIC"
	case "real", "float4":
		return "REAL"
	case "double precision", "float8":
		return "DOUBLE PRECISION"
	case "uuid":
		return "UUID"
	case "json":
		return "JSON"
	case "jsonb":
		return "JSONB"
	default:
		return strings.ToUpper(dbType)
	}
}

func (p *PostgresAdapter) FormatColumnType(column types.SchemaColumn) string {
	var parts []string

	// Add the base type
	parts = append(parts, column.Type)

	// Add PRIMARY KEY constraint
	if column.IsPrimary {
		parts = append(parts, "PRIMARY KEY")
	}

	// Add UNIQUE constraint (only if not primary key)
	if column.IsUnique && !column.IsPrimary {
		parts = append(parts, "UNIQUE")
	}

	// Add NOT NULL constraint (only if not primary key, as PRIMARY KEY implies NOT NULL)
	if !column.Nullable && !column.IsPrimary {
		parts = append(parts, "NOT NULL")
	}

	// Add foreign key reference
	if column.ForeignKeyTable != "" && column.ForeignKeyColumn != "" {
		parts = append(parts, fmt.Sprintf("REFERENCES \"%s\"(\"%s\")", column.ForeignKeyTable, column.ForeignKeyColumn))
		if column.OnDeleteAction != "" {
			parts = append(parts, fmt.Sprintf("ON DELETE %s", column.OnDeleteAction))
		}
	}

	// Add default value
	if column.Default != "" && !strings.Contains(column.Default, "nextval") {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", column.Default))
	}

	return strings.Join(parts, " ")
}

func (p *PostgresAdapter) formatPostgresType(dataType string, charMaxLength, numericPrecision, numericScale sql.NullInt64) string {
	switch dataType {
	case "character varying":
		if charMaxLength.Valid {
			return fmt.Sprintf("VARCHAR(%d)", charMaxLength.Int64)
		}
		return "VARCHAR"
	case "character":
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
	default:
		return p.MapColumnType(dataType)
	}
}
