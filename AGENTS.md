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
the macOS bundle is `NIL.app`. The SQLite table is still named `todos` even
though it holds both todos and notes — renaming would require a costly data
migration with no user-visible benefit. That is known drift, not a signal to
rename.

## (b) Where to start

Entry points:

- `main.go` — Wails entry point, window configuration.
- `app.go` — the `App` struct; every Wails-bound Go method lives here.
- `apiserver/apiserver.go` — HTTP API surface (routes/handlers/auth); used both by the GUI's embedded server and the standalone `nil serve-api` CLI subcommand.
- `cli/cli.go` — CLI command registry (`nil <command>`); package `cli`.
- `cmd/nil-mcp/` — standalone MCP stdio server (`main.go` + `mcp.go`),
  builds to the `nil-mcp` binary.
- `cmd/nil-recover/` — offline recovery tool that runs the v11 backfill on a
  closed DB. Used during the v1.3.0 → v1.3.1 incident; keep as a break-glass
  utility.
- `ingest/` — Go-side document conversion package. `MarkdownToDoc` (goldmark),
  `HTMLToDoc` (x/net/html, handles wikilink spans), `DocToHTML`,
  `DocToPlainText`, `ExtractRefIDs`. Every consumer that accepts raw markdown
  or HTML body input (HTTP API, CLI, MCP via HTTP, chat-bridge AI tools) routes
  through here.
- `frontend/src/pages/App.tsx` — root React component; single source of truth
  for app-level state and top-level event wiring. Read it fully before adding
  state or handlers.
- `frontend/src/lib/backend.ts` — typed `Partial<>` wrappers around the
  generated `Backend.Search` / `Backend.UpdateItem` / `Backend.GetInboxItems`
  calls. Use these instead of casting payloads with `as any` — the cast hides
  field-name typos at compile time (the v1.3.2 silent-search-bug class).

Docs:

- [README.md](README.md) — user-facing overview, shortcuts, Quick Add syntax,
  dev commands.
- [CLAUDE.md](CLAUDE.md) — full agent development guide: stack, project layout,
  backend/frontend conventions, data model, known issues.
- [ROADMAP.md](ROADMAP.md) — public-facing development direction (F1–F4).
- [INSTALL.md](INSTALL.md) — platform install instructions.
- `docs/` — `CLI.md`, `DISTRIBUTION.md`, `QUICKSTART.md`, `SETTINGS.md`, and
  `docs/user docs/`.
- [CHANGELOG.md](CHANGELOG.md) — release history (current version 1.3.3).

## (c) Key domain concepts

- **Items + Notes** — one model. The SQLite table is `todos`; the Go struct is
  `store.Item`. The `kind` column (renamed from `type` in schema v9) holds
  `todo`, `note`, or `scratch`; values are validated against the `kinds`
  registry table at the Go boundary. New built-in or plugin-supplied kinds
  register there. (Table name is legacy and deliberately not renamed without a
  migration plan.)
- **Notes storage (JSON-at-rest)** — TipTap notes are stored as ProseMirror
  document JSON in `notes_doc` (canonical), with `notes_html` as a write-time
  cache for fast previews and `notes_html_version` for cache invalidation. The
  earlier HTML-in-`notes_md` shape was retired in v7–v11; `notes_md` is
  deadweight until a v12 drop. Frontend save extracts both `editor.getJSON()`
  and `editor.getHTML()`; the dirty-check is JSON deep-equal. Servers that
  accept raw markdown/HTML go through `ingest/` to produce `notes_doc`.
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
- **MCP server** — `cmd/nil-mcp` exposes vault content to other agents via 15
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

**Git hooks (lefthook)** — one-time setup: run `lefthook install` from the repo
root. This activates `lefthook.yml`'s pre-commit (Go format/lint/vet +
frontend lint on staged files) and pre-push (`go test` + `make codegen-check`
— the Wails binding drift check) hooks. `make verify`/`make codegen-check`
already catch drift locally when a human remembers to run them; the pre-push
hook is what enforces it automatically. If hooks silently never fire, check
`git config core.hooksPath` — it should be unset or point at `.git/hooks`
(the default). A leftover absolute path from a previous clone location (e.g.
after moving/renaming the repo directory) silently disables all hooks with no
error; fix with `git config --unset core.hooksPath` then re-run
`lefthook install`.

Examples of agent intents:

- *Add a backend method* → add it to `app.go` on `*App`, then run
  `make codegen` so the TypeScript bindings stay in sync.
- *Change the DB schema* → edit `store/schema.sql` (fresh installs) **and** add a
  migration entry in `store/store.go` with an incremented `currentSchemaVersion`.
  Migrations should log convertErrs/ftsErrs counters to stderr and continue
  rather than aborting — that pattern is established in `migrateV11`. Never
  put data-shaping logic in a frontend `useEffect` gated on a localStorage
  flag; see the v1.3.0 incident in CHANGELOG.
- *Run NIL as a tool for another agent* → build and run the `nil-mcp` binary
  (`cmd/nil-mcp`); it speaks MCP over stdio.
- *Pass a partial search/update to the backend from the frontend* → use the
  `search` / `updateItem` / `getInboxItems` helpers in
  `frontend/src/lib/backend.ts`. Do **not** cast payloads with `as any`.

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
