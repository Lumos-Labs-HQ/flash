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
	ColumnName, ReferencedTable, ReferencedColumn, OnDeleteAction, OnUpdateAction string
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
	tables, enums, indexes, _, err := sm.parseSchemaDirAll(schemaDir)
	return tables, enums, indexes, err
}

func (sm *SchemaManager) parseSchemaDirAll(schemaDir string) ([]types.SchemaTable, []types.SchemaEnum, []types.SchemaIndex, []types.SchemaKeyspace, error) {
	tables, enums, indexes, keyspaces, _, err := sm.parseSchemaDirAllV2(schemaDir)
	return tables, enums, indexes, keyspaces, err
}

func (sm *SchemaManager) parseSchemaDirAllV2(schemaDir string) ([]types.SchemaTable, []types.SchemaEnum, []types.SchemaIndex, []types.SchemaKeyspace, []types.SchemaUDT, error) {
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to read schema directory: %w", err)
	}

	var allTables []types.SchemaTable
	var allEnums []types.SchemaEnum
	var allIndexes []types.SchemaIndex
	var allKeyspaces []types.SchemaKeyspace
	var allUDTs []types.SchemaUDT
	tableMap := make(map[string]*types.SchemaTable)

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
			return nil, nil, nil, nil, nil, fmt.Errorf("failed to read schema file %s: %w", filePath, err)
		}

		tables, enums, indexes, keyspaces, udts, err := sm.parseSchemaContentAllV2(string(content))
		if err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("failed to parse schema file %s: %w", filePath, err)
		}

		for _, table := range tables {
			if existing, ok := tableMap[table.Name]; ok {
				existingCols := make(map[string]bool)
				for _, col := range existing.Columns {
					existingCols[col.Name] = true
				}
				for _, col := range table.Columns {
					if !existingCols[col.Name] {
						existing.Columns = append(existing.Columns, col)
					}
				}
				existing.Indexes = append(existing.Indexes, table.Indexes...)
			} else {
				tableCopy := table
				tableMap[table.Name] = &tableCopy
			}
		}

		allEnums = append(allEnums, enums...)
		allIndexes = append(allIndexes, indexes...)
		allKeyspaces = append(allKeyspaces, keyspaces...)
		allUDTs = append(allUDTs, udts...)
	}

	for _, table := range tableMap {
		allTables = append(allTables, *table)
	}

	allTables, err = sm.sortTablesByDependencies(allTables)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return allTables, allEnums, allIndexes, allKeyspaces, allUDTs, nil
}

// sortTablesByDependencies sorts tables so that referenced tables come before referencing tables.
// Also validates that all referenced tables exist.
// Providers without FK support (ScyllaDB, ClickHouse) skip validation — REFERENCES in schemas
// are treated as documentation hints, not enforceable constraints.
func (sm *SchemaManager) sortTablesByDependencies(tables []types.SchemaTable) ([]types.SchemaTable, error) {
	tableMap := make(map[string]*types.SchemaTable)
	for i := range tables {
		tableMap[tables[i].Name] = &tables[i]
	}

	skipFKs := false
	if sm.adapter != nil {
		provider := sm.adapter.ProviderName()
		skipFKs = provider == "scylla" || provider == "clickhouse"
	}

	// Build dependency graph and validate references
	// dependencies[A] = [B, C] means table A depends on tables B and C (A has FK to B and C)
	dependencies := make(map[string][]string)
	for _, table := range tables {
		var deps []string
		for _, col := range table.Columns {
			if col.ForeignKeyTable != "" {
				if skipFKs {
					// ScyllaDB/ClickHouse don't support FKs — REFERENCES is documentation-only.
					// Ignore for dependency ordering and skip existence validation.
					continue
				}
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
	tables, enums, indexes, _, err := sm.ParseSchemaPathAll(schemaPath)
	return tables, enums, indexes, err
}

func (sm *SchemaManager) ParseSchemaPathAll(schemaPath string) ([]types.SchemaTable, []types.SchemaEnum, []types.SchemaIndex, []types.SchemaKeyspace, error) {
	tables, enums, indexes, keyspaces, _, err := sm.ParseSchemaPathAllV2(schemaPath)
	return tables, enums, indexes, keyspaces, err
}

func (sm *SchemaManager) ParseSchemaPathAllV2(schemaPath string) ([]types.SchemaTable, []types.SchemaEnum, []types.SchemaIndex, []types.SchemaKeyspace, []types.SchemaUDT, error) {
	info, err := os.Stat(schemaPath)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to stat schema path: %w", err)
	}

	if info.IsDir() {
		return sm.parseSchemaDirAllV2(schemaPath)
	}

	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to read schema file: %w", err)
	}
	var allTables []types.SchemaTable
	var allEnums []types.SchemaEnum
	var allIndexes []types.SchemaIndex
	var allKeyspaces []types.SchemaKeyspace
	var allUDTs []types.SchemaUDT

	allTables, allEnums, allIndexes, allKeyspaces, allUDTs, err = sm.parseSchemaContentAllV2(string(content))
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	allTables, err = sm.sortTablesByDependencies(allTables)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return allTables, allEnums, allIndexes, allKeyspaces, allUDTs, nil
}

func (sm *SchemaManager) ParseSchemaFileWithEnumsAndIndexes(schemaPath string) ([]types.SchemaTable, []types.SchemaEnum, []types.SchemaIndex, error) {
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read schema file: %w", err)
	}
	tables, enums, indexes, _, _, parseErr := sm.parseSchemaContentAllV2(string(content))
	if parseErr != nil {
		return nil, nil, nil, parseErr
	}
	tables, err = sm.sortTablesByDependencies(tables)
	if err != nil {
		return nil, nil, nil, err
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
	tables, enums, indexes, _, _, err := sm.parseSchemaContentAllV2(content)
	return tables, enums, indexes, err
}

func (sm *SchemaManager) parseSchemaContentAllV2(content string) ([]types.SchemaTable, []types.SchemaEnum, []types.SchemaIndex, []types.SchemaKeyspace, []types.SchemaUDT, error) {
	var tables []types.SchemaTable
	var enums []types.SchemaEnum
	var indexes []types.SchemaIndex
	var keyspaces []types.SchemaKeyspace
	var udts []types.SchemaUDT
	statements := sm.splitStatements(sm.cleanSQL(content))

	tableMap := make(map[string]*types.SchemaTable)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if sm.isCreateUDTStatement(stmt) {
			// CQL UDT: CREATE TYPE ks.type (field type, ...)
			if udt, err := sm.parseCreateUDTStatement(stmt); err == nil {
				udts = append(udts, udt)
			}
		} else if sm.isCreateTypeStatement(stmt) {
			if enum, err := sm.parseCreateTypeStatement(stmt); err == nil {
				enums = append(enums, enum)
			}
		} else if sm.isCreateViewStatement(stmt) {
			// MATERIALIZED VIEWs are stored as tables for diff tracking purposes.
			// They don't have parseable columns in the same way, so we capture
			// the raw SQL and store the view name + dummy column.
			if viewName, viewSQL, err := sm.parseCreateViewStatement(stmt); err == nil {
				tables = append(tables, types.SchemaTable{
					Name:    viewName,
					Columns: []types.SchemaColumn{{Name: "/* MATERIALIZED VIEW */", Type: viewSQL, IsPrimary: true}},
				})
				_ = viewSQL
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
		} else if sm.isCreateKeyspaceStatement(stmt) {
			if ks, err := sm.parseCreateKeyspaceStatement(stmt); err == nil {
				keyspaces = append(keyspaces, ks)
			}
		}
	}
	return tables, enums, indexes, keyspaces, udts, nil
}

func (sm *SchemaManager) GenerateSchemaDiff(ctx context.Context, targetSchemaPath string, snapshotPath string) (*types.SchemaDiff, error) {
	var currentTables []types.SchemaTable
	var currentEnums []types.SchemaEnum

	snap, err := LoadSchemaSnapshot(snapshotPath)
	if err != nil {
		fmt.Printf("⚠️  Schema snapshot corrupted (%v). Falling back to live database.\n", err)
	}

	if snap != nil && err == nil {
		currentTables = snap.Tables
		currentEnums = snap.Enums
		for i, table := range currentTables {
			for _, idx := range snap.Indexes {
				if strings.EqualFold(idx.Table, table.Name) {
					dup := false
					for _, existing := range table.Indexes {
						if existing.Name == idx.Name {
							dup = true
							break
						}
					}
					if !dup {
						currentTables[i].Indexes = append(currentTables[i].Indexes, idx)
					}
				}
			}
		}
	} else {
		// No snapshot — fall back to live database introspection.
		// If the DB has tables, use them as the baseline so we don't
		// regenerate CREATE TABLE for already-applied tables.
		currentTables, err = sm.adapter.GetCurrentSchema(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get current schema: %w", err)
		}

		// Filter system tables for ScyllaDB/Cassandra (internal keyspace tables)
		filtered := make([]types.SchemaTable, 0, len(currentTables))
		for _, t := range currentTables {
			name := strings.ToLower(t.Name)
			if strings.HasPrefix(name, "system.") || strings.HasPrefix(name, "system_") {
				continue
			}
			if name == "indexinfo" || name == "batchlog" || name == "batchlog_v2" {
				continue
			}
			filtered = append(filtered, t)
		}
		currentTables = filtered

		currentEnums, err = sm.adapter.GetCurrentEnums(ctx)
		if err != nil {
			currentEnums = []types.SchemaEnum{}
		}

		// DO NOT save live-DB state as snapshot here.
		// The snapshot is saved AFTER migration generation (in GenerateMigration)
		// from the parsed schema files, not the live DB. This keeps the snapshot
		// as the authoritative "last schema we generated a migration for."
	}

	// Parse target schema including keyspaces and UDTs
	targetTables, targetEnums, targetIndexes, targetKeyspaces, targetUDTs, err := sm.ParseSchemaPathAllV2(targetSchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target schema: %w", err)
	}

	diff := sm.compareSchemas(currentTables, targetTables, currentEnums, targetEnums, targetIndexes)
	sm.compareKeyspaces(snap, targetKeyspaces, diff)
	sm.compareUDTs(snap, targetUDTs, diff)

	return diff, nil
}

func (sm *SchemaManager) compareKeyspaces(snap *SchemaSnapshot, target []types.SchemaKeyspace, diff *types.SchemaDiff) {
	currentMap := make(map[string]types.SchemaKeyspace)
	if snap != nil {
		for _, ks := range snap.Keyspaces {
			currentMap[ks.Name] = ks
		}
	}

	for _, ks := range target {
		if _, exists := currentMap[ks.Name]; !exists {
			diff.NewKeyspaces = append(diff.NewKeyspaces, ks)
		}
	}
	// Note: DroppedKeyspaces not handled automatically — user must manually add DROP KEYSPACE
}

func (sm *SchemaManager) compareUDTs(snap *SchemaSnapshot, target []types.SchemaUDT, diff *types.SchemaDiff) {
	currentMap := make(map[string]types.SchemaUDT)
	if snap != nil {
		for _, u := range snap.UDTs {
			currentMap[u.Name] = u
		}
	}

	for _, u := range target {
		if _, exists := currentMap[u.Name]; !exists {
			diff.NewUDTs = append(diff.NewUDTs, u)
		}
	}
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
