# FlashORM Plugin System

FlashORM uses a modular plugin architecture that allows you to install only the features you need, significantly reducing binary size and installation footprint.

## Overview

The base FlashORM CLI is a minimal binary (~5-10 MB) that provides:
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
flash add-plug core  # same as above
```

## Usage Examples

### Scenario 1: Core-Only Workflow

```bash
# Install core plugin
flash add-plug core

# Initialize project
flash init --postgresql

# Create migrations
flash migrate "create users table"

# Apply migrations
flash apply

# Generate code
flash gen
```

### Scenario 2: Studio-Only Workflow

```bash
# Install studio plugin
flash add-plug studio

# Launch studio with direct DB connection
flash studio --db "postgresql://user:pass@localhost:5432/mydb"

# Or use config file
flash studio
```

### Scenario 3: Combined Workflow

```bash
# Install both (or use 'all' plugin)
flash add-plug core
flash add-plug studio

# OR
flash add-plug all

# Use CLI for migrations
flash init
flash migrate "add email column"
flash apply

# Use studio for data management
flash studio
```

## Plugin Compatibility

Plugins are designed to work independently and together:

- **Core alone:** Full CLI-based ORM workflow
- **Studio alone:** GUI-based database management with direct connections
- **Core + Studio:** Complete workflow (same as 'all' plugin)
- **All plugin:** Equivalent to installing both core and studio

### Plugin Fallback Logic

When you run a command, FlashORM checks:
1. Is the specific required plugin installed? (e.g., `core` for `migrate`)
2. If not, is the `all` plugin installed?
3. If neither, show installation instructions

This means if you have `all` installed, you don't need individual plugins.

## Managing Plugins

### List Installed Plugins

```bash
flash plugins
```

Output:
```
📦 Installed Plugins (1)

NAME   VERSION   INSTALLED    SIZE      COMMANDS
----   -------   ---------    ----      --------
core   latest    2025-11-20   30.0 MB   8 commands

💡 Add more plugins: flash add-plug <plugin-name>
💡 Remove plugins: flash rm-plug <plugin-name>
💡 Check online plugins: flash plugins --online
```

### Check Available Plugins Online

Check all available plugins from GitHub repository:

```bash
flash plugins --online
```

Output:
```
🌐 Fetching available plugins from GitHub...

📦 Available Plugins (3)

NAME     STATUS            VERSION (Latest)  DESCRIPTION
----     ------            ----------------  -----------
core     ✓ Installed       2.1.11           Complete ORM features
studio   Not Installed     2.1.11           Visual database editor
all      Not Installed     2.1.11           Complete package

📝 Plugin Details:

  core
    Description: Complete ORM features (migrations, codegen, export, schema management)
    Commands: 10 commands
    Latest Version: 2.1.11
    Installed Version: 2.1.11

  studio
    Description: Visual database editor and management interface
    Commands: 1 commands
    Latest Version: 2.1.11
    → Install: flash add-plug studio

  all
    Description: Complete package with all features (core + studio)
    Commands: 11 commands
    Latest Version: 2.1.11
    → Install: flash add-plug all
```

This helps you:
- See what plugins are available
- Check if updates are available for installed plugins
- Compare installed versions with latest releases

### Remove Plugins

```bash
# Remove a plugin (with confirmation)
flash rm-plug studio

# Force removal without confirmation
flash rm-plug studio --force
```

Output:
```
⚠️  This will remove plugin 'studio' (vlatest)
   Commands that will become unavailable: [studio]

Continue? (y/N): y
🗑️  Removing plugin 'studio'...
✅ Plugin 'studio' removed successfully!
```

### Update Plugins

To update a plugin, simply reinstall it:

```bash
# Reinstall to get latest version
flash add-plug core

# Or install specific version
flash add-plug core@2.2.0
```

If already installed, it will update:
```
🔄 Updating plugin 'core' from 2.1.10 to 2.1.11
📥 Downloading from: https://github.com/...
✅ Plugin 'core' installed successfully!
```

## How It Works

### Plugin Detection and Delegation

When you run a command (e.g., `flash migrate`), the base CLI:

1. **Intercepts the command** before Cobra processes it
2. **Checks command-to-plugin mapping** in `internal/plugin/registry.go`
3. **Looks for required plugin** (e.g., `core` for `migrate`)
4. **Falls back to `all` plugin** if specific plugin not found
5. **Executes plugin binary** if found, passing all arguments
6. **Shows helpful error** if no plugin found

### Plugin Execution Flow

```
User: flash migrate "create users"
  ↓
Execute() in cmd/root.go
  ↓
Check if "migrate" requires plugin → Yes, requires "core"
  ↓
Is "core" installed? → Check registry
  ├─ Yes → Execute ~/.flash/plugins/flash-plugin-core migrate "create users"
  └─ No  → Is "all" installed?
      ├─ Yes → Execute ~/.flash/plugins/flash-plugin-all migrate "create users"
      └─ No  → Show error: "Command 'migrate' requires plugin 'core'"
```

### Plugin Storage

Plugins are standalone binaries stored in `~/.flash/plugins/`:

**Linux/macOS:**
```
~/.flash/
├── plugins/
│   ├── flash-plugin-core
│   ├── flash-plugin-studio
│   └── flash-plugin-all
└── registry.json
```

**Windows:**
```
%USERPROFILE%\.flash\
├── plugins\
│   ├── flash-plugin-core.exe
│   ├── flash-plugin-studio.exe
│   └── flash-plugin-all.exe
└── registry.json
```

### Plugin Registry

Plugin metadata is stored in `~/.flash/registry.json`:

```json
{
  "plugins": {
    "core": {
      "name": "core",
      "version": "latest",
      "description": "Complete ORM features (migrations, codegen, export, schema management)",
      "commands": ["init", "migrate", "apply", "status", "pull", "reset", "raw", "branch", "checkout", "gen", "export"],
      "install_date": "2025-11-20T11:32:00Z",
      "size": 31457280
    }
  },
  "updated": "2025-11-20T11:32:00Z"
}
```

### Download Process

Plugins are downloaded from GitHub releases:

```
https://github.com/Lumos-Labs-HQ/flash/releases/latest/download/flash-plugin-core-linux-amd64
https://github.com/Lumos-Labs-HQ/flash/releases/latest/download/flash-plugin-core-darwin-amd64
https://github.com/Lumos-Labs-HQ/flash/releases/latest/download/flash-plugin-core-windows-amd64.exe
```

The plugin manager:
1. Detects your platform (OS) and architecture (amd64/arm64)
2. Constructs the download URL
3. Downloads the binary with progress indicator
4. Makes it executable (Unix systems)
5. Updates the registry
6. Shows installation confirmation

## Benefits

### 1. Reduced Binary Size
- Base CLI: ~5-10 MB (vs ~50+ MB monolithic)
- Install only what you need
- Faster downloads and installations
- Smaller Docker images

### 2. Flexible Installation
- Start minimal, add features as needed
- Different team members can install different plugins
- Easier to update specific components
- No unnecessary dependencies

### 3. Better CI/CD
- CI pipelines can install only `core` (no GUI overhead)
- Production containers stay lean
- Faster build times
- Reduced attack surface

### 4. No Postinstall Scripts
- Plugins downloaded on-demand from GitHub releases
- No npm postinstall complexity
- Works with strict npm policies (--ignore-scripts)
- Cleaner package installation

### 5. Cross-Platform
- All plugins available for Linux, macOS, Windows
- Both AMD64 and ARM64 architectures
- Consistent experience across platforms
- Single binary per platform

### 6. Independent Updates
- Update plugins without updating base CLI
- Test new plugin versions independently
- Rollback individual plugins if needed
- Faster release cycles

## CI/CD Integration

### GitHub Actions

```yaml
name: Database Migrations

on: [push]

jobs:
  migrate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install FlashORM
        run: npm install -g flashorm
      
      - name: Install core plugin only
        run: flash add-plug core
      
      - name: Apply migrations
        run: flash apply
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
```

### Docker

```dockerfile
FROM node:20-alpine

# Install FlashORM base CLI
RUN npm install -g flashorm

# Install only core plugin (no studio)
RUN flash add-plug core

# Your app
COPY . /app
WORKDIR /app

CMD ["flash", "apply"]
```

### GitLab CI

```yaml
migrate:
  image: node:20
  script:
    - npm install -g flashorm
    - flash add-plug core
    - flash apply
  only:
    - main
```

## Troubleshooting

### Plugin Command Not Found

**Problem:** Running a command shows "missing required plugin"

```bash
$ flash migrate
❌ Command 'migrate' requires plugin 'core'

📦 Install it using: flash add-plug core
```

**Solution:**
```bash
# Install the required plugin
flash add-plug core
```

### Plugin Download Fails

**Problem:** Network error during plugin installation

```bash
$ flash add-plug core
📦 Installing plugin 'core' version latest...
📥 Downloading from: https://github.com/...
❌ Download failed: HTTP 404
```

**Solution:**
```bash
# Check GitHub connectivity
curl -I https://github.com

# Check if releases exist
curl -I https://github.com/Lumos-Labs-HQ/flash/releases/latest

# Retry installation
flash add-plug core

# Try specific version if latest fails
flash add-plug core@2.1.11
```

### Plugin Already Installed

**Problem:** Message shows plugin already installed but command doesn't work

```bash
$ flash add-plug core
⚠️  Plugin 'core' version latest is already installed
$ flash migrate
Error: unknown command "migrate" for "flash"
```

**Solution:**
```bash
# Remove and reinstall
flash rm-plug core --force
flash add-plug core

# Verify installation
flash plugins
```

### Permission Denied (Unix)

**Problem:** Plugin binary not executable

```bash
$ flash migrate
Error: fork/exec ~/.flash/plugins/flash-plugin-core: permission denied
```

**Solution:**
```bash
# Make plugins executable
chmod +x ~/.flash/plugins/flash-plugin-*

# Verify
ls -la ~/.flash/plugins/
```

### Registry Corruption

**Problem:** Registry file is corrupted or invalid

**Solution:**
```bash
# Backup old registry
mv ~/.flash/registry.json ~/.flash/registry.json.bak

# Create new registry
flash plugins

# Reinstall plugins
flash add-plug core
```

## Developer Guide

### Building Plugins Locally

```bash
# Build all plugins for all platforms
make build-all

# Build specific plugin
cd plugins/core
GOOS=linux GOARCH=amd64 go build -o flash-plugin-core-linux-amd64 .

cd plugins/studio
GOOS=linux GOARCH=amd64 go build -o flash-plugin-studio-linux-amd64 .

cd plugins/all
GOOS=linux GOARCH=amd64 go build -o flash-plugin-all-linux-amd64 .
```

### Plugin Structure

Each plugin is a standalone Go application:

```
plugins/
├── core/
│   ├── main.go       # Entry point
│   ├── go.mod        # Dependencies
│   └── go.sum
├── studio/
│   ├── main.go
│   ├── go.mod
│   └── go.sum
└── all/
    ├── main.go
    ├── go.mod
    └── go.sum
```

**Example: `plugins/core/main.go`**
```go
package main

import (
	"fmt"
	"os"
	"github.com/Lumos-Labs-HQ/flash/cmd"
)

func main() {
	if err := cmd.ExecuteCorePlugin(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

### Adding New Commands

1. **Create command** in `cmd/mycommand.go`:
```go
package cmd

import "github.com/spf13/cobra"

var myCmd = &cobra.Command{
	Use:   "my-command",
	Short: "My new command",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Implementation
		return nil
	},
}

func init() {
	// Don't register with rootCmd!
	// Registration happens in plugin executors
}
```

2. **Add to plugin executor** in `cmd/plugin_executors.go`:
```go
func ExecuteCorePlugin() error {
	coreRoot := &cobra.Command{
		Use:   "flash",
		Short: "FlashORM - Core ORM Features",
	}

	// Add your command
	coreRoot.AddCommand(myCmd)
	
	// ... other commands
	
	return coreRoot.Execute()
}
```

3. **Update registry** in `internal/plugin/registry.go`:
```go
var CommandPluginMap = map[string]string{
	"my-command": "core",
	// ... other commands
}

var PluginCommands = map[string][]string{
	"core": {"init", "migrate", ..., "my-command"},
	// ... other plugins
}
```

4. **Rebuild plugin**:
```bash
cd plugins/core
go build -o flash-plugin-core .
```

### Testing Locally

```bash
# Build base CLI
go build -o flash .

# Build plugin
cd plugins/core
go build -o flash-plugin-core .
cd ../..

# Copy plugin to local directory
mkdir -p ~/.flash/plugins
cp plugins/core/flash-plugin-core ~/.flash/plugins/

# Create registry manually
cat > ~/.flash/registry.json << 'EOF'
{
  "plugins": {
    "core": {
      "name": "core",
      "version": "local",
      "description": "Complete ORM features",
      "commands": ["init", "migrate", "apply", "status", "pull", "reset", "raw", "branch", "gen", "export"],
      "install_date": "2025-11-20T00:00:00Z",
      "size": 31457280
    }
  },
  "updated": "2025-11-20T00:00:00Z"
}
EOF

# Test
./flash plugins
./flash migrate --help
```

## FAQ

**Q: Can I use the old monolithic binary?**  
A: Yes, older releases remain available. However, new features will only be added to the plugin system (v2.1+).

**Q: How do I update a plugin?**  
A: Simply reinstall it: `flash add-plug core`. It will detect and update automatically.

**Q: Can I install multiple versions of a plugin?**  
A: No, only one version of each plugin can be installed at a time.

**Q: Where are plugins downloaded from?**  
A: Plugins are downloaded from GitHub releases at `https://github.com/Lumos-Labs-HQ/flash/releases`.

**Q: Do I need both core and studio?**  
A: No. Install `all` for everything, or just the one you need. If you have `all`, you don't need individual plugins.

**Q: What happens if I have both 'core' and 'all' installed?**  
A: The system will use the specific plugin (`core`) first, then fall back to `all`. It's redundant but harmless.

**Q: Can I create custom plugins?**  
A: Yes, you can build custom plugins locally and place them in `~/.flash/plugins/`. Update the registry manually.

**Q: Are plugins signed?**  
A: Currently plugins are not signed. We verify downloads from official GitHub releases.

**Q: How much disk space do plugins use?**  
A: Core: ~30MB, Studio: ~29MB, All: ~30MB. Base CLI: ~5-10MB.

## Support

For issues or questions:
- **GitHub Issues:** https://github.com/Lumos-Labs-HQ/flash/issues
- **Documentation:** https://flash-orm.dev
- **Discord:** https://discord.gg/flash-orm

## Version Compatibility

Plugins should match the base CLI version:

| Base CLI | Core Plugin | Studio Plugin | All Plugin |
|----------|-------------|---------------|------------|
| v2.1.11  | v2.1.11     | v2.1.11       | v2.1.11    |
| v2.2.0   | v2.2.0      | v2.2.0        | v2.2.0     |

Always install matching versions for best compatibility.
