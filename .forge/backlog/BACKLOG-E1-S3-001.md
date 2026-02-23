---
id: BACKLOG-E1-S3-001
title: "Templates: storage (chat.db table + CRUD methods)"
priority: B
epic: EPIC-E1
sprint: 3
created_at: 2026-02-22
---

## Description
Add `templates` table to `chat.db` and implement CRUD store methods and Wails-bound backend methods. Foundation for all template features.

See ADR-005 for schema and design decisions.

## Schema
```sql
CREATE TABLE IF NOT EXISTS templates (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  slug          TEXT NOT NULL UNIQUE,
  name          TEXT NOT NULL,
  description   TEXT NOT NULL DEFAULT '',
  type          TEXT NOT NULL DEFAULT 'generation',
  prompt        TEXT NOT NULL,
  parameters    TEXT NOT NULL DEFAULT '[]',
  output_format TEXT NOT NULL DEFAULT 'markdown',
  created_at    TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
```

## Acceptance Criteria
- [ ] Schema added to `chatSchema` constant in `chat/store.go`
- [ ] `ChatStore.CreateTemplate()`, `GetTemplate(slug)`, `ListTemplates()`, `UpdateTemplate()`, `DeleteTemplate()` methods
- [ ] Wails-bound: `Backend.ListTemplates()`, `Backend.SaveTemplate()`, `Backend.DeleteTemplate()`
- [ ] Wails TypeScript bindings regenerated
- [ ] Seed: 2 built-in templates on first run: "weekly-review" and "vault-audit"

## Blocked by
Nothing — schema extension to existing chat.db
