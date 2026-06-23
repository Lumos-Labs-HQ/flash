#!/usr/bin/env pwsh
$REPO = "Lumos-Labs-HQ/flash"
$INSTALL_DIR = if ($env:FLASH_INSTALL) { $env:FLASH_INSTALL } else { "$HOME\.flash\bin" }
$BINARY_NAME = "flash.exe"

# ---- Platform detection ----
function Get-Platform {
    $os = "windows"
    $arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture) {
        "X64"  { "amd64" }
        "Arm64" { "arm64" }
        default { Write-Error "Unsupported architecture"; exit 1 }
    }
    return "$os-$arch"
}

# ---- Get latest release version from GitHub ----
function Get-LatestVersion {
    $apiUrl = "https://api.github.com/repos/$REPO/releases/latest"
    try {
        $response = Invoke-RestMethod -Uri $apiUrl -UseBasicParsing
        return $response.tag_name -replace "^v", ""
    } catch {
        Write-Error "Failed to fetch latest version: $_"
        exit 1
    }
}

# ---- Main install ----
function Main {
    Write-Host "Installing Flash ORM CLI..." -ForegroundColor Cyan
    Write-Host ""

    $version = Get-LatestVersion
    $downloadUrl = "https://github.com/$REPO/releases/download/v$version/flash-windows-$arch"

    Write-Host "  Platform: windows/$arch"
    Write-Host "  Version:  v$version"
    Write-Host ""

    # Create install directory
    New-Item -ItemType Directory -Force -Path $INSTALL_DIR | Out-Null

    # Download binary
    Write-Host "Downloading Flash ORM CLI..." -ForegroundColor Cyan
    $outPath = Join-Path $INSTALL_DIR $BINARY_NAME
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $outPath -UseBasicParsing
    } catch {
        Write-Error "Failed to download: $_"
        exit 1
    }

    Write-Host ""
    Write-Host "Flash ORM CLI v$version installed to: $outPath" -ForegroundColor Green
    Write-Host ""

    # Check if install dir is in PATH
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -and $userPath.Contains($INSTALL_DIR)) {
        Write-Host "✓ $INSTALL_DIR is already in your PATH" -ForegroundColor Green
        Write-Host ""
        Write-Host "Run 'flash --version' to verify." -ForegroundColor Cyan
    } else {
        Write-Host "⚠️  $INSTALL_DIR is NOT in your PATH." -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Add it to your PATH:"
        Write-Host ""
        $line = "[Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path', 'User') + ';$INSTALL_DIR', 'User')"
        Write-Host "  powershell -c `"$line`""
        Write-Host ""
        Write-Host "Then restart your terminal and run 'flash --version'."
        Write-Host ""
        Write-Host "Or use the binary directly:"
        Write-Host "  $outPath --version"
    }

    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Cyan
    Write-Host "  1. Run 'flash init' to create a new project"
    Write-Host "  2. See the docs: https://lumos-labs-hq.github.io/flash/"
}

Main
