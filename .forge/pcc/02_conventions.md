---
intent: pcc
updated_at: 2026-02-22
---

# 02 — Conventions

## Backend (Go)
- All Wails-exposed methods → `app.go` as methods on `*App`
- All DB logic → `store/store.go` as methods on `*Store`
- New tables → add to `store/schema.sql` (fresh installs) AND migration slice in `store/store.go`
- Migration: increment `currentSchemaVersion`, add idempotent entry (IF NOT EXISTS guards)
- After Go method signature changes → run `wails generate module` to regenerate TS bindings
- Build test: `go build .` from repo root

## Frontend (TypeScript/React)
- **Never edit** `frontend/wailsjs/` manually (auto-generated)
- Import bindings: `import * as Backend from "../../wailsjs/go/main/App"`
- Path alias: `@` → `frontend/src/`
- All global state → `src/pages/App.tsx` (read fully before adding state/handlers)
- Modal components: receive open/close state + callbacks as props; no self-managed visibility
- Styling: CSS custom properties only (`var(--term-bg)`, `var(--term-fg)`, `var(--term-accent)`, etc.)
- **No Tailwind/shadcn** until migration is planned and approved
- localStorage prefix: `nanite.` (e.g., `nanite.settings`, `nanite.viewMode`)

## Styling Constraints
- Always use `var(--term-*)` variables — never hardcode colors
- Theme variables live in `src/theme/theme.css`; type in `src/theme/theme.ts`
- Theme switcher hidden from Settings UI until Tailwind/shadcn migration

## Chat Addon — Additional Conventions
- Chat transcripts → separate DB (NOT user vault); default path in config dir
- Action Proposal → Approval → Execute: every mutating action requires explicit approval
- Audit log: every proposed/executed action gets a record with timestamp + actor + vault
- Dry-run mode: proposal only, no execution
- No actions outside repo root unless explicitly allowed in config
- Capability boundaries in config: which vaults, which operations (read|write|delete)
- Claude API key stored in `config.Config` (not env var); never logged

## Git / Forge
- Commit per Forge iteration (iteration mode)
- Branch prefix: `forge/`
- No worktrees currently (Wails dev env complication)
- PR mode: optional (not required for every task)

## Evidence
- Inspected: CLAUDE.md, app.go, store/store.go, config/config.go
- Updated: 2026-02-22
