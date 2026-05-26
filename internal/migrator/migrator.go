package migrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/Lumos-Labs-HQ/flash/internal/schema"
	"github.com/Lumos-Labs-HQ/flash/internal/types"
	"github.com/Lumos-Labs-HQ/flash/internal/utils"
)

type Migrator struct {
	adapter       database.DatabaseAdapter
	schemaManager *schema.SchemaManager
	migrationsDir string
	schemaPath    string
	provider      string // Database provider: sqlite, postgresql, mysql
	force         bool
	fileUtils     *utils.FileUtils
	inputUtils    *utils.InputUtils
	conflictUtils *utils.ConflictUtils
	fileCache     map[string][]byte // In-memory cache for migration file contents
}

func NewMigrator(cfg *config.Config) (*Migrator, error) {
	adapter := database.NewAdapter(cfg.Database.Provider)

	dbURL, err := cfg.GetDatabaseURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get database URL: %w", err)
	}

	if err := adapter.Connect(context.Background(), dbURL); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Migrator{
		adapter:       adapter,
		schemaManager: schema.NewSchemaManager(adapter),
		migrationsDir: cfg.MigrationsPath,
		schemaPath:    cfg.GetSchemaDir(), // Use schema directory instead of single file
		provider:      cfg.Database.Provider,
		force:         false,
		fileUtils:     &utils.FileUtils{},
		inputUtils:    &utils.InputUtils{},
		conflictUtils: &utils.ConflictUtils{},
	}, nil
}

func (m *Migrator) Close() error {
	return m.adapter.Close()
}

func (m *Migrator) SetForce(force bool) {
	m.force = force
}

// Core migration operations - simplified using utils
func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	return m.adapter.CreateMigrationsTable(ctx)
}

func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[string]*time.Time, error) {
	return m.adapter.GetAppliedMigrations(ctx)
}

func (m *Migrator) loadMigrationsFromDir() ([]types.Migration, error) {
	return m.fileUtils.LoadMigrationsFromDir(m.migrationsDir)
}

func (m *Migrator) hasConflicts(ctx context.Context, pendingMigrations []types.Migration) (bool, []types.MigrationConflict, error) {
	var allConflicts []types.MigrationConflict

	for _, migration := range pendingMigrations {
		conflicts, err := m.conflictUtils.DetectMigrationConflicts(ctx, migration, m.adapter)
		if err != nil {
			return false, nil, fmt.Errorf("failed to detect conflicts for migration %s: %w", migration.ID, err)
		}
		allConflicts = append(allConflicts, conflicts...)
	}

	return len(allConflicts) > 0, allConflicts, nil
}

func (m *Migrator) cleanupBrokenMigrationRecords(ctx context.Context) error {
	return m.adapter.CleanupBrokenMigrationRecords(ctx)
}

// GenerateMigration creates a new migration file - simplified
func (m *Migrator) GenerateMigration(ctx context.Context, name string, schemaPath string) error {
	if schemaPath == "" {
		schemaPath = m.schemaPath
	}

	// Use the local schema snapshot for diffing so we can generate migrations
	// even when previous ones haven't been applied yet.
	snapshotPath := schema.SnapshotPath(m.migrationsDir)

	diff, err := m.schemaManager.GenerateSchemaDiff(ctx, schemaPath, snapshotPath)
	if err != nil {
		return fmt.Errorf("failed to generate schema diff: %w", err)
	}

	filename := m.fileUtils.GenerateMigrationFilename(name)
	filepath := filepath.Join(m.migrationsDir, filename)

	var sqlContent string
	// Check for index changes too, not just tables and enums.
	if len(diff.NewTables) == 0 && len(diff.DroppedTables) == 0 && len(diff.ModifiedTables) == 0 &&
	   len(diff.NewEnums) == 0 && len(diff.DroppedEnums) == 0 &&
	   len(diff.NewIndexes) == 0 && len(diff.DroppedIndexes) == 0 {
		fmt.Println("No changes detected in schema, creating empty migration template")
		sqlContent = m.generateEmptyMigrationTemplate(name)
	} else {
		sqlContent, _ = m.generateSQLFromDiff(diff, name)
	}

	if err := os.WriteFile(filepath, []byte(sqlContent), 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	// After generating the migration, update the snapshot so the next
	// generation diffs against this new schema state.
	targetTables, targetEnums, targetIndexes, err := m.schemaManager.ParseSchemaPath(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to parse target schema for snapshot: %w", err)
	}

	// Include standalone indexes in the snapshot so they are not regenerated
	// on every subsequent migration.
	if err := schema.SaveSchemaSnapshot(snapshotPath, targetTables, targetEnums, targetIndexes); err != nil {
		return fmt.Errorf("failed to save schema snapshot: %w", err)
	}

	fmt.Printf("Generated migration: %s\n", filename)
	return nil
}

// generateSQLFromDiff creates SQL from schema differences with both UP and DOWN.
// It returns the formatted migration file and a bool indicating whether any
// executable (non-comment) SQL statements were generated.
func (m *Migrator) generateSQLFromDiff(diff *types.SchemaDiff, name string) (string, bool) {
	var upStatements []string
	var downStatements []string
	hasExecutableSQL := false

	dropTableSQL := func(tableName string) string {
		switch m.provider {
		case "sqlite", "sqlite3":
			return fmt.Sprintf("DROP TABLE IF EXISTS \"%s\";", tableName)
		default:
			return fmt.Sprintf("DROP TABLE IF EXISTS \"%s\" CASCADE;", tableName)
		}
	}

	for _, enum := range diff.NewEnums {
		values := make([]string, len(enum.Values))
		for i, v := range enum.Values {
			// Escape single quotes in enum values for SQL safety
			escapedValue := strings.ReplaceAll(v, "'", "''")
			values[i] = fmt.Sprintf("'%s'", escapedValue)
		}
		// Escape the enum name for both single-quoted string and double-quoted identifier
		escapedNameSingle := strings.ReplaceAll(enum.Name, "'", "''")
		escapedNameDouble := strings.ReplaceAll(enum.Name, "\"", "\"\"")

		// PostgreSQL-specific enum creation with existence guard
		if m.provider == "postgresql" || m.provider == "postgres" {
			enumSQL := fmt.Sprintf(`DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = '%s') THEN
        CREATE TYPE "%s" AS ENUM (%s);
    END IF;
END $$;`, escapedNameSingle, escapedNameDouble, strings.Join(values, ", "))
			upStatements = append(upStatements, enumSQL)
			hasExecutableSQL = true
			// DOWN: Drop enum (escape double quotes for identifier)
			downStatements = append([]string{fmt.Sprintf("DROP TYPE IF EXISTS \"%s\";", escapedNameDouble)}, downStatements...)
		} else if m.provider == "mysql" {
			// MySQL enums are inline on columns; standalone enum changes should not generate SQL here
			// because they are handled as column type changes in ModifiedTables.
			continue
		} else if m.provider == "sqlite" || m.provider == "sqlite3" {
			// SQLite does not support user-defined types; skip enum SQL generation
			continue
		}
	}

	// UP: Create new tables and their indexes
	for _, table := range diff.NewTables {
		sql := m.adapter.GenerateCreateTableSQL(table)
		if sql != "" {
			upStatements = append(upStatements, sql)
			hasExecutableSQL = true
		}

		for _, index := range table.Indexes {
			if strings.HasPrefix(index.Name, "sqlite_") {
				continue
			}
			indexSQL := m.adapter.GenerateAddIndexSQL(index)
			if indexSQL != "" {
				upStatements = append(upStatements, indexSQL)
				hasExecutableSQL = true
			}
		}

		downStatements = append([]string{dropTableSQL(table.Name)}, downStatements...)
		for _, index := range table.Indexes {
			if strings.HasPrefix(index.Name, "sqlite_") {
				continue
			}
			downStatements = append([]string{fmt.Sprintf("DROP INDEX IF EXISTS \"%s\";", index.Name)}, downStatements...)
		}
	}

	// UP: Modify existing tables
	for _, tableDiff := range diff.ModifiedTables {
		needsSQLiteRecreate := (m.provider == "sqlite" || m.provider == "sqlite3") &&
			len(tableDiff.ModifiedColumns) > 0 &&
			m.hasSignificantSQLiteModifications(tableDiff)

		if !needsSQLiteRecreate {
			// Add new columns
			for _, column := range tableDiff.NewColumns {
				sql := m.adapter.GenerateAddColumnSQL(tableDiff.Name, column)
				if sql != "" {
					upStatements = append(upStatements, sql)
					hasExecutableSQL = true
					// DOWN: Drop the added column
					downStatements = append([]string{m.adapter.GenerateDropColumnSQL(tableDiff.Name, column.Name)}, downStatements...)
				}
			}

			// Drop columns
			for _, column := range tableDiff.DroppedColumns {
				sql := m.adapter.GenerateDropColumnSQL(tableDiff.Name, column.Name)
				if sql != "" {
					upStatements = append(upStatements, sql)
					hasExecutableSQL = true
					// DOWN: Re-add the dropped column with its original definition
					downStatements = append([]string{m.adapter.GenerateAddColumnSQL(tableDiff.Name, column)}, downStatements...)
				}
			}
		}

		// Modified columns
		if len(tableDiff.ModifiedColumns) > 0 {
			if m.provider == "sqlite" || m.provider == "sqlite3" {
				if needsSQLiteRecreate {
					// SQLite does not support ALTER COLUMN. Recreate the table.
					// Table recreation inherently handles added/dropped columns too,
					// so we skip individual ALTER TABLE statements above when
					// needsSQLiteRecreate is true.
					recreateSQL := m.generateSQLiteTableRecreateSQL(tableDiff.OldTable, tableDiff.NewTable)
					if recreateSQL != "" {
						upStatements = append(upStatements, recreateSQL)
						hasExecutableSQL = true
						// DOWN: reverse recreation
						downRecreate := m.generateSQLiteTableRecreateSQL(tableDiff.NewTable, tableDiff.OldTable)
						downStatements = append([]string{downRecreate}, downStatements...)
					}
				}
				// else: cosmetic type changes (e.g. TEXT → VARCHAR(255)) are ignored
				// for SQLite since they have no semantic effect.
			} else {
				for _, colDiff := range tableDiff.ModifiedColumns {
					sql := m.adapter.GenerateAlterColumnSQL(tableDiff.Name, colDiff.NewColumn, colDiff.OldType)
					if sql != "" {
						upStatements = append(upStatements, sql)
						hasExecutableSQL = true
						// DOWN: Revert to old column definition
						revertSQL := m.adapter.GenerateAlterColumnSQL(tableDiff.Name, colDiff.OldColumn, colDiff.NewType)
						if revertSQL != "" {
							downStatements = append([]string{revertSQL}, downStatements...)
						}
					}
				}
			}
		}
	}

	// UP: Drop tables
	for _, tableName := range diff.DroppedTables {
		upStatements = append(upStatements, dropTableSQL(tableName))
		hasExecutableSQL = true
		// DOWN: We can't restore dropped tables, add a comment
		downStatements = append([]string{fmt.Sprintf("-- Cannot restore dropped table: %s (data lost)", tableName)}, downStatements...)
	}

	// UP: Drop enums
	for _, enumName := range diff.DroppedEnums {
		upStatements = append(upStatements, fmt.Sprintf("DROP TYPE IF EXISTS \"%s\";", enumName))
		hasExecutableSQL = true
		// DOWN: We can't fully restore dropped enums
		downStatements = append([]string{fmt.Sprintf("-- Cannot restore dropped enum: %s", enumName)}, downStatements...)
	}

	// Handle standalone index changes (drop first to avoid conflicts).
	for _, index := range diff.DroppedIndexes {
		upStatements = append(upStatements, m.adapter.GenerateDropIndexSQL(index))
		hasExecutableSQL = true
		// DOWN: We can't fully restore dropped indexes
		downStatements = append([]string{fmt.Sprintf("-- Cannot restore dropped index: %s", index.Name)}, downStatements...)
	}

	// UP: Add new indexes
	for _, index := range diff.NewIndexes {
		if strings.HasPrefix(index.Name, "sqlite_") {
			continue
		}
		indexSQL := m.adapter.GenerateAddIndexSQL(index)
		if indexSQL != "" {
			upStatements = append(upStatements, indexSQL)
			hasExecutableSQL = true
			// DOWN: Drop the added index
			downStatements = append([]string{fmt.Sprintf("DROP INDEX IF EXISTS \"%s\";", index.Name)}, downStatements...)
		}
	}

	return m.formatMigrationFileWithDown(name, upStatements, downStatements), hasExecutableSQL
}

func (m *Migrator) generateEmptyMigrationTemplate(name string) string {
	upStatements := []string{
		"-- Add your SQL statements here",
		"-- Example: CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(255) NOT NULL);",
	}

	return m.formatMigrationFile(name, upStatements)
}

func (m *Migrator) formatMigrationFile(name string, upStatements []string) string {
	return m.formatMigrationFileWithDown(name, upStatements, nil)
}

func (m *Migrator) formatMigrationFileWithDown(name string, upStatements []string, downStatements []string) string {
	timestamp := time.Now().Format("2006-01-02T15:04:05Z")

	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("-- Migration: %s\n", name))
	builder.WriteString(fmt.Sprintf("-- Created: %s\n\n", timestamp))

	// UP section
	builder.WriteString("-- +migrate Up\n")
	if len(upStatements) > 0 {
		for _, stmt := range upStatements {
			builder.WriteString(stmt)
			if !strings.HasSuffix(stmt, ";") {
				builder.WriteString(";")
			}
			builder.WriteString("\n")
		}
	} else {
		builder.WriteString("-- No migration statements\n")
	}

	// DOWN section
	builder.WriteString("\n-- +migrate Down\n")
	if len(downStatements) > 0 {
		for _, stmt := range downStatements {
			builder.WriteString(stmt)
			if !strings.HasSuffix(strings.TrimSpace(stmt), ";") && !strings.HasPrefix(strings.TrimSpace(stmt), "--") {
				builder.WriteString(";")
			}
			builder.WriteString("\n")
		}
	} else {
		builder.WriteString("-- Add rollback statements here\n")
	}

	return builder.String()
}

func (m *Migrator) PullSchema(ctx context.Context) ([]types.SchemaTable, error) {
	return m.adapter.GetCurrentSchema(ctx)
}

// generateSQLiteTableRecreateSQL generates the multi-statement SQL required to
// recreate a SQLite table when columns are modified (since SQLite does not
// support ALTER COLUMN). The pattern is:
//   1. Create a temporary table with the new schema
//   2. Copy data from old to new (matching columns only)
//   3. Drop the old table
//   4. Rename the temporary table
//   5. Recreate indexes
func (m *Migrator) generateSQLiteTableRecreateSQL(oldTable, newTable types.SchemaTable) string {
	var parts []string

	parts = append(parts, "PRAGMA foreign_keys=OFF;")

	// Create temporary table with the desired schema
	tempTable := newTable
	tempTable.Name = newTable.Name + "_new"
	tempTable.Indexes = nil // Indexes added after rename
	createSQL := m.adapter.GenerateCreateTableSQL(tempTable)
	// Replace "IF NOT EXISTS" with plain CREATE for clarity
	createSQL = strings.Replace(createSQL, "CREATE TABLE IF NOT EXISTS", "CREATE TABLE", 1)
	parts = append(parts, createSQL)

	// Build list of columns common to both tables for the INSERT
	oldColMap := make(map[string]bool, len(oldTable.Columns))
	for _, col := range oldTable.Columns {
		oldColMap[col.Name] = true
	}
	var commonCols []string
	for _, col := range newTable.Columns {
		if oldColMap[col.Name] {
			commonCols = append(commonCols, fmt.Sprintf(`"%s"`, col.Name))
		}
	}

	if len(commonCols) > 0 {
		cols := strings.Join(commonCols, ", ")
		parts = append(parts, fmt.Sprintf(
			`INSERT INTO "%s" (%s) SELECT %s FROM "%s";`,
			tempTable.Name, cols, cols, oldTable.Name,
		))
	}

	parts = append(parts, fmt.Sprintf(`DROP TABLE "%s";`, oldTable.Name))
	parts = append(parts, fmt.Sprintf(`ALTER TABLE "%s" RENAME TO "%s";`, tempTable.Name, newTable.Name))

	// Recreate standalone indexes
	for _, index := range newTable.Indexes {
		if strings.HasPrefix(index.Name, "sqlite_") {
			continue
		}
		idxSQL := m.adapter.GenerateAddIndexSQL(index)
		if idxSQL != "" {
			parts = append(parts, idxSQL)
		}
	}

	parts = append(parts, "PRAGMA foreign_keys=ON;")

	return strings.Join(parts, "\n")
}

// hasSignificantSQLiteModifications checks if any ModifiedColumn in the table diff
// represents a real semantic type change (e.g., TEXT → INTEGER) rather than a
// cosmetic one (e.g., TEXT → VARCHAR(255)) for SQLite.
func (m *Migrator) hasSignificantSQLiteModifications(tableDiff types.TableDiff) bool {
	for _, col := range tableDiff.ModifiedColumns {
		oldNorm := m.adapter.MapColumnType(col.OldType)
		newNorm := m.adapter.MapColumnType(col.NewType)
		if oldNorm != newNorm {
			return true
		}
		// Also check for non-type changes (nullable, default, primary key, etc.)
		if col.OldColumn.Nullable != col.NewColumn.Nullable {
			return true
		}
		if col.OldColumn.Default != col.NewColumn.Default {
			return true
		}
		if col.OldColumn.IsPrimary != col.NewColumn.IsPrimary {
			return true
		}
		if col.OldColumn.IsUnique != col.NewColumn.IsUnique {
			return true
		}
		if col.OldColumn.ForeignKeyTable != col.NewColumn.ForeignKeyTable {
			return true
		}
		if col.OldColumn.ForeignKeyColumn != col.NewColumn.ForeignKeyColumn {
			return true
		}
	}
	return false
}

func (m *Migrator) GenerateEmptyMigration(ctx context.Context, name string) error {
	filename := m.fileUtils.GenerateMigrationFilename(name)
	filepath := filepath.Join(m.migrationsDir, filename)

	sqlContent := m.generateEmptyMigrationTemplate(name)

	if err := os.WriteFile(filepath, []byte(sqlContent), 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	fmt.Printf("Generated empty migration: %s\n", filename)
	return nil
}

func (m *Migrator) askUserConfirmation(message string) bool {
	return m.inputUtils.AskConfirmation(message, m.force)
}
