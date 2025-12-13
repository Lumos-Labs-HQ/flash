---
title: TypeScript/JavaScript Guide
description: Complete guide to using Flash ORM with TypeScript and JavaScript
---

# Flash ORM - TypeScript/JavaScript Usage Guide

A comprehensive guide to using Flash ORM with Node.js, TypeScript, and JavaScript projects, featuring full type safety and async support.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Schema Definition](#schema-definition)
- [Migrations](#migrations)
- [Code Generation](#code-generation)
- [Working with Generated Code](#working-with-generated-code)
- [Async Support](#async-support)
- [Framework Integration](#framework-integration)
- [Best Practices](#best-practices)

## Installation

### Install Flash ORM CLI

```bash
npm install -g flashorm
```

Or use npx:
```bash
npx flashorm --version
```

## Quick Start

### 1. Create a New Project

```bash
mkdir myproject && cd myproject
npm init -y
npm install typescript @types/node ts-node --save-dev
```

### 2. Initialize Flash ORM

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
DATABASE_URL=mysql://user:password@localhost:3306/mydb

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

## Configuration

### flash.config.json

```json
{
  "version": "2",
  "schema_dir": "db/schema",
  "queries": "db/queries",
  "migrations_path": "db/migrations",
  "export_path": "db/export",
  "database": {
    "provider": "postgresql",
    "url_env": "DATABASE_URL"
  },
  "gen": {
    "js": {
      "enabled": true,
      "out": "flash_gen"
    }
  }
}
```

### TypeScript Configuration

```json
// tsconfig.json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "commonjs",
    "lib": ["ES2020"],
    "outDir": "./dist",
    "rootDir": "./",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true
  },
  "include": [
    "src/**/*",
    "flash_gen/**/*"
  ],
  "exclude": [
    "node_modules",
    "dist"
  ]
}
```

## Schema Definition

### Basic Schema

```sql
-- db/schema/users.sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- db/schema/posts.sql
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT FALSE,
    tags TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Advanced Schema Features

```sql
-- Enums
CREATE TYPE user_role AS ENUM ('admin', 'user', 'moderator');

-- Tables with enums
CREATE TABLE user_profiles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    role user_role DEFAULT 'user',
    bio TEXT,
    avatar_url VARCHAR(500)
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE UNIQUE INDEX idx_user_profiles_user_id ON user_profiles(user_id);

-- JSON columns (PostgreSQL)
CREATE TABLE user_preferences (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    preferences JSONB,
    settings JSON
);
```

## Migrations

### Creating Migrations

```bash
# Interactive migration creation
flash migrate

# Named migration
flash migrate "add user profiles table"

# Auto-generate migration from schema changes
flash migrate "update schema" --auto
```

### Migration Files

Migrations are stored in `db/migrations/` with timestamp prefixes:

```
db/migrations/
├── 20240101120000_initial_schema.up.sql
├── 20240101120000_initial_schema.down.sql
├── 20240102130000_add_user_profiles.up.sql
└── 20240102130000_add_user_profiles.down.sql
```

### Applying Migrations

```bash
# Apply all pending migrations
flash apply

# Apply specific number of migrations
flash apply --count 1

# Dry run (show what would be executed)
flash apply --dry-run
```

## Code Generation

### Generated Files

After running `flash gen`, you'll get:

```
flash_gen/
├── index.js        # Main exports
├── index.d.ts      # TypeScript definitions
├── database.js     # Database connection interface
├── database.d.ts
├── users.js        # User-related queries
├── users.d.ts
├── posts.js        # Post-related queries
├── posts.d.ts
└── ...
```

### Generated TypeScript Types

```typescript
// flash_gen/index.d.ts
export interface User {
  id: number;
  name: string;
  email: string;
  is_active: boolean;
  created_at: Date;
  updated_at: Date;
}

export interface Post {
  id: number;
  user_id: number;
  title: string;
  content: string | null;
  published: boolean;
  created_at: Date;
}

export interface CreateUserParams {
  name: string;
  email: string;
}

export interface UpdateUserParams {
  name?: string;
  email?: string;
}

export interface CreatePostParams {
  user_id: number;
  title: string;
  content?: string;
}
```

## Working with Generated Code

### Database Connection

```typescript
// src/database.ts
import { Pool } from 'pg';
import { New } from '../flash_gen/database';

const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
});

export const db = New(pool);

// Graceful shutdown
process.on('SIGINT', async () => {
  await pool.end();
  process.exit(0);
});

export { pool };
```

### Using Generated Queries

```typescript
// src/index.ts
import { db } from './database';

async function main() {
  try {
    // Create a user
    const userId = await db.createUser({
      name: 'John Doe',
      email: 'john@example.com',
    });
    console.log('Created user with ID:', userId);

    // Get user by ID
    const user = await db.getUserById(userId);
    console.log('User:', user);

    // Create a post
    const postId = await db.createPost({
      userId: userId,
      title: 'My First Post',
      content: 'This is the content of my first post.',
    });

    // Get posts by user
    const posts = await db.getPostsByUserId(userId);
    console.log('Posts:', posts);

    // Update user
    await db.updateUser(userId, {
      name: 'John Smith',
    });

    // Delete post
    await db.deletePost(postId);

  } catch (error) {
    console.error('Error:', error);
  }
}

main();
```

### Query Files

Create query files in `db/queries/`:

```sql
-- db/queries/users.sql
-- name: GetUserByID :one
SELECT id, name, email, is_active, created_at, updated_at
FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, name, email, is_active, created_at, updated_at
FROM users WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id;

-- name: UpdateUser :exec
UPDATE users SET name = COALESCE($2, name), email = COALESCE($3, email), updated_at = NOW()
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, name, email, is_active, created_at, updated_at
FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;
```

```sql
-- db/queries/posts.sql
-- name: CreatePost :one
INSERT INTO posts (user_id, title, content) VALUES ($1, $2, $3) RETURNING id;

-- name: GetPostByID :one
SELECT p.id, p.user_id, p.title, p.content, p.published, p.created_at,
       u.name as author_name, u.email as author_email
FROM posts p
JOIN users u ON p.user_id = u.id
WHERE p.id = $1;

-- name: GetPostsByUserID :many
SELECT id, user_id, title, content, published, created_at
FROM posts WHERE user_id = $1 ORDER BY created_at DESC;

-- name: UpdatePost :exec
UPDATE posts SET title = $2, content = $3, published = $4
WHERE id = $1;

-- name: DeletePost :exec
DELETE FROM posts WHERE id = $1;

-- name: GetPublishedPosts :many
SELECT id, user_id, title, content, published, created_at
FROM posts WHERE published = true ORDER BY created_at DESC;
```

## Async Support

### Async/Await Patterns

```typescript
// src/services/userService.ts
import { db } from '../database';

export class UserService {
  async createUser(userData: { name: string; email: string }) {
    try {
      const userId = await db.createUser(userData);
      const user = await db.getUserById(userId);
      return user;
    } catch (error) {
      if (error.code === '23505') { // PostgreSQL unique violation
        throw new Error('Email already exists');
      }
      throw error;
    }
  }

  async getUserWithPosts(userId: number) {
    const [user, posts] = await Promise.all([
      db.getUserById(userId),
      db.getPostsByUserId(userId),
    ]);

    return {
      ...user,
      posts,
    };
  }

  async updateUserProfile(userId: number, updates: Partial<User>) {
    await db.updateUser(userId, updates);
    return db.getUserById(userId);
  }
}
```

### Error Handling

```typescript
// src/utils/database.ts
export async function withTransaction<T>(
  operation: () => Promise<T>
): Promise<T> {
  const client = await pool.connect();

  try {
    await client.query('BEGIN');
    const result = await operation();
    await client.query('COMMIT');
    return result;
  } catch (error) {
    await client.query('ROLLBACK');
    throw error;
  } finally {
    client.release();
  }
}

// Usage
const userId = await withTransaction(async () => {
  const userId = await db.createUser({ name, email });
  await db.createUserProfile({ userId, bio });
  return userId;
});
```

### Connection Pooling

```typescript
// src/database.ts
import { Pool } from 'pg';

const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
  max: 20, // Maximum number of clients in the pool
  idleTimeoutMillis: 30000, // Close idle clients after 30 seconds
  connectionTimeoutMillis: 2000, // Return an error after 2 seconds if connection could not be established
});

// Monitor pool events
pool.on('connect', (client) => {
  console.log('New client connected to the pool');
});

pool.on('error', (err, client) => {
  console.error('Unexpected error on idle client', err);
});

export { pool };
```

## Framework Integration

### Express.js Integration

```typescript
// src/routes/users.ts
import express from 'express';
import { db } from '../database';
import { UserService } from '../services/userService';

const router = express.Router();
const userService = new UserService();

// GET /users
router.get('/users', async (req, res) => {
  try {
    const limit = parseInt(req.query.limit as string) || 10;
    const offset = parseInt(req.query.offset as string) || 0;

    const users = await db.listUsers({ limit, offset });
    res.json(users);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// GET /users/:id
router.get('/users/:id', async (req, res) => {
  try {
    const userId = parseInt(req.params.id);
    const user = await db.getUserById(userId);

    if (!user) {
      return res.status(404).json({ error: 'User not found' });
    }

    res.json(user);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// POST /users
router.post('/users', async (req, res) => {
  try {
    const { name, email } = req.body;

    if (!name || !email) {
      return res.status(400).json({ error: 'Name and email are required' });
    }

    const user = await userService.createUser({ name, email });
    res.status(201).json(user);
  } catch (error) {
    if (error.message === 'Email already exists') {
      return res.status(409).json({ error: error.message });
    }
    res.status(500).json({ error: error.message });
  }
});

// PUT /users/:id
router.put('/users/:id', async (req, res) => {
  try {
    const userId = parseInt(req.params.id);
    const updates = req.body;

    await db.updateUser(userId, updates);
    const user = await db.getUserById(userId);

    res.json(user);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// DELETE /users/:id
router.delete('/users/:id', async (req, res) => {
  try {
    const userId = parseInt(req.params.id);
    await db.deleteUser(userId);
    res.status(204).send();
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

export default router;
```

### Fastify Integration

```typescript
// src/routes/users.ts
import { FastifyInstance } from 'fastify';
import { db } from '../database';

export default async function userRoutes(fastify: FastifyInstance) {
  // GET /users
  fastify.get('/users', async (request, reply) => {
    const { limit = 10, offset = 0 } = request.query as any;

    const users = await db.listUsers({
      limit: parseInt(limit),
      offset: parseInt(offset),
    });

    return users;
  });

  // GET /users/:id
  fastify.get('/users/:id', async (request, reply) => {
    const { id } = request.params as any;
    const userId = parseInt(id);

    const user = await db.getUserById(userId);

    if (!user) {
      return reply.code(404).send({ error: 'User not found' });
    }

    return user;
  });

  // POST /users
  fastify.post('/users', async (request, reply) => {
    const { name, email } = request.body as any;

    if (!name || !email) {
      return reply.code(400).send({ error: 'Name and email are required' });
    }

    try {
      const user = await db.createUser({ name, email });
      return reply.code(201).send(user);
    } catch (error: any) {
      if (error.code === '23505') {
        return reply.code(409).send({ error: 'Email already exists' });
      }
      throw error;
    }
  });
}
```

### NestJS Integration

```typescript
// src/users/users.service.ts
import { Injectable } from '@nestjs/common';
import { db } from '../database';

@Injectable()
export class UsersService {
  async findAll(limit = 10, offset = 0) {
    return db.listUsers({ limit, offset });
  }

  async findOne(id: number) {
    return db.getUserById(id);
  }

  async create(userData: { name: string; email: string }) {
    return db.createUser(userData);
  }

  async update(id: number, updates: Partial<User>) {
    await db.updateUser(id, updates);
    return db.getUserById(id);
  }

  async remove(id: number) {
    return db.deleteUser(id);
  }
}

// src/users/users.controller.ts
import { Controller, Get, Post, Body, Param, Put, Delete, Query } from '@nestjs/common';
import { UsersService } from './users.service';

@Controller('users')
export class UsersController {
  constructor(private readonly usersService: UsersService) {}

  @Get()
  findAll(@Query('limit') limit = 10, @Query('offset') offset = 0) {
    return this.usersService.findAll(limit, offset);
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.usersService.findOne(+id);
  }

  @Post()
  create(@Body() createUserDto: { name: string; email: string }) {
    return this.usersService.create(createUserDto);
  }

  @Put(':id')
  update(@Param('id') id: string, @Body() updateUserDto: Partial<User>) {
    return this.usersService.update(+id, updateUserDto);
  }

  @Delete(':id')
  remove(@Param('id') id: string) {
    return this.usersService.remove(+id);
  }
}
```

## Best Practices

### Project Structure

```
myproject/
├── db/
│   ├── schema/          # Schema files
│   │   ├── users.sql
│   │   └── posts.sql
│   ├── queries/         # Query files
│   │   ├── users.sql
│   │   └── posts.sql
│   └── migrations/      # Migration files
├── flash_gen/           # Generated code (don't edit)
├── src/
│   ├── database.ts      # Database connection
│   ├── services/        # Business logic
│   ├── controllers/     # Route handlers
│   ├── middleware/      # Custom middleware
│   └── utils/           # Utility functions
├── test/                # Tests
├── dist/                # Compiled output
├── package.json
├── tsconfig.json
└── flash.config.json
```

### Error Handling

```typescript
// src/utils/errorHandler.ts
export class DatabaseError extends Error {
  constructor(message: string, public code?: string) {
    super(message);
    this.name = 'DatabaseError';
  }
}

export function handleDatabaseError(error: any): never {
  if (error.code === '23505') { // Unique violation
    throw new DatabaseError('Resource already exists', error.code);
  }

  if (error.code === '23503') { // Foreign key violation
    throw new DatabaseError('Related resource not found', error.code);
  }

  if (error.code === '23502') { // Not null violation
    throw new DatabaseError('Required field is missing', error.code);
  }

  throw new DatabaseError('Database operation failed', error.code);
}

// Usage
try {
  await db.createUser({ name, email });
} catch (error) {
  handleDatabaseError(error);
}
```

### Validation

```typescript
// src/validation/userValidation.ts
import Joi from 'joi';

export const createUserSchema = Joi.object({
  name: Joi.string().min(2).max(100).required(),
  email: Joi.string().email().required(),
});

export const updateUserSchema = Joi.object({
  name: Joi.string().min(2).max(100),
  email: Joi.string().email(),
}).min(1); // At least one field must be provided

// Usage
export function validateCreateUser(data: any) {
  return createUserSchema.validate(data);
}

export function validateUpdateUser(data: any) {
  return updateUserSchema.validate(data);
}
```

### Testing

```typescript
// src/__tests__/users.test.ts
import { db } from '../database';
import { UserService } from '../services/userService';

describe('UserService', () => {
  let userService: UserService;

  beforeEach(() => {
    userService = new UserService();
  });

  afterEach(async () => {
    // Clean up test data
    await db.deleteAllUsers(); // You'd need to add this query
  });

  it('should create a user', async () => {
    const userData = { name: 'Test User', email: 'test@example.com' };
    const user = await userService.createUser(userData);

    expect(user).toHaveProperty('id');
    expect(user.name).toBe(userData.name);
    expect(user.email).toBe(userData.email);
  });

  it('should get user by id', async () => {
    const userData = { name: 'Test User', email: 'test@example.com' };
    const createdUser = await userService.createUser(userData);
    const fetchedUser = await db.getUserById(createdUser.id);

    expect(fetchedUser).toEqual(createdUser);
  });

  it('should throw error for duplicate email', async () => {
    const userData = { name: 'Test User', email: 'test@example.com' };
    await userService.createUser(userData);

    await expect(userService.createUser(userData)).rejects.toThrow('Email already exists');
  });
});
```

### Performance Tips

- Use connection pooling properly
- Implement proper indexing in your schema
- Use prepared statements (automatic in generated code)
- Batch operations when possible
- Use appropriate data types
- Monitor query performance

### Security Best Practices

```typescript
// src/middleware/security.ts
import { Request, Response, NextFunction } from 'express';

// SQL injection prevention (automatic with parameterized queries)
// XSS prevention
export function sanitizeInput(req: Request, res: Response, next: NextFunction) {
  // Sanitize request body, params, and query
  // Implement your sanitization logic here
  next();
}

// Rate limiting
export function rateLimit(req: Request, res: Response, next: NextFunction) {
  // Implement rate limiting logic
  next();
}

// Authentication middleware
export function authenticate(req: Request, res: Response, next: NextFunction) {
  const token = req.headers.authorization?.replace('Bearer ', '');

  if (!token) {
    return res.status(401).json({ error: 'Authentication required' });
  }

  // Verify token and set user context
  // req.user = decodedUser;
  next();
}
```

## Troubleshooting

### Common Issues

**TypeScript compilation errors**
- Ensure `flash_gen` is included in `tsconfig.json`
- Check that generated types match your usage
- Regenerate code after schema changes: `flash gen`

**Database connection issues**
- Verify `DATABASE_URL` environment variable
- Check database server is running and accessible
- Ensure user has proper permissions

**Migration errors**
- Check migration status: `flash status`
- Ensure migrations are in correct order
- Test migrations on development before production

### Getting Help

- [GitHub Issues](https://github.com/Lumos-Labs-HQ/flash/issues)
- [TypeScript Documentation](https://www.typescriptlang.org/docs/)
- [Node.js Best Practices](https://github.com/goldbergyoni/nodebestpractices)
