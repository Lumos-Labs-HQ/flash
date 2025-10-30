# Publishing to NPM

## Setup (One-time)

1. **Create NPM account**: https://www.npmjs.com/signup

2. **Generate NPM token**:
   ```bash
   npm login
   npm token create
   ```

3. **Add token to GitHub Secrets**:
   - Go to: `https://github.com/Rana718/Graft/settings/secrets/actions`
   - Add secret: `NPM_TOKEN` = your token

## Publishing

### Automatic (via GitHub Actions)

1. **Update version** in `npm/package.json`

2. **Create and push tag**:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

3. GitHub Actions will automatically:
   - Build binaries for all platforms
   - Publish to npm

### Manual Publishing

```bash
# Build binaries
mkdir -p npm/bin
GOOS=linux GOARCH=amd64 go build -o npm/bin/graft-linux-x64 .
GOOS=linux GOARCH=arm64 go build -o npm/bin/graft-linux-arm64 .
GOOS=darwin GOARCH=amd64 go build -o npm/bin/graft-darwin-x64 .
GOOS=darwin GOARCH=arm64 go build -o npm/bin/graft-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o npm/bin/graft-win32-x64.exe .

# Publish
cd npm
npm publish
```

## Testing Before Publishing

```bash
# Test locally
cd npm
npm pack
npm install -g graft-1.0.0.tgz
graft --help
```

## Version Bumping

```bash
cd npm
npm version patch  # 1.0.0 -> 1.0.1
npm version minor  # 1.0.0 -> 1.1.0
npm version major  # 1.0.0 -> 2.0.0
```
