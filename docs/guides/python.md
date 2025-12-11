---
title: Python Guide
description: Complete guide to using Flash ORM with Python
---

# Flash ORM - Python Usage Guide

A comprehensive guide to using Flash ORM with Python projects, featuring async support and Pythonic APIs.

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
pip install flashorm
```

Verify installation:
```bash
flash --version
```

## Quick Start

### 1. Create a New Project

```bash
mkdir myproject && cd myproject
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
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
    is_active BOOLEAN DEFAULT TRUE,
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

### 6. Generate Python Code

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
    "python": {
      "enabled": true,
      "out": "flash_gen",
      "async": true
    }
  }
}
```

### Python Dependencies

```txt
# requirements.txt
asyncpg>=0.29.0      # PostgreSQL async driver
aiomysql>=0.1.1      # MySQL async driver
aiosqlite>=0.19.0    # SQLite async driver
python-dotenv>=1.0.0 # Environment variables
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
├── __init__.py      # Package initialization
├── database.py      # Database connection interface
├── users.py         # User-related queries
├── posts.py         # Post-related queries
└── ...
```

### Generated Python Types

```python
# flash_gen/__init__.py
from .database import new
from .users import Queries as UserQueries
from .posts import Queries as PostQueries

__all__ = ["new", "UserQueries", "PostQueries"]
```

```python
# flash_gen/database.py
from typing import Any, Protocol

class DatabaseConnection(Protocol):
    """Protocol for database connections."""
    async def execute(self, query: str, *args) -> Any: ...
    async def fetchrow(self, query: str, *args) -> dict | None: ...
    async def fetch(self, query: str, *args) -> list[dict]: ...

def new(db: DatabaseConnection) -> "Queries":
    """Create a new database client."""
    return Queries(db)

class Queries:
    def __init__(self, db: DatabaseConnection):
        self.db = db
```

## Working with Generated Code

### Database Connection

```python
# src/database.py
import os
import asyncpg
from flash_gen import new

DATABASE_URL = os.getenv('DATABASE_URL', 'postgresql://postgres:postgres@localhost:5432/flashorm_test')

async def create_pool():
    """Create database connection pool."""
    return await asyncpg.create_pool(DATABASE_URL)

async def get_database():
    """Get database instance."""
    pool = await create_pool()
    return new(pool)

# Usage
db = await get_database()
```

### Using Generated Queries

```python
# src/main.py
import asyncio
import os
from flash_gen import new
import asyncpg

DATABASE_URL = os.getenv('DATABASE_URL', 'postgresql://postgres:postgres@localhost:5432/flashorm_test')

async def main():
    # Create connection pool
    pool = await asyncpg.create_pool(DATABASE_URL)

    try:
        # Create database instance
        db = new(pool)

        # Create a user
        user_id = await db.create_user('John Doe', 'john@example.com')
        print(f'Created user with ID: {user_id}')

        # Get user by ID
        user = await db.get_user_by_id(user_id)
        print(f'User: {user}')

        # Create a post
        post_id = await db.create_post(user_id, 'My First Post', 'This is the content of my first post.')
        print(f'Created post with ID: {post_id}')

        # Get posts by user
        posts = await db.get_posts_by_user_id(user_id)
        print(f'Posts: {posts}')

        # Update user
        await db.update_user(user_id, name='John Smith')

        # Get updated user
        updated_user = await db.get_user_by_id(user_id)
        print(f'Updated user: {updated_user}')

        # Delete post
        await db.delete_post(post_id)

    finally:
        await pool.close()

if __name__ == '__main__':
    asyncio.run(main())
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

```python
# src/services/user_service.py
import asyncio
from typing import Optional, List
from flash_gen import new

class UserService:
    def __init__(self, db):
        self.db = db

    async def create_user(self, name: str, email: str) -> dict:
        """Create a new user."""
        try:
            user_id = await self.db.create_user(name, email)
            user = await self.db.get_user_by_id(user_id)
            return user
        except Exception as e:
            if 'unique constraint' in str(e).lower():
                raise ValueError('Email already exists')
            raise

    async def get_user_with_posts(self, user_id: int) -> Optional[dict]:
        """Get user with their posts."""
        user, posts = await asyncio.gather(
            self.db.get_user_by_id(user_id),
            self.db.get_posts_by_user_id(user_id)
        )

        if user:
            user['posts'] = posts
            return user
        return None

    async def update_user_profile(self, user_id: int, name: Optional[str] = None, email: Optional[str] = None) -> dict:
        """Update user profile."""
        await self.db.update_user(user_id, name=name, email=email)
        return await self.db.get_user_by_id(user_id)
```

### Error Handling

```python
# src/utils/database.py
import asyncio
from contextlib import asynccontextmanager
from typing import AsyncGenerator

@asynccontextmanager
async def transaction(db) -> AsyncGenerator:
    """Database transaction context manager."""
    try:
        await db.execute('BEGIN')
        yield db
        await db.execute('COMMIT')
    except Exception as e:
        await db.execute('ROLLBACK')
        raise e

# Usage
async def create_user_with_profile(db, name: str, email: str, bio: str):
    async with transaction(db) as tx:
        user_id = await tx.create_user(name, email)
        await tx.create_user_profile(user_id, bio)
        return user_id
```

### Connection Pooling

```python
# src/database.py
import asyncpg
from typing import Optional

class DatabaseManager:
    def __init__(self, dsn: str):
        self.dsn = dsn
        self.pool: Optional[asyncpg.Pool] = None

    async def connect(self):
        """Create connection pool."""
        if self.pool is None:
            self.pool = await asyncpg.create_pool(
                self.dsn,
                min_size=5,      # Minimum connections
                max_size=20,     # Maximum connections
                command_timeout=60,  # Command timeout in seconds
            )

    async def disconnect(self):
        """Close connection pool."""
        if self.pool:
            await self.pool.close()
            self.pool = None

    async def get_db(self):
        """Get database instance."""
        if self.pool is None:
            await self.connect()
        return new(self.pool)

# Global instance
db_manager = DatabaseManager(os.getenv('DATABASE_URL'))

async def get_database():
    return await db_manager.get_db()

# Graceful shutdown
async def shutdown():
    await db_manager.disconnect()
```

## Framework Integration

### FastAPI Integration

```python
# src/main.py
from fastapi import FastAPI, HTTPException, Depends
from typing import List, Optional
from flash_gen import new
import asyncpg

app = FastAPI(title="Flash ORM API")

# Database dependency
async def get_db():
    pool = await asyncpg.create_pool(os.getenv('DATABASE_URL'))
    try:
        yield new(pool)
    finally:
        await pool.close()

@app.get("/users", response_model=List[dict])
async def list_users(limit: int = 10, offset: int = 0, db=Depends(get_db)):
    """List users with pagination."""
    return await db.list_users(limit, offset)

@app.get("/users/{user_id}")
async def get_user(user_id: int, db=Depends(get_db)):
    """Get user by ID."""
    user = await db.get_user_by_id(user_id)
    if not user:
        raise HTTPException(status_code=404, detail="User not found")
    return user

@app.post("/users", status_code=201)
async def create_user(user_data: dict, db=Depends(get_db)):
    """Create a new user."""
    try:
        name = user_data['name']
        email = user_data['email']
        user_id = await db.create_user(name, email)
        user = await db.get_user_by_id(user_id)
        return user
    except Exception as e:
        if 'unique constraint' in str(e).lower():
            raise HTTPException(status_code=409, detail="Email already exists")
        raise HTTPException(status_code=500, detail=str(e))

@app.put("/users/{user_id}")
async def update_user(user_id: int, user_data: dict, db=Depends(get_db)):
    """Update user."""
    name = user_data.get('name')
    email = user_data.get('email')

    if not name and not email:
        raise HTTPException(status_code=400, detail="At least one field must be provided")

    try:
        await db.update_user(user_id, name=name, email=email)
        user = await db.get_user_by_id(user_id)
        if not user:
            raise HTTPException(status_code=404, detail="User not found")
        return user
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.delete("/users/{user_id}", status_code=204)
async def delete_user(user_id: int, db=Depends(get_db)):
    """Delete user."""
    try:
        await db.delete_user(user_id)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
```

### Django Integration

```python
# myapp/views.py
from django.http import JsonResponse
from django.views.decorators.csrf import csrf_exempt
from django.utils.decorators import method_decorator
from django.views import View
import json
import asyncio
from flash_gen import new
import asyncpg

class UserView(View):
    def setup(self, request, *args, **kwargs):
        super().setup(request, *args, **kwargs)
        # Create database connection (in production, use connection pooling)
        self.pool = asyncio.run(asyncpg.create_pool(os.getenv('DATABASE_URL')))
        self.db = new(self.pool)

    def get(self, request, user_id=None):
        """Handle GET requests."""
        if user_id:
            # Get single user
            user = asyncio.run(self.db.get_user_by_id(int(user_id)))
            if not user:
                return JsonResponse({'error': 'User not found'}, status=404)
            return JsonResponse(user)
        else:
            # List users
            limit = int(request.GET.get('limit', 10))
            offset = int(request.GET.get('offset', 0))
            users = asyncio.run(self.db.list_users(limit, offset))
            return JsonResponse(users, safe=False)

    @method_decorator(csrf_exempt)
    def post(self, request):
        """Handle POST requests."""
        try:
            data = json.loads(request.body)
            name = data['name']
            email = data['email']

            user_id = asyncio.run(self.db.create_user(name, email))
            user = asyncio.run(self.db.get_user_by_id(user_id))

            return JsonResponse(user, status=201)
        except KeyError:
            return JsonResponse({'error': 'Name and email are required'}, status=400)
        except Exception as e:
            if 'unique constraint' in str(e).lower():
                return JsonResponse({'error': 'Email already exists'}, status=409)
            return JsonResponse({'error': str(e)}, status=500)

    @method_decorator(csrf_exempt)
    def put(self, request, user_id):
        """Handle PUT requests."""
        try:
            data = json.loads(request.body)
            name = data.get('name')
            email = data.get('email')

            if not name and not email:
                return JsonResponse({'error': 'At least one field must be provided'}, status=400)

            asyncio.run(self.db.update_user(int(user_id), name=name, email=email))
            user = asyncio.run(self.db.get_user_by_id(int(user_id)))

            if not user:
                return JsonResponse({'error': 'User not found'}, status=404)

            return JsonResponse(user)
        except Exception as e:
            return JsonResponse({'error': str(e)}, status=500)

    @method_decorator(csrf_exempt)
    def delete(self, request, user_id):
        """Handle DELETE requests."""
        try:
            asyncio.run(self.db.delete_user(int(user_id)))
            return JsonResponse({}, status=204)
        except Exception as e:
            return JsonResponse({'error': str(e)}, status=500)
```

### Flask Integration

```python
# app.py
from flask import Flask, request, jsonify
import asyncio
import os
from flash_gen import new
import asyncpg

app = Flask(__name__)

# Database connection
pool = None
db = None

async def init_db():
    global pool, db
    pool = await asyncpg.create_pool(os.getenv('DATABASE_URL'))
    db = new(pool)

@app.before_first_request
def setup():
    asyncio.run(init_db())

@app.teardown_appcontext
def cleanup(error):
    if pool:
        asyncio.run(pool.close())

@app.route('/users', methods=['GET'])
def list_users():
    limit = int(request.args.get('limit', 10))
    offset = int(request.args.get('offset', 0))

    users = asyncio.run(db.list_users(limit, offset))
    return jsonify(users)

@app.route('/users/<int:user_id>', methods=['GET'])
def get_user(user_id):
    user = asyncio.run(db.get_user_by_id(user_id))
    if not user:
        return jsonify({'error': 'User not found'}), 404
    return jsonify(user)

@app.route('/users', methods=['POST'])
def create_user():
    try:
        data = request.get_json()
        name = data['name']
        email = data['email']

        user_id = asyncio.run(db.create_user(name, email))
        user = asyncio.run(db.get_user_by_id(user_id))

        return jsonify(user), 201
    except KeyError:
        return jsonify({'error': 'Name and email are required'}), 400
    except Exception as e:
        if 'unique constraint' in str(e).lower():
            return jsonify({'error': 'Email already exists'}), 409
        return jsonify({'error': str(e)}), 500

@app.route('/users/<int:user_id>', methods=['PUT'])
def update_user(user_id):
    try:
        data = request.get_json()
        name = data.get('name')
        email = data.get('email')

        if not name and not email:
            return jsonify({'error': 'At least one field must be provided'}), 400

        asyncio.run(db.update_user(user_id, name=name, email=email))
        user = asyncio.run(db.get_user_by_id(user_id))

        if not user:
            return jsonify({'error': 'User not found'}), 404

        return jsonify(user)
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/users/<int:user_id>', methods=['DELETE'])
def delete_user(user_id):
    try:
        asyncio.run(db.delete_user(user_id))
        return '', 204
    except Exception as e:
        return jsonify({'error': str(e)}), 500

if __name__ == '__main__':
    app.run(debug=True)
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
│   ├── database.py      # Database connection
│   ├── services/        # Business logic
│   │   ├── user_service.py
│   │   └── post_service.py
│   ├── routes/          # Route handlers
│   │   ├── users.py
│   │   └── posts.py
│   ├── models/          # Data models
│   └── utils/           # Utility functions
├── tests/               # Tests
│   ├── test_users.py
│   └── test_posts.py
├── requirements.txt
├── flash.config.json
└── main.py
```

### Error Handling

```python
# src/exceptions.py
class DatabaseError(Exception):
    """Base database exception."""
    pass

class NotFoundError(DatabaseError):
    """Resource not found."""
    pass

class ValidationError(DatabaseError):
    """Validation error."""
    pass

class ConflictError(DatabaseError):
    """Resource conflict."""
    pass

# src/utils/error_handler.py
def handle_database_error(error: Exception) -> None:
    """Handle database errors and raise appropriate exceptions."""
    error_msg = str(error).lower()

    if 'unique constraint' in error_msg or 'duplicate key' in error_msg:
        raise ConflictError('Resource already exists')
    elif 'foreign key' in error_msg:
        raise ValidationError('Related resource not found')
    elif 'not null' in error_msg:
        raise ValidationError('Required field is missing')
    elif 'no rows' in error_msg:
        raise NotFoundError('Resource not found')
    else:
        raise DatabaseError('Database operation failed')

# Usage
try:
    await db.create_user(name, email)
except Exception as e:
    handle_database_error(e)
```

### Validation

```python
# src/validation/user_validation.py
from pydantic import BaseModel, EmailStr, Field
from typing import Optional

class CreateUserRequest(BaseModel):
    name: str = Field(..., min_length=2, max_length=100)
    email: EmailStr

class UpdateUserRequest(BaseModel):
    name: Optional[str] = Field(None, min_length=2, max_length=100)
    email: Optional[EmailStr] = None

    class Config:
        validate_assignment = True

# Usage
@app.post("/users")
async def create_user(request: CreateUserRequest):
    # Validation is automatic
    user_id = await db.create_user(request.name, request.email)
    user = await db.get_user_by_id(user_id)
    return user
```

### Testing

```python
# tests/test_users.py
import pytest
import asyncio
from flash_gen import new
import asyncpg

@pytest.fixture
async def db():
    """Database fixture."""
    pool = await asyncpg.create_pool('postgresql://test:test@localhost:5432/testdb')
    yield new(pool)
    await pool.close()

@pytest.fixture(autouse=True)
async def cleanup(db):
    """Clean up test data."""
    yield
    # Clean up logic here

class TestUserService:
    @pytest.mark.asyncio
    async def test_create_user(self, db):
        """Test user creation."""
        user_id = await db.create_user('Test User', 'test@example.com')
        assert user_id > 0

        user = await db.get_user_by_id(user_id)
        assert user['name'] == 'Test User'
        assert user['email'] == 'test@example.com'

    @pytest.mark.asyncio
    async def test_get_user_by_id(self, db):
        """Test getting user by ID."""
        user_id = await db.create_user('Test User', 'test@example.com')
        user = await db.get_user_by_id(user_id)

        assert user is not None
        assert user['id'] == user_id

    @pytest.mark.asyncio
    async def test_duplicate_email_error(self, db):
        """Test duplicate email handling."""
        await db.create_user('User 1', 'test@example.com')

        with pytest.raises(Exception) as exc_info:
            await db.create_user('User 2', 'test@example.com')

        assert 'unique constraint' in str(exc_info.value).lower()
```

### Performance Tips

- Use connection pooling properly
- Implement proper indexing in your schema
- Use async/await consistently
- Batch operations when possible
- Use appropriate data types
- Monitor query performance with `EXPLAIN ANALYZE`

### Security Best Practices

```python
# src/middleware/security.py
from functools import wraps
import jwt
from flask import request, g
import os

JWT_SECRET = os.getenv('JWT_SECRET', 'your-secret-key')

def login_required(f):
    @wraps(f)
    def decorated_function(*args, **kwargs):
        token = request.headers.get('Authorization', '').replace('Bearer ', '')

        if not token:
            return {'error': 'Authentication required'}, 401

        try:
            payload = jwt.decode(token, JWT_SECRET, algorithms=['HS256'])
            g.user_id = payload['user_id']
        except jwt.ExpiredSignatureError:
            return {'error': 'Token expired'}, 401
        except jwt.InvalidTokenError:
            return {'error': 'Invalid token'}, 401

        return f(*args, **kwargs)
    return decorated_function

# Usage
@app.route('/users/profile')
@login_required
def get_profile():
    user = asyncio.run(db.get_user_by_id(g.user_id))
    return jsonify(user)
```

## Troubleshooting

### Common Issues

**Async issues**
- Ensure all database operations are awaited
- Use `asyncio.run()` for top-level async functions
- Check for mixed sync/async code

**Connection issues**
- Verify `DATABASE_URL` environment variable
- Check database server is running
- Ensure user has proper permissions

**Migration errors**
- Check migration status: `flash status`
- Ensure migrations are in correct order
- Test migrations on development before production

### Getting Help

- [GitHub Issues](https://github.com/Lumos-Labs-HQ/flash/issues)
- [Python Async Documentation](https://docs.python.org/3/library/asyncio.html)
- [FastAPI Documentation](https://fastapi.tiangolo.com/)
