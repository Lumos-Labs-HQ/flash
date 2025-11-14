package cmd

import (
	"fmt"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/gogen"
	"github.com/Lumos-Labs-HQ/flash/internal/jsgen"
	"github.com/Lumos-Labs-HQ/flash/internal/pygen"
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
- Python projects: Generate Python code with type hints

Configuration is read from flash.config.json`,

	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		generated := false
		if cfg.Gen.JS.Enabled {
			fmt.Println("🔨 Generating JavaScript code...")
			generator := jsgen.New(cfg)
			if err := generator.Generate(); err != nil {
				return fmt.Errorf("failed to generate JavaScript code: %w", err)
			}
			fmt.Println("🎉 JavaScript code generated successfully!")
			fmt.Printf("   Output: %s\n", cfg.Gen.JS.Out)
			generated = true
		}

		// Generate Python
		if cfg.Gen.Python.Enabled {
			fmt.Println("🔨 Generating Python code...")
			generator := pygen.New(cfg)
			if err := generator.Generate(); err != nil {
				return fmt.Errorf("failed to generate Python code: %w", err)
			}
			fmt.Println("🎉 Python code generated successfully!")
			fmt.Printf("   Output: %s\n", cfg.Gen.Python.Out)
			generated = true
		}

		// Generate Go (default if nothing else enabled)
		if !generated {
			fmt.Println("🔨 Generating Go code...")
			generator := gogen.New(cfg)
			if err := generator.Generate(); err != nil {
				return fmt.Errorf("failed to generate Go code: %w", err)
			}
			fmt.Println("🎉 Go code generated successfully!")
			fmt.Println("   Output: flash_gen/")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(genCmd)
}
