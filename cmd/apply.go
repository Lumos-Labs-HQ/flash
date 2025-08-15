package cmd

import (
	"context"
	"fmt"

	"github.com/Rana718/Graft/internal/config"
	"github.com/Rana718/Graft/internal/migrator"

	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply pending migrations",
	Long: `Apply all pending migrations to the database.
	This command will:
	1. Check for migration conflicts
	2. Prompt for backup if conflicts are detected
	3. Apply all pending migrations in order
	4. Update migration tracking table

	Use --force to skip confirmation prompts.`,
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

		return m.Apply(ctx, "", cfg.SchemaPath)
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
}
