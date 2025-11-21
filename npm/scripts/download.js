#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');

const VERSION = '2.2.0-beta2';
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

// Clean up binaries for other platforms
function cleanupOtherBinaries() {
    try {
        const files = fs.readdirSync(binDir);
        files.forEach(file => {
            const filePath = path.join(binDir, file);
            if (file.startsWith('flash') && file !== 'flash.js' && file !== binaryName) {
                try {
                    fs.unlinkSync(filePath);
                    console.log(`🧹 Removed unused binary: ${file}`);
                } catch (err) {
                    // Ignore cleanup errors
                }
            }
        });
    } catch (err) {
        // Ignore if bin dir doesn't exist
    }
}

// Skip if binary already exists for this platform
if (fs.existsSync(binaryPath)) {
    console.log(`✅ Binary already exists for ${platform}-${arch}`);
    cleanupOtherBinaries();
    process.exit(0);
}

console.log(`📦 Installing FlashORM v${VERSION} for ${platform}-${arch}...`);
console.log(`📥 Downloading from: ${downloadUrl}`);

if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
}

function downloadBinary(url) {
    return new Promise((resolve, reject) => {
        const file = fs.createWriteStream(binaryPath);

        https.get(url, (response) => {
            if (response.statusCode === 302 || response.statusCode === 301) {
                // Follow redirect
                file.close();
                fs.unlinkSync(binaryPath);
                downloadBinary(response.headers.location).then(resolve).catch(reject);
                return;
            }

            if (response.statusCode !== 200) {
                file.close();
                fs.unlinkSync(binaryPath);
                reject(new Error(`Download failed with status ${response.statusCode}`));
                return;
            }

            response.pipe(file);

            file.on('finish', () => {
                file.close(() => {
                    try {
                        fs.chmodSync(binaryPath, 0o755);
                        cleanupOtherBinaries();
                        resolve();
                    } catch (err) {
                        reject(err);
                    }
                });
            });

            file.on('error', (err) => {
                fs.unlink(binaryPath, () => { });
                reject(err);
            });
        }).on('error', (err) => {
            fs.unlink(binaryPath, () => { });
            reject(err);
        });
    });
}

// Main execution
if (require.main === module) {
    downloadBinary(downloadUrl)
        .then(() => {
            console.log(`✅ FlashORM installed successfully!`);
            console.log('');
            console.log('📦 Plugin System');
            console.log('   FlashORM uses a plugin-based architecture.');
            console.log('   The base CLI includes essential commands:');
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
        })
        .catch((err) => {
            console.error('❌ Download failed:', err.message);
            console.error('');
            console.error('Please try:');
            console.error('  1. Check your internet connection');
            console.error('  2. Verify the release exists on GitHub');
            console.error(`  3. Manual install: ${downloadUrl}`);
            process.exit(1);
        });
} else {
    // Being required as a module
    module.exports = downloadBinary(downloadUrl);
}
