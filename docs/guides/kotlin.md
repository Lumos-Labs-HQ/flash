---
title: Kotlin Guide
description: Complete guide to using Flash ORM with Kotlin
---

# Flash ORM - Kotlin Usage Guide

A comprehensive guide to using Flash ORM with Kotlin projects, featuring JDBC, Exposed, and R2DBC driver support.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Package Auto-Detection](#package-auto-detection)
- [Generated Code Overview](#generated-code-overview)
- [Driver Options](#driver-options)
- [Working with JDBC](#working-with-jdbc)
- [Working with Exposed](#working-with-exposed)
- [Working with R2DBC](#working-with-r2dbc)
- [ScyllaDB / Cassandra Support](#scylladb--cassandra-support)
- [Schema Definition](#schema-definition)
- [Writing Queries](#writing-queries)
- [Building and Running](#building-and-running)
- [Best Practices](#best-practices)

## Prerequisites

- **Kotlin 1.9+**
- **JVM 17+**
- **Gradle** (Kotlin DSL or Groovy) or **Maven** build system
- **Flash ORM CLI** installed via npm, pip, or direct binary

## Quick Start

### 1. Initialize a Kotlin Project

```bash
# Initialize with PostgreSQL
flash init --postgresql

# Flash ORM auto-detects Gradle/Maven Kotlin projects
```

### 2. Generate Code

```bash
# After defining schema and queries:
flash generate
```

### 3. Configure Your `flash.toml`

```toml
[gen.kotlin]
enabled = true
out = "src/main/kotlin/com/myapp/db"
package = "com.myapp.db"
driver = "jdbc"
```

## Configuration

### Flash TOML Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `enabled` | `false` | Enable Kotlin code generation |
| `out` | `flash_gen` | Output directory for generated Kotlin files |
| `package` | derived from `out` dir | Kotlin package name (e.g., `com.example.db`) |
| `driver` | `jdbc` | Driver type: `jdbc`, `exposed`, or `r2dbc` |

### Driver-Specific Config

```toml
[gen.kotlin]
enabled = true
out = "src/main/kotlin/com/myapp/db"
package = "com.myapp.db"
driver = "jdbc"
```

## Package Auto-Detection

Flash ORM automatically detects your Kotlin package from build files during `flash init`:

### Gradle Kotlin DSL (`build.gradle.kts`)

The package is derived from the `group` property:

```kotlin
group = "com.kotlin.app"
// Generates: package = "com.kotlin.app"
```

Also supports the `group(...)` function syntax:

```kotlin
group("com.kotlin.app")
```

### Gradle Groovy (`build.gradle`)

```groovy
group = 'com.example'
// Generates: package = "com.example"
```

### Maven with Kotlin (`pom.xml`)

```xml
<project>
    <groupId>com.example</groupId>
    <artifactId>kotlin-app</artifactId>
</project>
<!-- Generates: package = "com.example.kotlin-app" -->
```

## Generated Code Overview

Running `flash generate` produces:

### Models File

All models are generated in a single `Models.kt` file:

**`src/main/kotlin/com/myapp/db/Models.kt`**:
```kotlin
package com.myapp.db

import java.util.UUID
import java.time.LocalDateTime

data class Users(
    val id: UUID?,
    val name: String?,
    val email: String?,
    val createdAt: LocalDateTime?,
    val updatedAt: LocalDateTime?
)

enum class UserRole {
    ADMIN,
    USER,
    MODERATOR
}
```

### Per-Query-File Query Classes

Each `.sql` query file generates a dedicated query class:

**`src/main/kotlin/com/myapp/db/Users.kt`**:
```kotlin
package com.myapp.db

import java.sql.Connection
import java.sql.PreparedStatement

class UsersQueries(private val conn: Connection) {
    private val stmts = mutableMapOf<String, PreparedStatement>()

    fun close() { stmts.values.forEach { it.close() }; stmts.clear() }

    fun getUser(id: Int): Users? {
        val sql = """SELECT id, name, email FROM users WHERE id = ?"""
        val stmt = stmts.getOrPut("getUser") { conn.prepareStatement(sql) }
        stmt.setInt(1, id)
        stmt.executeQuery().use { rs ->
            return if (rs.next()) Users(rs.getObject("id", java.util.UUID::class.java), rs.getString("name"), rs.getString("email"), rs.getTimestamp("created_at")?.toLocalDateTime(), rs.getTimestamp("updated_at")?.toLocalDateTime()) else null
        }
    }
}
```

### Unified Query Interface

**`src/main/kotlin/com/myapp/db/Queries.kt`**:
```kotlin
package com.myapp.db

import java.sql.Connection

class Queries(private val conn: Connection) {
    private val users = UsersQueries(conn)

    companion object {
        fun newq(conn: Connection): Queries = Queries(conn)
    }

    fun getUser(id: Int): Users? = users.getUser(id)

    fun close() { users.close() }
}
```

## Driver Options

### JDBC (Default)

Default driver for standard JVM database access. Uses `java.sql.Connection` and `PreparedStatement` with statement caching.

**Best for:** Most applications, Spring Boot with JDBC, raw JDBC.

```toml
[gen.kotlin]
enabled = true
driver = "jdbc"
```

### Exposed

Generates code that uses JetBrains Exposed ORM's `Database` object and `exec()` method.

**Best for:** Projects already using Exposed ORM.

```toml
[gen.kotlin]
enabled = true
driver = "exposed"
```

Generated code wraps queries in a transaction:

```kotlin
import org.jetbrains.exposed.sql.Database

class UsersQueries(private val db: Database) {
    fun getUser(id: Int): Users? = transaction(db) {
        exec(sql, args = listOf(id)) { rs ->
            if (rs.next()) Users(/* ... */) else null
        }
    }
}
```

### R2DBC

Generates reactive code compatible with R2DBC `Connection`.

**Best for:** Reactive/WebFlux applications.

```toml
[gen.kotlin]
enabled = true
driver = "r2dbc"
```

Generated code uses `createStatement()` with bind parameters:

```kotlin
import io.r2dbc.spi.Connection

class UsersQueries(private val conn: Connection) {
    fun getUser(id: Int): Users? {
        val stmt = conn.createStatement(sql)
        stmt.bind(0, id)
        // Subscribe to stmt.execute() with reactor/coroutine bridge
        return null // Replace with reactive mapping
    }
}
```

> **Note:** R2DBC generates a blocking stub that you should bridge to your reactive framework (Project Reactor or Kotlin coroutines).

## Working with JDBC

### Using HikariCP Connection Pool

```kotlin
import com.zaxxer.hikari.HikariConfig
import com.zaxxer.hikari.HikariDataSource
import com.myapp.db.Queries

fun main() {
    val config = HikariConfig().apply {
        jdbcUrl = System.getenv("DATABASE_URL")
    }
    val ds = HikariDataSource(config)

    ds.connection.use { conn ->
        val db = Queries.newq(conn)
        val user = db.getUser(42)
        println(user)
    }
}
```

### Spring Boot Integration

```kotlin
import org.springframework.stereotype.Repository
import com.myapp.db.Queries
import javax.sql.DataSource

@Repository
class UserRepository(private val ds: DataSource) {
    fun findById(id: Int): Users? =
        ds.connection.use { conn ->
            Queries.newq(conn).getUser(id)
        }
}
```

## Working with Exposed

```kotlin
import org.jetbrains.exposed.sql.Database
import com.myapp.db.Queries

fun main() {
    val db = Database.connect(System.getenv("DATABASE_URL"))
    val queries = Queries.newq(db)
    val user = queries.getUser(42)
    println(user)
}
```

## Working with R2DBC

```kotlin
import io.r2dbc.spi.ConnectionFactories
import com.myapp.db.Queries

fun main() {
    val connectionFactory = ConnectionFactories.get(System.getenv("DATABASE_URL"))
    val mono = MonoSingle(connectionFactory.create()) { conn ->
        val queries = Queries.newq(conn)
        queries.getUser(42)
    }
}
```

## ScyllaDB / Cassandra Support

For ScyllaDB or Cassandra (CQL), the Kotlin generator produces code using the **DataStax Java Driver** (`CqlSession`).

### Configuration

```toml
[database]
provider = "scylla"
url_env = "DATABASE_URL"

[gen.kotlin]
enabled = true
out = "src/main/kotlin/com/myapp/db"
```

### Generated CQL Model

```kotlin
package com.myapp.db

import java.util.UUID
import java.time.LocalDateTime

data class Messages(
    val channelId: UUID?,
    val id: UUID?,
    val authorId: UUID?,
    val content: String?,
    val type: Int?,
    val mentionUserIds: Set<UUID>?,
    val createdAt: LocalDateTime?
)
```

### Usage with CqlSession

```kotlin
import com.datastax.oss.driver.api.core.CqlSession
import com.myapp.db.Queries

fun main() {
    CqlSession.builder().build().use { session ->
        val db = Queries.newq(session)
        val message = db.getMessage(channelId, messageId)
        println(message)
    }
}
```

## Schema Definition

Define your database schema in `db/schema/schema.sql`:

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    role user_role NOT NULL DEFAULT 'user',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE user_role AS ENUM ('admin', 'user', 'moderator');
```

## Writing Queries

Write parameterized queries in `.sql` files using annotations:

**`db/queries/users.sql`**:
```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT id, name, email FROM users ORDER BY name;

-- name: CreateUser :exec
INSERT INTO users (id, name, email, role) VALUES (?, ?, ?, ?);

-- name: CountUsersByRole :one
SELECT COUNT(*) FROM users WHERE role = ?;
```

Annotations:
- `:one` — Returns a single row (or null)
- `:many` — Returns a list of rows
- `:exec` — Returns affected row count

## Building and Running

### Gradle Kotlin DSL

```kotlin
dependencies {
    implementation("com.zaxxer:HikariCP:5.1.0")
    implementation("org.jetbrains.exposed:exposed-core:0.44.1") // for Exposed driver
    implementation("io.r2dbc:r2dbc-spi:1.0.0.RELEASE")          // for R2DBC driver
}
```

### Maven

```xml
<dependency>
    <groupId>com.zaxxer</groupId>
    <artifactId>HikariCP</artifactId>
    <version>5.1.0</version>
</dependency>
```

### Run

```bash
# Generate code
flash generate

# Build with Gradle
gradle build

# Build with Maven
mvn compile
```

## Best Practices

1. **Use nullable types** — Generated code uses `Type?` for all fields since database values may be null
2. **Place generated code in your source tree** — Set `out = "src/main/kotlin/com/myapp/db"` for Gradle/Maven
3. **Use `package` config** — Always set the package explicitly to match your project structure
4. **Use HikariCP for JDBC** — Connection pooling is essential for production
5. **Kotlin Coroutines with R2DBC** — Bridge generated stubs to kotlinx.coroutines for reactive flows
6. **Close resources** — Use `.use {}` blocks or `close()` on your Queries instance
7. **Run `flash generate` after schema changes** — Regenerate whenever you modify your schema or queries
