//go:build plugins
// +build plugins

package cmd

import (
	"context"
	"fmt"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/database"
	"github.com/Lumos-Labs-HQ/flash/internal/export"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export database tables",
	Long: `
Export all database tables (excluding migration table) to various formats.
Supported formats: json (default), csv, sqlite

Examples:
  flash export
  flash export --sqlite
  flash export --csv
  flash export --json`,
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

		// Determine format from flags
		format := "json"
		if csv, _ := cmd.Flags().GetBool("csv"); csv {
			format = "csv"
		} else if sqlite, _ := cmd.Flags().GetBool("sqlite"); sqlite {
			format = "sqlite"
		} else if jsonFlag, _ := cmd.Flags().GetBool("json"); jsonFlag {
			format = "json"
		}

		ctx := context.Background()

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

		exportPath, err := export.PerformExport(ctx, adapter, cfg.ExportPath, format)
		if err != nil {
			return err
		}

		if exportPath != "" {
			fmt.Printf("✅ Export completed: %s\n", exportPath)
		} else {
			fmt.Println("No export created (database is empty)")
		}

		return nil
	},
}

func init() {
	// Command is registered by plugin executors, not the base CLI
	exportCmd.Flags().BoolP("json", "j", false, "Export as JSON (default)")
	exportCmd.Flags().BoolP("csv", "c", false, "Export as CSV")
	exportCmd.Flags().BoolP("sqlite", "s", false, "Export as SQLite")
}
