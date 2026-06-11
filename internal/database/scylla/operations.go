package scylla

import (
	"context"
	"fmt"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func (a *Adapter) GenerateCreateTableSQL(table types.SchemaTable) string {
	ks := a.currentKeyspace()
	var lines []string
	var pk []string

	for _, col := range table.Columns {
		if col.IsPrimary {
			pk = append(pk, `"`+col.Name+`"`)
		}
	}

	lines = append(lines, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s"."%s" (`, ks, table.Name))
	for i, col := range table.Columns {
		comma := ","
		if i == len(table.Columns)-1 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf(`    "%s" %s%s`, col.Name, a.FormatColumnType(col), comma))
	}

	if len(pk) > 1 {
		pkStr := strings.Join(pk, ", ")
		lines = append(lines, fmt.Sprintf(`    PRIMARY KEY (%s)`, pkStr))
	} else if len(pk) == 1 {
		lines = append(lines, fmt.Sprintf(`    PRIMARY KEY (%s)`, pk[0]))
	} else if len(table.Columns) > 0 {
		lines = append(lines, fmt.Sprintf(`    PRIMARY KEY ("%s")`, table.Columns[0].Name))
	}

	lines = append(lines, ");")
	return strings.Join(lines, "\n")
}

func (a *Adapter) GenerateAddColumnSQL(tableName string, column types.SchemaColumn) string {
	return fmt.Sprintf(`ALTER TABLE "%s"."%s" ADD "%s" %s;`,
		a.currentKeyspace(), tableName, column.Name, a.FormatColumnType(column))
}

func (a *Adapter) GenerateDropColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf(`ALTER TABLE "%s"."%s" DROP "%s";`,
		a.currentKeyspace(), tableName, columnName)
}

func (a *Adapter) GenerateAlterColumnSQL(tableName string, column types.SchemaColumn, oldType string) string {
	if column.Type == oldType {
		return ""
	}
	return fmt.Sprintf(`ALTER TABLE "%s"."%s" ALTER "%s" TYPE %s;`,
		a.currentKeyspace(), tableName, column.Name, a.FormatColumnType(column))
}

func (a *Adapter) GenerateAddIndexSQL(index types.SchemaIndex) string {
	cols := strings.Join(index.Columns, `", "`)
	return fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "%s" ON "%s"."%s" ("%s");`,
		index.Name, a.currentKeyspace(), index.Table, cols)
}

func (a *Adapter) GenerateDropIndexSQL(index types.SchemaIndex) string {
	return fmt.Sprintf(`DROP INDEX IF EXISTS "%s"."%s";`, a.currentKeyspace(), index.Name)
}

func (a *Adapter) FormatColumnType(column types.SchemaColumn) string {
	return column.Type
}

func (a *Adapter) CheckTableExists(ctx context.Context, tableName string) (bool, error) {
	ks := a.currentKeyspace()
	var count int
	err := a.session.Query(
		`SELECT count(*) FROM system_schema.tables WHERE keyspace_name = ? AND table_name = ? ALLOW FILTERING`,
		ks, tableName,
	).WithContext(ctx).Scan(&count)
	return count > 0, err
}

func (a *Adapter) CheckColumnExists(ctx context.Context, tableName, columnName string) (bool, error) {
	ks := a.currentKeyspace()
	var count int
	err := a.session.Query(
		`SELECT count(*) FROM system_schema.columns WHERE keyspace_name = ? AND table_name = ? AND column_name = ? ALLOW FILTERING`,
		ks, tableName, columnName,
	).WithContext(ctx).Scan(&count)
	return count > 0, err
}

func (a *Adapter) CheckNotNullConstraint(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (a *Adapter) CheckForeignKeyConstraint(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (a *Adapter) CheckUniqueConstraint(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (a *Adapter) GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error) {
	ks := a.currentKeyspace()
	iter := a.session.Query(fmt.Sprintf(`SELECT * FROM "%s"."%s"`, ks, tableName)).WithContext(ctx).Iter()
	defer iter.Close()

	cols := iter.Columns()
	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = c.Name
	}

	var result []map[string]interface{}
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
		result = append(result, m)
	}
	return result, iter.Close()
}

func (a *Adapter) GetTableRowCount(ctx context.Context, tableName string) (int, error) {
	ks := a.currentKeyspace()
	var count int
	err := a.session.Query(fmt.Sprintf(`SELECT count(*) FROM "%s"."%s"`, ks, tableName)).WithContext(ctx).Scan(&count)
	return count, err
}

func (a *Adapter) GetAllTableRowCounts(ctx context.Context, tableNames []string) (map[string]int, error) {
	result := make(map[string]int, len(tableNames))
	for _, t := range tableNames {
		count, err := a.GetTableRowCount(ctx, t)
		if err != nil {
			result[t] = 0
			continue
		}
		result[t] = count
	}
	return result, nil
}

func (a *Adapter) DropTable(ctx context.Context, tableName string) error {
	ks := a.currentKeyspace()
	return a.session.Query(fmt.Sprintf(`DROP TABLE IF EXISTS "%s"."%s"`, ks, tableName)).WithContext(ctx).Exec()
}

func (a *Adapter) DropEnum(_ context.Context, _ string) error { return nil }
