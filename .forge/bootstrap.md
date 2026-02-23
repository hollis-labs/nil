---
intent: bootstrap
iteration: 10
generated: 2026-02-22
source_of_truth: forge.yaml, .forge/tasks/, .forge/pcc/, .forge/logs/, .forge/epics/, .forge/adr/
---

# NANITE Forge Bootstrap

**Iteration**: 10 (next to run)
**Generated**: 2026-02-22
**Active Epic**: EPIC-E1 — Chat Intelligence (Sprint 3)

---

## Task Counts

| Status | Count |
|---|---|
| todo | 0 (Sprint 2 complete) |
| done | 17 (TASK-001 through TASK-017) |
| backlog | 9 (E1 sprints 3–5) |

---

## Sprint 2 — Complete ✓

| Task | Status |
|------|--------|
| TASK-015: Vault Profile in system prompt | ✓ done |
| TASK-016: Nanite object glossary | ✓ done |
| TASK-017: `create_item` tool + DirectCreate permission tier | ✓ done |

---

## Sprint 3 — Templates (Next)

| Task | Priority | Backlog File |
|------|----------|-------------|
| TASK-018: Template storage (SQLite table + CRUD) | B | BACKLOG-E1-S3-001.md |
| TASK-019: Template tools (`list_templates`, `use_template`) | B | BACKLOG-E1-S3-002.md |
| TASK-020: Save-prompt UX when agent uses new pattern | B | BACKLOG-E1-S3-003.md |
| TASK-021: Template management UI in Settings | C | BACKLOG-E1-S3-004.md |

---

## Epic & ADR Index

| File | Purpose |
|------|---------|
| `.forge/epics/EPIC-E1-chat-intelligence.md` | Full 5-sprint roadmap for chat |
| `.forge/adr/ADR-005-template-storage.md` | Template storage format |

---

## Blockers

- None. Sprint 3 TASK-018 is unblocked.

---

## Quick Start (Iteration 10)

```
# Iteration 10 — Sprint 3, TASK-018: Template storage
# 1. Read .forge/backlog/BACKLOG-E1-S3-001.md for full spec
# 2. Read chat/store.go for existing chat.db schema patterns
# 3. Add templates table to chat DB schema + migration
# 4. Add Template struct to chat/models.go
# 5. Add CRUD methods: SaveTemplate, GetTemplate, ListTemplates, DeleteTemplate
# 6. Expose via Wails: SaveChatTemplate, GetChatTemplates, DeleteChatTemplate in app.go
# 7. wails generate module + go build . + npx tsc --noEmit
```

---

## Config Snapshot

- Storage: file backend → `.forge/tasks/`
- PCC: `.forge/pcc/` (5 files) ✓
- Milestones: M0 ✓ → M1 ✓ → M2 ✓ → M3 ✓ → E1-S1 ✓ → E1-S2 ✓ → E1-S3 (starting)
- Git: branch `main`
- Verified: `go build .` clean (iter 9); `npx tsc --noEmit` clean (iter 9)
