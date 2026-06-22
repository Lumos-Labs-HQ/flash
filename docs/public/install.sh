#!/usr/bin/env bash
set -euo pipefail

REPO="Lumos-Labs-HQ/flash"
INSTALL_DIR="${FLASH_INSTALL:-$HOME/.flash/bin}"
BINARY_NAME="flash"

# ---- Platform detection ----
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux)  os="linux" ;;
        Darwin) os="darwin" ;;
        *)      echo "Unsupported OS: $(uname -s)"; exit 1 ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) echo "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac

    echo "${os}-${arch}"
}

# ---- Get latest release version from GitHub ----
get_latest_version() {
    if command -v curl &>/dev/null; then
        curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
            | grep '"tag_name":' \
            | sed 's/.*"v\(.*\)".*/\1/'
    elif command -v wget &>/dev/null; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" \
            | grep '"tag_name":' \
            | sed 's/.*"v\(.*\)".*/\1/'
    else
        echo "Error: curl or wget required" >&2
        exit 1
    fi
}

# ---- Main install ----
main() {
    echo "Installing Flash ORM CLI..."
    echo

    local platform arch version download_url
    platform=$(detect_platform)
    os="${platform%-*}"
    arch="${platform#*-}"
    version=$(get_latest_version)

    # Windows uses .exe suffix
    local binary="${BINARY_NAME}"
    if [ "${os}" = "windows" ]; then
        binary="${binary}.exe"
    fi

    download_url="https://github.com/${REPO}/releases/download/v${version}/flash-${os}-${arch}"

    echo "  Platform: ${os}/${arch}"
    echo "  Version:  v${version}"
    echo "  URL:      ${download_url}"
    echo

    # Create install directory
    mkdir -p "${INSTALL_DIR}"

    # Download binary
    echo "Downloading Flash ORM CLI..."
    if command -v curl &>/dev/null; then
        curl -fsSL "${download_url}" -o "${INSTALL_DIR}/${binary}"
    elif command -v wget &>/dev/null; then
        wget -qO "${INSTALL_DIR}/${binary}" "${download_url}"
    fi

    chmod +x "${INSTALL_DIR}/${binary}"

    echo
    echo "Flash ORM CLI v${version} installed to: ${INSTALL_DIR}/${binary}"
    echo

    # Check if install dir is in PATH
    case ":${PATH}:" in
        *:"${INSTALL_DIR}":*)
            echo "✓ ${INSTALL_DIR} is already in your PATH"
            echo
            echo "Run 'flash --version' to verify."
            ;;
        *)
            echo "⚠️  ${INSTALL_DIR} is NOT in your PATH."
            echo
            echo "Add it to your shell profile:"
            echo
            echo "  bash/zsh:  echo 'export PATH=\"\$PATH:${INSTALL_DIR}\"' >> ~/.bashrc && source ~/.bashrc"
            echo "  zsh:       echo 'export PATH=\"\$PATH:${INSTALL_DIR}\"' >> ~/.zshrc && source ~/.zshrc"
            echo "  fish:      fish_add_path ${INSTALL_DIR}"
            echo
            echo "Then run 'flash --version' to verify."
            echo
            echo "Or use the binary directly:"
            echo "  ${INSTALL_DIR}/flash --version"
            ;;
    esac

    echo
    echo "Next steps:"
    echo "  1. Run 'flash init' to create a new project"
    echo "  2. See the docs: https://lumos-labs-hq.github.io/flash/"
}

main
