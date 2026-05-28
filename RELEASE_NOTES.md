# FlashORM Release Notes

## Version 2.4.2

### Security Hardening — SQL Studio

This release hardens the SQL Studio component against SQL injection, unauthorized access, information leakage, and denial-of-service attacks.

#### Parameterized Queries

All data-mutating operations (INSERT, UPDATE, DELETE) and filtered SELECTs now use driver-level parameterized queries instead of string interpolation. SQL injection through the Studio API is no longer possible.

New methods added to `DatabaseAdapter`:
- `ExecuteQueryWithArgs(ctx, query, args...)` — SELECT with bound parameters
- `ExecuteDMLWithArgs(ctx, query, args...)` — INSERT/UPDATE/DELETE with bound parameters

#### Adapter-Aware Identifier Quoting

Identifiers (table and column names) are now quoted using the correct syntax for each database:
- PostgreSQL / SQLite → `"name"`
- MySQL → `` `name` ``

New method: `QuoteIdentifier(name string) string`

This fixes DDL generation that previously used hardcoded double-quotes, which broke on MySQL.

#### Authentication & Bind Address Control

Two new flags on `flash studio`:

```bash
--host string         Host to bind to (default "127.0.0.1")
--auth-token string   Bearer token for API authentication
```

Binding to `0.0.0.0` without `--auth-token` is refused at startup:

```bash
# Safe — local only, no auth needed
flash studio

# Network-accessible — token required
flash studio --host 0.0.0.0 --auth-token mysecrettoken
```

All API requests are validated against the token using constant-time comparison to prevent timing attacks.

#### CORS Restriction

Cross-origin requests are now restricted to `localhost` and `127.0.0.1` origins. Requests from external origins are blocked.

#### Request Body Size Limit

Request bodies are capped at 10 MB to prevent memory exhaustion from oversized payloads.

#### Input Validation

All table and column name inputs are validated before use in queries. Valid identifiers must match `[a-zA-Z_][a-zA-Z0-9_]*` and cannot exceed 128 characters. Dotted qualified names (`schema.table`) are also validated per segment.

#### Error Sanitization

Raw database error messages are no longer sent to clients. Internal errors are logged server-side; clients receive a generic category:

| Error contains | Client response |
|---|---|
| `connection`, `connect` | `"database connection error"` |
| `timeout` | `"request timed out"` |
| `permission`, `access denied` | `"permission denied"` |
| Everything else | `"internal error"` |

### Bug Fixes

- SQL syntax errors and constraint violations from the query runner now return HTTP 400 instead of 500
- MySQL identifier quoting now uses backticks instead of double-quotes

### Testing

- New unit tests for `sanitizeError`, `classifySQLError`, `ValidateIdentifier`, `ValidateQualifiedIdentifier`, `AuthMiddleware`, `CORSMiddleware`, `MaxBytesMiddleware`
- New adapter tests for `QuoteIdentifier`, `ProviderName`, `ExecuteQueryWithArgs`, `ExecuteDMLWithArgs` on SQLite, PostgreSQL, and MySQL
- New integration tests for Studio auth token enforcement and the `0.0.0.0` bind guard

### Upgrade Notes

No breaking changes. Default behavior is unchanged — Studio still binds to `127.0.0.1` with no auth required for local use.

---

