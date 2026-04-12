# NIL

## agentrc
- If `.agentrc/boot-prompt.md` exists, read it first for session context.
- If the user says "Boot <agent>", look up the agent in `.agentrc/config.yaml` under `agents:`. Load each role file from `~/.agentrc/roles/` (using the `file:` path from `~/.agentrc/config.yaml` role definitions), load the listed skills, and read the project context file from `.agentrc/` if specified.
- If the user says "Boot <role>" and no agent matches, fall back to loading that single role from `~/.agentrc/roles/` by type directory (domain/, stack/, meta/).
- After context compaction, re-read the active role and project context files.
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
├── main.go              # Wails entry point, window config
├── app.go               # App struct, all Wails-bound Go methods
├── go.mod / go.sum      # module: github.com/hollis-labs/nil
├── wails.json           # App name (NIL), build config
├── build-local.sh       # Quick local macOS build
├── build-for-friends.sh # Cross-platform distribution build
├── config/
│   └── config.go        # OS-appropriate config/data paths, JSON config
├── store/
│   ├── models.go        # Item, SearchRequest structs
│   ├── schema.sql       # Embedded SQL: tables, triggers, FTS5, refs
│   └── store.go         # All DB logic: CRUD, search, migrations, refs
├── parse/
│   └── line.go          # todo.txt-inspired line parser
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
- Schema changes for existing users use the **migration slice** in `store/store.go`. Increment `currentSchemaVersion` and add a new `migration` entry. Migrations are idempotent (use `IF NOT EXISTS` guards, or catch "already exists" errors).
- Current schema version: **4** (inbox column).
- Build/test the Go layer with `go build .` from the project root.
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

The core entity is `todos` (DB table name unchanged) — it holds both **items** (`type='todo'`) and **notes** (`type='note'`). The Go struct is `store.Item`. Fields:

- `id`, `title`, `priority`, `completed`, `archived`, `created_at`, `updated_at`
- `due_at`, `threshold_at`, `recurrence_rule` (optional scheduling)
- `notes_md` — HTML string (TipTap output); also full-text indexed via FTS5
- `section` — `now` | `soon` | `anytime`
- `pinned`, `type` (`todo` | `note`)
- `inbox` — boolean (`INTEGER DEFAULT 0`); auto-set when title is blank on creation, or set explicitly via the "→ Inbox" button in the create modal; inbox items are excluded from normal search results
- Taxonomy via join tables: `todo_projects`, `todo_contexts`, `todo_tags`
- Inter-item references via `refs (source_id, target_id)` — synced on every save

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
DEBUG=1 open build/bin/NIL.app
```

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

- [x] **Dead file cleanup**: `app/app.go`, `store/store.go.bak`, debug scripts, stale `.md` files removed.
- [x] **App rename**: PLANCK → NANITE → NIL throughout (wails.json, main.go, index.html, config paths, build scripts, docs, localStorage keys).
- [x] **Item rename**: `Todo` struct → `Item`; Wails methods `CreateTodoFromLine` → `CreateItemFromLine`, `UpdateTodo` → `UpdateItem`, etc. DB `type` column values `'todo'`/`'note'` unchanged.
- [x] **Hide theme switcher**: Theme tab hidden from Settings UI (`SettingsTab` type, tab button commented out, content gated with `false &&`).
- [x] **Default tab migration**: `defaultSettings` now includes All tab for both todos and notes modes. `SettingsProvider` injects a notes-mode All tab on first load if missing.
- [x] **Escape closes EditItemModal**: Dirty detection + inline prompt configurable via Settings → General → Close & Save Behavior.
- [ ] **User-facing docs**: Keyboard shortcut reference, Quick Add syntax guide, Scope and Session documentation, cloud sync setup guide. See `docs/` for current state.
- [ ] **Code standardization**: Audit components for inline styles vs. CSS variables; standardize prop patterns across modals.
- [ ] **Stack evaluation**: Evaluate whether Wails v2 → v3, or an alternative framework, is the right long-term choice before the codebase grows further.
- [ ] **Frontend layout migration (Tailwind/shadcn)**: Audit the existing component library, plan a migration to Tailwind v4 + shadcn/ui that preserves the terminal aesthetic and the `--term-*` CSS variable theme system.
- [ ] **Fix wikilink click** (see Known Issues above).

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
- After any Go method signature change, run `wails generate module` to keep TypeScript bindings in sync.
- The `App.tsx` file is the single source of truth for app state. Read it fully before adding new state or handlers.
- TipTap editors run inside WKWebView on macOS. Standard DOM event handling has quirks — test click/keyboard interactions in the actual app, not just in a browser.
- Tailwind and shadcn/ui are **not yet installed**. Do not assume they are available until the migration priority has been completed. Until then, continue using the existing `var(--term-*)` CSS custom property system.
- **EditItemModal close flow**: Escape / Close / Cancel all route through `requestClose()`, which checks `isDirty()` and respects the `closeBehavior` setting (`ask` | `always` | `never`). The "→ Inbox" button is create-mode only and passes `extras.inbox = true` through `onSubmit` → `handleQuickAdd` / `handleQuickAddNote` → `UpdateItem`. Both App.tsx handlers must merge `extras.inbox` into the item before calling `UpdateItem`.
- **DB table name**: The SQLite table is still `todos` (renaming it would require a migration and is deferred). The Go struct is `store.Item`, Wails method is `UpdateItem`, etc. Do not rename the table without a proper migration plan.
