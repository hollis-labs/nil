---
intent: pcc
updated_at: 2026-02-22
---

# 00 — Project

## Identity
- **Name:** NANITE
- **Type:** Keyboard-driven personal task & note management desktop app
- **Stage:** Public beta; single-user, local-first
- **Framework:** Wails v2 (Go backend + WebView frontend)

## Active Addon Work
- **Chat Addon (F5 — Vault Chat):** Inception started iteration 1
  - Goal: Chat UI to search/CRUD vault content via natural language, with Action Proposal → Approval → Execute flow
  - Status: Artifacts phase complete; execution tasks defined

## Completed Features
- F1 Inbox (MVP shipped): fast capture, triage, `GetInboxCount/Items/ProcessInboxItem`
- Vault system: multi-vault registry, `vault.Manager`, active/inbox store routing
- HTTP API server (`api.go`): Bearer-auth CRUD + search + inbox endpoints (port 7765)
- CLI subcommands: `push`, `search`, `inbox`, `get`, `vaults` (separate binary)

## Active Config
- Config: `forge.yaml` at repo root
- Storage: file backend → `.forge/tasks/`
- PCC: `.forge/pcc/` (committed to git)
- Git: no worktrees (Wails dev environment), PR mode optional, branch prefix `forge/`
- Observability: verbose, run + decision logs enabled

## Evidence
- Bootstrapped: 2026-02-22 (Inception iteration 1, chat addon)
- Inspected: app.go, store/models.go, store/schema.sql, vault/manager.go, config/config.go
