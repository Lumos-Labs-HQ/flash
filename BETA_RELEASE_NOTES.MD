## FlashORM Beta Release v2.3.1-beta-dev

> **This is a BETA release** - Use for testing purposes only!
> This release may contain experimental features, bugs, or breaking changes.

### What's New

#### Plugin System (New Architecture)
- Modular plugin system: `core`, `studio`, `all`
- Install only what you need: `flash add-plug core`
- Automatic fallback to `all` plugin if installed
- Version management: latest, beta, or specific (`flash add-plug core@1.0.0`)
- Online plugin browser: `flash plugins --online`

#### SQL Studio - Export/Import
- Export database as Schema Only, Data Only, or Complete (schema + data)
- Import previously exported JSON data back into the database
- Available through the SQL Studio web interface

#### Dependency Cleanup
- Removed Fiber framework -> replaced with `net/http` 
- Removed Viper -> replaced with stdlib `encoding/json`
- Removed lib/pq -> `pgx/v5` is the only PostgreSQL driver
- Removed mapstructure -> only `json` struct tags
- ~8 fewer dependencies, smaller binary size

#### Bug Fixes
- Fixed `down` command missing from plugin registry
- Fixed CSS duplication across studio static assets

### Installation

```bash
# NPM
npm install -g flashorm@beta

# Python
pip install flashorm==2.3.1b1

# Go
go install github.com/Lumos-Labs-HQ/flash@v2.3.1-beta-dev

# Direct binary download from release assets, then:
flash add-plug all
```

### Installing Plugins

```bash
# Install individual plugins
flash add-plug core       # ORM + seeding
flash add-plug studio     # Visual database editor

# Or install everything
flash add-plug all

# Verify plugins
flash plugins
```

### What's Included

- **Base CLI binaries**: `flash-{platform}-{arch}`
- **Core Plugin binaries**: `flash-plugin-core-{platform}-{arch}` (includes seed)
- **Studio Plugin binaries**: `flash-plugin-studio-{platform}-{arch}`
- **All Plugin binaries**: `flash-plugin-all-{platform}-{arch}`

### Supported Platforms

- Linux (amd64, arm64)
- macOS (amd64, arm64 - Apple Silicon)
- Windows (amd64, arm64)
