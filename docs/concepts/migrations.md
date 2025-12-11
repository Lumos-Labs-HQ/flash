---
title: Migrations
description: Database migration management in Flash ORM
---

# Migrations

Flash ORM provides a robust migration system that safely manages database schema changes across development, staging, and production environments.

## Table of Contents

- [Migration Basics](#migration-basics)
- [Creating Migrations](#creating-migrations)
- [Applying Migrations](#applying-migrations)
- [Migration Files](#migration-files)
- [Safe Migration System](#safe-migration-system)
- [Branch-Aware Migrations](#branch-aware-migrations)
- [Migration Best Practices](#migration-best-practices)
- [Troubleshooting](#troubleshooting)

## Migration Basics

### What are Migrations?

Migrations are version-controlled database schema changes. Each migration has:

- **Up migration**: Forward changes (create tables, add columns, etc.)
- **Down migration**: Rollback changes (drop tables, remove columns, etc.)
- **Timestamp**: When the migration was created
- **Name**: Descriptive name of the changes

### Migration Workflow

```bash
# 1. Create migration
flash migrate "add user profiles table"

# 2. Edit migration files (created automatically)
# Edit: db/migrations/20240101120000_add_user_profiles_table.up.sql
# Edit: db/migrations/20240101120000_add_user_profiles_table.down.sql

# 3. Apply migration
flash apply

# 4. Check status
flash status
```

## Creating Migrations

### Interactive Migration Creation

```bash
flash migrate
```

You'll be prompted to enter a migration name:

```
Enter migration name: add user authentication
```

### Named Migration Creation

```bash
flash migrate "add user authentication"
flash migrate "create products table"
flash migrate "add foreign key constraints"
```

### Auto-Generated Migrations

Flash ORM can automatically generate migrations by comparing your schema files with the current database:

```bash
flash migrate "sync with schema" --auto
```

This analyzes your `db/schema/` files and creates appropriate up/down migrations.

## Applying Migrations

### Apply All Pending Migrations

```bash
flash apply
```

### Apply Specific Number

```bash
# Apply only the next migration
flash apply --count 1

# Apply next 3 migrations
flash apply --count 3
```

### Dry Run

See what would be executed without actually running it:

```bash
flash apply --dry-run
```

### Force Apply (Dangerous)

```bash
# Skip safety checks (use with caution)
flash apply --force
```

## Migration Files

### File Structure

Migrations are stored in `db/migrations/` with timestamp prefixes:

```
db/migrations/
├── 20240101120000_initial_schema.up.sql
├── 20240101120000_initial_schema.down.sql
├── 20240102130000_add_user_profiles.up.sql
├── 20240102130000_add_user_profiles.down.sql
├── 20240103140000_create_products.up.sql
└── 20240103140000_create_products.down.sql
```

### Up Migration Example

```sql
-- db/migrations/20240102130000_add_user_profiles.up.sql
BEGIN;

CREATE TABLE user_profiles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    bio TEXT,
    avatar_url VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_profiles_user_id ON user_profiles(user_id);
CREATE UNIQUE INDEX idx_user_profiles_unique_user ON user_profiles(user_id);

ALTER TABLE users ADD COLUMN profile_complete BOOLEAN DEFAULT FALSE;

COMMIT;
```

### Down Migration Example

```sql
-- db/migrations/20240102130000_add_user_profiles.down.sql
BEGIN;

DROP TABLE user_profiles;
ALTER TABLE users DROP COLUMN profile_complete;

COMMIT;
```

### Migration Tracking

Flash ORM tracks applied migrations in a special table:

```sql
-- Automatically created
CREATE TABLE _flash_migrations (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    checksum VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Safe Migration System

### Transaction-Based Execution

All migrations run in transactions for atomicity:

```sql
BEGIN;
-- Your migration SQL here
-- If any statement fails, entire migration rolls back
COMMIT;
```

### Pre-Flight Checks

Before applying migrations, Flash ORM performs safety checks:

- **Schema validation**: Ensures migration SQL is syntactically correct
- **Dependency checking**: Verifies required tables/constraints exist
- **Conflict detection**: Prevents conflicting schema changes
- **Data integrity**: Checks for potential data loss

### Automatic Rollback

If a migration fails, Flash ORM automatically rolls back all changes:

```bash
flash apply
# Migration fails halfway through
# Automatic rollback - database returns to previous state
```

### Checksum Verification

Each migration has a checksum to prevent tampering:

```sql
-- Migration record includes checksum
INSERT INTO _flash_migrations (id, name, checksum, applied_at)
VALUES ('20240102130000', 'add user profiles', 'abc123...', NOW());
```

## Branch-Aware Migrations

### Git-like Branching for Databases

Flash ORM supports database branching, allowing you to manage schema changes across branches:

```bash
# Create a feature branch
flash branch create feature/user-profiles

# Make schema changes
flash migrate "add profile fields"

# Switch branches
flash checkout main

# Merge when ready
flash branch merge feature/user-profiles
```

### Branch Metadata

Branch information is stored separately from migrations:

```
db/migrations/branches/
├── main.json
├── feature/user-profiles.json
└── hotfix/security-patch.json
```

### Conflict Resolution

When merging branches with conflicting migrations:

```bash
flash branch merge feature/user-profiles
# Conflict detected!
# Please resolve conflicts in: db/migrations/conflicts/

# After resolving conflicts
flash branch merge feature/user-profiles --resolved
```

## Migration Best Practices

### Naming Conventions

```bash
# Good migration names
flash migrate "create users table"
flash migrate "add email unique constraint"
flash migrate "create products and categories tables"
flash migrate "add user authentication system"

# Avoid vague names
flash migrate "fix"          # Too vague
flash migrate "update"       # Too vague
flash migrate "change"       # Too vague
```

### Keep Migrations Small

```sql
-- Good: Small, focused migration
-- 20240101120000_create_users.up.sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Bad: Large migration with multiple concerns
-- Avoid combining unrelated changes
```

### Always Provide Rollbacks

```sql
-- Up migration
ALTER TABLE users ADD COLUMN phone VARCHAR(20);

-- Down migration (required!)
ALTER TABLE users DROP COLUMN phone;
```

### Test Migrations

```bash
# Test on development first
flash apply

# Test rollback
flash reset  # Rollback all migrations
flash apply  # Reapply all migrations

# Test on staging
# Deploy to staging environment
flash apply
```

### Use Transactions Wisely

```sql
-- Good: Wrap multi-statement migrations in transactions
BEGIN;
CREATE TABLE orders (...);
CREATE TABLE order_items (...);
CREATE INDEX idx_orders_user_id ON orders(user_id);
COMMIT;

-- Avoid: Long-running transactions that lock tables
BEGIN;
UPDATE large_table SET status = 'processed'; -- This might lock the table for a long time
COMMIT;
```

### Document Breaking Changes

```sql
-- Document in migration comments
-- BREAKING CHANGE: Renames user_id to account_id in orders table
-- This affects: API endpoints, frontend code, reports
ALTER TABLE orders RENAME COLUMN user_id TO account_id;
```

## Migration Status

### Check Migration Status

```bash
flash status
```

Output:
```
Migration Status:
✅ 20240101120000 - initial schema
✅ 20240102130000 - add user profiles
⏳ 20240103140000 - create products table (pending)
❌ 20240104150000 - add payment system (failed)
```

### Migration History

```bash
# See applied migrations
flash status --applied

# See pending migrations
flash status --pending

# See failed migrations
flash status --failed
```

## Advanced Migration Features

### Conditional Migrations

```sql
-- Only run if table doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_preferences') THEN
        CREATE TABLE user_preferences (
            id SERIAL PRIMARY KEY,
            user_id INTEGER REFERENCES users(id),
            preferences JSONB
        );
    END IF;
END $$;
```

### Data Migrations

```sql
-- Migrate existing data
BEGIN;

-- Add new column with default
ALTER TABLE users ADD COLUMN full_name VARCHAR(255);

-- Populate with existing data
UPDATE users SET full_name = CONCAT(first_name, ' ', last_name)
WHERE full_name IS NULL;

-- Make it NOT NULL after data migration
ALTER TABLE users ALTER COLUMN full_name SET NOT NULL;

COMMIT;
```

### Migration Dependencies

```sql
-- Ensure dependent objects exist
-- name: ensure_users_table_exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users') THEN
        RAISE EXCEPTION 'Users table must exist before creating profiles';
    END IF;
END $$;

CREATE TABLE user_profiles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id)
);
```

## Troubleshooting

### Common Issues

**Migration fails with "table already exists"**

```bash
# Check current status
flash status

# Check if migration was partially applied
flash status --detailed

# Reset and reapply if needed
flash reset
flash apply
```

**Migration stuck in transaction**

```bash
# Check for long-running queries
SELECT * FROM pg_stat_activity WHERE state = 'active';

# Kill the stuck transaction if necessary
SELECT pg_cancel_backend(pid);
```

**Migration checksum mismatch**

```sql
-- Migration file was modified after being applied
-- Restore original file or create new migration for changes
flash migrate "fix modified migration"
```

**Foreign key constraint violations**

```sql
-- Check for orphaned records
SELECT * FROM child_table WHERE parent_id NOT IN (SELECT id FROM parent_table);

-- Clean up orphaned records before migration
DELETE FROM child_table WHERE parent_id NOT IN (SELECT id FROM parent_table);
```

### Recovery Procedures

**Recover from failed migration**

```bash
# Check what failed
flash status

# Manual rollback if needed
# Edit the down migration and run manually
psql $DATABASE_URL -f db/migrations/20240103140000_failed_migration.down.sql

# Mark migration as rolled back
# Edit _flash_migrations table directly (careful!)
```

**Reset database to clean state**

```bash
# Drop all tables and reapply from scratch
flash reset

# Or drop and recreate database
# Then reapply all migrations
flash apply
```

### Debugging Migrations

**Enable verbose logging**

```bash
flash apply --verbose
```

**Test migration on copy of production**

```bash
# Create backup
pg_dump production_db > backup.sql

# Restore to test database
createdb test_db
psql test_db < backup.sql

# Test migration
DATABASE_URL="postgres://.../test_db" flash apply
```

**Use migration validation**

```bash
# Validate migration SQL syntax
flash validate

# Check for potential issues
flash validate --strict
```

Remember: Migrations are permanent changes to your database schema. Always test thoroughly on development and staging environments before applying to production. Keep backups and have a rollback plan ready.
