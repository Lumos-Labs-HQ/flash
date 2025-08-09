package migrator

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Rana718/Graft/internal/backup"
	"github.com/Rana718/Graft/internal/types"
)

// Apply migrations with optional generation
func (m *Migrator) Apply(ctx context.Context, name string, schemaPath string) error {
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	if name != "" {
		if err := m.GenerateMigration(ctx, name, schemaPath); err != nil {
			return fmt.Errorf("failed to generate migration: %w", err)
		}
	}

	return m.ApplyWithConflictDetection(ctx)
}

// Apply migrations with conflict detection
func (m *Migrator) ApplyWithConflictDetection(ctx context.Context) error {
	if err := m.cleanupBrokenMigrationRecords(ctx); err != nil {
		log.Printf("Warning: Failed to cleanup broken migration records: %v", err)
	}

	migrations, err := m.loadMigrationsFromDir()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	var pendingMigrations []types.Migration
	for _, migration := range migrations {
		if _, exists := applied[migration.ID]; !exists {
			pendingMigrations = append(pendingMigrations, migration)
		}
	}

	if len(pendingMigrations) == 0 {
		log.Println("No pending migrations")
		return nil
	}

	log.Printf("Found %d pending migrations", len(pendingMigrations))

	hasConflicts, conflicts, err := m.hasConflicts(ctx, pendingMigrations)
	if err != nil {
		return fmt.Errorf("failed to check for conflicts: %w", err)
	}

	if hasConflicts {
		return m.handleConflictsInteractively(ctx, conflicts, pendingMigrations)
	}

	for _, migration := range pendingMigrations {
		if err := m.applySingleMigration(ctx, migration); err != nil {
			return err
		}
	}

	log.Println("All migrations applied successfully")
	return nil
}

func (m *Migrator) handleConflictsInteractively(ctx context.Context, conflicts []types.MigrationConflict, pendingMigrations []types.Migration) error {
	fmt.Println("\n⚠️  Conflicts detected:")
	fmt.Println()

	notNullConflicts := []types.MigrationConflict{}
	otherConflicts := []types.MigrationConflict{}

	for _, conflict := range conflicts {
		if conflict.Type == "not_null_constraint" {
			notNullConflicts = append(notNullConflicts, conflict)
		} else {
			otherConflicts = append(otherConflicts, conflict)
		}
	}

	if len(notNullConflicts) > 0 {
		for _, conflict := range notNullConflicts {
			fmt.Printf("❌ %s\n", conflict.Description)
		}
		fmt.Println()

		fmt.Println("How to resolve:")
		fmt.Println("1. Reset database (all data lost)")
		fmt.Println("2. Add default value to column")
		fmt.Println("3. Cancel migration")
		fmt.Println()

		choice := m.getUserChoice([]string{"1", "2", "3"}, "Please select an option (1-3): ")

		switch choice {
		case "1":
			return m.handleResetAndApply(ctx)
		case "2":
			return m.handleAddDefaultValue(notNullConflicts)
		case "3":
			return fmt.Errorf("migration cancelled")
		}
	}

	if len(otherConflicts) > 0 {
		fmt.Println("⚠️  Other warnings:")
		for _, conflict := range otherConflicts {
			fmt.Printf("   - %s: %s\n", conflict.Type, conflict.Description)
		}
		fmt.Println()

		if !m.askUserConfirmation("Continue with migration?") {
			return fmt.Errorf("migration cancelled")
		}
	}

	for _, migration := range pendingMigrations {
		if err := m.applySingleMigration(ctx, migration); err != nil {
			return err
		}
	}

	log.Println("🎉 All migrations applied successfully")
	return nil
}

func (m *Migrator) handleResetAndApply(ctx context.Context) error {
	fmt.Println("🗑️  This will drop all tables and apply all migrations")
	fmt.Println()

	if !m.askUserConfirmation("Are you sure you want to reset the database?") {
		return fmt.Errorf("reset cancelled")
	}

	if m.askUserConfirmation("Create backup before reset?") {
		backupPath, err := backup.PerformBackup(ctx, m.db, m.backupPath, "Pre-migration-reset backup")
		if err != nil {
			log.Printf("Backup failed: %v", err)
			if !m.askUserConfirmation("Continue without backup?") {
				return fmt.Errorf("backup failed, reset cancelled")
			}
		} else if backupPath != "" {
			log.Printf("Backup created: %s", backupPath)
		}
	}

	backupManager := backup.NewBackupManager(m.db, m.backupPath)
	tables, err := backupManager.GetAllTableNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	log.Println("Dropping tables...")
	for _, table := range tables {
		if _, err := m.db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS \"%s\" CASCADE", table)); err != nil {
			log.Printf("Failed to drop table %s: %v", table, err)
		}
	}

	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to recreate migrations table: %w", err)
	}

	log.Println("Applying all migrations...")
	allMigrations, err := m.loadMigrationsFromDir()
	if err != nil {
		return fmt.Errorf("failed to load all migrations: %w", err)
	}

	for _, migration := range allMigrations {
		if err := m.applySingleMigration(ctx, migration); err != nil {
			return err
		}
	}

	log.Println("Database reset and migrations applied")
	return nil
}

// Guide user to add default values
func (m *Migrator) handleAddDefaultValue(conflicts []types.MigrationConflict) error {
	fmt.Println()
	fmt.Println("Add DEFAULT values to these columns:")
	fmt.Println()

	for _, conflict := range conflicts {
		fmt.Printf("   • %s.%s\n", conflict.TableName, conflict.ColumnName)
	}

	fmt.Println()
	fmt.Println("Steps:")
	fmt.Println("1. Edit schema/migration file")
	fmt.Println("2. Add DEFAULT values for new columns")
	fmt.Println("3. Run migration again")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Printf("   ALTER TABLE \"%s\" ADD COLUMN \"%s\" VARCHAR(255) DEFAULT '' NOT NULL;\n",
		conflicts[0].TableName, conflicts[0].ColumnName)
	fmt.Println()

	return fmt.Errorf("add DEFAULT values and run migration again")
}

// Get user choice from options
func (m *Migrator) getUserChoice(validOptions []string, prompt string) string {
	if m.force {
		return validOptions[0]
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(prompt)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		for _, option := range validOptions {
			if response == option {
				return response
			}
		}

		fmt.Printf("Invalid option. Please choose from: %s\n", strings.Join(validOptions, ", "))
	}
}

// Apply single migration with tracking
func (m *Migrator) applySingleMigration(ctx context.Context, migration types.Migration) error {
	log.Printf("Applying: %s", migration.Name)

	migrationSQL, err := parseMigrationFile(migration.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file %s: %w", migration.FilePath, err)
	}

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction for migration %s: %w", migration.ID, err)
	}

	if _, err := tx.Exec(ctx, migrationSQL.Up); err != nil {
		tx.Rollback(ctx)
		errorMsg := fmt.Sprintf("failed to execute migration %s: %v", migration.ID, err)

		if strings.Contains(err.Error(), "null values") && strings.Contains(err.Error(), "23502") {
			errorMsg += "\n\nNOT NULL constraint error - add DEFAULT value or make column nullable"
		}

		return fmt.Errorf("%s", errorMsg)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", migration.ID, err)
	}

	if _, err := m.db.Exec(ctx, `
		INSERT INTO _graft_migrations (id, checksum, migration_name, started_at, finished_at, applied_steps_count, logs) 
		VALUES ($1, $2, $3, NOW(), NOW(), 1, 'Migration completed successfully')
		ON CONFLICT (id) DO UPDATE SET 
			finished_at = NOW(),
			applied_steps_count = 1,
			logs = 'Migration completed successfully'
	`, migration.ID, migration.Checksum, migration.Name); err != nil {
		return fmt.Errorf("failed to record migration completion %s: %w", migration.ID, err)
	}

	log.Printf("Applied: %s", migration.Name)
	return nil
}

// Show migration status
func (m *Migrator) Status(ctx context.Context) error {
	log.Println("Graft Migration Status")
	log.Println("=====================")

	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	migrations, err := m.loadMigrationsFromDir()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	var missingMigrations []string
	migrationMap := make(map[string]bool)
	for _, migration := range migrations {
		migrationMap[migration.ID] = true
	}

	for appliedID := range applied {
		if !migrationMap[appliedID] {
			missingMigrations = append(missingMigrations, appliedID)
		}
	}

	var statusItems []types.MigrationStatusItem
	appliedCount := 0
	pendingCount := 0

	for _, migration := range migrations {
		appliedAt, isApplied := applied[migration.ID]
		status := "PENDING"
		if isApplied {
			status = "APPLIED"
			appliedCount++
		} else {
			pendingCount++
		}

		statusItems = append(statusItems, types.MigrationStatusItem{
			ID:        migration.ID,
			Name:      migration.Name,
			Status:    status,
			AppliedAt: appliedAt,
		})
	}

	status := types.MigrationStatus{
		TotalMigrations:   len(migrations),
		AppliedMigrations: appliedCount,
		PendingMigrations: pendingCount,
	}

	fmt.Printf("Database: Connected\n")
	fmt.Printf("Total Migrations: %d\n", status.TotalMigrations)
	fmt.Printf("Applied: %d\n", status.AppliedMigrations)
	fmt.Printf("Pending: %d\n", status.PendingMigrations)

	// Report missing migrations
	if len(missingMigrations) > 0 {
		fmt.Printf("Missing Files: %d\n", len(missingMigrations))
		for _, missingID := range missingMigrations {
			fmt.Printf("   %s (applied but file missing)\n", missingID)
		}
	}

	fmt.Println()

	if len(statusItems) == 0 {
		fmt.Println("No migrations found.")
		return nil
	}

	fmt.Println("Migration History:")
	fmt.Println("------------------")
	for _, item := range statusItems {
		statusIcon := "❌"
		timeStr := "Not applied"

		if item.Status == "APPLIED" {
			statusIcon = "✅"
			if item.AppliedAt != nil {
				timeStr = item.AppliedAt.Format("2006-01-02 15:04:05")
			}
		}

		fmt.Printf("%s %s\n", statusIcon, item.Name)
		fmt.Printf("   ID: %s\n", item.ID)
		fmt.Printf("   Status: %s\n", item.Status)
		fmt.Printf("   Applied: %s\n\n", timeStr)
	}

	return nil
}

// Reset database
func (m *Migrator) Reset(ctx context.Context) error {
	log.Println("WARNING: This will drop all tables and data!")

	if !m.askUserConfirmation("Are you sure you want to reset the database?") {
		return nil
	}

	if m.askUserConfirmation("Create backup before reset?") {
		backupPath, err := backup.PerformBackup(ctx, m.db, m.backupPath, "Pre-reset backup")
		if err != nil {
			log.Printf("Backup failed: %v", err)
			if !m.askUserConfirmation("Continue without backup?") {
				return nil
			}
		} else if backupPath != "" {
			log.Printf("Backup created: %s", backupPath)
		}
	}

	backupManager := backup.NewBackupManager(m.db, m.backupPath)
	tables, err := backupManager.GetAllTableNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	// Drop tables
	for _, table := range tables {
		log.Printf("Dropping: %s", table)
		if _, err := m.db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)); err != nil {
			log.Printf("Failed to drop %s: %v", table, err)
		}
	}

	if m.askUserConfirmation("Delete migration files?") {
		if err := os.RemoveAll(m.migrationsPath); err != nil {
			log.Printf("Failed to remove files: %v", err)
		} else {
			log.Printf("Removed files from: %s", m.migrationsPath)
		}
	}

	log.Println("Database reset completed")
	return nil
}
