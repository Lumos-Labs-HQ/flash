#!/usr/bin/env node

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

const packageName = `graft-${mappedPlatform}-${mappedArch}`;

console.log(`✅ Graft installed successfully for ${platform}-${arch}`);
console.log(`   Using package: ${packageName}`);
console.log(`\n🚀 Run 'graft --help' to get started!`);

