---
intent: project_doc
class: plan
feature: F5 — Vault Chat
created_at: 2026-02-22
iteration: 1
---

# Plan — F5: Vault Chat

## Milestones

### M0 — Foundation (Schema + Data Layer) [Iterations 2–4]
**Goal**: All backend plumbing for chat, proposals, and audit works end-to-end.

Deliverables:
- `chat/` package: `ChatStore` (SQLite), `ChatSession`, `ChatMessage`, `ActionProposal`, `AuditEntry` structs
- `store/schema.sql` updated with `chat_` tables (fresh install)
- Schema migration v5: add chat tables to existing installs
- `config.ChatConfig` + `VaultCap` added to `config.Config`
- `chat/bridge.go`: LLM bridge (Claude API call, parse JSON action from response)
- `app.go`: `StartChatSession`, `EndChatSession`, `SendChatMessage`, `ApproveChatAction`, `DenyChatAction`, `GetChatHistory`, `GetActionAudit`, `GetChatConfig`, `SetChatConfig`
- `wails generate module` run; TypeScript bindings updated
- `go build .` passes

Verification: Unit-testable `ChatStore` CRUD; `go build .` green; TS bindings match Go signatures.

---

### M1 — MVP Frontend [Iterations 5–7]
**Goal**: Chat panel visible and functional in the app; full create/update/delete flow works.

Deliverables:
- `ChatPanel.tsx`: floating panel, message thread, input, session lifecycle
- `ChatMessage.tsx`: renders user/assistant turns
- `ActionCard.tsx`: renders proposal with diffs, approve/deny buttons
- `App.tsx`: `Cmd+Shift+C` shortcut wired; `ChatPanel` mounted conditionally
- Vault selector in chat context (read from `GetVaults`, show active vault name)
- Error state for missing API key
- Dry-run mode indicator in chat header

Verification: Open chat → type query → see results. Request action → see ActionCard → approve → item appears in vault. Deny → no side effect. Dry-run → proposal shown, execution blocked.

---

### M2 — Settings + Audit [Iteration 8]
**Goal**: Users can configure chat and inspect audit log from Settings.

Deliverables:
- Settings → Chat tab: API key field (masked), model selector, dry-run toggle, vault capability table (read/write/delete per vault)
- Audit log view (accessible from Settings or chat panel footer): last N actions, type, outcome, timestamp
- Graceful error handling: API errors, network failures, malformed LLM responses

Verification: Change API key in Settings → chat uses new key. Enable write for a vault → can approve create actions. Audit log shows entries.

---

### v1.0.0 Targets [Post-M2 Backlog Promotion]
These items are captured in backlog and can be promoted after M2:
- Named/saved chat sessions
- Chat-driven inbox triage mode
- Multi-vault cross-search in single query
- Chat transcript export (Markdown)
- Deterministic action classifier (keyword rules → taxonomy auto-assign)

---

## Task Breakdown (MVP, M0 + M1 + M2)

See `.forge/tasks/` for full task files.

| ID | Priority | Title | Milestone |
|---|---|---|---|
| TASK-20260222-001 | A | Chat DB schema + migration (v5) | M0 |
| TASK-20260222-002 | A | Go: ChatMessage model + Store methods | M0 |
| TASK-20260222-003 | A | Go: LLM bridge (Claude API + parse action) | M0 |
| TASK-20260222-004 | A | Go: ActionProposal flow (propose/approve/execute/audit) | M0 |
| TASK-20260222-005 | A | Wails bindings: expose chat + action methods | M0 |
| TASK-20260222-006 | B | Frontend: ChatPanel component | M1 |
| TASK-20260222-007 | B | Frontend: ActionCard component | M1 |
| TASK-20260222-008 | B | Frontend: wire ChatPanel into App.tsx | M1 |
| TASK-20260222-009 | B | Frontend: vault selector in chat | M1 |
| TASK-20260222-010 | C | Settings: chat config section | M2 |

## Dependency Graph
```
001 (schema) → 002 (store) → 003 (LLM bridge) → 004 (proposal flow) → 005 (Wails bindings)
                                                                              ↓
                                                              006 (ChatPanel) → 008 (App.tsx wire)
                                                              007 (ActionCard) ↗
                                                              009 (vault selector) → 008
                                                              010 (Settings) (independent after 005)
```

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| WKWebView click issues in ActionCard | Medium | Use button elements (not divs); test in actual app early |
| LLM response format drift | Medium | Strict JSON schema in system prompt; fallback to plain text if parse fails |
| Claude API key UX friction | Low | Clear error message in chat panel; link to Settings |
| chat.db migration fails | Low | IF NOT EXISTS guards on all tables; test with existing install |

## Evidence
- Created: 2026-02-22 (Inception iteration 1)
- Informed by: spec-lite.md, prd-lite.md, mvp.md, CLAUDE.md architecture
