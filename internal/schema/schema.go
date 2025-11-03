package schema

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/Rana718/Graft/internal/database"
	"github.com/Rana718/Graft/internal/types"
)

type foreignKeyConstraint struct {
	ColumnName, ReferencedTable, ReferencedColumn, OnDeleteAction string
}

type SchemaManager struct {
	adapter database.DatabaseAdapter
}

func NewSchemaManager(adapter database.DatabaseAdapter) *SchemaManager {
	return &SchemaManager{adapter: adapter}
}

func (sm *SchemaManager) ParseSchemaFile(schemaPath string) ([]types.SchemaTable, error) {
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}
	tables, _, _ := sm.parseSchemaContent(string(content))
	return tables, nil
}

func (sm *SchemaManager) ParseSchemaFileWithEnums(schemaPath string) ([]types.SchemaTable, []types.SchemaEnum, error) {
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read schema file: %w", err)
	}
	return sm.parseSchemaContent(string(content))
}

func (sm *SchemaManager) parseSchemaContent(content string) ([]types.SchemaTable, []types.SchemaEnum, error) {
	var tables []types.SchemaTable
	var enums []types.SchemaEnum
	statements := sm.splitStatements(sm.cleanSQL(content))

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if sm.isCreateTypeStatement(stmt) {
			if enum, err := sm.parseCreateTypeStatement(stmt); err == nil {
				enums = append(enums, enum)
			}
		} else if sm.isCreateTableStatement(stmt) {
			if table, err := sm.parseCreateTableStatement(stmt); err == nil {
				tables = append(tables, table)
			}
		}
	}
	return tables, enums, nil
}

func (sm *SchemaManager) GenerateSchemaDiff(ctx context.Context, targetSchemaPath string) (*types.SchemaDiff, error) {
	currentTables, err := sm.adapter.GetCurrentSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current schema: %w", err)
	}

	targetTables, targetEnums, err := sm.ParseSchemaFileWithEnums(targetSchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target schema: %w", err)
	}

	// Get current enums from database
	currentEnums, err := sm.adapter.GetCurrentEnums(ctx)
	if err != nil {
		// If the adapter doesn't support enums, just continue with empty list
		currentEnums = []types.SchemaEnum{}
	}

	return sm.compareSchemas(currentTables, targetTables, currentEnums, targetEnums), nil
}

func (sm *SchemaManager) GenerateSchemaSQL(tables []types.SchemaTable) string {
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })

	var parts []string
	for _, table := range tables {
		parts = append(parts, sm.adapter.GenerateCreateTableSQL(table))
		for _, index := range table.Indexes {
			parts = append(parts, sm.adapter.GenerateAddIndexSQL(index))
		}
	}
	return strings.Join(parts, "\n\n")
}

func (sm *SchemaManager) GenerateMigrationSQL(diff *types.SchemaDiff) string {
	var parts []string

	// Drop enums that are no longer needed (must be done before dropping tables)
	for _, enumName := range diff.DroppedEnums {
		parts = append(parts, fmt.Sprintf("DROP TYPE IF EXISTS \"%s\";", enumName))
	}

	for _, tableName := range diff.DroppedTables {
		parts = append(parts, fmt.Sprintf("DROP TABLE IF EXISTS \"%s\";", tableName))
	}

	// Create new enums (must be done before creating tables that use them)
	for _, enum := range diff.NewEnums {
		values := make([]string, len(enum.Values))
		for i, v := range enum.Values {
			values[i] = fmt.Sprintf("'%s'", v)
		}
		parts = append(parts, fmt.Sprintf("CREATE TYPE \"%s\" AS ENUM (%s);", enum.Name, strings.Join(values, ", ")))
	}

	for _, table := range diff.NewTables {
		parts = append(parts, sm.adapter.GenerateCreateTableSQL(table))
		for _, index := range table.Indexes {
			parts = append(parts, sm.adapter.GenerateAddIndexSQL(index))
		}
	}

	for _, tableDiff := range diff.ModifiedTables {
		for _, column := range tableDiff.NewColumns {
			parts = append(parts, sm.adapter.GenerateAddColumnSQL(tableDiff.Name, column))
		}
		for _, columnName := range tableDiff.DroppedColumns {
			parts = append(parts, sm.adapter.GenerateDropColumnSQL(tableDiff.Name, columnName))
		}
	}

	for _, indexName := range diff.DroppedIndexes {
		parts = append(parts, sm.adapter.GenerateDropIndexSQL(indexName))
	}
	for _, index := range diff.NewIndexes {
		parts = append(parts, sm.adapter.GenerateAddIndexSQL(index))
	}

	return strings.Join(parts, "\n\n")
}

func (sm *SchemaManager) compareSchemas(current, target []types.SchemaTable, currentEnums, targetEnums []types.SchemaEnum) *types.SchemaDiff {
	diff := &types.SchemaDiff{}
	currentMap, targetMap := sm.buildTableMaps(current, target)

	for _, targetTable := range target {
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

	sm.compareIndexes(current, target, diff)
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
			tableDiff.DroppedColumns = append(tableDiff.DroppedColumns, currentCol.Name)
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

// SQL Parsing helpers
func (sm *SchemaManager) cleanSQL(sql string) string {
	commentRegex := regexp.MustCompile(`--.*|/\*[\s\S]*?\*/`)
	sql = commentRegex.ReplaceAllString(sql, "")

	whitespaceRegex := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(whitespaceRegex.ReplaceAllString(sql, " "))
}

func (sm *SchemaManager) splitStatements(sql string) []string {
	statements := strings.Split(sql, ";")
	result := make([]string, 0, len(statements))

	for _, stmt := range statements {
		if stmt = strings.TrimSpace(stmt); stmt != "" {
			result = append(result, stmt)
		}
	}
	return result
}

func (sm *SchemaManager) isCreateTableStatement(stmt string) bool {
	matched, _ := regexp.MatchString(`(?i)^\s*CREATE\s+TABLE`, stmt)
	return matched
}

func (sm *SchemaManager) isCreateTypeStatement(stmt string) bool {
	matched, _ := regexp.MatchString(`(?i)^\s*CREATE\s+TYPE\s+\w+\s+AS\s+ENUM`, stmt)
	return matched
}

func (sm *SchemaManager) parseCreateTypeStatement(stmt string) (types.SchemaEnum, error) {
	// Match: CREATE TYPE enum_name AS ENUM ('value1', 'value2', ...)
	enumRegex := regexp.MustCompile(`(?i)CREATE\s+TYPE\s+(?:"?(\w+)"?|(\w+))\s+AS\s+ENUM\s*\(\s*([^)]+)\s*\)`)
	matches := enumRegex.FindStringSubmatch(stmt)

	if len(matches) < 4 {
		return types.SchemaEnum{}, fmt.Errorf("could not parse CREATE TYPE statement: %s", stmt)
	}

	// Extract enum name
	enumName := matches[1]
	if enumName == "" {
		enumName = matches[2]
	}

	// Extract values
	valuesStr := matches[3]
	valueRegex := regexp.MustCompile(`'([^']+)'`)
	valueMatches := valueRegex.FindAllStringSubmatch(valuesStr, -1)

	values := make([]string, 0, len(valueMatches))
	for _, match := range valueMatches {
		if len(match) > 1 {
			values = append(values, match[1])
		}
	}

	return types.SchemaEnum{
		Name:   enumName,
		Values: values,
	}, nil
}

func (sm *SchemaManager) parseCreateTableStatement(stmt string) (types.SchemaTable, error) {
	tableRegex := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:"?(\w+)"?|(\w+)|\x60(\w+)\x60)\s*\(`)
	matches := tableRegex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return types.SchemaTable{}, fmt.Errorf("could not extract table name from: %s", stmt)
	}

	tableName := sm.extractTableName(matches)
	if tableName == "" {
		return types.SchemaTable{}, fmt.Errorf("could not extract table name")
	}

	start, end := strings.Index(stmt, "("), strings.LastIndex(stmt, ")")
	if start == -1 || end == -1 {
		return types.SchemaTable{}, fmt.Errorf("invalid CREATE TABLE syntax")
	}

	columns, foreignKeys, err := sm.parseColumnDefinitionsAndConstraints(stmt[start+1 : end])
	if err != nil {
		return types.SchemaTable{}, err
	}

	sm.applyForeignKeys(columns, foreignKeys)

	return types.SchemaTable{
		Name:    tableName,
		Columns: columns,
		Indexes: []types.SchemaIndex{},
	}, nil
}

func (sm *SchemaManager) extractTableName(matches []string) string {
	for i := 1; i < len(matches); i++ {
		if matches[i] != "" {
			return matches[i]
		}
	}
	return ""
}

func (sm *SchemaManager) applyForeignKeys(columns []types.SchemaColumn, foreignKeys []foreignKeyConstraint) {
	for _, fk := range foreignKeys {
		for i := range columns {
			if columns[i].Name == fk.ColumnName {
				columns[i].ForeignKeyTable = fk.ReferencedTable
				columns[i].ForeignKeyColumn = fk.ReferencedColumn
				columns[i].OnDeleteAction = fk.OnDeleteAction
				break
			}
		}
	}
}

func (sm *SchemaManager) parseColumnDefinitionsAndConstraints(columnDefs string) ([]types.SchemaColumn, []foreignKeyConstraint, error) {
	var columns []types.SchemaColumn
	var foreignKeys []foreignKeyConstraint

	for _, colDef := range sm.splitColumnDefinitions(columnDefs) {
		if colDef = strings.TrimSpace(colDef); colDef == "" {
			continue
		}

		if sm.isTableConstraint(colDef) {
			if fk := sm.parseForeignKeyConstraint(colDef); fk != nil {
				foreignKeys = append(foreignKeys, *fk)
			}
			continue
		}

		column, err := sm.parseColumnDefinition(colDef)
		if err != nil {
			return nil, nil, err
		}
		columns = append(columns, column)
	}

	return columns, foreignKeys, nil
}

func (sm *SchemaManager) parseForeignKeyConstraint(constraint string) *foreignKeyConstraint {
	fkRegex := regexp.MustCompile(`(?i)FOREIGN\s+KEY\s*\(\s*(\w+)\s*\)\s+REFERENCES\s+(\w+)\s*\(\s*(\w+)\s*\)(?:\s+ON\s+DELETE\s+(CASCADE|SET\s+NULL|RESTRICT|NO\s+ACTION))?`)
	matches := fkRegex.FindStringSubmatch(constraint)

	if len(matches) >= 4 {
		fk := &foreignKeyConstraint{
			ColumnName:       matches[1],
			ReferencedTable:  matches[2],
			ReferencedColumn: matches[3],
		}
		if len(matches) >= 5 && matches[4] != "" {
			fk.OnDeleteAction = strings.ToUpper(matches[4])
		}
		return fk
	}
	return nil
}

func (sm *SchemaManager) splitColumnDefinitions(defs string) []string {
	var result []string
	var current strings.Builder
	parenLevel := 0

	for _, char := range defs {
		switch char {
		case '(':
			parenLevel++
			current.WriteRune(char)
		case ')':
			parenLevel--
			current.WriteRune(char)
		case ',':
			if parenLevel == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

func (sm *SchemaManager) isTableConstraint(def string) bool {
	def = strings.ToUpper(strings.TrimSpace(def))
	prefixes := []string{"PRIMARY KEY", "FOREIGN KEY", "UNIQUE", "CHECK", "CONSTRAINT"}

	for _, prefix := range prefixes {
		if strings.HasPrefix(def, prefix) {
			return true
		}
	}
	return false
}

func (sm *SchemaManager) parseColumnDefinition(colDef string) (types.SchemaColumn, error) {
	parts := strings.Fields(colDef)
	if len(parts) < 2 {
		return types.SchemaColumn{}, fmt.Errorf("invalid column definition: %s", colDef)
	}

	column := types.SchemaColumn{
		Name:     strings.Trim(parts[0], `"`),
		Nullable: true,
	}

	// Handle multi-word types like "TIMESTAMP WITH TIME ZONE"
	if len(parts) > 4 && strings.ToUpper(parts[1]) == "TIMESTAMP" && strings.ToUpper(parts[2]) == "WITH" && strings.ToUpper(parts[3]) == "TIME" && strings.ToUpper(parts[4]) == "ZONE" {
		column.Type = "TIMESTAMP WITH TIME ZONE"
	} else if len(parts) > 4 && strings.ToUpper(parts[1]) == "TIMESTAMP" && strings.ToUpper(parts[2]) == "WITHOUT" && strings.ToUpper(parts[3]) == "TIME" && strings.ToUpper(parts[4]) == "ZONE" {
		column.Type = "TIMESTAMP WITHOUT TIME ZONE"
	} else if len(parts) > 2 && strings.ToUpper(parts[1]) == "DOUBLE" && strings.ToUpper(parts[2]) == "PRECISION" {
		column.Type = "DOUBLE PRECISION"
	} else if len(parts) > 2 && strings.ToUpper(parts[1]) == "CHARACTER" && strings.ToUpper(parts[2]) == "VARYING" {
		column.Type = "CHARACTER VARYING"
	} else {
		column.Type = parts[1]
	}

	sm.parseColumnConstraints(&column, colDef)
	return column, nil
}

func (sm *SchemaManager) parseColumnConstraints(column *types.SchemaColumn, colDef string) {
	defUpper := strings.ToUpper(colDef)

	constraints := map[string]func(){
		"NOT NULL":       func() { column.Nullable = false },
		"PRIMARY KEY":    func() { column.IsPrimary = true },
		"UNIQUE":         func() { column.IsUnique = true },
		"AUTOINCREMENT":  func() { column.IsPrimary = true },
		"AUTO_INCREMENT": func() { column.IsPrimary = true },
		"SERIAL":         func() { column.IsPrimary = true },
		"BIGSERIAL":      func() { column.IsPrimary = true },
		"SMALLSERIAL":    func() { column.IsPrimary = true },
	}

	for constraint, action := range constraints {
		if strings.Contains(defUpper, constraint) {
			action()
		}
	}

	referencesRegex := regexp.MustCompile(`(?i)REFERENCES\s+(\w+)\s*\(\s*(\w+)\s*\)`)
	if matches := referencesRegex.FindStringSubmatch(colDef); len(matches) >= 3 {
		column.ForeignKeyTable = matches[1]
		column.ForeignKeyColumn = matches[2]

		onDeleteRegex := regexp.MustCompile(`(?i)ON\s+DELETE\s+(CASCADE|SET\s+NULL|RESTRICT|NO\s+ACTION)`)
		if onDeleteMatches := onDeleteRegex.FindStringSubmatch(colDef); len(onDeleteMatches) >= 2 {
			column.OnDeleteAction = strings.ToUpper(onDeleteMatches[1])
		}
	}

	defaultRegex := regexp.MustCompile(`(?i)\bDEFAULT\s+([^,\s]+|'[^']*'|\([^)]*\))`)
	if matches := defaultRegex.FindStringSubmatch(colDef); len(matches) > 1 {
		column.Default = matches[1]
	}
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
