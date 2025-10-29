const { execSync } = require('child_process');
const path = require('path');

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
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }
  
  const binaryName = platform === 'win32' ? 'graft.exe' : 'graft';
  const packageName = `graft-orm-${mappedPlatform}-${mappedArch}`;
  
  return require.resolve(`${packageName}/bin/${binaryName}`);
}

function exec(command, options = {}) {
  const binaryPath = getBinaryPath();
  return execSync(`"${binaryPath}" ${command}`, {
    encoding: 'utf8',
    ...options
  });
}

module.exports = {
  getBinaryPath,
  exec
};
