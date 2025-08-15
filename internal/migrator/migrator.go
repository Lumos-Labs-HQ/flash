package migrator

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Rana718/Graft/internal/config"
	"github.com/Rana718/Graft/internal/database"
	"github.com/Rana718/Graft/internal/schema"
	"github.com/Rana718/Graft/internal/types"
)

type Migrator struct {
	adapter       database.DatabaseAdapter
	schemaManager *schema.SchemaManager
	migrationsDir string
	schemaPath    string
	force         bool
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

	schemaManager := schema.NewSchemaManager(adapter)

	return &Migrator{
		adapter:       adapter,
		schemaManager: schemaManager,
		migrationsDir: cfg.MigrationsPath,
		schemaPath:    cfg.SchemaPath,
		force:         false,
	}, nil
}

func (m *Migrator) Close() error {
	return m.adapter.Close()
}

func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	return m.adapter.CreateMigrationsTable(ctx)
}

func (m *Migrator) applySingleMigration(ctx context.Context, migration types.Migration) error {
	log.Printf("Applying migration: %s", migration.ID)

	content, err := os.ReadFile(migration.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	if err := m.adapter.ExecuteMigration(ctx, string(content)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	checksum := fmt.Sprintf("%x", len(content))

	if err := m.adapter.RecordMigration(ctx, migration.ID, migration.Name, checksum); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	log.Printf("Successfully applied migration: %s", migration.ID)
	return nil
}

func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[string]*time.Time, error) {
	return m.adapter.GetAppliedMigrations(ctx)
}

func (m *Migrator) loadMigrationsFromDir() ([]types.Migration, error) {
	var migrations []types.Migration

	err := filepath.WalkDir(m.migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(d.Name(), ".sql") {
			return nil
		}

		migrationID := strings.TrimSuffix(d.Name(), ".sql")
		migrations = append(migrations, types.Migration{
			ID:       migrationID,
			Name:     migrationID,
			FilePath: path,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk migrations directory: %w", err)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})

	return migrations, nil
}

func (m *Migrator) hasConflicts(ctx context.Context, pendingMigrations []types.Migration) (bool, []types.MigrationConflict, error) {
	var allConflicts []types.MigrationConflict

	for _, migration := range pendingMigrations {
		conflicts, err := m.detectConflicts(ctx, migration)
		if err != nil {
			return false, nil, fmt.Errorf("failed to detect conflicts for migration %s: %w", migration.ID, err)
		}
		allConflicts = append(allConflicts, conflicts...)
	}

	return len(allConflicts) > 0, allConflicts, nil
}

func (m *Migrator) detectConflicts(ctx context.Context, migration types.Migration) ([]types.MigrationConflict, error) {
	var conflicts []types.MigrationConflict

	content, err := os.ReadFile(migration.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration file: %w", err)
	}

	migrationContent := string(content)

	// Look for ALTER TABLE ADD COLUMN statements with NOT NULL but no DEFAULT
	addColumnRegex := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+["']?(\w+)["']?\s+ADD\s+(?:COLUMN\s+)?["']?(\w+)["']?\s+[^;]*NOT\s+NULL`)
	matches := addColumnRegex.FindAllStringSubmatch(migrationContent, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			tableName := match[1]
			columnName := match[2]

			// Check if this line also contains DEFAULT - if so, skip it
			if strings.Contains(strings.ToUpper(match[0]), "DEFAULT") {
				continue
			}

			exists, err := m.adapter.CheckTableExists(ctx, tableName)
			if err != nil {
				log.Printf("Warning: Could not check if table %s exists: %v", tableName, err)
				continue
			}

			if exists {
				data, err := m.adapter.GetTableData(ctx, tableName)
				if err != nil {
					log.Printf("Warning: Could not check table data for %s: %v", tableName, err)
					continue
				}

				if len(data) > 0 {
					conflicts = append(conflicts, types.MigrationConflict{
						Type:        "not_null_constraint",
						TableName:   tableName,
						ColumnName:  columnName,
						Description: fmt.Sprintf("Adding NOT NULL column '%s' to table '%s' with existing data", columnName, tableName),
						Solutions: []string{
							"Add a DEFAULT value to the column",
							"Make the column nullable first, then update existing rows",
							"Reset the database if data loss is acceptable",
						},
					})
				}
			}
		}
	}

	return conflicts, nil
}

func (m *Migrator) cleanupBrokenMigrationRecords(ctx context.Context) error {
	return m.adapter.CleanupBrokenMigrationRecords(ctx)
}

func (m *Migrator) GenerateMigration(ctx context.Context, name string, schemaPath string) error {
	if schemaPath == "" {
		schemaPath = m.schemaPath
	}

	diff, err := m.schemaManager.GenerateSchemaDiff(ctx, schemaPath)
	if err != nil {
		return fmt.Errorf("failed to generate schema diff: %w", err)
	}

	if len(diff.NewTables) == 0 && len(diff.DroppedTables) == 0 && len(diff.ModifiedTables) == 0 {
		log.Println("No changes detected in schema")
		return nil
	}

	sqlContent := m.generateSQLFromDiff(diff)
	if sqlContent == "" {
		log.Println("No SQL generated from diff")
		return nil
	}

	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", timestamp, strings.ReplaceAll(name, " ", "_"))
	filepath := filepath.Join(m.migrationsDir, filename)

	if err := os.WriteFile(filepath, []byte(sqlContent), 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	log.Printf("Generated migration: %s", filename)
	return nil
}

func (m *Migrator) generateSQLFromDiff(diff *types.SchemaDiff) string {
	var sqlStatements []string

	// Generate SQL for new tables
	for _, table := range diff.NewTables {
		sql := m.adapter.GenerateCreateTableSQL(table)
		if sql != "" {
			sqlStatements = append(sqlStatements, sql)
		}
	}

	// Generate SQL for modified tables
	for _, tableDiff := range diff.ModifiedTables {
		// Add new columns with safety checks
		for _, column := range tableDiff.NewColumns {
			// For SQLite, we need to add a manual check since it doesn't support conditional DDL
			if strings.Contains(strings.ToLower(m.adapter.GenerateCreateTableSQL(types.SchemaTable{Name: "test"})), "if not exists") {
				// Adapter supports IF NOT EXISTS - use it directly
				sql := m.adapter.GenerateAddColumnSQL(tableDiff.Name, column)
				if sql != "" {
					sqlStatements = append(sqlStatements, sql)
				}
			} else {
				// For SQLite and other DBs that need application-level checks
				// Add a comment and then the SQL
				sqlStatements = append(sqlStatements, fmt.Sprintf("-- Adding column %s to table %s if it doesn't exist", column.Name, tableDiff.Name))
				sql := m.adapter.GenerateAddColumnSQL(tableDiff.Name, column)
				if sql != "" {
					sqlStatements = append(sqlStatements, sql)
				}
			}
		}

		// Drop columns
		for _, columnName := range tableDiff.DroppedColumns {
			sql := m.adapter.GenerateDropColumnSQL(tableDiff.Name, columnName)
			if sql != "" {
				sqlStatements = append(sqlStatements, sql)
			}
		}
	}

	// Generate SQL for dropped tables
	for _, tableName := range diff.DroppedTables {
		sqlStatements = append(sqlStatements, fmt.Sprintf("DROP TABLE IF EXISTS \"%s\";", tableName))
	}

	return strings.Join(sqlStatements, "\n\n")
}

func (m *Migrator) PullSchema(ctx context.Context) ([]types.SchemaTable, error) {
	return m.adapter.GetCurrentSchema(ctx)
}

func (m *Migrator) askUserConfirmation(message string) bool {
	if m.force {
		return true
	}

	fmt.Printf("%s (y/N): ", message)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}
