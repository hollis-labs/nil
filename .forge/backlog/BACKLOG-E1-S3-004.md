---
id: BACKLOG-E1-S3-004
title: "Templates: management UI in Settings"
priority: C
epic: EPIC-E1
sprint: 3
created_at: 2026-02-22
---

## Description
Add a "Templates" tab to SettingsModal. Lists saved templates with name, type, slug. Users can view, edit the prompt, and delete templates. This makes templates a first-class UI concept.

## Acceptance Criteria
- [ ] 'templates' added to SettingsTab union
- [ ] Templates tab content: list of templates as cards (name, type, description)
- [ ] Expand card: show prompt text in readonly textarea; Edit button opens inline editor
- [ ] Delete with confirmation
- [ ] "Create template" button: opens form (name, description, type, prompt, parameters)
- [ ] Changes saved via `Backend.SaveTemplate()` / `Backend.DeleteTemplate()`
- [ ] Uses `var(--term-*)` CSS variables only

## Blocked by
BACKLOG-E1-S3-001
