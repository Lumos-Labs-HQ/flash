# FlashORM Release Notes

## Version 2.4.0 — Latest Release

### Plugin System

The plugin architecture has been redesigned:

- **core** — ORM, migrations, and seeding. Installs automatically the first time any ORM command is run; no manual setup required.
- **studio** — Visual database editor. Optional, installed manually when needed.

The `all` plugin has been removed. A new `flash update` command updates all installed plugins. Use `--self` to also update the flash binary, or `--self-only` to update only the binary.

### SQL Studio — Export & Import

Export and import are now available directly from the SQL Studio interface. Three export modes are supported: Schema Only, Data Only, and Complete (schema + data).

**Performance**
- The full database schema is now fetched in a single query at the start of export and reused throughout, eliminating one query per table for schema introspection.
- Data export no longer issues a row count query before fetching — rows are paged directly until exhausted, removing an extra round-trip per table.

**User Experience**
- A full-screen progress overlay appears during export and import with live status messages at each stage.
- A progress bar transitions from an animated state while the server is working to a percentage fill as each phase completes.
- An accurate summary is shown on import completion — tables created, rows inserted, and any errors.

### Dependency Reduction

- Fiber framework removed — replaced with stdlib `net/http`
- Viper removed — replaced with stdlib `encoding/json`
- lib/pq removed — pgx/v5 is now the sole PostgreSQL driver
- mapstructure removed — plain `json` struct tags used throughout
- Approximately 8 fewer transitive dependencies; smaller binary size

### Bug Fixes

- Fixed `down` command missing from plugin registry
- Fixed CSS duplication across studio static assets

---

For detailed documentation, see:
- [Usage Guide — Go](docs/USAGE_GO.md)
- [Usage Guide — TypeScript](docs/USAGE_TYPESCRIPT.md)
- [Usage Guide — Python](docs/USAGE_PYTHON.md)
- [Contributing](docs/contributing.md)
