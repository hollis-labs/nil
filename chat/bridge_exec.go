package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hollis-labs/nil/ingest"
	"github.com/hollis-labs/nil/service/items"
	"github.com/hollis-labs/nil/store"
)

// chatSvc is the package-level items service used by tool execution handlers.
// Stateless; safe to share.
var chatSvc = items.New()

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
