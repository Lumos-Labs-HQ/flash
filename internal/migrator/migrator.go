package migrator

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migration represents a single database migration
type Migration struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Applied   bool       `json:"applied"`
	AppliedAt *time.Time `json:"applied_at,omitempty"`
	FilePath  string     `json:"file_path"`
	Checksum  string     `json:"checksum"`
	CreatedAt time.Time  `json:"created_at"`
}

// MigrationFile represents the structure of a migration SQL file
type MigrationSQL struct {
	Up   string
	Down string
}

// Migrator handles database migrations
type Migrator struct {
	db             *pgxpool.Pool
	migrationsPath string
	backupPath     string
	force          bool
}

// BackupData represents backup information
type BackupData struct {
	Timestamp string                 `json:"timestamp"`
	Version   string                 `json:"version"`
	Tables    map[string]interface{} `json:"tables"`
	Comment   string                 `json:"comment"`
}

// MigrationStatus represents migration status information
type MigrationStatus struct {
	TotalMigrations   int                   `json:"total_migrations"`
	AppliedMigrations int                   `json:"applied_migrations"`
	PendingMigrations int                   `json:"pending_migrations"`
	Migrations        []MigrationStatusItem `json:"migrations"`
	DatabaseStatus    string                `json:"database_status"`
}

// MigrationStatusItem represents individual migration status
type MigrationStatusItem struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	AppliedAt *time.Time `json:"applied_at,omitempty"`
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *pgxpool.Pool, migrationsPath, backupPath string, force bool) *Migrator {
	return &Migrator{
		db:             db,
		migrationsPath: migrationsPath,
		backupPath:     backupPath,
		force:          force,
	}
}

// createMigrationsTable creates the Graft-style migrations table
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
        );
    `
	_, err := m.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Check if we need to alter existing table for larger id column
	if err := m.ensureMigrationTableCompatibility(ctx); err != nil {
		return fmt.Errorf("failed to update migrations table: %w", err)
	}

	return nil
}

// ensureMigrationTableCompatibility updates the migrations table if needed
func (m *Migrator) ensureMigrationTableCompatibility(ctx context.Context) error {
	// Check current column size
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
		// Table doesn't exist yet, that's fine
		return nil
	}

	// If column is VARCHAR(36) or smaller, update it
	if strings.Contains(columnType, "character varying(36)") || strings.Contains(columnType, "varchar(36)") {
		log.Println("🔧 Updating migrations table to support longer migration IDs...")
		_, err := m.db.Exec(ctx, "ALTER TABLE _graft_migrations ALTER COLUMN id TYPE VARCHAR(255)")
		if err != nil {
			return fmt.Errorf("failed to update migration ID column: %w", err)
		}
		log.Println("✅ Migrations table updated successfully")
	}

	return nil
}

// sanitizeMigrationName cleans and validates migration names
func sanitizeMigrationName(name string) string {
	// Convert to lowercase and replace spaces with underscores
	cleanName := strings.ToLower(name)
	cleanName = strings.ReplaceAll(cleanName, " ", "_")

	// Remove special characters that could cause issues, keep only alphanumeric and underscores
	reg := regexp.MustCompile(`[^a-z0-9_]`)
	cleanName = reg.ReplaceAllString(cleanName, "_")

	// Remove consecutive underscores
	reg = regexp.MustCompile(`_+`)
	cleanName = reg.ReplaceAllString(cleanName, "_")

	// Remove leading/trailing underscores
	cleanName = strings.Trim(cleanName, "_")

	// Ensure it's not empty
	if cleanName == "" {
		cleanName = "migration"
	}

	return cleanName
}

func (m *Migrator) cleanupBrokenMigrationRecords(ctx context.Context) error {
	result, err := m.db.Exec(ctx, `
		DELETE FROM _graft_migrations 
		WHERE finished_at IS NULL 
		AND started_at < NOW() - INTERVAL '1 hour'
	`)
	if err != nil {
		return fmt.Errorf("failed to cleanup incomplete migrations: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("🧹 Cleaned up %d incomplete migration records", rowsAffected)
	}

	return nil
}

func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[string]*time.Time, error) {
	applied := make(map[string]*time.Time)

	query := `SELECT id, finished_at FROM _graft_migrations WHERE finished_at IS NOT NULL AND rolled_back_at IS NULL`
	rows, err := m.db.Query(ctx, query)
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

type MigrationConflict struct {
	Type        string
	TableName   string
	ColumnName  string
	Description string
	Solutions   []string
	Severity    string
}

func (m *Migrator) hasConflicts(ctx context.Context, migrations []Migration) (bool, []MigrationConflict, error) {
	var conflicts []MigrationConflict

	for _, migration := range migrations {
		migrationSQL, err := parseMigrationFile(migration.FilePath)
		if err != nil {
			continue
		}

		// Check for table existence conflicts
		if conflicts1, err := m.checkTableConflicts(ctx, migrationSQL); err == nil {
			conflicts = append(conflicts, conflicts1...)
		}

		// Check for NOT NULL column conflicts
		if conflicts2, err := m.checkNotNullConflicts(ctx, migrationSQL); err == nil {
			conflicts = append(conflicts, conflicts2...)
		}

		// Check for foreign key conflicts
		if conflicts3, err := m.checkForeignKeyConflicts(ctx, migrationSQL); err == nil {
			conflicts = append(conflicts, conflicts3...)
		}

		// Check for unique constraint conflicts
		if conflicts4, err := m.checkUniqueConflicts(ctx, migrationSQL); err == nil {
			conflicts = append(conflicts, conflicts4...)
		}
	}

	return len(conflicts) > 0, conflicts, nil
}

// checkTableConflicts checks for table existence conflicts
func (m *Migrator) checkTableConflicts(ctx context.Context, migrationSQL *MigrationSQL) ([]MigrationConflict, error) {
	var conflicts []MigrationConflict

	if strings.Contains(strings.ToUpper(migrationSQL.Up), "CREATE TABLE") {
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
					conflicts = append(conflicts, MigrationConflict{
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
	}

	return conflicts, nil
}

// checkNotNullConflicts checks for NOT NULL column addition conflicts
func (m *Migrator) checkNotNullConflicts(ctx context.Context, migrationSQL *MigrationSQL) ([]MigrationConflict, error) {
	var conflicts []MigrationConflict

	// Look for ALTER TABLE ADD COLUMN statements with NOT NULL
	addColumnRegex := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+"?(\w+)"?\s+ADD\s+COLUMN\s+"?(\w+)"?\s+([^;]+)`)
	matches := addColumnRegex.FindAllStringSubmatch(migrationSQL.Up, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			tableName := match[1]
			columnName := match[2]
			columnDef := strings.ToUpper(match[3])

			// Check if the column is NOT NULL and no default is provided
			hasNotNull := strings.Contains(columnDef, "NOT NULL")
			hasDefault := strings.Contains(columnDef, "DEFAULT")

			if hasNotNull && !hasDefault {
				// Check if the table has existing data
				var rowCount int
				err := m.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", tableName)).Scan(&rowCount)
				if err != nil {
					// If table doesn't exist yet, it's not a conflict
					continue
				}

				if rowCount > 0 {
					conflicts = append(conflicts, MigrationConflict{
						Type:        "not_null_constraint",
						TableName:   tableName,
						ColumnName:  columnName,
						Description: fmt.Sprintf("Cannot add NOT NULL column '%s' to table '%s' which contains %d existing rows", columnName, tableName, rowCount),
						Solutions: []string{
							fmt.Sprintf("Add a DEFAULT value: ALTER TABLE \"%s\" ADD COLUMN \"%s\" <type> DEFAULT <value> NOT NULL;", tableName, columnName),
							fmt.Sprintf("Make the column nullable first: ALTER TABLE \"%s\" ADD COLUMN \"%s\" <type>;", tableName, columnName),
							"Update existing rows first, then add NOT NULL constraint in separate migration",
							"Consider if this column really needs to be NOT NULL for existing data",
						},
						Severity: "error",
					})
				}
			}
		}
	}

	return conflicts, nil
}

// checkForeignKeyConflicts checks for foreign key constraint conflicts
func (m *Migrator) checkForeignKeyConflicts(ctx context.Context, migrationSQL *MigrationSQL) ([]MigrationConflict, error) {
	var conflicts []MigrationConflict

	// Look for FOREIGN KEY constraints
	fkRegex := regexp.MustCompile(`(?i)FOREIGN\s+KEY\s*\([^)]+\)\s*REFERENCES\s+"?(\w+)"?`)
	matches := fkRegex.FindAllStringSubmatch(migrationSQL.Up, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			referencedTable := match[1]

			// Check if referenced table exists
			var exists bool
			err := m.db.QueryRow(ctx,
				"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1 AND table_schema = 'public')",
				referencedTable).Scan(&exists)
			if err != nil {
				return nil, err
			}

			if !exists {
				conflicts = append(conflicts, MigrationConflict{
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

// checkUniqueConflicts checks for unique constraint conflicts
func (m *Migrator) checkUniqueConflicts(ctx context.Context, migrationSQL *MigrationSQL) ([]MigrationConflict, error) {
	var conflicts []MigrationConflict

	// Look for UNIQUE constraints on existing tables
	uniqueRegex := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+"?(\w+)"?\s+ADD\s+CONSTRAINT\s+\w+\s+UNIQUE\s*\(([^)]+)\)`)
	matches := uniqueRegex.FindAllStringSubmatch(migrationSQL.Up, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			tableName := match[1]
			columns := strings.Trim(match[2], " \"")

			// Check if table has existing data with duplicates
			query := fmt.Sprintf("SELECT COUNT(*) FROM (SELECT %s, COUNT(*) FROM \"%s\" GROUP BY %s HAVING COUNT(*) > 1) duplicates",
				columns, tableName, columns)

			var duplicateCount int
			err := m.db.QueryRow(ctx, query).Scan(&duplicateCount)
			if err != nil {
				// Table might not exist or query might be invalid, skip
				continue
			}

			if duplicateCount > 0 {
				conflicts = append(conflicts, MigrationConflict{
					Type:        "unique_constraint",
					TableName:   tableName,
					ColumnName:  columns,
					Description: fmt.Sprintf("Cannot add UNIQUE constraint on column(s) '%s' in table '%s' - duplicate values exist", columns, tableName),
					Solutions: []string{
						fmt.Sprintf("Remove duplicate values first: DELETE FROM \"%s\" WHERE ...", tableName),
						"Update duplicate values to make them unique",
						"Consider if the unique constraint is necessary",
						"Use a partial unique index if only some rows should be unique",
					},
					Severity: "error",
				})
			}
		}
	}

	return conflicts, nil
}

// askUserConfirmation prompts user for confirmation
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

// createBackup creates a backup of the database

// createBackup creates a backup of the database
func (m *Migrator) createBackup(ctx context.Context, comment string) (string, error) {
	log.Println("📦 Creating database backup...")

	if err := os.MkdirAll(m.backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get applied migrations: %w", err)
	}

	backup := BackupData{
		Timestamp: time.Now().Format("2006-01-02_15-04-05"),
		Version:   fmt.Sprintf("%d_migrations", len(applied)),
		Tables:    make(map[string]interface{}),
		Comment:   comment,
	}

	tables, err := m.getAllTableNames(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get table names: %w", err)
	}

	// Check if database has any meaningful data (excluding _graft_migrations)
	hasData := false
	for _, table := range tables {
		if table == "_graft_migrations" {
			continue
		}

		var count int
		err := m.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			log.Printf("⚠️  Warning: Failed to count rows in table %s: %v", table, err)
			continue
		}

		if count > 0 {
			hasData = true
			break
		}
	}

	// If no data exists and this isn't a forced backup, skip
	if !hasData && !strings.Contains(comment, "Manual backup") && !strings.Contains(comment, "Pre-reset") {
		log.Println("ℹ️  No data found in database, skipping backup creation")
		return "", nil
	}

	for _, table := range tables {
		rows, err := m.db.Query(ctx, fmt.Sprintf("SELECT * FROM %s", table))
		if err != nil {
			log.Printf("⚠️  Warning: Failed to backup table %s: %v", table, err)
			continue
		}

		fieldDescriptions := rows.FieldDescriptions()
		columns := make([]string, len(fieldDescriptions))
		for i, fd := range fieldDescriptions {
			columns[i] = string(fd.Name)
		}

		var tableData []map[string]interface{}

		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				log.Printf("⚠️  Warning: Failed to read row from %s: %v", table, err)
				continue
			}

			rowData := make(map[string]interface{})
			for i, column := range columns {
				rowData[column] = values[i]
			}
			tableData = append(tableData, rowData)
		}
		rows.Close()

		backup.Tables[table] = map[string]interface{}{
			"columns": columns,
			"data":    tableData,
		}

		log.Printf("✅ Backed up table %s with %d rows", table, len(tableData))
	}

	filename := fmt.Sprintf("backup_%s.json", backup.Timestamp)
	backupPath := filepath.Join(m.backupPath, filename)

	file, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(backup); err != nil {
		return "", fmt.Errorf("failed to write backup data: %w", err)
	}

	log.Printf("✅ Database backup created: %s", backupPath)
	return backupPath, nil
}

// getAllTableNames returns all table names in the database
func (m *Migrator) getAllTableNames(ctx context.Context) ([]string, error) {
	query := `
        SELECT table_name 
        FROM information_schema.tables 
        WHERE table_schema = 'public' 
        AND table_type = 'BASE TABLE'
        ORDER BY table_name
    `

	rows, err := m.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// GenerateMigration creates a new migration file with actual SQL from schema diffing
func (m *Migrator) GenerateMigration(ctx context.Context, name string, schemaPath string) error {
	if err := os.MkdirAll(m.migrationsPath, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	cleanName := sanitizeMigrationName(name)
	migrationID := fmt.Sprintf("%s_%s", timestamp, cleanName)

	// Ensure migration ID doesn't exceed database limits (max 200 chars for safety)
	if len(migrationID) > 200 {
		maxNameLength := 200 - len(timestamp) - 1 // -1 for underscore
		cleanName = cleanName[:maxNameLength]
		migrationID = fmt.Sprintf("%s_%s", timestamp, cleanName)
		log.Printf("⚠️  Migration name truncated to fit database limits: %s", cleanName)
	}

	var upSQL string
	var hasChanges bool

	// Generate schema diff if schema file exists
	if schemaPath != "" {
		if _, err := os.Stat(schemaPath); err == nil {
			log.Printf("🔍 Analyzing schema changes...")
			diff, err := m.generateSchemaDiff(ctx, schemaPath)
			if err != nil {
				log.Printf("⚠️  Warning: Failed to generate schema diff: %v", err)
				return fmt.Errorf("failed to analyze schema changes: %w", err)
			} else {
				log.Printf("🔄 Schema diff: %d new tables, %d dropped tables, %d modified tables",
					len(diff.NewTables), len(diff.DroppedTables), len(diff.ModifiedTables))

				// Check if there are any actual changes
				hasChanges = len(diff.NewTables) > 0 || len(diff.DroppedTables) > 0 || hasRealTableChanges(diff.ModifiedTables)

				if !hasChanges {
					fmt.Println("ℹ️  No meaningful schema changes detected - skipping migration creation")
					fmt.Println("💡 Only create migrations when you have actual schema changes to apply")
					return fmt.Errorf("no meaningful schema changes detected")
				}

				for _, table := range diff.NewTables {
					log.Printf("  + New table: %s", table.Name)
				}
				for _, table := range diff.ModifiedTables {
					log.Printf("  ~ Modified table: %s (%d new cols, %d dropped cols)",
						table.Name, len(table.NewColumns), len(table.DroppedColumns))
					for _, col := range table.NewColumns {
						log.Printf("    + New column: %s %s", col.Name, col.Type)
					}
				}

				upSQL, _ = generateMigrationSQL(diff)
			}
		} else {
			return fmt.Errorf("schema file not found: %s", schemaPath)
		}
	} else {
		return fmt.Errorf("schema path is required for migration generation")
	}

	// Create SQL migration file
	filename := fmt.Sprintf("%s.sql", migrationID)
	filePath := filepath.Join(m.migrationsPath, filename)

	migrationContent := fmt.Sprintf(`-- Migration: %s
-- Created: %s

%s
`, name, time.Now().Format("2006-01-02 15:04:05"), upSQL)

	if err := os.WriteFile(filePath, []byte(migrationContent), 0644); err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}

	// Store migration metadata in tracking table
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		log.Printf("⚠️  Warning: Failed to create migrations table: %v", err)
	}

	log.Printf("✨ Generated migration: %s", filePath)
	if strings.Contains(upSQL, "CREATE TABLE") || strings.Contains(upSQL, "ALTER TABLE") {
		log.Printf("📋 Migration contains schema changes")
	} else {
		log.Printf("📝 Edit the migration file to add your SQL commands")
	}

	return nil
}

// generateMigrationSQL generates SQL from schema diff (Prisma-style)
func generateMigrationSQL(diff *SchemaDiff) (string, string) {
	var upStatements []string

	// Generate CREATE TABLE statements for new tables only
	for _, table := range diff.NewTables {
		createSQL := generateCreateTableSQL(table)
		upStatements = append(upStatements, fmt.Sprintf("-- CreateTable\n%s", createSQL))
	}

	// Generate DROP TABLE statements for dropped tables
	for _, tableName := range diff.DroppedTables {
		upStatements = append(upStatements, fmt.Sprintf("-- DropTable\nDROP TABLE IF EXISTS \"%s\" CASCADE;", tableName))
	}

	// Generate ALTER TABLE statements for modified tables
	for _, tableDiff := range diff.ModifiedTables {
		// Add new columns - use clean format
		for _, newCol := range tableDiff.NewColumns {
			alterSQL := fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN \"%s\" %s",
				tableDiff.Name, newCol.Name, formatColumnType(newCol))
			upStatements = append(upStatements, fmt.Sprintf("-- AddColumn\n%s;", alterSQL))
		}

		// Drop columns
		for _, droppedCol := range tableDiff.DroppedColumns {
			upStatements = append(upStatements, fmt.Sprintf("-- DropColumn\nALTER TABLE \"%s\" DROP COLUMN IF EXISTS \"%s\";", tableDiff.Name, droppedCol))
		}

		// Modify columns - only include meaningful changes
		for _, modCol := range tableDiff.ModifiedColumns {
			for _, change := range modCol.Changes {
				// Determine the type of change for better comments
				changeType := "AlterColumn"
				if strings.Contains(change, "TYPE") {
					changeType = "AlterColumnType"
				} else if strings.Contains(change, "NOT NULL") {
					changeType = "AlterColumnNullability"
				} else if strings.Contains(change, "DEFAULT") {
					changeType = "AlterColumnDefault"
				}

				upStatements = append(upStatements, fmt.Sprintf("-- %s\n%s", changeType, change))
			}
		}
	}

	upSQL := strings.Join(upStatements, "\n\n")

	return upSQL, ""
}

// generateCreateTableSQL generates CREATE TABLE SQL from SchemaTable
func generateCreateTableSQL(table SchemaTable) string {
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

// formatColumnType formats a column type with constraints
func formatColumnType(col SchemaColumn) string {
	typeStr := col.Type

	if !col.Nullable {
		typeStr += " NOT NULL"
	}

	if col.Default != "" {
		typeStr += " DEFAULT " + col.Default
	}

	return typeStr
}

// loadMigrationsFromDir loads migrations from directory (SQL files)
func (m *Migrator) loadMigrationsFromDir() ([]Migration, error) {
	var migrations []Migration

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
			log.Printf("⚠️  Warning: Failed to parse migration file %s: %v", file.Name(), err)
			continue
		}

		// Extract migration ID from filename (remove .sql extension)
		migrationID := strings.TrimSuffix(file.Name(), ".sql")

		// Extract name from ID (everything after the timestamp)
		parts := strings.SplitN(migrationID, "_", 2)
		name := migrationID
		if len(parts) == 2 {
			name = strings.ReplaceAll(parts[1], "_", " ")
		}

		fileInfo, _ := file.Info()

		migration := Migration{
			ID:        migrationID,
			Name:      name,
			FilePath:  filePath,
			Checksum:  generateChecksum(migrationSQL.Up),
			CreatedAt: fileInfo.ModTime(),
		}

		migrations = append(migrations, migration)
	}

	// Sort migrations by creation time (filename timestamp)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})

	return migrations, nil
}

// parseMigrationFile parses a SQL migration file
func parseMigrationFile(filePath string) (*MigrationSQL, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// New format - everything after header comments is the migration SQL
	var sqlLines []string
	inHeader := true

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip header comments (lines starting with -- Migration: or -- Created:)
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

	return &MigrationSQL{
		Up:   upSQL,
		Down: "", // New format doesn't have down migrations
	}, nil
}

// generateChecksum generates a checksum for content
func generateChecksum(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// parseSchemaFile parses a schema file and returns table definitions
func parseSchemaFile(schemaPath string) ([]SchemaTable, error) {
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}

	var tables []SchemaTable
	lines := strings.Split(string(content), "\n")

	var currentTable *SchemaTable
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
				currentTable = &SchemaTable{
					Name:    tableName,
					Columns: []SchemaColumn{},
					Indexes: []SchemaIndex{},
				}
				inTableDef = true
			}
		} else if inTableDef && strings.Contains(line, ");") {
			// End of table definition
			if currentTable != nil {
				tables = append(tables, *currentTable)
				currentTable = nil
			}
			inTableDef = false
		} else if inTableDef && currentTable != nil {
			// Parse column definition
			if col := parseColumnDefinition(line); col.Name != "" {
				currentTable.Columns = append(currentTable.Columns, col)
			}
		}
	}

	return tables, nil
}

// parseColumnDefinition parses a column definition line
func parseColumnDefinition(line string) SchemaColumn {
	line = strings.TrimSpace(strings.TrimSuffix(line, ","))
	if line == "" || strings.HasPrefix(strings.ToUpper(line), "PRIMARY KEY") ||
		strings.HasPrefix(strings.ToUpper(line), "FOREIGN KEY") ||
		strings.HasPrefix(strings.ToUpper(line), "CONSTRAINT") {
		return SchemaColumn{}
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return SchemaColumn{}
	}

	col := SchemaColumn{
		Name:     parts[0],
		Nullable: true,
	}

	// Parse type - handle compound types like "SERIAL PRIMARY KEY"
	upperLine := strings.ToUpper(line)

	// Extract base type
	if strings.Contains(upperLine, "SERIAL") {
		if strings.Contains(upperLine, "PRIMARY KEY") {
			col.Type = "SERIAL PRIMARY KEY"
		} else {
			col.Type = "SERIAL"
		}
	} else {
		// Find the type (second word, but might be compound)
		typeStart := 1
		col.Type = parts[typeStart]

		// Handle VARCHAR(255), etc.
		if typeStart+1 < len(parts) && strings.HasPrefix(parts[typeStart+1], "(") {
			col.Type += parts[typeStart+1]
		}

		// Add constraints to type
		if strings.Contains(upperLine, "PRIMARY KEY") {
			col.Type += " PRIMARY KEY"
		}
		if strings.Contains(upperLine, "UNIQUE") && !strings.Contains(upperLine, "PRIMARY KEY") {
			col.Type += " UNIQUE"
		}
	}

	// Check for NOT NULL
	if strings.Contains(upperLine, "NOT NULL") {
		col.Nullable = false
	}

	// Extract default value
	if strings.Contains(upperLine, "DEFAULT") {
		defaultIdx := strings.Index(upperLine, "DEFAULT")
		remaining := line[defaultIdx+7:]
		parts := strings.Fields(remaining)
		if len(parts) > 0 {
			col.Default = parts[0]
			// Handle function calls like NOW()
			if len(parts) > 1 && parts[1] == "()" {
				col.Default += "()"
			}
		}
	}

	return col
}

// getCurrentSchema gets current database schema
func (m *Migrator) getCurrentSchema(ctx context.Context) ([]SchemaTable, error) {
	var tables []SchemaTable

	// Get all tables
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		AND table_name != '_graft_migrations'
		ORDER BY table_name
	`

	rows, err := m.db.Query(ctx, query)
	if err != nil {
		log.Printf("❌ Error querying tables: %v", err)
		return nil, err
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Printf("❌ Error scanning table name: %v", err)
			continue
		}
		tableNames = append(tableNames, tableName)
	}

	log.Printf("🗃️  Found %d tables in database: %v", len(tableNames), tableNames)

	// Get columns for each table
	for _, tableName := range tableNames {
		columns, err := m.getTableColumns(ctx, tableName)
		if err != nil {
			log.Printf("❌ Error getting columns for table %s: %v", tableName, err)
			continue
		}

		log.Printf("📋 Table '%s' has %d columns", tableName, len(columns))
		for _, col := range columns {
			log.Printf("  - %s: %s (nullable: %v)", col.Name, col.Type, col.Nullable)
		}

		tables = append(tables, SchemaTable{
			Name:    tableName,
			Columns: columns,
			Indexes: []SchemaIndex{}, // TODO: implement index detection
		})
	}

	return tables, nil
}

// getTableColumns gets columns for a specific table
func (m *Migrator) getTableColumns(ctx context.Context, tableName string) ([]SchemaColumn, error) {
	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`

	log.Printf("🔍 Querying columns for table: %s", tableName)
	rows, err := m.db.Query(ctx, query, tableName)
	if err != nil {
		log.Printf("❌ Error querying columns for table %s: %v", tableName, err)
		return nil, err
	}
	defer rows.Close()

	var columns []SchemaColumn
	for rows.Next() {
		var colName, dataType, isNullable string
		var colDefault *string

		if err := rows.Scan(&colName, &dataType, &isNullable, &colDefault); err != nil {
			log.Printf("❌ Error scanning column for table %s: %v", tableName, err)
			continue
		}

		col := SchemaColumn{
			Name:     colName,
			Type:     dataType,
			Nullable: isNullable == "YES",
		}

		if colDefault != nil {
			col.Default = *colDefault
		}

		log.Printf("  📝 Column: %s %s (nullable: %v, default: %v)", colName, dataType, col.Nullable, colDefault)
		columns = append(columns, col)
	}

	log.Printf("✅ Found %d columns for table %s", len(columns), tableName)
	return columns, nil
}

// generateSchemaDiff compares current schema with target schema
func (m *Migrator) generateSchemaDiff(ctx context.Context, targetSchemaPath string) (*SchemaDiff, error) {
	currentSchema, err := m.getCurrentSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current schema: %w", err)
	}

	targetSchema, err := parseSchemaFile(targetSchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target schema: %w", err)
	}

	diff := &SchemaDiff{
		NewTables:      []SchemaTable{},
		DroppedTables:  []string{},
		ModifiedTables: []TableDiff{},
		NewIndexes:     []SchemaIndex{},
		DroppedIndexes: []string{},
	}

	// Create maps for easier lookup
	currentTables := make(map[string]SchemaTable)
	for _, table := range currentSchema {
		currentTables[table.Name] = table
	}

	targetTables := make(map[string]SchemaTable)
	for _, table := range targetSchema {
		targetTables[table.Name] = table
	}

	// Find new tables (tables that exist in target but not in current)
	for _, targetTable := range targetSchema {
		if _, exists := currentTables[targetTable.Name]; !exists {
			diff.NewTables = append(diff.NewTables, targetTable)
		}
	}

	// Find dropped tables (tables that exist in current but not in target)
	for _, currentTable := range currentSchema {
		if _, exists := targetTables[currentTable.Name]; !exists {
			diff.DroppedTables = append(diff.DroppedTables, currentTable.Name)
		}
	}

	// Find modified tables (tables that exist in both but have different columns)
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

// compareTableColumns compares columns between two tables
func compareTableColumns(current, target SchemaTable) TableDiff {
	diff := TableDiff{
		Name:            target.Name,
		NewColumns:      []SchemaColumn{},
		DroppedColumns:  []string{},
		ModifiedColumns: []ColumnDiff{},
	}

	currentCols := make(map[string]SchemaColumn)
	for _, col := range current.Columns {
		currentCols[col.Name] = col
	}

	targetCols := make(map[string]SchemaColumn)
	for _, col := range target.Columns {
		targetCols[col.Name] = col
	}

	// Find new columns
	for _, targetCol := range target.Columns {
		if _, exists := currentCols[targetCol.Name]; !exists {
			diff.NewColumns = append(diff.NewColumns, targetCol)
		}
	}

	// Find dropped columns
	for _, currentCol := range current.Columns {
		if _, exists := targetCols[currentCol.Name]; !exists {
			diff.DroppedColumns = append(diff.DroppedColumns, currentCol.Name)
		}
	}

	// Find modified columns - only include actual meaningful differences
	for colName, targetCol := range targetCols {
		if currentCol, exists := currentCols[colName]; exists {
			var changes []string

			// Check type changes - be more intelligent about type comparison
			if !isEquivalentType(currentCol.Type, targetCol.Type) {
				// Debug logging to understand what's happening
				log.Printf("🔍 Type difference detected for column %s.%s:", target.Name, colName)
				log.Printf("  Current (DB): %s -> normalized: %s", currentCol.Type, normalizeTypeForComparison(currentCol.Type))
				log.Printf("  Target (Schema): %s -> normalized: %s", targetCol.Type, normalizeTypeForComparison(targetCol.Type))

				// Handle SERIAL types specially - they cannot be altered after creation
				if strings.Contains(strings.ToUpper(targetCol.Type), "SERIAL") {
					// Skip SERIAL type changes as they cannot be altered
					log.Printf("⚠️  Warning: Cannot alter column %s.%s to SERIAL type - SERIAL columns must be created initially", target.Name, colName)
				} else {
					targetType := extractDataType(targetCol.Type)
					changes = append(changes, fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" TYPE %s;",
						target.Name, colName, targetType))
				}
			}

			// Check nullable changes - but skip for primary key columns
			if !isPrimaryKeyColumn(targetCol) && currentCol.Nullable != targetCol.Nullable {
				if targetCol.Nullable {
					changes = append(changes, fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" DROP NOT NULL;",
						target.Name, colName))
				} else {
					changes = append(changes, fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" SET NOT NULL;",
						target.Name, colName))
				}
			}

			// Check default changes - but skip for SERIAL columns and be smarter about defaults
			if !strings.Contains(strings.ToUpper(targetCol.Type), "SERIAL") && !isEquivalentDefault(currentCol.Default, targetCol.Default) {
				if targetCol.Default != "" {
					changes = append(changes, fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" SET DEFAULT %s;",
						target.Name, colName, targetCol.Default))
				} else {
					changes = append(changes, fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" DROP DEFAULT;",
						target.Name, colName))
				}
			}

			if len(changes) > 0 {
				diff.ModifiedColumns = append(diff.ModifiedColumns, ColumnDiff{
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

// hasRealTableChanges checks if table changes contain meaningful modifications
func hasRealTableChanges(modifiedTables []TableDiff) bool {
	for _, tableDiff := range modifiedTables {
		// New or dropped columns are always real changes
		if len(tableDiff.NewColumns) > 0 || len(tableDiff.DroppedColumns) > 0 {
			return true
		}

		// Check if modified columns have real changes
		for _, modCol := range tableDiff.ModifiedColumns {
			if len(modCol.Changes) > 0 {
				return true
			}
		}
	}
	return false
}

// isPrimaryKeyColumn checks if a column is a primary key
func isPrimaryKeyColumn(col SchemaColumn) bool {
	return strings.Contains(strings.ToUpper(col.Type), "PRIMARY KEY") ||
		strings.Contains(strings.ToUpper(col.Type), "SERIAL")
}

// isEquivalentType checks if two types are equivalent (handles PostgreSQL type variations)
func isEquivalentType(currentType, targetType string) bool {
	current := normalizeTypeForComparison(currentType)
	target := normalizeTypeForComparison(targetType)
	return current == target
}

// isEquivalentDefault checks if two defaults are equivalent
func isEquivalentDefault(currentDefault, targetDefault string) bool {
	// Handle common default variations
	current := strings.TrimSpace(currentDefault)
	target := strings.TrimSpace(targetDefault)

	// Both empty
	if current == "" && target == "" {
		return true
	}

	// Handle NOW() variations
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

// normalizeTypeForComparison normalizes types specifically for comparison
func normalizeTypeForComparison(pgType string) string {
	// Clean up the type string - remove constraints like PRIMARY KEY, UNIQUE, NOT NULL
	cleaned := pgType
	cleaned = strings.ReplaceAll(cleaned, "PRIMARY KEY", "")
	cleaned = strings.ReplaceAll(cleaned, "UNIQUE", "")
	cleaned = strings.ReplaceAll(cleaned, "NOT NULL", "")

	// Remove DEFAULT clauses
	if idx := strings.Index(strings.ToUpper(cleaned), "DEFAULT"); idx != -1 {
		cleaned = cleaned[:idx]
	}

	cleaned = strings.TrimSpace(cleaned)
	normalized := strings.ToUpper(cleaned)

	// Handle SERIAL types - they become integer in the database
	if strings.Contains(normalized, "SERIAL") {
		if strings.Contains(normalized, "BIGSERIAL") {
			return "INTEGER" // Normalize to common type
		}
		return "INTEGER"
	}

	// Handle timestamp variations
	if strings.Contains(normalized, "TIMESTAMP WITH TIME ZONE") {
		return "TIMESTAMP WITH TIME ZONE"
	}
	if strings.Contains(normalized, "TIMESTAMP WITHOUT TIME ZONE") || normalized == "TIMESTAMP" {
		return "TIMESTAMP WITHOUT TIME ZONE"
	}

	// Handle varchar variations - this is key for the issue
	if strings.HasPrefix(normalized, "VARCHAR") || strings.HasPrefix(normalized, "CHARACTER VARYING") {
		// For VARCHAR, ignore length differences for now to avoid unnecessary migrations
		// Both VARCHAR(255) and CHARACTER VARYING should normalize to VARCHAR
		return "VARCHAR"
	}

	// Handle TEXT type
	if normalized == "TEXT" {
		return "TEXT"
	}

	// Handle INTEGER variations
	if normalized == "INTEGER" || normalized == "INT" || normalized == "INT4" {
		return "INTEGER"
	}

	// Handle BIGINT variations
	if normalized == "BIGINT" || normalized == "INT8" {
		return "BIGINT"
	}

	return normalized
} // extractDataType extracts just the data type part from a column definition
func extractDataType(columnType string) string {
	// Remove constraints that shouldn't be part of the data type
	typeStr := columnType
	typeStr = strings.ReplaceAll(typeStr, "PRIMARY KEY", "")
	typeStr = strings.ReplaceAll(typeStr, "UNIQUE", "")
	typeStr = strings.ReplaceAll(typeStr, "NOT NULL", "")

	// Remove DEFAULT clauses
	if idx := strings.Index(strings.ToUpper(typeStr), "DEFAULT"); idx != -1 {
		typeStr = typeStr[:idx]
	}

	// Handle SERIAL types - convert to their underlying integer types
	upperType := strings.ToUpper(strings.TrimSpace(typeStr))
	if strings.Contains(upperType, "SERIAL") {
		if strings.Contains(upperType, "BIGSERIAL") {
			return "BIGINT"
		}
		return "INTEGER"
	}

	// Return cleaned type
	return strings.TrimSpace(typeStr)
}

// SchemaTable represents a table in the schema
type SchemaTable struct {
	Name    string
	Columns []SchemaColumn
	Indexes []SchemaIndex
}

// SchemaColumn represents a column in a table
type SchemaColumn struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
}

// SchemaIndex represents an index
type SchemaIndex struct {
	Name    string
	Table   string
	Columns []string
	Unique  bool
}

// SchemaDiff represents differences between schemas
type SchemaDiff struct {
	NewTables      []SchemaTable
	DroppedTables  []string
	ModifiedTables []TableDiff
	NewIndexes     []SchemaIndex
	DroppedIndexes []string
}

// TableDiff represents changes to a table
type TableDiff struct {
	Name            string
	NewColumns      []SchemaColumn
	DroppedColumns  []string
	ModifiedColumns []ColumnDiff
}

// ColumnDiff represents changes to a column
type ColumnDiff struct {
	Name    string
	OldType string
	NewType string
	Changes []string
}
