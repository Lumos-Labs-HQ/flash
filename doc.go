// Flash is a powerful, database-agnostic ORM and migration CLI tool built in Go.
//
// It supports multiple databases — PostgreSQL, MySQL, SQLite, ScyllaDB / Apache Cassandra,
// and ClickHouse — and provides type-safe code generation for Go, TypeScript/JavaScript,
// and Python. FlashORM Studio gives you a visual web-based editor for all supported databases
// plus MongoDB and Redis.
//
// # Quick Start
//
//	flash init --postgresql
//	flash migrate "create users table"
//	flash apply
//	flash gen
//
// # Commands
//
// Project Setup:
//
//	flash init          — Initialize a new project (--postgresql, --mysql, --sqlite, --scylla, --clickhouse)
//
// Migration Lifecycle:
//
//	flash migrate       — Create a new migration from schema diffs
//	flash apply         — Apply pending migrations
//	flash down          — Rollback the last applied migration
//	flash status        — Show migration status and pending changes
//	flash reset         — Reset database to clean state
//	flash pull          — Extract schema from an existing database
//
// Code Generation:
//
//	flash gen           — Generate type-safe client code (Go, TypeScript, Python)
//
// Schema Branching (Git-like):
//
//	flash branch        — Manage database schema branches
//	flash checkout      — Switch between schema branches
//
// Data Operations:
//
//	flash seed          — Seed database with realistic fake data
//	flash export        — Export data (JSON, CSV, SQLite)
//	flash raw           — Execute raw SQL files or inline queries
//
// Visual Editor:
//
//	flash studio        — Launch FlashORM Studio (SQL, MongoDB, Redis)
//
// Plugins & Updates:
//
//	flash plugins       — List installed plugins
//	flash add-plug      — Install a plugin
//	flash rm-plug       — Remove a plugin
//	flash update        — Update Flash CLI and plugins to latest
//
// # Database Support
//
//	PostgreSQL          — Full ORM, migrations, codegen, Studio
//	MySQL               — Full ORM, migrations, codegen, Studio
//	SQLite              — Full ORM, migrations, codegen, Studio
//	ScyllaDB / Cassandra — ORM (beta), migrations, gocql codegen, Studio
//	ClickHouse          — ORM (beta), migrations, codegen, Studio
//	MongoDB             — Studio visual management only
//	Redis               — Studio visual management only
//
// # Installation
//
//	go install github.com/Lumos-Labs-HQ/flash@latest
//
// # Documentation
//
// Full docs: https://github.com/Lumos-Labs-HQ/flash
package main
