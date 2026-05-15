# AGENTS.md — NIL

Orientation for agents working in this repository. For deep code conventions see
[CLAUDE.md](CLAUDE.md); this file is the higher-level "why and where".

## (a) What is this, and why

NIL is a keyboard-driven personal task and note management desktop app, currently
in public beta. It is a Wails v2 application — a Go backend with a React/TypeScript
frontend rendered in a WebView.

The product thesis: tasks and notes are the *same object* with different
affordances. NIL fills the gap between pure task managers (Todoist, Things) and
pure note apps (Obsidian, Logseq) by giving both a single data model, a
todo.txt-inspired capture syntax, and a fast keyboard-first UI with a terminal
aesthetic. The closest competitive reference is Logseq, but task-first.

Lineage note: the app was formerly named **PLANCK**, then **NANITE**. It was
renamed to **NIL** so the "Nanite" name could be reused by the CLI agent
framework. The Go module is `github.com/hollis-labs/nil`; the binary is `nil`;
the macOS bundle is `NIL.app`. Some legacy strings (`.agentrc/config.yaml`
agent descriptions, the `todos` DB table name) still say "Nanite"/"todo" — that
is known drift, not a signal to rename.

## (b) Where to start

Entry points:

- `main.go` — Wails entry point, window configuration.
- `app.go` — the `App` struct; every Wails-bound Go method lives here.
- `api.go` — HTTP API surface.
- `cli/cli.go` — CLI command registry (`nil <command>`); package `cli`.
- `cmd/nil-mcp/` — standalone MCP stdio server (`main.go` + `mcp.go`),
  builds to the `nil-mcp` binary.
- `frontend/src/pages/App.tsx` — root React component; single source of truth
  for app-level state and top-level event wiring. Read it fully before adding
  state or handlers.

Docs:

- [README.md](README.md) — user-facing overview, shortcuts, Quick Add syntax,
  dev commands.
- [CLAUDE.md](CLAUDE.md) — full agent development guide: stack, project layout,
  backend/frontend conventions, data model, known issues.
- [ROADMAP.md](ROADMAP.md) — public-facing development direction (F1–F4).
- [INSTALL.md](INSTALL.md) — platform install instructions.
- `docs/` — `CLI.md`, `DISTRIBUTION.md`, `QUICKSTART.md`, `SETTINGS.md`, and
  `docs/user docs/`.
- [CHANGELOG.md](CHANGELOG.md) — release history (current version 1.3.0).

## (c) Key domain concepts

- **Items + Notes** — one model. The SQLite table is `todos`; the Go struct is
  `store.Item`. The `type` column is `todo` or `note`. (Table name is legacy and
  deliberately not renamed without a migration plan.)
- **Vaults** — each vault is a self-contained SQLite database plus file tree;
  multi-vault is supported. Portable, offline-first.
- **Quick Add syntax** — todo.txt-inspired: `(A) Buy milk +project @context #tag
  due:2026-02-25 t:2026-02-20 rec:2w`. Parsed in `parse/line.go`.
- **Scopes** — `now` / `soon` / `anytime` views for managing focus.
- **Sessions** — a temporary context filter; new items auto-inherit session tags.
- **Inbox** — fast capture with zero taxonomy friction (F1, MVP shipped). Empty
  or deliberately-routed items are flagged `inbox` and excluded from normal
  search/views until triaged.
- **Wikilinks** — `@reference` chips link items, with backlinks. Built on
  TipTap's Mention extension.
- **Search** — FTS5 full-text search shipped. Semantic/embedding search is
  proposed (ADR-004) but not yet built.
- **Terminal aesthetic** — the custom CSS theme system (`--term-*` variables) is
  intentional product design, not tech debt. A Tailwind v4 + shadcn/ui migration
  is planned that must preserve it.
- **MCP server** — `cmd/nil-mcp` exposes vault content to other agents via 12
  JSON-RPC tools (`nil_search`, `nil_create_item`, `nil_list_inbox`, etc.).

## (d) Common operations

```bash
make help            # canonical local workflow surface
make verify          # full local verification path (lint, test, frontend build,
                     #   binding drift check, wails build)
make build           # build the desktop app (wails build)
make test            # Go test suite (go test ./...)
make lint            # Go format + lint + frontend lint
make codegen         # regenerate Wails TS bindings (wails generate module)
make codegen-check   # regenerate + fail if tracked bindings changed

wails dev            # run in dev mode with hot reload
./build-local.sh     # quick local macOS build
./build-for-friends.sh   # cross-platform distribution build

DEBUG=1 open build/bin/NIL.app   # launch with WebKit inspector
```

Examples of agent intents:

- *Add a backend method* → add it to `app.go` on `*App`, then run
  `make codegen` so the TypeScript bindings stay in sync.
- *Change the DB schema* → edit `store/schema.sql` (fresh installs) **and** add a
  migration entry in `store/store.go` with an incremented `currentSchemaVersion`.
- *Run NIL as a tool for another agent* → build and run the `nil-mcp` binary
  (`cmd/nil-mcp`); it speaks MCP over stdio.

## (e) Where to look for more

- Code conventions, data model details, known issues: [CLAUDE.md](CLAUDE.md).
- Build/distribution specifics: `docs/DISTRIBUTION.md`.
- CLI surface: `docs/CLI.md`.
- Roadmap and feature staging: [ROADMAP.md](ROADMAP.md).
- Project source-of-truth metadata for agents: `.agent-ops/project.yaml`.
- Portfolio-level knowledge (composition points, gaps, integration paths):
  `~/dev/agent-os/knowledge/projects/nil.md`.
- Current active priorities for sessions: `.agentrc/boot-prompt.md` (note: the
  big open item is a frontend Biome cleanup pass blocking clean commits).
