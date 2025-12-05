//go:build dev
// +build dev

package cmd

func RegisterBaseCommands() {
	// Core ORM commands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(rawCmd)
	rootCmd.AddCommand(genCmd)
	rootCmd.AddCommand(exportCmd)

	// Branch commands
	rootCmd.AddCommand(branchCmd)
	rootCmd.AddCommand(checkoutCmd)

	// Studio command
	rootCmd.AddCommand(studioCmd)

	// Plugin management (for consistency)
	rootCmd.AddCommand(pluginsCmd)
	rootCmd.AddCommand(addPluginCmd)
	rootCmd.AddCommand(removePluginCmd)
}
