---
title: Plugin System
description: Flash ORM's modular plugin architecture
---

# Flash ORM Plugin System

Flash ORM uses a modular plugin architecture that allows you to install only the features you need, significantly reducing binary size and installation footprint.

## Overview

The base Flash ORM CLI is a minimal binary (~5-10 MB) that provides:
- Version information (`flash --version`)
- Plugin management (`flash plugins`, `flash add-plug`, `flash rm-plug`)
- Command metadata and help
- Automatic plugin requirement detection and delegation

All actual ORM functionality is provided through plugins that you install separately.

## Available Plugins

### 1. Core Plugin (`core`)

**Size:** ~30 MB  
**Description:** Complete ORM features (migrations, codegen, export, schema management)

**Includes:**
- `init` - Initialize a new FlashORM project
- `migrate` - Create new migration files
- `apply` - Apply pending migrations to database
- `status` - Check migration status
- `pull` - Pull current database schema
- `reset` - Reset database and reapply all migrations
- `raw` - Execute raw SQL queries or files
- `branch` - Manage database schema branches
- `checkout` - Switch between schema branches
- `gen` - Generate type-safe code (Go, TypeScript, Python)
- `export` - Export database to JSON, CSV, or SQLite

**Use Case:** Production environments, CI/CD pipelines, developers who prefer CLI workflows

### 2. Studio Plugin (`studio`)

**Size:** ~29 MB  
**Description:** Visual database editor and management interface

**Includes:**
- `studio` - Launch web-based database GUI
  - View and edit table data
  - Visual schema editor
  - SQL query runner
  - Relationship visualization
  - Branch management interface

**Use Case:** Developers who prefer visual tools, database administration, rapid prototyping

### 3. All Plugin (`all`)

**Size:** ~30 MB  
**Description:** Complete package combining core + studio

**Includes:** All commands from both `core` and `studio` plugins

**Use Case:** Full-featured local development, teams using both CLI and GUI workflows

## Installation

### First Time Setup

When you first install FlashORM, you get only the base CLI:

```bash
# Install via npm (base CLI only)
npm install -g flashorm

# Or via pip
pip install flashorm

# Or download binary directly
curl -sL https://github.com/Lumos-Labs-HQ/flash/releases/latest/download/flash-linux-amd64 -o flash
chmod +x flash
```

### Installing Plugins

Install the plugin(s) you need:

```bash
# Option 1: Core ORM features only (smallest footprint)
flash add-plug core

# Option 2: Studio only (for GUI-based workflows)
flash add-plug studio

# Option 3: Everything (most convenient)
flash add-plug all
```

### Version-Specific Installation

```bash
# Install specific version
flash add-plug core@2.1.11

# Install latest version (default)
flash add-plug core@latest

# Install beta version
flash add-plug core@beta
```

## Plugin Management

### List Installed Plugins

```bash
flash plugins
```

Output:
```
Installed Plugins:
✅ core v2.1.11 - Complete ORM features
✅ studio v2.1.11 - Visual database editor

Available Plugins:
- all v2.1.11 - Complete package (core + studio)
```

### Update Plugins

```bash
# Update all plugins
flash plugins update

# Update specific plugin
flash plugins update core

# Check for updates
flash plugins outdated
```

### Remove Plugins

```bash
# Remove specific plugin
flash rm-plug studio

# Remove all plugins
flash rm-plug all
```

## Plugin Architecture

### Plugin Structure

Each plugin is a self-contained binary with its own dependencies:

```
~/.flash/plugins/
├── core/
│   ├── flash-plugin-core  # Plugin binary
│   ├── manifest.json      # Plugin metadata
│   └── checksums.txt      # File integrity
├── studio/
│   ├── flash-plugin-studio
│   ├── manifest.json
│   └── checksums.txt
```

### Plugin Manifest

```json
{
  "name": "core",
  "version": "2.1.11",
  "description": "Complete ORM features",
  "author": "Lumos Labs",
  "commands": [
    "init",
    "migrate",
    "apply",
    "status",
    "pull",
    "reset",
    "raw",
    "branch",
    "checkout",
    "gen",
    "export"
  ],
  "dependencies": [],
  "platforms": ["linux", "darwin", "windows"],
  "architectures": ["amd64", "arm64"],
  "checksums": {
    "linux-amd64": "abc123...",
    "darwin-amd64": "def456...",
    "windows-amd64": "ghi789..."
  }
}
```

### Plugin Loading

When you run a command, Flash ORM:

1. **Checks if command exists** in base CLI
2. **If not found**, looks for plugin that provides the command
3. **Loads plugin binary** and delegates execution
4. **Passes arguments and environment** to plugin

```bash
# User runs: flash migrate "add users table"
# Flash ORM:
# 1. Command "migrate" not in base CLI
# 2. Checks plugin registry for "migrate" command
# 3. Finds "core" plugin provides "migrate"
# 4. Executes: ~/.flash/plugins/core/flash-plugin-core migrate "add users table"
```

## Development

### Building Plugins

Plugins are built as separate Go binaries with build tags:

```go
//go:build plugins

package main

import (
    "github.com/Lumos-Labs-HQ/flash/cmd"
)

func main() {
    // Register plugin-specific commands
    cmd.RegisterCoreCommands()

    if err := cmd.Execute(); err != nil {
        // Handle error
    }
}
```

### Plugin Commands

```go
// cmd/migrate.go
//go:build plugins

package cmd

import (
    "github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
    Use:   "migrate [name]",
    Short: "Create a new migration",
    Long:  `Create a new migration file with the specified name.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Migration logic here
        return nil
    },
}

func init() {
    rootCmd.AddCommand(migrateCmd)
}
```

### Plugin Registry

The plugin registry manages plugin metadata and dependencies:

```go
type PluginRegistry struct {
    plugins map[string]*PluginInfo
}

type PluginInfo struct {
    Name         string
    Version      string
    Commands     []string
    Dependencies []string
    Path         string
    Checksum     string
}
```

## Distribution

### Plugin Repository

Plugins are distributed through GitHub Releases:

```
https://github.com/Lumos-Labs-HQ/flash/releases/download/v2.1.11/
├── flash-linux-amd64.tar.gz          # Base CLI
├── flash-plugin-core-linux-amd64.tar.gz
├── flash-plugin-studio-linux-amd64.tar.gz
├── flash-plugin-all-linux-amd64.tar.gz
```

### Automatic Downloads

When installing plugins, Flash ORM:

1. **Fetches manifest** from GitHub API
2. **Determines platform/architecture**
3. **Downloads appropriate binary**
4. **Verifies checksum**
5. **Extracts to plugin directory**
6. **Updates registry**

### Offline Installation

For air-gapped environments:

```bash
# Download plugins manually
curl -L -o core.tar.gz https://github.com/Lumos-Labs-HQ/flash/releases/download/v2.1.11/flash-plugin-core-linux-amd64.tar.gz

# Install from local file
flash add-plug core --file ./core.tar.gz
```

## Security

### Plugin Verification

All plugins are verified before installation:

- **Checksum validation** against published manifest
- **Signature verification** (planned)
- **Sandbox execution** with limited permissions

### Secure Updates

Plugin updates require explicit confirmation:

```bash
flash plugins update
# Output: "Update core from v2.1.10 to v2.1.11? (y/N)"
```

### Permission Model

Plugins run with the same permissions as the base CLI:

- **No elevated privileges**
- **Access to user files and databases**
- **Network access for database connections**
- **No system-level modifications**

## Performance

### Startup Time

Plugin architecture optimizes startup:

- **Base CLI**: ~50ms startup time
- **Plugin loading**: ~100-200ms additional
- **Total**: ~150-250ms (comparable to monolithic CLI)

### Memory Usage

Memory-efficient design:

- **Base CLI**: ~10MB RAM
- **Core plugin**: ~25MB additional
- **Studio plugin**: ~30MB additional (includes web server)

### Disk Space

Minimal footprint:

- **Base CLI**: ~8MB
- **Core plugin**: ~22MB
- **Studio plugin**: ~21MB
- **Total with all plugins**: ~51MB

## Use Cases

### Development Environments

```bash
# Minimal setup for CI/CD
npm install -g flashorm
flash add-plug core

# Full development setup
flash add-plug all
```

### Production Deployments

```bash
# Production servers (CLI only)
flash add-plug core

# Development servers (with studio)
flash add-plug all
```

### Team Workflows

```bash
# Backend developers
flash add-plug core

# Full-stack developers
flash add-plug all

# Database administrators
flash add-plug studio
```

### Resource-Constrained Environments

```bash
# IoT devices, containers
flash add-plug core  # Minimal footprint

# Development containers
flash add-plug all   # Full functionality
```

## Troubleshooting

### Plugin Installation Issues

**Permission denied**
```bash
# Check plugin directory permissions
ls -la ~/.flash/

# Fix permissions
chmod 755 ~/.flash/
chmod 755 ~/.flash/plugins/
```

**Network issues**
```bash
# Check connectivity
curl -I https://github.com/Lumos-Labs-HQ/flash/releases

# Use proxy if needed
export HTTPS_PROXY=http://proxy.company.com:8080
flash add-plug core
```

**Checksum verification failed**
```bash
# Clear plugin cache
rm -rf ~/.flash/plugins/
flash add-plug core
```

### Plugin Loading Issues

**Plugin not found**
```bash
# Check plugin installation
flash plugins

# Reinstall plugin
flash rm-plug core
flash add-plug core
```

**Command not available**
```bash
# Check if plugin provides command
flash plugins --commands

# Update plugin
flash plugins update core
```

### Performance Issues

**Slow plugin loading**
```bash
# Check disk I/O
iostat -x 1

# Check available memory
free -h

# Restart with clean state
flash rm-plug all
flash add-plug core
```

## Future Enhancements

### Planned Features

- **Plugin Marketplace**: Community plugin repository
- **Plugin Dependencies**: Automatic dependency resolution
- **Plugin Updates**: Background update notifications
- **Plugin Sandboxing**: Enhanced security isolation
- **Plugin Metrics**: Usage and performance monitoring

### Custom Plugins

Support for user-created plugins:

```go
// Custom plugin example
type CustomPlugin struct{}

func (p *CustomPlugin) Name() string { return "analytics" }
func (p *CustomPlugin) Commands() []cobra.Command {
    return []cobra.Command{
        {
            Use:   "analytics",
            Short: "Run analytics queries",
            RunE:  runAnalytics,
        },
    }
}
```

The plugin system makes Flash ORM incredibly flexible and efficient. You can install exactly the features you need, when you need them, without bloat or unnecessary dependencies.
