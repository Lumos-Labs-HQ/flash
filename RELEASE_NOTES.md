# FlashORM v2.0.0 Release Notes

## 🎉 Major Release: FlashORM Studio, Node.js/TypeScript Support & Massive Performance Improvements

We're excited to announce FlashORM v2.0.0, a groundbreaking release that introduces **FlashORM Studio** (visual database editor), first-class Node.js/TypeScript support, raw SQL command execution, and significant performance improvements!

## 🚀 What's New

### 🎨 FlashORM Studio - Visual Database Editor

**The biggest feature in v2.0.0!** A powerful web-based database management interface with three interactive pages.

**Features:**
- ✅ **Data Browser** - View, edit, and manage table data with real-time updates
- ✅ **SQL Editor** - Execute SQL queries with CodeMirror syntax highlighting
- ✅ **Schema Visualization** - Interactive database diagram with React + ReactFlow
- ✅ Inline cell editing with double-click
- ✅ Pagination and search across all tables
- ✅ CSV export from SQL query results
- ✅ Auto-opens in browser on launch
- ✅ Similar to Prisma Studio but faster and lighter

**Usage:**
```bash
# Launch studio (auto-detects config)
FlashORM studio

# Custom port
FlashORM studio --port 3000

# Connect to any database directly
FlashORM studio --db "postgres://user:pass@localhost:5432/mydb"
```

**Studio Pages:**

1. **Data Browser (`/`)** - Supabase-like table editor
   - Real-time inline editing
   - Batch save changes
   - Add/delete rows with modals
   - Foreign key relationship hints
   - Pagination (50 rows per page)

2. **SQL Editor (`/sql`)** - Execute custom queries
   - CodeMirror with Material Darker theme
   - Ctrl+Enter to execute
   - Split-pane resizable interface
   - CSV export from results
   - Query history

3. **Schema Visualization (`/schema`)** - Interactive ER diagram
   - React + ReactFlow rendering
   - Automatic layout with Dagre algorithm
   - Drag-and-drop tables
   - Zoom and pan controls
   - Foreign key arrows
   - MiniMap for navigation

**Performance:**
- 95% fewer database queries with batch optimization
- Connection pooling for PostgreSQL
- Prepared statement caching
- Single query for all table row counts

### ⚡ Raw SQL Command

Execute raw SQL files or queries directly against your database!

**Features:**
- ✅ Execute SQL files
- ✅ Execute inline SQL queries
- ✅ Auto-detection (file vs query)
- ✅ Formatted table output for SELECT queries
- ✅ Transaction support for DML/DDL statements
- ✅ Multi-statement execution

**Usage:**
```bash
# Execute SQL file
FlashORM raw script.sql
FlashORM raw migrations/seed.sql

# Execute inline query
FlashORM raw -q "SELECT * FROM users WHERE active = true"
FlashORM raw "SELECT COUNT(*) FROM orders"

# Force file mode
FlashORM raw --file queries/complex_query.sql
```

**Output Formatting:**
- SELECT queries: Beautiful table output with columns and rows
- DML/DDL statements: Success confirmation with execution count
- Errors: Clear error messages with line numbers

**Use Cases:**
- Quick database queries without writing code
- Testing SQL before adding to migrations
- Running seed scripts
- Database maintenance tasks
- Ad-hoc data analysis

### 🟢 Node.js/TypeScript Support

FlashORM now generates fully type-safe JavaScript/TypeScript code for Node.js projects!

**Features:**
- ✅ Automatic project detection (detects `package.json`)
- ✅ Type-safe query methods with TypeScript definitions
- ✅ Full IntelliSense support in VS Code
- ✅ Zero runtime overhead
- ✅ PostgreSQL, MySQL, and SQLite support

**Example:**
```typescript
import { New } from './FlashORM_gen/database';
import { Pool } from 'pg';

const db = New(new Pool({ connectionString: DATABASE_URL }));

// Fully type-safe!
const user = await db.createUser('Alice', 'alice@example.com');
const users = await db.listUsers();
```

**Generated Types:**
```typescript
export interface Users {
  id: number | null;
  name: string;
  email: string;
  created_at: Date;
  updated_at: Date;
}

export class Queries {
  getUser(id: number): Promise<Users | null>;
  createUser(name: string, email: string): Promise<Users | null>;
  listUsers(): Promise<Users[]>;
}
```

### 📦 NPM Package Distribution

Install FlashORM via NPM for seamless integration with Node.js projects:

```bash
npm install -g FlashORM-orm
```

**Features:**
- ✅ Automatic binary download from GitHub releases
- ✅ Cross-platform support (Linux, macOS, Windows)
- ✅ Multi-architecture (x64, ARM64)
- ✅ Small package size (~3KB, downloads binary on install)
- ✅ Works with npm, yarn, pnpm, and bun

### ⚡ Performance Improvements

FlashORM v2.0.0 now significantly outperforms popular ORMs:

| Operation | FlashORM v2.0 | Drizzle | Prisma | Improvement |
|-----------|------------|---------|--------|-------------|
| Insert 1000 Users | **158ms** | 224ms | 230ms | **1.4x faster** |
| Insert 10 Cat + 5K Posts + 15K Comments | **2410ms** | 3028ms | 3977ms | **1.3x faster** |
| Complex Query x500 | **4071ms** | 12500ms | 56322ms | **3x-14x faster** |
| Mixed Workload x1000 | **186ms** | 1174ms | 10863ms | **6x-58x faster** |
| Stress Test x2000 | **122ms** | 160ms | 223ms | **1.3x-1.8x faster** |
| **TOTAL** | **6947ms** | **17149ms** | **71551ms** | **2.5x-10x faster** |

**Performance Optimizations:**

1. **Studio Optimizations**
   - Batch query optimization (95% fewer queries)
   - Single query for all table row counts
   - Connection pooling with pgxpool
   - Prepared statement caching
   - Efficient pagination

2. **Code Generation**
   - Zero runtime overhead
   - Compiled queries at generation time
   - Type-safe without reflection
   - Minimal memory allocation

3. **Database Adapters**
   - PostgreSQL: pgxpool with Supabase/PgBouncer compatibility
   - MySQL: Optimized connection pooling
   - SQLite: In-memory caching for metadata
   - Transaction batching for migrations

4. **Query Execution**
   - Prepared statement caching
   - Batch operations for bulk data
   - Index-aware query generation
   - Streaming for large exports

### 🎨 PostgreSQL ENUM Support

Full support for PostgreSQL ENUM types with automatic TypeScript type generation:

**Schema:**
```sql
CREATE TYPE user_role AS ENUM ('admin', 'user', 'guest');

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    role user_role NOT NULL DEFAULT 'user'
);
```

**Generated TypeScript:**
```typescript
export type UserRole = 'admin' | 'user' | 'guest';

export interface Users {
  id: number | null;
  role: UserRole;
}
```

### 🛡️ Enhanced Conflict Detection

Improved migration conflict detection and resolution:

- ✅ Detects table conflicts
- ✅ Detects column conflicts
- ✅ Detects constraint violations
- ✅ Interactive resolution prompts
- ✅ Automatic export before destructive operations
- ✅ Safe database reset with full migration replay

**Example:**
```bash
⚠️  Migration conflicts detected:
  - Table 'users' already exists
  - Column 'email' conflicts with existing column

Reset database to resolve conflicts? (y/n): y
Create export before applying? (y/n): y
📦 Creating export...
✅ Export created successfully
```

### 🔍 Schema Introspection Improvements

Enhanced `FlashORM pull` command with better schema extraction:

- ✅ Improved foreign key detection
- ✅ Better constraint handling
- ✅ ENUM type extraction
- ✅ Index preservation
- ✅ Backup option before overwrite

```bash
FlashORM pull --backup
```

### 📤 Export System Enhancements

Improved export functionality with better data handling:

- ✅ Faster JSON export
- ✅ Improved CSV formatting
- ✅ Better SQLite compatibility
- ✅ Metadata preservation
- ✅ Large dataset handling

## 🔧 Improvements

### FlashORM Studio
- **Web Interface**: Full-featured visual database editor
- **Three Interactive Pages**: Data browser, SQL editor, and schema visualization
- **React Integration**: Modern React 18.2.0 for schema diagrams
- **Embedded Assets**: All frontend code embedded in single binary
- **Auto-Browser**: Automatically opens in default browser
- **Connection Pooling**: Optimized for high-performance queries

### Raw SQL Execution
- **File Support**: Execute .sql files directly
- **Inline Queries**: Run queries from command line
- **Auto-Detection**: Smart detection of file vs query
- **Formatted Output**: Beautiful table formatting for results
- **Multi-Statement**: Execute multiple SQL statements in one file

### Code Generation
- **JavaScript/TypeScript Generator**: New `jsgen` package for Node.js code generation
- **Type Safety**: Full TypeScript type definitions for all queries
- **Query Optimization**: Prepared statement caching for better performance
- **Error Handling**: Better error messages with context

### CLI Experience
- **Better Output**: Improved colored output and formatting
- **Progress Indicators**: Clear progress for long-running operations
- **Error Messages**: More helpful error messages with solutions
- **Interactive Prompts**: Better user interaction for confirmations

### Configuration
- **Auto-Detection**: Automatically detects Node.js projects
- **Flexible Config**: Support for both Go and JS code generation
- **Environment Variables**: Better .env file handling
- **Gen Config**: New `gen.js` and `gen.go` configuration sections

### Documentation
- **Complete Examples**: Full TypeScript and JavaScript examples
- **API Documentation**: Comprehensive API docs for generated code
- **Studio Guide**: Complete guide for using FlashORM Studio
- **Technology Stack**: Detailed documentation of all dependencies
- **Migration Guide**: Guide for migrating from other ORMs
- **Performance Guide**: Tips for optimal performance

## 🐛 Bug Fixes

- Fixed transaction rollback in MySQL adapter
- Fixed ENUM type parsing in PostgreSQL
- Fixed schema diff generation for complex foreign keys
- Fixed export path creation on Windows
- Fixed binary download on ARM64 systems
- Fixed postinstall script for Bun users
- Fixed migration checksum calculation
- Fixed concurrent migration detection
- Fixed connection pooling on Supabase
- Fixed raw command file detection on Windows

## 📦 Installation

### NPM (New!)
```bash
npm install -g FlashORM-orm
```

### Go
```bash
go install github.com/Lumos-Labs-HQ/flash@latest
```

### Binary Download
Download from [GitHub Releases](https://github.com/Lumos-Labs-HQ/flash/releases/tag/v1.7.0)

## 📚 Documentation

- [Main Documentation](https://github.com/Lumos-Labs-HQ/flash)
- [NPM Package README](https://www.npmjs.com/package/FlashORM-orm)
- [TypeScript Examples](https://github.com/Lumos-Labs-HQ/flash/tree/main/example/ts)
- [How It Works](https://github.com/Lumos-Labs-HQ/flash/blob/main/docs/HOW_IT_WORKS.md)
- [Technology Stack](https://github.com/Lumos-Labs-HQ/flash/blob/main/docs/TECHNOLOGY_STACK.md)
- [Contributing Guide](https://github.com/Lumos-Labs-HQ/flash/blob/main/docs/CONTRIBUTING.md)

## 🙏 Acknowledgments

Special thanks to:
- All contributors who helped with testing and feedback
- Prisma Studio for UI inspiration

## 🔮 What's Next (v2.1.0)

- 🎯 Studio: Table relationship editor
- � Studio: Advanced search and filters
- �🐍 Python code generation support
- 🦀 Rust code generation support

## 📝 Full Changelog

### Added
- **FlashORM Studio**: Complete visual database editor with 3 pages
  - Data browser with inline editing (`internal/studio`)
  - SQL editor with CodeMirror syntax highlighting
  - Schema visualization with React + ReactFlow
  - Fiber v2.52.9 backend with embedded templates
  - RESTful API endpoints for all operations
- **Raw SQL Command**: Execute SQL files or inline queries (`cmd/raw.go`)
- Node.js/TypeScript code generation (`internal/jsgen`)
- NPM package distribution (`npm/`)
- PostgreSQL ENUM support with TypeScript types
- Enhanced conflict detection and resolution
- Automatic project type detection
- Performance benchmarks and comparisons
- Prepared statement caching
- GitHub Actions workflow for NPM releases
- Comprehensive TypeScript examples
- Programmatic API for Node.js
- Studio performance optimizations (batch queries, connection pooling)
- React 18.2.0 integration for schema diagrams
- ReactFlow 12.8.4 with Dagre layout algorithm
- CodeMirror 5.65.2 for SQL editing
- Iconify for modern icon system
- Embedded file system for single binary distribution

### Changed
- Improved schema introspection algorithm
- Better error messages with context
- Enhanced export system performance
- Updated documentation with TypeScript examples
- Improved CLI output formatting
- Studio uses connection pooling for better performance
- ReactFlow-based schema visualization replaces canvas-based approach
- Embedded assets for zero external dependencies

### Fixed
- Transaction rollback in MySQL adapter
- ENUM type parsing in PostgreSQL
- Schema diff for complex foreign keys
- Export path creation on Windows
- Binary download on ARM64
- Postinstall script for Bun
- Migration checksum calculation
- Concurrent migration detection
- Studio connection pooling on Supabase/PgBouncer
- ReactFlow layout for large schemas (100+ tables)
- CodeMirror Material Darker theme loading
- Raw command file detection on Windows paths

### Performance
- 2.5x faster than Drizzle on average
- 10x faster than Prisma on average
- Optimized query execution with prepared statements
- Reduced memory usage in code generation
- Faster migration application with transaction batching
- Studio: 95% fewer database queries with batch optimization
- Studio: Single query for all table row counts
- Studio: Connection pooling with pgxpool
- Raw command: Efficient streaming for large result sets

## 🐛 Known Issues

- Bun users need to run `bun pm trust FlashORM-orm` after installation
- Windows ARM64 support is experimental
- MySQL ENUM support is limited (use VARCHAR with CHECK constraint)
- Studio: Large schemas (200+ tables) may have slow initial load
- Studio: No authentication (local development tool only)

## 💬 Feedback

We'd love to hear your feedback! Please:
- 🐛 [Report bugs](https://github.com/Lumos-Labs-HQ/flash/issues)
- 💡 [Request features](https://github.com/Lumos-Labs-HQ/flash/issues)
- ⭐ [Star the repo](https://github.com/Lumos-Labs-HQ/flash)
- 🐦 Share on social media

---

**Download:** [v2.0.0 Release](https://github.com/Lumos-Labs-HQ/flash/releases/tag/v2.0.0)

**NPM:** `npm install -g FlashORM-orm`

**Go:** `go install github.com/Lumos-Labs-HQ/flash@latest`

**Try FlashORM Studio:** `FlashORM studio`
