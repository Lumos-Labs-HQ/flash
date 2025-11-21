//go:build plugins
// +build plugins

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
	coreRoot.AddCommand(statusCmd)
	coreRoot.AddCommand(pullCmd)
	coreRoot.AddCommand(resetCmd)
	coreRoot.AddCommand(rawCmd)
	coreRoot.AddCommand(branchCmd)
	coreRoot.AddCommand(checkoutCmd)

	coreRoot.AddCommand(genCmd)

	coreRoot.AddCommand(exportCmd)

	return coreRoot.Execute()
}

func ExecuteStudioPlugin() error {
	studioRoot := &cobra.Command{
		Use:   "flash",
		Short: "FlashORM Studio Plugin",
	}

	studioRoot.AddCommand(studioCmd)

	return studioRoot.Execute()
}

func ExecuteAllPlugin() error {
	allRoot := &cobra.Command{
		Use:   "flash",
		Short: "FlashORM - Complete Package",
	}

	// Add all core commands
	allRoot.AddCommand(initCmd)
	allRoot.AddCommand(migrateCmd)
	allRoot.AddCommand(applyCmd)
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

	return allRoot.Execute()
}
