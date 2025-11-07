#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');

const VERSION = '1.6.0';
const REPO = 'Rana718/Graft';

const platform = process.platform;
const arch = process.arch;

const platformMap = {
  'darwin': 'darwin',
  'linux': 'linux',
  'win32': 'windows'
};

const archMap = {
  'x64': 'amd64',
  'arm64': 'arm64'
};

const mappedPlatform = platformMap[platform];
const mappedArch = archMap[arch];

if (!mappedPlatform || !mappedArch) {
  console.error(`❌ Unsupported platform: ${platform}-${arch}`);
  process.exit(1);
}

const binaryName = platform === 'win32' ? 'graft.exe' : 'graft';
const downloadName = `graft-${mappedPlatform}-${mappedArch}${platform === 'win32' ? '.exe' : ''}`;
const downloadUrl = `https://github.com/${REPO}/releases/download/v${VERSION}/${downloadName}`;

const binDir = path.join(__dirname, '..', 'bin');
const binaryPath = path.join(binDir, binaryName);

console.log(`📦 Installing Graft v${VERSION} for ${platform}-${arch}...`);
console.log(`📥 Downloading from: ${downloadUrl}`);

if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

const file = fs.createWriteStream(binaryPath);

https.get(downloadUrl, (response) => {
  if (response.statusCode === 302 || response.statusCode === 301) {
    https.get(response.headers.location, (redirectResponse) => {
      redirectResponse.pipe(file);
      file.on('finish', () => {
        file.close(() => {
          fs.chmodSync(binaryPath, 0o755);
          console.log(`✅ Graft installed successfully!`);
          console.log(`🚀 Run 'graft --help' to get started!`);
        });
      });
    });
  } else {
    response.pipe(file);
    file.on('finish', () => {
      file.close(() => {
        fs.chmodSync(binaryPath, 0o755);
        console.log(`✅ Graft installed successfully!`);
        console.log(`🚀 Run 'graft --help' to get started!`);
      });
    });
  }
}).on('error', (err) => {
  fs.unlinkSync(binaryPath);
  console.error(`❌ Download failed: ${err.message}`);
  console.error(`Please check: ${downloadUrl}`);
  process.exit(1);
});
