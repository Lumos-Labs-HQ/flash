package migrator

import (
	"strings"

	"github.com/Rana718/Graft/internal/types"
)

// Checks if table changes contain meaningful modifications
func hasRealTableChanges(modifiedTables []types.TableDiff) bool {
	for _, tableDiff := range modifiedTables {
		if len(tableDiff.NewColumns) > 0 || len(tableDiff.DroppedColumns) > 0 {
			return true
		}
		for _, modCol := range tableDiff.ModifiedColumns {
			if len(modCol.Changes) > 0 {
				return true
			}
		}
	}
	return false
}

// Checks if a column is a primary key
func isPrimaryKeyColumn(col types.SchemaColumn) bool {
	return strings.Contains(strings.ToUpper(col.Type), "PRIMARY KEY") ||
		strings.Contains(strings.ToUpper(col.Type), "SERIAL")
}

// Checks if two types are equivalent
func isEquivalentType(currentType, targetType string) bool {
	current := normalizeTypeForComparison(currentType)
	target := normalizeTypeForComparison(targetType)
	return current == target
}

// Checks if two defaults are equivalent
func isEquivalentDefault(currentDefault, targetDefault string) bool {
	current := strings.TrimSpace(currentDefault)
	target := strings.TrimSpace(targetDefault)

	if current == "" && target == "" {
		return true
	}

	currentUpper := strings.ToUpper(current)
	targetUpper := strings.ToUpper(target)

	nowVariations := []string{"NOW()", "CURRENT_TIMESTAMP", "CURRENT_TIMESTAMP()"}

	currentIsNow := false
	targetIsNow := false

	for _, variation := range nowVariations {
		if strings.Contains(currentUpper, variation) {
			currentIsNow = true
		}
		if strings.Contains(targetUpper, variation) {
			targetIsNow = true
		}
	}

	if currentIsNow && targetIsNow {
		return true
	}

	return current == target
}

// Normalizes PostgreSQL type for comparison
func normalizeTypeForComparison(pgType string) string {
	cleaned := pgType
	cleaned = strings.ReplaceAll(cleaned, "PRIMARY KEY", "")
	cleaned = strings.ReplaceAll(cleaned, "UNIQUE", "")
	cleaned = strings.ReplaceAll(cleaned, "NOT NULL", "")

	if idx := strings.Index(strings.ToUpper(cleaned), "DEFAULT"); idx != -1 {
		cleaned = cleaned[:idx]
	}

	cleaned = strings.TrimSpace(cleaned)
	normalized := strings.ToUpper(cleaned)

	if strings.Contains(normalized, "SERIAL") {
		if strings.Contains(normalized, "BIGSERIAL") {
			return "INTEGER"
		}
		return "INTEGER"
	}

	if strings.Contains(normalized, "TIMESTAMP WITH TIME ZONE") {
		return "TIMESTAMP WITH TIME ZONE"
	}
	if strings.Contains(normalized, "TIMESTAMP WITHOUT TIME ZONE") || normalized == "TIMESTAMP" {
		return "TIMESTAMP WITHOUT TIME ZONE"
	}

	if strings.HasPrefix(normalized, "VARCHAR") || strings.HasPrefix(normalized, "CHARACTER VARYING") {
		return "VARCHAR"
	}

	if normalized == "TEXT" {
		return "TEXT"
	}

	if normalized == "INTEGER" || normalized == "INT" || normalized == "INT4" {
		return "INTEGER"
	}

	if normalized == "BIGINT" || normalized == "INT8" {
		return "BIGINT"
	}

	return normalized
}

// Extracts data type from column definition
func extractDataType(columnType string) string {
	typeStr := columnType
	typeStr = strings.ReplaceAll(typeStr, "PRIMARY KEY", "")
	typeStr = strings.ReplaceAll(typeStr, "UNIQUE", "")
	typeStr = strings.ReplaceAll(typeStr, "NOT NULL", "")

	if idx := strings.Index(strings.ToUpper(typeStr), "DEFAULT"); idx != -1 {
		typeStr = typeStr[:idx]
	}

	upperType := strings.ToUpper(strings.TrimSpace(typeStr))
	if strings.Contains(upperType, "SERIAL") {
		if strings.Contains(upperType, "BIGSERIAL") {
			return "BIGINT"
		}
		return "INTEGER"
	}

	return strings.TrimSpace(typeStr)
}

// Parses column definition from schema line
func ParseColumnDefinition(line string) types.SchemaColumn {
	line = strings.TrimSpace(strings.TrimSuffix(line, ","))
	if line == "" || strings.HasPrefix(strings.ToUpper(line), "PRIMARY KEY") ||
		strings.HasPrefix(strings.ToUpper(line), "FOREIGN KEY") ||
		strings.HasPrefix(strings.ToUpper(line), "CONSTRAINT") {
		return types.SchemaColumn{}
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return types.SchemaColumn{}
	}

	col := types.SchemaColumn{
		Name:     parts[0],
		Nullable: true,
	}

	upperLine := strings.ToUpper(line)

	if strings.Contains(upperLine, "SERIAL") {
		if strings.Contains(upperLine, "PRIMARY KEY") {
			col.Type = "SERIAL PRIMARY KEY"
		} else {
			col.Type = "SERIAL"
		}
	} else {
		typeStart := 1
		col.Type = parts[typeStart]

		if typeStart+1 < len(parts) && strings.HasPrefix(parts[typeStart+1], "(") {
			col.Type += parts[typeStart+1]
		}

		if strings.Contains(upperLine, "PRIMARY KEY") {
			col.Type += " PRIMARY KEY"
		}
		if strings.Contains(upperLine, "UNIQUE") && !strings.Contains(upperLine, "PRIMARY KEY") {
			col.Type += " UNIQUE"
		}
	}

	if strings.Contains(upperLine, "NOT NULL") {
		col.Nullable = false
	}

	if strings.Contains(upperLine, "DEFAULT") {
		defaultIdx := strings.Index(upperLine, "DEFAULT")
		remaining := line[defaultIdx+7:]
		parts := strings.Fields(remaining)
		if len(parts) > 0 {
			col.Default = parts[0]
			if len(parts) > 1 && parts[1] == "()" {
				col.Default += "()"
			}
		}
	}

	return col
}

// Compares columns between tables
func compareTableColumns(current, target types.SchemaTable) types.TableDiff {
	diff := types.TableDiff{
		Name:            target.Name,
		NewColumns:      []types.SchemaColumn{},
		DroppedColumns:  []string{},
		ModifiedColumns: []types.ColumnDiff{},
	}

	currentCols := make(map[string]types.SchemaColumn)
	for _, col := range current.Columns {
		currentCols[col.Name] = col
	}

	targetCols := make(map[string]types.SchemaColumn)
	for _, col := range target.Columns {
		targetCols[col.Name] = col
	}

	for _, targetCol := range target.Columns {
		if _, exists := currentCols[targetCol.Name]; !exists {
			diff.NewColumns = append(diff.NewColumns, targetCol)
		}
	}

	for _, currentCol := range current.Columns {
		if _, exists := targetCols[currentCol.Name]; !exists {
			diff.DroppedColumns = append(diff.DroppedColumns, currentCol.Name)
		}
	}

	for colName, targetCol := range targetCols {
		if currentCol, exists := currentCols[colName]; exists {
			var changes []string

			if !isEquivalentType(currentCol.Type, targetCol.Type) {
				if !strings.Contains(strings.ToUpper(targetCol.Type), "SERIAL") {
					targetType := extractDataType(targetCol.Type)
					changes = append(changes, "ALTER TABLE \""+target.Name+"\" ALTER COLUMN \""+colName+"\" TYPE "+targetType+";")
				}
			}

			if !isPrimaryKeyColumn(targetCol) && currentCol.Nullable != targetCol.Nullable {
				if targetCol.Nullable {
					changes = append(changes, "ALTER TABLE \""+target.Name+"\" ALTER COLUMN \""+colName+"\" DROP NOT NULL;")
				} else {
					changes = append(changes, "ALTER TABLE \""+target.Name+"\" ALTER COLUMN \""+colName+"\" SET NOT NULL;")
				}
			}

			if !strings.Contains(strings.ToUpper(targetCol.Type), "SERIAL") && !isEquivalentDefault(currentCol.Default, targetCol.Default) {
				if targetCol.Default != "" {
					changes = append(changes, "ALTER TABLE \""+target.Name+"\" ALTER COLUMN \""+colName+"\" SET DEFAULT "+targetCol.Default+";")
				} else {
					changes = append(changes, "ALTER TABLE \""+target.Name+"\" ALTER COLUMN \""+colName+"\" DROP DEFAULT;")
				}
			}

			if len(changes) > 0 {
				diff.ModifiedColumns = append(diff.ModifiedColumns, types.ColumnDiff{
					Name:    colName,
					OldType: currentCol.Type,
					NewType: targetCol.Type,
					Changes: changes,
				})
			}
		}
	}

	return diff
}
