# FlashORM Release Notes

## Version 2.2.2 - Latest Release

### 🐛 Bug Fixes

#### .env File Preservation
- Fixed issue where `flash init` would overwrite existing `.env` files
- Now preserves all existing environment variables
- If `.env` exists with `DATABASE_URL`, file is left unchanged
- If `.env` exists without `DATABASE_URL`, appends it with a comment
- Only creates new `.env` if no file exists

### 📦 Installation

**NPM (Node.js/TypeScript)**
```bash
npm install -g flashorm
```

**Python**
```bash
pip install flashorm
```

**Go**
```bash
go install github.com/Lumos-Labs-HQ/flash@latest
```

---

For detailed documentation, see:
- [Usage Guide - Go](docs/USAGE_GO.md)
- [Usage Guide - TypeScript](docs/USAGE_TYPESCRIPT.md)
- [Usage Guide - Python](docs/USAGE_PYTHON.md)
- [Contributing](docs/CONTRIBUTING.md)
