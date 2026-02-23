---
id: BACKLOG-E1-S2-001
title: "Chat: Vault Profile injected into system prompt"
priority: A
epic: EPIC-E1
sprint: 2
created_at: 2026-02-22
---

## Description
Compute a concise "Vault Profile" on session open and inject it into the system prompt. This gives Claude immediate context about the vault's structure and user's working style before any tool calls.

## Profile Contents (computed, not stored)
- Active project count and top 5 project names
- Common contexts (top 5 by usage)
- Item counts: open todos, notes, overdue, inbox
- Recent activity: items completed this week, items created this week
- Working style signals: avg priority distribution, note-heaviness ratio

## Profile Format (injected into system prompt after capabilities line)
```
Vault snapshot (2026-02-22):
- 47 open todos · 23 notes · 3 overdue · 5 inbox
- Projects: +nanite, +client-work, +personal, +research
- Contexts: @dev, @meetings, @email
- This week: 8 completed, 12 created
```

## Acceptance Criteria
- [ ] `ComputeVaultProfile(ctx, store) string` function in a new `chat/profile.go`
- [ ] Profile injected between the capabilities line and the "## Your responsibilities" section in `buildSystemPrompt`
- [ ] Profile computed once per `SendChatMessage` call (cheap: uses `GetStats` from TASK-014)
- [ ] Graceful fallback: if store is nil or query fails, skip profile silently
- [ ] Profile length capped at ~200 tokens

## Blocked by
TASK-20260222-014 (needs GetStats)
