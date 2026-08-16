package main

// JSON-RPC 2.0 stdio transport: wire types, the read/dispatch loop, and the
// shared HTTP-proxy helper every tool handler goes through. Tool
// implementations live in tools.go; tool schema data lives in
// tool_schemas.go.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	case "nil_get_backrefs":
		return s.toolGetBackrefs(args)
	case "nil_list_item_ids":
		return s.toolListItemIDs(args)
	case "nil_create_item":
		return s.toolCreateItem(args)
	case "nil_create_items":
		return s.toolCreateItemsBatch(args)
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
