//go:build plugin_all
// +build plugin_all

package cmd

import (
	"github.com/spf13/cobra"
)

func ExecuteAllPlugin() error {
	allRoot := &cobra.Command{
		Use:   "flash",
		Short: "FlashORM - Complete Package",
	}

	// Add all core commands
	allRoot.AddCommand(initCmd)
	allRoot.AddCommand(migrateCmd)
	allRoot.AddCommand(applyCmd)
	allRoot.AddCommand(downCmd)
	allRoot.AddCommand(statusCmd)
	allRoot.AddCommand(pullCmd)
	allRoot.AddCommand(resetCmd)
	allRoot.AddCommand(rawCmd)
	allRoot.AddCommand(branchCmd)
	allRoot.AddCommand(checkoutCmd)
	allRoot.AddCommand(genCmd)
	allRoot.AddCommand(exportCmd)

	// Add studio command
	allRoot.AddCommand(studioCmd)

	// Add seed command
	allRoot.AddCommand(seedCmd)

	return allRoot.Execute()
}
