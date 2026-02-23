---
id: ADR-002
title: "notes_text plain-text field alongside notes_md HTML"
status: accepted
date: 2026-02-22
deciders: [engineering]
supersedes: []
---

# ADR-002 — notes_text Plain-Text Field

## Context

TipTap stores notes as HTML in `notes_md`. This field is used for FTS5 full-text search and is included in AI tool results. HTML tags are noise for both use cases: FTS5 matches `<p>` and `<strong>` as tokens, and AI context windows waste tokens on markup. Additionally, agents that generate or modify notes should work with clean text, not HTML.

## Decision

Add a `notes_text TEXT` column to the `todos` table (migration v5). Keep `notes_md` unchanged for the TipTap editor. Maintain `notes_text` as the plain-text projection of `notes_md`, stripped of HTML tags.

## Implementation

1. **Migration v5**: `ALTER TABLE todos ADD COLUMN notes_text TEXT`.
2. **schema.sql**: add column to fresh-install schema.
3. **FTS5**: add `notes_text` to `todos_fts` virtual table (migration drops and recreates the FTS index, or adds a new content= column).
4. **CreateItem / UpdateItem**: strip HTML from `notes_md` and write to `notes_text`. Use Go `html.UnescapeString` + regex tag stripping (no CGo dependency). Preserve paragraph breaks as newlines.
5. **AI tool results**: `executeSearchVault` and `executeGetItem` use `notes_text` instead of `notes_md` — eliminates HTML tags from Claude's context window.
6. **Backfill**: one-time migration query updates existing rows: `UPDATE todos SET notes_text = ... WHERE notes_text IS NULL`. Run as part of migration v5.

## Rationale

- **FTS5 quality**: searching plain text produces better matches than searching HTML. Tags like `<p>`, `<strong>`, `<ul>` currently pollute the FTS5 token set.
- **Token efficiency**: AI tools currently truncate notes at 400 chars to avoid token blowout. With plain text, the same token budget covers significantly more content.
- **Agent write path**: future agents generating note content work in markdown/plain text natively. Conversion to TipTap HTML happens at the editor layer, not the agent layer.
- **No schema duplication risk**: `notes_md` is the source of truth for the editor. `notes_text` is a derived projection, always regenerated from `notes_md` on save.

## Consequences

- All `CreateItem` / `UpdateItem` paths must strip HTML. A helper `stripHTML(s string) string` function in the `store` package handles this.
- The FTS5 virtual table must be updated. This is a breaking migration (drop + recreate FTS table) — handled idempotently in the migration slice.
- `notes_text` may be slightly lossy for complex formatting (tables, code blocks). Acceptable — it's for search and AI context, not rendering.
- Priority: **Sprint 1 / TASK-013** — higher than originally planned because FTS5 quality and AI token efficiency both benefit immediately.
