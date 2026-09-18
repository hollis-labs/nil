package main

// apiClient is the thin HTTP client every tool handler proxies through to
// reach NIL's own local HTTP API (ADR-0005: nil-mcp has zero direct
// dependency on store/vault). Tool implementations live in tools.go; tool
// registration and schema data live in tool_schemas.go.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type apiClient struct {
	base   string
	apiKey string
	http   *http.Client
}

func newAPIClient(base, apiKey string) *apiClient {
	return &apiClient{
		base:   base,
		apiKey: apiKey,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

// do performs an HTTP request against the NIL API.
// body may be nil (GET/DELETE) or any JSON-serializable value (POST/PUT).
// vaultID may be "" to use the active vault.
// Returns the raw `data` field from the response envelope, or an error.
func (c *apiClient) do(ctx context.Context, method, path string, body any, vaultID string) (json.RawMessage, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-Agent-Source", "nil-mcp")
	if vaultID != "" {
		req.Header.Set("X-Vault-ID", vaultID)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
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

// decodeArgs re-marshals the untyped tool-call arguments map go-mcp decodes
// and unmarshals it into a typed args struct, so every handler in tools.go
// keeps the same typed-struct decode shape it had under the hand-rolled
// JSON-RPC transport.
func decodeArgs(args map[string]any, out any) error {
	b, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("marshal arguments: %w", err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	return nil
}

// decodeResult unmarshals a raw API response body into an untyped value so a
// go-mcp ToolHandler can return it directly. go-mcp then JSON-marshals it
// into both CallToolResult.StructuredContent (SEP-2106) and mirrored text
// content, so a client reading either representation sees the same data.
func decodeResult(data json.RawMessage) (any, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return v, nil
}
