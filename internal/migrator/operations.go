package migrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// Apply applies pending migrations for development
func (m *Migrator) Apply(ctx context.Context, name string, schemaPath string) error {
	log.Println("🚀 Running graft apply...")

	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Check for conflicts first
	migrations, err := m.loadMigrationsFromDir()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	hasConflicts, conflicts, err := m.hasConflicts(ctx, migrations)
	if err != nil {
		return fmt.Errorf("failed to check for conflicts: %w", err)
	}

	if hasConflicts {
		log.Println("⚠️  WARNING: Potential conflicts detected:")
		for _, conflict := range conflicts {
			log.Printf("  • %s", conflict)
		}

		if m.askUserConfirmation("Do you want to backup all tables before proceeding?") {
			backupPath, err := m.createBackup(ctx, "Pre-migration backup due to conflicts")
			if err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}
			if backupPath != "" {
				log.Printf("✅ Backup created at: %s", backupPath)
			}

			if m.askUserConfirmation("Reset database and clear all migrations?") {
				if err := m.Reset(ctx); err != nil {
					return fmt.Errorf("failed to reset database: %w", err)
				}
			}
		}

		if !m.askUserConfirmation("Continue with migration despite conflicts?") {
			log.Println("❌ Migration cancelled by user")
			return nil
		}
	}

	// Generate new migration if name provided
	if name != "" {
		if err := m.GenerateMigration(name, schemaPath); err != nil {
			return fmt.Errorf("failed to generate migration: %w", err)
		}

		// Reload migrations after generating new one
		migrations, err = m.loadMigrationsFromDir()
		if err != nil {
			return fmt.Errorf("failed to reload migrations: %w", err)
		}
	}

	// Apply all pending migrations
	return m.Deploy(ctx)
}

// Deploy applies all pending migrations
func (m *Migrator) Deploy(ctx context.Context) error {
	log.Println("🚀 Running graft deploy...")

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

	for _, migration := range pendingMigrations {
		log.Printf("⏳ Applying migration: %s", migration.Name)

		// Start migration record
		if _, err := m.db.Exec(ctx, `
			INSERT INTO _graft_migrations (id, checksum, migration_name, started_at) 
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (id) DO NOTHING
		`, migration.ID, migration.Checksum, migration.Name); err != nil {
			return fmt.Errorf("failed to start migration record %s: %w", migration.ID, err)
		}

		// Execute migration in transaction
		tx, err := m.db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to start transaction for migration %s: %w", migration.ID, err)
		}

		if _, err := tx.Exec(ctx, migration.Up); err != nil {
			tx.Rollback(ctx)

			// Log failure
			m.db.Exec(ctx, `
				UPDATE _graft_migrations 
				SET logs = $1, finished_at = NOW() 
				WHERE id = $2
			`, fmt.Sprintf("Migration failed: %v", err), migration.ID)

			return fmt.Errorf("failed to execute migration %s: %w", migration.ID, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.ID, err)
		}

		// Mark as completed
		if _, err := m.db.Exec(ctx, `
			UPDATE _graft_migrations 
			SET finished_at = NOW(), applied_steps_count = 1, logs = 'Migration completed successfully'
			WHERE id = $1
		`, migration.ID); err != nil {
			return fmt.Errorf("failed to mark migration as complete %s: %w", migration.ID, err)
		}

		log.Printf("✅ Applied migration: %s", migration.Name)
	}

	log.Println("🎉 All migrations applied successfully")
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
	fmt.Printf("Pending: %d\n\n", status.PendingMigrations)

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

// Restore restores from backup
func (m *Migrator) Restore(ctx context.Context, backupPath string) error {
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	log.Println("🔄 WARNING: This will overwrite all existing data!")

	if !m.askUserConfirmation("Are you sure you want to restore from backup?") {
		log.Println("❌ Restore cancelled")
		return nil
	}

	return m.restoreFromBackup(ctx, backupPath)
}

// restoreFromBackup restores database from backup file
func (m *Migrator) restoreFromBackup(ctx context.Context, backupPath string) error {
	log.Printf("🔄 Restoring database from backup: %s", backupPath)

	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	var backup BackupData
	if err := json.NewDecoder(file).Decode(&backup); err != nil {
		return fmt.Errorf("failed to decode backup file: %w", err)
	}

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	tables, err := m.getAllTableNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	// Restore tables
	for _, tableName := range tables {
		if tableName == "_graft_migrations" {
			continue
		}

		tableData, exists := backup.Tables[tableName]
		if !exists {
			log.Printf("⚠️  Table %s not found in backup, skipping...", tableName)
			continue
		}

		tableMap := tableData.(map[string]interface{})
		columns := tableMap["columns"].([]interface{})
		data := tableMap["data"].([]interface{})

		if len(data) == 0 {
			log.Printf("ℹ️  No data to restore for table %s", tableName)
			continue
		}

		// Clear existing data
		if _, err := tx.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", tableName)); err != nil {
			log.Printf("⚠️  Warning: Failed to truncate %s: %v", tableName, err)
		}

		// Prepare column names for INSERT
		columnNames := make([]string, len(columns))
		for i, col := range columns {
			columnNames[i] = col.(string)
		}

		// Restore data
		for _, row := range data {
			rowMap := row.(map[string]interface{})
			values := make([]interface{}, len(columnNames))
			placeholders := make([]string, len(columnNames))

			for i, colName := range columnNames {
				values[i] = rowMap[colName]
				placeholders[i] = fmt.Sprintf("$%d", i+1)
			}

			columnStr := strings.Join(columnNames, ", ")
			placeholderStr := strings.Join(placeholders, ", ")
			query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, columnStr, placeholderStr)

			if _, err := tx.Exec(ctx, query, values...); err != nil {
				log.Printf("⚠️  Warning: Failed to restore row in %s: %v", tableName, err)
			}
		}

		log.Printf("✅ Restored table %s with %d rows", tableName, len(data))
	}

	// Restore _graft_migrations table
	if migrationsData, exists := backup.Tables["_graft_migrations"]; exists {
		tableMap := migrationsData.(map[string]interface{})
		data := tableMap["data"].([]interface{})

		for _, row := range data {
			rowMap := row.(map[string]interface{})
			_, err := tx.Exec(ctx, `
				INSERT INTO _graft_migrations (id, checksum, finished_at, migration_name, logs, rolled_back_at, started_at, applied_steps_count) 
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
				ON CONFLICT (id) DO NOTHING
			`, rowMap["id"], rowMap["checksum"], rowMap["finished_at"], rowMap["migration_name"],
				rowMap["logs"], rowMap["rolled_back_at"], rowMap["started_at"], rowMap["applied_steps_count"])

			if err != nil {
				log.Printf("⚠️  Warning: Failed to restore migration record: %v", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit restore transaction: %w", err)
	}

	log.Printf("✅ Database restored successfully from backup created at %s", backup.Timestamp)
	return nil
}
