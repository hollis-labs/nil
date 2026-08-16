package chat

// toolDef is the Anthropic tool definition format.
type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// buildTools returns all tool definitions available to the bridge.
func buildTools() []toolDef {
	return []toolDef{
		{
			Name:        "search_vault",
			Description: "Search the NIL vault for items (tasks and notes). Use this for any question about vault contents. You may call it multiple times with different parameters.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Full-text search terms. Omit or leave empty to match all items (use filters instead).",
					},
					"type": map[string]any{
						"type":        "string",
						"enum":        []string{"all", "todo", "note"},
						"description": "Item type. 'all' returns both tasks and notes (default when omitted). 'todo' = tasks only. 'note' = notes only.",
					},
					"statuses": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string", "enum": []string{"open", "completed", "archived", "overdue", "today"}},
						"description": "Status filters. Leave empty to include all statuses. 'open' = incomplete & not archived.",
					},
					"priorities": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Priority letter filters, e.g. [\"A\", \"B\"].",
					},
					"projects": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Filter by project names (without the + prefix).",
					},
					"contexts": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Filter by context names (without the @ prefix).",
					},
					"tags": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Filter by tag names (without the # prefix).",
					},
					"sort_by": map[string]any{
						"type":        "string",
						"enum":        []string{"created_at", "updated_at", "due_at", "title"},
						"description": "Sort field. Use 'created_at' + sort_dir='desc' for 'most recently created'.",
					},
					"sort_dir": map[string]any{
						"type":        "string",
						"enum":        []string{"asc", "desc"},
						"description": "Sort direction. 'desc' = newest/largest first.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"minimum":     1,
						"maximum":     50,
						"description": "Maximum results to return. Default 20.",
					},
				},
			},
		},
		{
			Name:        "get_item",
			Description: "Fetch a single vault item by its numeric ID, returning the complete record including full note content. Use this when a search_vault result has a truncated note and you need the full text for synthesis or quoting.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "integer",
						"description": "The numeric id of the item to retrieve.",
					},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "list_taxonomy",
			Description: "List all projects, contexts, and tags in the vault with their item counts. Call this when the user asks what projects/contexts/tags exist, or any 'what's in my vault' question. Prefer this over searching for taxonomy terms — it is faster and complete.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "create_item",
			Description: "Create a new item (todo or note) in the vault immediately, without requiring user approval. Only use this tool when DirectCreate is true in Capabilities. For other vaults, emit a JSON action block instead.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type": map[string]any{
						"type":        "string",
						"enum":        []string{"todo", "note"},
						"description": "Item type. Default: todo.",
					},
					"title": map[string]any{
						"type":        "string",
						"description": "Item title. Required.",
					},
					"priority": map[string]any{
						"type":        "string",
						"enum":        []string{"A", "B", "C", "D"},
						"description": "Priority letter A–D (optional).",
					},
					"section": map[string]any{
						"type":        "string",
						"enum":        []string{"now", "soon", "anytime"},
						"description": "Urgency section. Default: anytime.",
					},
					"projects": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Project names (without + prefix).",
					},
					"contexts": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Context names (without @ prefix).",
					},
					"tags": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Tag names (without # prefix).",
					},
					"notes": map[string]any{
						"type":        "string",
						"description": "Plain-text note content (optional).",
					},
				},
				"required": []string{"title"},
			},
		},
		{
			Name:        "get_vault_stats",
			Description: "Return aggregate counts for the vault: total items, open/completed/archived counts, overdue count, inbox count, by type (todo/note), and by section (now/soon/anytime). Use this for weekly reviews, progress summaries, or 'how many X do I have' questions.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "list_templates",
			Description: "List all available workflow templates. Returns [{slug, name, description, type, parameters}]. Call this when the user asks for a weekly review, audit, report, or any recurring workflow — then call use_template with the chosen slug.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "use_template",
			Description: "Render a workflow template by substituting {{param}} placeholders with the provided values. Returns the rendered prompt string. Execute the rendered prompt as your next set of instructions.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"slug": map[string]any{
						"type":        "string",
						"description": "The template slug from list_templates.",
					},
					"params": map[string]any{
						"type":                 "object",
						"description":          "Key-value pairs for {{param}} substitution. Keys match the template's parameters list. Missing keys are left as-is.",
						"additionalProperties": map[string]any{"type": "string"},
					},
				},
				"required": []string{"slug"},
			},
		},
	}
}
