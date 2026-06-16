package migrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	return newMigratorInternal(cfg, false)
}

// NewMigratorForGenerate creates a Migrator for flash migrate — skips DB connection
// when a valid schema snapshot already exists, since generation only needs the snapshot + schema files.
func NewMigratorForGenerate(cfg *config.Config) (*Migrator, error) {
	snapshotPath := schema.SnapshotPath(cfg.MigrationsPath)
	snap, _ := schema.LoadSchemaSnapshot(snapshotPath)
	if snap != nil {
		// Snapshot exists — no DB needed for diff generation
		return newMigratorInternal(cfg, true)
	}
	return newMigratorInternal(cfg, false)
}

func newMigratorInternal(cfg *config.Config, skipConnect bool) (*Migrator, error) {
	adapter := database.NewAdapter(cfg.Database.Provider)

	if !skipConnect {
		dbURL, err := cfg.GetDatabaseURL()
		if err != nil {
			return nil, fmt.Errorf("failed to get database URL: %w", err)
		}
		if err := adapter.Connect(context.Background(), dbURL); err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
	}

	return &Migrator{
		adapter:       adapter,
		schemaManager: schema.NewSchemaManager(adapter),
		migrationsDir: cfg.MigrationsPath,
		schemaPath:    cfg.GetSchemaDir(),
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
	// ScyllaDB/Cassandra has no NOT NULL constraint enforcement — skip conflict detection entirely.
	if m.provider == "scylla" || m.provider == "scylladb" || m.provider == "cassandra" {
		return false, nil, nil
	}

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
	// Check for index changes and keyspace changes too, not just tables and enums.
	if len(diff.NewTables) == 0 && len(diff.DroppedTables) == 0 && len(diff.ModifiedTables) == 0 &&
		len(diff.NewEnums) == 0 && len(diff.DroppedEnums) == 0 && len(diff.ModifiedEnums) == 0 &&
		len(diff.NewIndexes) == 0 && len(diff.DroppedIndexes) == 0 &&
		len(diff.NewKeyspaces) == 0 && len(diff.DroppedKeyspaces) == 0 &&
		len(diff.NewUDTs) == 0 && len(diff.DroppedUDTs) == 0 &&
		len(diff.NewRawStatements) == 0 {
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
	targetTables, targetEnums, targetIndexes, targetKeyspaces, targetUDTs, targetRaw, err := m.schemaManager.ParseSchemaPathAllV2(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to parse target schema for snapshot: %w", err)
	}

	if err := schema.SaveSchemaSnapshotFullV3(snapshotPath, targetTables, targetEnums, targetIndexes, targetKeyspaces, targetUDTs, targetRaw); err != nil {
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
		case "scylla", "scylladb", "cassandra":
			// ScyllaDB uses keyspace-qualified names (ks.table). If the name
			// already has a dot, quote each part separately.
			if idx := strings.Index(tableName, "."); idx >= 0 {
				ks := strings.TrimSpace(tableName[:idx])
				tbl := strings.TrimSpace(tableName[idx+1:])
				return fmt.Sprintf(`DROP TABLE IF EXISTS "%s"."%s";`, strings.Trim(ks, `"`), strings.Trim(tbl, `"`))
			}
			return fmt.Sprintf(`DROP TABLE IF EXISTS "%s"."%s";`, m.adapter.QuoteIdentifier(tableName), tableName)
		default:
			switch m.provider {
			case "sqlite", "sqlite3":
				return fmt.Sprintf("DROP TABLE IF EXISTS \"%s\";", tableName)
			case "mysql":
				return fmt.Sprintf("DROP TABLE IF EXISTS `%s`;", tableName)
			case "clickhouse":
				return fmt.Sprintf("DROP TABLE IF EXISTS `%s`;", tableName)
			default:
				return fmt.Sprintf("DROP TABLE IF EXISTS \"%s\" CASCADE;", tableName)
			}
		}
	}

	dropIndexSQL := func(index types.SchemaIndex) string {
		switch m.provider {
		case "scylla", "scylladb", "cassandra":
			return m.adapter.GenerateDropIndexSQL(index)
		default:
			return fmt.Sprintf("DROP INDEX IF EXISTS \"%s\";", index.Name)
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
		} else if m.provider == "clickhouse" {
			// ClickHouse does not support user-defined enum types at the database level
			continue
		} else if m.provider == "scylla" || m.provider == "scylladb" || m.provider == "cassandra" {
			// ScyllaDB/Cassandra has no user-defined enum types
			continue
		}
	}

	// UP: Create new keyspaces FIRST (ScyllaDB/Cassandra) — UDTs and tables live inside keyspaces
	for _, ks := range diff.NewKeyspaces {
		ksSQL := fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS \"%s\" WITH REPLICATION = %s", ks.Name, ks.Replication)
		if ks.DurableWrites != nil && !*ks.DurableWrites {
			ksSQL += " AND DURABLE_WRITES = false"
		}
		ksSQL += ";"
		upStatements = append(upStatements, ksSQL)
		hasExecutableSQL = true
		downStatements = append([]string{fmt.Sprintf("DROP KEYSPACE IF EXISTS \"%s\";", ks.Name)}, downStatements...)
	}

	// UP: Create new CQL UDTs (ScyllaDB/Cassandra) — must be before tables that reference them
	for _, udt := range diff.NewUDTs {
		var fields []string
		for _, f := range udt.Fields {
			fields = append(fields, fmt.Sprintf("%s %s", f.Name, f.Type))
		}
		udtSQL := fmt.Sprintf("CREATE TYPE IF NOT EXISTS %s (%s);", udt.Name, strings.Join(fields, ", "))
		upStatements = append(upStatements, udtSQL)
		hasExecutableSQL = true
		downStatements = append([]string{fmt.Sprintf("DROP TYPE IF EXISTS %s;", udt.Name)}, downStatements...)
	}

	// UP: Emit raw passthrough statements (DOMAIN types, composite types, PARTITION OF, functions, triggers).
	// These are emitted before tables since DOMAINs/composite types may be referenced by table columns.
	for _, raw := range diff.NewRawStatements {
		upStatements = append(upStatements, raw)
		hasExecutableSQL = true
	}

	// UP: Create new tables and their indexes.
	// Defer MATERIALIZED VIEWs — they must be created AFTER the tables they reference.
	var viewUpStatements []string
	var viewDownStatements []string
	for _, table := range diff.NewTables {
		if len(table.Columns) == 1 && (strings.HasPrefix(table.Columns[0].Name, "/* VIEW */") || strings.HasPrefix(table.Columns[0].Name, "/* MATERIALIZED VIEW */")) {
			viewSQL := table.Columns[0].Type
			isMaterialized := strings.HasPrefix(table.Columns[0].Name, "/* MATERIALIZED VIEW */")
			// Inject IF NOT EXISTS for idempotent apply
			if !strings.Contains(strings.ToUpper(viewSQL), "IF NOT EXISTS") {
				if isMaterialized {
					viewSQL = strings.Replace(viewSQL, "CREATE MATERIALIZED VIEW", "CREATE MATERIALIZED VIEW IF NOT EXISTS", 1)
				} else {
					viewSQL = strings.Replace(viewSQL, "CREATE VIEW", "CREATE VIEW", 1)
				}
			}
			viewUpStatements = append(viewUpStatements, viewSQL)
			if isMaterialized {
				viewDownStatements = append(viewDownStatements, fmt.Sprintf("DROP MATERIALIZED VIEW IF EXISTS \"%s\";", table.Name))
			} else {
				viewDownStatements = append(viewDownStatements, fmt.Sprintf("DROP VIEW IF EXISTS \"%s\";", table.Name))
			}
			// Ensure referenced tables exist: parse the view SQL for table names
			// and emit CREATE TABLE IF NOT EXISTS for any that aren't in newTables.
			if m.provider == "scylla" || m.provider == "scylladb" || m.provider == "cassandra" {
				rawViewSQL := table.Columns[0].Type
				for _, refName := range extractRefTables(rawViewSQL) {
					if !isTableInNewTables(refName, diff.NewTables) {
						refTable := findTableInSchema(refName, m.schemaManager, m.schemaPath)
						if refTable != nil {
							sql := m.adapter.GenerateCreateTableSQL(*refTable)
							if sql != "" {
								upStatements = append(upStatements, sql)
								hasExecutableSQL = true
							}
						}
					}
				}
			}
			continue
		}

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
			downStatements = append([]string{dropIndexSQL(index)}, downStatements...)
		}
	}
	// Append views after all tables
	upStatements = append(upStatements, viewUpStatements...)
	hasExecutableSQL = hasExecutableSQL || len(viewUpStatements) > 0
	downStatements = append(viewDownStatements, downStatements...)

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
					recreateSQL := m.generateSQLiteTableRecreateSQL(tableDiff.OldTable, tableDiff.NewTable)
					if recreateSQL != "" {
						upStatements = append(upStatements, recreateSQL)
						hasExecutableSQL = true
						downRecreate := m.generateSQLiteTableRecreateSQL(tableDiff.NewTable, tableDiff.OldTable)
						downStatements = append([]string{downRecreate}, downStatements...)
					}
				}
			} else {
				for _, colDiff := range tableDiff.ModifiedColumns {
					// 1. Type change
					if colDiff.OldType != colDiff.NewType {
						sql := m.adapter.GenerateAlterColumnSQL(tableDiff.Name, colDiff.NewColumn, colDiff.OldType)
						if sql != "" {
							upStatements = append(upStatements, sql)
							hasExecutableSQL = true
							revertSQL := m.adapter.GenerateAlterColumnSQL(tableDiff.Name, colDiff.OldColumn, colDiff.NewType)
							if revertSQL != "" {
								downStatements = append([]string{revertSQL}, downStatements...)
							}
						}
					}

					// 2. Nullable change (PostgreSQL / MySQL)
					if colDiff.NullableChanged {
						var nullSQL, nullDownSQL string
						switch m.provider {
						case "postgresql", "postgres":
							if colDiff.NewColumn.Nullable {
								nullSQL = fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" DROP NOT NULL;", tableDiff.Name, colDiff.Name)
								nullDownSQL = fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" SET NOT NULL;", tableDiff.Name, colDiff.Name)
							} else {
								nullSQL = fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" SET NOT NULL;", tableDiff.Name, colDiff.Name)
								nullDownSQL = fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" DROP NOT NULL;", tableDiff.Name, colDiff.Name)
							}
						case "mysql":
							// MySQL MODIFY COLUMN already handles nullable in GenerateAlterColumnSQL
						}
						if nullSQL != "" {
							upStatements = append(upStatements, nullSQL)
							hasExecutableSQL = true
							downStatements = append([]string{nullDownSQL}, downStatements...)
						}
					}

					// 3. Default change (PostgreSQL / MySQL)
					if colDiff.DefaultChanged {
						var defSQL, defDownSQL string
						switch m.provider {
						case "postgresql", "postgres":
							if colDiff.NewColumn.Default != "" {
								defSQL = fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" SET DEFAULT %s;", tableDiff.Name, colDiff.Name, colDiff.NewColumn.Default)
								if colDiff.OldColumn.Default != "" {
									defDownSQL = fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" SET DEFAULT %s;", tableDiff.Name, colDiff.Name, colDiff.OldColumn.Default)
								} else {
									defDownSQL = fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" DROP DEFAULT;", tableDiff.Name, colDiff.Name)
								}
							} else {
								defSQL = fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" DROP DEFAULT;", tableDiff.Name, colDiff.Name)
								if colDiff.OldColumn.Default != "" {
									defDownSQL = fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" SET DEFAULT %s;", tableDiff.Name, colDiff.Name, colDiff.OldColumn.Default)
								}
							}
						case "mysql":
							// MySQL MODIFY COLUMN already handles default in GenerateAlterColumnSQL
						}
						if defSQL != "" {
							upStatements = append(upStatements, defSQL)
							hasExecutableSQL = true
							if defDownSQL != "" {
								downStatements = append([]string{defDownSQL}, downStatements...)
							}
						}
					}

					// 4. CHECK constraint change
					if colDiff.OldColumn.Check != colDiff.NewColumn.Check {
						constraintName := fmt.Sprintf("%s_%s_check", tableDiff.Name, colDiff.Name)
						switch m.provider {
						case "postgresql", "postgres":
							if colDiff.OldColumn.Check != "" {
								// Drop old CHECK
								dropCheck := fmt.Sprintf("ALTER TABLE \"%s\" DROP CONSTRAINT IF EXISTS \"%s\";", tableDiff.Name, constraintName)
								upStatements = append(upStatements, dropCheck)
								hasExecutableSQL = true
								if colDiff.NewColumn.Check == "" {
									downStatements = append([]string{fmt.Sprintf("ALTER TABLE \"%s\" ADD CONSTRAINT \"%s\" CHECK (%s);", tableDiff.Name, constraintName, colDiff.OldColumn.Check)}, downStatements...)
								}
							}
							if colDiff.NewColumn.Check != "" {
								// Add new CHECK
								addCheck := fmt.Sprintf("ALTER TABLE \"%s\" ADD CONSTRAINT \"%s\" CHECK (%s);", tableDiff.Name, constraintName, colDiff.NewColumn.Check)
								upStatements = append(upStatements, addCheck)
								hasExecutableSQL = true
								downStatements = append([]string{fmt.Sprintf("ALTER TABLE \"%s\" DROP CONSTRAINT IF EXISTS \"%s\";", tableDiff.Name, constraintName)}, downStatements...)
							}
						case "mysql":
							if colDiff.OldColumn.Check != "" {
								upStatements = append(upStatements, fmt.Sprintf("ALTER TABLE `%s` DROP CHECK `%s`;", tableDiff.Name, constraintName))
								hasExecutableSQL = true
							}
							if colDiff.NewColumn.Check != "" {
								upStatements = append(upStatements, fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` CHECK (%s);", tableDiff.Name, constraintName, colDiff.NewColumn.Check))
								hasExecutableSQL = true
							}
						}
					}

					// 5. UNIQUE change (PostgreSQL / MySQL — SQLite handled by table recreate)
					if !colDiff.OldColumn.IsUnique && colDiff.NewColumn.IsUnique {
						indexName := fmt.Sprintf("%s_%s_key", tableDiff.Name, colDiff.Name)
						switch m.provider {
						case "postgresql", "postgres":
							upStatements = append(upStatements, fmt.Sprintf("ALTER TABLE \"%s\" ADD CONSTRAINT \"%s\" UNIQUE (\"%s\");", tableDiff.Name, indexName, colDiff.Name))
							downStatements = append([]string{fmt.Sprintf("ALTER TABLE \"%s\" DROP CONSTRAINT IF EXISTS \"%s\";", tableDiff.Name, indexName)}, downStatements...)
						case "mysql":
							upStatements = append(upStatements, fmt.Sprintf("ALTER TABLE `%s` ADD UNIQUE INDEX `%s` (`%s`);", tableDiff.Name, indexName, colDiff.Name))
							downStatements = append([]string{fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`;", tableDiff.Name, indexName)}, downStatements...)
						}
						hasExecutableSQL = true
					} else if colDiff.OldColumn.IsUnique && !colDiff.NewColumn.IsUnique {
						indexName := fmt.Sprintf("%s_%s_key", tableDiff.Name, colDiff.Name)
						switch m.provider {
						case "postgresql", "postgres":
							upStatements = append(upStatements, fmt.Sprintf("ALTER TABLE \"%s\" DROP CONSTRAINT IF EXISTS \"%s\";", tableDiff.Name, indexName))
							downStatements = append([]string{fmt.Sprintf("ALTER TABLE \"%s\" ADD CONSTRAINT \"%s\" UNIQUE (\"%s\");", tableDiff.Name, indexName, colDiff.Name)}, downStatements...)
						case "mysql":
							upStatements = append(upStatements, fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`;", tableDiff.Name, indexName))
							downStatements = append([]string{fmt.Sprintf("ALTER TABLE `%s` ADD UNIQUE INDEX `%s` (`%s`);", tableDiff.Name, indexName, colDiff.Name)}, downStatements...)
						}
						hasExecutableSQL = true
					}

					// 6. GENERATED expression change (PostgreSQL only)
					if colDiff.GeneratedChanged {
						switch m.provider {
						case "postgresql", "postgres":
							// Drop the old generated column and re-add with new expression
							dropSQL := m.adapter.GenerateDropColumnSQL(tableDiff.Name, colDiff.Name)
							upStatements = append(upStatements, dropSQL)
							hasExecutableSQL = true
							addSQL := m.adapter.GenerateAddColumnSQL(tableDiff.Name, colDiff.NewColumn)
							if addSQL != "" {
								upStatements = append(upStatements, addSQL)
							}
							// DOWN: drop new and re-add old
							downStatements = append([]string{
								m.adapter.GenerateDropColumnSQL(tableDiff.Name, colDiff.Name),
								m.adapter.GenerateAddColumnSQL(tableDiff.Name, colDiff.OldColumn),
							}, downStatements...)
						case "mysql":
							// MySQL: MODIFY COLUMN with GENERATED ALWAYS AS
							genSQL := fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN `%s` %s GENERATED ALWAYS AS (%s) STORED;",
								tableDiff.Name, colDiff.Name, colDiff.NewColumn.Type, colDiff.NewColumn.Generated)
							upStatements = append(upStatements, genSQL)
							hasExecutableSQL = true
							genDown := fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN `%s` %s GENERATED ALWAYS AS (%s) STORED;",
								tableDiff.Name, colDiff.Name, colDiff.OldColumn.Type, colDiff.OldColumn.Generated)
							downStatements = append([]string{genDown}, downStatements...)
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
		if m.provider == "clickhouse" || m.provider == "sqlite" || m.provider == "sqlite3" || m.provider == "mysql" || m.provider == "scylla" || m.provider == "scylladb" || m.provider == "cassandra" {
			continue
		}
		upStatements = append(upStatements, fmt.Sprintf("DROP TYPE IF EXISTS \"%s\";", enumName))
		hasExecutableSQL = true
		downStatements = append([]string{fmt.Sprintf("-- Cannot restore dropped enum: %s", enumName)}, downStatements...)
	}

	// UP: Add values to existing enums (PostgreSQL only — ALTER TYPE ... ADD VALUE)
	for _, enumDiff := range diff.ModifiedEnums {
		for _, val := range enumDiff.AddValues {
			escaped := strings.ReplaceAll(val, "'", "''")
			if m.provider == "postgresql" || m.provider == "postgres" {
				sql := fmt.Sprintf("ALTER TYPE \"%s\" ADD VALUE IF NOT EXISTS '%s';", enumDiff.Name, escaped)
				upStatements = append(upStatements, sql)
				hasExecutableSQL = true
				downStatements = append([]string{fmt.Sprintf("-- Cannot remove enum value '%s' from \"%s\" (PostgreSQL limitation)", val, enumDiff.Name)}, downStatements...)
			}
		}
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
//  1. Create a temporary table with the new schema
//  2. Copy data from old to new (matching columns only)
//  3. Drop the old table
//  4. Rename the temporary table
//  5. Recreate indexes
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
		if col.OldColumn.Check != col.NewColumn.Check {
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

// extractRefTables extracts table names from a CREATE MATERIALIZED VIEW statement's SELECT clause.
func extractRefTables(viewSQL string) []string {
	upper := strings.ToUpper(viewSQL)
	var tables []string
	seen := map[string]bool{}
	fromRe := regexp.MustCompile(`(?i)\bFROM\s+(\S+)`)
	for _, m := range fromRe.FindAllStringSubmatch(viewSQL, -1) {
		name := strings.TrimSpace(m[1])
		if !seen[name] {
			seen[name] = true
			tables = append(tables, name)
		}
	}
	joinRe := regexp.MustCompile(`(?i)\bJOIN\s+(\S+)`)
	for _, m := range joinRe.FindAllStringSubmatch(upper, -1) {
		name := strings.TrimSpace(m[1])
		if !seen[name] {
			seen[name] = true
			tables = append(tables, name)
		}
	}
	return tables
}

// isTableInNewTables checks if a table is already being created in this migration.
func isTableInNewTables(name string, newTables []types.SchemaTable) bool {
	for _, t := range newTables {
		if strings.EqualFold(t.Name, name) {
			return true
		}
		if dotIdx := strings.LastIndex(t.Name, "."); dotIdx >= 0 {
			if strings.EqualFold(t.Name[dotIdx+1:], name) {
				return true
			}
		}
		if dotIdx := strings.LastIndex(name, "."); dotIdx >= 0 {
			if strings.EqualFold(t.Name, name[dotIdx+1:]) {
				return true
			}
		}
	}
	return false
}

// findTableInSchema finds a table from the schema files by name.
func findTableInSchema(name string, sm *schema.SchemaManager, schemaPath string) *types.SchemaTable {
	tables, _, _, _, _ := sm.ParseSchemaPathAll(schemaPath)
	for _, t := range tables {
		if strings.EqualFold(t.Name, name) {
			return &t
		}
		if dotIdx := strings.LastIndex(t.Name, "."); dotIdx >= 0 {
			if strings.EqualFold(t.Name[dotIdx+1:], name) {
				return &t
			}
		}
		if dotIdx := strings.LastIndex(name, "."); dotIdx >= 0 {
			if strings.EqualFold(t.Name, name[dotIdx+1:]) {
				return &t
			}
		}
	}
	return nil
}

func (m *Migrator) askUserConfirmation(message string) bool {
	return m.inputUtils.AskConfirmation(message, m.force)
}
