---
intent: pcc
updated_at: 2026-02-22
---

# 04 — Backlog

## Active Sprint: MVP Chat Addon (F5)

### MVP Tasks (promoted, in `.forge/tasks/`)
See `.forge/tasks/` for full task files. Summary:

| ID | Priority | Title | Status |
|---|---|---|---|
| TASK-20260222-001 | A | Chat DB schema + migration (v5) | todo |
| TASK-20260222-002 | A | Go backend: ChatMessage model + Store methods | todo |
| TASK-20260222-003 | A | Go backend: LLM bridge (Claude API call + streaming) | todo |
| TASK-20260222-004 | A | Go backend: ActionProposal flow (propose/approve/execute/audit) | todo |
| TASK-20260222-005 | A | Wails bindings: expose chat + action methods to frontend | todo |
| TASK-20260222-006 | B | Frontend: ChatPanel component (message thread + input) | todo |
| TASK-20260222-007 | B | Frontend: ActionCard component (inline approve/deny UI) | todo |
| TASK-20260222-008 | B | Frontend: wire ChatPanel into App.tsx (Cmd+Shift+C shortcut) | todo |
| TASK-20260222-009 | B | Frontend: vault selector in chat context | todo |
| TASK-20260222-010 | C | Settings: chat config section (API key, model, dry-run, vault caps) | todo |

### Backlog (not yet promoted)
| ID | Priority | Title |
|---|---|---|
| BACKLOG-20260222-001 | B | Saved chat sessions (named conversations) |
| BACKLOG-20260222-002 | B | Chat-driven inbox triage mode |
| BACKLOG-20260222-003 | C | Multi-vault cross-search in chat |
| BACKLOG-20260222-004 | C | Deterministic action classifier (keyword rules → taxonomy auto-assign) |
| BACKLOG-20260222-005 | C | Chat transcript export (Markdown) |

### Non-goals for MVP
- No AI-assisted routing or batch triage (backlog)
- No streaming responses to UI (polling or full-response only for MVP)
- No voice input
- No third-party LLM providers (Claude only for MVP)
- No shared/cloud chat history

## Evidence
- Updated: 2026-02-22 (Inception iteration 1)
