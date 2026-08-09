package migrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/types"
	"github.com/Lumos-Labs-HQ/flash/internal/utils"
)

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
	content, err := m.getMigrationContent(filePath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	var downSQL strings.Builder
	inDown := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for migrate markers (case-insensitive)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(trimmed, "--") && strings.Contains(lower, "+migrate down") {
			inDown = true
			continue
		}
		if strings.HasPrefix(trimmed, "--") && strings.Contains(lower, "+migrate up") {
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
	return m.adapter.RemoveMigrationRecord(ctx, migrationID)
}

// isDatabaseEmpty returns true when no user tables exist (excluding _flash_migrations).
func (m *Migrator) isDatabaseEmpty(ctx context.Context) (bool, error) {
	tables, err := m.adapter.GetAllTableNames(ctx)
	if err != nil {
		return false, err
	}
	for _, t := range tables {
		if t != "_flash_migrations" {
			return false, nil
		}
	}
	return true, nil
}

// clearMigrationRecords deletes all rows from _flash_migrations so migrations
// can be re-applied from scratch.
func (m *Migrator) clearMigrationRecords(ctx context.Context) error {
	if m.provider == "clickhouse" {
		return m.adapter.ExecuteMigration(ctx, "ALTER TABLE _flash_migrations DELETE WHERE 1=1")
	}
	return m.adapter.ExecuteMigration(ctx, "DELETE FROM _flash_migrations")
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

	// ScyllaDB/Cassandra: just drop the entire keyspace — cascades to all tables/views/types/UDTs
	if m.provider == "scylla" || m.provider == "scylladb" || m.provider == "cassandra" {
		keyspaces, err := m.adapter.GetKeyspaces(ctx)
		if err != nil {
			return fmt.Errorf("failed to get keyspaces: %w", err)
		}
		for _, ks := range keyspaces {
			if ks == "system" || ks == "system_schema" || ks == "system_auth" || ks == "system_distributed" || ks == "system_traces" {
				continue
			}
			dropSQL := fmt.Sprintf("DROP KEYSPACE IF EXISTS \"%s\"", ks)
			fmt.Printf("  Dropping keyspace: %s\n", ks)
			if err := m.adapter.ExecuteMigration(ctx, dropSQL); err != nil {
				fmt.Printf("Warning: Failed to drop keyspace %s: %v\n", ks, err)
			}
		}
		fmt.Println("✅ Database reset completed")
		return nil
	}

	// Drop all tables first
	tables, err := m.adapter.GetAllTableNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	// MySQL requires disabling foreign key checks to drop tables with FK constraints
	if m.provider == "mysql" {
		if err := m.adapter.ExecuteMigration(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err != nil {
			fmt.Printf("Warning: Failed to disable FK checks: %v\n", err)
		}
	}

	for _, table := range tables {
		if err := m.adapter.DropTable(ctx, table); err != nil {
			fmt.Printf("Warning: Failed to drop table %s: %v\n", table, err)
		}
	}

	// Re-enable foreign key checks for MySQL
	if m.provider == "mysql" {
		_ = m.adapter.ExecuteMigration(ctx, "SET FOREIGN_KEY_CHECKS = 1")
	}

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
