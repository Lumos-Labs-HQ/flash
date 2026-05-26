package schema

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/Lumos-Labs-HQ/flash/internal/types"
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

// ParseSchemaFile parses a single schema file (legacy support)
func (sm *SchemaManager) ParseSchemaFile(schemaPath string) ([]types.SchemaTable, error) {
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}
	tables, _, _ := sm.parseSchemaContent(string(content))
	return tables, nil
}

// ParseSchemaDir parses all .sql files in a directory
func (sm *SchemaManager) ParseSchemaDir(schemaDir string) ([]types.SchemaTable, []types.SchemaEnum, []types.SchemaIndex, error) {
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read schema directory: %w", err)
	}

	var allTables []types.SchemaTable
	var allEnums []types.SchemaEnum
	var allIndexes []types.SchemaIndex
	tableMap := make(map[string]*types.SchemaTable)

	// Sort entries for consistent ordering
	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, fileName := range sqlFiles {
		filePath := fmt.Sprintf("%s/%s", schemaDir, fileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read schema file %s: %w", filePath, err)
		}

		tables, enums, indexes, err := sm.parseSchemaContentWithIndexes(string(content))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse schema file %s: %w", filePath, err)
		}

		// Merge tables (handle same table in multiple files)
		for _, table := range tables {
			if existing, ok := tableMap[table.Name]; ok {
				// Merge columns (avoid duplicates)
				existingCols := make(map[string]bool)
				for _, col := range existing.Columns {
					existingCols[col.Name] = true
				}
				for _, col := range table.Columns {
					if !existingCols[col.Name] {
						existing.Columns = append(existing.Columns, col)
					}
				}
				// Merge indexes
				existing.Indexes = append(existing.Indexes, table.Indexes...)
			} else {
				tableCopy := table
				tableMap[table.Name] = &tableCopy
			}
		}

		allEnums = append(allEnums, enums...)
		allIndexes = append(allIndexes, indexes...)
	}

	// Convert map back to slice
	for _, table := range tableMap {
		allTables = append(allTables, *table)
	}

	// Validate foreign key references and sort tables by dependencies
	allTables, err = sm.sortTablesByDependencies(allTables)
	if err != nil {
		return nil, nil, nil, err
	}

	return allTables, allEnums, allIndexes, nil
}

// sortTablesByDependencies sorts tables so that referenced tables come before referencing tables
// Also validates that all referenced tables exist
func (sm *SchemaManager) sortTablesByDependencies(tables []types.SchemaTable) ([]types.SchemaTable, error) {
	tableMap := make(map[string]*types.SchemaTable)
	for i := range tables {
		tableMap[tables[i].Name] = &tables[i]
	}

	// Build dependency graph and validate references
	// dependencies[A] = [B, C] means table A depends on tables B and C (A has FK to B and C)
	dependencies := make(map[string][]string)
	for _, table := range tables {
		var deps []string
		for _, col := range table.Columns {
			if col.ForeignKeyTable != "" {
				// Validate that referenced table exists
				if _, exists := tableMap[col.ForeignKeyTable]; !exists {
					return nil, fmt.Errorf("table '%s' references non-existent table '%s' (column '%s' has REFERENCES %s(%s))",
						table.Name, col.ForeignKeyTable, col.Name, col.ForeignKeyTable, col.ForeignKeyColumn)
				}
				if col.ForeignKeyTable != table.Name {
					deps = append(deps, col.ForeignKeyTable)
				}
			}
		}
		dependencies[table.Name] = deps
	}

	// Topological sort using Kahn's algorithm
	// dependencies[A] = [B, C] means A depends on B and C (A has FK to B and C)
	// We want B and C created BEFORE A, so A's in-degree = number of dependencies
	var sorted []types.SchemaTable
	inDegree := make(map[string]int)

	// Build reverse adjacency list: dependents[dep] = []tables that depend on dep
	// This avoids O(T×D) inner scan during the sort loop
	dependents := make(map[string][]string)

	// Ensure every table has an entry so tables with no FKs are still processed.
	for _, table := range tables {
		inDegree[table.Name] = 0
	}

	// Calculate in-degree and build reverse adjacency list
	for tableName, deps := range dependencies {
		inDegree[tableName] = len(deps)
		for _, dep := range deps {
			dependents[dep] = append(dependents[dep], tableName)
		}
	}

	// Find all tables with no dependencies (in-degree = 0)
	var queue []string
	for tableName, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, tableName)
		}
	}
	
	// Process tables (sort queue only once for determinism)
	sort.Strings(queue)

	for len(queue) > 0 {
		tableName := queue[0]
		queue = queue[1:]

		if table, exists := tableMap[tableName]; exists {
			sorted = append(sorted, *table)
		}

		// Reduce in-degree for all tables that depend on this one — O(1) lookup via reverse adjacency
		for _, depTableName := range dependents[tableName] {
			inDegree[depTableName]--
			if inDegree[depTableName] == 0 {
				// Insert in sorted position to maintain determinism
				insertPos := 0
				for insertPos < len(queue) && queue[insertPos] < depTableName {
					insertPos++
				}
				queue = append(queue[:insertPos], append([]string{depTableName}, queue[insertPos:]...)...)
			}
		}
	}

	// Check for circular dependencies
	if len(sorted) != len(tables) {
		// Find tables involved in circular dependency
		var circular []string
		for tableName, degree := range inDegree {
			if degree > 0 {
				circular = append(circular, tableName)
			}
		}
		return nil, fmt.Errorf("circular foreign key dependency detected among tables: %v", circular)
	}

	return sorted, nil
}

// ParseSchemaPath parses schema from either a file or directory
func (sm *SchemaManager) ParseSchemaPath(schemaPath string) ([]types.SchemaTable, []types.SchemaEnum, []types.SchemaIndex, error) {
	info, err := os.Stat(schemaPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to stat schema path: %w", err)
	}

	if info.IsDir() {
		return sm.ParseSchemaDir(schemaPath)
	}

	// It's a file - use legacy method
	tables, enums, err := sm.ParseSchemaFileWithEnums(schemaPath)
	if err != nil {
		return nil, nil, nil, err
	}

	// Validate foreign key references and sort tables by dependencies
	tables, err = sm.sortTablesByDependencies(tables)
	if err != nil {
		return nil, nil, nil, err
	}

	// Extract indexes from tables
	var indexes []types.SchemaIndex
	for _, table := range tables {
		indexes = append(indexes, table.Indexes...)
	}

	return tables, enums, indexes, nil
}

func (sm *SchemaManager) ParseSchemaFileWithEnums(schemaPath string) ([]types.SchemaTable, []types.SchemaEnum, error) {
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read schema file: %w", err)
	}
	return sm.parseSchemaContent(string(content))
}

func (sm *SchemaManager) parseSchemaContent(content string) ([]types.SchemaTable, []types.SchemaEnum, error) {
	tables, enums, _, err := sm.parseSchemaContentWithIndexes(content)
	return tables, enums, err
}

func (sm *SchemaManager) parseSchemaContentWithIndexes(content string) ([]types.SchemaTable, []types.SchemaEnum, []types.SchemaIndex, error) {
	var tables []types.SchemaTable
	var enums []types.SchemaEnum
	var indexes []types.SchemaIndex
	statements := sm.splitStatements(sm.cleanSQL(content))

	tableMap := make(map[string]*types.SchemaTable)

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
				tableMap[table.Name] = &tables[len(tables)-1]
			}
		} else if sm.isCreateIndexStatement(stmt) {
			if index, err := sm.parseCreateIndexStatement(stmt); err == nil {
				indexes = append(indexes, index)
				if table, ok := tableMap[index.Table]; ok {
					table.Indexes = append(table.Indexes, index)
				}
			}
		}
	}
	return tables, enums, indexes, nil
}

func (sm *SchemaManager) GenerateSchemaDiff(ctx context.Context, targetSchemaPath string, snapshotPath string) (*types.SchemaDiff, error) {
	var currentTables []types.SchemaTable
	var currentEnums []types.SchemaEnum
	var currentIndexes []types.SchemaIndex

	// 1. Prefer the local schema snapshot (accurate even when migrations are pending).
	snap, err := LoadSchemaSnapshot(snapshotPath)
	if err != nil {
		// Corrupted snapshot → warn and fall back to DB
		fmt.Printf("⚠️  Schema snapshot corrupted (%v). Falling back to live database.\n", err)
	}
	if snap != nil && err == nil {
		currentTables = snap.Tables
		currentEnums = snap.Enums
		currentIndexes = snap.Indexes
	} else {
		// 2. Snapshot missing or invalid → fall back to live database introspection.
		currentTables, err = sm.adapter.GetCurrentSchema(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get current schema: %w", err)
		}

		currentEnums, err = sm.adapter.GetCurrentEnums(ctx)
		if err != nil {
			currentEnums = []types.SchemaEnum{}
		}
		// Extract indexes from current tables since DB introspection returns them inline
		for _, table := range currentTables {
			currentIndexes = append(currentIndexes, table.Indexes...)
		}
	}

	// Use the new ParseSchemaPath that handles both files and directories
	// Standalone CREATE INDEX statements are kept separate from table definitions.
	targetTables, targetEnums, targetIndexes, err := sm.ParseSchemaPath(targetSchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target schema: %w", err)
	}

	// Pass both tables and standalone indexes to compareSchemas
	diff := sm.compareSchemas(currentTables, targetTables, currentEnums, targetEnums, targetIndexes)

	return diff, nil
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
		for _, column := range tableDiff.DroppedColumns {
			parts = append(parts, sm.adapter.GenerateDropColumnSQL(tableDiff.Name, column.Name))
		}
	}

	for _, index := range diff.DroppedIndexes {
		parts = append(parts, sm.adapter.GenerateDropIndexSQL(index))
	}
	for _, index := range diff.NewIndexes {
		parts = append(parts, sm.adapter.GenerateAddIndexSQL(index))
	}

	return strings.Join(parts, "\n\n")
}
