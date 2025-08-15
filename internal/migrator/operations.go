package migrator

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Rana718/Graft/internal/types"
)

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
			return fmt.Errorf("failed to apply migration %s: %w", migration.ID, err)
		}
	}

	log.Println("All migrations applied successfully")
	return nil
}

func (m *Migrator) handleConflictsInteractively(ctx context.Context, conflicts []types.MigrationConflict, pendingMigrations []types.Migration) error {
	fmt.Println("\n  Conflicts detected:")
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
			fmt.Printf(" %s: %s\n", conflict.Type, conflict.Description)
			for _, solution := range conflict.Solutions {
				fmt.Printf("    %s\n", solution)
			}
			fmt.Println()
		}

		choice := m.getUserChoice([]string{"add-defaults", "skip", "reset"}, "Choose action for NOT NULL conflicts")
		switch choice {
		case "add-defaults":
			return m.handleAddDefaultValue(notNullConflicts)
		case "skip":
			log.Println("Skipping migrations due to conflicts")
			return nil
		case "reset":
			return m.handleResetAndApply(ctx)
		}
	}

	if len(otherConflicts) > 0 {
		for _, conflict := range otherConflicts {
			fmt.Printf("❌ %s: %s\n", conflict.Type, conflict.Description)
			for _, solution := range conflict.Solutions {
				fmt.Printf("   💡 %s\n", solution)
			}
			fmt.Println()
		}

		choice := m.getUserChoice([]string{"continue", "skip", "reset"}, "Choose action for other conflicts")
		switch choice {
		case "continue":
			log.Println("  Continuing despite conflicts...")
		case "skip":
			log.Println("Skipping migrations due to conflicts")
			return nil
		case "reset":
			return m.handleResetAndApply(ctx)
		}
	}

	for _, migration := range pendingMigrations {
		if err := m.applySingleMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.ID, err)
		}
	}

	log.Println(" All migrations applied successfully")
	return nil
}

func (m *Migrator) handleResetAndApply(ctx context.Context) error {
	fmt.Println("  This will drop all tables and apply all migrations")
	fmt.Println()

	if !m.askUserConfirmation("Are you sure you want to reset the database?") {
		return fmt.Errorf("database reset cancelled by user")
	}

	if m.askUserConfirmation("Create backup before reset?") {
		log.Println("  Backup creation not yet implemented with adapter pattern")
	}

	tables, err := m.adapter.GetAllTableNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	log.Println("Dropping tables...")
	for _, table := range tables {
		if err := m.adapter.DropTable(ctx, table); err != nil {
			log.Printf("Warning: Failed to drop table %s: %v", table, err)
		}
	}

	log.Println("Applying all migrations...")
	allMigrations, err := m.loadMigrationsFromDir()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	for _, migration := range allMigrations {
		if err := m.applySingleMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.ID, err)
		}
	}

	log.Println("Database reset and migrations applied")
	return nil
}

func (m *Migrator) handleAddDefaultValue(conflicts []types.MigrationConflict) error {
	fmt.Println()
	fmt.Println("Add DEFAULT values to these columns:")
	fmt.Println()

	for _, conflict := range conflicts {
		fmt.Printf("📋 Table: %s, Column: %s\n", conflict.TableName, conflict.ColumnName)
	}

	fmt.Println()
	fmt.Println("Steps:")
	fmt.Println("1. Edit schema/migration file")
	fmt.Println("2. Add DEFAULT values for new columns")
	fmt.Println("3. Run migration again")
	fmt.Println()
	fmt.Println("Example:")
	if len(conflicts) > 0 {
		fmt.Printf("   ALTER TABLE \"%s\" ADD COLUMN \"%s\" VARCHAR(255) DEFAULT '' NOT NULL;\n",
			conflicts[0].TableName, conflicts[0].ColumnName)
	}
	fmt.Println()

	return fmt.Errorf("add DEFAULT values and run migration again")
}

func (m *Migrator) getUserChoice(validOptions []string, prompt string) string {
	if m.force {
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

	fmt.Printf("� Migration Status\n")
	fmt.Printf("==================\n\n")
	fmt.Printf("Total migrations: %d\n", len(migrations))
	fmt.Printf("Applied: %d\n", len(applied))
	fmt.Printf("Pending: %d\n", len(migrations)-len(applied))
	fmt.Println()

	if len(migrations) == 0 {
		fmt.Println("No migrations found")
		return nil
	}

	fmt.Println("Migration Details:")
	fmt.Println("------------------")

	for _, migration := range migrations {
		status := " Pending"
		timestamp := ""

		if appliedTime, exists := applied[migration.ID]; exists && appliedTime != nil {
			status = " Applied"
			timestamp = fmt.Sprintf(" (applied: %s)", appliedTime.Format("2006-01-02 15:04:05"))
		}

		fmt.Printf("%-50s %s%s\n", migration.ID, status, timestamp)
	}

	return nil
}

func (m *Migrator) Reset(ctx context.Context) error {
	fmt.Println("🗑️  This will drop all tables and data in your database!")
	fmt.Println()

	if !m.askUserConfirmation("Are you sure you want to reset the database?") {
		fmt.Println("Database reset cancelled")
		return nil
	}

	if m.askUserConfirmation("Create backup before reset?") {
		log.Println("⚠️  Backup creation not yet implemented with adapter pattern")
	}

	tables, err := m.adapter.GetAllTableNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	log.Println("Dropping tables...")
	for _, table := range tables {
		if err := m.adapter.DropTable(ctx, table); err != nil {
			log.Printf("Warning: Failed to drop table %s: %v", table, err)
		}
	}

	log.Println("Database reset completed")
	return nil
}
