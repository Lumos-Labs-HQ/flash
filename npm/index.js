const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');

function getBinaryPath() {
  const platform = process.platform;
  const binaryName = platform === 'win32' ? 'graft.exe' : 'graft';
  const binaryPath = path.join(__dirname, 'bin', binaryName);
  
  if (!fs.existsSync(binaryPath)) {
    throw new Error('Graft binary not found. Please reinstall: npm install -g graft-orm');
  }
  
  return binaryPath;
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
