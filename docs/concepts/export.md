---
title: Data Export
description: Export database data in multiple formats
---

# Data Export

Flash ORM provides powerful export capabilities to extract your database data in various formats for backup, migration, analysis, or sharing.

## Table of Contents

- [Export Formats](#export-formats)
- [Basic Export](#basic-export)
- [Advanced Export](#advanced-export)
- [Filtering & Querying](#filtering--querying)
- [Performance Considerations](#performance-considerations)
- [Use Cases](#use-cases)
- [Troubleshooting](#troubleshooting)

## Export Formats

### JSON Format

```bash
flash export --format json
```

Creates a structured JSON file:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0",
  "tables": {
    "users": [
      {
        "id": 1,
        "name": "John Doe",
        "email": "john@example.com",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "posts": [
      {
        "id": 1,
        "user_id": 1,
        "title": "My First Post",
        "content": "Hello world!",
        "created_at": "2024-01-02T00:00:00Z"
      }
    ]
  }
}
```

### CSV Format

```bash
flash export --format csv
```

Creates separate CSV files for each table:

```
users.csv
posts.csv
comments.csv
```

Each CSV includes headers and properly escaped data.

### SQLite Format

```bash
flash export --format sqlite
```

Creates a new SQLite database file with the same schema and data. Perfect for:

- Creating portable database copies
- Testing with SQLite locally
- Sharing data without requiring PostgreSQL/MySQL

## Basic Export

### Export All Data

```bash
# Export to default location (db/export/)
flash export

# Export to specific directory
flash export --output ./backups/

# Export with custom filename
flash export --output ./backup.json
```

### Export Specific Tables

```bash
# Export only users and posts tables
flash export --tables users,posts

# Export all except certain tables
flash export --exclude _flash_migrations,sessions
```

### Export Options

```bash
flash export \
  --format json \
  --output ./data/ \
  --tables users,posts \
  --compress  # Create .gz compressed files
```

## Advanced Export

### Schema-Only Export

```bash
# Export schema without data
flash export --schema-only

# Useful for creating empty database templates
flash export --schema-only --format sqlite --output template.db
```

### Data-Only Export

```bash
# Export data without schema
flash export --data-only

# Useful for seeding databases
flash export --data-only --format json --output seeds/
```

### Incremental Export

```bash
# Export only records modified after a date
flash export --since "2024-01-01"

# Export only records modified in the last 24 hours
flash export --since "24 hours ago"
```

### Parallel Export

For large databases, Flash ORM automatically uses parallel processing:

- **Concurrent Table Export**: Multiple tables exported simultaneously
- **Batch Processing**: Large tables processed in chunks
- **Memory Management**: Controlled memory usage for large exports

## Filtering & Querying

### Custom WHERE Clauses

```bash
# Export only active users
flash export --where "users.is_active = true"

# Export posts from specific user
flash export --where "posts.user_id = 123"

# Export recent data
flash export --where "created_at > '2024-01-01'"
```

### JOIN-Based Filtering

```bash
# Export users who have posts
flash export \
  --query "SELECT DISTINCT u.* FROM users u JOIN posts p ON u.id = p.user_id"

# Export with related data
flash export \
  --query "SELECT u.*, COUNT(p.id) as post_count FROM users u LEFT JOIN posts p ON u.id = p.user_id GROUP BY u.id"
```

### Limit & Offset

```bash
# Export first 1000 records from each table
flash export --limit 1000

# Export records 1000-2000 from each table
flash export --limit 1000 --offset 1000
```

## Performance Considerations

### Large Database Export

For databases with millions of records:

```bash
# Use parallel processing (automatic)
flash export --parallel 4

# Export in batches
flash export --batch-size 10000

# Compress output
flash export --compress
```

### Memory Optimization

```bash
# Control memory usage
flash export --memory-limit 512MB

# Use streaming for large exports
flash export --stream
```

### Index Usage

Flash ORM automatically uses appropriate indexes for filtering:

- **Date Filters**: Uses date indexes when available
- **ID Filters**: Uses primary key indexes
- **Foreign Key Filters**: Uses foreign key indexes

## Use Cases

### Database Backup

```bash
# Daily backup
flash export --format json --compress --output ./backups/daily/

# Schema backup
flash export --schema-only --format sql --output schema.sql
```

### Data Migration

```bash
# Export from production
flash export --format json --output migration_data.json

# Import to staging
flash import --file migration_data.json
```

### Development Seeding

```bash
# Export sample data
flash export --limit 100 --format json --output seeds/

# Create development fixtures
flash export --where "environment = 'development'" --output fixtures/
```

### Analytics & Reporting

```bash
# Export user behavior data
flash export \
  --tables users,sessions,events \
  --where "created_at >= '2024-01-01'" \
  --format csv \
  --output analytics/

# Export for BI tools
flash export --format json --compress --output bi_data.json
```

### Testing

```bash
# Create test fixtures
flash export --limit 50 --format json --output tests/fixtures/

# Export test database state
flash export --format sqlite --output tests/test.db
```

## Export Configuration

### flash.config.json

```json
{
  "export": {
    "default_format": "json",
    "output_dir": "db/export",
    "compress": true,
    "parallel": true,
    "batch_size": 1000,
    "exclude_tables": ["_flash_migrations", "sessions"]
  }
}
```

### Environment Variables

```bash
# Override config with environment
EXPORT_FORMAT=json
EXPORT_COMPRESS=true
EXPORT_PARALLEL=4
```

## Import Functionality

While primarily an export tool, Flash ORM also supports basic import:

```bash
# Import JSON data
flash import --file backup.json

# Import CSV data
flash import --file data.csv --table users

# Import SQLite database
flash import --file backup.db
```

## Monitoring & Logging

### Export Progress

```bash
# Verbose output
flash export --verbose

# Progress bar (default)
flash export

# Quiet mode
flash export --quiet
```

### Export Logs

```bash
# Log export operations
flash export --log export.log

# Log errors only
flash export --log errors.log --log-level error
```

### Performance Metrics

```bash
# Show timing information
flash export --metrics

# Output:
# Export completed in 45.2 seconds
# Tables processed: 15
# Total records: 1,234,567
# Average speed: 27,300 records/second
```

## Troubleshooting

### Common Issues

**Export fails with "permission denied"**
```bash
# Check output directory permissions
ls -la db/export/

# Create directory if needed
mkdir -p db/export/

# Change permissions
chmod 755 db/export/
```

**Memory errors on large exports**
```bash
# Reduce batch size
flash export --batch-size 1000

# Use streaming
flash export --stream

# Increase memory limit
flash export --memory-limit 1GB
```

**Slow exports**
```bash
# Check database indexes
flash status --indexes

# Add missing indexes
CREATE INDEX CONCURRENTLY idx_users_created_at ON users(created_at);

# Use parallel processing
flash export --parallel 8
```

**Incomplete exports**
```bash
# Check export logs
cat export.log

# Verify table counts
flash export --count-only

# Resume interrupted export
flash export --resume
```

### Validation

```bash
# Validate export file
flash export --validate backup.json

# Check data integrity
flash export --checksum backup.json
```

### Recovery

```bash
# List available backups
flash export --list

# Restore from backup
flash import --file backup.json

# Compare with current database
flash export --diff backup.json
```

## Best Practices

### Backup Strategy

1. **Regular Backups**: Schedule daily/weekly backups
2. **Multiple Formats**: Keep JSON for flexibility, SQLite for testing
3. **Compression**: Always compress large exports
4. **Verification**: Regularly test backup restoration

### Security

1. **Encrypt Sensitive Data**: Encrypt exports containing PII
2. **Access Control**: Restrict access to export files
3. **Secure Storage**: Store backups in secure locations
4. **Retention Policy**: Define backup retention periods

### Performance

1. **Off-Peak Hours**: Run large exports during low-traffic periods
2. **Resource Monitoring**: Monitor database and system resources
3. **Incremental Backups**: Use `--since` for frequent small backups
4. **Parallel Processing**: Use multiple cores for large exports

### Organization

1. **Naming Convention**: Use timestamps in filenames
   ```
   backup_20240115_143000.json.gz
   ```

2. **Directory Structure**:
   ```
   backups/
   ├── daily/
   ├── weekly/
   ├── monthly/
   └── schema/
   ```

3. **Documentation**: Keep backup logs and metadata

Data export is a critical part of database management. Flash ORM's export system ensures your data is safely extracted, properly formatted, and ready for any use case - from simple backups to complex data migrations.
