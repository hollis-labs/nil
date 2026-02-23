---
id: BACKLOG-E1-S3-003
title: "Templates: save-prompt UX when agent uses new pattern"
priority: B
epic: EPIC-E1
sprint: 3
created_at: 2026-02-22
---

## Description
When Claude generates a response using a structure that looks like a reusable template (detected by Claude itself via a `suggest_template` action block), prompt the user in the chat UI: "Want to save this as a template?" with pre-filled name and description fields.

## Mechanism
Claude emits a special action block alongside its text response:
```json
{
  "suggest_template": {
    "name": "Weekly Review",
    "slug": "weekly-review",
    "description": "Summary of completed, incomplete, and upcoming items",
    "prompt": "Generate a weekly review for {{vault_name}} for {{week_start}} to {{week_end}}..."
  }
}
```
Backend detects this block, returns it in ChatResponse as `TemplateSuggestion`.
Frontend shows a dismissible card: "Save as template? [Name field] [Save] [Dismiss]"
On save: calls `Backend.SaveTemplate()`.

## Acceptance Criteria
- [ ] System prompt: instructs Claude to emit `suggest_template` block when it detects a novel reusable pattern (max once per session)
- [ ] `extractAction` or new `extractSuggestion` function detects the block
- [ ] `ChatResponse.TemplateSuggestion *TemplateSuggestion` field
- [ ] Frontend: `TemplateSuggestionCard` component in ChatMessage
- [ ] Save calls `Backend.SaveTemplate()`; success dismisses card and shows toast

## Blocked by
BACKLOG-E1-S3-001, BACKLOG-E1-S3-002
