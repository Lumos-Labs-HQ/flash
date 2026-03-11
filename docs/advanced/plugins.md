---
title: Plugin System
description: Flash ORM's modular plugin architecture
---

# Flash ORM Plugin System

Flash ORM uses a modular plugin architecture that allows you to install only the features you need, significantly reducing binary size and installation footprint.

## Overview

The base Flash ORM CLI is a minimal binary that provides:
- Version information (`flash --version`)
- Plugin management (`flash plugins`, `flash add-plug`, `flash rm-plug`)
- Command metadata and help
- Automatic plugin download on first use

All actual ORM functionality is provided through plugins.

## Available Plugins

### Core Plugin (`core`)

**Description:** Complete ORM features — migrations, code generation, seeding, schema management, and export.

**Includes:**
- `init` — Initialize a new FlashORM project
- `migrate` — Create new migration files
- `apply` — Apply pending migrations
- `status` — Check migration status
- `pull` — Pull current database schema
- `reset` — Reset database and reapply all migrations
- `down` — Rollback migrations
- `raw` — Execute raw SQL queries
- `branch` — Manage database schema branches
- `gen` — Generate type-safe code (Go, TypeScript, Python)
- `export` — Export database to JSON, CSV, or SQLite
- `seed` — Seed database with realistic fake data

**Auto-install:** The core plugin installs automatically the first time you run any ORM command. You do not need to install it manually.

### Studio Plugin (`studio`)

**Description:** Visual database editor and management interface.

**Includes:**
- `studio` — Launch the web-based database GUI
  - Browse and edit table data
  - Export and import database (Schema Only, Data Only, Complete)
  - Visual schema viewer with relationship graph
  - SQL query runner with CSV export
  - Branch management interface

**Install manually** when you need the visual editor:
```bash
flash add-plug studio
```

## Plugin Management

### List Installed Plugins

```bash
flash plugins
```

### Install a Plugin

```bash
flash add-plug core
flash add-plug studio
```

### Remove a Plugin

```bash
flash rm-plug studio
```

### Update Plugins

```bash
# Update all installed plugins
flash update

# Update plugins and the flash binary
flash update --self

# Update only the flash binary
flash update --self-only
```

## How Plugin Loading Works

When you run a command, Flash ORM:

1. Checks if the command is built into the base CLI
2. If not, checks the plugin registry for which plugin provides the command
3. If the required plugin is not installed, downloads it automatically (core only)
4. Delegates execution to the plugin binary with your arguments and environment

## Plugin Storage

Plugins are stored in your home directory:

```
~/.flash/plugins/
├── flash-plugin-core
├── flash-plugin-studio
└── registry.json
```

## Distribution

Plugins are distributed through GitHub Releases and downloaded automatically when needed. Flash ORM determines your platform and architecture, downloads the appropriate binary, verifies its checksum, and saves it to the plugin directory.

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
