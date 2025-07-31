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
	Up        string     `json:"up"`
	Down      string     `json:"down"`
	Checksum  string     `json:"checksum"`
	CreatedAt time.Time  `json:"created_at"`
}

// MigrationFile represents the structure of a migration file
type MigrationFile struct {
	Migration Migration `json:"migration"`
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

// createMigrationsTable creates the Prisma-style migrations table
func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	query := `
        CREATE TABLE IF NOT EXISTS _prisma_migrations (
            id VARCHAR(36) PRIMARY KEY,
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
	return err
}

// getAppliedMigrations returns list of applied migrations
func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[string]*time.Time, error) {
	applied := make(map[string]*time.Time)

	query := `SELECT id, finished_at FROM _prisma_migrations WHERE finished_at IS NOT NULL AND rolled_back_at IS NULL`
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

// hasConflicts checks for migration conflicts
func (m *Migrator) hasConflicts(ctx context.Context, migrations []Migration) (bool, []string, error) {
	var conflicts []string

	// Check for table conflicts
	for _, migration := range migrations {
		if strings.Contains(strings.ToUpper(migration.Up), "CREATE TABLE") {
			tableRegex := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)`)
			matches := tableRegex.FindStringSubmatch(migration.Up)
			if len(matches) >= 2 {
				tableName := matches[1]

				var exists bool
				err := m.db.QueryRow(ctx,
					"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1 AND table_schema = 'public')",
					tableName).Scan(&exists)
				if err != nil {
					return false, nil, err
				}

				if exists && tableName != "_prisma_migrations" {
					conflicts = append(conflicts, fmt.Sprintf("Table '%s' already exists", tableName))
				}
			}
		}
	}

	return len(conflicts) > 0, conflicts, nil
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

// GenerateMigration creates a new migration file
func (m *Migrator) GenerateMigration(name string) error {
	if err := os.MkdirAll(m.migrationsPath, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	cleanName := strings.ReplaceAll(strings.ToLower(name), " ", "_")
	migrationID := fmt.Sprintf("%s_%s", timestamp, cleanName)

	upSQL := fmt.Sprintf(`-- CreateTable or AlterTable: %s
-- Add your SQL commands here

-- Example:
-- CREATE TABLE example (
--     id SERIAL PRIMARY KEY,
--     name VARCHAR(255) NOT NULL,
--     created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
-- );`, name)

	downSQL := `-- Drop or alter statements to reverse the migration
-- Add your reverse SQL commands here

-- Example:
-- DROP TABLE IF EXISTS example CASCADE;`

	migration := Migration{
		ID:        migrationID,
		Name:      name,
		Up:        upSQL,
		Down:      downSQL,
		Checksum:  generateChecksum(upSQL),
		CreatedAt: time.Now(),
	}

	migrationFile := MigrationFile{Migration: migration}

	filename := fmt.Sprintf("%s.json", migrationID)
	filePath := filepath.Join(m.migrationsPath, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(migrationFile); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	log.Printf("✨ Generated migration: %s", filePath)
	return nil
}

// loadMigrationsFromDir loads migrations from directory
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
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(m.migrationsPath, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("⚠️  Warning: Failed to read migration file %s: %v", file.Name(), err)
			continue
		}

		var migrationFile MigrationFile
		if err := json.Unmarshal(data, &migrationFile); err != nil {
			log.Printf("⚠️  Warning: Failed to parse migration file %s: %v", file.Name(), err)
			continue
		}

		migrations = append(migrations, migrationFile.Migration)
	}

	// Sort migrations by creation time
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].CreatedAt.Before(migrations[j].CreatedAt)
	})

	return migrations, nil
}

// generateChecksum generates a checksum for content
func generateChecksum(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}
