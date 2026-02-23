---
intent: bootstrap
iteration: 12
generated: 2026-02-22
source_of_truth: forge.yaml, .forge/tasks/, .forge/pcc/, .forge/logs/, .forge/epics/, .forge/adr/
---

# NANITE Forge Bootstrap

**Iteration**: 12 (next to run)
**Generated**: 2026-02-22
**Active Epic**: EPIC-E1 — Chat Intelligence (Sprint 3)

---

## Task Counts

| Status | Count |
|---|---|
| todo | 0 (Sprint 3 in progress) |
| done | 19 (TASK-001 through TASK-019) |
| backlog | 7 (E1 sprints 3–5, minus TASK-018/019) |

---

## Sprint 3 — In Progress

| Task | Status |
|------|--------|
| TASK-018: Template storage (SQLite table + CRUD) | ✓ done |
| TASK-019: Template tools (`list_templates`, `use_template`) | ✓ done |
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

- None. TASK-020 is unblocked.

---

## Quick Start (Iteration 12)

```
# Iteration 12 — Sprint 3, TASK-020: Save-prompt UX
# 1. Read .forge/backlog/BACKLOG-E1-S3-003.md for full spec
# 2. Understand the suggest_template action block pattern from ADR-005
# 3. Add suggest_template action type to extractAction() in chat/bridge.go
# 4. Add SaveTemplateSuggestion struct to chat/models.go
# 5. Update ChatResponse to carry TemplateSuggestion when detected
# 6. Frontend: detect suggestion in ChatPanel, show inline save prompt
# 7. go build . + npx tsc --noEmit
```

---

## Config Snapshot

- Storage: file backend → `.forge/tasks/`
- PCC: `.forge/pcc/` (5 files) ✓
- Milestones: M0 ✓ → M1 ✓ → M2 ✓ → M3 ✓ → E1-S1 ✓ → E1-S2 ✓ → E1-S3 (in progress)
- Git: branch `main`
- Verified: `go build .` clean (iter 11); `npx tsc --noEmit` clean (iter 11)
