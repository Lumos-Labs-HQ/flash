---
title: Redis Studio
description: Visual Redis management interface
---

# Redis Studio

FlashORM includes a powerful Redis management interface inspired by Upstash, providing a beautiful and intuitive way to manage your Redis databases.

## Table of Contents

- [Quick Start](#quick-start)
- [Key Browser](#key-browser)
- [CLI Terminal](#cli-terminal)
- [Statistics Dashboard](#statistics-dashboard)
- [Memory Analysis](#memory-analysis)
- [Slow Log Viewer](#slow-log-viewer)
- [Lua Script Editor](#lua-script-editor)
- [Pub/Sub Management](#pubsub-management)
- [Configuration Editor](#configuration-editor)
- [ACL Management](#acl-management)
- [Cluster & Replication](#cluster--replication)
- [Export/Import](#exportimport)
- [Bulk Operations](#bulk-operations)
- [Connection Options](#connection-options)
- [Keyboard Shortcuts](#keyboard-shortcuts)

## Quick Start

```bash
# Start Redis Studio
flash studio redis --url "redis://localhost:6379"

# With password
flash studio redis --url "redis://:password@localhost:6379"

# Custom port
flash studio redis --url "redis://localhost:6379" --port 3000

# Open without browser auto-launch
flash studio redis --url "redis://localhost:6379" --no-browser
```

## Key Browser

### 🗂️ Browse and Manage Keys

- View all keys with type indicators (STRING, LIST, SET, HASH, ZSET)
- Search keys with pattern matching (e.g., `user:*`)
- View key details including TTL, type, and value
- Create, edit, and delete keys
- Bulk delete multiple keys at once

### Supported Data Types

| Type | View | Edit | Create | Delete |
|------|------|------|--------|--------|
| STRING | ✅ | ✅ | ✅ | ✅ |
| LIST | ✅ | ✅ | ✅ | ✅ |
| SET | ✅ | ✅ | ✅ | ✅ |
| HASH | ✅ | ✅ | ✅ | ✅ |
| ZSET | ✅ | ✅ | ✅ | ✅ |
| STREAM | ✅ | - | - | ✅ |

### Database Selector

Switch between Redis databases (db0-db15):
- Dropdown selector in the navigation bar
- Shows key count per database
- State persists across sessions

## CLI Terminal

### 💻 Full Redis CLI Experience

Interactive command-line interface with full Redis command support:

```
redis> SET mykey "hello"
OK
redis> GET mykey
"hello"
redis> HSET user:1 name "John" age 30
(integer) 2
redis> KEYS user:*
1) "user:1"
```

**Features:**
- Command history with ↑↓ arrow keys
- Tab completion for commands
- Syntax highlighting
- Multi-line command support
- Command output formatting

## Statistics Dashboard

### 📊 Server Information

Real-time server statistics including:
- **Memory**: Used memory, peak memory, RSS memory, fragmentation ratio
- **Clients**: Connected clients, blocked clients
- **Keys**: Total keys, keys with expiry
- **Commands**: Total commands processed, commands per second
- **Server**: Redis version, uptime, mode (standalone/cluster)
- **Replication**: Role, connected slaves

## Memory Analysis

### 🧠 Per-Key Memory Usage

Analyze memory consumption across your Redis instance:

```bash
# Access via Memory tab in Redis Studio
```

**Features:**
- Per-key memory usage analysis
- Memory usage by type (STRING, HASH, LIST, etc.)
- Memory overview with peak/used/RSS statistics
- Fragmentation ratio monitoring
- Sort keys by memory consumption
- Pattern-based memory analysis

**Memory Overview Dashboard:**
| Metric | Description |
|--------|-------------|
| Used Memory | Current memory usage |
| Peak Memory | Maximum memory ever used |
| RSS Memory | Resident Set Size |
| Fragmentation Ratio | Memory fragmentation level |

## Slow Log Viewer

### 🐌 Query Performance Analysis

View slow queries to identify performance bottlenecks:

**Features:**
- View slow query history with timestamps
- Duration in microseconds/milliseconds
- Full command text
- Client address and name
- Clear slow log functionality

**Slow Log Entry:**
| Field | Description |
|-------|-------------|
| ID | Unique entry identifier |
| Time | When the query was executed |
| Duration | How long the query took |
| Command | The Redis command that was slow |
| Client | Client address that issued the command |

## Lua Script Editor

### 📜 Execute and Manage Lua Scripts

Full-featured Lua scripting environment:

**Features:**
- Write and execute Lua scripts directly
- KEYS and ARGV parameter support
- Script loading (SCRIPT LOAD) with SHA return
- Execute scripts by SHA (EVALSHA)
- Script result display with duration
- Flush all loaded scripts

**Example Script:**
```lua
-- Return the sum of two values
local a = redis.call('GET', KEYS[1])
local b = redis.call('GET', KEYS[2])
return tonumber(a) + tonumber(b)
```

**Usage:**
1. Enter your Lua script in the editor
2. Add KEYS (comma-separated): `key1, key2`
3. Add ARGV (comma-separated): `arg1, arg2`
4. Click "Execute" to run or "Load Script" to cache

## Pub/Sub Management

### 📢 Publish Messages and View Channels

Manage Redis Pub/Sub functionality:

**Features:**
- Publish messages to any channel
- View all active channels
- See subscriber counts per channel
- Pattern subscriber count
- Real-time channel list refresh

**Publish a Message:**
1. Enter channel name
2. Enter message content
3. Click "Publish"
4. See number of subscribers that received the message

## Configuration Editor

### ⚙️ View and Modify Redis Configuration

Manage Redis server configuration in real-time:

**Features:**
- View all configuration parameters
- Filter by pattern (e.g., `max*`)
- Edit configuration values inline
- Save individual configuration changes
- Rewrite configuration file
- Reset server statistics

**Common Configuration Parameters:**
| Parameter | Description |
|-----------|-------------|
| maxmemory | Maximum memory limit |
| maxclients | Maximum client connections |
| timeout | Client timeout |
| tcp-keepalive | TCP keepalive interval |
| databases | Number of databases |

## ACL Management

### 🔐 Access Control Lists (Redis 6.0+)

Manage Redis users and permissions:

**Features:**
- View all ACL users with their rules
- Create new users with custom permissions
- Delete users
- View ACL security log
- Clear ACL log entries

**ACL Log Entries:**
| Field | Description |
|-------|-------------|
| Reason | Why access was denied |
| Context | Command context |
| Object | Key or command that was accessed |
| Username | User that attempted the action |
| Age | How long ago the event occurred |

::: warning
ACL features require Redis 6.0 or higher.
:::

## Cluster & Replication

### 🔄 View Cluster and Replication Status

Monitor your Redis deployment topology:

**Replication Info:**
- Role (master/slave)
- Connected slaves count
- Master host and port (for replicas)
- Master link status
- Slave details (IP, port, state, offset)

**Cluster Info (when enabled):**
- Cluster state (ok/fail)
- Known nodes count
- Slots status (ok, pfail, fail)
- Cluster size
- Node details (ID, address, flags, slots)

## Export/Import

### 📤 Export Keys to JSON

Export your Redis data for backup or migration:

**Export Features:**
- Export by pattern (e.g., `user:*` or `*` for all)
- Preview export data before downloading
- Download as JSON file with timestamps
- Includes key type, value, and TTL

**Export Format:**
```json
{
  "keys": [
    {
      "key": "user:1",
      "type": "hash",
      "ttl": -1,
      "value": {"name": "John", "age": "30"}
    }
  ],
  "count": 1,
  "exported_at": "2024-01-15T10:30:00Z"
}
```

### 📥 Import Keys from JSON

Restore or migrate Redis data:

**Import Features:**
- Paste JSON directly or upload file
- Overwrite existing keys option
- Import summary (imported/skipped counts)
- Supports all data types

## Bulk Operations

### ⏰ Bulk TTL Update

Set expiration for multiple keys at once:

```bash
# Set TTL for all user keys
Pattern: user:*
TTL: 3600 (1 hour)
```

**Features:**
- Pattern-based key selection
- Set TTL in seconds
- Use 0 or -1 to remove expiration
- Shows count of updated keys

### 🧹 Database Purge

Clear all keys from the current database:
- Requires confirmation
- Uses FLUSHDB command
- Affects only selected database

## Connection Options

### Local Redis

```bash
flash studio redis --url "redis://localhost:6379"
```

### Remote Redis with Authentication

```bash
flash studio redis --url "redis://user:password@redis.example.com:6379"
```

### Redis with TLS/SSL

```bash
flash studio redis --url "rediss://user:pass@redis.example.com:6379"
```

### Specific Database

```bash
flash studio redis --url "redis://localhost:6379/1"
```

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `↑` / `↓` | Navigate command history (CLI) |
| `Tab` | Autocomplete command (CLI) |
| `Ctrl+L` | Clear terminal (CLI) |
| `Ctrl+C` | Cancel current command (CLI) |
| `Enter` | Execute command (CLI) |

## UI Features

### Dark Theme
Redis Studio uses a modern dark theme optimized for long sessions.

### State Persistence
Your preferences are saved across browser sessions:
- Selected database
- Active tab
- Search patterns

### Responsive Design
Works on desktop and tablet devices with adaptive layouts.

## Tips & Best Practices

### Searching Keys Efficiently

Use SCAN instead of KEYS for large databases:
```
redis> SCAN 0 MATCH user:* COUNT 100
```

### Monitoring Memory

Regular memory analysis helps identify:
- Large keys consuming resources
- Memory fragmentation issues
- Keys without expiration (potential memory leaks)

### Lua Script Caching

For frequently-used scripts:
1. Load the script once with "Load Script"
2. Save the returned SHA
3. Execute using EVALSHA for better performance

### Security Best Practices

- Use ACL to restrict user permissions
- Monitor ACL log for unauthorized access attempts
- Set appropriate TTLs to prevent memory issues
- Use TLS for remote connections
