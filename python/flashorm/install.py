#!/usr/bin/env python3
import os
import sys
import platform
import urllib.request
import stat

VERSION = '2.1.11'
REPO = 'Lumos-Labs-HQ/flash'

def cleanup_binaries(bin_dir, keep_binary):
    """Remove binaries for other platforms"""
    if not os.path.exists(bin_dir):
        return
    
    for filename in os.listdir(bin_dir):
        filepath = os.path.join(bin_dir, filename)
        if filename.startswith('flash') and filename != 'flash' and filename != 'flash.exe' and filename != keep_binary:
            try:
                os.remove(filepath)
                print(f"🧹 Cleaned up: {filename}")
            except:
                pass

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
    
    print(f"📦 Installing FlashORM Base CLI v{VERSION} for {system}-{machine}...")
    print(f"📥 Downloading from: {download_url}")
    
    if not os.path.exists(bin_dir):
        os.makedirs(bin_dir, exist_ok=True)
    
    try:
        urllib.request.urlretrieve(download_url, binary_path)
        os.chmod(binary_path, os.stat(binary_path).st_mode | stat.S_IEXEC)
        
        # Clean up binaries for other platforms
        cleanup_binaries(bin_dir, binary_name)
        
        print("✅ FlashORM Base CLI installed successfully!")
        print("")
        print("📦 Plugin System")
        print("   FlashORM now uses a plugin-based architecture.")
        print("   The base CLI includes only essential commands:")
        print("     • flash --version    (show version)")
        print("     • flash plugins      (list plugins)")
        print("     • flash add-plug     (install plugins)")
        print("     • flash rm-plug      (remove plugins)")
        print("")
        print("   Install plugins for ORM functionality:")
        print("")
        print("   flash add-plug core    # ORM features (migrations, codegen, export)")
        print("   flash add-plug studio  # Visual database editor")
        print("   flash add-plug all     # Everything (core + studio)")
        print("")
        print("🚀 Run 'flash --help' to get started!")
    except Exception as err:
        if os.path.exists(binary_path):
            os.unlink(binary_path)
        print(f"❌ Download failed: {err}", file=sys.stderr)
        print(f"Please check: {download_url}", file=sys.stderr)
        print("")
        print("You can also download manually from:", file=sys.stderr)
        print(f"  https://github.com/{REPO}/releases/tag/v{VERSION}", file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__':
    install()

