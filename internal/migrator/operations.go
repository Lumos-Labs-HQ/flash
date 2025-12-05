package migrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
	"github.com/Lumos-Labs-HQ/flash/internal/utils"
)

// Apply runs migrations with optional generation
func (m *Migrator) Apply(ctx context.Context, name, schemaPath string) error {
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

// ApplyWithConflictDetection applies pending migrations with conflict detection
func (m *Migrator) ApplyWithConflictDetection(ctx context.Context) error {
	_ = m.cleanupBrokenMigrationRecords(ctx)

	migrations, err := m.loadMigrationsFromDir()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	pending := utils.FilterPendingMigrations(migrations, applied)
	if len(pending) == 0 {
		fmt.Println("No pending migrations")
		return nil
	}

	fmt.Printf("Found %d pending migrations\n", len(pending))

	if hasConflicts, conflicts, err := m.hasConflicts(ctx, pending); err != nil {
		return fmt.Errorf("failed to check for conflicts: %w", err)
	} else if hasConflicts {
		return m.handleConflictsInteractively(ctx, conflicts, pending)
	}

	return m.applyMigrations(ctx, pending)
}

// handleConflictsInteractively handles migration conflicts interactively
func (m *Migrator) handleConflictsInteractively(ctx context.Context, conflicts []types.MigrationConflict, pending []types.Migration) error {
	fmt.Println("⚠️  Migration conflicts detected:")
	for _, c := range conflicts {
		fmt.Printf("  - %s\n", c.Description)
	}
	fmt.Println()

	if m.force {
		fmt.Println("🚀 Force flag detected - resetting database and applying migrations...")
		return m.handleResetAndApply(ctx)
	}

	input := &utils.InputUtils{}
	choice := input.GetUserChoice([]string{"y", "n"}, "Reset database to resolve conflicts? This will drop all tables and data", false)

	if strings.ToLower(choice) != "y" {
		fmt.Println("Migration aborted due to conflicts")
		return fmt.Errorf("migration aborted due to conflicts")
	}

	if input.GetUserChoice([]string{"y", "n"}, "Create export before applying?", false) == "y" {
		fmt.Println("📦 Creating export...")
		if err := m.createExport(); err != nil {
			fmt.Printf("⚠️  Export failed: %v\n   Continuing without export...\n", err)
		} else {
			fmt.Println("✅ Export created successfully")
		}
	}

	return m.handleResetAndApply(ctx)
}

// handleResetAndApply resets DB and applies all migrations
func (m *Migrator) handleResetAndApply(ctx context.Context) error {
	fmt.Println("🔄 Resetting database and applying all migrations...")
	tables, err := m.adapter.GetAllTableNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	for _, table := range tables {
		if err := m.adapter.DropTable(ctx, table); err != nil {
			fmt.Printf("Warning: Failed to drop table %s: %v\n", table, err)
		}
	}

	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to recreate migrations table: %w", err)
	}

	allMigrations, err := m.loadMigrationsFromDir()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	return m.applyMigrations(ctx, allMigrations)
}

// applyMigrations applies migrations safely - each in its own transaction
func (m *Migrator) applyMigrations(ctx context.Context, migrations []types.Migration) error {
	if len(migrations) == 0 {
		return nil
	}

	fmt.Printf("📦 Applying %d migration(s)...\n", len(migrations))

	for i, migration := range migrations {
		fmt.Printf("  [%d/%d] %s\n", i+1, len(migrations), migration.ID)

		if err := m.applySingleMigrationSafely(ctx, migration); err != nil {
			fmt.Printf("❌ Failed at migration: %s\n", migration.ID)
			fmt.Printf("   Error: %v\n", err)
			fmt.Println("   Transaction rolled back. Fix the error and run 'flash apply' again.")
			return fmt.Errorf("migration %s failed: %w", migration.ID, err)
		}

		fmt.Printf("      ✅ Applied\n")
	}

	fmt.Println("✅ All migrations applied successfully")
	return nil
}

// applySingleMigrationSafely applies migration and records it in a single transaction
func (m *Migrator) applySingleMigrationSafely(ctx context.Context, migration types.Migration) error {
	content, err := os.ReadFile(migration.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	checksum := fmt.Sprintf("%x", len(content))

	// Extract only the UP section from the migration
	upSQL := extractUpSQL(string(content))

	// Use the combined method that does both operations in a single transaction
	if err := m.adapter.ExecuteAndRecordMigration(ctx, migration.ID, migration.Name, checksum, upSQL); err != nil {
		return err
	}

	return nil
}

// extractUpSQL extracts only the UP migration SQL from a migration file
// Migration files may contain both -- +migrate Up and -- +migrate Down sections
func extractUpSQL(content string) string {
	lines := strings.Split(content, "\n")
	var upLines []string
	inUpSection := false
	hasMarkers := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		lowerLine := strings.ToLower(trimmedLine)

		// Check for +migrate Up marker
		if strings.Contains(lowerLine, "+migrate up") {
			inUpSection = true
			hasMarkers = true
			continue
		}

		// Check for +migrate Down marker - stop collecting
		if strings.Contains(lowerLine, "+migrate down") {
			inUpSection = false
			continue
		}

		// If we're in the UP section, collect the line
		if inUpSection {
			upLines = append(upLines, line)
		}
	}

	// If no markers found, return entire content (legacy format)
	if !hasMarkers {
		return content
	}

	return strings.Join(upLines, "\n")
}

// createExport creates a database export using the adapter
func (m *Migrator) createExport() error {
	ctx := context.Background()

	tables, err := m.adapter.GetAllTableNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	var dataTables []string
	for _, table := range tables {
		if table != "_flash_migrations" {
			dataTables = append(dataTables, table)
		}
	}

	if len(dataTables) == 0 {
		return nil
	}

	exportData := types.BackupData{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Version:   "1.0",
		Tables:    make(map[string]interface{}),
		Comment:   "Pre-conflict export",
	}

	for _, table := range dataTables {
		data, err := m.adapter.GetTableData(ctx, table)
		if err != nil {
			fmt.Printf("Warning: Failed to export table %s: %v\n", table, err)
			continue
		}
		if len(data) > 0 {
			exportData.Tables[table] = data
		}
	}

	exportDir := "db_export"
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	filename := fmt.Sprintf("export_%s.json",
		time.Now().Format("2006-01-02_15-04-05"))
	exportPath := filepath.Join(exportDir, filename)

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	if err := os.WriteFile(exportPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	fmt.Printf("✅ Export saved to: %s\n", exportPath)
	return nil
}

// Reset drops all tables and optionally exports data
func (m *Migrator) Reset(ctx context.Context, force bool) error {
	fmt.Println("🗑️  This will drop all tables and data!")

	// Skip confirmation if force flag is set
	if !force {
		if !m.askUserConfirmation("Are you sure you want to reset the database?") {
			fmt.Println("Database reset cancelled")
			return nil
		}

		if m.askUserConfirmation("Create export before reset?") {
			fmt.Println("📦 Creating export...")
			if err := m.createExport(); err != nil {
				fmt.Printf("⚠️  Export failed: %v\n", err)
			}
		}
	} else {
		fmt.Println("⚡ Force mode: Skipping confirmations and backup")
	}

	// Drop all tables first
	tables, err := m.adapter.GetAllTableNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	m.adapter.ExecuteMigration(ctx, "SET FOREIGN_KEY_CHECKS = 0")

	for _, table := range tables {
		if err := m.adapter.DropTable(ctx, table); err != nil {
			fmt.Printf("Warning: Failed to drop table %s: %v\n", table, err)
		}
	}

	m.adapter.ExecuteMigration(ctx, "SET FOREIGN_KEY_CHECKS = 1")

	// Drop all enums
	enums, err := m.adapter.GetCurrentEnums(ctx)
	if err == nil {
		for _, enum := range enums {
			if err := m.adapter.DropEnum(ctx, enum.Name); err != nil {
				fmt.Printf("Warning: Failed to drop enum %s: %v\n", enum.Name, err)
			}
		}
	}

	fmt.Println("✅ Database reset completed")
	return nil
}

// Status prints migration status
func (m *Migrator) Status(ctx context.Context) error {
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

	pendingCount := 0
	for _, migration := range migrations {
		if _, exists := applied[migration.ID]; !exists {
			pendingCount++
		}
	}

	fmt.Println("🗂️  Migration Status")
	fmt.Println("==================")
	fmt.Printf("Total: %d | Applied: %d | Pending: %d\n\n", len(migrations), len(applied), pendingCount)

	if len(migrations) == 0 && len(applied) == 0 {
		fmt.Println("No migrations found")
		return nil
	}

	if len(migrations) == 0 && len(applied) > 0 {
		fmt.Println("⚠️  Warning: No migration files found, but database has applied migrations.")
		fmt.Println("   This usually means migration files were deleted.")
		fmt.Println("\nApplied migrations in database:")
		fmt.Printf("%-16s  %-30s  %-10s  %s\n", "ID", "NAME", "STATUS", "APPLIED AT")
		fmt.Printf("%-16s  %-30s  %-10s  %s\n", "──────────────", "──────────────────────────────", "──────────", "───────────────────")
		for id, t := range applied {
			migrationID, migrationName := splitMigrationID(id)
			timestamp := ""
			if t != nil {
				timestamp = t.Format("2006-01-02 15:04:05")
			}
			fmt.Printf("%-16s  %-30s  %-10s  %s\n", migrationID, migrationName, "Applied", timestamp)
		}
		return nil
	}

	fmt.Printf("%-16s  %-30s  %-10s  %s\n", "ID", "NAME", "STATUS", "APPLIED AT")
	fmt.Printf("%-16s  %-30s  %-10s  %s\n", "──────────────", "──────────────────────────────", "──────────", "───────────────────")
	for _, migration := range migrations {
		migrationID, migrationName := splitMigrationID(migration.ID)
		status := "Pending"
		timestamp := "-"
		if t, exists := applied[migration.ID]; exists && t != nil {
			status = "Applied"
			timestamp = t.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("%-16s  %-30s  %-10s  %s\n", migrationID, migrationName, status, timestamp)
	}

	// Check for orphaned migrations in database
	orphanedCount := 0
	for id := range applied {
		found := false
		for _, migration := range migrations {
			if migration.ID == id {
				found = true
				break
			}
		}
		if !found {
			orphanedCount++
		}
	}

	if orphanedCount > 0 {
		fmt.Printf("\n⚠️  Warning: %d migration(s) in database have no corresponding file\n", orphanedCount)
	}

	return nil
}

// splitMigrationID splits a migration ID like "20251204234836_add_phone_column" into ID and name
func splitMigrationID(fullID string) (string, string) {
	// Migration IDs are typically formatted as: YYYYMMDDHHMMSS_name
	if len(fullID) < 15 {
		return fullID, ""
	}

	// Find the first underscore after the timestamp
	for i := 14; i < len(fullID); i++ {
		if fullID[i] == '_' {
			return fullID[:i], fullID[i+1:]
		}
	}

	// If no underscore found, try to split at position 14 (timestamp length)
	if len(fullID) > 14 && fullID[14] == '_' {
		return fullID[:14], fullID[15:]
	}

	return fullID, ""
}

// Down rolls back the last migration or to a specific migration ID
func (m *Migrator) Down(ctx context.Context, targetMigrationID string, steps int) error {
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

	// Get applied migrations in order (most recent first)
	var appliedMigrations []types.Migration
	for _, migration := range migrations {
		if _, exists := applied[migration.ID]; exists {
			appliedMigrations = append(appliedMigrations, migration)
		}
	}

	if len(appliedMigrations) == 0 {
		fmt.Println("No migrations to roll back")
		return nil
	}

	// Reverse to get most recent first
	for i, j := 0, len(appliedMigrations)-1; i < j; i, j = i+1, j-1 {
		appliedMigrations[i], appliedMigrations[j] = appliedMigrations[j], appliedMigrations[i]
	}

	// Determine which migrations to roll back
	var toRollback []types.Migration
	if targetMigrationID != "" {
		// Roll back to specific migration
		found := false
		for _, migration := range appliedMigrations {
			if strings.HasPrefix(migration.ID, targetMigrationID) || migration.ID == targetMigrationID {
				found = true
				break
			}
			toRollback = append(toRollback, migration)
		}
		if !found {
			return fmt.Errorf("migration %s not found in applied migrations", targetMigrationID)
		}
	} else if steps > 0 {
		// Roll back specific number of steps
		if steps > len(appliedMigrations) {
			steps = len(appliedMigrations)
		}
		toRollback = appliedMigrations[:steps]
	} else {
		// Roll back last migration
		toRollback = appliedMigrations[:1]
	}

	if len(toRollback) == 0 {
		fmt.Println("No migrations to roll back")
		return nil
	}

	// Check for data loss and prompt for export
	hasDataLoss := false
	for _, migration := range toRollback {
		downSQL := m.extractDownSQL(migration.FilePath)
		if strings.Contains(strings.ToUpper(downSQL), "DROP TABLE") ||
			strings.Contains(strings.ToUpper(downSQL), "DROP COLUMN") ||
			strings.Contains(strings.ToUpper(downSQL), "TRUNCATE") {
			hasDataLoss = true
			break
		}
	}

	if hasDataLoss && !m.force {
		fmt.Println("⚠️  Warning: Rolling back these migrations may result in data loss!")

		input := &utils.InputUtils{}
		if input.GetUserChoice([]string{"y", "n"}, "Create export before rollback?", false) == "y" {
			fmt.Println("📦 Creating export...")
			if err := m.createExport(); err != nil {
				fmt.Printf("⚠️  Export failed: %v\n", err)
				if input.GetUserChoice([]string{"y", "n"}, "Continue without export?", false) != "y" {
					return fmt.Errorf("rollback cancelled")
				}
			} else {
				fmt.Println("✅ Export created successfully")
			}
		}

		if input.GetUserChoice([]string{"y", "n"}, "Proceed with rollback?", false) != "y" {
			return fmt.Errorf("rollback cancelled")
		}
	}

	fmt.Printf("📦 Rolling back %d migration(s)...\n", len(toRollback))

	for i, migration := range toRollback {
		fmt.Printf("  [%d/%d] Rolling back %s\n", i+1, len(toRollback), migration.ID)

		downSQL := m.extractDownSQL(migration.FilePath)
		if downSQL == "" || strings.TrimSpace(downSQL) == "-- Add rollback statements here" {
			fmt.Printf("    ⚠️  No down migration found for %s\n", migration.ID)
			if !m.force {
				input := &utils.InputUtils{}
				if input.GetUserChoice([]string{"y", "n"}, "Skip this migration and continue?", false) != "y" {
					return fmt.Errorf("rollback cancelled - no down migration for %s", migration.ID)
				}
			}
			continue
		}

		// Execute down migration
		if err := m.adapter.ExecuteMigration(ctx, downSQL); err != nil {
			return fmt.Errorf("failed to execute down migration %s: %w", migration.ID, err)
		}

		// Remove from migrations table
		if err := m.removeMigrationRecord(ctx, migration.ID); err != nil {
			return fmt.Errorf("failed to remove migration record %s: %w", migration.ID, err)
		}

		fmt.Printf("      ✅ Rolled back\n")
	}

	fmt.Println("✅ Rollback completed successfully")
	return nil
}

// extractDownSQL extracts the DOWN section from a migration file
func (m *Migrator) extractDownSQL(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	var downSQL strings.Builder
	inDown := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for migrate markers
		if strings.HasPrefix(trimmed, "-- +migrate Down") {
			inDown = true
			continue
		}
		if strings.HasPrefix(trimmed, "-- +migrate Up") {
			inDown = false
			continue
		}

		if inDown {
			downSQL.WriteString(line)
			downSQL.WriteString("\n")
		}
	}

	return strings.TrimSpace(downSQL.String())
}

// removeMigrationRecord removes a migration record from the tracking table
func (m *Migrator) removeMigrationRecord(ctx context.Context, migrationID string) error {
	query := fmt.Sprintf("DELETE FROM _flash_migrations WHERE id = '%s'", migrationID)
	return m.adapter.ExecuteMigration(ctx, query)
}
