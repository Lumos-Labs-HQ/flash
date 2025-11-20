//go:build plugins
// +build plugins

package cmd

import (
	"context"
	"fmt"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/pull"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull database schema to update local schema file",
	Long: `
Pull the current database schema and update the local schema file.
This command introspects the current database and generates a schema.sql file
that reflects the current state of the database. This is useful for:
- Synchronizing your local schema with the database after manual changes
- Creating an initial schema file from an existing database
- Keeping your schema file up-to-date with the current database state

The command will:
1. Connect to the database
2. Introspect all tables, columns, and constraints
3. Generate a comprehensive schema.sql file
4. Optionally create a backup of the existing schema file`,

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

		ctx := context.Background()

		backup, _ := cmd.Flags().GetBool("backup")
		outputPath, _ := cmd.Flags().GetString("output")

		pullService, err := pull.NewService(cfg)
		if err != nil {
			return fmt.Errorf("failed to create pull service: %w", err)
		}
		defer pullService.Close()

		opts := pull.Options{
			Backup:     backup,
			OutputPath: outputPath,
		}

		return pullService.PullSchema(ctx, opts)
	},
}

func init() {
	// Command is registered by plugin executors, not the base CLI
	pullCmd.Flags().BoolP("backup", "b", false, "Create backup of existing schema file before overwriting")
	pullCmd.Flags().StringP("output", "o", "", "Custom output file path (overrides config schema_path)")
}
