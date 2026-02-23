---
id: BACKLOG-E1-S5-002
title: "Automation: auto-classifier background worker"
priority: C
epic: EPIC-E1
sprint: 5
created_at: 2026-02-22
---

## Description
A background goroutine that watches for new inbox items and proposes taxonomy (projects, contexts, tags, section) using deterministic rules first, AI classification as fallback. Users review proposals in the Inbox view.

## Classification Pipeline

### Stage 1: Deterministic Rules
Pattern matching on title/notes:
- Contains "call" / "phone" / "meeting" → `@phone` / `@meetings`
- Contains "email" / "send" → `@email`
- Contains URL → `@research` + tag `#link`
- Title starts with "read" / "watch" → `#media`
- etc. (user-configurable rules in settings)

### Stage 2: AI Classification (fallback)
If deterministic rules produce low-confidence results:
- Calls Claude with: item title + notes + existing taxonomy (from `list_taxonomy`)
- Returns: suggested projects, contexts, tags, section, priority
- Proposed as a structured `classify` action (not an `update`) — shown in Inbox view as "AI suggestion"

### User Review
Inbox view gains "AI Suggestions" sub-section. Each suggestion shows proposed taxonomy with Accept/Edit/Dismiss actions. Accept → `UpdateItem` with taxonomy applied.

## Acceptance Criteria
- [ ] `classifier.go` in `chat` package: `Classify(ctx, item, store) *ClassificationProposal`
- [ ] Stage 1: configurable rule set (JSON in settings)
- [ ] Stage 2: AI fallback (calls Bridge with a classification-specific system prompt)
- [ ] Background goroutine: polls inbox every N minutes (or triggered by `CreateItem` signal)
- [ ] `ClassificationProposal` stored (new table or extended `action_proposals`)
- [ ] Inbox view: shows AI suggestions with Accept/Dismiss
- [ ] Setting: enable/disable auto-classifier, configure stage 2 threshold

## Blocked by
BACKLOG-E1-S5-001 (headless session infrastructure)
TASK-20260222-014 (list_taxonomy tool needed for AI stage)
