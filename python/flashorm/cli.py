#!/usr/bin/env python3
import os
import sys
import subprocess
import platform

def main():
    binary_name = 'flash.exe' if platform.system().lower() == 'windows' else 'flash'
    bin_path = os.path.join(os.path.dirname(__file__), 'bin', binary_name)
    
    if not os.path.exists(bin_path):
        print(f"❌ flash binary not found at {bin_path}")
        print("Please reinstall: pip install --force-reinstall flashorm")
        sys.exit(1)
    
    try:
        result = subprocess.run([bin_path] + sys.argv[1:])
        sys.exit(result.returncode)
    except Exception as e:
        print(f"❌ Error running flash: {e}")
        sys.exit(1)

if __name__ == '__main__':
    main()
