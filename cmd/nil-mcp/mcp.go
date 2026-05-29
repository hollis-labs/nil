package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// JSON-RPC 2.0 wire types
// ---------------------------------------------------------------------------

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // absent on notifications
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string { return e.Message }

// ---------------------------------------------------------------------------
// MCP tool types
// ---------------------------------------------------------------------------

type toolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string                `json:"type"`
	Properties map[string]schemaProp `json:"properties,omitempty"`
	Required   []string              `json:"required,omitempty"`
}

type schemaProp struct {
	Type        string      `json:"type,omitempty"`
	Description string      `json:"description,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Default     any         `json:"default,omitempty"`
	Items       *schemaProp `json:"items,omitempty"`
}

type toolResult struct {
	Content []contentItem `json:"content"`
	IsError bool          `json:"isError"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ---------------------------------------------------------------------------
// Tool argument structs (decoded from JSON)
// ---------------------------------------------------------------------------

type argsSearch struct {
	Q        string   `json:"q"`
	Kind     string   `json:"kind"`
	Tags     []string `json:"tags"`
	Contexts []string `json:"contexts"`
	Projects []string `json:"projects"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	VaultID  string   `json:"vault_id"`
}

type argsGetItem struct {
	ID      int    `json:"id"`
	VaultID string `json:"vault_id"`
}

// Notes input on create/update accepts exactly one of notes_doc / notes_md /
// notes_html. The HTTP API does the conversion server-side via ingest;
// MCP just forwards whichever field is set.
type argsCreateItem struct {
	Title     string   `json:"title"`
	Kind      string   `json:"kind"`
	NotesDoc  string   `json:"notes_doc"`
	NotesMD   string   `json:"notes_md"`
	NotesHTML string   `json:"notes_html"`
	Priority  string   `json:"priority"`
	DueAt     string   `json:"due_at"`
	Section   string   `json:"section"`
	Tags      []string `json:"tags"`
	Contexts  []string `json:"contexts"`
	Projects  []string `json:"projects"`
	VaultID   string   `json:"vault_id"`
}

type argsUpdateItem struct {
	ID        int      `json:"id"`
	Title     string   `json:"title"`
	Kind      string   `json:"kind"`
	NotesDoc  string   `json:"notes_doc"`
	NotesMD   string   `json:"notes_md"`
	NotesHTML string   `json:"notes_html"`
	Priority  string   `json:"priority"`
	DueAt     string   `json:"due_at"`
	Section   string   `json:"section"`
	Tags      []string `json:"tags"`
	Contexts  []string `json:"contexts"`
	Projects  []string `json:"projects"`
	VaultID   string   `json:"vault_id"`
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
// Server
// ---------------------------------------------------------------------------

type Server struct {
	apiBase string
	apiKey  string
	client  *http.Client
}

func NewServer(apiBase, apiKey string) *Server {
	return &Server{
		apiBase: apiBase,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Run is the main read loop: one JSON object per line on in, responses to out.
func (s *Server) Run(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	enc := json.NewEncoder(out)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		// Recover from any panic in a single request.
		func() {
			defer func() {
				if r := recover(); r != nil {
					// We can't know the ID here, but try to parse it quickly.
					var raw rpcRequest
					_ = json.Unmarshal(line, &raw)
					if raw.ID != nil {
						_ = enc.Encode(rpcResponse{
							JSONRPC: "2.0",
							ID:      raw.ID,
							Error:   &rpcError{Code: -32603, Message: fmt.Sprintf("internal panic: %v", r)},
						})
					}
				}
			}()

			var req rpcRequest
			if err := json.Unmarshal(line, &req); err != nil {
				_ = enc.Encode(rpcResponse{
					JSONRPC: "2.0",
					ID:      json.RawMessage(`null`),
					Error:   &rpcError{Code: -32700, Message: "Parse error: " + err.Error()},
				})
				return
			}

			// Notifications have no id — never respond.
			if req.ID == nil {
				s.handleNotification(req)
				return
			}

			result, rpcErr := s.handle(req)
			resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
			if rpcErr != nil {
				resp.Error = rpcErr
			} else {
				resp.Result = result
			}
			_ = enc.Encode(resp)
		}()
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "nil-mcp: stdin read error: %v\n", err)
	}
}

func (s *Server) handleNotification(req rpcRequest) {
	// Nothing to do — notifications are fire-and-forget.
}

// handle dispatches a request by method and returns (result, error).
func (s *Server) handle(req rpcRequest) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "nil-mcp", "version": "1.0.0"},
			"instructions": "NIL personal task and note manager. Use nil_list_vaults to discover available vaults. " +
				"Pass vault_id in tool calls to target a specific vault; omit to use the active vault. " +
				"The inbox is a fast-capture area — use nil_create_inbox to add items without taxonomy friction, " +
				"then nil_process_inbox when ready to promote them.",
		}, nil

	case "ping":
		return map[string]any{}, nil

	case "tools/list":
		return map[string]any{"tools": toolList()}, nil

	case "tools/call":
		return s.dispatchToolCall(req.Params)

	default:
		return nil, &rpcError{Code: -32601, Message: "Method not found"}
	}
}

// dispatchToolCall decodes params and routes to the correct tool handler.
func (s *Server) dispatchToolCall(params json.RawMessage) (any, *rpcError) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid params: " + err.Error()}
	}

	text, toolErr := s.callTool(p.Name, p.Arguments)
	if toolErr != nil {
		return toolResult{ //nolint:nilerr
			Content: []contentItem{{Type: "text", Text: toolErr.Error()}},
			IsError: true,
		}, nil
	}
	return toolResult{
		Content: []contentItem{{Type: "text", Text: text}},
		IsError: false,
	}, nil
}

// callTool routes to the appropriate tool handler function.
func (s *Server) callTool(name string, args json.RawMessage) (string, error) {
	if args == nil {
		args = json.RawMessage(`{}`)
	}
	switch name {
	case "nil_list_vaults":
		return s.toolListVaults()
	case "nil_search":
		return s.toolSearch(args)
	case "nil_get_item":
		return s.toolGetItem(args)
	case "nil_create_item":
		return s.toolCreateItem(args)
	case "nil_update_item":
		return s.toolUpdateItem(args)
	case "nil_delete_item":
		return s.toolDeleteItem(args)
	case "nil_toggle_complete":
		return s.toolToggleComplete(args)
	case "nil_archive":
		return s.toolArchive(args)
	case "nil_list_inbox":
		return s.toolListInbox(args)
	case "nil_create_inbox":
		return s.toolCreateInbox(args)
	case "nil_process_inbox":
		return s.toolProcessInbox(args)
	case "nil_get_taxonomy":
		return s.toolGetTaxonomy(args)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// ---------------------------------------------------------------------------
// HTTP helper
// ---------------------------------------------------------------------------

// apiDo performs an HTTP request against the NIL API.
// body may be nil (GET/DELETE) or any JSON-serializable value (POST/PUT).
// vaultID may be "" to use the active vault.
// Returns the raw `data` field from the response envelope, or an error.
func (s *Server) apiDo(method, path string, body any, vaultID string) (json.RawMessage, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, s.apiBase+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("X-API-Key", s.apiKey)
	req.Header.Set("X-Agent-Source", "nil-mcp")
	if vaultID != "" {
		req.Header.Set("X-Vault-ID", vaultID)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return nil, fmt.Errorf("nil app is not running (connection refused) — start NIL and ensure the API is enabled in Settings")
		}
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	var envelope struct {
		OK    bool            `json:"ok"`
		Data  json.RawMessage `json:"data"`
		Error string          `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}

	if !envelope.OK || resp.StatusCode >= 300 {
		msg := envelope.Error
		if msg == "" {
			msg = fmt.Sprintf("API error (HTTP %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", msg)
	}

	return envelope.Data, nil
}

// prettyJSON formats raw JSON for human-readable output.
func prettyJSON(raw json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return string(raw)
	}
	return buf.String()
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

	data, err := s.apiDo("POST", "/api/v1/items", body, a.VaultID)
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

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------

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
			Description: "Search todos and notes in a NIL vault using full-text search.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"q":         strProp("Search query"),
					"kind":      strEnumProp("Item kind filter", "todo", "note", "scratch", "all"),
					"tags":      strArrayProp("Filter by tags"),
					"contexts":  strArrayProp("Filter by contexts"),
					"projects":  strArrayProp("Filter by projects"),
					"page":      intProp("Page number (0-based)"),
					"page_size": intProp("Results per page (default 20)"),
					"vault_id":  vaultIDProp,
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
			Name:        "nil_create_item",
			Description: "Create a new item in a NIL vault (not the inbox). Notes body accepts markdown (notes_md), HTML (notes_html), or pre-built TipTap doc JSON (notes_doc); precedence is doc > md > html.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"title":      strProp("Item title"),
					"kind":       strEnumProp("Item kind", "todo", "note", "scratch"),
					"notes_md":   strProp("Markdown body (converted server-side to TipTap doc)"),
					"notes_html": strProp("HTML body (converted server-side to TipTap doc)"),
					"notes_doc":  strProp("Pre-built TipTap doc JSON (skips server conversion)"),
					"priority":   strEnumProp("Priority", "A", "B", "C"),
					"due_at":     strProp("Due date in RFC3339 format"),
					"section":    strEnumProp("Section", "now", "soon", "anytime"),
					"tags":       strArrayProp("Tags"),
					"contexts":   strArrayProp("Contexts"),
					"projects":   strArrayProp("Projects"),
					"vault_id":   vaultIDProp,
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "nil_update_item",
			Description: "Update fields of an existing item. Only provided fields are changed. Notes body accepts markdown (notes_md), HTML (notes_html), or pre-built TipTap doc JSON (notes_doc).",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"id":         intProp("Item ID"),
					"title":      strProp("New title"),
					"kind":       strEnumProp("New kind", "todo", "note", "scratch"),
					"notes_md":   strProp("New markdown body (converted server-side)"),
					"notes_html": strProp("New HTML body (converted server-side)"),
					"notes_doc":  strProp("New pre-built TipTap doc JSON"),
					"priority":   strEnumProp("Priority", "A", "B", "C"),
					"due_at":     strProp("Due date in RFC3339 format"),
					"section":    strEnumProp("Section", "now", "soon", "anytime"),
					"tags":       strArrayProp("Tags (replaces existing)"),
					"contexts":   strArrayProp("Contexts (replaces existing)"),
					"projects":   strArrayProp("Projects (replaces existing)"),
					"vault_id":   vaultIDProp,
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
