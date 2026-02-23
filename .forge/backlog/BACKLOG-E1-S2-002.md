---
id: BACKLOG-E1-S2-002
title: "Chat: Nanite object glossary in system prompt"
priority: B
epic: EPIC-E1
sprint: 2
created_at: 2026-02-22
---

## Description
Add a concise Nanite-specific object glossary to the system prompt so Claude understands NANITE concepts without the user explaining them. Eliminates need for users to say "in NANITE, a section means..." in every session.

## Glossary Contents
- **Item** — a todo or note. Type field: `todo` | `note`.
- **Section** — priority bucket within todos: `now` (urgent), `soon` (this week/cycle), `anytime` (someday/maybe).
- **Priority** — single letter A–D (optional). A = highest. Distinct from Section.
- **Inbox** — capture queue. Items with blank title or flagged explicitly. Excluded from normal search by default.
- **Vault** — one SQLite database. A user may have multiple vaults (work, personal, etc.).
- **Taxonomy** — projects (+prefix), contexts (@prefix), tags (#prefix). All optional per item.
- **Threshold** — date before which an item is hidden from views (not yet active).
- **Recurrence** — todo.txt-style recurrence rule (e.g. `+1w` = every week).

## Acceptance Criteria
- [ ] Glossary section added to `systemPromptTemplate` between vault snapshot and responsibilities
- [ ] Kept under 300 tokens (condensed prose or bullet list)
- [ ] Includes actionable guidance: "When user says 'urgent', map to section=now or priority=A; when they say 'someday', map to section=anytime"

## Blocked by
Nothing — standalone system prompt change.
