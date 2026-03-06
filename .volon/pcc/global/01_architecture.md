---
intent: pcc_global
project: nanite
updated_at: "2026-03-04"
---

# Architecture

## Tech stack

- **Backend:** Go 1.24, Wails v2, modernc.org/sqlite (pure Go, no CGo)
- **Frontend:** React 18, TypeScript, Vite 3, TipTap v3.7.2 (ProseMirror)
- **Styling:** CSS custom properties (`--term-*` terminal/dark theme); Tailwind/shadcn migration planned
- **Distribution:** macOS universal, Windows NSIS, Linux AppImage

## Key components

### Entry point (`main.go`)
Wails window configuration and app initialization.

### App struct (`app.go`)
All Wails-bound Go methods. Single file is the API surface between frontend and backend.

### Store (`store/`)
- `store.go` -- all DB logic: CRUD, search, migrations, refs, FTS5
- `models.go` -- Item, SearchRequest structs
- `schema.sql` -- embedded SQL (tables, triggers, FTS5, refs). Current schema version: 4

### Parser (`parse/line.go`)
todo.txt-inspired line parser for quick-add syntax.

### Config (`config/config.go`)
OS-appropriate config/data paths, JSON configuration, vault paths.

### Vault (`vault/manager.go`)
Multi-vault support; each vault has its own SQLite database.

### Chat (`chat/`)
Chat/AI bridge with models, profiles, actions, and store.

### API (`api.go`)
REST API server bound to `127.0.0.1` only. Endpoints: POST/GET inbox, GET search, GET items by ID. Bearer-token auth.

### CLI (`cli/cli.go`)
CLI interface for headless operations.

### Frontend (`frontend/src/`)
- `pages/App.tsx` -- root component, all state, event wiring (large file)
- `components/` -- modal and UI components
- `lib/` -- utilities, TipTap extensions (WikilinkExtension)
- `theme/` -- theme types, presets, ThemeProvider

## Data model

Core entity: `todos` table (holds both items and notes). Fields: id, title, priority, completed, archived, section (now/soon/anytime), pinned, type (todo/note), inbox, notes_md (HTML), due_at, threshold_at, recurrence_rule. Taxonomy via join tables. Inter-item refs for wikilinks.

## Evidence
- Last refreshed: 2026-03-04 (mentat PCC bootstrap)
- Sources: CLAUDE.md, app.go, store/, api.go, frontend/src/, config/
