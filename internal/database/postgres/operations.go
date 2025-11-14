package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func (p *Adapter) tableExists(tableName string) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = $1 AND table_schema = 'public'
		)
	`, tableName).Scan(&exists)
	return exists, err
}

func (p *Adapter) columnExists(tableName, columnName string) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = $1 AND column_name = $2
		)
	`, tableName, columnName).Scan(&exists)
	return exists, err
}

func (p *Adapter) constraintExists(tableName, constraintName, constraintType string) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints 
			WHERE table_name = $1 AND constraint_name = $2 AND constraint_type = $3 AND table_schema = 'public'
		)
	`, tableName, constraintName, constraintType).Scan(&exists)
	return exists, err
}

func (p *Adapter) CheckTableExists(ctx context.Context, tableName string) (bool, error) {
	return p.tableExists(tableName)
}

func (p *Adapter) CheckColumnExists(ctx context.Context, tableName, columnName string) (bool, error) {
	return p.columnExists(tableName, columnName)
}

func (p *Adapter) CheckNotNullConstraint(ctx context.Context, tableName, columnName string) (bool, error) {
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

func (p *Adapter) CheckForeignKeyConstraint(ctx context.Context, tableName, constraintName string) (bool, error) {
	return p.constraintExists(tableName, constraintName, "FOREIGN KEY")
}

func (p *Adapter) CheckUniqueConstraint(ctx context.Context, tableName, constraintName string) (bool, error) {
	return p.constraintExists(tableName, constraintName, "UNIQUE")
}

func (p *Adapter) GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT column_name, udt_name 
		FROM information_schema.columns 
		WHERE table_name = $1 AND table_schema = 'public'
		ORDER BY ordinal_position`

	columnRows, err := p.pool.Query(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get column info: %w", err)
	}

	var selectCols []string
	for columnRows.Next() {
		var colName, udtName string
		if err := columnRows.Scan(&colName, &udtName); err != nil {
			columnRows.Close()
			return nil, err
		}

		if !isStandardPostgresType(udtName) {
			selectCols = append(selectCols, fmt.Sprintf(`"%s"::text`, colName))
		} else {
			selectCols = append(selectCols, fmt.Sprintf(`"%s"`, colName))
		}
	}
	columnRows.Close()

	if len(selectCols) == 0 {
		return []map[string]interface{}{}, nil
	}

	selectQuery := fmt.Sprintf("SELECT %s FROM \"%s\"", strings.Join(selectCols, ", "), tableName)
	rows, err := p.pool.Query(ctx, selectQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query table %s: %w", tableName, err)
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

	return result, rows.Err()
}

func isStandardPostgresType(udtName string) bool {
	standardTypes := map[string]bool{
		"int2": true, "int4": true, "int8": true,
		"smallint": true, "integer": true, "bigint": true,
		"float4": true, "float8": true, "real": true, "double precision": true,
		"numeric": true, "decimal": true,
		"varchar": true, "char": true, "text": true, "bpchar": true,
		"bool": true, "boolean": true,
		"date": true, "time": true, "timetz": true,
		"timestamp": true, "timestamptz": true, "interval": true,
		"uuid": true, "json": true, "jsonb": true, "bytea": true,
		"xml": true, "money": true,
		"point": true, "line": true, "lseg": true, "box": true,
		"path": true, "polygon": true, "circle": true,
		"inet": true, "cidr": true, "macaddr": true,
		"bit": true, "varbit": true,
		"tsvector": true, "tsquery": true,
	}
	return standardTypes[strings.ToLower(udtName)]
}

func (p *Adapter) GetTableRowCount(ctx context.Context, tableName string) (int, error) {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", tableName)
	err := p.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count rows in table %s: %w", tableName, err)
	}
	return count, nil
}

func (p *Adapter) GetAllTableRowCounts(ctx context.Context, tableNames []string) (map[string]int, error) {
	if len(tableNames) == 0 {
		return make(map[string]int), nil
	}

	var queryParts []string
	for _, tableName := range tableNames {
		queryParts = append(queryParts, fmt.Sprintf("SELECT '%s' as table_name, COUNT(*) as row_count FROM \"%s\"", tableName, tableName))
	}

	query := strings.Join(queryParts, " UNION ALL ")
	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to batch count table rows: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int, len(tableNames))
	for rows.Next() {
		var tableName string
		var count int
		if err := rows.Scan(&tableName, &count); err != nil {
			return nil, fmt.Errorf("failed to scan batch count result: %w", err)
		}
		result[tableName] = count
	}

	return result, nil
}

func (p *Adapter) DropTable(ctx context.Context, tableName string) error {
	_, err := p.pool.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName))
	return err
}

func (p *Adapter) DropEnum(ctx context.Context, enumName string) error {
	_, err := p.pool.Exec(ctx, fmt.Sprintf("DROP TYPE IF EXISTS %s CASCADE", enumName))
	return err
}

func (p *Adapter) GenerateCreateTableSQL(table types.SchemaTable) string {
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

func (p *Adapter) GenerateAddColumnSQL(tableName string, column types.SchemaColumn) string {
	return fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN IF NOT EXISTS \"%s\" %s;",
		tableName, column.Name, p.FormatColumnType(column))
}

func (p *Adapter) GenerateDropColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE \"%s\" DROP COLUMN IF EXISTS \"%s\";", tableName, columnName)
}

func (p *Adapter) GenerateAddIndexSQL(index types.SchemaIndex) string {
	unique := ""
	if index.Unique {
		unique = "UNIQUE "
	}
	columns := strings.Join(index.Columns, ", ")
	return fmt.Sprintf("CREATE %sINDEX \"%s\" ON \"%s\" (%s);", unique, index.Name, index.Table, columns)
}

func (p *Adapter) GenerateDropIndexSQL(indexName string) string {
	return fmt.Sprintf("DROP INDEX IF EXISTS \"%s\";", indexName)
}

func (p *Adapter) FormatColumnType(column types.SchemaColumn) string {
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
