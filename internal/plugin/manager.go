package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
)

const (
	DefaultRepo  = "Lumos-Labs-HQ/flash"
	PluginPrefix = "flash-plugin-"
)

// Manager handles plugin operations
type Manager struct {
	config   PluginConfig
	registry *PluginRegistry
}

// plugin manager
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	pluginDir := filepath.Join(homeDir, ".flash", "plugins")
	registryFile := filepath.Join(homeDir, ".flash", "registry.json")

	config := PluginConfig{
		PluginDir:    pluginDir,
		RegistryFile: registryFile,
		DefaultRepo:  DefaultRepo,
	}

	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugin directory: %w", err)
	}

	manager := &Manager{
		config: config,
	}

	if err := manager.loadRegistry(); err != nil {
		manager.registry = &PluginRegistry{
			Plugins: make(map[string]PluginInfo),
			Updated: time.Now(),
		}
		if err := manager.saveRegistry(); err != nil {
			return nil, fmt.Errorf("failed to save registry: %w", err)
		}
	}

	return manager, nil
}

// InstallPlugin downloads and installs a plugin
func (m *Manager) InstallPlugin(name, version string) error {
	validPlugins := GetAllPlugins()
	isValid := false
	for _, validName := range validPlugins {
		if validName == name {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("unknown plugin '%s', available plugins: %v", name, validPlugins)
	}

	if version == "" {
		version = "latest"
	}

	if info, exists := m.registry.Plugins[name]; exists {
		if info.Version == version {
			color.Yellow("⚠️  Plugin '%s' version %s is already installed", name, version)
			return nil
		}
		color.Cyan("🔄 Updating plugin '%s' from %s to %s", name, info.Version, version)
	} else {
		color.Cyan("📦 Installing plugin '%s' version %s...", name, version)
	}

	binaryName := m.getBinaryName(name)
	downloadName := m.getDownloadName(name)

	var downloadURL string
	if version == "latest" {
		latestVersion, err := m.getLatestStableReleaseVersion()
		if err != nil {
			return fmt.Errorf("failed to get latest release version: %w", err)
		}
		downloadURL = fmt.Sprintf("https://github.com/%s/releases/download/%s/%s",
			m.config.DefaultRepo, latestVersion, downloadName)
	} else if version == "beta" {
		latestBeta, err := m.getLatestBetaReleaseVersion()
		if err != nil {
			return fmt.Errorf("failed to get latest beta release version: %w", err)
		}
		downloadURL = fmt.Sprintf("https://github.com/%s/releases/download/%s/%s",
			m.config.DefaultRepo, latestBeta, downloadName)
	} else {
		versionTag := version
		if !strings.HasPrefix(version, "v") {
			versionTag = "v" + version
		}
		downloadURL = fmt.Sprintf("https://github.com/%s/releases/download/%s/%s",
			m.config.DefaultRepo, versionTag, downloadName)
	}

	pluginPath := filepath.Join(m.config.PluginDir, binaryName)
	if err := m.downloadFile(downloadURL, pluginPath); err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}

	if err := os.Chmod(pluginPath, 0755); err != nil {
		return fmt.Errorf("failed to make plugin executable: %w", err)
	}

	fileInfo, err := os.Stat(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to get plugin file info: %w", err)
	}

	if fileInfo.Size() == 0 {
		os.Remove(pluginPath)
		return fmt.Errorf("downloaded plugin file is empty (download may have failed)")
	}

	if runtime.GOOS != "windows" {
		if err := m.validateBinary(pluginPath); err != nil {
			os.Remove(pluginPath)
			return fmt.Errorf("downloaded file is not a valid executable binary: %w", err)
		}
	}

	m.registry.Plugins[name] = PluginInfo{
		Name:        name,
		Version:     version,
		Description: GetPluginDescription(name),
		Commands:    GetPluginCommands(name),
		InstallDate: time.Now(),
		Size:        fileInfo.Size(),
	}
	m.registry.Updated = time.Now()

	if err := m.saveRegistry(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	color.Green("✅ Plugin '%s' installed successfully!", name)
	color.Cyan("📝 Available commands: %s", strings.Join(GetPluginCommands(name), ", "))

	return nil
}

// RemovePlugin removes an installed plugin
func (m *Manager) RemovePlugin(name string) error {
	if _, exists := m.registry.Plugins[name]; !exists {
		return fmt.Errorf("plugin '%s' is not installed", name)
	}

	color.Cyan("🗑️  Removing plugin '%s'...", name)

	binaryName := m.getBinaryName(name)
	pluginPath := filepath.Join(m.config.PluginDir, binaryName)
	fileRemoveErr := os.Remove(pluginPath)
	if fileRemoveErr != nil && !os.IsNotExist(fileRemoveErr) {
		color.Yellow("⚠️  Warning: Failed to remove plugin binary: %v", fileRemoveErr)
		color.Yellow("    You may need to manually delete: %s", pluginPath)
	}

	delete(m.registry.Plugins, name)
	m.registry.Updated = time.Now()

	if err := m.saveRegistry(); err != nil {
		if fileRemoveErr == nil {
			color.Red("❌ Failed to update registry after removing binary")
			color.Yellow("⚠️  Plugin '%s' is in an inconsistent state", name)
		}
		return fmt.Errorf("failed to save registry: %w", err)
	}

	if fileRemoveErr != nil && !os.IsNotExist(fileRemoveErr) {
		color.Yellow("⚠️  Plugin '%s' removed from registry but binary file could not be deleted", name)
	} else {
		color.Green("✅ Plugin '%s' removed successfully!", name)
	}

	return nil
}

// ListPlugins returns all installed plugins
func (m *Manager) ListPlugins() []PluginInfo {
	if m.registry == nil || m.registry.Plugins == nil {
		return []PluginInfo{}
	}
	plugins := make([]PluginInfo, 0, len(m.registry.Plugins))
	for _, info := range m.registry.Plugins {
		plugins = append(plugins, info)
	}
	return plugins
}

// IsPluginInstalled checks if a plugin is installed
func (m *Manager) IsPluginInstalled(name string) bool {
	if m.registry == nil || m.registry.Plugins == nil {
		return false
	}
	_, exists := m.registry.Plugins[name]
	return exists
}

// GetPluginPath returns the path to a plugin binary
func (m *Manager) GetPluginPath(name string) string {
	binaryName := m.getBinaryName(name)
	return filepath.Join(m.config.PluginDir, binaryName)
}

// ExecutePlugin executes a plugin command
func (m *Manager) ExecutePlugin(pluginName string, args []string) error {
	if !m.IsPluginInstalled(pluginName) {
		return fmt.Errorf("plugin '%s' is not installed", pluginName)
	}

	pluginPath := m.GetPluginPath(pluginName)

	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin binary not found at %s (registry may be corrupted)", pluginPath)
	}

	cmd := exec.Command(pluginPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("plugin '%s' exited with code %d", pluginName, exitErr.ExitCode())
		}
		return fmt.Errorf("failed to execute plugin '%s': %w", pluginName, err)
	}

	return nil
}

// GetPluginInfo returns information about an installed plugin
func (m *Manager) GetPluginInfo(name string) (PluginInfo, error) {
	info, exists := m.registry.Plugins[name]
	if !exists {
		return PluginInfo{}, fmt.Errorf("plugin '%s' is not installed", name)
	}
	return info, nil
}

// FetchAvailablePlugins fetches metadata about available plugins from GitHub
func (m *Manager) FetchAvailablePlugins() ([]AvailablePlugin, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", m.config.DefaultRepo)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	var releases []struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}

	release := releases[0]
	version := strings.TrimPrefix(release.TagName, "v")

	platform := runtime.GOOS
	arch := runtime.GOARCH

	availablePlugins := []AvailablePlugin{}
	pluginNames := GetAllPlugins()

	for _, name := range pluginNames {
		expectedName := fmt.Sprintf("flash-plugin-%s-%s-%s", name, platform, arch)
		if platform == "windows" {
			expectedName += ".exe"
		}

		hasAssets := false
		var totalSize int64
		var currentPlatformSize int64

		for _, asset := range release.Assets {
			if strings.HasPrefix(asset.Name, fmt.Sprintf("flash-plugin-%s-", name)) {
				hasAssets = true
				totalSize += asset.Size

				if asset.Name == expectedName {
					currentPlatformSize = asset.Size
				}
			}
		}

		if hasAssets {
			size := currentPlatformSize
			if size == 0 && totalSize > 0 {
				assetCount := 0
				for _, asset := range release.Assets {
					if strings.HasPrefix(asset.Name, fmt.Sprintf("flash-plugin-%s-", name)) {
						assetCount++
					}
				}
				if assetCount > 0 {
					size = totalSize / int64(assetCount)
				}
			}

			availablePlugins = append(availablePlugins, AvailablePlugin{
				Name:        name,
				Version:     version,
				Description: GetPluginDescription(name),
				Commands:    GetPluginCommands(name),
				Size:        size,
			})
		}
	}

	return availablePlugins, nil
}

// downloadFile downloads a file from a URL
func (m *Manager) downloadFile(url, filepath string) error {
	color.Cyan("📥 Downloading from: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("plugin binary not found (404). The plugin may not be built for your platform (%s/%s) or the release doesn't exist", runtime.GOOS, runtime.GOARCH)
		}
		return fmt.Errorf("download failed with status: %s (code: %d)", resp.Status, resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Show progress with animation
	totalSize := resp.ContentLength
	downloaded := int64(0)
	buffer := make([]byte, 32*1024) // 32KB buffer
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerIdx := 0

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			_, writeErr := out.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write file: %w", writeErr)
			}
			downloaded += int64(n)

			if totalSize > 0 {
				percentage := float64(downloaded) / float64(totalSize) * 100
				barWidth := 40
				filledWidth := int(percentage * float64(barWidth) / 100)
				bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)

				downloadedMB := float64(downloaded) / (1024 * 1024)
				totalMB := float64(totalSize) / (1024 * 1024)

				fmt.Printf("\r%s %s %.1f%% (%.1f/%.1f MB)",
					color.CyanString(spinner[spinnerIdx]),
					bar,
					percentage,
					downloadedMB,
					totalMB)
				spinnerIdx = (spinnerIdx + 1) % len(spinner)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
	}
	fmt.Println()

	return nil
}

// getBinaryName returns the platform-specific binary name
func (m *Manager) getBinaryName(pluginName string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("%s%s.exe", PluginPrefix, pluginName)
	}
	return fmt.Sprintf("%s%s", PluginPrefix, pluginName)
}

// getDownloadName returns the download file name for the current platform
func (m *Manager) getDownloadName(pluginName string) string {
	platform := runtime.GOOS
	arch := runtime.GOARCH

	if runtime.GOOS == "windows" {
		return fmt.Sprintf("%s%s-%s-%s.exe", PluginPrefix, pluginName, platform, arch)
	}
	return fmt.Sprintf("%s%s-%s-%s", PluginPrefix, pluginName, platform, arch)
}

// validateBinary validates that the downloaded file is a valid executable binary
func (m *Manager) validateBinary(pluginPath string) error {
	file, err := os.Open(pluginPath)
	if err != nil {
		return err
	}
	defer file.Close()

	magic := make([]byte, 4)
	if _, err := file.Read(magic); err != nil {
		return err
	}

	// Check for ELF (Linux) or Mach-O (macOS) magic numbers
	isELF := magic[0] == 0x7f && magic[1] == 0x45 && magic[2] == 0x4c && magic[3] == 0x46
	isMachO64 := magic[0] == 0xcf && magic[1] == 0xfa && magic[2] == 0xed && magic[3] == 0xfe
	isMachO32 := magic[0] == 0xce && magic[1] == 0xfa && magic[2] == 0xed && magic[3] == 0xfe

	if !isELF && !isMachO64 && !isMachO32 {
		return fmt.Errorf("invalid binary format")
	}

	return nil
}

// getLatestStableReleaseVersion fetches the latest stable (non-prerelease) version from GitHub
func (m *Manager) getLatestStableReleaseVersion() (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", m.config.DefaultRepo)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	var releases []struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no releases found")
	}

	// Find the first stable release (non-prerelease)
	for _, release := range releases {
		if !release.Prerelease {
			return release.TagName, nil
		}
	}

	// If no stable releases found, return the latest (even if prerelease)
	return releases[0].TagName, nil
}

// getLatestBetaReleaseVersion fetches the latest beta (prerelease) version from GitHub
func (m *Manager) getLatestBetaReleaseVersion() (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", m.config.DefaultRepo)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	var releases []struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no releases found")
	}

	// Find the first prerelease
	for _, release := range releases {
		if release.Prerelease {
			return release.TagName, nil
		}
	}

	return "", fmt.Errorf("no beta releases found")
}

// loadRegistry loads the plugin registry from disk
func (m *Manager) loadRegistry() error {
	data, err := os.ReadFile(m.config.RegistryFile)
	if err != nil {
		return err
	}

	var registry PluginRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return err
	}

	m.registry = &registry
	return nil
}

// saveRegistry saves the plugin registry to disk
func (m *Manager) saveRegistry() error {
	data, err := json.MarshalIndent(m.registry, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.config.RegistryFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(m.config.RegistryFile, data, 0644)
}
