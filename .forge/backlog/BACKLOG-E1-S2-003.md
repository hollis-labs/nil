---
id: BACKLOG-E1-S2-003
title: "Chat: create_item tool with tiered permissions"
priority: B
epic: EPIC-E1
sprint: 2
created_at: 2026-02-22
---

## Description
Add a `create_item` tool that allows Claude to create items directly when the vault capability allows it, bypassing the Propose→Approve flow for low-friction creates. Tiered permission model: proposal flow (safe default) vs direct creation (user opt-in per vault).

## Permission Tiers
- **Tier 0** (current default): all mutations go through Propose→Approve
- **Tier 1** (new): creates allowed directly; updates/deletes still require approval
- **Tier 2** (future): all mutations allowed directly (power users only)

VaultCap gains a `DirectCreate bool` field in config.

## Tool Definition
`create_item` input:
```json
{
  "type": "todo|note",
  "title": "...",
  "priority": "A|B|C",
  "section": "now|soon|anytime",
  "projects": [],
  "contexts": [],
  "tags": [],
  "notes": ""
}
```

If DirectCreate=true: calls `vaultStore.CreateItem()` immediately, returns new item ID.
If DirectCreate=false: falls back to existing JSON action block proposal mechanism.

## Acceptance Criteria
- [ ] `VaultCap.DirectCreate bool` in `config/config.go` + Settings UI toggle
- [ ] `create_item` tool registered in bridge
- [ ] Permission check before execution
- [ ] On success: returns `{id, title, created_at}`
- [ ] System prompt: "Use create_item tool for creates when DirectCreate is enabled; otherwise emit action block"

## Blocked by
TASK-20260222-014
