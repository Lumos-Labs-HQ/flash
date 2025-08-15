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

// Temporary structure for parsing foreign key constraints
type foreignKeyConstraint struct {
	ColumnName       string
	ReferencedTable  string
	ReferencedColumn string
	OnDeleteAction   string
}

// SchemaManager handles schema parsing and generation
type SchemaManager struct {
	adapter database.DatabaseAdapter
}

// NewSchemaManager creates a new schema manager
func NewSchemaManager(adapter database.DatabaseAdapter) *SchemaManager {
	return &SchemaManager{
		adapter: adapter,
	}
}

// ParseSchemaFile parses a schema file and returns table definitions
func (sm *SchemaManager) ParseSchemaFile(schemaPath string) ([]types.SchemaTable, error) {
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	return sm.parseSchemaContent(string(content))
}

// parseSchemaContent parses SQL content and extracts table definitions
func (sm *SchemaManager) parseSchemaContent(content string) ([]types.SchemaTable, error) {
	var tables []types.SchemaTable

	// Remove comments and normalize whitespace
	content = sm.cleanSQL(content)

	// Split into statements
	statements := sm.splitStatements(content)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		// Check if it's a CREATE TABLE statement
		if sm.isCreateTableStatement(stmt) {
			table, err := sm.parseCreateTableStatement(stmt)
			if err != nil {
				return nil, fmt.Errorf("failed to parse CREATE TABLE statement: %w", err)
			}
			tables = append(tables, table)
		}
	}

	return tables, nil
}

// GenerateSchemaDiff compares current database schema with target schema
func (sm *SchemaManager) GenerateSchemaDiff(ctx context.Context, targetSchemaPath string) (*types.SchemaDiff, error) {
	// Get current database schema
	currentTables, err := sm.adapter.GetCurrentSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current schema: %w", err)
	}

	// Parse target schema from file
	targetTables, err := sm.ParseSchemaFile(targetSchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target schema: %w", err)
	}

	return sm.compareSchemas(currentTables, targetTables), nil
}

// GenerateSchemaSQL generates SQL content from schema tables
func (sm *SchemaManager) GenerateSchemaSQL(tables []types.SchemaTable) string {
	var builder strings.Builder

	// Sort tables by name for consistent output
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})

	for i, table := range tables {
		if i > 0 {
			builder.WriteString("\n\n")
		}

		// Generate CREATE TABLE statement
		builder.WriteString(sm.adapter.GenerateCreateTableSQL(table))

		// Generate indexes
		for _, index := range table.Indexes {
			builder.WriteString("\n")
			builder.WriteString(sm.adapter.GenerateAddIndexSQL(index))
		}
	}

	return builder.String()
}

// GenerateMigrationSQL generates SQL for a schema diff
func (sm *SchemaManager) GenerateMigrationSQL(diff *types.SchemaDiff) string {
	var builder strings.Builder

	// Drop tables first
	for _, tableName := range diff.DroppedTables {
		builder.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS \"%s\";\n\n", tableName))
	}

	// Create new tables
	for _, table := range diff.NewTables {
		builder.WriteString(sm.adapter.GenerateCreateTableSQL(table))
		builder.WriteString("\n\n")

		// Add indexes for new tables
		for _, index := range table.Indexes {
			builder.WriteString(sm.adapter.GenerateAddIndexSQL(index))
			builder.WriteString("\n")
		}
		if len(table.Indexes) > 0 {
			builder.WriteString("\n")
		}
	}

	// Modify existing tables
	for _, tableDiff := range diff.ModifiedTables {
		// Drop columns
		for _, columnName := range tableDiff.DroppedColumns {
			builder.WriteString(sm.adapter.GenerateDropColumnSQL(tableDiff.Name, columnName))
			builder.WriteString("\n")
		}

		// Add new columns
		for _, column := range tableDiff.NewColumns {
			builder.WriteString(sm.adapter.GenerateAddColumnSQL(tableDiff.Name, column))
			builder.WriteString("\n")
		}

		// Modify existing columns (this is complex and database-specific)
		for _, columnDiff := range tableDiff.ModifiedColumns {
			builder.WriteString(fmt.Sprintf("-- TODO: Modify column %s.%s from %s to %s\n",
				tableDiff.Name, columnDiff.Name, columnDiff.OldType, columnDiff.NewType))
		}

		if len(tableDiff.DroppedColumns) > 0 || len(tableDiff.NewColumns) > 0 || len(tableDiff.ModifiedColumns) > 0 {
			builder.WriteString("\n")
		}
	}

	// Drop indexes
	for _, indexName := range diff.DroppedIndexes {
		builder.WriteString(sm.adapter.GenerateDropIndexSQL(indexName))
		builder.WriteString("\n")
	}

	// Add new indexes
	for _, index := range diff.NewIndexes {
		builder.WriteString(sm.adapter.GenerateAddIndexSQL(index))
		builder.WriteString("\n")
	}

	return strings.TrimSpace(builder.String())
}

// compareSchemas compares two sets of tables and returns differences
func (sm *SchemaManager) compareSchemas(current, target []types.SchemaTable) *types.SchemaDiff {
	diff := &types.SchemaDiff{}

	// Create maps for easier lookup
	currentMap := make(map[string]types.SchemaTable)
	targetMap := make(map[string]types.SchemaTable)

	for _, table := range current {
		currentMap[table.Name] = table
	}

	for _, table := range target {
		targetMap[table.Name] = table
	}

	// Find new and modified tables
	for _, targetTable := range target {
		if currentTable, exists := currentMap[targetTable.Name]; !exists {
			// New table
			diff.NewTables = append(diff.NewTables, targetTable)
		} else {
			// Potentially modified table
			tableDiff := sm.compareTablesForDiff(currentTable, targetTable)
			if tableDiff != nil {
				diff.ModifiedTables = append(diff.ModifiedTables, *tableDiff)
			}
		}
	}

	// Find dropped tables
	for _, currentTable := range current {
		if _, exists := targetMap[currentTable.Name]; !exists {
			diff.DroppedTables = append(diff.DroppedTables, currentTable.Name)
		}
	}

	// Compare indexes
	sm.compareIndexes(current, target, diff)

	return diff
}

// compareTablesForDiff compares two tables and returns differences
func (sm *SchemaManager) compareTablesForDiff(current, target types.SchemaTable) *types.TableDiff {
	tableDiff := &types.TableDiff{Name: target.Name}
	hasChanges := false

	// Create maps for easier lookup
	currentCols := make(map[string]types.SchemaColumn)
	targetCols := make(map[string]types.SchemaColumn)

	for _, col := range current.Columns {
		currentCols[col.Name] = col
	}

	for _, col := range target.Columns {
		targetCols[col.Name] = col
	}

	// Find new and modified columns
	for _, targetCol := range target.Columns {
		if currentCol, exists := currentCols[targetCol.Name]; !exists {
			// New column
			tableDiff.NewColumns = append(tableDiff.NewColumns, targetCol)
			hasChanges = true
		} else {
			// Check for modifications
			if !sm.columnsEqual(currentCol, targetCol) {
				tableDiff.ModifiedColumns = append(tableDiff.ModifiedColumns, types.ColumnDiff{
					Name:    targetCol.Name,
					OldType: currentCol.Type,
					NewType: targetCol.Type,
					Changes: sm.getColumnChanges(currentCol, targetCol),
				})
				hasChanges = true
			}
		}
	}

	// Find dropped columns
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

// compareIndexes compares indexes between current and target schemas
func (sm *SchemaManager) compareIndexes(current, target []types.SchemaTable, diff *types.SchemaDiff) {
	currentIndexes := make(map[string]types.SchemaIndex)
	targetIndexes := make(map[string]types.SchemaIndex)

	// Collect all indexes
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

	// Find new indexes
	for name, index := range targetIndexes {
		if _, exists := currentIndexes[name]; !exists {
			diff.NewIndexes = append(diff.NewIndexes, index)
		}
	}

	// Find dropped indexes
	for name := range currentIndexes {
		if _, exists := targetIndexes[name]; !exists {
			diff.DroppedIndexes = append(diff.DroppedIndexes, name)
		}
	}
}

// Helper methods for parsing SQL

func (sm *SchemaManager) cleanSQL(sql string) string {
	// Remove single-line comments
	re := regexp.MustCompile(`--.*`)
	sql = re.ReplaceAllString(sql, "")

	// Remove multi-line comments
	re = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	sql = re.ReplaceAllString(sql, "")

	// Normalize whitespace
	re = regexp.MustCompile(`\s+`)
	sql = re.ReplaceAllString(sql, " ")

	return strings.TrimSpace(sql)
}

func (sm *SchemaManager) splitStatements(sql string) []string {
	// Simple split by semicolon (could be improved for edge cases)
	statements := strings.Split(sql, ";")
	var result []string

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			result = append(result, stmt)
		}
	}

	return result
}

func (sm *SchemaManager) isCreateTableStatement(stmt string) bool {
	return regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE`).MatchString(stmt)
}

func (sm *SchemaManager) parseCreateTableStatement(stmt string) (types.SchemaTable, error) {
	// Extract table name - handle IF NOT EXISTS syntax
	re := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:"?(\w+)"?|(\w+)|\x60(\w+)\x60)\s*\(`)
	matches := re.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return types.SchemaTable{}, fmt.Errorf("could not extract table name from: %s", stmt)
	}

	tableName := matches[1]
	if tableName == "" {
		tableName = matches[2]
	}
	if tableName == "" {
		tableName = matches[3] // For backtick-quoted names
	}

	// Extract column definitions
	start := strings.Index(stmt, "(")
	end := strings.LastIndex(stmt, ")")
	if start == -1 || end == -1 {
		return types.SchemaTable{}, fmt.Errorf("invalid CREATE TABLE syntax")
	}

	columnDefs := stmt[start+1 : end]
	columns, foreignKeys, err := sm.parseColumnDefinitionsAndConstraints(columnDefs)
	if err != nil {
		return types.SchemaTable{}, err
	}

	// Apply foreign key constraints to columns
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

	return types.SchemaTable{
		Name:    tableName,
		Columns: columns,
		Indexes: []types.SchemaIndex{}, // Indexes are parsed separately
	}, nil
}

func (sm *SchemaManager) parseColumnDefinitions(columnDefs string) ([]types.SchemaColumn, error) {
	var columns []types.SchemaColumn

	// Split by comma, but be careful about commas inside parentheses
	colDefs := sm.splitColumnDefinitions(columnDefs)

	for _, colDef := range colDefs {
		colDef = strings.TrimSpace(colDef)
		if colDef == "" {
			continue
		}

		// Skip constraints that are not column definitions
		if sm.isTableConstraint(colDef) {
			continue
		}

		column, err := sm.parseColumnDefinition(colDef)
		if err != nil {
			return nil, err
		}

		columns = append(columns, column)
	}

	return columns, nil
}

func (sm *SchemaManager) parseColumnDefinitionsAndConstraints(columnDefs string) ([]types.SchemaColumn, []foreignKeyConstraint, error) {
	var columns []types.SchemaColumn
	var foreignKeys []foreignKeyConstraint

	// Split by comma, but be careful about commas inside parentheses
	colDefs := sm.splitColumnDefinitions(columnDefs)

	for _, colDef := range colDefs {
		colDef = strings.TrimSpace(colDef)
		if colDef == "" {
			continue
		}

		// Check if this is a table-level FOREIGN KEY constraint
		if sm.isTableConstraint(colDef) {
			fk := sm.parseForeignKeyConstraint(colDef)
			if fk != nil {
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
	constraint = strings.TrimSpace(constraint)

	// Match FOREIGN KEY (column) REFERENCES table(column) [ON DELETE action]
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
	return strings.HasPrefix(def, "PRIMARY KEY") ||
		strings.HasPrefix(def, "FOREIGN KEY") ||
		strings.HasPrefix(def, "UNIQUE") ||
		strings.HasPrefix(def, "CHECK") ||
		strings.HasPrefix(def, "CONSTRAINT")
}

func (sm *SchemaManager) parseColumnDefinition(colDef string) (types.SchemaColumn, error) {
	parts := strings.Fields(colDef)
	if len(parts) < 2 {
		return types.SchemaColumn{}, fmt.Errorf("invalid column definition: %s", colDef)
	}

	column := types.SchemaColumn{
		Name:     strings.Trim(parts[0], `"`),
		Type:     parts[1],
		Nullable: true, // Default to nullable
	}

	// Parse constraints and modifiers
	defUpper := strings.ToUpper(colDef)

	if strings.Contains(defUpper, "NOT NULL") {
		column.Nullable = false
	}

	// Check for PRIMARY KEY
	if strings.Contains(defUpper, "PRIMARY KEY") {
		column.IsPrimary = true
	}

	// Check for UNIQUE constraint
	if strings.Contains(defUpper, "UNIQUE") {
		column.IsUnique = true
	}

	// For SQLite, check for AUTOINCREMENT (which implies PRIMARY KEY)
	if strings.Contains(defUpper, "AUTOINCREMENT") {
		column.IsPrimary = true
	}

	// For MySQL, check for AUTO_INCREMENT (which implies PRIMARY KEY)
	if strings.Contains(defUpper, "AUTO_INCREMENT") {
		column.IsPrimary = true
	}

	// For PostgreSQL, check for SERIAL types (which implies PRIMARY KEY)
	if strings.Contains(defUpper, "SERIAL") || strings.Contains(defUpper, "BIGSERIAL") || strings.Contains(defUpper, "SMALLSERIAL") {
		column.IsPrimary = true
	}

	// Parse foreign key references
	if referencesMatch := regexp.MustCompile(`(?i)REFERENCES\s+(\w+)\s*\(\s*(\w+)\s*\)`).FindStringSubmatch(colDef); len(referencesMatch) >= 3 {
		column.ForeignKeyTable = referencesMatch[1]
		column.ForeignKeyColumn = referencesMatch[2]

		// Parse ON DELETE action
		if onDeleteMatch := regexp.MustCompile(`(?i)ON\s+DELETE\s+(CASCADE|SET\s+NULL|RESTRICT|NO\s+ACTION)`).FindStringSubmatch(colDef); len(onDeleteMatch) >= 2 {
			column.OnDeleteAction = strings.ToUpper(onDeleteMatch[1])
		}
	}

	// Extract default value
	if defaultMatch := regexp.MustCompile(`(?i)DEFAULT\s+([^,\s]+|'[^']*'|\([^)]*\))`).FindStringSubmatch(colDef); len(defaultMatch) > 1 {
		column.Default = defaultMatch[1]
	}

	return column, nil
}

// Helper methods for comparison

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

	if old.Type != new.Type {
		changes = append(changes, fmt.Sprintf("type changed from %s to %s", old.Type, new.Type))
	}

	if old.Nullable != new.Nullable {
		if new.Nullable {
			changes = append(changes, "made nullable")
		} else {
			changes = append(changes, "made not nullable")
		}
	}

	if old.Default != new.Default {
		changes = append(changes, fmt.Sprintf("default changed from %s to %s", old.Default, new.Default))
	}

	if old.IsPrimary != new.IsPrimary {
		if new.IsPrimary {
			changes = append(changes, "made primary key")
		} else {
			changes = append(changes, "removed primary key")
		}
	}

	if old.IsUnique != new.IsUnique {
		if new.IsUnique {
			changes = append(changes, "made unique")
		} else {
			changes = append(changes, "removed unique constraint")
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
