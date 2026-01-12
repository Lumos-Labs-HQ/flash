package schema

import (
	"fmt"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

func (sm *SchemaManager) compareSchemas(current, target []types.SchemaTable, currentEnums, targetEnums []types.SchemaEnum, targetIndexes []types.SchemaIndex) *types.SchemaDiff {
	diff := &types.SchemaDiff{}
	currentMap, targetMap := sm.buildTableMaps(current, target)

	// Merge standalone indexes into target tables
	// These are CREATE INDEX statements that appear outside of CREATE TABLE
	for _, index := range targetIndexes {
		if table, exists := targetMap[index.Table]; exists {
			// Check if index isn't already in the table (avoid duplicates)
			isDuplicate := false
			for _, existingIndex := range table.Indexes {
				if existingIndex.Name == index.Name {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				table.Indexes = append(table.Indexes, index)
				targetMap[index.Table] = table
			}
		}
	}

	for _, targetTable := range targetMap {
		if currentTable, exists := currentMap[targetTable.Name]; !exists {
			diff.NewTables = append(diff.NewTables, targetTable)
		} else if tableDiff := sm.compareTablesForDiff(currentTable, targetTable); tableDiff != nil {
			diff.ModifiedTables = append(diff.ModifiedTables, *tableDiff)
		}
	}

	for _, currentTable := range current {
		if _, exists := targetMap[currentTable.Name]; !exists {
			diff.DroppedTables = append(diff.DroppedTables, currentTable.Name)
		}
	}

	sm.compareIndexes(current, sm.tableMapsToSlice(targetMap), diff)
	sm.compareEnums(currentEnums, targetEnums, diff)
	return diff
}

func (sm *SchemaManager) buildTableMaps(current, target []types.SchemaTable) (map[string]types.SchemaTable, map[string]types.SchemaTable) {
	currentMap := make(map[string]types.SchemaTable, len(current))
	targetMap := make(map[string]types.SchemaTable, len(target))

	for _, table := range current {
		currentMap[table.Name] = table
	}
	for _, table := range target {
		targetMap[table.Name] = table
	}
	return currentMap, targetMap
}

// tableMapsToSlice converts target table map back to slice for comparison
func (sm *SchemaManager) tableMapsToSlice(targetMap map[string]types.SchemaTable) []types.SchemaTable {
	var tables []types.SchemaTable
	for _, table := range targetMap {
		tables = append(tables, table)
	}
	return tables
}

func (sm *SchemaManager) compareTablesForDiff(current, target types.SchemaTable) *types.TableDiff {
	tableDiff := &types.TableDiff{Name: target.Name}
	currentCols, targetCols := sm.buildColumnMaps(current.Columns, target.Columns)
	hasChanges := false

	for _, targetCol := range target.Columns {
		if currentCol, exists := currentCols[targetCol.Name]; !exists {
			tableDiff.NewColumns = append(tableDiff.NewColumns, targetCol)
			hasChanges = true
		} else if !sm.columnsEqual(currentCol, targetCol) {
			tableDiff.ModifiedColumns = append(tableDiff.ModifiedColumns, types.ColumnDiff{
				Name:    targetCol.Name,
				OldType: currentCol.Type,
				NewType: targetCol.Type,
				Changes: sm.getColumnChanges(currentCol, targetCol),
			})
			hasChanges = true
		}
	}

	for _, currentCol := range current.Columns {
		if _, exists := targetCols[currentCol.Name]; !exists {
			// Store full column info for DOWN migration
			tableDiff.DroppedColumns = append(tableDiff.DroppedColumns, currentCol)
			hasChanges = true
		}
	}

	if hasChanges {
		return tableDiff
	}
	return nil
}

func (sm *SchemaManager) buildColumnMaps(current, target []types.SchemaColumn) (map[string]types.SchemaColumn, map[string]types.SchemaColumn) {
	currentCols := make(map[string]types.SchemaColumn, len(current))
	targetCols := make(map[string]types.SchemaColumn, len(target))

	for _, col := range current {
		currentCols[col.Name] = col
	}
	for _, col := range target {
		targetCols[col.Name] = col
	}
	return currentCols, targetCols
}

func (sm *SchemaManager) compareIndexes(current, target []types.SchemaTable, diff *types.SchemaDiff) {
	currentIndexes, targetIndexes := sm.buildIndexMaps(current, target)

	for name, index := range targetIndexes {
		if _, exists := currentIndexes[name]; !exists {
			diff.NewIndexes = append(diff.NewIndexes, index)
		}
	}

	for name := range currentIndexes {
		if _, exists := targetIndexes[name]; !exists {
			diff.DroppedIndexes = append(diff.DroppedIndexes, name)
		}
	}
}

func (sm *SchemaManager) compareEnums(current, target []types.SchemaEnum, diff *types.SchemaDiff) {
	currentMap := make(map[string]types.SchemaEnum)
	targetMap := make(map[string]types.SchemaEnum)

	for _, enum := range current {
		currentMap[enum.Name] = enum
	}
	for _, enum := range target {
		targetMap[enum.Name] = enum
	}

	// Find new enums
	for _, targetEnum := range target {
		if _, exists := currentMap[targetEnum.Name]; !exists {
			diff.NewEnums = append(diff.NewEnums, targetEnum)
		}
	}

	// Find dropped enums
	for _, currentEnum := range current {
		if _, exists := targetMap[currentEnum.Name]; !exists {
			diff.DroppedEnums = append(diff.DroppedEnums, currentEnum.Name)
		}
	}
}

func (sm *SchemaManager) buildIndexMaps(current, target []types.SchemaTable) (map[string]types.SchemaIndex, map[string]types.SchemaIndex) {
	currentIndexes := make(map[string]types.SchemaIndex)
	targetIndexes := make(map[string]types.SchemaIndex)

	for _, table := range current {
		for _, index := range table.Indexes {
			currentIndexes[index.Name] = index
		}
	}
	for _, table := range target {
		for _, index := range table.Indexes {
			targetIndexes[index.Name] = index
		}
	}
	return currentIndexes, targetIndexes
}

// Comparison helpers
func (sm *SchemaManager) columnsEqual(a, b types.SchemaColumn) bool {
	return a.Name == b.Name &&
		a.Type == b.Type &&
		a.Nullable == b.Nullable &&
		a.Default == b.Default &&
		a.IsPrimary == b.IsPrimary &&
		a.IsUnique == b.IsUnique &&
		a.ForeignKeyTable == b.ForeignKeyTable &&
		a.ForeignKeyColumn == b.ForeignKeyColumn &&
		a.OnDeleteAction == b.OnDeleteAction
}

func (sm *SchemaManager) getColumnChanges(old, new types.SchemaColumn) []string {
	var changes []string

	changeChecks := []struct {
		condition bool
		message   string
	}{
		{old.Type != new.Type, fmt.Sprintf("type changed from %s to %s", old.Type, new.Type)},
		{old.Nullable && !new.Nullable, "made not nullable"},
		{!old.Nullable && new.Nullable, "made nullable"},
		{old.Default != new.Default, fmt.Sprintf("default changed from %s to %s", old.Default, new.Default)},
		{!old.IsPrimary && new.IsPrimary, "made primary key"},
		{old.IsPrimary && !new.IsPrimary, "removed primary key"},
		{!old.IsUnique && new.IsUnique, "made unique"},
		{old.IsUnique && !new.IsUnique, "removed unique constraint"},
	}

	for _, check := range changeChecks {
		if check.condition {
			changes = append(changes, check.message)
		}
	}

	if old.ForeignKeyTable != new.ForeignKeyTable || old.ForeignKeyColumn != new.ForeignKeyColumn {
		if new.ForeignKeyTable != "" {
			changes = append(changes, fmt.Sprintf("added foreign key reference to %s(%s)", new.ForeignKeyTable, new.ForeignKeyColumn))
		} else {
			changes = append(changes, "removed foreign key reference")
		}
	}

	if old.OnDeleteAction != new.OnDeleteAction {
		changes = append(changes, fmt.Sprintf("foreign key action changed from %s to %s", old.OnDeleteAction, new.OnDeleteAction))
	}

	return changes
}
