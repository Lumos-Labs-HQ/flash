package plugin

import (
	"testing"
)

func TestGetRequiredPlugin_KnownCommands(t *testing.T) {
	coreCommands := []string{
		"init", "migrate", "apply", "down", "status",
		"pull", "reset", "raw", "branch", "checkout",
		"gen", "export", "seed",
	}
	for _, cmd := range coreCommands {
		plugin, ok := GetRequiredPlugin(cmd)
		if !ok {
			t.Errorf("command %q: expected plugin mapping, got none", cmd)
		}
		if plugin != "core" {
			t.Errorf("command %q: plugin = %q, want core", cmd, plugin)
		}
	}
}

func TestGetRequiredPlugin_Studio(t *testing.T) {
	plugin, ok := GetRequiredPlugin("studio")
	if !ok {
		t.Error("studio: expected plugin mapping, got none")
	}
	if plugin != "studio" {
		t.Errorf("studio: plugin = %q, want studio", plugin)
	}
}

func TestGetRequiredPlugin_Unknown(t *testing.T) {
	_, ok := GetRequiredPlugin("nonexistent_command")
	if ok {
		t.Error("nonexistent_command: expected no mapping, got one")
	}
}

func TestGetPluginDescription_Known(t *testing.T) {
	for _, name := range []string{"core", "studio"} {
		desc := GetPluginDescription(name)
		if desc == "No description available" {
			t.Errorf("plugin %q has no description", name)
		}
	}
}

func TestGetPluginDescription_Unknown(t *testing.T) {
	desc := GetPluginDescription("nonexistent")
	if desc != "No description available" {
		t.Errorf("unknown plugin description = %q, want 'No description available'", desc)
	}
}

func TestGetPluginCommands_Core(t *testing.T) {
	cmds := GetPluginCommands("core")
	if len(cmds) == 0 {
		t.Error("core plugin should have commands")
	}
	// Verify key commands are present
	cmdSet := make(map[string]bool, len(cmds))
	for _, c := range cmds {
		cmdSet[c] = true
	}
	for _, expected := range []string{"init", "migrate", "apply", "gen", "seed"} {
		if !cmdSet[expected] {
			t.Errorf("core plugin missing command %q", expected)
		}
	}
}

func TestGetPluginCommands_Unknown(t *testing.T) {
	cmds := GetPluginCommands("nonexistent")
	if len(cmds) != 0 {
		t.Errorf("unknown plugin commands = %v, want empty", cmds)
	}
}

func TestGetAllPlugins_ContainsCoreAndStudio(t *testing.T) {
	plugins := GetAllPlugins()
	pluginSet := make(map[string]bool, len(plugins))
	for _, p := range plugins {
		pluginSet[p] = true
	}
	if !pluginSet["core"] {
		t.Error("GetAllPlugins missing 'core'")
	}
	if !pluginSet["studio"] {
		t.Error("GetAllPlugins missing 'studio'")
	}
}

func TestCommandPluginMap_Completeness(t *testing.T) {
	// Every command in CommandPluginMap must reference a plugin in PluginDescriptions.
	for cmd, pluginName := range CommandPluginMap {
		if _, ok := PluginDescriptions[pluginName]; !ok {
			t.Errorf("command %q maps to unknown plugin %q", cmd, pluginName)
		}
	}
}

func TestPluginCommands_AllMappedInRegistry(t *testing.T) {
	// Every command listed in PluginCommands must appear in CommandPluginMap.
	for pluginName, cmds := range PluginCommands {
		for _, cmd := range cmds {
			if mapped, ok := CommandPluginMap[cmd]; !ok {
				t.Errorf("command %q from plugin %q not in CommandPluginMap", cmd, pluginName)
			} else if mapped != pluginName {
				t.Errorf("command %q: CommandPluginMap says %q, PluginCommands says %q", cmd, mapped, pluginName)
			}
		}
	}
}
