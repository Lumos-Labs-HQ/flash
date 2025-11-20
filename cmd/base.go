package cmd

// RegisterBaseCommands registers only the base CLI commands (plugin management)
func RegisterBaseCommands() {
	// Plugin management commands
	rootCmd.AddCommand(pluginsCmd)
	rootCmd.AddCommand(addPluginCmd)
	rootCmd.AddCommand(removePluginCmd)
}
