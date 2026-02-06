//go:build plugin_core || plugin_seed || plugin_all || dev
// +build plugin_core plugin_seed plugin_all dev

package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/seeder"
	"github.com/spf13/cobra"
)

func GetSeedCommand() *cobra.Command {
	return seedCmd
}

var seedCmd = &cobra.Command{
	Use:   "seed [tables...]",
	Short: "Generate and insert random test data",
	Long: `
Generate and insert random test data into database tables.

Examples:
  flash seed                          # Seed all tables with 10 records each
  flash seed --count 100              # Seed all tables with 100 records each
  flash seed users                    # Seed only 'users' table
  flash seed users posts --count 50   # Seed specific tables with 50 records
  flash seed --relations              # Include foreign key relationships
  flash seed --truncate --count 100   # Clear tables before seeding
  flash seed users:100 posts:500      # Custom count per table`,

	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}

		// Parse flags with proper error handling
		count, err := cmd.Flags().GetInt("count")
		if err != nil {
			return fmt.Errorf("invalid count flag: %w", err)
		}
		relations, err := cmd.Flags().GetBool("relations")
		if err != nil {
			return fmt.Errorf("invalid relations flag: %w", err)
		}
		truncate, err := cmd.Flags().GetBool("truncate")
		if err != nil {
			return fmt.Errorf("invalid truncate flag: %w", err)
		}
		batch, err := cmd.Flags().GetInt("batch")
		if err != nil {
			return fmt.Errorf("invalid batch flag: %w", err)
		}
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return fmt.Errorf("invalid force flag: %w", err)
		}
		noTransaction, err := cmd.Flags().GetBool("no-transaction")
		if err != nil {
			return fmt.Errorf("invalid no-transaction flag: %w", err)
		}

		// Parse table-specific counts
		tableCounts := make(map[string]int)
		var specificTables []string

		for _, arg := range args {
			if strings.Contains(arg, ":") {
				parts := strings.Split(arg, ":")
				if len(parts) == 2 {
					table := parts[0]
					tableCount, err := strconv.Atoi(parts[1])
					if err != nil {
						return fmt.Errorf("invalid count for table %s: %s", table, parts[1])
					}
					tableCounts[table] = tableCount
					specificTables = append(specificTables, table)
				}
			} else {
				specificTables = append(specificTables, arg)
			}
		}

		// If specific tables provided without counts, use default count
		for _, table := range specificTables {
			if _, exists := tableCounts[table]; !exists {
				tableCounts[table] = count
			}
		}

		seedConfig := seeder.SeedConfig{
			Count:         count,
			Tables:        tableCounts,
			Relations:     relations,
			Truncate:      truncate,
			Batch:         batch,
			Force:         force,
			NoTransaction: noTransaction,
		}

		ctx := context.Background()
		s, err := seeder.NewSeeder(cfg)
		if err != nil {
			return fmt.Errorf("failed to create seeder: %w", err)
		}
		defer s.Close()

		return s.Seed(ctx, seedConfig)
	},
}

func init() {
	seedCmd.Flags().IntP("count", "c", 10, "Number of records to generate per table")
	seedCmd.Flags().BoolP("relations", "r", false, "Include foreign key relationships")
	seedCmd.Flags().BoolP("truncate", "t", false, "Truncate tables before seeding")
	seedCmd.Flags().IntP("batch", "b", 100, "Batch size for inserts")
	seedCmd.Flags().BoolP("force", "f", false, "Skip confirmations and continue on errors")
	seedCmd.Flags().Bool("no-transaction", false, "Disable transaction wrapping (each batch commits separately)")
}
