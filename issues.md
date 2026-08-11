# Filing an issue for FlashORM

Good issues get fixed fast. A vague one ("gen is broken") usually can't be acted
on at all. This template is the structure every FlashORM issue should follow — a
real title, a substantive description, and (for bugs) concrete reproduction steps.

> **Tip:** the CLI can generate a complete, pre-filled report for you — including
> version, OS, and database provider — and submit it or hand you a ready-to-post
> link. See the bottom of this file.

---

## Title

One specific line. Prefix it by kind so it's triageable at a glance:

- `[Bug] gen: RETURNING clause dropped for SQLite :one inserts`
- `[Feature] support composite cache keys in @cache dep purging`
- `[Docs] flash.toml cache.prefix default is undocumented`

Avoid: `bug`, `help`, `it doesn't work`, `test`.

---

## Description

What you were doing and what went wrong, in a sentence or three. Include the
command you ran and the language target if relevant (Go / TS / Python / Kotlin /
Java / Rust).

## Steps to Reproduce

Numbered, minimal, copy-pasteable:

```
1. flash init --postgresql
2. add the query below to db/queries/users.sql
3. flash gen
```

Include the smallest schema/query that triggers the problem:

```sql
-- name: GetUser :one
SELECT id, name FROM users WHERE id = $1;
```

## Expected behavior

What should have happened.

## Actual behavior

What actually happened. **Paste the exact error output** (verbatim, in a code
block) — not a paraphrase.

## Environment

| | |
|---|---|
| Flash version | `flash --version` |
| OS / arch | e.g. linux/amd64 |
| Go version | `go version` (if building from source) |
| DB provider | postgresql / mysql / sqlite / clickhouse / scylla |
| Cache enabled | yes / no |

## Additional context

Anything else — related config, a `flash.toml` excerpt, links, screenshots, a
branch that reproduces it.

---

## Generate this automatically

Instead of filling this out by hand, run:

```
flash issues \
  -k bug \
  -t "gen: RETURNING dropped for SQLite :one inserts" \
  -b "flash gen produces an insert helper that never returns the row." \
  --repro "1. flash init --sqlite  2. add an INSERT ... RETURNING :one query  3. flash gen" \
  --expected "the generated function returns the inserted row" \
  --actual "the function returns void; RETURNING is stripped from the SQL"
```

- `--print` composes the report and prints it **without submitting** (review first).
- `-y` submits non-interactively.
- Environment details are collected and attached automatically.

The command validates the report and refuses to file low-effort issues, so what
lands in the tracker is always actionable.
