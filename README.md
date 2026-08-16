# NIL

Keyboard-driven personal task and note management.

## What is NIL?

NIL is a minimal desktop app for capturing and managing tasks and notes without leaving the keyboard. It uses a todo.txt-inspired input syntax with extensions for due dates, recurrence, and tagging — all stored locally in a SQLite database.

## Key Features

- **Keyboard-first** — create, search, and navigate without touching the mouse
- **todo.txt-inspired syntax** — `(A) Buy milk +shopping @errands #personal due:2026-02-25`
- **Items + Notes** — one data model for both task items and long-form notes with rich-text editing
- **Inbox** — fast capture with zero taxonomy friction; triage later
- **Sessions** — temporary context filter for focused work; new items auto-inherit session tags
- **Scopes** — Now / Soon / Anytime views to manage focus
- **Custom Tabs** — save common searches as persistent tab filters
- **Wikilinks** — `@reference` chips link items together with backlinks
- **Cloud sync** — store your database in iCloud Drive, Dropbox, or any folder

## Install

See [INSTALL.md](INSTALL.md) for platform-specific instructions (macOS, Windows, Linux).

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl+N` | New Item |
| `Cmd/Ctrl+Shift+N` | New Note |
| `Cmd/Ctrl+S` | Quick Search |
| `Shift+Shift` | Quick Search (alias) |
| `Cmd/Ctrl+I` | Open Inbox |
| `Cmd/Ctrl+Shift+M` | Toggle Items / Notes mode |
| `Escape` | Close modal / back |

## Quick Add Syntax

```
(A) Buy milk +shopping @errands #personal due:2026-02-25
```

| Token | Meaning |
|-------|---------|
| `(A)` | Priority — A, B, or C |
| `+project` | Project tag |
| `@context` | Context tag |
| `#tag` | Flexible label |
| `due:YYYY-MM-DD` | Due date |
| `t:YYYY-MM-DD` | Threshold (hide until date) |
| `rec:Nd/Nw/Nm` | Recurrence (days/weeks/months) |

## Development

```bash
# Show the canonical local workflow surface
make help

# Run the full local verification path
make verify

# Individual checks
make lint
make test
make frontend-build
make codegen-check

# Run in dev mode (hot reload)
wails dev

# Build the desktop app locally
make build

# Cross-platform distribution build
./build-for-friends.sh

# Regenerate Wails TypeScript bindings after Go changes
make codegen
```

**Debug mode** (opens WebKit inspector on startup):
```bash
DEBUG=1 open build/bin/NIL.app
```

`make verify` is the canonical local verification path for Nil. It runs Go format and lint checks, frontend lint, the Go test suite, a standalone frontend production build, a Wails binding drift check, and a full `wails build`.

## Contributing

NIL is in public beta. Bug reports and feature requests are welcome — please open an issue on GitHub.

## License

Copyright © 2026 Hollis Labs. All rights reserved.
