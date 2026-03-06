---
intent: pcc_global
project: nanite
updated_at: "2026-03-04"
---

# Project Identity

**NANITE** is a keyboard-driven personal task and note management desktop app. It uses a todo.txt-inspired input syntax with extensions for due dates, recurrence, and tagging -- all stored locally in a SQLite database.

Currently in public beta. Part of the Hollis Labs suite.

## Goals

- Keyboard-first task and note management with zero-taxonomy-friction inbox
- todo.txt-inspired quick-add syntax with extensions (due dates, recurrence, projects, contexts, tags)
- Dual data model: items (tasks) and notes (rich text via TipTap)
- Scope-based views (Now / Soon / Anytime), sessions for focused work
- Custom tabs with saved search filters
- Wikilinks (`@reference` chips) with backlinks
- Local-first SQLite storage with optional cloud sync (iCloud Drive, Dropbox)
- REST API for external integrations (inbox, search, item retrieval)
- Multi-vault support for separated data stores

## Non-goals

- Not a project management tool
- Not cloud-native; sync is folder-based only
- No AI features in core (planned for future addon system)

## Active configuration

- Module: `nanite` (Go 1.24)
- Framework: Wails v2 (Go backend + WebView frontend)
- Frontend: React 18, TypeScript, Vite 3
- Rich text: TipTap v3.7.2 (ProseMirror-based)
- Distribution: macOS (universal), Windows (NSIS), Linux (AppImage)

## Evidence
- Last refreshed: 2026-03-04 (mentat PCC bootstrap)
- Sources: README.md, CLAUDE.md, go.mod, wails.json
