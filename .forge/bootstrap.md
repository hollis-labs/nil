---
intent: bootstrap
iteration: 4
generated: 2026-02-22
source_of_truth: forge.yaml, .forge/tasks/, .forge/pcc/, .forge/logs/
---

# NANITE Forge Bootstrap

**Iteration**: 4 (next to run)
**Generated**: 2026-02-22
**Feature**: F5 — Vault Chat (M1 complete; one task remains for M2)

---

## Task Counts

| Status | Count |
|---|---|
| todo | 1 |
| done | 9 (TASK-001 through 009) |
| backlog | 5 |
| **total tasks** | **10** |

---

## Top Next Task

1. **TASK-20260222-010** (C) — Settings: chat config section
   - Add "Chat" tab to SettingsModal
   - API key field (masked), model selector, dry-run toggle, vault capability table
   - Read `SettingsModal.tsx` fully before modifying (it is large)
   - ← **START HERE**

After TASK-010: **all 10 MVP tasks complete** — MVP Definition of Done can be verified.

---

## Backlog (not yet promoted)

| ID | Priority | Title |
|---|---|---|
| BACKLOG-20260222-001 | B | Saved named chat sessions |
| BACKLOG-20260222-002 | B | Chat-driven inbox triage mode |
| BACKLOG-20260222-003 | C | Multi-vault cross-search |
| BACKLOG-20260222-004 | C | Deterministic action classifier |
| BACKLOG-20260222-005 | C | Chat transcript export |

---

## Blockers

- None. TASK-010 is unblocked (depends only on TASK-005, which is done).

---

## M1 Summary (iteration 3)

Complete frontend for F5 Vault Chat:
- `ActionCard.tsx`: create/update/delete proposal renderer with approve/deny (WKWebView-safe `<button>`)
- `ChatMessage.tsx`: user/assistant turn with embedded ActionCard for proposals
- `ChatPanel.tsx`: right-side drawer (400px), session lifecycle, vault selector, thread, input
- `App.tsx`: `chatOpen` state, `Cmd+Shift+C` shortcut, `ChatPanel` mounted
- TypeScript: `npx tsc --noEmit` clean ✓ | Go: `go build .` clean ✓

---

## Latest Run Log

`.forge/logs/run-20260222-iter03.md`
4 tasks completed (TASK-006–009): ActionCard, ChatMessage, ChatPanel, App.tsx wiring + vault selector.

---

## Quick Start (Iteration 4)

```
# Iteration 4 — M2 Settings
# TASK-20260222-010: Settings chat config section
# 1. Read frontend/src/components/SettingsModal.tsx fully
# 2. Add "Chat" tab type to SettingsTab union
# 3. Add tab button in tab bar
# 4. Add chat config section content:
#    - API key (password input, masked)
#    - Model selector (haiku / sonnet)
#    - Dry-run toggle
#    - Vault caps table (per vault: Read/Write/Delete checkboxes)
# 5. Load via Backend.GetChatConfig(); save via Backend.SetChatConfig()
# 6. npx tsc --noEmit + go build . green
# After: MVP complete — verify DoD checklist in .forge/artifacts/mvp.md
```

---

## Config Snapshot

- Storage: file backend → `.forge/tasks/`
- PCC: `.forge/pcc/` (5 files) ✓
- Milestones: M0 ✓ → M1 ✓ → M2 (1 task remaining)
- Git: branch `main`; iteration 3 commit pending
- Verified: `go build .` clean; `npx tsc --noEmit` clean
