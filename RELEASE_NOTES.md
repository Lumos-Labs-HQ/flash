# FlashORM v2.1.1 Release Notes

## 🎉 New Features

### 🐍 Python Async/Sync Code Generation

**Flexible Python Generation:**
- ✅ **Async Mode (Default)** - Generate async/await Python code for high-performance applications
- ✅ **Sync Mode** - Generate synchronous Python code for simpler applications or legacy codebases
- ✅ **Driver-Agnostic MySQL** - MySQL code now works with ANY Python MySQL driver (pymysql, aiomysql, mysql-connector-python, etc.)
- ✅ **No Library Dependencies** - Generated code adapts to your chosen driver automatically

**Configuration:**
```json
{
  "gen": {
    "python": {
      "enabled": true,
      "async": false  // true = async (default), false = sync
    }
  }
}
```

**Async Example (async: true):**
```python
import asyncio
import aiomysql
from flash_gen.database import new

async def main():
    pool = await aiomysql.create_pool(
        host='localhost',
        user='root',
        password='password',
        db='mydb'
    )
    db = new(pool)
    
    # Async operations with await
    user = await db.create_user('Alice', 'alice@example.com')
    users = await db.get_all_users()
    count = await db.get_user_count()  # Single-column queries return values directly
    
    pool.close()
    await pool.wait_closed()

asyncio.run(main())
```

**Sync Example (async: false):**
```python
import pymysql
from flash_gen.database import new

def main():
    conn = pymysql.connect(
        host='localhost',
        user='root',
        password='password',
        db='mydb'
    )
    db = new(conn)
    
    # Synchronous operations - no await needed
    user = db.create_user('Alice', 'alice@example.com')
    users = db.get_all_users()
    count = db.get_user_count()
    
    conn.close()

main()
```

**Driver Compatibility:**
```python
# Works with aiomysql (async)
import aiomysql
pool = await aiomysql.create_pool(...)
db = new(pool)

# Works with PyMySQL (sync)
import pymysql
conn = pymysql.connect(...)
db = new(conn)

# Works with mysql-connector-python (sync)
import mysql.connector
conn = mysql.connector.connect(...)
db = new(conn)

# Works with any driver that supports cursor() and execute()
```

**Key Improvements:**
- 🔧 **Flexible** - Switch between async/sync without regenerating your SQL queries
- 🎯 **Smart Type Detection** - Single-column queries return primitives, multi-column returns typed dataclasses
- 📦 **Universal MySQL Support** - Works with any MySQL driver's cursor interface

## 📦 Installation

### NPM
```bash
npm install -g flashorm
```

### Go
```bash
go install github.com/Lumos-Labs-HQ/flash
```

### Python
```bash
pip install flashorm
```

### Download
Download from [GitHub Releases](https://github.com/Lumos-Labs-HQ/flash/releases/tag/v2.1.0)
Download from [NPM](https://www.npmjs.com/package/flashorm)
Download from [PYPI](https://pypi.org/project/flashorm/2.1.0/)

## 📚 Documentation

- [Main Documentation](https://github.com/Lumos-Labs-HQ/flash)
- [Go Examples](https://github.com/Lumos-Labs-HQ/flash/tree/main/example/go)
- [TypeScript Examples](https://github.com/Lumos-Labs-HQ/flash/tree/main/example/ts)
- [Python Examples](https://github.com/Lumos-Labs-HQ/flash/tree/main/example/python)
- [Studio Guide](https://github.com/Lumos-Labs-HQ/flash#-studio-visual-database-editor)
- [Technology Stack](https://github.com/Lumos-Labs-HQ/flash/blob/main/docs/TECHNOLOGY_STACK.md)

## 💬 Feedback

- 🐛 [Report bugs](https://github.com/Lumos-Labs-HQ/flash/issues)
- 💡 [Request features](https://github.com/Lumos-Labs-HQ/flash/issues)
- ⭐ [Star the repo](https://github.com/Lumos-Labs-HQ/flash)

---
