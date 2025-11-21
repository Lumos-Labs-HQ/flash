//go:build plugins
// +build plugins

package cmd

import (
	"github.com/spf13/cobra"
)

// ExecuteCorePlugin executes core ORM commands (everything except studio)
func ExecuteCorePlugin() error {
	// Create a root command for 'core' plugin
	coreRoot := &cobra.Command{
		Use:   "flash",
		Short: "FlashORM - Core ORM Features",
	}

	// Add all core commands
	// Migration and schema management commands
	coreRoot.AddCommand(initCmd)
	coreRoot.AddCommand(migrateCmd)
	coreRoot.AddCommand(applyCmd)
	coreRoot.AddCommand(statusCmd)
	coreRoot.AddCommand(pullCmd)
	coreRoot.AddCommand(resetCmd)
	coreRoot.AddCommand(rawCmd)
	coreRoot.AddCommand(branchCmd)
	coreRoot.AddCommand(checkoutCmd)

	// Codegen command
	coreRoot.AddCommand(genCmd)

	// Export command
	coreRoot.AddCommand(exportCmd)

	return coreRoot.Execute()
}

// ExecuteStudioPlugin executes the studio plugin command
func ExecuteStudioPlugin() error {
	// Create a root command for studio plugin
	studioRoot := &cobra.Command{
		Use:   "flash",
		Short: "FlashORM Studio Plugin",
	}

	// Add studio command
	studioRoot.AddCommand(studioCmd)

	return studioRoot.Execute()
}

// ExecuteAllPlugin executes all commands (core + studio)
func ExecuteAllPlugin() error {
	// Create a root command for 'all' plugin
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
