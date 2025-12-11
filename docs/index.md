---
layout: home

hero:
  name: "Flash ORM"
  text: "Lightning-Fast Database ORM"
  tagline: A powerful, database-agnostic ORM built in Go with Prisma-like functionality and blazing performance
  image:
    src: /hero-image.png
    alt: Flash ORM
  actions:
    - theme: brand
      text: Get Started
      link: /getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/Lumos-Labs-HQ/flash

features:
  - icon: 🗃️
    title: Multi-Database Support
    details: PostgreSQL, MySQL, SQLite, and MongoDB support with a unified API. Switch databases without rewriting code.

  - icon: ⚡
    title: Blazing Fast Performance
    details: Outperforms Drizzle and Prisma by up to 10x in benchmarks. Optimized for real-world workloads.

  - icon: 🔄
    title: Smart Migrations
    details: Transaction-based migration system with automatic rollback, conflict detection, and branch-aware management.

  - icon: 🎯
    title: Type-Safe Code Generation
    details: Generate type-safe code for Go, TypeScript/JavaScript, and Python with full IDE autocomplete support.

  - icon: 📊
    title: Visual Database Studio
    details: FlashORM Studio provides a visual interface for managing your database, editing data, and creating migrations.

  - icon: 🌿
    title: Git-like Branching
    details: Manage database schema changes across branches like you manage code. Merge, diff, and resolve conflicts.

  - icon: 📤
    title: Smart Export System
    details: Export your data to JSON, CSV, or SQLite with automatic relationship handling and filtering.

  - icon: 🔍
    title: Schema Introspection
    details: Pull schema from existing databases and generate migrations automatically. Perfect for legacy projects.

  - icon: 🛡️
    title: Safe by Default
    details: Automatic conflict detection, transaction-based operations, and comprehensive validation keep your data safe.

  - icon: 🟢
    title: Node.js First-Class Support
    details: Native JavaScript/TypeScript support with async/await and full type definitions.

  - icon: 🐍
    title: Python Ready
    details: Full Python support with async operations and Pythonic API design.

  - icon: 🔌
    title: Extensible Plugin System
    details: Extend FlashORM with plugins for custom functionality and integrations.
---

## Quick Start

Get up and running in minutes:

::: code-group

```bash [npm]
# Install globally
npm install -g flashorm

# Initialize your project
flash init

# Create your schema
# Edit db/schema/schema.sql

# Generate migrations
flash migrate

# Generate code
flash gen
```

```bash [Python]
# Install via pip
pip install flashorm

# Initialize your project
flash init

# Create your schema
# Edit db/schema/schema.sql

# Generate migrations
flash migrate

# Generate code
flash gen
```

```bash [Go]
# Install Flash ORM
go install github.com/Lumos-Labs-HQ/flash@latest

# Initialize your project
flash init

# Create your schema
# Edit db/schema/schema.sql

# Generate migrations
flash migrate

# Generate code
flash gen
```

:::

## Performance That Matters

FlashORM delivers exceptional performance in real-world scenarios:

| Operation | FlashORM | Drizzle | Prisma |
|-----------|----------|---------|--------|
| Insert 1000 Users | **149ms** | 224ms | 230ms |
| Complex Query x500 | **3156ms** | 12500ms | 56322ms |
| Mixed Workload x1000 | **186ms** | 1174ms | 10863ms |
| **Total Time** | **5980ms** | **17149ms** | **71510ms** |

::: tip
FlashORM is **2.8x faster** than Drizzle and **11.9x faster** than Prisma in comprehensive benchmarks.
:::

## Why Flash ORM?

<div class="feature-grid">

### 🎨 Familiar Developer Experience
If you've used Prisma, you'll feel right at home. FlashORM provides a similar CLI and workflow while adding powerful features.

### 🚀 Built for Production
Transaction-based migrations, automatic conflict detection, and comprehensive error handling make FlashORM production-ready out of the box.

### 🔧 Flexible & Powerful
From simple CRUD operations to complex queries, schema introspection, and visual database management - FlashORM handles it all.

### 🌍 Multi-Language Support
Write your backend in Go, TypeScript, or Python. FlashORM generates idiomatic, type-safe code for all three.

</div>

## What's Next?

<div class="vp-doc">

- **[Getting Started](/getting-started)** - Set up your first Flash ORM project
- **[Go Guide](/guides/go)** - Learn Flash ORM with Go
- **[TypeScript Guide](/guides/typescript)** - Learn Flash ORM with TypeScript
- **[Python Guide](/guides/python)** - Learn Flash ORM with Python
- **[Core Concepts](/concepts/schema)** - Understand how Flash ORM works

</div>

<style>
.feature-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1.5rem;
  margin: 2rem 0;
}

.feature-grid > div {
  padding: 1.5rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  transition: border-color 0.2s;
}

.feature-grid > div:hover {
  border-color: var(--vp-c-brand);
}

.feature-grid h3 {
  margin-top: 0;
}
</style>
