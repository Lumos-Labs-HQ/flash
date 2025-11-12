package utils

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
)

type FileUtils struct{}

// LoadMigrationsFromDir loads migration files from a directory
func (f *FileUtils) LoadMigrationsFromDir(migrationsDir string) ([]types.Migration, error) {
	migrations := make([]types.Migration, 0, 32) // Pre-allocate with reasonable capacity

	err := filepath.WalkDir(migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".sql") {
			return err
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

// GenerateMigrationFilename creates a timestamped migration filename
func (f *FileUtils) GenerateMigrationFilename(name string) string {
	timestamp := time.Now().Format("20060102150405")
	cleanName := strings.ReplaceAll(name, " ", "_")
	return fmt.Sprintf("%s_%s.sql", timestamp, cleanName)
}

type InputUtils struct{}

// GetUserChoice prompts user for choice from valid options
func (i *InputUtils) GetUserChoice(validOptions []string, prompt string, force bool) string {
	if force {
		return validOptions[0]
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s (%s): ", prompt, strings.Join(validOptions, "/"))
		input, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(strings.ToLower(input))

		for _, option := range validOptions {
			if choice == option {
				return choice
			}
		}
		fmt.Printf("Invalid option. Please choose from: %s\n", strings.Join(validOptions, ", "))
	}
}

// AskConfirmation asks user for yes/no confirmation
func (i *InputUtils) AskConfirmation(message string, force bool) bool {
	if force {
		return true
	}
	fmt.Printf("%s (y/N): ", message)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

// ConflictUtils - Migration conflict detection and handling
type ConflictUtils struct{}

// DetectMigrationConflicts checks for potential conflicts in migration content
func (c *ConflictUtils) DetectMigrationConflicts(ctx context.Context, migration types.Migration, adapter interface{}) ([]types.MigrationConflict, error) {
	var conflicts []types.MigrationConflict

	content, err := os.ReadFile(migration.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration file: %w", err)
	}

	migrationContent := string(content)

	// Check for ALTER TABLE ADD COLUMN NOT NULL without DEFAULT
	addColumnRegex := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+["\'\x60]?(\w+)["\'\x60]?\s+ADD\s+(?:COLUMN\s+)?(?:IF\s+NOT\s+EXISTS\s+)?["\'\x60]?(\w+)["\'\x60]?\s+.*?NOT\s+NULL.*?(?:;|$)`)
	matches := addColumnRegex.FindAllStringSubmatch(migrationContent, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			tableName := match[1]
			columnName := match[2]

			// Skip if has DEFAULT value
			if strings.Contains(strings.ToUpper(match[0]), "DEFAULT") {
				continue
			}

			hasData := c.tableHasData(ctx, adapter, tableName)
			if !hasData {
				continue // No conflict if table is empty or doesn't exist
			}

			conflicts = append(conflicts, types.MigrationConflict{
				Type:        "not_null_constraint",
				TableName:   tableName,
				ColumnName:  columnName,
				Description: fmt.Sprintf("Table '%s' has data: adding NOT NULL column '%s' without DEFAULT will fail", tableName, columnName),
				Solutions: []string{
					"Add a DEFAULT value to the column",
					"Make the column nullable first, then update existing rows",
					"Reset the database if data loss is acceptable",
				},
			})
		}
	}

	return conflicts, nil
}

// tableHasData checks if a table exists and has data
func (c *ConflictUtils) tableHasData(ctx context.Context, adapter interface{}, tableName string) bool {
	type tableChecker interface {
		CheckTableExists(ctx context.Context, tableName string) (bool, error)
		GetTableData(ctx context.Context, tableName string) ([]map[string]interface{}, error)
	}

	checker, ok := adapter.(tableChecker)
	if !ok {
		return true 
	}

	// Check if table exists
	exists, err := checker.CheckTableExists(ctx, tableName)
	if err != nil || !exists {
		return false 
	}

	// Check if table has data
	data, err := checker.GetTableData(ctx, tableName)
	if err != nil {
		return true 
	}

	return len(data) > 0 
}

type SQLUtils struct{}

// FilterPendingMigrations returns migrations that haven't been applied
func FilterPendingMigrations(migrations []types.Migration, applied map[string]*time.Time) []types.Migration {
	var pending []types.Migration
	for _, migration := range migrations {
		if _, exists := applied[migration.ID]; !exists {
			pending = append(pending, migration)
		}
	}
	return pending
}
