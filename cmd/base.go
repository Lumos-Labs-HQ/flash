//go:build !dev
// +build !dev

package cmd

func RegisterBaseCommands() {
	rootCmd.AddCommand(pluginsCmd)
	rootCmd.AddCommand(addPluginCmd)
	rootCmd.AddCommand(removePluginCmd)
}
