# FlashORM Release Notes

## Version 2.4.3

**SQL Studio — UI Overhaul & Bug Fixes**

### New
- **Data viewer redesign** — Neon-style table grid with monospace font, sticky typed column headers (`col_name type`), 38px rows, solid grid lines
- **Resizable columns** — drag any column header edge to resize; persists per table
- **Cell popup editor** — click any cell to open a floating editor (Ctrl+Enter saves, Esc cancels, Set NULL hidden for NOT NULL columns)
- **Empty table schema view** — tables with no rows now show column schema as a grid (name, type, nullable, constraints)
- **SQL editor redesign** — matches data page dark theme; results table uses same grid with resizable columns and value type colors
- **Schema page** — navbar, topbar, edit panel and buttons updated to consistent dark theme
- **Filter panel** — redesigned as a floating dark card with per-row bordered inputs
- **FK navigation** — clicking a foreign key value opens a wider popup; "Go to table" now auto-applies the filter

### Fixed
- PostgreSQL `$0` placeholder bug — all SQL parameters now correctly 1-based (`$1`, `$2`, ...)
- Server-side filters were silently failing on PostgreSQL due to `$0` placeholder
- Set NULL on NOT NULL columns now blocked at both UI (button hidden) and backend (pre-check before DB hit)
- Real DB error messages shown for save/SQL failures instead of generic "internal error"
- Toast notifications forced to bottom-right with correct green/red colors
- Import: enum type casing mismatch (Prisma uppercase vs PostgreSQL lowercase)
- Import: composite primary keys on junction tables
- Import: `_TEXT → TEXT[]` array type normalization, `ARRAY[]` default casting
- Import: float64 scientific notation (`1.487365e+06`) for INT columns


---
