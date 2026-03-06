---
intent: pcc_global
project: nanite
updated_at: "2026-03-04"
---

# Architectural Decisions

No formal ADRs recorded yet. Key design decisions documented in CLAUDE.md:

## Inferred design choices

- **Wails v2 framework:** Go backend + WebView frontend; under evaluation for v3 or alternative
- **Pure Go SQLite (modernc.org/sqlite):** no CGo dependency for cross-platform builds
- **Single-file app state:** all Wails-exposed methods in `app.go`; all DB logic in `store/store.go`
- **Schema migration pattern:** embedded `schema.sql` for fresh installs + migration slice in `store.go` for existing users; idempotent guards required
- **DB table name retained:** SQLite table is still `todos` despite Go struct rename to `Item`; renaming deferred due to migration complexity
- **CSS custom properties over Tailwind:** `--term-*` variable system for terminal aesthetic; Tailwind/shadcn migration planned but not started
- **Theme switcher hidden:** UI tab hidden until Tailwind migration complete; theme system intact in code
- **Inbox as first-class concept:** blank-title items auto-route to inbox; dedicated view with batch operations
- **EditItemModal close flow:** `requestClose()` with dirty detection and configurable close behavior (`ask`/`always`/`never`)
- **REST API localhost-only:** bound to `127.0.0.1` with bearer-token auth for external integrations
- **Multi-vault support:** each vault has separate SQLite database; vault manager handles selection

## Evidence
- Last refreshed: 2026-03-04 (mentat PCC bootstrap)
- Sources: CLAUDE.md, app.go, store/store.go, api.go, vault/manager.go
