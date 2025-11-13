#!/usr/bin/env python3
import os
import sys
import platform
import urllib.request
import stat

VERSION = '2.0.8'
REPO = 'Lumos-Labs-HQ/flash'

def install():
    system = platform.system().lower()
    machine = platform.machine().lower()
    
    platform_map = {'darwin': 'darwin', 'linux': 'linux', 'windows': 'windows'}
    arch_map = {'x86_64': 'amd64', 'amd64': 'amd64', 'arm64': 'arm64', 'aarch64': 'arm64'}
    
    mapped_platform = platform_map.get(system)
    mapped_arch = arch_map.get(machine)
    
    if not mapped_platform or not mapped_arch:
        print(f"❌ Unsupported platform: {system}-{machine}", file=sys.stderr)
        sys.exit(1)
    
    binary_name = 'flash.exe' if system == 'windows' else 'flash'
    download_name = f"flash-{mapped_platform}-{mapped_arch}{'.exe' if system == 'windows' else ''}"
    download_url = f"https://github.com/{REPO}/releases/download/v{VERSION}/{download_name}"
    
    bin_dir = os.path.join(os.path.dirname(__file__), 'bin')
    binary_path = os.path.join(bin_dir, binary_name)
    
    print(f"📦 Installing flash v{VERSION} for {system}-{machine}...")
    print(f"📥 Downloading from: {download_url}")
    
    if not os.path.exists(bin_dir):
        os.makedirs(bin_dir, exist_ok=True)
    
    try:
        urllib.request.urlretrieve(download_url, binary_path)
        os.chmod(binary_path, os.stat(binary_path).st_mode | stat.S_IEXEC)
        print("✅ flash installed successfully!")
        print("🚀 Run 'flash --help' to get started!")
    except Exception as err:
        if os.path.exists(binary_path):
            os.unlink(binary_path)
        print(f"❌ Download failed: {err}", file=sys.stderr)
        print(f"Please check: {download_url}", file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__':
    install()
