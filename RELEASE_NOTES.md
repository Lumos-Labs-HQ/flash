# FlashORM Release Notes

## Version 2.2.21 - Latest Release

### 🐛 Bug Fixes

#### Go Code Generator
- Fixed unnecessary imports in generated `models.go`
- `database/sql` is now only imported when nullable types are used
- `time` package is now only imported when timestamp/date fields exist

#### JavaScript Code Generator
- Removed redundant `.d.ts` files (`users.d.ts`, `database.d.ts`)
- Now only generates `index.d.ts` for TypeScript type definitions

#### Schema Parser
- Fixed folder-based schema parsing to properly use `schema_dir` config
- Query validator now works correctly with split schema files

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
