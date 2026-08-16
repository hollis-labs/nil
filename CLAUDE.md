# NIL

## Session bootstrap
- `.agentrc/` and `.forge/` (the old planning-doc conventions this section used to point at) were retired 2026-08-16 — those files no longer exist. Do not look for them.
- Per-project source of truth is now `.agent-ops/project.yaml` (machine-readable summary: purpose, build command, doc pointers, roadmap phase) plus `AGENTS.md` (narrative: entry points, domain concepts, "where to look" guidance). Read both when starting a session on this repo.
- Live task/plan tracking is in Torque, project `PRJ-20260417-0005` ("Nil") — not in checklists in this file. See "Immediate Priorities" below for the pointer.
- Do not guess when uncertain. Stop and ask.
- Prefer focused, minimal output. No trailing summaries.
- Sub-agent output stays in the sub-agent. Main context gets one-line confirmations.

---

# NIL — Agent Development Guide

NIL is a keyboard-driven personal task and note management desktop app. It is currently in public beta.

---

## Stack

| Layer | Technology |
|-------|-----------|
| Framework | Wails v2 (Go backend + WebView frontend) |
| Backend | Go 1.24, SQLite via `modernc.org/sqlite` (pure Go, no CGo) |
| Frontend | React 18, TypeScript, Vite 3 |
| Rich text | TipTap v3.7.2 (ProseMirror-based) |
| Styling | CSS custom properties (terminal/dark theme system); **Tailwind/shadcn not yet installed — migration planned** |
| Distribution | macOS (universal), Windows (NSIS installer), Linux (AppImage) |

---

## Project Layout

```
nil/
├── main.go              # Wails entry point, window config; dispatches bare CLI args to cli.Run() before wails.Run()
├── app.go               # App struct, all Wails-bound Go methods (embeds the same apiserver used standalone)
├── go.mod / go.sum      # module: github.com/hollis-labs/nil
├── wails.json           # App name (NIL), build config
├── Makefile             # Canonical local verification path: build/test/lint/verify/codegen(-check)
├── build-local.sh       # Quick local macOS build
├── build-for-friends.sh # Cross-platform distribution build
├── apiserver/
│   └── apiserver.go     # HTTP API (routes/handlers/auth); used by both the embedded GUI server and standalone `nil serve-api`
├── service/items/
│   └── items.go         # Shared business-logic layer (create/update/search defaults, notes-input resolution, notes_text rendering) used by app.go, apiserver, cli, chat/bridge.go
├── cli/
│   └── cli.go            # `nil <command>` CLI registry; package cli; includes `nil serve-api` (headless API server)
├── cmd/
│   ├── nil-mcp/          # Standalone MCP stdio server (main.go + mcp.go) → `nil-mcp` binary; proxies tool calls to the HTTP API
│   └── nil-recover/      # Offline recovery tool for a closed DB (break-glass utility)
├── chat/
│   └── bridge.go         # AI chat integration (direct Anthropic API, not a CLI subprocess — see historical ADR-001 rationale)
├── ingest/                # Doc conversion: MarkdownToDoc, HTMLToDoc, DocToHTML, DocToPlainText, ExtractRefIDs
├── vault/                 # Multi-vault manager: vault registry, active-vault switching, per-vault *store.Store
├── config/
│   └── config.go         # OS-appropriate config/data paths, JSON config (incl. API port/key)
├── store/
│   ├── models.go         # Item, SearchRequest structs
│   ├── schema.sql        # Embedded SQL: tables, triggers, FTS5, refs, kinds registry
│   └── store.go          # All DB logic: CRUD, search, migrations, refs
├── parse/
│   └── line.go            # todo.txt-inspired line parser
└── frontend/
    ├── package.json
    ├── vite.config.ts   # Path alias: @ → ./src
    ├── src/
    │   ├── main.tsx
    │   ├── style.css        # Global styles
    │   ├── pages/
    │   │   └── App.tsx      # Root component, all state, event wiring
    │   ├── components/      # Modal and UI components
    │   ├── lib/             # Utilities, extensions, CSS
    │   └── theme/           # Theme types, presets, ThemeProvider
    └── wailsjs/             # Auto-generated Wails TS bindings (do not edit)
        ├── go/main/App.d.ts
        ├── go/main/App.js
        └── go/models.ts
```

---

## Key Conventions

### Backend (Go)

- All Wails-exposed methods live in **`app.go`** as methods on `*App`.
- All database logic lives in **`store/store.go`** as methods on `*Store`.
- Shared create/update/search logic (notes-input resolution, defaulting, `notes_text` rendering) lives in **`service/items.Service`** (`service/items/items.go`) — `app.go`, `apiserver/apiserver.go`, `cli/cli.go`, and `chat/bridge.go` all route through it rather than duplicating logic. Add new cross-surface behavior here, not per-consumer.
- The HTTP API (routes/handlers/auth) lives in **`apiserver/apiserver.go`**, used both by the GUI's embedded server (gated on `cfg.APIEnabled`) and the standalone `nil serve-api` CLI subcommand (headless, no Wails GUI required) — same `apiserver.New(cfg, mgr)` constructor either way.
- Schema is defined in **`store/schema.sql`** (embedded at compile time). Add new tables here for fresh installs.
- Schema changes for existing users use the **migration slice** in `store/store.go`. Increment `currentSchemaVersion` and add a new `migration` entry. Migrations are idempotent (use `IF NOT EXISTS` guards, or catch "already exists" errors). If a migration adds a column that also needs an index, create the index in `Open()` **strictly after** `runMigrations` returns — `schema.sql`'s unconditional exec runs before migrations on every `Open()` call, so an index on a not-yet-migrated column fails outright (see the `external_ref` index for the pattern).
- Current schema version: **12** — items live in the `todos` table with a `kind` column (`todo`/`note`/`scratch`/registered custom kinds, validated against a `kinds` registry table), not the legacy `type` column. Notes are stored as ProseMirror/TipTap document JSON in `notes_doc` (canonical), with `notes_html` as a write-time render cache. See `AGENTS.md` for the fuller schema-evolution narrative (JSON-at-rest migration, v7–v11).
- Canonical build/verify path: **`make verify`** (lint + test + frontend-build + codegen-check + build). Individual targets: `make build`, `make test`, `make lint`, `make codegen`, `make codegen-check`. `go build ./...`/`go test ./...` alone do **not** catch stale generated Wails bindings — always run `make codegen-check` (or full `make build`) after changing a Go type used by a Wails-bound method signature (e.g. `store.SearchRequest`, `store.Item`), even if the change looks backend-only. This has silently broken `make build` before without `go build`/`go test` noticing.
- After adding or changing Go methods bound to Wails, regenerate TypeScript bindings: `make codegen` (wraps `wails generate module`).

### Frontend (TypeScript/React)

- **`frontend/wailsjs/`** is auto-generated — never edit these files manually.
- Import generated bindings as: `import * as Backend from "../../wailsjs/go/main/App"` (adjust relative depth as needed).
- Path alias `@` resolves to `frontend/src/`.
- All global app state and top-level event handlers live in **`src/pages/App.tsx`**. This file is large; look before adding new state or handlers.
- Modal components receive open/close state and callbacks as props. They do not manage their own visibility in app-level state.
- Styling uses CSS custom properties (`--term-bg`, `--term-fg`, `--term-accent`, etc.) defined in `src/theme/theme.css` and driven by `ThemeProvider`. Do not hardcode colors — always use theme variables.
- The `@` autocomplete in the notes editor uses TipTap's Mention extension (see `src/lib/WikilinkExtension.tsx` and `src/components/WikilinkSuggestion.tsx`).

### Data Model

The core entity is `todos` (DB table name unchanged) — it holds **items**, **notes**, and any other registered **kind** (e.g. `scratch`). The Go struct is `store.Item`. Fields:

- `id`, `title`, `priority`, `completed`, `archived`, `created_at`, `updated_at`
- `due_at`, `threshold_at`, `recurrence_rule` (optional scheduling)
- `notes_doc` — TipTap/ProseMirror document JSON (canonical body storage); `notes_html` — write-time HTML render cache (`notes_html_version` for cache invalidation). External-facing surfaces (API/CLI/MCP) also expose a computed `notes_text` plaintext rendering (via `ingest.DocToPlainText`, `service/items.Service.PlainText`) — not a stored column, derived at response time. The legacy `notes_md` column is deadweight (retired in the v7–v11 JSON-at-rest migration).
- `section` — `now` | `soon` | `anytime`
- `pinned`, `kind` (`todo` | `note` | `scratch` | other registered kinds — renamed from `type` in schema v9, validated against the `kinds` registry table)
- `inbox` — boolean (`INTEGER DEFAULT 0`); auto-set when title is blank on creation, or set explicitly via the "→ Inbox" button in the create modal; inbox items are excluded from normal search results
- `api_source` — free-text label of which surface wrote the item (`cli`, `nil-mcp`, etc.)
- `external_ref` — optional writer-supplied idempotency key for external push flows; re-creating with the same `external_ref` (within the same vault) overwrites the existing row's create-payload fields instead of duplicating it (preserves `completed`/`archived`/`created_at`)
- Taxonomy via join tables: `todo_projects`, `todo_contexts`, `todo_tags`
- Inter-item references via `refs (source_id, target_id)` — synced on every save; readable back out via `GetBackrefs` (exposed on all three external surfaces, not just the GUI)

### localStorage Keys

All localStorage keys use the `nil.` prefix:
- `nil.settings` — app settings (tabs, showCompleted, etc.)
- `nil.viewMode` — scope | date
- `nil.appMode` — todos | notes
- `nil.inputMode` — search | add
- `nil.sessionProfiles` — saved session profiles
- `nil.activeSession` — current active session
- `nil.taskTemplates` — saved task templates

### Theme System

Themes are defined in `src/theme/theme.ts` as `TermTheme` objects. The `ThemeProvider` injects them as CSS custom properties. All UI uses `var(--term-*)` variables. Add new theme variables to the `TermTheme` type and all presets together.

The theme switcher is **hidden from the Settings UI** until the Tailwind/shadcn migration is complete. The theme system itself remains intact in code.

---

## Development Commands

```bash
# Full local verification path (lint + test + frontend-build + codegen-check + build)
make verify

# Individual targets
make build           # wails build
make test            # go test ./...
make lint            # format-check + go-lint + frontend-lint
make codegen         # wails generate module
make codegen-check   # regenerate bindings, fail if frontend/wailsjs drifted

# Run in dev mode (hot reload)
wails dev

# Run the HTTP API headlessly, no Wails GUI required
go run . serve-api [--port N]

# Build macOS app locally
./build-local.sh

# Cross-platform distribution build (macOS + Windows; Linux run on Linux)
./build-for-friends.sh

# Type-check frontend only
cd frontend && npx tsc --noEmit
```

**Debug mode** (opens WebKit inspector on startup):
```bash
DEBUG=1 open build/bin/NIL.app
```

**Git hooks**: `lefthook.yml` defines pre-commit (Go format/lint/vet + frontend
lint) and pre-push (`go test` + `make codegen-check`) hooks so Wails binding
drift is caught automatically, not just when a human remembers `make verify`.
One-time setup: `lefthook install`. See "Git hooks (lefthook)" in AGENTS.md
for the stale-`core.hooksPath` gotcha if hooks silently don't fire.

---

## Known Issues / Backlog

### Wikilink click navigation (WKWebView)
- **Status**: Backlogged
- **What works**: `@` mention insertion, chip rendering, `refs` table sync, backlinks section in NotesModal.
- **What doesn't**: Clicking a `@reference` chip inside the TipTap editor does not open the referenced item's modal.
- **Root cause**: WKWebView (macOS WebView) suppresses `click` events after ProseMirror handles `mousedown`. All attempted fixes (React onClick, container delegation, `handleDOMEvents`, native capture-phase listener) confirmed that `preventDefault` works but the state-update callback does not fire reliably.
- **Next steps**: Attach Safari devtools to the live WKWebView process (`Develop → [device]`) to confirm where the chain breaks. Alternative approach: render a small "open" button next to chips that lives outside ProseMirror's DOM.

### Import/Export not working
- **Status**: Backlogged
- **Note**: Need to revisit to understand the current state and plan proper fix/feature.

---

## Immediate Priorities

**Live task tracking moved to Torque** (project `PRJ-20260417-0005` / "Nil") as of 2026-08-16 — this checklist is historical and will not be kept current. Check Torque for what's actually outstanding rather than trusting this list. Notable active tracked work: the "Nil modernization baseline" portfolio (`CW-20260424-0026` and its phase tasks — verification workflow is done; backend/frontend test coverage, an architecture doc + ADRs, splitting `cli.go`/`store.go`/`app.go`/`App.tsx`, and a Wails-binding stale-check are still outstanding) and the completed "External Data Access API" epic (`EP-20260816-0003` — headless `serve-api`, bulk list w/ `updated_since`, `notes_text`, backlinks, deletion-signal ID-diff endpoint, batch create + `external_ref`).

Historical, already-done items (kept for context, not action items):
- App rename PLANCK → NANITE → NIL; `Todo` struct → `Item`; theme switcher hidden pending Tailwind/shadcn.
- Default-tab migration; Escape-closes-EditItemModal dirty-detection flow.

Still-relevant unresolved items not yet in Torque as of this writing:
- **User-facing docs**: Keyboard shortcut reference, Quick Add syntax guide, Scope and Session documentation, cloud sync setup guide. See `docs/` for current state.
- **Fix wikilink click** (see Known Issues above).

---

## Feature Roadmap

### F1 — Inbox (Fast Capture + Triage)

**Status**: MVP shipped. See remaining items below.

**Philosophy**: Never lose an idea to taxonomy friction. Users can create an item or note with no title or metadata — this is intentional. Empty or minimally-filled items are automatically flagged as **inbox items** and surfaced in a dedicated Inbox view rather than cluttering the main Items or Notes views.

**What's implemented (MVP)**:
- `inbox` boolean column in DB (migration v4); auto-set when title is blank on creation
- Inbox items excluded from all normal search/item/notes results
- `GetInboxCount`, `GetInboxItems`, `ProcessInboxItem` backend methods + Wails bindings
- Inbox count badge in tab bar (dimmed when 0, accented when items present)
- `Cmd+I` / `Ctrl+I` keyboard shortcut to open Inbox
- `InboxView` component: search, per-item actions (Edit, Archive, Delete, Process), batch selection with bulk Process/Archive/Delete, keyboard navigation
- Optimistic UI updates — view reflects changes immediately, then reloads from backend
- Empty quick-add (blank title) routes to inbox instead of being rejected
- **"→ Inbox" button** in EditItemModal (create mode only): deliberately routes any item to inbox
- **`UpdateItem` persists `inbox` field** (bug fix: the SQL UPDATE previously omitted the column)

**Remaining / future iterations**:
- Saved named inbox views (filter presets stored in localStorage)
- Deterministic router/classifier (keyword rules → auto-assign taxonomy on capture)
- AI-assisted routing and batch triage mode
- Advanced filter syntax within inbox (taxonomy, date ranges, negative terms)

---

### F2 — Advanced Search & Recall

**Goal**: Make retrieval fast and precise. Users should always be able to find what they're looking for.

**Core capabilities**:
- Support negative search terms, field-scoped queries, date range filters, type filters, and taxonomy filters — building on the existing FTS5 infrastructure.
- User-defined **result templates** (Markdown only for v1, with basic dynamic tags like `{{title}}`, `{{created_at}}`, `{{tags}}`).

**Data & search evaluation**:
- Audit the current data model and FTS5 setup to identify refactoring opportunities.
- Expose a user-facing control for **speed vs. depth** trade-off (fast FTS5 vs. slower ranked fuzzy/semantic search).

---

### F3 — Addon System *(low priority)*

A plugin architecture that allows first- and third-party addons to introduce new data types and behaviors without bloating the core app.

**Planned first-party addons**:
- **Journal** — time-stamped daily entries with prompt support
- **Scheduler** — recurring reminders and time-based triggers
- **Broadcast** — send messages/updates to a list of recipients or channels
- **Contacts** — lightweight contact records linkable to items/notes
- **Calendar** — event and deadline visualization

---

### F4 — Automations & Flows *(low priority)*

AI-assisted automation builder for creating recurring workflows: daily/weekly digests, custom newsletters, report generation, and any repeatable process that can be templated.

---

## Agent Notes

- When modifying the schema, always update **both** `store/schema.sql` (for fresh installs) **and** add a migration in `store/store.go` (for existing users).
- After any Go method signature change — or any change to a type used as a Wails-bound method's parameter/return (e.g. `store.Item`, `store.SearchRequest`), even if the change looks backend-only — run `make codegen` (or `make codegen-check` to fail loudly on drift) to keep TypeScript bindings in sync. `go build`/`go test` do not catch this class of bug.
- The `App.tsx` file is the single source of truth for app state. Read it fully before adding new state or handlers.
- TipTap editors run inside WKWebView on macOS. Standard DOM event handling has quirks — test click/keyboard interactions in the actual app, not just in a browser.
- Tailwind and shadcn/ui are **not yet installed**. Do not assume they are available until the migration priority has been completed. Until then, continue using the existing `var(--term-*)` CSS custom property system.
- **EditItemModal close flow**: Escape / Close / Cancel all route through `requestClose()`, which checks `isDirty()` and respects the `closeBehavior` setting (`ask` | `always` | `never`). The "→ Inbox" button is create-mode only and passes `extras.inbox = true` through `onSubmit` → `handleQuickAdd` / `handleQuickAddNote` → `UpdateItem`. Both App.tsx handlers must merge `extras.inbox` into the item before calling `UpdateItem`.
- **DB table name**: The SQLite table is still `todos` (renaming it would require a migration and is deferred). The Go struct is `store.Item`, Wails method is `UpdateItem`, etc. Do not rename the table without a proper migration plan.
