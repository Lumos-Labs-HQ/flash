package cmd

import (
	"fmt"

	"Rana718/Graft/internal/config"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize graft in the current project",
	Long: `Initialize graft in the current project by creating a default configuration file
and necessary directories.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if config.IsInitialized() {
			return fmt.Errorf("graft is already initialized in this project")
		}

		return config.InitializeProject()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
