---
intent: pcc
updated_at: 2026-02-22
---

# 01 — Architecture

## Stack
| Layer | Tech |
|---|---|
| Framework | Wails v2 (Go + WKWebView/WebView2) |
| Backend | Go 1.24, SQLite via `modernc.org/sqlite` (pure Go, no CGo) |
| Frontend | React 18 + TypeScript + Vite 3 |
| Rich text | TipTap v3.7.2 (ProseMirror) |
| Styling | CSS custom properties (`--term-*`); Tailwind/shadcn migration planned |

## Key Packages
- `main.go` — Wails entry point, window config
- `app.go` — App struct; all Wails-bound Go methods (612 lines)
- `store/store.go` — All DB logic: CRUD, search, migrations, FTS5 (751 lines)
- `store/models.go` — `Item`, `SearchRequest` structs
- `store/schema.sql` — Embedded SQL; FTS5 via `todos_fts`; schema version **4**
- `config/config.go` — OS config paths; `Config{Vaults[], ActiveVaultID, InboxPath, APIEnabled, APIPort, APIKey}`
- `vault/manager.go` — `vault.Manager`: multi-vault open/close/switch, inbox store isolation
- `api.go` — HTTP API server: Bearer auth, `/items`, `/search`, `/inbox`, `/vaults`
- `parse/line.go` — todo.txt-inspired line parser

## Data Model
Table: `todos` (DB name unchanged). Core fields:
- `id`, `title`, `priority`, `completed`, `archived`, `created_at`, `updated_at`
- `section` (now|soon|anytime), `type` (todo|note), `inbox` (INTEGER DEFAULT 0)
- `notes_md` (HTML/TipTap output), `source_line`, `pinned`, `api_source`
- `due_at`, `threshold_at`, `recurrence_rule`

Taxonomy: join tables `todo_projects`, `todo_contexts`, `todo_tags`
References: `refs(source_id, target_id)` — inter-item wikilinks
FTS5: `todos_fts(title, notes_md)` — content-synced via triggers

Schema version: **4** (inbox column). Migrations in `store/store.go` migration slice.

## Vault Architecture
- `config.Vault{ID, Name, Path, CreatedAt}` — registry entry
- `config.Config.Vaults[]` — all registered vaults
- `config.Config.ActiveVaultID` — current vault
- `config.Config.InboxPath` — shared inbox DB (separate from all vaults)
- `vault.Manager` — opens active vault + inbox on startup; `ActiveStore()`, `InboxStore()`
- Each vault is an independent SQLite DB in a user-chosen directory

## Frontend Architecture
- `src/pages/App.tsx` — single source of truth for all app state (~large file)
- `frontend/wailsjs/` — auto-generated Wails TS bindings (never edit manually)
- `src/components/` — modal and UI components (props: open/close state + callbacks)
- `src/lib/` — utilities, TipTap extensions
- `src/theme/` — ThemeProvider, `TermTheme`, CSS vars

## HTTP API (api.go)
- Port: 7765 (default), Bearer auth via `APIKey`
- Routes: `GET /items`, `POST /items`, `PUT /items/:id`, `DELETE /items/:id`
- Routes: `GET /search`, `GET /inbox`, `POST /inbox/:id/process`, `GET /vaults`
- Enabled/disabled via `cfg.APIEnabled` toggle

## Evidence
- Inspected: app.go, store/models.go, store/schema.sql, vault/manager.go, config/config.go, api.go
- Updated: 2026-02-22
