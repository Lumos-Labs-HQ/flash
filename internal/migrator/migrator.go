package migrator

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Rana718/Graft/internal/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Migrator struct {
	db             *pgxpool.Pool
	migrationsPath string
	backupPath     string
	force          bool
}

func NewMigrator(db *pgxpool.Pool, migrationsPath, backupPath string, force bool) *Migrator {
	return &Migrator{
		db:             db,
		migrationsPath: migrationsPath,
		backupPath:     backupPath,
		force:          force,
	}
}

// Creates migrations tracking table
func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	query := `
        CREATE TABLE IF NOT EXISTS _graft_migrations (
            id VARCHAR(255) PRIMARY KEY,
            checksum VARCHAR(64) NOT NULL,
            finished_at TIMESTAMP WITH TIME ZONE,
            migration_name VARCHAR(255) NOT NULL,
            logs TEXT,
            rolled_back_at TIMESTAMP WITH TIME ZONE,
            started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
            applied_steps_count INTEGER NOT NULL DEFAULT 0
        );`

	if _, err := m.db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	return m.ensureMigrationTableCompatibility(ctx)
}

// Updates migration table schema
func (m *Migrator) ensureMigrationTableCompatibility(ctx context.Context) error {
	var columnType string
	err := m.db.QueryRow(ctx, `
		SELECT data_type || CASE 
			WHEN character_maximum_length IS NOT NULL 
			THEN '(' || character_maximum_length || ')' 
			ELSE '' 
		END as column_type
		FROM information_schema.columns 
		WHERE table_name = '_graft_migrations' 
		AND column_name = 'id'
		AND table_schema = current_schema()
	`).Scan(&columnType)

	if err != nil {
		return nil
	}

	if strings.Contains(columnType, "character varying(36)") || strings.Contains(columnType, "varchar(36)") {
		_, err := m.db.Exec(ctx, "ALTER TABLE _graft_migrations ALTER COLUMN id TYPE VARCHAR(255)")
		if err != nil {
			return fmt.Errorf("failed to update migration ID column: %w", err)
		}
	}

	return nil
}

// Sanitizes migration name
func sanitizeMigrationName(name string) string {
	cleanName := strings.ToLower(name)
	cleanName = strings.ReplaceAll(cleanName, " ", "_")
	cleanName = regexp.MustCompile(`[^a-z0-9_]`).ReplaceAllString(cleanName, "_")
	cleanName = regexp.MustCompile(`_+`).ReplaceAllString(cleanName, "_")
	cleanName = strings.Trim(cleanName, "_")

	if cleanName == "" {
		cleanName = "migration"
	}
	return cleanName
}

// Cleans incomplete migration records
func (m *Migrator) cleanupBrokenMigrationRecords(ctx context.Context) error {
	result, err := m.db.Exec(ctx, `
		DELETE FROM _graft_migrations 
		WHERE finished_at IS NULL 
		AND started_at < NOW() - INTERVAL '1 hour'
	`)
	if err != nil {
		return fmt.Errorf("failed to cleanup incomplete migrations: %w", err)
	}

	if rowsAffected := result.RowsAffected(); rowsAffected > 0 {
		log.Printf("Cleaned up %d incomplete migration records", rowsAffected)
	}
	return nil
}

// Gets applied migrations
func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[string]*time.Time, error) {
	applied := make(map[string]*time.Time)
	rows, err := m.db.Query(ctx,
		`SELECT id, finished_at FROM _graft_migrations WHERE finished_at IS NOT NULL AND rolled_back_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var finishedAt *time.Time
		if err := rows.Scan(&id, &finishedAt); err != nil {
			return nil, err
		}
		applied[id] = finishedAt
	}
	return applied, nil
}

// Checks for migration conflicts
func (m *Migrator) hasConflicts(ctx context.Context, migrations []types.Migration) (bool, []types.MigrationConflict, error) {
	var conflicts []types.MigrationConflict

	for _, migration := range migrations {
		migrationSQL, err := parseMigrationFile(migration.FilePath)
		if err != nil {
			continue
		}

		checks := []func(context.Context, *types.MigrationSQL) ([]types.MigrationConflict, error){
			m.checkTableConflicts,
			m.checkNotNullConflicts,
			m.checkForeignKeyConflicts,
			m.checkUniqueConflicts,
		}

		for _, check := range checks {
			if conflictList, err := check(ctx, migrationSQL); err == nil {
				conflicts = append(conflicts, conflictList...)
			}
		}
	}
	return len(conflicts) > 0, conflicts, nil
}

// Checks table existence conflicts
func (m *Migrator) checkTableConflicts(ctx context.Context, migrationSQL *types.MigrationSQL) ([]types.MigrationConflict, error) {
	var conflicts []types.MigrationConflict
	if !strings.Contains(strings.ToUpper(migrationSQL.Up), "CREATE TABLE") {
		return conflicts, nil
	}

	tableRegex := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?"?(\w+)"?`)
	matches := tableRegex.FindAllStringSubmatch(migrationSQL.Up, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			tableName := match[1]
			var exists bool
			err := m.db.QueryRow(ctx,
				"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1 AND table_schema = 'public')",
				tableName).Scan(&exists)
			if err != nil {
				return nil, err
			}

			if exists && tableName != "_graft_migrations" {
				conflicts = append(conflicts, types.MigrationConflict{
					Type:        "table_exists",
					TableName:   tableName,
					Description: fmt.Sprintf("Table '%s' already exists in the database", tableName),
					Solutions: []string{
						"Drop the existing table manually if it's safe to do so",
						"Modify the migration to use 'CREATE TABLE IF NOT EXISTS'",
						"Rename the table in your schema to avoid the conflict",
					},
					Severity: "error",
				})
			}
		}
	}
	return conflicts, nil
}

// Checks NOT NULL constraint conflicts
func (m *Migrator) checkNotNullConflicts(ctx context.Context, migrationSQL *types.MigrationSQL) ([]types.MigrationConflict, error) {
	var conflicts []types.MigrationConflict
	addColumnRegex := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+"?(\w+)"?\s+ADD\s+COLUMN\s+"?(\w+)"?\s+([^;]+)`)
	matches := addColumnRegex.FindAllStringSubmatch(migrationSQL.Up, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			tableName := match[1]
			columnName := match[2]
			columnDef := strings.ToUpper(match[3])

			hasNotNull := strings.Contains(columnDef, "NOT NULL")
			hasDefault := strings.Contains(columnDef, "DEFAULT")

			if hasNotNull && !hasDefault {
				var rowCount int
				if err := m.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", tableName)).Scan(&rowCount); err != nil {
					continue
				}

				if rowCount > 0 {
					conflicts = append(conflicts, types.MigrationConflict{
						Type:        "not_null_constraint",
						TableName:   tableName,
						ColumnName:  columnName,
						Description: fmt.Sprintf("Cannot add NOT NULL column '%s' to table '%s' which contains %d existing rows", columnName, tableName, rowCount),
						Solutions: []string{
							fmt.Sprintf("Add a DEFAULT value: ALTER TABLE \"%s\" ADD COLUMN \"%s\" <type> DEFAULT <value> NOT NULL;", tableName, columnName),
							fmt.Sprintf("Make the column nullable first: ALTER TABLE \"%s\" ADD COLUMN \"%s\" <type>;", tableName, columnName),
							"Update existing rows first, then add NOT NULL constraint in separate migration",
						},
						Severity: "error",
					})
				}
			}
		}
	}
	return conflicts, nil
}

// Checks foreign key conflicts
func (m *Migrator) checkForeignKeyConflicts(ctx context.Context, migrationSQL *types.MigrationSQL) ([]types.MigrationConflict, error) {
	var conflicts []types.MigrationConflict
	fkRegex := regexp.MustCompile(`(?i)FOREIGN\s+KEY\s*\([^)]+\)\s*REFERENCES\s+"?(\w+)"?`)
	matches := fkRegex.FindAllStringSubmatch(migrationSQL.Up, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			referencedTable := match[1]
			var exists bool
			err := m.db.QueryRow(ctx,
				"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1 AND table_schema = 'public')",
				referencedTable).Scan(&exists)
			if err != nil {
				return nil, err
			}

			if !exists {
				conflicts = append(conflicts, types.MigrationConflict{
					Type:        "foreign_key",
					TableName:   referencedTable,
					Description: fmt.Sprintf("Foreign key references table '%s' which does not exist", referencedTable),
					Solutions: []string{
						fmt.Sprintf("Create table '%s' first in a separate migration", referencedTable),
						"Remove the foreign key constraint and add it later",
						"Check if the referenced table name is correct",
					},
					Severity: "error",
				})
			}
		}
	}
	return conflicts, nil
}

// Checks unique constraint conflicts
func (m *Migrator) checkUniqueConflicts(ctx context.Context, migrationSQL *types.MigrationSQL) ([]types.MigrationConflict, error) {
	var conflicts []types.MigrationConflict
	uniqueRegex := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+"?(\w+)"?\s+ADD\s+CONSTRAINT\s+\w+\s+UNIQUE\s*\(([^)]+)\)`)
	matches := uniqueRegex.FindAllStringSubmatch(migrationSQL.Up, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			tableName := match[1]
			columns := strings.Trim(match[2], " \"")

			query := fmt.Sprintf("SELECT COUNT(*) FROM (SELECT %s, COUNT(*) FROM \"%s\" GROUP BY %s HAVING COUNT(*) > 1) duplicates",
				columns, tableName, columns)

			var duplicateCount int
			if err := m.db.QueryRow(ctx, query).Scan(&duplicateCount); err != nil {
				continue
			}

			if duplicateCount > 0 {
				conflicts = append(conflicts, types.MigrationConflict{
					Type:        "unique_constraint",
					TableName:   tableName,
					ColumnName:  columns,
					Description: fmt.Sprintf("Cannot add UNIQUE constraint on column(s) '%s' in table '%s' - duplicate values exist", columns, tableName),
					Solutions: []string{
						fmt.Sprintf("Remove duplicate values first: DELETE FROM \"%s\" WHERE ...", tableName),
						"Update duplicate values to make them unique",
						"Use a partial unique index if only some rows should be unique",
					},
					Severity: "error",
				})
			}
		}
	}
	return conflicts, nil
}

// Gets user confirmation
func (m *Migrator) askUserConfirmation(message string) bool {
	if m.force {
		return true
	}
	fmt.Printf("🤔 %s (y/N): ", message)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "yes" || response == "y"
}

// Generates new migration file
func (m *Migrator) GenerateMigration(ctx context.Context, name string, schemaPath string) error {
	if err := os.MkdirAll(m.migrationsPath, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	cleanName := sanitizeMigrationName(name)
	migrationID := fmt.Sprintf("%s_%s", timestamp, cleanName)

	if len(migrationID) > 200 {
		maxNameLength := 200 - len(timestamp) - 1
		cleanName = cleanName[:maxNameLength]
		migrationID = fmt.Sprintf("%s_%s", timestamp, cleanName)
	}

	var upSQL string
	var hasChanges bool

	if schemaPath != "" {
		if _, err := os.Stat(schemaPath); err == nil {
			diff, err := m.generateSchemaDiff(ctx, schemaPath)
			if err != nil {
				return fmt.Errorf("failed to analyze schema changes: %w", err)
			}

			hasChanges = len(diff.NewTables) > 0 || len(diff.DroppedTables) > 0 || hasRealTableChanges(diff.ModifiedTables)
			if !hasChanges {
				return fmt.Errorf("no meaningful schema changes detected")
			}

			upSQL = generateMigrationSQL(diff)
		} else {
			return fmt.Errorf("schema file not found: %s", schemaPath)
		}
	} else {
		return fmt.Errorf("schema path is required for migration generation")
	}

	filename := fmt.Sprintf("%s.sql", migrationID)
	filePath := filepath.Join(m.migrationsPath, filename)

	migrationContent := fmt.Sprintf(`-- Migration: %s
-- Created: %s

%s`, name, time.Now().Format("2006-01-02 15:04:05"), upSQL)

	if err := os.WriteFile(filePath, []byte(migrationContent), 0644); err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}

	if err := m.createMigrationsTable(ctx); err != nil {
		log.Printf("Warning: Failed to create migrations table: %v", err)
	}

	log.Printf("Generated migration: %s", filePath)
	return nil
}

// Generates migration SQL from schema diff
func generateMigrationSQL(diff *types.SchemaDiff) string {
	var upStatements []string

	for _, table := range diff.NewTables {
		createSQL := generateCreateTableSQL(table)
		upStatements = append(upStatements, createSQL)
	}

	for _, tableName := range diff.DroppedTables {
		upStatements = append(upStatements, fmt.Sprintf("DROP TABLE IF EXISTS \"%s\" CASCADE;", tableName))
	}

	for _, tableDiff := range diff.ModifiedTables {
		for _, newCol := range tableDiff.NewColumns {
			alterSQL := fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN IF NOT EXISTS \"%s\" %s",
				tableDiff.Name, newCol.Name, formatColumnType(newCol))
			upStatements = append(upStatements, alterSQL+";")
		}

		for _, droppedCol := range tableDiff.DroppedColumns {
			upStatements = append(upStatements, fmt.Sprintf("ALTER TABLE \"%s\" DROP COLUMN IF EXISTS \"%s\";", tableDiff.Name, droppedCol))
		}

		for _, modCol := range tableDiff.ModifiedColumns {
			upStatements = append(upStatements, modCol.Changes...)
		}
	}

	return strings.Join(upStatements, "\n\n")
}

// Generates CREATE TABLE SQL
func generateCreateTableSQL(table types.SchemaTable) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("CREATE TABLE \"%s\" (", table.Name))

	for i, col := range table.Columns {
		colDef := fmt.Sprintf("    \"%s\" %s", col.Name, formatColumnType(col))
		if i < len(table.Columns)-1 {
			colDef += ","
		}
		lines = append(lines, colDef)
	}

	lines = append(lines, ");")
	return strings.Join(lines, "\n")
}

// Formats column type with constraints
func formatColumnType(col types.SchemaColumn) string {
	typeStr := col.Type
	if !col.Nullable {
		typeStr += " NOT NULL"
	}
	if col.Default != "" {
		typeStr += " DEFAULT " + col.Default
	}
	return typeStr
}

// Loads migrations from directory
func (m *Migrator) loadMigrationsFromDir() ([]types.Migration, error) {
	var migrations []types.Migration

	if _, err := os.Stat(m.migrationsPath); os.IsNotExist(err) {
		return migrations, nil
	}

	files, err := os.ReadDir(m.migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		filePath := filepath.Join(m.migrationsPath, file.Name())
		migrationSQL, err := parseMigrationFile(filePath)
		if err != nil {
			continue
		}

		migrationID := strings.TrimSuffix(file.Name(), ".sql")
		parts := strings.SplitN(migrationID, "_", 2)
		name := migrationID
		if len(parts) == 2 {
			name = strings.ReplaceAll(parts[1], "_", " ")
		}

		fileInfo, _ := file.Info()
		migration := types.Migration{
			ID:        migrationID,
			Name:      name,
			FilePath:  filePath,
			Checksum:  generateChecksum(migrationSQL.Up),
			CreatedAt: fileInfo.ModTime(),
		}

		migrations = append(migrations, migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})

	return migrations, nil
}

// Parses SQL migration file
func parseMigrationFile(filePath string) (*types.MigrationSQL, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	var sqlLines []string
	inHeader := true

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inHeader && (strings.HasPrefix(trimmed, "-- Migration:") ||
			strings.HasPrefix(trimmed, "-- Created:") ||
			trimmed == "") {
			if strings.HasPrefix(trimmed, "-- Created:") {
				inHeader = false
			}
			continue
		}

		inHeader = false
		sqlLines = append(sqlLines, line)
	}

	upSQL := strings.TrimSpace(strings.Join(sqlLines, "\n"))
	return &types.MigrationSQL{Up: upSQL}, nil
}

// Generates checksum for content
func generateChecksum(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// parseSchemaFile parses a schema file and returns table definitions
func parseSchemaFile(schemaPath string) ([]types.SchemaTable, error) {
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}

	var tables []types.SchemaTable
	lines := strings.Split(string(content), "\n")

	var currentTable *types.SchemaTable
	inTableDef := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		upperLine := strings.ToUpper(line)

		// Detect CREATE TABLE
		if strings.HasPrefix(upperLine, "CREATE TABLE") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				tableName := strings.Trim(parts[2], "();")
				currentTable = &types.SchemaTable{
					Name:    tableName,
					Columns: []types.SchemaColumn{},
					Indexes: []types.SchemaIndex{},
				}
				inTableDef = true
			}
		} else if inTableDef && strings.Contains(line, ");") {

			if currentTable != nil {
				tables = append(tables, *currentTable)
				currentTable = nil
			}
			inTableDef = false
		} else if inTableDef && currentTable != nil {
			if col := ParseColumnDefinition(line); col.Name != "" {
				currentTable.Columns = append(currentTable.Columns, col)
			}
		}
	}

	return tables, nil
}

// Gets current database schema
func (m *Migrator) getCurrentSchema(ctx context.Context) ([]types.SchemaTable, error) {
	var tables []types.SchemaTable

	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		AND table_name != '_graft_migrations'
		ORDER BY table_name`

	rows, err := m.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}
		tableNames = append(tableNames, tableName)
	}

	for _, tableName := range tableNames {
		columns, err := m.getTableColumns(ctx, tableName)
		if err != nil {
			continue
		}

		tables = append(tables, types.SchemaTable{
			Name:    tableName,
			Columns: columns,
			Indexes: []types.SchemaIndex{},
		})
	}

	return tables, nil
}

// Gets columns for specific table
func (m *Migrator) getTableColumns(ctx context.Context, tableName string) ([]types.SchemaColumn, error) {
	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position`

	rows, err := m.db.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []types.SchemaColumn
	for rows.Next() {
		var colName, dataType, isNullable string
		var colDefault *string

		if err := rows.Scan(&colName, &dataType, &isNullable, &colDefault); err != nil {
			continue
		}

		col := types.SchemaColumn{
			Name:     colName,
			Type:     dataType,
			Nullable: isNullable == "YES",
		}

		if colDefault != nil {
			col.Default = *colDefault
		}

		columns = append(columns, col)
	}

	return columns, nil
}

// Generates schema diff between current and target
func (m *Migrator) generateSchemaDiff(ctx context.Context, targetSchemaPath string) (*types.SchemaDiff, error) {
	currentSchema, err := m.getCurrentSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current schema: %w", err)
	}

	targetSchema, err := parseSchemaFile(targetSchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target schema: %w", err)
	}

	diff := &types.SchemaDiff{
		NewTables:      []types.SchemaTable{},
		DroppedTables:  []string{},
		ModifiedTables: []types.TableDiff{},
		NewIndexes:     []types.SchemaIndex{},
		DroppedIndexes: []string{},
	}

	currentTables := make(map[string]types.SchemaTable)
	for _, table := range currentSchema {
		currentTables[table.Name] = table
	}

	targetTables := make(map[string]types.SchemaTable)
	for _, table := range targetSchema {
		targetTables[table.Name] = table
	}

	for _, targetTable := range targetSchema {
		if _, exists := currentTables[targetTable.Name]; !exists {
			diff.NewTables = append(diff.NewTables, targetTable)
		}
	}

	for _, currentTable := range currentSchema {
		if _, exists := targetTables[currentTable.Name]; !exists {
			diff.DroppedTables = append(diff.DroppedTables, currentTable.Name)
		}
	}

	for tableName, targetTable := range targetTables {
		if currentTable, exists := currentTables[tableName]; exists {
			tableDiff := compareTableColumns(currentTable, targetTable)
			if len(tableDiff.NewColumns) > 0 || len(tableDiff.DroppedColumns) > 0 || len(tableDiff.ModifiedColumns) > 0 {
				diff.ModifiedTables = append(diff.ModifiedTables, tableDiff)
			}
		}
	}

	return diff, nil
}
