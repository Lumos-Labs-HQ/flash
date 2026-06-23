---
title: Java Guide
description: Complete guide to using Flash ORM with Java
---

# Flash ORM - Java Usage Guide

A comprehensive guide to using Flash ORM with Java projects, featuring JDBC, jOOQ, and Hibernate driver support.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Package Auto-Detection](#package-auto-detection)
- [Generated Code Overview](#generated-code-overview)
- [Driver Options](#driver-options)
- [Working with JDBC](#working-with-jdbc)
- [Working with jOOQ](#working-with-jooq)
- [Working with Hibernate](#working-with-hibernate)
- [ScyllaDB / Cassandra Support](#scylladb--cassandra-support)
- [Schema Definition](#schema-definition)
- [Writing Queries](#writing-queries)
- [Building and Running](#building-and-running)
- [Best Practices](#best-practices)

## Prerequisites

- **Java 17+** (records require Java 16+)
- **Maven** or **Gradle** build system
- **Flash ORM CLI** installed via npm, pip, or direct binary

## Quick Start

### 1. Initialize a Java Project

```bash
# Initialize with PostgreSQL
flash init --postgresql

# Flash ORM auto-detects Maven/Gradle and configures the Java generator
```

### 2. Generate Code

```bash
# After defining schema and queries:
flash generate
```

### 3. Configure Your `flash.toml`

```toml
[gen.java]
enabled = true
out = "src/main/java/com/myapp/db"
package = "com.myapp.db"
driver = "jdbc"
```

## Configuration

### Flash TOML Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `enabled` | `false` | Enable Java code generation |
| `out` | `flash_gen` | Output directory for generated Java files |
| `package` | derived from `out` dir | Java package name (e.g., `com.example.db`) |
| `driver` | `jdbc` | Driver type: `jdbc`, `jooq`, or `hibernate` |

### Driver-Specific Config

```toml
[gen.java]
enabled = true
out = "src/main/java/com/myapp/db"
package = "com.myapp.db"
driver = "jdbc"
```

## Package Auto-Detection

Flash ORM automatically detects your Java package from build files during `flash init`:

### Maven (`pom.xml`)

The package is derived from `<groupId>.<artifactId>`:

```xml
<project>
    <groupId>com.example</groupId>
    <artifactId>my-service</artifactId>
</project>
<!-- Generates: package = "com.example.my-service" -->
```

### Gradle (`build.gradle`)

The package is derived from the `group` property:

```groovy
group = 'org.myapp'
// Generates: package = "org.myapp"
```

## Generated Code Overview

Running `flash generate` produces:

### Per-Table Model Files

Each table gets its own file as a Java `record`:

**`src/main/java/com/myapp/db/Users.java`**:
```java
package com.myapp.db;

import java.util.UUID;
import java.time.LocalDateTime;

public record Users(
    UUID id,
    String name,
    String email,
    LocalDateTime createdAt,
    LocalDateTime updatedAt
) {}
```

### Per-Enum Enum Files

Each enum gets its own file:

**`src/main/java/com/myapp/db/UserRole.java`**:
```java
package com.myapp.db;

public enum UserRole {
    ADMIN,
    USER,
    MODERATOR;
}
```

### Per-Query-File Query Classes

Each `.sql` query file generates a dedicated query class:

**`src/main/java/com/myapp/db/UsersQueries.java`**:
```java
package com.myapp.db;

import java.sql.Connection;
import java.sql.PreparedStatement;

public class UsersQueries {
    private final Connection conn;
    private final java.util.Map<String, PreparedStatement> stmts = new java.util.HashMap<>();

    public UsersQueries(Connection conn) { this.conn = conn; }

    public Users getUser(int id) throws java.sql.SQLException {
        // ... generated method from SQL query
    }

    public void close() throws java.sql.SQLException {
        for (var s : stmts.values()) s.close();
        stmts.clear();
    }
}
```

### Unified Query Interface

**`src/main/java/com/myapp/db/Queries.java`**:
```java
package com.myapp.db;

import java.sql.Connection;

public class Queries {
    private final UsersQueries users;

    Queries(Connection conn) {
        this.users = new UsersQueries(conn);
    }

    public static Queries newq(Connection conn) {
        return new Queries(conn);
    }

    public Users getUser(int id) throws java.sql.SQLException {
        return this.users.getUser(id);
    }

    public void close() throws java.sql.SQLException {
        this.users.close();
    }
}
```

## Driver Options

### JDBC (Default)

Default driver for standard Java database access. Uses `java.sql.Connection` and `PreparedStatement` with statement caching.

**Best for:** Most applications, Spring Boot with JDBC template, raw JDBC.

```toml
[gen.java]
enabled = true
driver = "jdbc"
```

### jOOQ

Generates code compatible with the jOOQ `DSLContext` API.

**Best for:** Existing jOOQ projects or teams wanting typesafe SQL builder.

```toml
[gen.java]
enabled = true
driver = "jooq"
```

Generated code:
```java
import org.jooq.DSLContext;

public Users getUser(DSLContext ctx, int id) throws java.sql.SQLException {
    final String sql = """
            SELECT id, name, email FROM users WHERE id = ?
            """;
    return ctx.fetchOne(sql, id).into(Users.class);
}
```

### Hibernate

Generates code using Jakarta Persistence `EntityManager`.

**Best for:** Existing Hibernate/JPA projects.

```toml
[gen.java]
enabled = true
driver = "hibernate"
```

Generated code:
```java
import jakarta.persistence.EntityManager;

public Users getUser(EntityManager em, int id) throws java.sql.SQLException {
    var q = em.createNativeQuery(sql, Users.class);
    q.setParameter(1, id);
    return (Users) q.getSingleResult();
}
```

## Working with JDBC

### Creating a Connection

```java
import java.sql.Connection;
import java.sql.DriverManager;
import com.myapp.db.Queries;

public class Main {
    public static void main(String[] args) throws Exception {
        var conn = DriverManager.getenv("DATABASE_URL");
        var db = Queries.newq(conn);

        // Query methods are generated based on your SQL files
        var user = db.getUser(42);
        System.out.println(user.name());

        db.close();
        conn.close();
    }
}
```

### Connection Pooling with HikariCP

```java
import com.zaxxer.hikari.HikariConfig;
import com.zaxxer.hikari.HikariDataSource;
import com.myapp.db.Queries;

HikariConfig config = new HikariConfig();
config.setJdbcUrl(System.getenv("DATABASE_URL"));
var ds = new HikariDataSource(config);

try (var conn = ds.getConnection()) {
    var db = Queries.newq(conn);
    var users = db.listUsers();
    users.forEach(u -> System.out.println(u.name()));
}
```

## Working with jOOQ

```java
import org.jooq.DSLContext;
import org.jooq.impl.DSL;
import com.myapp.db.Queries;

var conn = DriverManager.getConnection(System.getenv("DATABASE_URL"));
DSLContext ctx = DSL.using(conn);
var db = Queries.newq(ctx);

var user = db.getUser(42);
System.out.println(user.name());
```

## Working with Hibernate

```java
import jakarta.persistence.EntityManager;
import jakarta.persistence.EntityManagerFactory;
import jakarta.persistence.Persistence;
import com.myapp.db.Queries;

EntityManagerFactory emf = Persistence.createEntityManagerFactory("my-pu");
EntityManager em = emf.createEntityManager();
var db = Queries.newq(em);

var user = db.getUser(42);
System.out.println(user.name());
```

## ScyllaDB / Cassandra Support

For ScyllaDB or Cassandra (CQL), the Java generator produces code using the **DataStax Java Driver** (`CqlSession`).

### Configuration

```toml
[database]
provider = "scylla"
url_env = "DATABASE_URL"

[gen.java]
enabled = true
out = "src/main/java/com/myapp/db"
```

### Generated CQL Model

```java
package com.myapp.db;

import java.util.UUID;
import java.time.LocalDateTime;
import java.util.Set;

public record Messages(
    UUID channelId,
    UUID id,
    UUID authorId,
    String content,
    int type,
    Set<UUID> mentionUserIds,
    LocalDateTime createdAt
) {}
```

### Usage with CqlSession

```java
import com.datastax.oss.driver.api.core.CqlSession;
import com.myapp.db.Queries;

try (var session = CqlSession.builder().build()) {
    var db = Queries.newq(session);
    var message = db.getMessage(channelId, messageId);
    System.out.println(message.content());
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

### Maven

Add to `pom.xml`:
```xml
<dependency>
    <groupId>com.zaxxer</groupId>
    <artifactId>HikariCP</artifactId>
    <version>5.1.0</version>
</dependency>
```

### Gradle

```kotlin
dependencies {
    implementation("com.zaxxer:HikariCP:5.1.0")
}
```

### Run

```bash
# Generate code
flash generate

# Build with Maven
mvn compile

# Build with Gradle
gradle build
```

## Best Practices

1. **Use Maven/Gradle for dependencies** — Let the build system manage JDBC drivers and connection pooling
2. **Place generated code in your source tree** — Set `out = "src/main/java/com/myapp/db"` for Maven/Gradle
3. **Use `package` config** — Always set the package explicitly in `flash.toml` to avoid issues
4. **Use connection pooling** — Wrap connections with HikariCP for production
5. **Close resources** — Always call `db.close()` on your Queries instance
6. **jOOQ for complex queries** — Use jOOQ driver if you need dynamic query building alongside generated code
7. **Hibernate for existing JPA projects** — Use Hibernate driver if you already use EntityManager
8. **Run `flash generate` after schema changes** — Regenerate code whenever you modify your schema
