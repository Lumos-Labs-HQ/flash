# Graft v1.7.0 Release Notes

## 🎉 Major Release: Node.js/TypeScript Support & Performance Improvements

We're excited to announce Graft v1.7.0, a major release that brings first-class Node.js/TypeScript support, significant performance improvements, and enhanced developer experience!

## 🚀 What's New

### 🟢 Node.js/TypeScript Support

Graft now generates fully type-safe JavaScript/TypeScript code for Node.js projects!

**Features:**
- ✅ Automatic project detection (detects `package.json`)
- ✅ Type-safe query methods with TypeScript definitions
- ✅ Full IntelliSense support in VS Code
- ✅ Zero runtime overhead
- ✅ PostgreSQL, MySQL, and SQLite support

**Example:**
```typescript
import { New } from './graft_gen/database';
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

Install Graft via NPM for seamless integration with Node.js projects:

```bash
npm install -g graft-orm
```

**Features:**
- ✅ Automatic binary download from GitHub releases
- ✅ Cross-platform support (Linux, macOS, Windows)
- ✅ Multi-architecture (x64, ARM64)
- ✅ Small package size (~3KB, downloads binary on install)
- ✅ Works with npm, yarn, pnpm, and bun

### ⚡ Performance Improvements

Graft now significantly outperforms popular ORMs:

| Operation | Graft | Drizzle | Prisma | Improvement |
|-----------|-------|---------|--------|-------------|
| Insert 1000 Users | **158ms** | 224ms | 230ms | **1.4x faster** |
| Insert 10 Cat + 5K Posts + 15K Comments | **2410ms** | 3028ms | 3977ms | **1.3x faster** |
| Complex Query x500 | **4071ms** | 12500ms | 56322ms | **3x-14x faster** |
| Mixed Workload x1000 | **186ms** | 1174ms | 10863ms | **6x-58x faster** |
| Stress Test x2000 | **122ms** | 160ms | 223ms | **1.3x-1.8x faster** |
| **TOTAL** | **6947ms** | **17149ms** | **71551ms** | **2.5x-10x faster** |

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

Enhanced `graft pull` command with better schema extraction:

- ✅ Improved foreign key detection
- ✅ Better constraint handling
- ✅ ENUM type extraction
- ✅ Index preservation
- ✅ Backup option before overwrite

```bash
graft pull --backup
```

### 📤 Export System Enhancements

Improved export functionality with better data handling:

- ✅ Faster JSON export
- ✅ Improved CSV formatting
- ✅ Better SQLite compatibility
- ✅ Metadata preservation
- ✅ Large dataset handling

## 🔧 Improvements

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

### Documentation
- **Complete Examples**: Full TypeScript and JavaScript examples
- **API Documentation**: Comprehensive API docs for generated code
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

## 📦 Installation

### NPM (New!)
```bash
npm install -g graft-orm
```

### Go
```bash
go install github.com/Lumos-Labs-HQ/graft@latest
```

### Binary Download
Download from [GitHub Releases](https://github.com/Lumos-Labs-HQ/graft/releases/tag/v1.7.0)

## 🔄 Migration Guide

### From v1.6.x to v1.7.0

No breaking changes! Simply upgrade:

```bash
# NPM
npm install -g graft-orm@latest

# Go
go install github.com/Lumos-Labs-HQ/graft@latest
```

### Enabling JavaScript Code Generation

Update your `graft.config.json`:

```json
{
  "gen": {
    "js": {
      "enabled": true,
      "out": "graft_gen"
    }
  }
}
```

Or run `graft init` in a Node.js project (with `package.json`).

## 📚 Documentation

- [Main Documentation](https://github.com/Lumos-Labs-HQ/graft)
- [NPM Package README](https://www.npmjs.com/package/graft-orm)
- [TypeScript Examples](https://github.com/Lumos-Labs-HQ/graft/tree/main/example/ts)
- [How It Works](https://github.com/Lumos-Labs-HQ/graft/blob/main/docs/HOW_IT_WORKS.md)
- [Contributing Guide](https://github.com/Lumos-Labs-HQ/graft/blob/main/docs/CONTRIBUTING.md)

## 🙏 Acknowledgments

Special thanks to:
- All contributors who helped with testing and feedback
- The Go and Node.js communities
- SQLC project for inspiration

## 🔮 What's Next (v1.8.0)

- 🐍 Python code generation support
- 🦀 Rust code generation support

## 📝 Full Changelog

### Added
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

### Changed
- Improved schema introspection algorithm
- Better error messages with context
- Enhanced export system performance
- Updated documentation with TypeScript examples
- Improved CLI output formatting

### Fixed
- Transaction rollback in MySQL adapter
- ENUM type parsing in PostgreSQL
- Schema diff for complex foreign keys
- Export path creation on Windows
- Binary download on ARM64
- Postinstall script for Bun
- Migration checksum calculation
- Concurrent migration detection

### Performance
- 2.5x faster than Drizzle
- 10x faster than Prisma
- Optimized query execution
- Reduced memory usage
- Faster migration application

## 🐛 Known Issues

- Bun users need to run `bun pm trust graft-orm` after installation
- Windows ARM64 support is experimental
- MySQL ENUM support is limited (use VARCHAR with CHECK constraint)

## 💬 Feedback

We'd love to hear your feedback! Please:
- 🐛 [Report bugs](https://github.com/Lumos-Labs-HQ/graft/issues)
- 💡 [Request features](https://github.com/Lumos-Labs-HQ/graft/issues)
- ⭐ [Star the repo](https://github.com/Lumos-Labs-HQ/graft)
- 🐦 Share on social media

---

**Download:** [v1.7.0 Release](https://github.com/Lumos-Labs-HQ/graft/releases/tag/v1.7.0)

**NPM:** `npm install -g graft-orm`

**Go:** `go install github.com/Lumos-Labs-HQ/graft@latest`
