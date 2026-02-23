---
id: BACKLOG-E1-S3-002
title: "Templates: list_templates + use_template tools in Bridge"
priority: B
epic: EPIC-E1
sprint: 3
created_at: 2026-02-22
---

## Description
Register `list_templates` and `use_template` tools in the Bridge. Claude can discover available templates and invoke them with parameter substitution.

## Tool Definitions

### `list_templates`
No input. Returns `[{slug, name, description, type, parameters}]`.
Claude uses this when a user asks for a "weekly review" or when it detects a repeatable workflow.

### `use_template`
Input: `{slug: string, params: {key: value}}`.
Returns the rendered prompt string with `{{param}}` substitutions applied.
Claude then executes the rendered prompt as its next instruction.

## Acceptance Criteria
- [ ] `BridgeStore` extended with `ListTemplates()` + `GetTemplate(slug)` methods
- [ ] `list_templates` tool: calls store, returns JSON array
- [ ] `use_template` tool: fetches template, applies Go `strings.ReplaceAll` for each `{{param}}`, returns rendered string
- [ ] Missing params: returned as-is (Claude fills them from context)
- [ ] System prompt: "When user asks for a report or recurring task, call list_templates first to find an appropriate template"
- [ ] Both tools added to `buildTools()` alongside search_vault

## Blocked by
BACKLOG-E1-S3-001 (needs template storage)
