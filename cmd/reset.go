package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"Rana718/Graft/internal/backup"
	"Rana718/Graft/internal/config"
	"Rana718/Graft/internal/db"
	"Rana718/Graft/internal/migration"
	"github.com/spf13/cobra"
)

// resetCmd represents the reset command
var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Drop all database data and re-apply all migrations",
	Long: `Drop all database data and re-apply all migrations from scratch.
This is a destructive operation that will delete all data in your database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !config.IsInitialized() {
			return fmt.Errorf("graft is not initialized. Run 'graft init' first")
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Connect to database
		conn, err := db.NewConnection(cfg)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer conn.Close()

		// Create migrations table if it doesn't exist
		if err := conn.CreateMigrationsTable(); err != nil {
			return fmt.Errorf("failed to create migrations table: %w", err)
		}

		// Check for force flag
		force, _ := cmd.Flags().GetBool("force")
		
		// Prompt for backup unless force is used
		if !force {
			fmt.Printf("⚠️  This will drop all database data and re-apply all migrations.\n")
			fmt.Print("Do you want to create a backup first? (yes/no): ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}
			response = strings.TrimSpace(strings.ToLower(response))
			
			if response == "yes" || response == "y" {
				backupManager, err := backup.NewManager(cfg)
				if err != nil {
					return fmt.Errorf("failed to create backup manager: %w", err)
				}

				fmt.Println("🔄 Creating backup...")
				_, err = backupManager.CreateBackup(conn)
				if err != nil {
					return fmt.Errorf("failed to create backup: %w", err)
				}
			}

			fmt.Print("\nAre you sure you want to reset the database? (yes/no): ")
			response, err = reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "yes" && response != "y" {
				fmt.Println("❌ Reset cancelled")
				return nil
			}
		}

		fmt.Println("🔄 Dropping all tables...")
		if err := conn.DropAllTables(); err != nil {
			return fmt.Errorf("failed to drop tables: %w", err)
		}

		// Clear migration records
		if _, err := conn.DB.Exec("DELETE FROM graft_migrations"); err != nil {
			return fmt.Errorf("failed to clear migration records: %w", err)
		}

		// Get migration manager
		migrationManager, err := migration.NewManager(cfg)
		if err != nil {
			return fmt.Errorf("failed to create migration manager: %w", err)
		}

		// Get all local migrations
		localMigrations, err := migrationManager.GetLocalMigrations()
		if err != nil {
			return fmt.Errorf("failed to get local migrations: %w", err)
		}

		if len(localMigrations) == 0 {
			fmt.Println("✅ Database reset completed (no migrations to apply)")
			return nil
		}

		fmt.Printf("🔄 Re-applying %d migration(s)...\n", len(localMigrations))

		// Apply all migrations
		for _, migration := range localMigrations {
			fmt.Printf("🔄 Applying migration: %s\n", migration.Name)

			// Validate migration
			if err := migrationManager.ValidateMigration(migration); err != nil {
				return fmt.Errorf("migration validation failed: %w", err)
			}

			// Execute migration
			if err := conn.ExecuteSQL(migration.Content); err != nil {
				return fmt.Errorf("failed to execute migration %s: %w", migration.Name, err)
			}

			// Record migration
			if err := conn.RecordMigration(migration.Name, migration.Checksum); err != nil {
				return fmt.Errorf("failed to record migration %s: %w", migration.Name, err)
			}

			fmt.Printf("✅ Applied migration: %s\n", migration.Name)
		}

		fmt.Printf("\n🎉 Database reset completed! Applied %d migration(s)\n", len(localMigrations))

		// Run SQLC if configured
		if cfg.SQLCConfigPath != "" {
			fmt.Println("🔄 Running SQLC generate...")
			if err := runSQLCGenerate(); err != nil {
				fmt.Printf("⚠️  SQLC generate failed: %v\n", err)
			} else {
				fmt.Println("✅ SQLC generate completed")
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}
