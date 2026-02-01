---
title: MongoDB Studio
description: Visual MongoDB management interface
---

# MongoDB Studio

FlashORM MongoDB Studio provides a powerful, visual interface for managing your MongoDB databases, similar to MongoDB Compass but integrated directly into FlashORM.

::: info Note
MongoDB Studio is a **visual management tool only**. FlashORM's ORM features (migrations, code generation, seeding) are designed for SQL databases (PostgreSQL, MySQL, SQLite). MongoDB Studio provides database browsing, document editing, and query execution - not ORM functionality.
:::

## Table of Contents

- [Quick Start](#quick-start)
- [Database Browser](#database-browser)
- [Collection Management](#collection-management)
- [Document Operations](#document-operations)
- [Query Interface](#query-interface)
- [Bulk Operations](#bulk-operations)
- [Database Statistics](#database-statistics)
- [Connection Options](#connection-options)

## Quick Start

```bash
# Start MongoDB Studio
flash studio mongodb --url "mongodb://localhost:27017"

# With authentication
flash studio mongodb --url "mongodb://user:password@localhost:27017/mydb"

# MongoDB Atlas connection
flash studio mongodb --url "mongodb+srv://user:password@cluster.mongodb.net/mydb"

# Custom port
flash studio mongodb --url "mongodb://localhost:27017" --port 3000
```

## Database Browser

### 📂 Browse Databases and Collections

The left sidebar displays all databases and their collections:

**Features:**
- View all databases in your MongoDB instance
- Expand databases to see collections
- Collection document counts
- Quick collection selection
- Create new databases
- Delete databases with confirmation

### Database Operations

| Operation | Description |
|-----------|-------------|
| Create Database | Creates a new database with an initial collection |
| Delete Database | Drops entire database (requires confirmation) |
| Refresh | Reload database and collection list |

## Collection Management

### 📁 Manage Collections

Right-click context menu provides collection management options:

**Context Menu Options:**
- **View Documents** - Browse collection documents
- **Create Index** - Add new indexes
- **View Indexes** - See existing indexes
- **Drop Collection** - Delete the collection
- **Rename Collection** - Change collection name
- **Collection Stats** - View collection statistics

### Create New Collection

1. Right-click on a database
2. Select "Create Collection"
3. Enter collection name
4. Optionally configure validation rules

### Collection Statistics

View detailed collection information:
- Document count
- Average document size
- Total data size
- Index count and size
- Storage statistics

## Document Operations

### 📄 View Documents

Browse documents with a clean, syntax-highlighted JSON viewer:

**Features:**
- Paginated document listing (configurable page size)
- Syntax-highlighted JSON display
- Expandable/collapsible nested objects
- Copy document as JSON
- View raw BSON data

### Create Documents

Add new documents to a collection:

```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "age": 30,
  "tags": ["developer", "nodejs"],
  "address": {
    "city": "New York",
    "country": "USA"
  }
}
```

**Features:**
- JSON editor with syntax validation
- Auto-generated `_id` if not provided
- Support for all BSON types
- Bulk insert multiple documents

### Edit Documents

Click on any document to edit inline:

**Features:**
- Direct JSON editing
- Field-level updates
- Validation before save
- Preserve document structure

### Delete Documents

**Single Document:**
- Click delete button on document row
- Requires confirmation

**Bulk Delete:**
- Select multiple documents using checkboxes
- Click "Delete Selected"
- Uses efficient `$in` operator
- Shows count of deleted documents

## Query Interface

### 🔍 Find Documents

Use MongoDB query syntax to search documents:

```javascript
// Find by field
{ "name": "John" }

// Find with operators
{ "age": { "$gt": 25 } }

// Find with multiple conditions
{ "status": "active", "role": "admin" }

// Find with regex
{ "email": { "$regex": ".*@gmail.com" } }
```

### Query Options

| Option | Description |
|--------|-------------|
| Filter | MongoDB query filter |
| Projection | Fields to include/exclude |
| Sort | Sort order (`{ "created_at": -1 }`) |
| Limit | Maximum documents to return |
| Skip | Number of documents to skip |

### Aggregation Pipeline

Execute aggregation pipelines:

```javascript
[
  { "$match": { "status": "active" } },
  { "$group": { "_id": "$category", "count": { "$sum": 1 } } },
  { "$sort": { "count": -1 } }
]
```

## Bulk Operations

### 🗑️ Bulk Delete Documents

Delete multiple documents efficiently:

1. Select documents using checkboxes
2. Click "Delete Selected" button
3. Confirm the deletion
4. Documents are deleted using `$in` operator for efficiency

**Features:**
- Select all on current page
- Clear selection
- Shows selection count
- Efficient batch deletion

### Bulk Update (Coming Soon)

Update multiple documents at once with:
- Field updates
- Array operations
- Increment/decrement values

## Database Statistics

### 📊 View Database Metrics

Access comprehensive statistics for any database:

**Database Stats:**
- Total collections
- Total documents
- Data size
- Storage size
- Index size
- Average object size

**Collection Stats:**
- Document count
- Data size
- Index information
- Capped collection status

## Connection Options

### Local MongoDB

```bash
flash studio mongodb --url "mongodb://localhost:27017"
```

### With Authentication

```bash
flash studio mongodb --url "mongodb://user:password@localhost:27017/mydb?authSource=admin"
```

### MongoDB Atlas

```bash
flash studio mongodb --url "mongodb+srv://user:password@cluster.mongodb.net/mydb"
```

### Replica Set

```bash
flash studio mongodb --url "mongodb://host1:27017,host2:27017,host3:27017/mydb?replicaSet=rs0"
```

### Connection with Options

```bash
flash studio mongodb --url "mongodb://localhost:27017/mydb?maxPoolSize=10&minPoolSize=5"
```

## UI Features

### Dark Theme

MongoDB Studio uses a modern dark theme optimized for long sessions, consistent with the FlashORM design language.

### Collection Selection

Active collection highlighting makes it easy to see which collection you're working with:
- Highlighted background on selected collection
- Breadcrumb navigation showing database > collection
- Quick switch between collections

### Responsive Layout

- Resizable sidebar
- Collapsible panels
- Works on desktop and tablet devices

## Tips & Best Practices

### Efficient Queries

Use indexes for frequently queried fields:
```javascript
// Create index in Studio
{ "email": 1 }  // Single field ascending
{ "user_id": 1, "created_at": -1 }  // Compound index
```

### Document Validation

Set up schema validation for data integrity:
```javascript
{
  "$jsonSchema": {
    "bsonType": "object",
    "required": ["email", "name"],
    "properties": {
      "email": {
        "bsonType": "string",
        "pattern": "^.+@.+$"
      }
    }
  }
}
```

### Bulk Operations

For large deletions, use bulk delete instead of deleting one by one:
- More efficient network usage
- Faster operation
- Atomic within the batch

### Backup Before Delete

Before deleting databases or performing bulk operations:
1. Use `flash export` to backup data
2. Verify the backup
3. Then proceed with deletion

## Troubleshooting

### Connection Issues

**Cannot connect to MongoDB:**
```bash
# Check if MongoDB is running
mongosh --eval "db.adminCommand('ping')"

# Verify connection string
flash studio mongodb --url "mongodb://localhost:27017" --verbose
```

**Authentication failed:**
- Verify username and password
- Check authSource parameter
- Ensure user has appropriate roles

### Performance Issues

**Slow document loading:**
- Add indexes on frequently queried fields
- Use projection to limit returned fields
- Implement pagination for large collections

**Memory usage:**
- Limit page size for large documents
- Use aggregation with `$limit` and `$skip`
- Avoid loading entire collections

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+N` | Create new document |
| `Ctrl+S` | Save current document |
| `Ctrl+F` | Open find dialog |
| `Delete` | Delete selected documents |
| `Escape` | Cancel current operation |

## API Reference

MongoDB Studio exposes a REST API for automation:

```bash
# List databases
GET /api/databases

# List collections
GET /api/databases/:db/collections

# Query documents
POST /api/databases/:db/collections/:collection/find
Content-Type: application/json
{ "filter": {}, "limit": 10 }

# Insert document
POST /api/databases/:db/collections/:collection/insert
Content-Type: application/json
{ "document": { "name": "John" } }

# Delete documents
POST /api/databases/:db/collections/:collection/delete
Content-Type: application/json
{ "ids": ["id1", "id2", "id3"] }
```
