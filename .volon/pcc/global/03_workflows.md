---
intent: pcc_global
project: nanite
updated_at: "2026-03-04"
---

# Workflows

## Development

```bash
# Run in dev mode (hot reload)
wails dev

# Build macOS app locally
./build-local.sh

# Cross-platform distribution build (macOS + Windows)
./build-for-friends.sh

# Regenerate Wails TypeScript bindings after Go changes
wails generate module

# Type-check frontend only
cd frontend && npx tsc --noEmit
```

## Debug mode

```bash
DEBUG=1 open build/bin/NANITE.app   # opens WebKit inspector on startup
```

Attach Safari devtools via `Develop > [device]` for WKWebView debugging.

## API server

The REST API starts automatically with the app, bound to `127.0.0.1` on a configured port. Endpoints:

- `POST /api/v1/inbox` -- create inbox item
- `GET /api/v1/inbox` -- list inbox items
- `GET /api/v1/search` -- search items
- `GET /api/v1/items/{id}` -- get item by ID

All endpoints require bearer-token auth.

## CLI

```bash
nanite <command>          # headless CLI operations
nanite vaults             # list vaults and IDs
nanite push <content>     # push item to inbox
```

The `nanite` binary is typically symlinked from the macOS app bundle to `/usr/local/bin/nanite`.

## Quick-add syntax

```
(A) Buy milk +shopping @errands #personal due:2026-02-25
```

Supports: priority `(A/B/C)`, `+project`, `@context`, `#tag`, `due:YYYY-MM-DD`, `t:YYYY-MM-DD` (threshold), `rec:Nd/Nw/Nm` (recurrence).

## Evidence
- Last refreshed: 2026-03-04 (mentat PCC bootstrap)
- Sources: README.md, CLAUDE.md, api.go, cli/cli.go
