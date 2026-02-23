---
intent: bootstrap
iteration: 11
generated: 2026-02-22
source_of_truth: forge.yaml, .forge/tasks/, .forge/pcc/, .forge/logs/, .forge/epics/, .forge/adr/
---

# NANITE Forge Bootstrap

**Iteration**: 11 (next to run)
**Generated**: 2026-02-22
**Active Epic**: EPIC-E1 — Chat Intelligence (Sprint 3)

---

## Task Counts

| Status | Count |
|---|---|
| todo | 0 (Sprint 3 in progress) |
| done | 18 (TASK-001 through TASK-018) |
| backlog | 8 (E1 sprints 3–5, minus TASK-018) |

---

## Sprint 3 — In Progress

| Task | Status |
|------|--------|
| TASK-018: Template storage (SQLite table + CRUD) | ✓ done |
| TASK-019: Template tools (`list_templates`, `use_template`) | backlog |
| TASK-020: Save-prompt UX when agent uses new pattern | backlog |
| TASK-021: Template management UI in Settings | backlog |

---

## Epic & ADR Index

| File | Purpose |
|------|---------|\
| `.forge/epics/EPIC-E1-chat-intelligence.md` | Full 5-sprint roadmap for chat |
| `.forge/adr/ADR-005-template-storage.md` | Template storage format |

---

## Blockers

- None. TASK-019 is unblocked.

---

## Quick Start (Iteration 11)

```
# Iteration 11 — Sprint 3, TASK-019: Template tools
# 1. Read .forge/backlog/BACKLOG-E1-S3-002.md for full spec
# 2. Read chat/bridge.go for current buildTools() and executeTool patterns
# 3. Add list_templates tool: calls ChatStore.ListTemplates via BridgeStore extension (or pass store separately)
# 4. Add use_template tool: renders {{param}} substitution, returns rendered prompt
# 5. Update BridgeRequest to carry TemplateStore (or extend BridgeStore interface)
# 6. go build . + npx tsc --noEmit
```

---

## Config Snapshot

- Storage: file backend → `.forge/tasks/`
- PCC: `.forge/pcc/` (5 files) ✓
- Milestones: M0 ✓ → M1 ✓ → M2 ✓ → M3 ✓ → E1-S1 ✓ → E1-S2 ✓ → E1-S3 (in progress)
- Git: branch `main`
- Verified: `go build .` clean (iter 10); `npx tsc --noEmit` clean (iter 10)
