# Release Notes - v1.6.0

## 🎉 Major Features

### 📤 **New Export System**
- **Multiple Export Formats**: JSON, CSV, and SQLite exports
- **Smart Data Export**: Automatically excludes migration tables
- **Timestamped Files**: All exports include timestamps for organization
- **Metadata Preservation**: JSON exports include version and timestamp metadata

```bash
# Export as JSON (default)
graft export

# Export as CSV (individual files per table)
graft export --csv

# Export as SQLite (portable database file)
graft export --sqlite
```

### 🔒 **Safe Migration System**
- **Transaction-Based Execution**: Each migration runs in its own transaction
- **Automatic Rollback**: Failed migrations automatically roll back without corrupting database
- **Enhanced Error Messages**: Clear error reporting with recovery instructions
- **Migration State Tracking**: Improved migration tracking and conflict detection

```bash
# Safe migration application with automatic rollback on failure
graft apply
```

### 🔧 **Enhanced Database Support**
- **PostgreSQL Optimization**: Improved connection pooling for Supabase/PgBouncer compatibility
- **Connection Pool Optimization**: Better resource management and performance
- **Exec Mode Support**: Enhanced compatibility with connection poolers

## 🚀 **Improvements**

### **CLI Enhancements**
- Updated `graft reset` command with export option before destructive operations
- Enhanced `graft status` with more detailed migration information
- Improved error messages across all commands
- Better user prompts and confirmations

### **Configuration Updates**
- Changed `backup_path` to `export_path` in configuration
- Enhanced project initialization templates
- Better default configuration values

### **Performance Optimizations**
- Optimized connection pooling for all database adapters
- Improved memory management for large exports
- Enhanced query performance with better prepared statement usage

## 🐛 **Bug Fixes**
- Fixed migration rollback issues in PostgreSQL adapter
- Resolved connection pool leaks
- Fixed schema parsing edge cases
- Improved error handling in export operations

## 📚 **Documentation**
- **Complete Documentation Overhaul**: All docs updated to reflect current features
- **New Export System Documentation**: Comprehensive guide for all export formats
- **Safe Migration Guide**: Detailed explanation of transaction safety
- **Updated Examples**: All examples reflect current v1.6.0 functionality

## 🔄 **Breaking Changes**
- Configuration field `backup_path` renamed to `export_path`
- Removed legacy backup commands (replaced with export system)
- Updated project structure to include `db/export/` directory

## 📦 **Installation**

### Download Binary
Download the appropriate binary for your platform from the release assets.

### Using Go Install
```bash
go install github.com/Rana718/Graft@v1.6.0
```

### From Source
```bash
git clone https://github.com/Rana718/Graft.git
cd Graft
git checkout v1.6.0
make build-all
```

## 🔍 **Migration Guide from v1.5.0**

### Update Configuration
```json
{
  "schema_path": "db/schema/schema.sql",
  "migrations_path": "db/migrations",
  "sqlc_config_path": "sqlc.yml",
  "export_path": "db/export",  // Changed from "backup_path"
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  }
}
```

### Update Commands
```bash
# Old backup command
graft backup "my backup"

# New export commands
graft export --json    # JSON format
graft export --csv     # CSV format  
graft export --sqlite  # SQLite format
```

## 🙏 **Acknowledgments**
- Thanks to all contributors who helped with testing and feedback
- Special thanks for PostgreSQL pooler compatibility improvements
- Community feedback on migration safety features

## 📋 **Full Changelog**
- Added comprehensive export system with multiple formats
- Implemented transaction-based safe migration execution
- Enhanced PostgreSQL adapter with pooler optimization
- Updated all documentation to reflect current features
- Improved error handling and user experience
- Fixed various bugs and performance issues

---

**Full Changelog**: https://github.com/Rana718/Graft/compare/v1.5.0...v1.6.0
