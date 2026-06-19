# flash 2.5.1

**Release Date:** 2026-06-19

## 🔧 Bug Fixes

### ScyllaDB / CQL Studio

- **SQL Editor now renders correctly** — Fixed blank editor caused by a JS syntax error (`/**` comment stripped) that prevented `SqlHints` from loading, and added `EditorView.theme()` to ensure proper height and text color.

- **`USE keyspace` works correctly** — Multi-statement input (`USE ndiscord; SELECT * FROM messages`) is now split into individual statements and executed sequentially. The `USE` statement properly switches the active keyspace on the adapter, and subsequent queries use that keyspace automatically.

- **Auto-qualify unqualified table names** — Queries like `SELECT * FROM users` are automatically rewritten to `SELECT * FROM ks.users` using the active keyspace, so users don't need to qualify every table name.

- **No double-qualification** — `SELECT * FROM ndiscord.messages` is left unchanged; only unqualified references are prefixed.

- **CQL autocomplete — keyspace and table suggestions** — Schema cache now stores only `ks.table` qualified keys (no plain-name collisions between keyspaces). Tables from all user keyspaces are fetched from `system_schema.columns`.

- **`FROM` / `JOIN` now suggests keyspace names first** — After `SELECT * FROM `, typing suggests keyspace names so users can type `ndiscord.` to get table completions via dot-completion.

- **Dot-completion triggers immediately** — Typing `ndiscord.` now immediately shows tables in that keyspace without requiring an additional character.

- **`SELECT` keyword shown at statement start** — Statement-level keywords (`SELECT`, `INSERT`, `USE`, etc.) now appear as suggestions when the cursor is at the start of a line.

- **Context-aware completions** — Autocomplete rewritten with a clean `switch` on context keyword: `FROM/JOIN` → tables/keyspaces, `SELECT` → columns + `*`, `WHERE/AND/OR` → columns, `USE` → keyspaces only, `ORDER/GROUP` → `BY`.

- **Edit / Save / Add Row fixed for CQL tables** — `ValidateIdentifier` was rejecting `ks.table` names. All table name validations now use `ValidateQualifiedIdentifier`.

- **Correct SQL quoting for CQL** — `quote("ndiscord.messages")` now produces `"ndiscord"."messages"` instead of `"ndiscord.messages"`.

- **System keyspaces filtered** — `system`, `system_schema`, `system_auth`, `system_distributed`, `system_traces`, and DSE internal keyspaces are excluded from keyspace completions and metrics.

- **CQL metrics improved** — Metrics now query all user keyspaces from `system_schema.tables`, show per-table row counts with correct `"ks"."table"` quoting, and display active keyspace count.

- **"Query executed successfully" shown for SELECT** — `getQueryType` now detects query type from the last statement in multi-statement input, so `USE ks; SELECT * FROM table` correctly displays the result grid.

- **Autocomplete font size increased** — Completion popup text size bumped from 12px to 14px for better readability.

---

# flash 2.5.0
