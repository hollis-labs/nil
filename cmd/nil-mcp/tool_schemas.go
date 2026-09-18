package main

// Tool registration: schema-building helpers, the per-tool annotation-hint
// table, and registerTools(), which wires every nil_* tool into a go-mcp
// server.Server. No behavior lives here; the corresponding implementations
// live in tools.go, and the HTTP client they call through lives in client.go.

import (
	gomcp "github.com/hollis-labs/go-mcp/server"
)

// toolHints is the explicit, hand-reviewed MCP tool-annotation set for every
// tool this server registers. go-mcp requires ReadOnlyHint/DestructiveHint/
// IdempotentHint/OpenWorldHint to be always explicit, never inferred from a
// tool's name -- this table is the human decision the framework refuses to
// make for us. OpenWorldHint is false for every nil-mcp tool: each one
// operates on a NIL vault's own closed-domain item/inbox store via the local
// HTTP API, never an open-ended external system.
type toolHints struct {
	ReadOnly    bool
	Destructive bool
	Idempotent  bool
}

var toolAnnotations = map[string]toolHints{
	"nil_list_vaults":     {true, false, true},
	"nil_search":          {true, false, true},
	"nil_get_item":        {true, false, true},
	"nil_get_backrefs":    {true, false, true},
	"nil_list_item_ids":   {true, false, true},
	"nil_create_item":     {false, false, false},
	"nil_create_items":    {false, false, false},
	"nil_update_item":     {false, false, true},
	"nil_delete_item":     {false, true, false},
	"nil_toggle_complete": {false, false, true},
	"nil_archive":         {false, false, true},
	"nil_list_inbox":      {true, false, true},
	"nil_create_inbox":    {false, false, false},
	"nil_process_inbox":   {false, false, false},
	"nil_get_taxonomy":    {true, false, true},
}

// registerTool looks up name's annotation hints and registers it against s.
// Every registerTools call site below must go through this instead of
// s.RegisterTool directly, so a tool can never ship with silently-defaulted
// hints.
func registerTool(s *gomcp.Server, name, description string, inputSchema any, handler gomcp.ToolHandler) {
	hints, ok := toolAnnotations[name]
	if !ok {
		panic("nil-mcp: no annotation hints for tool " + name)
	}
	s.RegisterTool(gomcp.Tool{
		Name:            name,
		Description:     description,
		InputSchema:     inputSchema,
		Handler:         handler,
		ReadOnlyHint:    hints.ReadOnly,
		DestructiveHint: hints.Destructive,
		IdempotentHint:  hints.Idempotent,
		OpenWorldHint:   false,
	})
}

// ---------------------------------------------------------------------------
// JSON Schema property builders
// ---------------------------------------------------------------------------

func strProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func intProp(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func boolProp(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

func strEnumProp(desc string, vals ...string) map[string]any {
	return map[string]any{"type": "string", "description": desc, "enum": vals}
}

func strArrayProp(desc string) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": desc,
		"items":       map[string]any{"type": "string"},
	}
}

func objArrayProp(desc string) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": desc,
		"items":       map[string]any{"type": "object"},
	}
}

func vaultIDProp() map[string]any {
	return strProp("Vault ID. Omit to use the active vault.")
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

func registerTools(s *gomcp.Server, c *apiClient) {
	registerTool(s, "nil_list_vaults",
		"List all NIL vaults. Returns id, name, path, active flag.",
		gomcp.EmptyObjectSchema(),
		c.toolListVaults)

	registerTool(s, "nil_search",
		"Search todos and notes in a NIL vault using full-text search. For bulk/incremental sync (\"everything that changed recently\"), pass kind=\"all\" (or omit kind) with updated_since; this call is single-vault, so loop over nil_list_vaults for a cross-vault sync.",
		gomcp.ObjectSchema(map[string]any{
			"q":             strProp("Search query"),
			"kind":          strEnumProp("Item kind filter. Omit or use \"all\" to return every kind (default).", "todo", "note", "scratch", "all"),
			"tags":          strArrayProp("Filter by tags"),
			"contexts":      strArrayProp("Filter by contexts"),
			"projects":      strArrayProp("Filter by projects"),
			"page":          intProp("Page number (0-based)"),
			"page_size":     intProp("Results per page (default 20)"),
			"vault_id":      vaultIDProp(),
			"updated_since": strProp("RFC3339 timestamp (e.g. 2026-08-01T00:00:00Z). Only return items updated on or after this instant. Omit for no time filter."),
		}),
		c.toolSearch)

	registerTool(s, "nil_get_item",
		"Get a single item by its numeric ID.",
		gomcp.ObjectSchema(map[string]any{
			"id":       intProp("Item ID"),
			"vault_id": vaultIDProp(),
		}, "id"),
		c.toolGetItem)

	registerTool(s, "nil_get_backrefs",
		"List items that link to the given item via a wikilink (backlinks / \"what links here\"). Returns an empty list if the item exists but nothing links to it.",
		gomcp.ObjectSchema(map[string]any{
			"id":       intProp("Item ID to find backlinks for"),
			"vault_id": vaultIDProp(),
		}, "id"),
		c.toolGetBackrefs)

	registerTool(s, "nil_list_item_ids",
		"List every current item's id + updated_at in a vault (no title, no notes body, no taxonomy). For deletion detection: Nil has no soft-delete/tombstone concept, so a sync consumer diffs this full current-ID set against its own known-ID set on each sync cycle — any previously-seen ID missing here has been genuinely deleted. Deliberately has no updated_since filter (unlike nil_search): it always needs the full ID set, not an incremental slice, or existing IDs outside the window would look deleted. This call is single-vault, so loop over nil_list_vaults for a cross-vault check.",
		gomcp.ObjectSchema(map[string]any{
			"kind":     strEnumProp("Item kind filter. Omit or use \"all\" to return every kind (default).", "todo", "note", "scratch", "all"),
			"vault_id": vaultIDProp(),
		}),
		c.toolListItemIDs)

	registerTool(s, "nil_create_item",
		"Create a new item in a NIL vault (not the inbox). Notes body accepts markdown (notes_md), HTML (notes_html), or pre-built TipTap doc JSON (notes_doc); precedence is doc > md > html.",
		gomcp.ObjectSchema(map[string]any{
			"title":        strProp("Item title"),
			"kind":         strEnumProp("Item kind", "todo", "note", "scratch"),
			"notes_md":     strProp("Markdown body (converted server-side to TipTap doc)"),
			"notes_html":   strProp("HTML body (converted server-side to TipTap doc)"),
			"notes_doc":    strProp("Pre-built TipTap doc JSON (skips server conversion)"),
			"priority":     strEnumProp("Priority", "A", "B", "C"),
			"due_at":       strProp("Due date in RFC3339 format"),
			"section":      strEnumProp("Section", "now", "soon", "anytime"),
			"tags":         strArrayProp("Tags"),
			"contexts":     strArrayProp("Contexts"),
			"projects":     strArrayProp("Projects"),
			"vault_id":     vaultIDProp(),
			"external_ref": strProp("Optional idempotency key identifying this item's corresponding record on your side. Re-calling with the same external_ref (in the same vault) updates the existing item instead of creating a duplicate — safe to retry/re-push."),
		}, "title"),
		c.toolCreateItem)

	registerTool(s, "nil_create_items",
		"Create multiple items in a NIL vault in one call (batch create). All-or-nothing: if any item fails (e.g. an invalid kind), nothing in the batch is created and the error names which item (by index) and why — fix it and retry the whole batch. Each item supports the same fields as nil_create_item, including external_ref for idempotent re-push (an item whose external_ref matches an existing row updates it in place instead of duplicating it).",
		gomcp.ObjectSchema(map[string]any{
			"items":    objArrayProp("Items to create. Each entry accepts the same fields as nil_create_item (minus vault_id, which applies once to the whole call)."),
			"vault_id": vaultIDProp(),
		}, "items"),
		c.toolCreateItemsBatch)

	registerTool(s, "nil_update_item",
		"Update fields of an existing item. Only provided fields are changed. Notes body accepts markdown (notes_md), HTML (notes_html), or pre-built TipTap doc JSON (notes_doc).",
		gomcp.ObjectSchema(map[string]any{
			"id":           intProp("Item ID"),
			"title":        strProp("New title"),
			"kind":         strEnumProp("New kind", "todo", "note", "scratch"),
			"notes_md":     strProp("New markdown body (converted server-side)"),
			"notes_html":   strProp("New HTML body (converted server-side)"),
			"notes_doc":    strProp("New pre-built TipTap doc JSON"),
			"priority":     strEnumProp("Priority", "A", "B", "C"),
			"due_at":       strProp("Due date in RFC3339 format"),
			"section":      strEnumProp("Section", "now", "soon", "anytime"),
			"tags":         strArrayProp("Tags (replaces existing)"),
			"contexts":     strArrayProp("Contexts (replaces existing)"),
			"projects":     strArrayProp("Projects (replaces existing)"),
			"vault_id":     vaultIDProp(),
			"external_ref": strProp("Set/change/clear this item's external idempotency key."),
		}, "id"),
		c.toolUpdateItem)

	registerTool(s, "nil_delete_item",
		"Permanently delete an item from a vault.",
		gomcp.ObjectSchema(map[string]any{
			"id":       intProp("Item ID"),
			"vault_id": vaultIDProp(),
		}, "id"),
		c.toolDeleteItem)

	registerTool(s, "nil_toggle_complete",
		"Mark a todo as complete or incomplete.",
		gomcp.ObjectSchema(map[string]any{
			"id":        intProp("Item ID"),
			"completed": boolProp("true to mark complete, false to mark incomplete"),
			"vault_id":  vaultIDProp(),
		}, "id", "completed"),
		c.toolToggleComplete)

	registerTool(s, "nil_archive",
		"Archive or unarchive an item.",
		gomcp.ObjectSchema(map[string]any{
			"id":       intProp("Item ID"),
			"archived": boolProp("true to archive, false to unarchive"),
			"vault_id": vaultIDProp(),
		}, "id", "archived"),
		c.toolArchive)

	registerTool(s, "nil_list_inbox",
		"List items in the NIL inbox (fast-capture area awaiting triage).",
		gomcp.ObjectSchema(map[string]any{
			"q":         strProp("Optional search query"),
			"page":      intProp("Page number (0-based)"),
			"page_size": intProp("Results per page"),
		}),
		c.toolListInbox)

	registerTool(s, "nil_create_inbox",
		"Fast-capture a new item to the NIL inbox. No taxonomy required — great for quick ideas. Notes body accepts markdown (notes_md), HTML (notes_html), or pre-built TipTap doc JSON (notes_doc).",
		gomcp.ObjectSchema(map[string]any{
			"title":      strProp("Item title (can be blank)"),
			"notes_md":   strProp("Markdown body (converted server-side)"),
			"notes_html": strProp("HTML body (converted server-side)"),
			"notes_doc":  strProp("Pre-built TipTap doc JSON"),
			"kind":       strEnumProp("Item kind", "todo", "note", "scratch"),
			"priority":   strEnumProp("Priority", "A", "B", "C"),
			"due_at":     strProp("Due date in RFC3339 format"),
			"tags":       strArrayProp("Tags"),
			"contexts":   strArrayProp("Contexts"),
			"projects":   strArrayProp("Projects"),
		}),
		c.toolCreateInbox)

	registerTool(s, "nil_process_inbox",
		"Promote an inbox item to a regular vault item (removes inbox flag).",
		gomcp.ObjectSchema(map[string]any{
			"id": intProp("Inbox item ID"),
		}, "id"),
		c.toolProcessInbox)

	registerTool(s, "nil_get_taxonomy",
		"Get all projects, contexts, and tags used in a vault with item counts.",
		gomcp.ObjectSchema(map[string]any{
			"vault_id": vaultIDProp(),
		}),
		c.toolGetTaxonomy)
}
