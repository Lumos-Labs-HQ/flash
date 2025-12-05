# FlashORM - TypeScript/JavaScript Usage Guide

A comprehensive guide to using FlashORM with Node.js, TypeScript, and JavaScript projects.

## Table of Contents
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Schema Definition](#schema-definition)
- [Migrations](#migrations)
- [Code Generation](#code-generation)
- [Working with Generated Code](#working-with-generated-code)
- [Advanced Usage](#advanced-usage)
- [Best Practices](#best-practices)

---

## Installation

### Install FlashORM CLI

```bash
npm install -g flashorm
```

Or use npx:
```bash
npx flashorm --version
```

Verify installation:
```bash
flash --version
```

---

## Quick Start

### 1. Create a New Project

```bash
mkdir myproject && cd myproject
npm init -y
npm install typescript @types/node ts-node --save-dev
```

### 2. Initialize FlashORM

```bash
# For PostgreSQL
flash init --postgresql

# For MySQL
flash init --mysql

# For SQLite
flash init --sqlite
```

### 3. Set Database URL

Create a `.env` file:
```env
# PostgreSQL
DATABASE_URL=postgres://user:password@localhost:5432/mydb

# MySQL
DATABASE_URL=user:password@tcp(localhost:3306)/mydb

# SQLite
DATABASE_URL=sqlite://./data.db
```

### 4. Define Your Schema

Edit `db/schema/schema.sql`:
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 5. Create and Apply Migration

```bash
flash migrate "initial schema"
flash apply
```

### 6. Generate TypeScript Code

```bash
flash gen
```

---

## Configuration

### flash.config.json

```json
{
  "version": "2",
  "schema_dir": "db/schema",
  "queries": "db/queries/",
  "migrations_path": "db/migrations",
  "export_path": "db/export",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "js": {
      "enabled": true,
      "output": "flash_gen",
      "typescript": true
    }
  }
}
```

### TypeScript Configuration

Add to `tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "commonjs",
    "strict": true,
    "esModuleInterop": true,
    "outDir": "./dist",
    "rootDir": "./src"
  },
  "include": ["src/**/*", "flash_gen/**/*"]
}
```

---

## Schema Definition

### Table Definition

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Indexes

```sql
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role_active ON users(role, is_active);
CREATE UNIQUE INDEX idx_users_email_unique ON users(email);
```

### Foreign Keys

```sql
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    tags TEXT[]
);
```

---

## Migrations

### Create Migration

```bash
flash migrate "add phone to users"
```

Creates `db/migrations/20251204123456_add_phone_to_users.sql`:

```sql
-- Migration: add_phone_to_users
-- Created: 2025-12-04T12:34:56Z

-- +migrate Up
ALTER TABLE users ADD COLUMN phone VARCHAR(20);

-- +migrate Down
ALTER TABLE users DROP COLUMN phone;
```

### Apply Migrations

```bash
# Apply all pending migrations
flash apply

# Apply with force (skip confirmation)
flash apply --force
```

### Check Status

```bash
flash status
```

### Rollback Migration

```bash
flash down
```

---

## Code Generation

### Generate TypeScript Types

```bash
flash gen
```

Generated files in `flash_gen/`:

```
flash_gen/
├── index.ts       # Main exports
├── types.ts       # TypeScript interfaces
├── database.ts    # Database connection
└── users.ts       # User model
```

### Generated Types

```typescript
// flash_gen/types.ts

export interface User {
    id: number;
    name: string;
    email: string;
    role: string;
    isActive: boolean;
    metadata: Record<string, any> | null;
    createdAt: Date;
    updatedAt: Date;
}

export interface Post {
    id: number;
    userId: number;
    title: string;
    content: string | null;
    tags: string[] | null;
    createdAt: Date;
}

export interface CreateUserInput {
    name: string;
    email: string;
    role?: string;
    isActive?: boolean;
    metadata?: Record<string, any>;
}

export interface UpdateUserInput {
    name?: string;
    email?: string;
    role?: string;
    isActive?: boolean;
    metadata?: Record<string, any>;
}
```

---

## Working with Generated Code

### Installation of Database Driver

```bash
# PostgreSQL
npm install pg @types/pg

# MySQL
npm install mysql2

# SQLite
npm install better-sqlite3 @types/better-sqlite3
```

### Basic Usage (PostgreSQL)

```typescript
// src/index.ts
import { Pool } from 'pg';
import { FlashDB, User, CreateUserInput } from '../flash_gen';

async function main() {
    // Create database connection
    const pool = new Pool({
        connectionString: process.env.DATABASE_URL
    });

    const db = new FlashDB(pool);

    // Create a user
    const newUser: CreateUserInput = {
        name: 'John Doe',
        email: 'john@example.com',
        role: 'admin'
    };
    
    const user = await db.users.create(newUser);
    console.log('Created user:', user);

    // Get all users
    const users = await db.users.findMany();
    console.log('All users:', users);

    // Find user by ID
    const foundUser = await db.users.findById(user.id);
    console.log('Found user:', foundUser);

    // Update user
    const updatedUser = await db.users.update(user.id, {
        name: 'Jane Doe'
    });
    console.log('Updated user:', updatedUser);

    // Delete user
    await db.users.delete(user.id);
    console.log('User deleted');

    await pool.end();
}

main().catch(console.error);
```

### Query Builder

```typescript
// Complex queries with type safety
const activeAdmins = await db.users.findMany({
    where: {
        role: 'admin',
        isActive: true
    },
    orderBy: {
        createdAt: 'desc'
    },
    limit: 10
});

// With relations
const usersWithPosts = await db.users.findMany({
    include: {
        posts: true
    }
});

// Raw query with type
const result = await db.query<User>(
    'SELECT * FROM users WHERE email = $1',
    ['john@example.com']
);
```

### Transactions

```typescript
import { FlashDB } from '../flash_gen';

async function transferCredits(
    db: FlashDB,
    fromUserId: number,
    toUserId: number,
    amount: number
) {
    await db.transaction(async (tx) => {
        // Deduct from sender
        await tx.users.update(fromUserId, {
            credits: { decrement: amount }
        });
        
        // Add to receiver
        await tx.users.update(toUserId, {
            credits: { increment: amount }
        });
        
        // Log transaction
        await tx.transactions.create({
            fromUserId,
            toUserId,
            amount,
            type: 'transfer'
        });
    });
}
```

---

## Advanced Usage

### Pull Schema from Existing Database

```bash
flash pull
```

### Export Database

```bash
# Export to JSON
flash export --json

# Export to CSV
flash export --csv

# Export to SQLite file
flash export --sqlite
```

### Raw SQL Execution

```bash
# Execute SQL command
flash raw "SELECT COUNT(*) FROM users;"

# Execute SQL file
flash raw ./scripts/seed.sql
```

### Studio (Visual Editor)

```bash
flash studio
```

Opens web interface at `http://localhost:8080`

---

## Express.js Integration

### Setup

```typescript
// src/app.ts
import express from 'express';
import { Pool } from 'pg';
import { FlashDB } from '../flash_gen';

const app = express();
app.use(express.json());

const pool = new Pool({
    connectionString: process.env.DATABASE_URL
});

const db = new FlashDB(pool);

// Middleware to attach db to request
app.use((req, res, next) => {
    req.db = db;
    next();
});

// Routes
app.get('/users', async (req, res) => {
    const users = await req.db.users.findMany();
    res.json(users);
});

app.get('/users/:id', async (req, res) => {
    const user = await req.db.users.findById(parseInt(req.params.id));
    if (!user) {
        return res.status(404).json({ error: 'User not found' });
    }
    res.json(user);
});

app.post('/users', async (req, res) => {
    try {
        const user = await req.db.users.create(req.body);
        res.status(201).json(user);
    } catch (error) {
        res.status(400).json({ error: error.message });
    }
});

app.put('/users/:id', async (req, res) => {
    const user = await req.db.users.update(
        parseInt(req.params.id),
        req.body
    );
    res.json(user);
});

app.delete('/users/:id', async (req, res) => {
    await req.db.users.delete(parseInt(req.params.id));
    res.status(204).send();
});

app.listen(3000, () => {
    console.log('Server running on port 3000');
});
```

### Type Augmentation

```typescript
// src/types/express.d.ts
import { FlashDB } from '../../flash_gen';

declare global {
    namespace Express {
        interface Request {
            db: FlashDB;
        }
    }
}
```

---

## Next.js Integration

### API Routes

```typescript
// pages/api/users/index.ts
import { NextApiRequest, NextApiResponse } from 'next';
import { Pool } from 'pg';
import { FlashDB } from '../../../flash_gen';

const pool = new Pool({
    connectionString: process.env.DATABASE_URL
});

const db = new FlashDB(pool);

export default async function handler(
    req: NextApiRequest,
    res: NextApiResponse
) {
    switch (req.method) {
        case 'GET':
            const users = await db.users.findMany();
            return res.json(users);
            
        case 'POST':
            const user = await db.users.create(req.body);
            return res.status(201).json(user);
            
        default:
            res.setHeader('Allow', ['GET', 'POST']);
            return res.status(405).end(`Method ${req.method} Not Allowed`);
    }
}
```

---

## Best Practices

### 1. Use Environment Variables

```typescript
// config/database.ts
import { Pool } from 'pg';

const pool = new Pool({
    connectionString: process.env.DATABASE_URL,
    max: 20,
    idleTimeoutMillis: 30000,
    connectionTimeoutMillis: 2000,
});

export default pool;
```

### 2. Handle Errors Gracefully

```typescript
import { FlashDB, DatabaseError } from '../flash_gen';

try {
    await db.users.create({ email: 'duplicate@email.com' });
} catch (error) {
    if (error instanceof DatabaseError) {
        if (error.code === '23505') {
            // Unique constraint violation
            throw new Error('Email already exists');
        }
    }
    throw error;
}
```

### 3. Use Type Guards

```typescript
import { User, isUser } from '../flash_gen';

function processData(data: unknown): User {
    if (!isUser(data)) {
        throw new Error('Invalid user data');
    }
    return data;
}
```

### 4. Connection Pooling

```typescript
// Singleton pattern for database connection
let db: FlashDB | null = null;

export function getDB(): FlashDB {
    if (!db) {
        const pool = new Pool({
            connectionString: process.env.DATABASE_URL,
            max: 10
        });
        db = new FlashDB(pool);
    }
    return db;
}
```

### 5. Validation with Zod

```typescript
import { z } from 'zod';
import { CreateUserInput } from '../flash_gen';

const CreateUserSchema = z.object({
    name: z.string().min(1).max(100),
    email: z.string().email(),
    role: z.enum(['admin', 'user', 'guest']).optional()
});

function validateCreateUser(input: unknown): CreateUserInput {
    return CreateUserSchema.parse(input);
}
```

---

## Troubleshooting

### Connection Issues

```bash
# Check database URL
echo $DATABASE_URL

# Test connection
flash status
```

### Type Errors

Regenerate types after schema changes:
```bash
flash gen
```

### Missing Types

Ensure TypeScript includes generated files:
```json
{
  "include": ["src/**/*", "flash_gen/**/*"]
}
```

---

## Type Mapping

| PostgreSQL | MySQL | SQLite | TypeScript |
|------------|-------|--------|------------|
| SERIAL | INT AUTO_INCREMENT | INTEGER | number |
| BIGSERIAL | BIGINT AUTO_INCREMENT | INTEGER | bigint |
| VARCHAR | VARCHAR | TEXT | string |
| TEXT | TEXT | TEXT | string |
| BOOLEAN | TINYINT(1) | INTEGER | boolean |
| TIMESTAMP | DATETIME | TEXT | Date |
| JSONB | JSON | TEXT | Record<string, any> |
| TEXT[] | - | - | string[] |

---

## Resources

- [FlashORM GitHub](https://github.com/Lumos-Labs-HQ/flash)
- [npm Package](https://www.npmjs.com/package/flashorm)
- [Release Notes](../RELEASE_NOTES.md)
- [Contributing Guide](CONTRIBUTING.md)
