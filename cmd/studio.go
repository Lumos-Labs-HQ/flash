package cmd

import (
	"fmt"
	"os"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/studio"
	"github.com/spf13/cobra"
)

var studioCmd = &cobra.Command{
	Use:   "studio",
	Short: "Open FlashORM Studio - Visual database editor",
	Long: `
Launch FlashORM Studio, a web-based interface for viewing and editing your database.
Similar to Prisma Studio, it provides an intuitive UI for managing your data.

The studio will start a local web server and open in your default browser.

Examples:
  flash studio
  flash studio --db "postgres://user:pass@localhost:5432/mydb"
  flash studio --port 3000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}

		dbURL, _ := cmd.Flags().GetString("db")
		if dbURL != "" {
			os.Setenv(cfg.Database.URLEnv, dbURL)
			fmt.Printf("📊 Using database: %s\n", maskDBURL(dbURL))
		}

		port, _ := cmd.Flags().GetInt("port")
		browser, _ := cmd.Flags().GetBool("browser")

		server := studio.NewServer(cfg, port)
		return server.Start(browser)
	},
}

func init() {
	rootCmd.AddCommand(studioCmd)
	studioCmd.Flags().IntP("port", "p", 5555, "Port to run studio on")
	studioCmd.Flags().BoolP("browser", "b", true, "Open browser automatically")
	studioCmd.Flags().String("db", "", "Database URL (overrides config/env)")
}

func maskDBURL(url string) string {
	if len(url) < 20 {
		return "***"
	}
	if idx := len(url) / 2; idx > 0 {
		return url[:10] + "***" + url[len(url)-10:]
	}
	return "***"
}
