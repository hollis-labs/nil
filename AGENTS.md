# NIL

NIL is a keyboard-driven desktop app for personal tasks and notes: a Wails v2
shell around a Go backend and a React/TypeScript frontend rendered in a WebView.
Tasks and notes are one object with different affordances — a single
`kind`-discriminated table, todo.txt-inspired capture syntax, FTS5 search, and
one SQLite file per vault that the user owns outright. It is not a server, a
sync service, or multi-user: the HTTP API binds `127.0.0.1`, and the only host
NIL ever calls out to is `api.anthropic.com`, from the chat bridge. The app was
called PLANCK and then NANITE; "Nanite" now names an unrelated CLI agent
framework in another repo.

## Start Here

- `docs/architecture/ARCHITECTURE.md` is the canonical system model — the five
  surfaces, data flow, and trust boundaries. Read it before changing structure.
- `docs/adr/README.md` indexes the decisions and why they were made.
- `docs/CONVENTIONS.md` holds frontend/data working details and open known
  issues.
- `main.go` dispatches a bare subcommand to `cli.Run`, otherwise starts the GUI.
- `app.go` and `app_*.go` hold every Wails-bound method on `*App`.
- `service/items/items.go` is the shared create/update/search layer. All four
  surfaces route through it; new cross-surface behavior belongs here, not in a
  consumer.
- `apiserver/apiserver.go`, `cli/cli.go`, `cmd/nil-mcp/`, `chat/bridge.go` are
  those surfaces.
- `store/schema.sql` and `store/migrations.go` own the schema.
- `ingest/` converts markdown and HTML to the stored document JSON; every
  surface that accepts a raw body goes through it. `parse/line.go` owns Quick
  Add syntax.
- `frontend/src/pages/App.tsx` holds app-level state and top-level wiring. Read
  it before adding either.

## Commands

```bash
make verify          # lint, tests, frontend build, binding drift check, build
make test            # go test ./... plus the Vitest suite
make codegen-check   # regenerate Wails bindings, fail on drift
wails dev            # hot-reload dev mode
```

`make verify` is the gate before a commit. Run `lefthook install` once to wire
pre-commit lint and a pre-push `go test` + `make codegen-check`. If hooks never
fire, check `git config core.hooksPath` for a stale absolute path left by an
earlier clone location — it disables every hook with no error.

## Boundaries

`frontend/wailsjs/` is generated; never hand-edit it. Any change to a Go type
carried by a Wails-bound signature — `store.Item`, `store.SearchRequest` —
leaves those bindings stale, and `go build` and `go test` both pass anyway. Run
`make codegen` after such a change; `make codegen-check` is the only thing that
catches it.

A schema change takes two edits: `store/schema.sql` for fresh installs, and a
migration in `store/migrations.go` with `currentSchemaVersion` bumped.
`schema.sql` execs unconditionally on every `Open()` *before* migrations run, so
an index on a column a migration has yet to add must be created in `Open()`
after `runMigrations` returns, not in `schema.sql` — `store/store.go:97` is the
worked example.

The SQLite table is `todos` and holds every kind, notes included; the Go struct
is `store.Item`. The mismatch is deliberate. Renaming needs a migration plan.

Tailwind and shadcn/ui are not installed. Color and chrome come from the
`--term-*` custom properties under `frontend/src/theme/`; never hardcode them. A
migration that preserves that system is planned but has not landed.

Commit `6a9ef34` deleted `.forge/` and `.agentrc/`, and a later pass removed the
last config that still named them. A path cited in a comment, a config file or a
doc is not evidence that the path exists — check before following one.
