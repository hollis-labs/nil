# PLANCK — Agent Development Guide

PLANCK is a keyboard-driven personal task and note management desktop app. It is currently in alpha/beta, being shared with a small group of friends before wider release.

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
todo-app/
├── main.go              # Wails entry point, window config
├── app.go               # App struct, all Wails-bound Go methods
├── go.mod / go.sum
├── wails.json           # App name (PLANCK), build config
├── build-local.sh       # Quick local macOS build
├── build-for-friends.sh # Cross-platform distribution build
├── config/
│   └── config.go        # OS-appropriate config/data paths, JSON config
├── store/
│   ├── models.go        # Todo, SearchRequest structs
│   ├── schema.sql       # Embedded SQL: tables, triggers, FTS5, refs
│   └── store.go         # All DB logic: CRUD, search, migrations, refs
├── parse/
│   └── line.go          # todo.txt line parser
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
- Schema is defined in **`store/schema.sql`** (embedded at compile time). Add new tables here for fresh installs.
- Schema changes for existing users use the **migration slice** in `store/store.go`. Increment `currentSchemaVersion` and add a new `migration` entry. Migrations are idempotent (use `IF NOT EXISTS`, `IF NOT EXISTS` guards, or catch "already exists" errors).
- Current schema version: **4** (inbox column).
- Build/test the Go layer with `go build .` from the project root (not `go build ./...` — the `app/` directory has a legacy stub that may fail).
- After adding or changing Go methods bound to Wails, regenerate TypeScript bindings: `wails generate module`.

### Frontend (TypeScript/React)

- **`frontend/wailsjs/`** is auto-generated — never edit these files manually.
- Import generated bindings as: `import * as Backend from "../../wailsjs/go/main/App"` (adjust relative depth as needed).
- Path alias `@` resolves to `frontend/src/`.
- All global app state and top-level event handlers live in **`src/pages/App.tsx`**. This file is large; look before adding new state or handlers.
- Modal components receive open/close state and callbacks as props. They do not manage their own visibility in app-level state.
- Styling uses CSS custom properties (`--term-bg`, `--term-fg`, `--term-accent`, etc.) defined in `src/theme/theme.css` and driven by `ThemeProvider`. Do not hardcode colors — always use theme variables.
- The `@` autocomplete in the notes editor uses TipTap's Mention extension (see `src/lib/WikilinkExtension.tsx` and `src/components/WikilinkSuggestion.tsx`).

### Data Model

The core entity is `todos` — it holds both **todos** (`type='todo'`) and **notes** (`type='note'`). Fields:

- `id`, `title`, `priority`, `completed`, `archived`, `created_at`, `updated_at`
- `due_at`, `threshold_at`, `recurrence_rule` (optional scheduling)
- `notes_md` — HTML string (TipTap output); also full-text indexed via FTS5
- `section` — `now` | `soon` | `anytime`
- `pinned`, `type` (`todo` | `note`)
- `inbox` — boolean (`INTEGER DEFAULT 0`); auto-set when title is blank on creation; inbox items are excluded from normal search results
- Taxonomy via join tables: `todo_projects`, `todo_contexts`, `todo_tags`
- Inter-item references via `refs (source_id, target_id)` — synced on every save

### Theme System

Themes are defined in `src/theme/theme.ts` as `TermTheme` objects. The `ThemeProvider` injects them as CSS custom properties. All UI uses `var(--term-*)` variables. Add new theme variables to the `TermTheme` type and all presets together.

---

## Development Commands

```bash
# Run in dev mode (hot reload)
wails dev

# Build macOS app locally
./build-local.sh

# Cross-platform distribution build (macOS + Windows; Linux run on Linux)
./build-for-friends.sh

# Regenerate Wails TypeScript bindings after Go changes
wails generate module

# Type-check frontend only
cd frontend && npx tsc --noEmit
```

**Debug mode** (opens WebKit inspector on startup):
```bash
DEBUG=1 open build/bin/PLANCK.app
```

---

## Known Issues / Backlog

### Wikilink click navigation (WKWebView)
- **Status**: Backlogged
- **What works**: `@` mention insertion, chip rendering, `refs` table sync, backlinks section in NotesModal.
- **What doesn't**: Clicking a `@reference` chip inside the TipTap editor does not open the referenced item's modal.
- **Root cause**: WKWebView (macOS WebView) suppresses `click` events after ProseMirror handles `mousedown`. All attempted fixes (React onClick, container delegation, `handleDOMEvents`, native capture-phase listener) confirmed that `preventDefault` works but the state-update callback does not fire reliably.
- **Next steps**: Attach Safari devtools to the live WKWebView process (`Develop → [device]`) to confirm where the chain breaks. Alternative approach: render a small "open" button next to chips that lives outside ProseMirror's DOM.

---

## Immediate Priorities

- [ ] **Rename/cleanup**: The project folder is still named `todo-app-starter/todo-app`. The app is called PLANCK. Clean up directory names, stale `.md` files, and the leftover `app/app.go` stub.
- [ ] **Hide theme switcher**: Temporarily hide the theme menu option from the UI. The theme system stays in the codebase; it just won't be user-accessible until the Tailwind/shadcn migration is complete.
- [ ] **Default tab migration**: Add a schema migration so that the Notes and Todos views each have an "All" tab by default when no user-created tabs exist. Both views should fall back gracefully to showing all items if no scopes have been defined.
- [ ] **Escape closes add modal**: When the Quick Add / new todo or note modal is open, pressing Escape should close it, consistent with how all other modals behave.
- [ ] **User-facing docs**: Consolidate the many scattered `.md` files into coherent end-user documentation covering:
  - The input syntax is loosely inspired by todo.txt, not a strict implementation — clarify this throughout
  - The toggle on the search/add icon next to the main input (switches between search mode and add mode)
  - Keyboard usage, example workflows, philosophy, and a full keyboard shortcut reference
  - How simple and advanced Scopes work
  - How the Session Manager works: while a session is active, any item created automatically inherits the session's tags/contexts for faster capture
  - Quick Add syntax on the main view (todo.txt-inspired inline syntax)
  - How main-view search works
  - Cloud sync setup guide for iCloud, OneDrive, Dropbox, and Google Drive
- [ ] **Code standardization**: Audit components for inline styles vs. CSS variables; standardize prop patterns across modals.
- [ ] **Stack evaluation**: Evaluate whether Wails v2 → v3, or an alternative framework, is the right long-term choice before the codebase grows further.
- [ ] **Frontend layout migration (Tailwind/shadcn)**: The current styling uses hand-rolled CSS custom properties with no utility framework. This project normally uses Tailwind + shadcn/ui. Audit the existing component library, plan a migration to Tailwind v4 + shadcn/ui that preserves the terminal aesthetic and the `--term-*` CSS variable theme system. This is part of the stack evaluation — do the layout/styling analysis before committing to the migration approach.
- [ ] **Fix wikilink click** (see Known Issues above).

---

## Feature Roadmap

### F1 — Inbox (Fast Capture + Triage)

**Status**: MVP shipped. See remaining items below.

**Philosophy**: Never lose an idea to taxonomy friction. Users can create a todo or note with no title or metadata — this is intentional. Empty or minimally-filled items are automatically flagged as **inbox items** and surfaced in a dedicated Inbox view rather than cluttering the main Todo or Notes views.

**What's implemented (MVP)**:
- `inbox` boolean column in DB (migration v4); auto-set when title is blank on creation
- Inbox items excluded from all normal search/todo/notes results
- `GetInboxCount`, `GetInboxItems`, `ProcessInboxItem` backend methods + Wails bindings
- Inbox count badge in tab bar (dimmed when 0, accented when items present)
- `Cmd+I` / `Ctrl+I` keyboard shortcut to open Inbox
- `InboxView` component (`src/components/InboxView.tsx`): search, per-item actions (Edit, Archive, Delete, Process), batch selection with bulk Process/Archive/Delete (sequential writes to avoid SQLite locking), keyboard navigation (↑↓, x=select, p=process, a=archive, d=delete, Enter=edit, Escape=back)
- Optimistic UI updates — view reflects changes immediately, then reloads from backend
- Empty quick-add (blank title) routes to inbox instead of being rejected

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
- User-defined **result templates** (Markdown only for v1, with basic dynamic tags like `{{title}}`, `{{created_at}}`, `{{tags}}`) to control how results are displayed. Sensible defaults ship out of the box.

**Data & search evaluation**:
- Audit the current data model and FTS5 setup to identify refactoring opportunities that better support multiple search strategies.
- Expose a user-facing control for **speed vs. depth** trade-off (e.g., fast prefix/FTS5 search vs. slower ranked fuzzy/semantic search), with sensible defaults so most users never need to think about it.

---

### F3 — Addon System *(low priority)*

A plugin architecture that allows first- and third-party addons to introduce new data types and behaviors without bloating the core app.

**Design principles**:
- Addons can define new item types (e.g., Journal entry, Contact) and new behaviors (e.g., scheduled prompts, broadcast messages).
- First-party addons serve as both useful features and reference implementations for third-party developers.
- Keep the core clean; addons are opt-in.

**Planned first-party addons**:
- **Journal** — time-stamped daily entries with prompt support
- **Scheduler** — recurring reminders and time-based triggers
- **Broadcast** — send messages/updates to a list of recipients or channels
- **Contacts** — lightweight contact records linkable to todos/notes
- **Calendar** — event and deadline visualization

---

### F4 — Automations & Flows *(low priority)*

AI-assisted automation builder for creating recurring workflows: daily/weekly digests, custom newsletters, report generation, and any repeatable process that can be templated.

- Addon/plugin-based so users install only what they need and the community can contribute.
- AI helps construct and refine flows from natural language descriptions.
- Pairs naturally with F3 addons (e.g., a Scheduler addon triggering a Broadcast flow).

---

## File Cleanup Candidates

These files likely predate refactors and can be reviewed for deletion:

- `app/app.go` — appears to be a legacy stub (causes `go build ./...` to fail)
- `store/store.go.bak` — backup file
- `debug_search.js`, `test-theme-properties.js` — ad-hoc debug scripts
- `test-build` — unclear artifact
- `launch-debug.sh`, `fix-database-location.sh` — check if still needed
- Many top-level `*.md` files — consolidate or delete

---

## Agent Notes

- When modifying the schema, always update **both** `store/schema.sql` (for fresh installs) **and** add a migration in `store/store.go` (for existing users).
- After any Go method signature change, run `wails generate module` to keep TypeScript bindings in sync.
- The `App.tsx` file is the single source of truth for app state. Read it fully before adding new state or handlers.
- TipTap editors run inside WKWebView on macOS. Standard DOM event handling has quirks — test click/keyboard interactions in the actual app, not just in a browser.
- Tailwind and shadcn/ui are **not yet installed**. Do not assume they are available until the migration priority (see Immediate Priorities) has been completed. Until then, continue using the existing `var(--term-*)` CSS custom property system.
