package pull

import (
	"fmt"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func (s *Service) generateTableSQLClean(table types.SchemaTable, indexes []types.SchemaIndex) string {
	var sb strings.Builder
	provider := s.config.Database.Provider

	// Collect FK constraint lines (deferred after columns)
	type fkEntry struct {
		col types.SchemaColumn
	}
	var fkCols []fkEntry
	for _, col := range table.Columns {
		if col.ForeignKeyTable != "" && col.ForeignKeyColumn != "" {
			fkCols = append(fkCols, fkEntry{col})
		}
	}

	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", table.Name))

	totalLines := len(table.Columns) + len(fkCols)
	for i, col := range table.Columns {
		needComma := i < totalLines-1
		sb.WriteString(fmt.Sprintf("    %s %s", col.Name, col.Type))

		if col.IsIdentity {
			sb.WriteString(" GENERATED ALWAYS AS IDENTITY PRIMARY KEY")
		} else {
			if col.IsPrimary {
				sb.WriteString(" PRIMARY KEY")
			}
			// AUTOINCREMENT is SQLite-only; PostgreSQL/MySQL use SERIAL/AUTO_INCREMENT in the type
			if col.IsAutoIncrement && !col.IsPrimary && (provider == "sqlite" || provider == "sqlite3") {
				sb.WriteString(" AUTOINCREMENT")
			}
			if !col.Nullable && !col.IsPrimary {
				sb.WriteString(" NOT NULL")
			}
			if col.IsUnique && !col.IsPrimary {
				sb.WriteString(" UNIQUE")
			}
			if col.Default != "" {
				sb.WriteString(fmt.Sprintf(" DEFAULT %s", col.Default))
			}
			if col.Generated != "" {
				sb.WriteString(fmt.Sprintf(" GENERATED ALWAYS AS (%s) STORED", col.Generated))
			}
			if col.Check != "" {
				sb.WriteString(fmt.Sprintf(" CHECK (%s)", col.Check))
			}
		}

		if needComma {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	for i, fk := range fkCols {
		col := fk.col
		line := fmt.Sprintf("    FOREIGN KEY (%s) REFERENCES %s(%s)",
			col.Name, col.ForeignKeyTable, col.ForeignKeyColumn)
		if col.OnDeleteAction != "" && col.OnDeleteAction != "NO ACTION" {
			line += fmt.Sprintf(" ON DELETE %s", col.OnDeleteAction)
		}
		if col.OnUpdateAction != "" && col.OnUpdateAction != "NO ACTION" {
			line += fmt.Sprintf(" ON UPDATE %s", col.OnUpdateAction)
		}
		if i < len(fkCols)-1 {
			line += ","
		}
		sb.WriteString(line + "\n")
	}

	sb.WriteString(");")

	for _, idx := range indexes {
		if strings.HasSuffix(idx.Name, "_pkey") || idx.Name == "PRIMARY" || strings.HasPrefix(idx.Name, "sqlite_") {
			continue
		}
		unique := ""
		if idx.Unique {
			unique = "UNIQUE "
		}
		method := ""
		if idx.Method != "" && idx.Method != "btree" {
			method = fmt.Sprintf(" USING %s", strings.ToUpper(idx.Method))
		}
		idxSQL := fmt.Sprintf("\nCREATE %sINDEX %s ON %s%s (%s)",
			unique, idx.Name, table.Name, method, strings.Join(idx.Columns, ", "))
		if idx.Where != "" {
			idxSQL += fmt.Sprintf(" WHERE %s", idx.Where)
		}
		sb.WriteString(idxSQL + ";")
	}

	return sb.String()
}

func (s *Service) generateTableSQL(table types.SchemaTable, indexes []types.SchemaIndex) string {
	return s.generateTableSQLClean(table, indexes) + "\n"
}

func (s *Service) generateEnumSQL(enums []types.SchemaEnum) string {
	if len(enums) == 0 {
		return ""
	}
	var parts []string
	for _, enum := range enums {
		values := make([]string, len(enum.Values))
		for i, v := range enum.Values {
			escaped := strings.ReplaceAll(v, "'", "''")
			values[i] = fmt.Sprintf("'%s'", escaped)
		}
		parts = append(parts, fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);", enum.Name, strings.Join(values, ", ")))
	}
	return strings.Join(parts, "\n")
}
