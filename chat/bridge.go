// Package chat implements NIL's AI chat bridge: a direct HTTP integration with
// the Anthropic Messages API (no SDK, no CLI subprocess — see historical ADR-001).
//
// bridge.go holds the entry point (Bridge.Send), the agentic tool-use loop, and
// the raw HTTP transport (callAPI) plus the wire-format types for the Anthropic
// API. The rest of the bridge's concerns live alongside it in this package:
//   - bridge_tools.go   — tool schema declarations (buildTools)
//   - bridge_exec.go    — tool-execution handlers (executeTool and friends)
//   - bridge_prompt.go  — system prompt templating and action-block extraction
//   - models.go         — shared request/response/session types
//   - profile.go        — vault snapshot computation for prompt injection
//   - actions.go        — the propose/approve action-runner
//   - store.go          — chat.db (sessions/messages/proposals/templates)
package chat

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

const claudeAPIURL = "https://api.anthropic.com/v1/messages"
const claudeAPIVersion = "2023-06-01"
const defaultModel = "claude-haiku-4-5-20251001"
const defaultMaxTokens = 2048
const maxToolIterations = 5

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
