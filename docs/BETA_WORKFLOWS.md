# Release Workflow System

## Workflow Architecture

### Reusable Workflows

#### `build-binaries.yml` (Reusable)
Core workflow that builds all binaries (base CLI + plugins) for all platforms.
- **Used by:** Both stable and beta release workflows
- **Outputs:** All compiled binaries as artifacts
- **Platforms:** Linux, macOS, Windows (amd64 & arm64)

### Release Workflows

#### `release.yml` - Stable Release
Triggered by stable version tags (e.g., `v1.0.0`, `v2.1.0`)

**Trigger:** `v*` tags **EXCEPT** tags containing "beta"

**What it does:**
1. Calls `build-binaries.yml` to compile all binaries
2. Creates a GitHub Release (marked as stable)
3. Uploads all binaries and checksums
4. Triggers NPM and PyPI stable releases

**Tag examples:**
- ✅ `v1.0.0` - Triggers stable release
- ✅ `v2.1.3` - Triggers stable release
- ❌ `v1.0.0-beta` - Does NOT trigger (beta workflow handles this)
- ❌ `v2.0.0beta1` - Does NOT trigger (beta workflow handles this)

#### `beta-release.yml` - Beta Release
Triggered by beta version tags (e.g., `v1.0.0-beta`, `v2.0.0-beta1`)

**Trigger:** Tags matching `v*-beta*` or `v*beta*`

**What it does:**
1. Calls `build-binaries.yml` to compile all binaries
2. Creates a GitHub Release (marked as **prerelease**)
3. Uploads all binaries and checksums
4. Publishes to NPM with `@beta` tag
5. Publishes to PyPI with beta version

**Tag examples:**
- ✅ `v1.0.0-beta` - Triggers beta release
- ✅ `v2.0.0-beta1` - Triggers beta release
- ✅ `v1.5.0beta` - Triggers beta release
- ❌ `v1.0.0` - Does NOT trigger (stable workflow handles this)

### Publication Workflows

#### `npmrelease.yml` - NPM Stable Release
- **Trigger:** After successful stable release
- **Check:** Skips if tag contains "beta"
- **Action:** Publishes to npm with `latest` tag

#### `pypi-release.yml` - PyPI Stable Release
- **Trigger:** After successful stable release
- **Check:** Skips if tag contains "beta"
- **Action:** Publishes to PyPI

## Release Process

### Creating a Stable Release

1. **Update version** in code (if needed)
2. **Create and push tag:**
   ```bash
   git tag v1.2.0
   git push origin v1.2.0
   ```
3. **Automated steps:**
   - Builds all binaries
   - Creates GitHub release (stable)
   - Publishes to npm (latest tag)
   - Publishes to PyPI

### Creating a Beta Release

1. **Update version** in code (if needed)
2. **Create and push beta tag:**
   ```bash
   git tag v1.2.0-beta1
   git push origin v1.2.0-beta1
   ```
3. **Automated steps:**
   - Builds all binaries
   - Creates GitHub release (marked as prerelease)
   - Publishes to npm with `@beta` tag
   - Publishes to PyPI with beta version

**Install beta from npm:**
```bash
npm install -g flashorm@beta
```

**Install beta from PyPI:**
```bash
pip install flashorm==1.2.0b1
```

## Tag Naming Conventions

### Stable Releases
- `v1.0.0` - Major release
- `v1.1.0` - Minor release
- `v1.1.1` - Patch release

### Beta Releases
- `v1.0.0-beta` - Beta without number
- `v1.0.0-beta1` - First beta
- `v1.0.0-beta2` - Second beta
- `v2.0.0beta` - Alternative format (also works)

## Workflow Separation Logic

The system ensures beta and stable releases don't interfere:

1. **Tag pattern matching:**
   - Stable: `v*` excluding beta patterns
   - Beta: `v*-beta*` and `v*beta*`

2. **Runtime checks:**
   - Release workflow exits if beta tag detected
   - NPM/PyPI workflows skip if beta tag detected

3. **Publication tagging:**
   - Stable: Published as `latest` on npm
   - Beta: Published as `beta` on npm

## Build Artifacts

All workflows produce the same binaries:

### Base CLI (5-10MB)
- `flash-linux-amd64`
- `flash-linux-arm64`
- `flash-windows-amd64.exe`
- `flash-windows-arm64.exe`
- `flash-darwin-amd64`
- `flash-darwin-arm64`

### Core Plugin (~25MB)
- `flash-plugin-core-{platform}-{arch}`

### Studio Plugin (~18MB)
- `flash-plugin-studio-{platform}-{arch}`

### All Plugin (~40MB)
- `flash-plugin-all-{platform}-{arch}`

### Checksums
- `checksums.txt` - SHA256 hashes for all files

## Testing the Workflows

### Local Testing
Build locally using the Makefile:
```bash
make build-all          # Build base CLI
make build-plugins      # Build all plugins
```

### Testing Beta Release
1. Create a beta tag locally: `git tag v0.0.1-beta-test`
2. Push to a test branch/fork
3. Verify only beta workflow triggers
4. Check GitHub release is marked as "prerelease"
5. Delete test tag: `git push --delete origin v0.0.1-beta-test`

### Testing Stable Release
1. Create a stable tag locally: `git tag v0.0.1-test`
2. Push to a test branch/fork
3. Verify only stable workflow triggers
4. Check GitHub release is NOT marked as "prerelease"
5. Delete test tag: `git push --delete origin v0.0.1-test`

## Troubleshooting

### Beta workflow triggered for stable tag
- Check tag doesn't contain "beta" anywhere
- Verify tag matches pattern `v*` exactly

### Stable workflow triggered for beta tag
- Ensure tag contains "beta" (e.g., `v1.0.0-beta`)
- Check tag matches beta patterns

### NPM/PyPI published for beta release
- Verify beta checks in npmrelease.yml and pypi-release.yml
- Beta should publish with `@beta` tag on npm

### Both workflows triggered
- Only one workflow should match any given tag
- Review tag name carefully
- Check workflow trigger patterns

## Maintenance

### Adding New Platforms
Edit `build-binaries.yml` to add build targets.

### Changing Build Flags
Modify the `-ldflags` in `build-binaries.yml`.

### Updating Dependencies
Change Go version in `build-binaries.yml`:
```yaml
go-version: '1.24.2'  # Update this
```

## Security Notes

Required secrets:
- `NPM_TOKEN` - For npm publication
- `PYPI_API_TOKEN` - For PyPI publication
- `GITHUB_TOKEN` - Automatically provided

These should be configured in repository settings.
