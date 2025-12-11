# FlashORM - Python Usage Guide

A comprehensive guide to using FlashORM with Python projects, featuring async support.

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

---

## Installation

### Install FlashORM CLI

```bash
pip install flashorm
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
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
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
    "python": {
      "enabled": true,
      "output": "flash_gen",
      "async": true
    }
  }
}
```

### Install Database Drivers

```bash
# PostgreSQL
pip install asyncpg psycopg2-binary

# MySQL
pip install aiomysql pymysql

# SQLite
pip install aiosqlite

# All
pip install python-dotenv
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
CREATE INDEX idx_users_role ON users(role);
```

### Foreign Keys

```sql
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT
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
flash apply
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

### Generate Python Models

```bash
flash gen
```

Generated files in `flash_gen/`:

```
flash_gen/
├── __init__.py
├── models.py       # Pydantic models
├── database.py     # Database connection
├── users.py        # User queries
└── posts.py        # Post queries
```

### Generated Models

```python
# flash_gen/models.py
from datetime import datetime
from typing import Optional, Dict, Any
from pydantic import BaseModel, Field


class User(BaseModel):
    id: int
    name: str
    email: str
    role: str = "user"
    is_active: bool = True
    metadata: Optional[Dict[str, Any]] = None
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True


class CreateUserInput(BaseModel):
    name: str = Field(..., max_length=100)
    email: str = Field(..., max_length=255)
    role: str = Field(default="user", max_length=50)
    is_active: bool = True
    metadata: Optional[Dict[str, Any]] = None


class UpdateUserInput(BaseModel):
    name: Optional[str] = Field(None, max_length=100)
    email: Optional[str] = Field(None, max_length=255)
    role: Optional[str] = Field(None, max_length=50)
    is_active: Optional[bool] = None
    metadata: Optional[Dict[str, Any]] = None


class Post(BaseModel):
    id: int
    user_id: int
    title: str
    content: Optional[str] = None
    published: bool = False
    created_at: datetime

    class Config:
        from_attributes = True
```

---

## Working with Generated Code

### Synchronous Usage

```python
# main.py
import os
from dotenv import load_dotenv
from flash_gen import FlashDB, CreateUserInput

load_dotenv()

# Create database connection
db = FlashDB(os.getenv("DATABASE_URL"))

# Create a user
new_user = CreateUserInput(
    name="John Doe",
    email="john@example.com",
    role="admin"
)
user = db.users.create(new_user)
print(f"Created user: {user}")

# Get all users
users = db.users.find_many()
for u in users:
    print(f"User: {u.name} ({u.email})")

# Find user by ID
user = db.users.find_by_id(1)

# Find user by email
user = db.users.find_one(email="john@example.com")

# Update user
updated_user = db.users.update(user.id, name="Jane Doe")

# Delete user
db.users.delete(user.id)

# Close connection
db.close()
```

---

## Async Support

### Async Usage

```python
# async_main.py
import asyncio
import os
from dotenv import load_dotenv
from flash_gen import AsyncFlashDB, CreateUserInput

load_dotenv()


async def main():
    # Create async database connection
    db = await AsyncFlashDB.connect(os.getenv("DATABASE_URL"))

    # Create a user
    new_user = CreateUserInput(
        name="John Doe",
        email="john@example.com"
    )
    user = await db.users.create(new_user)
    print(f"Created user: {user}")

    # Get all users
    users = await db.users.find_many()
    async for u in users:
        print(f"User: {u.name}")

    # Find with filters
    active_users = await db.users.find_many(
        where={"is_active": True},
        order_by={"created_at": "desc"},
        limit=10
    )

    # Update user
    updated = await db.users.update(
        user.id,
        name="Jane Doe"
    )

    # Delete user
    await db.users.delete(user.id)

    # Close connection
    await db.close()


if __name__ == "__main__":
    asyncio.run(main())
```

### Context Manager

```python
async def main():
    async with AsyncFlashDB.connect(os.getenv("DATABASE_URL")) as db:
        users = await db.users.find_many()
        # Connection automatically closed when exiting context
```

### Transactions

```python
async def transfer_credits(
    db: AsyncFlashDB,
    from_user_id: int,
    to_user_id: int,
    amount: float
):
    async with db.transaction() as tx:
        # Deduct from sender
        await tx.users.update(
            from_user_id,
            credits={"decrement": amount}
        )
        
        # Add to receiver
        await tx.users.update(
            to_user_id,
            credits={"increment": amount}
        )
        
        # Log transaction
        await tx.transactions.create({
            "from_user_id": from_user_id,
            "to_user_id": to_user_id,
            "amount": amount,
            "type": "transfer"
        })
        
        # Automatically commits if no exception
```

---

## Framework Integration

### FastAPI

```python
# app/main.py
from fastapi import FastAPI, Depends, HTTPException
from contextlib import asynccontextmanager
import os
from dotenv import load_dotenv

from flash_gen import AsyncFlashDB, User, CreateUserInput, UpdateUserInput

load_dotenv()

# Database instance
db: AsyncFlashDB = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    global db
    db = await AsyncFlashDB.connect(os.getenv("DATABASE_URL"))
    yield
    # Shutdown
    await db.close()


app = FastAPI(lifespan=lifespan)


async def get_db() -> AsyncFlashDB:
    return db


@app.get("/users", response_model=list[User])
async def list_users(
    skip: int = 0,
    limit: int = 10,
    db: AsyncFlashDB = Depends(get_db)
):
    return await db.users.find_many(skip=skip, limit=limit)


@app.get("/users/{user_id}", response_model=User)
async def get_user(
    user_id: int,
    db: AsyncFlashDB = Depends(get_db)
):
    user = await db.users.find_by_id(user_id)
    if not user:
        raise HTTPException(status_code=404, detail="User not found")
    return user


@app.post("/users", response_model=User, status_code=201)
async def create_user(
    user_data: CreateUserInput,
    db: AsyncFlashDB = Depends(get_db)
):
    return await db.users.create(user_data)


@app.put("/users/{user_id}", response_model=User)
async def update_user(
    user_id: int,
    user_data: UpdateUserInput,
    db: AsyncFlashDB = Depends(get_db)
):
    user = await db.users.update(user_id, **user_data.model_dump(exclude_unset=True))
    if not user:
        raise HTTPException(status_code=404, detail="User not found")
    return user


@app.delete("/users/{user_id}", status_code=204)
async def delete_user(
    user_id: int,
    db: AsyncFlashDB = Depends(get_db)
):
    success = await db.users.delete(user_id)
    if not success:
        raise HTTPException(status_code=404, detail="User not found")
```

### Flask

```python
# app.py
from flask import Flask, jsonify, request
import os
from dotenv import load_dotenv

from flash_gen import FlashDB, CreateUserInput

load_dotenv()

app = Flask(__name__)
db = FlashDB(os.getenv("DATABASE_URL"))


@app.route('/users', methods=['GET'])
def list_users():
    users = db.users.find_many()
    return jsonify([u.model_dump() for u in users])


@app.route('/users/<int:user_id>', methods=['GET'])
def get_user(user_id: int):
    user = db.users.find_by_id(user_id)
    if not user:
        return jsonify({'error': 'User not found'}), 404
    return jsonify(user.model_dump())


@app.route('/users', methods=['POST'])
def create_user():
    data = request.get_json()
    user_input = CreateUserInput(**data)
    user = db.users.create(user_input)
    return jsonify(user.model_dump()), 201


@app.route('/users/<int:user_id>', methods=['PUT'])
def update_user(user_id: int):
    data = request.get_json()
    user = db.users.update(user_id, **data)
    if not user:
        return jsonify({'error': 'User not found'}), 404
    return jsonify(user.model_dump())


@app.route('/users/<int:user_id>', methods=['DELETE'])
def delete_user(user_id: int):
    success = db.users.delete(user_id)
    if not success:
        return jsonify({'error': 'User not found'}), 404
    return '', 204


@app.teardown_appcontext
def close_db(error):
    db.close()


if __name__ == '__main__':
    app.run(debug=True)
```

### Django

```python
# myapp/db.py
import os
from flash_gen import FlashDB

# Global database instance
_db = None

def get_db() -> FlashDB:
    global _db
    if _db is None:
        _db = FlashDB(os.getenv("DATABASE_URL"))
    return _db


# myapp/views.py
from django.http import JsonResponse
from django.views import View
from .db import get_db
from flash_gen import CreateUserInput
import json


class UserListView(View):
    def get(self, request):
        db = get_db()
        users = db.users.find_many()
        return JsonResponse([u.model_dump() for u in users], safe=False)

    def post(self, request):
        db = get_db()
        data = json.loads(request.body)
        user_input = CreateUserInput(**data)
        user = db.users.create(user_input)
        return JsonResponse(user.model_dump(), status=201)


class UserDetailView(View):
    def get(self, request, user_id):
        db = get_db()
        user = db.users.find_by_id(user_id)
        if not user:
            return JsonResponse({'error': 'User not found'}, status=404)
        return JsonResponse(user.model_dump())

    def put(self, request, user_id):
        db = get_db()
        data = json.loads(request.body)
        user = db.users.update(user_id, **data)
        if not user:
            return JsonResponse({'error': 'User not found'}, status=404)
        return JsonResponse(user.model_dump())

    def delete(self, request, user_id):
        db = get_db()
        success = db.users.delete(user_id)
        if not success:
            return JsonResponse({'error': 'User not found'}, status=404)
        return JsonResponse({}, status=204)
```

---

## Advanced Usage

### Raw Queries

```python
# Synchronous
results = db.query("SELECT * FROM users WHERE role = %s", ["admin"])

# Async
results = await db.query("SELECT * FROM users WHERE role = $1", ["admin"])

# Execute (no return)
db.execute("UPDATE users SET is_active = false WHERE last_login < %s", [cutoff_date])
```

### Complex Queries

```python
# With filters
users = await db.users.find_many(
    where={
        "role": "admin",
        "is_active": True,
        "created_at": {"gte": datetime(2025, 1, 1)}
    },
    order_by=[
        {"created_at": "desc"},
        {"name": "asc"}
    ],
    skip=0,
    limit=10
)

# With relations
users_with_posts = await db.users.find_many(
    include={"posts": True}
)

# Count
count = await db.users.count(where={"is_active": True})

# Exists
exists = await db.users.exists(email="john@example.com")
```

### Bulk Operations

```python
# Bulk create
users = await db.users.create_many([
    CreateUserInput(name="User 1", email="user1@example.com"),
    CreateUserInput(name="User 2", email="user2@example.com"),
    CreateUserInput(name="User 3", email="user3@example.com"),
])

# Bulk update
await db.users.update_many(
    where={"is_active": False},
    data={"role": "inactive"}
)

# Bulk delete
await db.users.delete_many(
    where={"last_login": {"lt": cutoff_date}}
)
```

---

## Best Practices

### 1. Use Connection Pooling

```python
from flash_gen import AsyncFlashDB

db = await AsyncFlashDB.connect(
    os.getenv("DATABASE_URL"),
    pool_size=20,
    max_overflow=10
)
```

### 2. Handle Errors

```python
from flash_gen import DatabaseError, NotFoundError

try:
    user = await db.users.create(CreateUserInput(
        email="duplicate@email.com"
    ))
except DatabaseError as e:
    if "unique constraint" in str(e):
        raise ValueError("Email already exists")
    raise
```

### 3. Use Type Hints

```python
from typing import List, Optional
from flash_gen import User, AsyncFlashDB


async def get_active_users(
    db: AsyncFlashDB,
    limit: int = 10
) -> List[User]:
    return await db.users.find_many(
        where={"is_active": True},
        limit=limit
    )


async def get_user_by_email(
    db: AsyncFlashDB,
    email: str
) -> Optional[User]:
    return await db.users.find_one(email=email)
```

### 4. Environment Management

```python
# config.py
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    database_url: str
    debug: bool = False
    
    class Config:
        env_file = ".env"


settings = Settings()
```

### 5. Testing

```python
# tests/conftest.py
import pytest
import asyncio
from flash_gen import AsyncFlashDB


@pytest.fixture(scope="session")
def event_loop():
    loop = asyncio.get_event_loop_policy().new_event_loop()
    yield loop
    loop.close()


@pytest.fixture
async def db():
    db = await AsyncFlashDB.connect("sqlite:///:memory:")
    await db.migrate()
    yield db
    await db.close()


# tests/test_users.py
import pytest
from flash_gen import CreateUserInput


@pytest.mark.asyncio
async def test_create_user(db):
    user = await db.users.create(
        CreateUserInput(name="Test", email="test@example.com")
    )
    assert user.id is not None
    assert user.name == "Test"


@pytest.mark.asyncio
async def test_find_user(db):
    await db.users.create(
        CreateUserInput(name="Test", email="test@example.com")
    )
    user = await db.users.find_one(email="test@example.com")
    assert user is not None
    assert user.name == "Test"
```

---

## Type Mapping

| PostgreSQL | MySQL | SQLite | Python |
|------------|-------|--------|--------|
| SERIAL | INT AUTO_INCREMENT | INTEGER | int |
| BIGSERIAL | BIGINT AUTO_INCREMENT | INTEGER | int |
| VARCHAR | VARCHAR | TEXT | str |
| TEXT | TEXT | TEXT | str |
| BOOLEAN | TINYINT(1) | INTEGER | bool |
| TIMESTAMP | DATETIME | TEXT | datetime |
| JSONB | JSON | TEXT | Dict[str, Any] |
| FLOAT | FLOAT | REAL | float |
| NUMERIC | DECIMAL | NUMERIC | Decimal |

---

## Troubleshooting

### Connection Issues

```bash
# Check database URL
echo $DATABASE_URL

# Test connection
flash status
```

### Import Errors

```bash
# Ensure flash_gen is in path
export PYTHONPATH="${PYTHONPATH}:$(pwd)"

# Or install as package
pip install -e .
```

### Async Issues

```python
# Wrong: Running async in sync context
users = db.users.find_many()  # Error!

# Correct: Use asyncio.run()
users = asyncio.run(db.users.find_many())

# Or in async context
async def main():
    users = await db.users.find_many()
```

---

## Resources

- [FlashORM GitHub](https://github.com/Lumos-Labs-HQ/flash)
- [PyPI Package](https://pypi.org/project/flashorm/)
- [Release Notes](/releases.md)
- [Contributing Guide](/contributing)
