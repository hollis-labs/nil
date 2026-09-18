# NIL

NIL is a keyboard-driven, local-first task and note manager: a Wails v2
desktop app (Go backend, React/TypeScript frontend) that stores tasks and
notes as one data model in a single SQLite vault you own outright. It also
ships an MCP server (`nil-mcp`), so agents can read and write that same vault
directly instead of going through the UI.

> **Public beta, built in the open.** NIL is in daily personal use and its
> data model and MCP surface are stable. It's still early: the frontend is
> mid-migration to Tailwind/shadcn, the addon system doesn't exist yet, and
> interfaces can change without notice. Bug reports and feature requests are
> welcome on GitHub.

## What it is today

- **Keyboard-first capture** with a todo.txt-inspired Quick Add syntax —
  `(A) Buy milk +shopping @errands #personal due:2026-02-25`.
- **One data model for tasks and notes** (`Item`, `kind`-discriminated), FTS5
  full-text search, and wikilink-style `@reference` chips with backlinks.
- **Inbox, Sessions, Scopes, and Custom Tabs** for zero-friction capture,
  focused work blocks, and saved searches.
- **Five surfaces over one vault**: the GUI (Wails in-process bridge), a local
  HTTP API bound to `127.0.0.1`, a CLI (`nil serve-api` runs the same API
  headless), an MCP server (`nil-mcp`, stdio), and a chat bridge that is the
  only thing NIL calls out to the network for (`api.anthropic.com`). All five
  funnel through the same `service/items` layer, so nothing gets a second copy
  of the create/update/search logic.
- **Cross-platform builds** (macOS, Windows, Linux) with cloud sync via any
  synced folder (iCloud Drive, Dropbox, etc.) — NIL doesn't run its own sync
  service.

## Where it sits in the stack

```
   agents / assistants     Claude Code, Nanite, Claude Desktop, any MCP client
         │  MCP (stdio): nil_search, nil_create_item, nil_process_inbox, ...
    ┌─────────┐
    │  NIL    │   one vault, one SQLite file, one owner
    └─────────┘
         │  same data, four other doors in
   GUI · HTTP API (127.0.0.1) · CLI · embedded chat (api.anthropic.com)
```

Where Nanite is the cockpit an agent runs inside, NIL is a data owner an agent
reaches *into* — it doesn't run agents itself, it exposes fourteen MCP tools
(`nil_search`, `nil_create_item`, `nil_update_item`, `nil_process_inbox`,
`nil_get_backrefs`, and others) so anything that speaks MCP can read or file
into the same vault a human is looking at in the GUI.

## Examples

**Daily driver.** Chrispian captures tasks and notes throughout the day with
Quick Add syntax, triages the Inbox, and uses Sessions to scope a tab to
whatever he's focused on right now — no mouse required.

**Composition.** A coding agent (Claude Code, Nanite) finishes a piece of work
and calls `nil_create_item` over MCP to log a follow-up directly into the same
vault, tagged and due-dated, instead of leaving a note in chat that gets lost.
An agent doing research calls `nil_search` to check whether something's
already tracked before creating a duplicate. Inside the app itself, the
embedded chat bridge lets a NIL user ask Claude about their own tasks without
exporting anything.

## Roadmap

Full detail in [ROADMAP.md](ROADMAP.md); near-term highlights:

- **Frontend layout migration** to Tailwind v4 + shadcn/ui, preserving the
  current terminal aesthetic.
- **AI-assisted inbox routing and batch triage** — deterministic rules first,
  AI-assisted assignment on top.
- **Advanced & semantic search** — field-scoped queries, date ranges, and a
  speed/depth trade-off between FTS5 and ranked fuzzy/semantic search.
- **Addon system** (low priority) — Journal, Scheduler, Broadcast, Contacts,
  Calendar as opt-in first-party addons.
- **AI-assisted automation builder** — natural-language-authored recurring
  flows (digests, newsletters, reports), pairing with the addon system.

## Install

See [INSTALL.md](INSTALL.md) for platform-specific instructions (macOS,
Windows, Linux).

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

`make verify` is the canonical local verification path for Nil. It runs Go
format and lint checks, frontend lint, the Go test suite, a standalone
frontend production build, a Wails binding drift check, and a full
`wails build`.

## Contributing

NIL is in public beta. Bug reports and feature requests are welcome — please
open an issue on GitHub.

## License

NIL is open source under the MIT License — see [LICENSE](LICENSE).
