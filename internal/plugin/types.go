package plugin

import (
	"time"
)

// PluginInfo represents metadata about an installed plugin
type PluginInfo struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Commands    []string  `json:"commands"`
	InstallDate time.Time `json:"install_date"`
	Size        int64     `json:"size"`
}

// AvailablePlugin represents metadata about a plugin available for installation
type AvailablePlugin struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Commands    []string `json:"commands"`
	Size        int64    `json:"size,omitempty"`
	DownloadURL string   `json:"download_url,omitempty"`
}

// PluginManifest represents the manifest file for a plugin
type PluginManifest struct {
	Name           string        `json:"name"`
	Version        string        `json:"version"`
	Description    string        `json:"description"`
	Commands       []CommandInfo `json:"commands"`
	MinCoreVersion string        `json:"min_core_version"`
	Author         string        `json:"author"`
	Repository     string        `json:"repository"`
}

// CommandInfo represents information about a command provided by a plugin
type CommandInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
}

// PluginRegistry represents the local registry of installed plugins
type PluginRegistry struct {
	Plugins map[string]PluginInfo `json:"plugins"`
	Updated time.Time             `json:"updated"`
}

// PluginConfig represents configuration for the plugin system
type PluginConfig struct {
	PluginDir    string `json:"plugin_dir"`
	RegistryFile string `json:"registry_file"`
	DefaultRepo  string `json:"default_repo"`
}

// DownloadProgress represents download progress information
type DownloadProgress struct {
	Total      int64
	Downloaded int64
	Percentage float64
}
