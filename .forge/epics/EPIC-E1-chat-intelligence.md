---
id: EPIC-E1
title: "Chat Intelligence — Vault-Aware Agentic Chat"
status: active
created_at: 2026-02-22
owner: forge/orchestrator
---

# EPIC E1 — Chat Intelligence

## Vision

The NANITE chat agent should be the fastest, most natural way to work with vault data. Users must be able to ask questions, sort/filter results, synthesize notes, generate new content, and kick off multi-step analyses — all from a single conversational interface. The agent is primed to understand NANITE objects (items, notes, vaults, inbox, sections, taxonomy) and requires no explanation from the user about what those concepts mean.

Long-term, the agent must support scheduled automations ("weekly review every Sunday"), background classification of new inbox items, template-driven consistent output, and retrieval-augmented search over large vaults. All of this sits on top of the same in-process Anthropic API integration — no external CLI tools, no subprocess communication.

## Design Principles

1. **In-process over external** — The agent runs inside Wails alongside the data. No IPC, no subprocess, no external dependency beyond the Anthropic API.
2. **Deterministic first, AI second** — Use rules, sorts, and filters wherever possible. AI fills the gaps (semantic search, synthesis, generation).
3. **Tool use as the data interface** — Claude requests data explicitly via tools rather than receiving pre-baked context dumps.
4. **Templates for consistency** — Repeatable workflows (weekly review, vault audit) use saved templates so output is predictable and idempotent.
5. **Transparency** — Every tool call is logged. Cache hits are flagged. Users always know what the agent did.
6. **User controls mutations** — Propose → Approve flow for writes. Never execute without explicit confirmation.

## Sprints

### Sprint 1 — Foundation (current)
Core instrumentation and tool expansion. Highest leverage per effort.

| Task | Priority | Status |
|------|----------|--------|
| TASK-012: Tool result persistence + in-session cache | A | done |
| TASK-013: `notes_text` plain-text column (migration v5) | A | todo |
| TASK-014: Expanded tools: `get_item`, `list_taxonomy`, `get_vault_stats` | A | todo |

### Sprint 2 — Context & Quality
Make the agent feel like it knows the vault before the first question.

| Task | Priority | Status |
|------|----------|--------|
| TASK-015: Vault Profile injected into system prompt | A | todo |
| TASK-016: System prompt Nanite object glossary | B | todo |
| TASK-017: `create_item` tool with permission levels | B | todo |

### Sprint 3 — Templates
Consistent, deterministic output for repeatable workflows.

| Task | Priority | Status |
|------|----------|--------|
| TASK-018: Template storage (SQLite table + CRUD) | B | todo |
| TASK-019: Template tools (`list_templates`, `use_template`) | B | todo |
| TASK-020: Save-prompt UX when agent uses new pattern | B | todo |
| TASK-021: Template management UI in Settings | C | todo |

### Sprint 4 — Semantic Search / RAG
Enable "find related" queries that FTS5 cannot handle.

| Task | Priority | Status |
|------|----------|--------|
| TASK-022: Background embeddings generation pipeline | B | todo |
| TASK-023: `semantic_search` tool + hybrid FTS5+semantic re-rank | B | todo |

### Sprint 5 — Automation
Scheduled jobs, headless agent sessions, auto-classifier.

| Task | Priority | Status |
|------|----------|--------|
| TASK-024: Job scheduler + headless agent sessions | B | todo |
| TASK-025: Auto-classifier background worker | C | todo |

## ADRs

- [ADR-001](../adr/ADR-001-direct-api-vs-cli-headless.md) — Direct Anthropic API vs CLI headless tools
- [ADR-002](../adr/ADR-002-notes-plain-text-field.md) — notes_text plain-text field alongside HTML
- [ADR-003](../adr/ADR-003-tool-result-caching.md) — In-session tool result caching strategy
- [ADR-004](../adr/ADR-004-embeddings-storage.md) — Embeddings storage approach for semantic search
- [ADR-005](../adr/ADR-005-template-storage.md) — Template storage format

## Success Metrics

- "Show me last 10 items" → correct sorted results (fixed in M3 / TASK-011) ✓
- "Find all notes about X" → semantic results, not just keyword matches (Sprint 4)
- "Weekly review" → consistent structured output from template (Sprint 3)
- Tool calls logged and auditable in chat.db (Sprint 1 / TASK-012) ✓
- Cache hit rate > 30% for follow-up sorting/filtering questions (Sprint 1 / TASK-012) ✓
