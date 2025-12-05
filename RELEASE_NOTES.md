# FlashORM Release Notes

## Version 2.2.1 - Latest Release

### 🚀 New Features

#### Schema Folder Support
- Support for organizing schemas in a dedicated folder (`schema_dir` config option)
- Automatically discovers and merges multiple `.sql` files from the schema directory
- Files are processed alphabetically for consistent ordering
- Each table can have its own file for better organization

#### Smart Pull Command
- Enhanced `flash pull` with intelligent file management
- **Auto-comment feature**: When tables are dropped from the database, their schema files are automatically commented out (not deleted)
- Preserves your schema history while reflecting actual database state
- Easy to restore tables by uncommenting the files

#### Index Support
- Full parsing and generation of database indexes
- Supports unique indexes and multi-column indexes
- Preserved during pull and migration operations

#### Down Migrations
- Support for `-- +migrate Down` markers in migration files
- Rollback capability with `flash down` command
- Automatic rollback on migration failures

### 💡 Improvements

#### CLI Output
- Improved `flash status` with separate ID and NAME columns
- Better aligned table output for `flash plugins --online`
- Version and commit information for plugins

#### Code Generation
- Enhanced Go, TypeScript, and Python generators
- Better handling of nullable fields
- Improved type mappings

#### Studio Enhancements
- MongoDB Studio with Compass-like interface
- Redis Studio with real CLI terminal
- Visual schema designer with relationship visualization
- Auto-migration creation from schema changes

### 🔧 Configuration

```json
{
  "version": "2",
  "schema_dir": "db/schema",
  "queries": "db/queries/",
  "migrations_path": "db/migrations",
  "export_path": "db/export",
  "database": {
    "provider": "sqlite",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "go": { "enabled": true }
  }
}
```

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


For detailed documentation, see:
- [Usage Guide - Go](docs/USAGE_GO.md)
- [Usage Guide - TypeScript](docs/USAGE_TYPESCRIPT.md)
- [Usage Guide - Python](docs/USAGE_PYTHON.md)
- [Contributing](docs/CONTRIBUTING.md)
