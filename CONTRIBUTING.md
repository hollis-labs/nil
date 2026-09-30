# Contributing

How a change gets from your clone into `main`. This is deliberately short: most
of what you need is written somewhere closer to the thing it describes.

## Before your first change

`README.md` has the install and development commands. Run `lefthook install`
once per clone — a tracked `lefthook.yml` installs no hooks by itself, and a
clone that skips it has no pre-commit lint or pre-push checks and says nothing
about it. If hooks never fire, check `git config core.hooksPath` for a stale
path left by an earlier clone location.

`AGENTS.md` is the fastest orientation to the layout, and
`docs/architecture/ARCHITECTURE.md` is the canonical system model. Read the
latter before changing structure.

## The sequence

1. **Branch.** `<type>/<short-slug>`, where the type matches the change —
   `feat`, `fix`, `docs`, `chore`. Nothing enforces this; it is what the
   history does.
2. **Change one thing.** A branch carrying two unrelated changes costs the
   reviewer the ability to accept one and question the other.
3. **Run the gate** before you push:
   ```
   make verify
   ```
   It runs Go format and lint checks, frontend lint, the Go and Vitest suites,
   a frontend production build, the Wails binding drift check, and a full
   build.
4. **Push and open a pull request.** A maintainer will review it.

Commit subjects follow the conventional-commit shape — a type, an optional
scope, a colon, then the summary.

## What a pull request should carry

The reviewer was not there when you made the decisions. State what the change
does, what it deliberately leaves alone, and the evidence that it works — the
commands you ran and what came back, not a claim that it passes. For a UI
change, describe what you saw running under `wails dev`.

Add a line to `CHANGELOG.md` under `[Unreleased]`.

## The one that cannot be undone

**Schema changes.** Each vault is a SQLite file its owner keeps, and NIL
migrates it in place. A schema change takes two edits — `store/schema.sql` for
fresh installs and a migration in `store/migrations.go` with
`currentSchemaVersion` bumped — and a migration that has shipped must never be
edited. `schema.sql` runs before migrations on every open, so an index on a
column a migration has yet to add belongs in `Open()` after `runMigrations`
(see `store/store.go`). Test a migration against a copy of an older vault, not
only a fresh one.

## Things that surprise people

- **`frontend/wailsjs/` is generated.** Changing a Go type carried by a
  Wails-bound signature leaves the bindings stale while `go build` and
  `go test` still pass. Run `make codegen`; `make codegen-check` is the only
  thing that catches it.
- **The table is `todos`, the Go struct is `store.Item`**, and notes live in
  the same table. The mismatch is deliberate; renaming needs a migration plan.
- **New cross-surface behavior goes in `service/items`**, not in the GUI, CLI,
  HTTP API, MCP server or chat bridge that calls it.
- **No Tailwind or shadcn.** Colors come from the `--term-*` custom properties
  under `frontend/src/theme/`; do not hardcode them.

## What this does not cover

- **Which change is worth making.** Open an issue to discuss larger features
  before building them.
- **Release builds and signing.** Maintainers handle those; see
  `docs/DISTRIBUTION.md`.
