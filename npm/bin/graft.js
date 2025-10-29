#!/usr/bin/env node

const { spawn } = require('child_process');
const fs = require('fs');

function getBinaryPath() {
  const platform = process.platform;
  const arch = process.arch;
  
  const platformMap = {
    'darwin': 'darwin',
    'linux': 'linux',
    'win32': 'win32'
  };
  
  const archMap = {
    'x64': 'x64',
    'arm64': 'arm64'
  };
  
  const mappedPlatform = platformMap[platform];
  const mappedArch = archMap[arch];
  
  if (!mappedPlatform || !mappedArch) {
    console.error(`❌ Unsupported platform: ${platform}-${arch}`);
    process.exit(1);
  }
  
  const binaryName = platform === 'win32' ? 'graft.exe' : 'graft';
  const packageName = `graft-${mappedPlatform}-${mappedArch}`;
  
  try {
    const binaryPath = require.resolve(`${packageName}/bin/${binaryName}`);
    return binaryPath;
  } catch (e) {
    console.error(`❌ Failed to find Graft binary for ${platform}-${arch}`);
    console.error('   Please ensure the platform-specific package is installed.');
    process.exit(1);
  }
}

function main() {
  const binaryPath = getBinaryPath();
  
  if (!fs.existsSync(binaryPath)) {
    console.error(`❌ Binary not found at: ${binaryPath}`);
    process.exit(1);
  }
  
  const args = process.argv.slice(2);
  const child = spawn(binaryPath, args, {
    stdio: 'inherit',
    windowsHide: true
  });
  
  child.on('exit', (code) => {
    process.exit(code || 0);
  });
  
  child.on('error', (err) => {
    console.error('❌ Failed to start Graft:', err);
    process.exit(1);
  });
}

main();

