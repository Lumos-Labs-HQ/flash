package migrator

import (
	"context"
	"fmt"
)

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

	// Check for orphaned migrations in database — O(N+M) with map
	migrationFileSet := make(map[string]struct{}, len(migrations))
	for _, migration := range migrations {
		migrationFileSet[migration.ID] = struct{}{}
	}
	orphanedCount := 0
	for id := range applied {
		if _, found := migrationFileSet[id]; !found {
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
