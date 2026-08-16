package main

// toolList() is pure data — the MCP tool-schema catalog advertised to
// clients via tools/list. No behavior lives here; the corresponding
// implementations live in tools.go and the wire types they're built from
// (toolDef, inputSchema, schemaProp) live in rpc.go.

func toolList() []toolDef {
	strProp := func(desc string) schemaProp {
		return schemaProp{Type: "string", Description: desc}
	}
	intProp := func(desc string) schemaProp {
		return schemaProp{Type: "integer", Description: desc}
	}
	boolProp := func(desc string) schemaProp {
		return schemaProp{Type: "boolean", Description: desc}
	}
	strEnumProp := func(desc string, vals ...string) schemaProp {
		return schemaProp{Type: "string", Description: desc, Enum: vals}
	}
	strArrayProp := func(desc string) schemaProp {
		return schemaProp{
			Type:        "array",
			Description: desc,
			Items:       &schemaProp{Type: "string"},
		}
	}
	vaultIDProp := strProp("Vault ID. Omit to use the active vault.")

	return []toolDef{
		{
			Name:        "nil_list_vaults",
			Description: "List all NIL vaults. Returns id, name, path, active flag.",
			InputSchema: inputSchema{Type: "object"},
		},
		{
			Name:        "nil_search",
			Description: "Search todos and notes in a NIL vault using full-text search. For bulk/incremental sync (\"everything that changed recently\"), pass kind=\"all\" (or omit kind) with updated_since; this call is single-vault, so loop over nil_list_vaults for a cross-vault sync.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"q":             strProp("Search query"),
					"kind":          strEnumProp("Item kind filter. Omit or use \"all\" to return every kind (default).", "todo", "note", "scratch", "all"),
					"tags":          strArrayProp("Filter by tags"),
					"contexts":      strArrayProp("Filter by contexts"),
					"projects":      strArrayProp("Filter by projects"),
					"page":          intProp("Page number (0-based)"),
					"page_size":     intProp("Results per page (default 20)"),
					"vault_id":      vaultIDProp,
					"updated_since": strProp("RFC3339 timestamp (e.g. 2026-08-01T00:00:00Z). Only return items updated on or after this instant. Omit for no time filter."),
				},
			},
		},
		{
			Name:        "nil_get_item",
			Description: "Get a single item by its numeric ID.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"id":       intProp("Item ID"),
					"vault_id": vaultIDProp,
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "nil_get_backrefs",
			Description: "List items that link to the given item via a wikilink (backlinks / \"what links here\"). Returns an empty list if the item exists but nothing links to it.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"id":       intProp("Item ID to find backlinks for"),
					"vault_id": vaultIDProp,
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "nil_list_item_ids",
			Description: "List every current item's id + updated_at in a vault (no title, no notes body, no taxonomy). For deletion detection: Nil has no soft-delete/tombstone concept, so a sync consumer diffs this full current-ID set against its own known-ID set on each sync cycle — any previously-seen ID missing here has been genuinely deleted. Deliberately has no updated_since filter (unlike nil_search): it always needs the full ID set, not an incremental slice, or existing IDs outside the window would look deleted. This call is single-vault, so loop over nil_list_vaults for a cross-vault check.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"kind":     strEnumProp("Item kind filter. Omit or use \"all\" to return every kind (default).", "todo", "note", "scratch", "all"),
					"vault_id": vaultIDProp,
				},
			},
		},
		{
			Name:        "nil_create_item",
			Description: "Create a new item in a NIL vault (not the inbox). Notes body accepts markdown (notes_md), HTML (notes_html), or pre-built TipTap doc JSON (notes_doc); precedence is doc > md > html.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
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
					"vault_id":     vaultIDProp,
					"external_ref": strProp("Optional idempotency key identifying this item's corresponding record on your side. Re-calling with the same external_ref (in the same vault) updates the existing item instead of creating a duplicate — safe to retry/re-push."),
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "nil_create_items",
			Description: "Create multiple items in a NIL vault in one call (batch create). All-or-nothing: if any item fails (e.g. an invalid kind), nothing in the batch is created and the error names which item (by index) and why — fix it and retry the whole batch. Each item supports the same fields as nil_create_item, including external_ref for idempotent re-push (an item whose external_ref matches an existing row updates it in place instead of duplicating it).",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"items": {
						Type:        "array",
						Description: "Items to create. Each entry accepts the same fields as nil_create_item (minus vault_id, which applies once to the whole call).",
						Items: &schemaProp{
							Type: "object",
						},
					},
					"vault_id": vaultIDProp,
				},
				Required: []string{"items"},
			},
		},
		{
			Name:        "nil_update_item",
			Description: "Update fields of an existing item. Only provided fields are changed. Notes body accepts markdown (notes_md), HTML (notes_html), or pre-built TipTap doc JSON (notes_doc).",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
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
					"vault_id":     vaultIDProp,
					"external_ref": strProp("Set/change/clear this item's external idempotency key."),
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "nil_delete_item",
			Description: "Permanently delete an item from a vault.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"id":       intProp("Item ID"),
					"vault_id": vaultIDProp,
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "nil_toggle_complete",
			Description: "Mark a todo as complete or incomplete.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"id":        intProp("Item ID"),
					"completed": boolProp("true to mark complete, false to mark incomplete"),
					"vault_id":  vaultIDProp,
				},
				Required: []string{"id", "completed"},
			},
		},
		{
			Name:        "nil_archive",
			Description: "Archive or unarchive an item.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"id":       intProp("Item ID"),
					"archived": boolProp("true to archive, false to unarchive"),
					"vault_id": vaultIDProp,
				},
				Required: []string{"id", "archived"},
			},
		},
		{
			Name:        "nil_list_inbox",
			Description: "List items in the NIL inbox (fast-capture area awaiting triage).",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"q":         strProp("Optional search query"),
					"page":      intProp("Page number (0-based)"),
					"page_size": intProp("Results per page"),
				},
			},
		},
		{
			Name:        "nil_create_inbox",
			Description: "Fast-capture a new item to the NIL inbox. No taxonomy required — great for quick ideas. Notes body accepts markdown (notes_md), HTML (notes_html), or pre-built TipTap doc JSON (notes_doc).",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
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
				},
			},
		},
		{
			Name:        "nil_process_inbox",
			Description: "Promote an inbox item to a regular vault item (removes inbox flag).",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"id": intProp("Inbox item ID"),
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "nil_get_taxonomy",
			Description: "Get all projects, contexts, and tags used in a vault with item counts.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"vault_id": vaultIDProp,
				},
			},
		},
	}
}
