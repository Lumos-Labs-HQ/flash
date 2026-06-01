//go:build plugin_core
// +build plugin_core

package cmd

import (
	"github.com/spf13/cobra"
)

func ExecuteCorePlugin() error {
	coreRoot := &cobra.Command{
		Use:   "flash",
		Short: "FlashORM - Core ORM Features",
	}

	// Add all core commands
	coreRoot.AddCommand(initCmd)
	coreRoot.AddCommand(migrateCmd)
	coreRoot.AddCommand(applyCmd)
	coreRoot.AddCommand(downCmd)
	coreRoot.AddCommand(statusCmd)
	coreRoot.AddCommand(pullCmd)
	coreRoot.AddCommand(resetCmd)
	coreRoot.AddCommand(rawCmd)
	coreRoot.AddCommand(branchCmd)
	coreRoot.AddCommand(checkoutCmd)
	coreRoot.AddCommand(genCmd)
	coreRoot.AddCommand(exportCmd)
	coreRoot.AddCommand(seedCmd)

	return coreRoot.Execute()
}
