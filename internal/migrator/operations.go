package migrator

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
)

func (m *Migrator) Apply(ctx context.Context, name string, schemaPath string) error {
	log.Println("🚀 Running graft apply...")

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

func (m *Migrator) ApplyWithConflictDetection(ctx context.Context) error {
	if err := m.cleanupBrokenMigrationRecords(ctx); err != nil {
		log.Printf("⚠️  Warning: Failed to cleanup broken migration records: %v", err)
	}

	migrations, err := m.loadMigrationsFromDir()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	var pendingMigrations []Migration
	for _, migration := range migrations {
		if _, exists := applied[migration.ID]; !exists {
			pendingMigrations = append(pendingMigrations, migration)
		}
	}

	if len(pendingMigrations) == 0 {
		log.Println("✅ No pending migrations")
		return nil
	}

	log.Printf("📋 Found %d pending migrations", len(pendingMigrations))

	log.Println("🔍 Checking for migration conflicts...")
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

	log.Println("🎉 All migrations applied successfully")
	return nil
}

func (m *Migrator) handleConflictsInteractively(ctx context.Context, conflicts []MigrationConflict, pendingMigrations []Migration) error {
	fmt.Println("\n⚠️  There are conflicts that need to be resolved:")
	fmt.Println()

	// Show summary of conflicts
	notNullConflicts := []MigrationConflict{}
	otherConflicts := []MigrationConflict{}

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

		fmt.Println("The database contains data that would be affected by this migration.")
		fmt.Println("How would you like to resolve this?")
		fmt.Println()
		fmt.Println("1. Reset the database (⚠️  all data will be lost)")
		fmt.Println("2. Add a default value to the new column")
		fmt.Println("3. Cancel the migration")
		fmt.Println()

		choice := m.getUserChoice([]string{"1", "2", "3"}, "Please select an option (1-3): ")

		switch choice {
		case "1":
			return m.handleResetAndApply(ctx, pendingMigrations)
		case "2":
			return m.handleAddDefaultValue(ctx, notNullConflicts, pendingMigrations)
		case "3":
			log.Println("❌ Migration cancelled by user")
			return fmt.Errorf("migration cancelled")
		}
	}

	if len(otherConflicts) > 0 {
		fmt.Println("⚠️  Other warnings detected:")
		for _, conflict := range otherConflicts {
			fmt.Printf("   - %s: %s\n", conflict.Type, conflict.Description)
		}
		fmt.Println()

		if !m.askUserConfirmation("Continue with migration?") {
			log.Println("❌ Migration cancelled by user")
			return fmt.Errorf("migration cancelled due to warnings")
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

func (m *Migrator) handleResetAndApply(ctx context.Context, pendingMigrations []Migration) error {
	fmt.Println()
	fmt.Println("🗑️  This will:")
	fmt.Println("   • Drop all tables and data")
	fmt.Println("   • Apply all pending migrations")
	fmt.Println("   • Recreate tables with new schema")
	fmt.Println()

	if !m.askUserConfirmation("Are you sure you want to reset the database?") {
		log.Println("❌ Reset cancelled")
		return fmt.Errorf("reset cancelled")
	}

	if m.askUserConfirmation("Create a backup before reset?") {
		backupPath, err := m.createBackup(ctx, "Pre-migration-reset backup")
		if err != nil {
			log.Printf("⚠️  Warning: Failed to create backup: %v", err)
			if !m.askUserConfirmation("Continue without backup?") {
				log.Println("❌ Reset cancelled")
				return fmt.Errorf("backup failed, reset cancelled")
			}
		} else if backupPath != "" {
			log.Printf("✅ Backup created at: %s", backupPath)
		}
	}

	tables, err := m.getAllTableNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	log.Println("🗑️  Dropping all tables...")
	for _, table := range tables {
		log.Printf("   Dropping table: %s", table)
		if _, err := m.db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS \"%s\" CASCADE", table)); err != nil {
			log.Printf("⚠️  Warning: Failed to drop table %s: %v", table, err)
		}
	}

	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to recreate migrations table: %w", err)
	}

	// Load ALL migrations (not just pending ones) and apply them in order
	log.Println("📦 Applying all migrations from the beginning...")
	allMigrations, err := m.loadMigrationsFromDir()
	if err != nil {
		return fmt.Errorf("failed to load all migrations: %w", err)
	}

	for _, migration := range allMigrations {
		if err := m.applySingleMigration(ctx, migration); err != nil {
			return err
		}
	}

	log.Println("🎉 Database reset and migrations applied successfully")
	return nil
}

// handleAddDefaultValue guides user through adding default values
func (m *Migrator) handleAddDefaultValue(ctx context.Context, conflicts []MigrationConflict, pendingMigrations []Migration) error {
	fmt.Println()
	fmt.Println("📝 To proceed, you need to add DEFAULT values to the following columns:")
	fmt.Println()

	for _, conflict := range conflicts {
		fmt.Printf("   • %s.%s\n", conflict.TableName, conflict.ColumnName)
	}

	fmt.Println()
	fmt.Println("Please:")
	fmt.Println("1. Edit your schema file or migration file")
	fmt.Println("2. Add DEFAULT values for the new columns")
	fmt.Println("3. Run the migration again")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Printf("   ALTER TABLE \"%s\" ADD COLUMN \"%s\" VARCHAR(255) DEFAULT '' NOT NULL;\n",
		conflicts[0].TableName, conflicts[0].ColumnName)
	fmt.Println()

	return fmt.Errorf("please add DEFAULT values and run migration again")
}

// getUserChoice prompts user for a choice from given options
func (m *Migrator) getUserChoice(validOptions []string, prompt string) string {
	if m.force {
		return validOptions[0] // Return first option if force is enabled
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

// applySingleMigration applies a single migration with proper error handling and tracking
func (m *Migrator) applySingleMigration(ctx context.Context, migration Migration) error {
	log.Printf("⏳ Applying migration: %s", migration.Name)

	// Read migration SQL
	migrationSQL, err := parseMigrationFile(migration.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file %s: %w", migration.FilePath, err)
	}

	// Execute migration in transaction (DO NOT insert tracking record yet)
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction for migration %s: %w", migration.ID, err)
	}

	// Execute the migration SQL
	if _, err := tx.Exec(ctx, migrationSQL.Up); err != nil {
		tx.Rollback(ctx)

		// Create a more helpful error message
		errorMsg := fmt.Sprintf("failed to execute migration %s: %v", migration.ID, err)

		// Check if it's a NOT NULL constraint error
		if strings.Contains(err.Error(), "null values") && strings.Contains(err.Error(), "23502") {
			errorMsg += "\n\n💡 This error occurs when adding a NOT NULL column to a table that already contains data."
			errorMsg += "\n   To fix this:"
			errorMsg += "\n   1. Add a DEFAULT value to the column definition, or"
			errorMsg += "\n   2. Make the column nullable initially, or"
			errorMsg += "\n   3. Update existing rows before adding the constraint"
			errorMsg += "\n"
			errorMsg += "\n   Example: ALTER TABLE table_name ADD COLUMN column_name TYPE DEFAULT 'default_value' NOT NULL;"
		}

		return fmt.Errorf("%s", errorMsg)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", migration.ID, err)
	}

	// Only insert tracking record AFTER successful application
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

	log.Printf("✅ Applied migration: %s", migration.Name)
	return nil
}

// Status shows migration status
func (m *Migrator) Status(ctx context.Context) error {
	log.Println("📊 Graft Migration Status")
	log.Println("========================")

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

	// Check for missing migration files
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

	var statusItems []MigrationStatusItem
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

		statusItems = append(statusItems, MigrationStatusItem{
			ID:        migration.ID,
			Name:      migration.Name,
			Status:    status,
			AppliedAt: appliedAt,
		})
	}

	status := MigrationStatus{
		TotalMigrations:   len(migrations),
		AppliedMigrations: appliedCount,
		PendingMigrations: pendingCount,
		Migrations:        statusItems,
		DatabaseStatus:    "connected",
	}

	fmt.Printf("Database: Connected\n")
	fmt.Printf("Total Migrations: %d\n", status.TotalMigrations)
	fmt.Printf("Applied: %d\n", status.AppliedMigrations)
	fmt.Printf("Pending: %d\n", status.PendingMigrations)

	// Report missing migrations
	if len(missingMigrations) > 0 {
		fmt.Printf("⚠️  Missing Migration Files: %d\n", len(missingMigrations))
		for _, missingID := range missingMigrations {
			fmt.Printf("   ❌ %s (applied but file missing)\n", missingID)
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

// Reset resets the database
func (m *Migrator) Reset(ctx context.Context) error {
	log.Println("🗑️  WARNING: This will drop all tables and data!")

	if !m.askUserConfirmation("Are you sure you want to reset the database?") {
		log.Println("❌ Reset cancelled")
		return nil
	}

	// Create backup before reset
	if m.askUserConfirmation("Create a backup before reset?") {
		backupPath, err := m.createBackup(ctx, "Pre-reset backup")
		if err != nil {
			log.Printf("⚠️  Warning: Failed to create backup: %v", err)
			if !m.askUserConfirmation("Continue without backup?") {
				log.Println("❌ Reset cancelled")
				return nil
			}
		} else if backupPath != "" {
			log.Printf("✅ Backup created at: %s", backupPath)
		}
	}

	// Get all tables
	tables, err := m.getAllTableNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	// Drop all tables
	for _, table := range tables {
		log.Printf("🗑️  Dropping table: %s", table)
		if _, err := m.db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)); err != nil {
			log.Printf("⚠️  Warning: Failed to drop table %s: %v", table, err)
		}
	}

	// Remove migration files
	if m.askUserConfirmation("Delete all migration files?") {
		if err := os.RemoveAll(m.migrationsPath); err != nil {
			log.Printf("⚠️  Warning: Failed to remove migration files: %v", err)
		} else {
			log.Printf("🗑️  Removed migration files from: %s", m.migrationsPath)
		}
	}

	log.Println("✅ Database reset completed")
	return nil
}

// Backup creates a manual backup
func (m *Migrator) Backup(ctx context.Context, comment string) error {
	if comment == "" {
		comment = "Manual backup"
	}

	backupPath, err := m.createBackup(ctx, comment)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Backup completed: %s\n", backupPath)
	return nil
}
