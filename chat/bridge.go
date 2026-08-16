package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hollis-labs/nil/ingest"
	"github.com/hollis-labs/nil/service/items"
	"github.com/hollis-labs/nil/store"
)

// chatSvc is the package-level items service used by tool execution handlers.
// Stateless; safe to share.
var chatSvc = items.New()

const claudeAPIURL = "https://api.anthropic.com/v1/messages"
const claudeAPIVersion = "2023-06-01"
const defaultModel = "claude-haiku-4-5-20251001"
const defaultMaxTokens = 2048
const maxToolIterations = 5

// systemPromptTemplate is injected per-request with vault/session context substituted.
const systemPromptTemplate = `You are a NIL vault assistant. NIL is a personal task and note management app.
Today's date: {date}
Vault: {vault_name} | Mode: {mode}
Capabilities — Read: true | Write: {write_cap} | Delete: {delete_cap} | DirectCreate: {direct_create_cap}
{vault_profile}
## NIL Concepts
- **Item** — a todo (` + "`type=todo`" + `) or note (` + "`type=note`" + `).
- **Section** — urgency bucket for todos: ` + "`now`" + ` (urgent/active), ` + "`soon`" + ` (this week), ` + "`anytime`" + ` (someday/maybe). Map "urgent" → now, "someday" → anytime.
- **Priority** — single letter A–D (optional). A = highest. Independent of section.
- **Inbox** — capture queue; excluded from normal search. Items with blank titles auto-route here.
- **Vault** — one SQLite database; a user may have multiple (work, personal, etc.).
- **Taxonomy** — projects (` + "`+prefix`" + `), contexts (` + "`@prefix`" + `), tags (` + "`#prefix`" + `). All optional per item.
- **Threshold** — date before which an item is hidden from views (not yet actionable).
- **Recurrence** — todo.txt-style rule, e.g. ` + "`+1w`" + ` = weekly.

## Your responsibilities

For SEARCH or READ queries:
- Use the search_vault tool to retrieve items from the vault. You may call it multiple times with different parameters.
- Summarise the results clearly. If a search returns nothing, try alternate terms or relax filters.
- When asked for "last N items", use sort_by="created_at", sort_dir="desc", limit=N.
- When asked about notes specifically, use type="note". For tasks/todos, use type="todo". Otherwise use type="all".
- When asked "what's in my vault", "what projects/contexts/tags exist", or any taxonomy question, call list_taxonomy — it is faster and more accurate than searching.
- When a search result contains a truncated note and you need the full content, call get_item with the item's id.
- For "how many items", weekly reviews, or progress summaries, call get_vault_stats.
- For recurring workflows, reports, reviews, or audits, call list_templates first to discover available templates, then call use_template with the appropriate parameters to render it before executing.

For MUTATING requests (create / update / delete):
- For CREATE requests: if DirectCreate is true in Capabilities, call the create_item tool to create items immediately. Otherwise, emit a JSON action block for the user to review.
- For UPDATE and DELETE requests: always emit a JSON action block — never execute directly.
- Emit the action as a fenced JSON block at the END of your response, like this:

` + "```" + `json
{
  "action": {
    "type": "create",
    "item_type": "todo",
    "vault_id": "{vault_id}",
    "payload": {
      "title": "...",
      "priority": "A",
      "section": "now",
      "projects": [],
      "contexts": [],
      "tags": []
    }
  }
}
` + "```" + `

For updates, include a "diff" field: {"field_name": ["old_value", "new_value"]}.
For deletes, include "payload": {"id": <item_id>, "title": "<title>"}.
For updates, also include the full updated item in "payload".

Always address the user's intent first in plain text, then emit the action block.
If the vault does not have write or delete capability, explain that and do not emit an action.`

// VaultCaps describes what chat operations are permitted on a vault.
type VaultCaps struct {
	Read         bool
	Write        bool
	Delete       bool
	DirectCreate bool // when true, create_item tool executes directly without Propose→Approve
}

// BridgeRequest carries everything needed for one LLM turn (potentially multiple API calls).
type BridgeRequest struct {
	APIKey      string
	Model       string // optional; falls back to defaultModel
	VaultID     string
	VaultName   string
	Caps        VaultCaps
	DryRun      bool
	History     []ChatMessage // prior turns (for context window)
	UserMessage string
	Store       BridgeStore   // for tool execution; nil disables tools
	Templates   TemplateStore // for list_templates / use_template; nil disables those tools
	ToolCache   *ToolCache    // optional in-session result cache; nil disables caching
}

// Bridge calls the Claude API and parses the response.
// It carries no per-session state; callers manage persistence via ChatStore.
type Bridge struct{}

// --- Internal API types ---

// apiContent is one block within a message's content array.
type apiContent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`          // tool_use
	Name      string          `json:"name,omitempty"`        // tool_use
	Input     json.RawMessage `json:"input,omitempty"`       // tool_use
	ToolUseID string          `json:"tool_use_id,omitempty"` // tool_result
	Content   string          `json:"content,omitempty"`     // tool_result
	IsError   bool            `json:"is_error,omitempty"`    // tool_result
}

// apiMsg is one message in the Anthropic messages array.
// Content is either a string (for simple text) or []apiContent (for structured turns).
type apiMsg struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// apiResponse is the parsed Anthropic response envelope.
type apiResponse struct {
	Content    []apiContent `json:"content"`
	StopReason string       `json:"stop_reason"`
}

// toolDef is the Anthropic tool definition format.
type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// vaultSearchInput mirrors the search_vault tool's input schema.
type vaultSearchInput struct {
	Query      string   `json:"query"`
	Type       string   `json:"type"`
	Statuses   []string `json:"statuses"`
	Priorities []string `json:"priorities"`
	Projects   []string `json:"projects"`
	Contexts   []string `json:"contexts"`
	Tags       []string `json:"tags"`
	SortBy     string   `json:"sort_by"`
	SortDir    string   `json:"sort_dir"`
	Limit      int      `json:"limit"`
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

// Send executes an agentic tool-use loop with the Claude API.
// Claude may call search_vault multiple times before producing its final text response.
// If the assistant emits a JSON action block, ChatResponse.Proposal is populated.
func (b *Bridge) Send(ctx context.Context, req BridgeRequest) (*ChatResponse, error) {
	if req.APIKey == "" {
		return &ChatResponse{
			Message: ChatMessage{Role: "assistant", Content: ""},
			Error:   "No API key configured — add one in Settings → Chat",
		}, nil
	}

	model := req.Model
	if model == "" {
		model = defaultModel
	}

	vaultProfile := ComputeVaultProfile(ctx, req.Store)
	systemPrompt := buildSystemPrompt(req, vaultProfile)
	tools := buildTools()

	// Build initial message list from session history.
	var msgs []apiMsg
	start := 0
	if len(req.History) > 20 {
		start = len(req.History) - 20
	}
	for _, m := range req.History[start:] {
		if m.Role == "system" {
			continue
		}
		msgs = append(msgs, apiMsg{Role: m.Role, Content: m.Content})
	}
	msgs = append(msgs, apiMsg{Role: "user", Content: req.UserMessage})

	// Agentic tool-use loop: up to maxToolIterations API calls per user message.
	var allToolCalls []ToolCallRecord
	for iter := 0; iter < maxToolIterations; iter++ {
		apiRsp, err := b.callAPI(ctx, req.APIKey, model, systemPrompt, msgs, tools)
		if err != nil {
			return &ChatResponse{
				Message:   ChatMessage{Role: "assistant", Content: ""},
				ToolCalls: allToolCalls,
				Error:     err.Error(),
			}, nil
		}

		switch apiRsp.StopReason {
		case "end_turn":
			text := extractTextFromContent(apiRsp.Content)
			proposal, cleanText := extractAction(text, req.VaultID)
			return &ChatResponse{
				Message:   ChatMessage{Role: "assistant", Content: cleanText},
				Proposal:  proposal,
				ToolCalls: allToolCalls,
			}, nil

		case "tool_use":
			// Append the assistant's full turn (may include text + tool_use blocks).
			msgs = append(msgs, apiMsg{Role: "assistant", Content: apiRsp.Content})

			// Execute each tool call, cache results, collect records.
			var results []apiContent
			for _, block := range apiRsp.Content {
				if block.Type != "tool_use" {
					continue
				}
				result, cacheHit := b.executeTool(ctx, block, req.Store, req.Caps, req.ToolCache, req.Templates)
				allToolCalls = append(allToolCalls, ToolCallRecord{
					ToolName:   block.Name,
					InputJSON:  string(block.Input),
					ResultJSON: result,
					CacheHit:   cacheHit,
				})
				results = append(results, apiContent{
					Type:      "tool_result",
					ToolUseID: block.ID,
					Content:   result,
				})
			}
			msgs = append(msgs, apiMsg{Role: "user", Content: results})

		default:
			// max_tokens or unknown — extract whatever text is present.
			text := extractTextFromContent(apiRsp.Content)
			proposal, cleanText := extractAction(text, req.VaultID)
			return &ChatResponse{
				Message:   ChatMessage{Role: "assistant", Content: cleanText},
				Proposal:  proposal,
				ToolCalls: allToolCalls,
			}, nil
		}
	}

	return &ChatResponse{
		Message:   ChatMessage{Role: "assistant", Content: "I reached the maximum search steps without a final answer. Please try rephrasing your question."},
		ToolCalls: allToolCalls,
	}, nil
}

// callAPI makes one HTTP POST to the Anthropic messages endpoint.
func (b *Bridge) callAPI(ctx context.Context, apiKey, model, system string, msgs []apiMsg, tools []toolDef) (*apiResponse, error) {
	payload := map[string]any{
		"model":      model,
		"max_tokens": defaultMaxTokens,
		"system":     system,
		"tools":      tools,
		"messages":   msgs,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("chat/bridge: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("chat/bridge: build http request: %w", err)
	}
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", claudeAPIVersion)
	httpReq.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("chat/bridge: api call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chat/bridge: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var raw struct {
		Content    []apiContent `json:"content"`
		StopReason string       `json:"stop_reason"`
	}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("chat/bridge: parse api response: %w", err)
	}

	return &apiResponse{Content: raw.Content, StopReason: raw.StopReason}, nil
}

// executeTool dispatches a tool_use block to the appropriate implementation.
// Returns (result, cacheHit). Cache is checked before execution and populated after.
func (b *Bridge) executeTool(ctx context.Context, block apiContent, st BridgeStore, caps VaultCaps, cache *ToolCache, tmpl TemplateStore) (string, bool) {
	// Template tools don't require the vault store.
	if block.Name == "list_templates" || block.Name == "use_template" {
		if cache != nil {
			if cached, ok := cache.Get(block.Name, block.Input); ok {
				return cached, true
			}
		}
		var result string
		switch block.Name {
		case "list_templates":
			result = b.executeListTemplates(ctx, tmpl)
		case "use_template":
			result = b.executeUseTemplate(ctx, block.Input, tmpl)
		}
		if cache != nil {
			cache.Set(block.Name, block.Input, result)
		}
		return result, false
	}

	if st == nil {
		return `{"error": "vault not available"}`, false
	}
	// Cache check.
	if cache != nil {
		if cached, ok := cache.Get(block.Name, block.Input); ok {
			return cached, true
		}
	}
	var result string
	switch block.Name {
	case "search_vault":
		result = b.executeSearchVault(ctx, block.Input, st)
	case "get_item":
		result = b.executeGetItem(ctx, block.Input, st)
	case "list_taxonomy":
		result = b.executeListTaxonomy(ctx, st)
	case "get_vault_stats":
		result = b.executeGetVaultStats(ctx, st)
	case "create_item":
		result = b.executeCreateItem(ctx, block.Input, st, caps)
	default:
		result = fmt.Sprintf(`{"error": "unknown tool %q"}`, block.Name)
	}
	// Populate cache.
	if cache != nil {
		cache.Set(block.Name, block.Input, result)
	}
	return result, false
}

// executeSearchVault runs a structured search against the vault store.
func (b *Bridge) executeSearchVault(ctx context.Context, inputJSON json.RawMessage, st BridgeStore) string {
	var input vaultSearchInput
	if err := json.Unmarshal(inputJSON, &input); err != nil {
		return `{"error": "invalid tool input"}`
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}

	req := store.SearchRequest{
		Query:      input.Query,
		Kind:       input.Type,
		Statuses:   input.Statuses,
		Priorities: input.Priorities,
		Projects:   input.Projects,
		Contexts:   input.Contexts,
		Tags:       input.Tags,
		SortBy:     input.SortBy,
		SortDir:    input.SortDir,
		PageSize:   limit,
	}

	var results []store.Item
	var err error
	if realStore, ok := st.(*store.Store); ok {
		results, err = chatSvc.Search(ctx, realStore, req)
	} else {
		// Fallback for non-concrete BridgeStore implementations (none today,
		// but keep the interface honoring path for testability).
		results, err = st.Search(ctx, req)
	}
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	if len(results) == 0 {
		return `{"results": [], "count": 0, "note": "No items matched the search criteria."}`
	}

	// Build a slim projection to keep token count manageable.
	type slimItem struct {
		ID        int64    `json:"id"`
		Type      string   `json:"type"`
		Title     string   `json:"title"`
		Priority  string   `json:"priority,omitempty"`
		Section   string   `json:"section,omitempty"`
		Completed bool     `json:"completed,omitempty"`
		Archived  bool     `json:"archived,omitempty"`
		DueAt     string   `json:"due_at,omitempty"`
		Projects  []string `json:"projects,omitempty"`
		Contexts  []string `json:"contexts,omitempty"`
		Tags      []string `json:"tags,omitempty"`
		CreatedAt string   `json:"created_at"`
		UpdatedAt string   `json:"updated_at"`
		Notes     string   `json:"notes,omitempty"`
	}

	slim := make([]slimItem, 0, len(results))
	for _, r := range results {
		priority := ""
		if r.Priority != nil {
			priority = *r.Priority
		}
		dueAt := ""
		if r.DueAt != nil {
			dueAt = *r.DueAt
		}
		notes, _ := ingest.DocToPlainText(r.NotesDoc)
		if len(notes) > 400 {
			notes = notes[:400] + "…"
		}
		slim = append(slim, slimItem{
			ID:        r.ID,
			Type:      r.Kind,
			Title:     r.Title,
			Priority:  priority,
			Section:   r.Section,
			Completed: r.Completed,
			Archived:  r.Archived,
			DueAt:     dueAt,
			Projects:  r.Projects,
			Contexts:  r.Contexts,
			Tags:      r.Tags,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
			Notes:     notes,
		})
	}

	out, err := json.Marshal(map[string]any{"results": slim, "count": len(slim)})
	if err != nil {
		return `{"error": "marshal error"}`
	}
	return string(out)
}

// executeGetItem fetches a single item by ID and returns full content (no truncation).
func (b *Bridge) executeGetItem(ctx context.Context, inputJSON json.RawMessage, st BridgeStore) string {
	var input struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(inputJSON, &input); err != nil || input.ID == 0 {
		return `{"error": "get_item requires a valid integer id"}`
	}
	item, err := st.GetItem(ctx, input.ID)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	priority := ""
	if item.Priority != nil {
		priority = *item.Priority
	}
	dueAt := ""
	if item.DueAt != nil {
		dueAt = *item.DueAt
	}
	notesText, _ := ingest.DocToPlainText(item.NotesDoc)
	out, _ := json.Marshal(map[string]any{
		"id":         item.ID,
		"type":       item.Kind,
		"title":      item.Title,
		"priority":   priority,
		"section":    item.Section,
		"completed":  item.Completed,
		"archived":   item.Archived,
		"due_at":     dueAt,
		"projects":   item.Projects,
		"contexts":   item.Contexts,
		"tags":       item.Tags,
		"notes":      notesText,
		"created_at": item.CreatedAt,
		"updated_at": item.UpdatedAt,
	})
	return string(out)
}

// executeListTaxonomy returns all projects, contexts, and tags with item counts.
func (b *Bridge) executeListTaxonomy(ctx context.Context, st BridgeStore) string {
	projects, contexts, tags, err := st.GetTaxonomyWithCounts(ctx)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	if projects == nil {
		projects = []store.TaxonomyItem{}
	}
	if contexts == nil {
		contexts = []store.TaxonomyItem{}
	}
	if tags == nil {
		tags = []store.TaxonomyItem{}
	}
	out, _ := json.Marshal(map[string]any{
		"projects": projects,
		"contexts": contexts,
		"tags":     tags,
	})
	return string(out)
}

// executeGetVaultStats returns aggregate vault counts.
func (b *Bridge) executeGetVaultStats(ctx context.Context, st BridgeStore) string {
	stats, err := st.GetStats(ctx)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	out, _ := json.Marshal(stats)
	return string(out)
}

// executeCreateItem creates an item directly in the vault when DirectCreate is enabled,
// or returns an error directing Claude to use the action block flow instead.
func (b *Bridge) executeCreateItem(ctx context.Context, inputJSON json.RawMessage, st BridgeStore, caps VaultCaps) string {
	if !caps.DirectCreate {
		return `{"error": "DirectCreate is not enabled for this vault. Emit a JSON action block for the user to review and approve instead."}`
	}
	var input struct {
		Type     string   `json:"type"`
		Title    string   `json:"title"`
		Priority string   `json:"priority"`
		Section  string   `json:"section"`
		Projects []string `json:"projects"`
		Contexts []string `json:"contexts"`
		Tags     []string `json:"tags"`
		Notes    string   `json:"notes"`
	}
	if err := json.Unmarshal(inputJSON, &input); err != nil {
		return `{"error": "invalid tool input"}`
	}
	if input.Title == "" {
		return `{"error": "title is required"}`
	}
	realStore, ok := st.(*store.Store)
	if !ok {
		return `{"error": "internal: create requires a concrete *store.Store"}`
	}
	var pri *string
	if input.Priority != "" {
		p := input.Priority
		pri = &p
	}
	created, err := chatSvc.Create(ctx, realStore, items.CreateInput{
		Title:    input.Title,
		Kind:     input.Type,
		Section:  input.Section,
		Priority: pri,
		Projects: input.Projects,
		Contexts: input.Contexts,
		Tags:     input.Tags,
		NotesMD:  input.Notes,
	})
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	out, _ := json.Marshal(map[string]any{
		"id":         created.ID,
		"title":      created.Title,
		"type":       created.Kind,
		"section":    created.Section,
		"created_at": created.CreatedAt,
	})
	return string(out)
}

// executeListTemplates returns a slim listing of all templates (no prompt body to save tokens).
func (b *Bridge) executeListTemplates(ctx context.Context, tmpl TemplateStore) string {
	if tmpl == nil {
		return `{"error": "template store not available"}`
	}
	templates, err := tmpl.ListTemplates(ctx)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	type slimTemplate struct {
		Slug        string `json:"slug"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
		Parameters  string `json:"parameters"`
	}
	slim := make([]slimTemplate, 0, len(templates))
	for _, t := range templates {
		slim = append(slim, slimTemplate{
			Slug:        t.Slug,
			Name:        t.Name,
			Description: t.Description,
			Type:        t.Type,
			Parameters:  t.Parameters,
		})
	}
	out, _ := json.Marshal(map[string]any{"templates": slim, "count": len(slim)})
	return string(out)
}

// executeUseTemplate renders a template by substituting {{param}} placeholders.
func (b *Bridge) executeUseTemplate(ctx context.Context, inputJSON json.RawMessage, tmpl TemplateStore) string {
	if tmpl == nil {
		return `{"error": "template store not available"}`
	}
	var input struct {
		Slug   string            `json:"slug"`
		Params map[string]string `json:"params"`
	}
	if err := json.Unmarshal(inputJSON, &input); err != nil || input.Slug == "" {
		return `{"error": "use_template requires a slug"}`
	}
	t, err := tmpl.GetTemplate(ctx, input.Slug)
	if err != nil {
		return fmt.Sprintf(`{"error": "template %q not found"}`, input.Slug)
	}
	rendered := t.Prompt
	for k, v := range input.Params {
		rendered = strings.ReplaceAll(rendered, "{{"+k+"}}", v)
	}
	out, _ := json.Marshal(map[string]any{
		"slug":            t.Slug,
		"name":            t.Name,
		"rendered_prompt": rendered,
	})
	return string(out)
}

// extractTextFromContent joins all "text" content blocks from an API response.
func extractTextFromContent(content []apiContent) string {
	var parts []string
	for _, c := range content {
		if c.Type == "text" && c.Text != "" {
			parts = append(parts, c.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// buildSystemPrompt substitutes placeholders into the system prompt template.
// vaultProfile is the pre-computed snapshot string from ComputeVaultProfile;
// pass "" to omit the profile section cleanly.
func buildSystemPrompt(req BridgeRequest, vaultProfile string) string {
	mode := "live"
	if req.DryRun {
		mode = "DRY RUN — proposals only, no execution"
	}
	writeCap, deleteCap, directCreateCap := "false", "false", "false"
	if req.Caps.Write {
		writeCap = "true"
	}
	if req.Caps.Delete {
		deleteCap = "true"
	}
	if req.Caps.DirectCreate {
		directCreateCap = "true"
	}
	p := systemPromptTemplate
	p = strings.ReplaceAll(p, "{vault_name}", req.VaultName)
	p = strings.ReplaceAll(p, "{vault_id}", req.VaultID)
	p = strings.ReplaceAll(p, "{mode}", mode)
	p = strings.ReplaceAll(p, "{write_cap}", writeCap)
	p = strings.ReplaceAll(p, "{delete_cap}", deleteCap)
	p = strings.ReplaceAll(p, "{direct_create_cap}", directCreateCap)
	p = strings.ReplaceAll(p, "{date}", time.Now().Format("2006-01-02"))
	if vaultProfile != "" {
		p = strings.ReplaceAll(p, "{vault_profile}", "\n"+vaultProfile)
	} else {
		p = strings.ReplaceAll(p, "\n{vault_profile}", "")
	}
	return p
}

// jsonBlockRe matches a fenced ```json ... ``` block (non-greedy, dot-matches-newline).
var jsonBlockRe = regexp.MustCompile("(?s)```json\\s*(\\{.*?\\})\\s*```")

// actionEnvelope mirrors the JSON structure the LLM emits for mutating actions.
type actionEnvelope struct {
	Action struct {
		Type     string         `json:"type"`
		ItemType string         `json:"item_type"`
		VaultID  string         `json:"vault_id"`
		Payload  store.Item     `json:"payload"`
		Diff     map[string]any `json:"diff"`
	} `json:"action"`
}

// extractAction scans rawText for an embedded JSON action block.
// Returns the parsed ActionProposal stub (not yet persisted) and the cleaned text.
func extractAction(rawText, vaultID string) (*ActionProposal, string) {
	matches := jsonBlockRe.FindStringSubmatch(rawText)
	if matches == nil {
		return nil, rawText
	}
	jsonStr := matches[1]
	cleanText := strings.TrimSpace(jsonBlockRe.ReplaceAllString(rawText, ""))

	var env actionEnvelope
	if err := json.Unmarshal([]byte(jsonStr), &env); err != nil || env.Action.Type == "" {
		return nil, rawText
	}

	resolvedVaultID := env.Action.VaultID
	if resolvedVaultID == "" {
		resolvedVaultID = vaultID
	}

	proposal := &ActionProposal{
		ActionType: env.Action.Type,
		ItemType:   env.Action.ItemType,
		VaultID:    resolvedVaultID,
		Payload:    env.Action.Payload,
		Diff:       env.Action.Diff,
		Status:     "pending",
	}
	return proposal, cleanText
}
