#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');

const VERSION = '2.1.11';
const REPO = 'Lumos-Labs-HQ/flash';

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

const binaryName = platform === 'win32' ? 'flash.exe' : 'flash';
const downloadName = `flash-${mappedPlatform}-${mappedArch}${platform === 'win32' ? '.exe' : ''}`;
const downloadUrl = `https://github.com/${REPO}/releases/download/v${VERSION}/${downloadName}`;

const binDir = path.join(__dirname, '..', 'bin');
const binaryPath = path.join(binDir, binaryName);

console.log(`📦 Installing FlashORM Base CLI v${VERSION} for ${platform}-${arch}...`);
console.log(`📥 Downloading from: ${downloadUrl}`);

if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

// Clean up any existing binaries (other platforms)
const cleanupBinaries = () => {
  const files = fs.readdirSync(binDir);
  files.forEach(file => {
    const filePath = path.join(binDir, file);
    if (file.startsWith('flash') && file !== 'flash.js' && file !== binaryName) {
      try {
        fs.unlinkSync(filePath);
        console.log(`🧹 Cleaned up: ${file}`);
      } catch (err) {
        // Ignore cleanup errors
      }
    }
  });
};

const file = fs.createWriteStream(binaryPath);

https.get(downloadUrl, (response) => {
  if (response.statusCode === 302 || response.statusCode === 301) {
    https.get(response.headers.location, (redirectResponse) => {
      redirectResponse.pipe(file);
      file.on('finish', () => {
        file.close(() => {
          fs.chmodSync(binaryPath, 0o755);
          cleanupBinaries();
          console.log(`✅ FlashORM Base CLI installed successfully!`);
          console.log('');
          console.log('📦 Plugin System');
          console.log('   FlashORM now uses a plugin-based architecture.');
          console.log('   The base CLI includes only essential commands:');
          console.log('     • flash --version    (show version)');
          console.log('     • flash plugins      (list plugins)');
          console.log('     • flash add-plug     (install plugins)');
          console.log('     • flash rm-plug      (remove plugins)');
          console.log('');
          console.log('   Install plugins for ORM functionality:');
          console.log('');
          console.log('   flash add-plug core    # ORM features (migrations, codegen, export)');
          console.log('   flash add-plug studio  # Visual database editor');
          console.log('   flash add-plug all     # Everything (core + studio)');
          console.log('');
          console.log(`🚀 Run 'flash --help' to get started!`);
        });
      });
    });
  } else {
    response.pipe(file);
    file.on('finish', () => {
      file.close(() => {
        fs.chmodSync(binaryPath, 0o755);
        cleanupBinaries();
        console.log(`✅ FlashORM Base CLI installed successfully!`);
        console.log('');
        console.log('📦 Plugin System');
        console.log('   FlashORM now uses a plugin-based architecture.');
        console.log('   The base CLI includes only essential commands:');
        console.log('     • flash --version    (show version)');
        console.log('     • flash plugins      (list plugins)');
        console.log('     • flash add-plug     (install plugins)');
        console.log('     • flash rm-plug      (remove plugins)');
        console.log('');
        console.log('   Install plugins for ORM functionality:');
        console.log('');
        console.log('   flash add-plug core    # ORM features (migrations, codegen, export)');
        console.log('   flash add-plug studio  # Visual database editor');
        console.log('   flash add-plug all     # Everything (core + studio)');
        console.log('');
        console.log(`🚀 Run 'flash --help' to get started!`);
      });
    });
  }
}).on('error', (err) => {
  if (fs.existsSync(binaryPath)) {
    fs.unlinkSync(binaryPath);
  }
  console.error(`❌ Download failed: ${err.message}`);
  console.error(`Please check: ${downloadUrl}`);
  console.error('');
  console.error('You can also download manually from:');
  console.error(`  https://github.com/${REPO}/releases/tag/v${VERSION}`);
  process.exit(1);
});
