package cmd

import (
	"fmt"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/gogen"
	"github.com/Lumos-Labs-HQ/flash/internal/jsgen"
	"github.com/spf13/cobra"
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate code from SQL",
	Long: `
Generate type-safe code from SQL queries.
Automatically detects project type and generates appropriate code:
- Go projects: Generate Go code with custom generator
- Node.js projects: Generate JavaScript code with type annotations

Configuration is read from flash.config.json`,

	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Generate based on configuration
		if cfg.Gen.JS.Enabled {
			fmt.Println("🔨 Generating JavaScript code...")
			generator := jsgen.New(cfg)
			if err := generator.Generate(); err != nil {
				return fmt.Errorf("failed to generate JavaScript code: %w", err)
			}
			fmt.Println("🎉 JavaScript code generated successfully!")
			fmt.Printf("   Output: %s\n", cfg.Gen.JS.Out)
		} else {
			fmt.Println("🔨 Generating Go code...")
			generator := gogen.New(cfg)
			if err := generator.Generate(); err != nil {
				return fmt.Errorf("failed to generate Go code: %w", err)
			}
			fmt.Println("🎉 Go code generated successfully!")
			fmt.Println("   Output: FlashORM_gen/")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(genCmd)
}
