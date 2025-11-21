package plugin

// CommandPluginMap maps command names to their required plugin
var CommandPluginMap = map[string]string{
	// Core ORM commands - requires core plugin
	"init":     "core",
	"migrate":  "core",
	"apply":    "core",
	"status":   "core",
	"pull":     "core",
	"reset":    "core",
	"raw":      "core",
	"branch":   "core",
	"checkout": "core",
	"gen":      "core",
	"export":   "core",

	// Studio commands - requires studio plugin
	"studio": "studio",
}

// PluginDescriptions provides descriptions for each plugin
var PluginDescriptions = map[string]string{
	"core":   "Complete ORM features (migrations, codegen, export, schema management)",
	"studio": "Visual database editor and management interface",
	"all":    "Complete package with all features (core + studio)",
}

// PluginCommands lists all commands provided by each plugin
var PluginCommands = map[string][]string{
	"core":   {"init", "migrate", "apply", "status", "pull", "reset", "raw", "branch", "checkout", "gen", "export"},
	"studio": {"studio"},
	"all":    {"init", "migrate", "apply", "status", "pull", "reset", "raw", "branch", "checkout", "gen", "export", "studio"},
}

// GetRequiredPlugin returns the plugin name required for a given command
func GetRequiredPlugin(command string) (string, bool) {
	plugin, exists := CommandPluginMap[command]
	return plugin, exists
}

// GetPluginDescription returns the description for a plugin
func GetPluginDescription(pluginName string) string {
	if desc, exists := PluginDescriptions[pluginName]; exists {
		return desc
	}
	return "No description available"
}

// GetPluginCommands returns the list of commands provided by a plugin
func GetPluginCommands(pluginName string) []string {
	if commands, exists := PluginCommands[pluginName]; exists {
		return commands
	}
	return []string{}
}

// GetAllPlugins returns a list of all available plugins
func GetAllPlugins() []string {
	plugins := make([]string, 0, len(PluginDescriptions))
	for name := range PluginDescriptions {
		plugins = append(plugins, name)
	}
	return plugins
}
