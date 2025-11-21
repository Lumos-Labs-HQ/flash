#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

const platform = process.platform;
const binaryName = platform === 'win32' ? 'flash.exe' : 'flash';
const binaryPath = path.join(__dirname, binaryName);

// If binary doesn't exist, try to download it
if (!fs.existsSync(binaryPath)) {
  console.log('📥 Binary not found. Downloading...');
  
  // Import and wait for download
  const downloadPromise = require('../scripts/download.js');
  
  downloadPromise
    .then(() => {
      if (!fs.existsSync(binaryPath)) {
        console.error('❌ Download completed but binary not found. Please try: npm install flashorm --force');
        process.exit(1);
      }
      executeBinary();
    })
    .catch((err) => {
      console.error('❌ Failed to download flash binary');
      console.error('Error:', err.message);
      console.error('');
      console.error('Please try: npm install flashorm --force');
      process.exit(1);
    });
} else {
  // Binary exists, execute it
  executeBinary();
}

function executeBinary() {
  const child = spawn(binaryPath, process.argv.slice(2), {
    stdio: 'inherit',
    windowsHide: true
  });

  child.on('exit', (code) => {
    process.exit(code || 0);
  });

  child.on('error', (err) => {
    console.error('❌ Failed to start flash:', err);
    process.exit(1);
  });
}
