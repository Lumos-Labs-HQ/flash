---
title: Redis Studio
description: Visual Redis management interface
---

# Redis Studio

FlashORM includes a powerful Redis management interface inspired by Upstash, providing a beautiful and intuitive way to manage your Redis databases.

## Quick Start

```bash
# Start Redis Studio
flash studio --redis "redis://localhost:6379"

# With password
flash studio --redis "redis://:password@localhost:6379"

# Custom port
flash studio --redis "redis://localhost:6379" --port 3000
```

## Features

### 🗂️ Key Browser

- View all keys with type indicators (STRING, LIST, SET, HASH, ZSET)
- Search keys with pattern matching (e.g., `user:*`)
- View key details including TTL
- Create, edit, and delete keys

### 💻 Real CLI Terminal

Full Redis CLI with command history and autocomplete:

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

**CLI Features:**
- Command history with ↑↓ arrow keys
- Tab completion for commands
- Syntax highlighting
- Multi-line command support

### 📊 Statistics Dashboard

- Memory usage and peak memory
- Connected clients count
- Total keys per database
- Commands processed
- Server uptime
- Redis version info

### 🗄️ Database Selector

Switch between Redis databases (db0-db15):

```
redis> SELECT 1
OK
redis> DBSIZE
(integer) 42
```

### ⏰ TTL Management

- View remaining TTL for any key
- Set expiration on keys
- Remove expiration (make persistent)
- Bulk TTL operations

### 🧹 Database Operations

- Purge all keys (FLUSHDB)
- View database statistics
- Monitor memory usage
- Connection management

## Supported Data Types

| Type | View | Edit | Create |
|------|------|------|--------|
| STRING | ✅ | ✅ | ✅ |
| LIST | ✅ | ✅ | ✅ |
| SET | ✅ | ✅ | ✅ |
| HASH | ✅ | ✅ | ✅ |
| ZSET | ✅ | ✅ | ✅ |
| STREAM | ✅ | - | - |

## Connection Options

### Local Redis

```bash
flash studio --redis "redis://localhost:6379"
```

### Remote Redis

```bash
flash studio --redis "redis://user:pass@redis.example.com:6379"
```

### Redis with TLS

```bash
flash studio --redis "rediss://user:pass@redis.example.com:6379"
```

### Redis Cluster

```bash
flash studio --redis "redis://localhost:6379?cluster=true"
```

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `↑` / `↓` | Navigate command history |
| `Tab` | Autocomplete command |
| `Ctrl+L` | Clear terminal |
| `Ctrl+C` | Cancel current command |
| `Enter` | Execute command |

## Tips

### Searching Keys

Use patterns to find keys:
```
redis> KEYS user:*        # All user keys
redis> KEYS *:session:*   # All session keys
redis> SCAN 0 MATCH user:* COUNT 100
```

### Monitoring

Watch commands in real-time:
```
redis> MONITOR
```

### Memory Analysis

```
redis> MEMORY STATS
redis> MEMORY USAGE mykey
redis> INFO memory
```
