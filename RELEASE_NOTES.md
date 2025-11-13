# FlashORM v2.1.0 Release Notes

## 🎉 Major Features

### 🐍 Python Support

**Full Python Async Support:**
- ✅ **Python Code Generation** - Generate type-safe Python code from SQL queries
- ✅ **asyncpg Support** - First-class PostgreSQL async support with asyncpg
- ✅ **aiomysql Support** - MySQL async operations with aiomysql
- ✅ **aiosqlite Support** - SQLite async support with aiosqlite
- ✅ **Type Hints** - Full type annotations with dataclasses for models
- ✅ **Query Classes** - Generated Queries class with all your SQL operations
- ✅ **Database Factory** - Simple `new(db)` function to create database clients

**Usage Example:**
```python
import asyncio
import asyncpg
from flash_gen.database import new

async def main():
    pool = await asyncpg.create_pool('postgresql://...')
    db = new(pool)
    
    user = await db.create_user('Alice', 'alice@example.com')
    users = await db.get_all_users()
    
    await pool.close()

asyncio.run(main())
```

**Configuration:**
```json
{
  "gen": {
    "python": {
      "enabled": true,
    }
  }
}
```

### 📊 Enhanced FlashORM Studio

**New Studio Features:**
- ✅ **Add Tables** - Create new tables directly from the Studio UI
- ✅ **Edit Tables** - Modify existing table structures with visual editor
- ✅ **Auto-Migration Creation** - Automatically generates migration files from schema changes
- ✅ **Improved UI/UX** - Better visibility with enhanced background dots and contrast
- ✅ **Data Management** - Insert, update, and delete records with spreadsheet-like interface
- ✅ **Relationship Visualization** - Interactive schema diagram showing all foreign keys

**Schema Visualizer:**

![Schema Visualizer](img/s1.png)
*Hover over connection lines or tables to see context menu with quick actions*

**Table Editor:**

![Table Editor](img/s2.png)
*Visual table editor for modifying schema with auto-migration generation*

**Studio Usage:**
```bash
# Auto-detect from config
flash studio

# Or specify database directly
flash studio --db "postgresql://user:pass@localhost/db"

# Custom port
flash studio --port 3000
```

### ⚡ Performance Optimizations

**Memory & Speed Improvements:**
- ✅ **Reduced RAM Usage** - Optimized memory allocation for large-scale operations
- ✅ **Efficient Code Generation** - Reduced memory footprint during code generation
- ✅ **Buffer Pre-allocation** - Smart buffer sizing reduces allocations by 40%
- ✅ **Batch Operations** - Optimized batch inserts and updates for better throughput

### 🐛 Bug Fixes & Improvements

**SQL Parser Enhancements:**
- ✅ **Fixed CAST AS Alias Extraction** - Now correctly uses the last `AS` occurrence for aliases, not the first one inside `CAST()` expressions
- ✅ **UNION Query Support** - Proper detection and handling of UNION queries (queries starting with `(` containing SELECT)
- ✅ **Column Validation** - Skips column validation for UNION queries to prevent false errors
- ✅ **Nested SELECT Handling** - Improved extraction of SELECT columns from nested/parenthesized queries

**Validation & Error Messages:**
- ✅ **Duplicate Parameter Detection** - Prevents duplicate parameter names in generated code
- ✅ **CASE/CAST Boundary Detection** - Better parsing of SQL keywords within expressions
- ✅ **Improved Error Messages** - Clearer validation errors with helpful context


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
