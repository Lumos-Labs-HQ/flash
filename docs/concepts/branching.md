---
title: Database Branching
description: Git-like branching for database schemas
---

# Database Branching

Flash ORM introduces Git-like branching for database schemas, allowing you to manage database changes across development branches just like you manage code changes.

## Table of Contents

- [Why Database Branching?](#why-database-branching)
- [Branch Basics](#branch-basics)
- [Creating Branches](#creating-branches)
- [Switching Branches](#switching-branches)
- [Making Changes](#making-changes)
- [Merging Branches](#merging-branches)
- [Conflict Resolution](#conflict-resolution)
- [Branch Management](#branch-management)
- [Advanced Workflows](#advanced-workflows)

## Why Database Branching?

### The Problem

Traditional database development has challenges:

- **Schema conflicts** between development branches
- **Manual migration management** across environments
- **Difficult rollbacks** when features are abandoned
- **Complex deployment coordination** for schema changes

### The Solution

Database branching provides:

- **Isolated schema changes** per branch
- **Automatic conflict detection** and resolution
- **Branch-specific migrations** that follow code branches
- **Safe merging** with preview and rollback capabilities

## Branch Basics

### Branch Structure

```
database/
├── main/
│   ├── migrations/
│   │   ├── 001_initial.sql
│   │   └── 002_add_users.sql
│   └── schema.sql
├── feature/user-auth/
│   ├── migrations/
│   │   ├── 003_add_auth_tables.sql
│   │   └── 004_add_sessions.sql
│   └── schema.sql
└── hotfix/security/
    ├── migrations/
    └── schema.sql
```

### Branch Metadata

Each branch tracks:

- **Parent branch**: Which branch it was created from
- **Created date**: When the branch was created
- **Applied migrations**: Which migrations have been applied
- **Schema checksum**: Current schema state
- **Merge history**: Previous merges and conflicts

## Creating Branches

### Create from Current Branch

```bash
# Create a new feature branch
flash branch create feature/user-profiles

# Create a bug fix branch
flash branch create bugfix/user-validation

# Create from specific branch
flash branch create feature/new-ui --from main
```

### Branch Naming Conventions

```bash
# Feature branches
flash branch create feature/user-authentication
flash branch create feature/payment-system
flash branch create feature/admin-dashboard

# Bug fix branches
flash branch create bugfix/login-validation
flash branch create bugfix/data-corruption

# Hotfix branches (for production issues)
flash branch create hotfix/security-patch
flash branch create hotfix/critical-bug

# Release branches
flash branch create release/v2.1.0
```

### Branch Options

```bash
# Create and switch immediately
flash branch create feature/new-api --switch

# Create with description
flash branch create feature/user-profiles --description "Add user profile management"

# Create empty branch (no migrations)
flash branch create experiment/empty-schema --empty
```

## Switching Branches

### Switch to Branch

```bash
# Switch to feature branch
flash branch switch feature/user-profiles

# Or use checkout (alias)
flash checkout feature/user-profiles

# Switch back to main
flash checkout main
```

### Branch Status

```bash
# See current branch
flash branch status

# Output:
# Current branch: feature/user-profiles
# Parent: main
# Created: 2024-01-15 14:30:00
# Migrations: 3 applied, 0 pending
# Schema: modified
```

### Branch List

```bash
# List all branches
flash branch list

# Output:
# main (current)
# ├── feature/user-profiles
# ├── feature/payment-system
# └── bugfix/validation
```

## Making Changes

### Schema Changes in Branches

```bash
# Switch to feature branch
flash checkout feature/user-profiles

# Make schema changes
flash migrate "add profile table"

# Edit schema files
# db/schema/profiles.sql

# Apply changes
flash apply

# Generate code
flash gen
```

### Branch-Specific Migrations

Migrations created in a branch are branch-specific:

```
db/migrations/branches/feature/user-profiles/
├── 20240115143000_add_profiles_table.up.sql
├── 20240115143000_add_profiles_table.down.sql
└── metadata.json
```

### Committing Changes

```bash
# Commit branch changes
flash branch commit "Add user profiles feature"

# This creates a checkpoint for potential rollback
```

## Merging Branches

### Fast-Forward Merge

```bash
# Switch to target branch
flash checkout main

# Merge feature branch
flash branch merge feature/user-profiles

# If no conflicts, automatically merges
```

### Three-Way Merge

When branches have diverged:

```bash
# Merge with conflict resolution
flash branch merge feature/user-profiles --strategy three-way

# Preview merge
flash branch merge feature/user-profiles --preview
```

### Merge Options

```bash
# Dry run merge
flash branch merge feature/user-profiles --dry-run

# Force merge (skip safety checks)
flash branch merge feature/user-profiles --force

# Squash merge (combine all migrations)
flash branch merge feature/user-profiles --squash
```

## Conflict Resolution

### Automatic Conflict Detection

Flash ORM detects conflicts in:

- **Table creations**: Same table created in both branches
- **Column additions**: Same column added with different types
- **Index conflicts**: Conflicting index definitions
- **Constraint conflicts**: Foreign key or unique constraints

### Conflict Resolution Process

```bash
# Attempt merge
flash branch merge feature/user-profiles

# If conflicts found:
# Conflict detected in table 'users':
# - Branch main: added column 'phone' VARCHAR(20)
# - Branch feature/user-profiles: added column 'phone' TEXT
#
# Choose resolution:
# 1. Keep main version
# 2. Keep feature version
# 3. Manual resolution

flash branch resolve --choice 1
```

### Manual Resolution

```bash
# Edit conflict manually
flash branch resolve --manual

# Opens conflict resolution interface
# Choose which changes to keep
# Edit migration SQL if needed

# Mark as resolved
flash branch resolve --mark-resolved
```

### Conflict Types

#### Schema Conflicts

```sql
-- Conflict: Column type mismatch
-- Branch A: ALTER TABLE users ADD COLUMN phone VARCHAR(20);
-- Branch B: ALTER TABLE users ADD COLUMN phone TEXT;

-- Resolution: Choose one or create new migration
ALTER TABLE users ADD COLUMN phone VARCHAR(50);
```

#### Migration Conflicts

```sql
-- Conflict: Same table created twice
-- Branch A: CREATE TABLE profiles (...);
-- Branch B: CREATE TABLE profiles (...);

-- Resolution: Merge table definitions or rename
```

## Branch Management

### Branch Information

```bash
# Detailed branch info
flash branch info feature/user-profiles

# Output:
# Branch: feature/user-profiles
# Parent: main
# Created: 2024-01-15 14:30:00
# Author: john.doe
# Status: active
# Migrations: 5 applied
# Conflicts: 0 pending
```

### Branch Cleanup

```bash
# Delete merged branch
flash branch delete feature/completed-feature

# Force delete (even if not merged)
flash branch delete feature/abandoned --force

# Clean up old branches
flash branch prune --older-than 30days
```

### Branch History

```bash
# See branch creation and merge history
flash branch log

# See specific branch history
flash branch log feature/user-profiles
```

## Advanced Workflows

### Feature Development Workflow

```bash
# 1. Create feature branch
flash branch create feature/user-dashboard

# 2. Develop feature
flash migrate "add dashboard tables"
flash apply
flash gen

# 3. Test changes
npm test
flash studio  # Visual verification

# 4. Commit branch
flash branch commit "Implement user dashboard"

# 5. Merge to main
flash checkout main
flash branch merge feature/user-dashboard

# 6. Deploy
flash apply  # Apply to production
```

### Hotfix Workflow

```bash
# 1. Create hotfix from production/main
flash branch create hotfix/critical-security-fix --from main

# 2. Fix the issue
flash migrate "fix security vulnerability"
flash apply

# 3. Test thoroughly
# Run security tests, integration tests

# 4. Merge directly to main
flash checkout main
flash branch merge hotfix/critical-security-fix --force

# 5. Deploy immediately
flash apply --force
```

### Release Workflow

```bash
# 1. Create release branch
flash branch create release/v2.1.0 --from main

# 2. Final testing
flash checkout release/v2.1.0
# Run full test suite

# 3. Tag release
git tag v2.1.0
git push origin v2.1.0

# 4. Merge back to main
flash checkout main
flash branch merge release/v2.1.0

# 5. Clean up
flash branch delete release/v2.1.0
```

### Collaborative Development

```bash
# Team member A
flash branch create feature/user-auth
flash migrate "add auth tables"
flash branch commit "Add authentication tables"

# Team member B
flash pull  # Get latest changes
flash branch create feature/user-profiles
flash migrate "add profile tables"

# Merge workflow
flash checkout main
flash branch merge feature/user-auth
flash branch merge feature/user-profiles
```

## Branch Configuration

### flash.config.json

```json
{
  "branching": {
    "enabled": true,
    "default_strategy": "three-way",
    "auto_commit": true,
    "conflict_resolution": "interactive",
    "storage_path": "db/branches"
  }
}
```

### Branch Policies

```json
{
  "branch_policies": {
    "main": {
      "protected": true,
      "required_reviews": 2,
      "auto_merge": false
    },
    "release/*": {
      "protected": true,
      "required_tests": true
    },
    "feature/*": {
      "max_age_days": 90,
      "auto_prune": true
    }
  }
}
```

## Troubleshooting

### Common Issues

**Branch switch fails**
```bash
# Check for uncommitted changes
flash status

# Commit or stash changes
flash branch commit "Work in progress"

# Then switch
flash checkout other-branch
```

**Merge conflicts**
```bash
# List conflicts
flash branch conflicts

# Resolve interactively
flash branch resolve

# Or abort merge
flash branch merge --abort
```

**Branch corruption**
```bash
# Validate branch integrity
flash branch validate feature/broken-branch

# Repair branch
flash branch repair feature/broken-branch

# Recreate if necessary
flash branch recreate feature/broken-branch
```

### Recovery Procedures

**Recover from failed merge**
```bash
# Abort merge
flash branch merge --abort

# Reset to previous state
flash reset --hard HEAD~1

# Clean up
flash branch cleanup
```

**Fix branch metadata**
```bash
# Rebuild branch metadata
flash branch rebuild-metadata

# Verify integrity
flash branch verify
```

## Performance Considerations

### Branch Storage

- **Efficient Storage**: Branches share unchanged migrations
- **Compression**: Automatic compression of branch data
- **Cleanup**: Automatic cleanup of old branch data

### Merge Performance

- **Incremental Diffs**: Only compare changed schemas
- **Parallel Processing**: Fast conflict detection
- **Memory Efficient**: Stream processing for large schemas

## Integration with Git

### Git Integration

```bash
# Link database branches to git branches
flash branch link feature/user-auth --git-branch feature/user-auth

# Auto-switch database branch on git checkout
# Add to .git/hooks/post-checkout:
#!/bin/bash
flash checkout $(git branch --show-current)
```

### CI/CD Integration

```yaml
# .github/workflows/pr.yml
name: PR Checks
on: [pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Setup Database
        run: |
          flash checkout ${{ github.head_ref }}
          flash apply
      - name: Run Tests
        run: npm test
```

Database branching transforms how teams work with database schemas. By treating database changes like code changes, teams can develop features independently, resolve conflicts safely, and deploy with confidence. This approach eliminates the traditional pain points of database development while maintaining data integrity and team productivity.
