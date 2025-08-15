package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/Rana718/Graft/internal/backup"
	"github.com/Rana718/Graft/internal/config"
	"github.com/Rana718/Graft/internal/database"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup [comment]",
	Short: "Create a database backup",
	Long: `Create a manual backup of the database.
	The backup includes all table data and migration history in JSON format.

	The backup will be saved in the backup directory specified in your config
	with a timestamp-based filename.

	Examples:
	  graft backup "before major update"
	  graft backup "pre-production backup"
	  graft backup  # Creates backup with default comment`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}

		if err := cfg.EnsureDirectories(); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}

		var comment string
		if len(args) > 0 {
			comment = strings.Join(args, " ")
		} else {
			comment = "Manual backup"
		}

		ctx := context.Background()

		// Create database adapter based on provider
		adapter := database.NewAdapter(cfg.Database.Provider)

		dbURL, err := cfg.GetDatabaseURL()
		if err != nil {
			return err
		}

		if err := adapter.Connect(ctx, dbURL); err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer adapter.Close()

		if err := adapter.Ping(ctx); err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}

		// Perform backup using backup package
		backupPath, err := backup.PerformBackupWithAdapter(ctx, adapter, cfg.BackupPath, comment)
		if err != nil {
			return err
		}

		if backupPath != "" {
			fmt.Printf("✅ Backup completed: %s\n", backupPath)
		} else {
			fmt.Println("No backup created (database is empty)")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
}
