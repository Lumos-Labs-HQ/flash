#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

const platform = process.platform;
const binaryName = platform === 'win32' ? 'graft.exe' : 'graft';
const binaryPath = path.join(__dirname, binaryName);

if (!fs.existsSync(binaryPath)) {
  console.error('❌ Graft binary not found. Please reinstall: npm install -g graft-orm');
  process.exit(1);
}

const child = spawn(binaryPath, process.argv.slice(2), {
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
