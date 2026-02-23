---
id: ADR-005
title: "Template storage format for agentic workflows"
status: proposed
date: 2026-02-22
deciders: [engineering, product]
---

# ADR-005 — Template Storage Format

## Context

Templates define repeatable agentic workflows: weekly reviews, vault audits, content generation patterns. They need to be persisted, listed by the agent via a `list_templates` tool, invoked via `use_template`, and created when the agent detects a new reusable pattern.

## Options Considered

### Option A: JSON file (`~/.config/nanite/templates.json`)
Simple, human-editable, no migration needed.
**Weakness**: No per-template metadata queries. No atomic updates. Hard to surface in UI.

### Option B: SQLite table in vault DB
Fits in the existing schema. Queryable. Transactional.
**Weakness**: Templates are vault-agnostic (a "weekly review" template applies to any vault). Putting them in a vault DB means duplicating across vaults.

### Option C: SQLite table in config DB (separate `templates.db` or in `chat.db`)
Vault-agnostic. Queryable. Templates shared across all vaults.
**Weakness**: Another DB file to manage.

### Option D: SQLite table in `chat.db`
Templates are primarily a chat feature. They live alongside chat sessions. One DB, clear ownership.

**Selected.**

## Decision: Option D — SQLite table in `chat.db`

Add a `templates` table to `chat.db`.

## Schema

```sql
CREATE TABLE IF NOT EXISTS templates (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  slug         TEXT NOT NULL UNIQUE,       -- machine-readable ID for tool use
  name         TEXT NOT NULL,              -- human-readable display name
  description  TEXT NOT NULL DEFAULT '',   -- shown in list_templates tool result
  type         TEXT NOT NULL DEFAULT 'generation',  -- generation | query | analysis
  prompt       TEXT NOT NULL,              -- the template body (may include {{params}})
  parameters   TEXT NOT NULL DEFAULT '[]', -- JSON array of param names
  output_format TEXT NOT NULL DEFAULT 'markdown',
  created_at   TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);
```

## Template Parameter Syntax

Use `{{param_name}}` for substitution. The agent fills parameters before executing. Example:

```
Generate a weekly review for {{vault_name}} for the week of {{week_start}} to {{week_end}}.
Include: what was completed, what was not completed, suggested focus for next week, and any opportunities to explore.
Use get_vault_stats and search_vault tools to gather data before writing the review.
```

## Save-Prompt UX

When the agent generates content using a structure it hasn't seen before (detected heuristically by the agent itself — it emits a `suggest_template` action block), the frontend prompts: "Save this as a template?" with pre-filled name/description fields. User confirms → `Backend.SaveTemplate()`.

## `list_templates` Tool

Returns all templates as a JSON array: `[{slug, name, description, type, parameters}]`. Agent selects appropriate template, substitutes parameters, executes.

## `use_template` Tool

Input: `{slug, params: {key: value}}`. Returns the rendered prompt string. Agent then executes it within the same turn.

## Consequences

- Templates table added to `chat.db` schema (Sprint 3 migration)
- New Wails-bound methods: `ListTemplates`, `GetTemplate`, `SaveTemplate`, `DeleteTemplate`
- Bridge gains two new tools: `list_templates`, `use_template`
- Settings UI: template management tab (list, edit, delete) — Sprint 3 final task
- Scope: templates are global (not vault-scoped). If vault-specific templates become needed, add a `vault_id` column (nullable = global).
