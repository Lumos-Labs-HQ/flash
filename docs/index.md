---
layout: home

hero:
  name: "Flash ORM"
  text: "Lightning-Fast Database ORM"
  tagline: A powerful, database-agnostic ORM built in Go with Prisma-like functionality and blazing performance
  image:
    src: /logo.png
    alt: Flash ORM
  actions:
    - theme: brand
      text: Get Started
      link: /getting-started
    - theme: alt
      text: Try Studio
      link: /concepts/studio
    - theme: alt
      text: View on GitHub
      link: https://github.com/SpreadSheets600/flash-orm

features:
  - icon: 🗃️
    title: Multi-Database Support
    details: PostgreSQL, MySQL, SQLite, and MongoDB support with a unified API. Switch databases without rewriting code.
    link: /concepts/schema
    linkText: Learn about schemas

  - icon: ⚡
    title: Blazing Fast Performance
    details: Outperforms Drizzle and Prisma by up to 10x in benchmarks. Optimized for real-world workloads.
    link: /advanced/performance
    linkText: View benchmarks

  - icon: 🔄
    title: Smart Migrations
    details: Transaction-based migration system with automatic rollback, conflict detection, and branch-aware management.
    link: /concepts/migrations
    linkText: Explore migrations

  - icon: 🎯
    title: Type-Safe Code Generation
    details: Generate type-safe code for Go, TypeScript/JavaScript, and Python with full IDE autocomplete support.
    link: /concepts/code-generation
    linkText: See code generation

  - icon: 📖
    title: Complete Examples
    details: Copy-paste ready examples for every feature — CRUD, relationships, migrations, seeding, and full workflows.
    link: /examples/
    linkText: Browse examples

  - icon: 📊
    title: Visual Database Studio
    details: FlashORM Studio provides a visual interface for managing your database, editing data, and creating migrations.
    link: /concepts/studio
    linkText: Try the studio

  - icon: 🌿
    title: Git-like Branching
    details: Manage database schema changes across branches like you manage code. Merge, diff, and resolve conflicts.
    link: /concepts/branching
    linkText: Learn branching

  - icon: 📤
    title: Smart Export System
    details: Export your data to JSON, CSV, or SQLite with automatic relationship handling and filtering.
    link: /concepts/export
    linkText: Export data

  - icon: 🔍
    title: Schema Introspection
    details: Pull schema from existing databases and generate migrations automatically. Perfect for legacy projects.
    link: /advanced/how-it-works
    linkText: How it works

  - icon: 🛡️
    title: Safe by Default
    details: Automatic conflict detection, transaction-based operations, and comprehensive validation keep your data safe.
    link: /advanced/how-it-works
    linkText: Safety features

  - icon: 🟢
    title: Node.js First-Class Support
    details: Native JavaScript/TypeScript support with async/await and full type definitions.
    link: /guides/typescript
    linkText: TypeScript guide

  - icon: 🐍
    title: Python Ready
    details: Full Python support with async operations and Pythonic API design.
    link: /guides/python
    linkText: Python guide

  - icon: 🔌
    title: Extensible Plugin System
    details: Extend FlashORM with plugins for custom functionality and integrations.
    link: /advanced/plugins
    linkText: Plugin system
---
