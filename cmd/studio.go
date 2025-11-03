package cmd

import (
	"fmt"

	"github.com/Rana718/Graft/internal/config"
	"github.com/Rana718/Graft/internal/studio"
	"github.com/spf13/cobra"
)

var studioCmd = &cobra.Command{
	Use:   "studio",
	Short: "Open Graft Studio - Visual database editor",
	Long: `
Launch Graft Studio, a web-based interface for viewing and editing your database.
Similar to Prisma Studio, it provides an intuitive UI for managing your data.

The studio will start a local web server and open in your default browser.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w", err)
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
}
