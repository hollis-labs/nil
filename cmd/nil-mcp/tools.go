package main

// Per-tool argument structs and handler implementations. Each handler is
// independent and routes through apiClient.do (client.go) to the NIL HTTP
// API. Tool registration and schema data describing these tools to MCP
// clients live in tool_schemas.go.

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ---------------------------------------------------------------------------
// Tool argument structs (decoded from the untyped args map via decodeArgs)
// ---------------------------------------------------------------------------

type argsSearch struct {
	Q            string   `json:"q"`
	Kind         string   `json:"kind"`
	Tags         []string `json:"tags"`
	Contexts     []string `json:"contexts"`
	Projects     []string `json:"projects"`
	Page         int      `json:"page"`
	PageSize     int      `json:"page_size"`
	VaultID      string   `json:"vault_id"`
	UpdatedSince string   `json:"updated_since"`
}

type argsGetItem struct {
	ID      int    `json:"id"`
	VaultID string `json:"vault_id"`
}

type argsGetBackrefs struct {
	ID      int    `json:"id"`
	VaultID string `json:"vault_id"`
}

type argsListItemIDs struct {
	Kind    string `json:"kind"`
	VaultID string `json:"vault_id"`
}

// Notes input on create/update accepts exactly one of notes_doc / notes_md /
// notes_html. The HTTP API does the conversion server-side via ingest;
// MCP just forwards whichever field is set.
type argsCreateItem struct {
	Title       string   `json:"title"`
	Kind        string   `json:"kind"`
	NotesDoc    string   `json:"notes_doc"`
	NotesMD     string   `json:"notes_md"`
	NotesHTML   string   `json:"notes_html"`
	Priority    string   `json:"priority"`
	DueAt       string   `json:"due_at"`
	Section     string   `json:"section"`
	Tags        []string `json:"tags"`
	Contexts    []string `json:"contexts"`
	Projects    []string `json:"projects"`
	VaultID     string   `json:"vault_id"`
	ExternalRef string   `json:"external_ref"`
}

// argsCreateItemBatchEntry is one element of argsCreateItemsBatch.Items —
// deliberately the same field set as argsCreateItem (minus vault_id, which
// applies once to the whole batch call, not per item).
type argsCreateItemBatchEntry struct {
	Title       string   `json:"title"`
	Kind        string   `json:"kind"`
	NotesDoc    string   `json:"notes_doc"`
	NotesMD     string   `json:"notes_md"`
	NotesHTML   string   `json:"notes_html"`
	Priority    string   `json:"priority"`
	DueAt       string   `json:"due_at"`
	Section     string   `json:"section"`
	Tags        []string `json:"tags"`
	Contexts    []string `json:"contexts"`
	Projects    []string `json:"projects"`
	ExternalRef string   `json:"external_ref"`
}

type argsCreateItemsBatch struct {
	Items   []argsCreateItemBatchEntry `json:"items"`
	VaultID string                     `json:"vault_id"`
}

type argsUpdateItem struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Kind        string   `json:"kind"`
	NotesDoc    string   `json:"notes_doc"`
	NotesMD     string   `json:"notes_md"`
	NotesHTML   string   `json:"notes_html"`
	Priority    string   `json:"priority"`
	DueAt       string   `json:"due_at"`
	Section     string   `json:"section"`
	Tags        []string `json:"tags"`
	Contexts    []string `json:"contexts"`
	Projects    []string `json:"projects"`
	VaultID     string   `json:"vault_id"`
	ExternalRef string   `json:"external_ref"`
}

type argsDeleteItem struct {
	ID      int    `json:"id"`
	VaultID string `json:"vault_id"`
}

type argsToggleComplete struct {
	ID        int    `json:"id"`
	Completed bool   `json:"completed"`
	VaultID   string `json:"vault_id"`
}

type argsArchive struct {
	ID       int    `json:"id"`
	Archived bool   `json:"archived"`
	VaultID  string `json:"vault_id"`
}

type argsListInbox struct {
	Q        string `json:"q"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type argsCreateInbox struct {
	Title     string   `json:"title"`
	NotesDoc  string   `json:"notes_doc"`
	NotesMD   string   `json:"notes_md"`
	NotesHTML string   `json:"notes_html"`
	Kind      string   `json:"kind"`
	Priority  string   `json:"priority"`
	DueAt     string   `json:"due_at"`
	Tags      []string `json:"tags"`
	Contexts  []string `json:"contexts"`
	Projects  []string `json:"projects"`
}

type argsProcessInbox struct {
	ID int `json:"id"`
}

type argsGetTaxonomy struct {
	VaultID string `json:"vault_id"`
}

// ---------------------------------------------------------------------------
// Tool handlers
// ---------------------------------------------------------------------------

func (c *apiClient) toolListVaults(ctx context.Context, _ map[string]any) (any, error) {
	data, err := c.do(ctx, "GET", "/api/v1/vaults", nil, "")
	if err != nil {
		return nil, err
	}
	return decodeResult(data)
}

func (c *apiClient) toolSearch(ctx context.Context, args map[string]any) (any, error) {
	var a argsSearch
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}

	q := url.Values{}
	if a.Q != "" {
		q.Set("q", a.Q)
	}
	if a.Kind != "" && a.Kind != "all" {
		q.Set("kind", a.Kind)
	}
	for _, t := range a.Tags {
		q.Add("tags", t)
	}
	for _, cx := range a.Contexts {
		q.Add("contexts", cx)
	}
	for _, p := range a.Projects {
		q.Add("projects", p)
	}
	if a.Page > 0 {
		q.Set("page", strconv.Itoa(a.Page))
	}
	if a.PageSize > 0 {
		q.Set("page_size", strconv.Itoa(a.PageSize))
	}
	if a.UpdatedSince != "" {
		q.Set("updated_since", a.UpdatedSince)
	}

	path := "/api/v1/search"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}

	data, err := c.do(ctx, "GET", path, nil, a.VaultID)
	if err != nil {
		return nil, err
	}
	return decodeResult(data)
}

func (c *apiClient) toolGetItem(ctx context.Context, args map[string]any) (any, error) {
	var a argsGetItem
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.ID == 0 {
		return nil, fmt.Errorf("id is required")
	}

	data, err := c.do(ctx, "GET", fmt.Sprintf("/api/v1/items/%d", a.ID), nil, a.VaultID)
	if err != nil {
		return nil, err
	}
	return decodeResult(data)
}

// toolGetBackrefs is a separate tool rather than a `backrefs` field folded
// into nil_get_item's response: every other per-item read/action in this
// tool list (nil_toggle_complete, nil_archive, nil_list_inbox,
// nil_get_taxonomy) is already its own tool rather than a field bolted onto
// nil_get_item, so this follows the dominant existing pattern. It also keeps
// nil_get_item cheap — a plain fetch doesn't pay for the refs JOIN unless the
// caller actually wants backlinks.
func (c *apiClient) toolGetBackrefs(ctx context.Context, args map[string]any) (any, error) {
	var a argsGetBackrefs
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.ID == 0 {
		return nil, fmt.Errorf("id is required")
	}

	data, err := c.do(ctx, "GET", fmt.Sprintf("/api/v1/items/%d/backrefs", a.ID), nil, a.VaultID)
	if err != nil {
		return nil, err
	}
	return decodeResult(data)
}

// toolListItemIDs is a separate tool rather than a flag on nil_search: it
// backs the deletion/change-signal endpoint (GET /api/v1/items/ids), whose
// entire point is a minimal id+updated_at response an external sync
// consumer can cheaply diff against its own known-ID set to detect
// deletions — Nil has no soft-delete/tombstone concept, so this is the only
// way such a consumer learns an item was hard-deleted. Deliberately does
// NOT accept updated_since (unlike nil_search): this call always needs the
// FULL current-ID set to diff against, not an incremental slice, or
// currently-existing IDs outside the window would read as false deletions.
func (c *apiClient) toolListItemIDs(ctx context.Context, args map[string]any) (any, error) {
	var a argsListItemIDs
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}

	q := url.Values{}
	if a.Kind != "" && a.Kind != "all" {
		q.Set("kind", a.Kind)
	}

	path := "/api/v1/items/ids"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}

	data, err := c.do(ctx, "GET", path, nil, a.VaultID)
	if err != nil {
		return nil, err
	}
	return decodeResult(data)
}

func (c *apiClient) toolCreateItem(ctx context.Context, args map[string]any) (any, error) {
	var a argsCreateItem
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	body := map[string]any{
		"title": a.Title,
	}
	if a.Kind != "" {
		body["kind"] = a.Kind
	}
	if a.NotesDoc != "" {
		body["notes_doc"] = a.NotesDoc
	}
	if a.NotesMD != "" {
		body["notes_md"] = a.NotesMD
	}
	if a.NotesHTML != "" {
		body["notes_html"] = a.NotesHTML
	}
	if a.Priority != "" {
		body["priority"] = a.Priority
	}
	if a.DueAt != "" {
		body["due_at"] = a.DueAt
	}
	if a.Section != "" {
		body["section"] = a.Section
	}
	if len(a.Tags) > 0 {
		body["tags"] = a.Tags
	}
	if len(a.Contexts) > 0 {
		body["contexts"] = a.Contexts
	}
	if len(a.Projects) > 0 {
		body["projects"] = a.Projects
	}
	if a.ExternalRef != "" {
		body["external_ref"] = a.ExternalRef
	}

	data, err := c.do(ctx, "POST", "/api/v1/items", body, a.VaultID)
	if err != nil {
		return nil, err
	}
	return decodeResult(data)
}

// toolCreateItemsBatch is a separate tool rather than an "items" array
// accepted by nil_create_item: this epic's established precedent is one
// concern per tool (see nil_get_backrefs / nil_list_item_ids each getting
// their own tool instead of a flag bolted onto nil_get_item / nil_search),
// and a batch call has a materially different contract from a single create
// — different response shape (a list, not one item) and all-or-nothing
// failure semantics across every entry, not just one. Proxies straight to
// POST /api/v1/items/batch, which is itself a dedicated route for the same
// reasons (see apiserver.go's handleCreateItemsBatch doc comment).
func (c *apiClient) toolCreateItemsBatch(ctx context.Context, args map[string]any) (any, error) {
	var a argsCreateItemsBatch
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if len(a.Items) == 0 {
		return nil, fmt.Errorf("items must be a non-empty array")
	}
	for i, it := range a.Items {
		if it.Title == "" {
			return nil, fmt.Errorf("items[%d]: title is required", i)
		}
	}

	bodyItems := make([]map[string]any, len(a.Items))
	for i, it := range a.Items {
		item := map[string]any{"title": it.Title}
		if it.Kind != "" {
			item["kind"] = it.Kind
		}
		if it.NotesDoc != "" {
			item["notes_doc"] = it.NotesDoc
		}
		if it.NotesMD != "" {
			item["notes_md"] = it.NotesMD
		}
		if it.NotesHTML != "" {
			item["notes_html"] = it.NotesHTML
		}
		if it.Priority != "" {
			item["priority"] = it.Priority
		}
		if it.DueAt != "" {
			item["due_at"] = it.DueAt
		}
		if it.Section != "" {
			item["section"] = it.Section
		}
		if len(it.Tags) > 0 {
			item["tags"] = it.Tags
		}
		if len(it.Contexts) > 0 {
			item["contexts"] = it.Contexts
		}
		if len(it.Projects) > 0 {
			item["projects"] = it.Projects
		}
		if it.ExternalRef != "" {
			item["external_ref"] = it.ExternalRef
		}
		bodyItems[i] = item
	}

	data, err := c.do(ctx, "POST", "/api/v1/items/batch", map[string]any{"items": bodyItems}, a.VaultID)
	if err != nil {
		return nil, err
	}
	return decodeResult(data)
}

func (c *apiClient) toolUpdateItem(ctx context.Context, args map[string]any) (any, error) {
	var a argsUpdateItem
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.ID == 0 {
		return nil, fmt.Errorf("id is required")
	}

	// Build a map with only the fields explicitly provided. args is the raw
	// (already-decoded) tool-call arguments, so presence is a plain key check
	// against it — no need to re-marshal/re-unmarshal to detect which keys
	// were present, unlike the old json.RawMessage-based transport.
	body := map[string]any{}
	if _, ok := args["title"]; ok {
		body["title"] = a.Title
	}
	if _, ok := args["kind"]; ok {
		body["kind"] = a.Kind
	}
	if _, ok := args["notes_doc"]; ok {
		body["notes_doc"] = a.NotesDoc
	}
	if _, ok := args["notes_md"]; ok {
		body["notes_md"] = a.NotesMD
	}
	if _, ok := args["notes_html"]; ok {
		body["notes_html"] = a.NotesHTML
	}
	if _, ok := args["priority"]; ok {
		body["priority"] = a.Priority
	}
	if _, ok := args["due_at"]; ok {
		body["due_at"] = a.DueAt
	}
	if _, ok := args["section"]; ok {
		body["section"] = a.Section
	}
	if _, ok := args["tags"]; ok {
		body["tags"] = a.Tags
	}
	if _, ok := args["contexts"]; ok {
		body["contexts"] = a.Contexts
	}
	if _, ok := args["projects"]; ok {
		body["projects"] = a.Projects
	}
	if _, ok := args["external_ref"]; ok {
		body["external_ref"] = a.ExternalRef
	}

	data, err := c.do(ctx, "PUT", fmt.Sprintf("/api/v1/items/%d", a.ID), body, a.VaultID)
	if err != nil {
		return nil, err
	}
	return decodeResult(data)
}

func (c *apiClient) toolDeleteItem(ctx context.Context, args map[string]any) (any, error) {
	var a argsDeleteItem
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.ID == 0 {
		return nil, fmt.Errorf("id is required")
	}

	if _, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/v1/items/%d", a.ID), nil, a.VaultID); err != nil {
		return nil, err
	}
	return fmt.Sprintf("Item %d deleted successfully", a.ID), nil
}

func (c *apiClient) toolToggleComplete(ctx context.Context, args map[string]any) (any, error) {
	var a argsToggleComplete
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.ID == 0 {
		return nil, fmt.Errorf("id is required")
	}

	body := map[string]any{"completed": a.Completed}
	data, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/items/%d/complete", a.ID), body, a.VaultID)
	if err != nil {
		return nil, err
	}
	return decodeResult(data)
}

func (c *apiClient) toolArchive(ctx context.Context, args map[string]any) (any, error) {
	var a argsArchive
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.ID == 0 {
		return nil, fmt.Errorf("id is required")
	}

	body := map[string]any{"archived": a.Archived}
	data, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/items/%d/archive", a.ID), body, a.VaultID)
	if err != nil {
		return nil, err
	}
	return decodeResult(data)
}

func (c *apiClient) toolListInbox(ctx context.Context, args map[string]any) (any, error) {
	var a argsListInbox
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}

	q := url.Values{}
	if a.Q != "" {
		q.Set("q", a.Q)
	}
	if a.Page > 0 {
		q.Set("page", strconv.Itoa(a.Page))
	}
	if a.PageSize > 0 {
		q.Set("page_size", strconv.Itoa(a.PageSize))
	}

	path := "/api/v1/inbox"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}

	data, err := c.do(ctx, "GET", path, nil, "")
	if err != nil {
		return nil, err
	}
	return decodeResult(data)
}

func (c *apiClient) toolCreateInbox(ctx context.Context, args map[string]any) (any, error) {
	var a argsCreateInbox
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}

	body := map[string]any{}
	if a.Title != "" {
		body["title"] = a.Title
	}
	if a.NotesDoc != "" {
		body["notes_doc"] = a.NotesDoc
	}
	if a.NotesMD != "" {
		body["notes_md"] = a.NotesMD
	}
	if a.NotesHTML != "" {
		body["notes_html"] = a.NotesHTML
	}
	if a.Kind != "" {
		body["kind"] = a.Kind
	}
	if a.Priority != "" {
		body["priority"] = a.Priority
	}
	if a.DueAt != "" {
		body["due_at"] = a.DueAt
	}
	if len(a.Tags) > 0 {
		body["tags"] = a.Tags
	}
	if len(a.Contexts) > 0 {
		body["contexts"] = a.Contexts
	}
	if len(a.Projects) > 0 {
		body["projects"] = a.Projects
	}

	data, err := c.do(ctx, "POST", "/api/v1/inbox", body, "")
	if err != nil {
		return nil, err
	}
	return decodeResult(data)
}

func (c *apiClient) toolProcessInbox(ctx context.Context, args map[string]any) (any, error) {
	var a argsProcessInbox
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.ID == 0 {
		return nil, fmt.Errorf("id is required")
	}

	if _, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/inbox/%d/process", a.ID), map[string]any{}, ""); err != nil {
		return nil, err
	}
	return fmt.Sprintf("Inbox item %d processed successfully", a.ID), nil
}

func (c *apiClient) toolGetTaxonomy(ctx context.Context, args map[string]any) (any, error) {
	var a argsGetTaxonomy
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}

	data, err := c.do(ctx, "GET", "/api/v1/taxonomy", nil, a.VaultID)
	if err != nil {
		return nil, err
	}
	return decodeResult(data)
}
