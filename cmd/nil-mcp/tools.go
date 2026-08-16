package main

// Per-tool argument structs and handler implementations. Each handler is
// independent and routes through Server.apiDo (rpc.go) to the NIL HTTP API.
// The JSON-RPC transport that calls into these lives in rpc.go; the schema
// data describing these tools to MCP clients lives in tool_schemas.go.

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ---------------------------------------------------------------------------
// Tool argument structs (decoded from JSON)
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

func (s *Server) toolListVaults() (string, error) {
	data, err := s.apiDo("GET", "/api/v1/vaults", nil, "")
	if err != nil {
		return "", err
	}
	return prettyJSON(data), nil
}

func (s *Server) toolSearch(args json.RawMessage) (string, error) {
	var a argsSearch
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
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
	for _, c := range a.Contexts {
		q.Add("contexts", c)
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

	data, err := s.apiDo("GET", path, nil, a.VaultID)
	if err != nil {
		return "", err
	}
	return prettyJSON(data), nil
}

func (s *Server) toolGetItem(args json.RawMessage) (string, error) {
	var a argsGetItem
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.ID == 0 {
		return "", fmt.Errorf("id is required")
	}

	data, err := s.apiDo("GET", fmt.Sprintf("/api/v1/items/%d", a.ID), nil, a.VaultID)
	if err != nil {
		return "", err
	}
	return prettyJSON(data), nil
}

// toolGetBackrefs is a separate tool rather than a `backrefs` field folded
// into nil_get_item's response: every other per-item read/action in this
// tool list (nil_toggle_complete, nil_archive, nil_list_inbox,
// nil_get_taxonomy) is already its own tool rather than a field bolted onto
// nil_get_item, so this follows the dominant existing pattern. It also keeps
// nil_get_item cheap — a plain fetch doesn't pay for the refs JOIN unless the
// caller actually wants backlinks.
func (s *Server) toolGetBackrefs(args json.RawMessage) (string, error) {
	var a argsGetBackrefs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.ID == 0 {
		return "", fmt.Errorf("id is required")
	}

	data, err := s.apiDo("GET", fmt.Sprintf("/api/v1/items/%d/backrefs", a.ID), nil, a.VaultID)
	if err != nil {
		return "", err
	}
	return prettyJSON(data), nil
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
func (s *Server) toolListItemIDs(args json.RawMessage) (string, error) {
	var a argsListItemIDs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	q := url.Values{}
	if a.Kind != "" && a.Kind != "all" {
		q.Set("kind", a.Kind)
	}

	path := "/api/v1/items/ids"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}

	data, err := s.apiDo("GET", path, nil, a.VaultID)
	if err != nil {
		return "", err
	}
	return prettyJSON(data), nil
}

func (s *Server) toolCreateItem(args json.RawMessage) (string, error) {
	var a argsCreateItem
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.Title == "" {
		return "", fmt.Errorf("title is required")
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

	data, err := s.apiDo("POST", "/api/v1/items", body, a.VaultID)
	if err != nil {
		return "", err
	}
	return prettyJSON(data), nil
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
func (s *Server) toolCreateItemsBatch(args json.RawMessage) (string, error) {
	var a argsCreateItemsBatch
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if len(a.Items) == 0 {
		return "", fmt.Errorf("items must be a non-empty array")
	}
	for i, it := range a.Items {
		if it.Title == "" {
			return "", fmt.Errorf("items[%d]: title is required", i)
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

	data, err := s.apiDo("POST", "/api/v1/items/batch", map[string]any{"items": bodyItems}, a.VaultID)
	if err != nil {
		return "", err
	}
	return prettyJSON(data), nil
}

func (s *Server) toolUpdateItem(args json.RawMessage) (string, error) {
	var a argsUpdateItem
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.ID == 0 {
		return "", fmt.Errorf("id is required")
	}

	// Build a map with only the fields explicitly provided.
	// We decode again into a raw map to detect which keys were present.
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(args, &raw)

	body := map[string]any{}
	if _, ok := raw["title"]; ok {
		body["title"] = a.Title
	}
	if _, ok := raw["kind"]; ok {
		body["kind"] = a.Kind
	}
	if _, ok := raw["notes_doc"]; ok {
		body["notes_doc"] = a.NotesDoc
	}
	if _, ok := raw["notes_md"]; ok {
		body["notes_md"] = a.NotesMD
	}
	if _, ok := raw["notes_html"]; ok {
		body["notes_html"] = a.NotesHTML
	}
	if _, ok := raw["priority"]; ok {
		body["priority"] = a.Priority
	}
	if _, ok := raw["due_at"]; ok {
		body["due_at"] = a.DueAt
	}
	if _, ok := raw["section"]; ok {
		body["section"] = a.Section
	}
	if _, ok := raw["tags"]; ok {
		body["tags"] = a.Tags
	}
	if _, ok := raw["contexts"]; ok {
		body["contexts"] = a.Contexts
	}
	if _, ok := raw["projects"]; ok {
		body["projects"] = a.Projects
	}
	if _, ok := raw["external_ref"]; ok {
		body["external_ref"] = a.ExternalRef
	}

	data, err := s.apiDo("PUT", fmt.Sprintf("/api/v1/items/%d", a.ID), body, a.VaultID)
	if err != nil {
		return "", err
	}
	return prettyJSON(data), nil
}

func (s *Server) toolDeleteItem(args json.RawMessage) (string, error) {
	var a argsDeleteItem
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.ID == 0 {
		return "", fmt.Errorf("id is required")
	}

	if _, err := s.apiDo("DELETE", fmt.Sprintf("/api/v1/items/%d", a.ID), nil, a.VaultID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Item %d deleted successfully", a.ID), nil
}

func (s *Server) toolToggleComplete(args json.RawMessage) (string, error) {
	var a argsToggleComplete
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.ID == 0 {
		return "", fmt.Errorf("id is required")
	}

	body := map[string]any{"completed": a.Completed}
	data, err := s.apiDo("POST", fmt.Sprintf("/api/v1/items/%d/complete", a.ID), body, a.VaultID)
	if err != nil {
		return "", err
	}
	return prettyJSON(data), nil
}

func (s *Server) toolArchive(args json.RawMessage) (string, error) {
	var a argsArchive
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.ID == 0 {
		return "", fmt.Errorf("id is required")
	}

	body := map[string]any{"archived": a.Archived}
	data, err := s.apiDo("POST", fmt.Sprintf("/api/v1/items/%d/archive", a.ID), body, a.VaultID)
	if err != nil {
		return "", err
	}
	return prettyJSON(data), nil
}

func (s *Server) toolListInbox(args json.RawMessage) (string, error) {
	var a argsListInbox
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
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

	data, err := s.apiDo("GET", path, nil, "")
	if err != nil {
		return "", err
	}
	return prettyJSON(data), nil
}

func (s *Server) toolCreateInbox(args json.RawMessage) (string, error) {
	var a argsCreateInbox
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
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

	data, err := s.apiDo("POST", "/api/v1/inbox", body, "")
	if err != nil {
		return "", err
	}
	return prettyJSON(data), nil
}

func (s *Server) toolProcessInbox(args json.RawMessage) (string, error) {
	var a argsProcessInbox
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.ID == 0 {
		return "", fmt.Errorf("id is required")
	}

	if _, err := s.apiDo("POST", fmt.Sprintf("/api/v1/inbox/%d/process", a.ID), map[string]any{}, ""); err != nil {
		return "", err
	}
	return fmt.Sprintf("Inbox item %d processed successfully", a.ID), nil
}

func (s *Server) toolGetTaxonomy(args json.RawMessage) (string, error) {
	var a argsGetTaxonomy
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	data, err := s.apiDo("GET", "/api/v1/taxonomy", nil, a.VaultID)
	if err != nil {
		return "", err
	}
	return prettyJSON(data), nil
}
