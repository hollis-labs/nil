---
intent: pcc_global
project: nanite
updated_at: "2026-03-04"
---

# Conventions

## Backend (Go)

- All Wails-exposed methods in `app.go` as methods on `*App`
- All DB logic in `store/store.go` as methods on `*Store`
- Schema in `store/schema.sql` (embedded at compile time)
- Schema changes: increment `currentSchemaVersion` in `store/store.go`, add idempotent migration entry
- After Go method changes: run `wails generate module` to regenerate TypeScript bindings

## Frontend (TypeScript/React)

- `frontend/wailsjs/` is auto-generated -- never edit manually
- Path alias `@` resolves to `frontend/src/`
- All app state and top-level handlers in `src/pages/App.tsx`
- Styling: CSS custom properties (`var(--term-*)`) -- no hardcoded colors
- Tailwind/shadcn NOT installed yet; migration planned
- localStorage keys prefixed with `nanite.`

## Naming conventions

- DB table: `todos` (not renamed; Go struct is `store.Item`)
- Wails methods: `CreateItemFromLine`, `UpdateItem`, etc.
- Item types: `'todo'` or `'note'` (DB column values)
- Theme variables: `--term-bg`, `--term-fg`, `--term-accent`, etc.

## Test commands

```bash
wails dev                          # dev mode with hot reload
./build-local.sh                   # build macOS app
./build-for-friends.sh             # cross-platform distribution
wails generate module              # regenerate TS bindings
cd frontend && npx tsc --noEmit    # type-check frontend
DEBUG=1 open build/bin/NANITE.app  # debug mode (WebKit inspector)
```

## Build and distribution

- `go build .` from project root for Go layer only
- Wails handles full app bundling
- Cross-platform: macOS + Windows via `build-for-friends.sh`; Linux requires Linux build host

## Evidence
- Last refreshed: 2026-03-04 (mentat PCC bootstrap)
- Sources: CLAUDE.md, README.md, build-local.sh, build-for-friends.sh
