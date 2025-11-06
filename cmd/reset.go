package cmd

import (
	"context"
	"fmt"

	"github.com/Lumos-Labs-HQ/graft/internal/config"
	"github.com/Lumos-Labs-HQ/graft/internal/migrator"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the database",
	Long: `
Reset the database by dropping all tables and data.
This is a destructive operation that will:

1. Prompt for confirmation (unless --force is used)
2. Offer to create a backup before reset
3. Drop all tables in the database
4. Optionally remove migration files

⚠️  WARNING: This will permanently delete all data in your database!

Use --force to skip all confirmation prompts.`,
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

		m, err := migrator.NewMigrator(cfg)
		if err != nil {
			return fmt.Errorf("failed to create migrator: %w", err)
		}
		defer m.Close()

		force, _ := cmd.Flags().GetBool("force")

		return m.Reset(ctx, force)
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}
